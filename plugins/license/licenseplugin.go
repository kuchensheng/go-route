//go:build (linux && cgo) || (darwin && cgo) || (freebsd && cgo)
// +build linux,cgo darwin,cgo freebsd,cgo

// Package license OS鉴权插件
package main

import "C"
import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/oliveagle/jsonpath"
	"github.com/robfig/cron"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"os"
)

var Req *http.Request
var W http.Response

var lc *LicenseConf
var hasLic = true

type LicenseConf struct {
	LicenseHost string `json:"host"`
	LicenseUrl  string `json:"url"`
}

// init 函数的目的是在插件模块加载的时候自动执行一些我们要做的事情，比如：自动将方法和类型注册到插件平台、输出插件信息等等。
func init() {
	fmt.Println("授权校验插件信息")
	file, err := os.OpenFile("license.json", os.O_RDONLY, 0666)
	lc = &LicenseConf{
		LicenseHost: "isc-license-service:9013",
		LicenseUrl:  "/api/core/license/valid",
	}
	if err == nil {
		data, err := io.ReadAll(file)
		if err == nil {
			err = json.Unmarshal(data, lc)
			if err != nil {
				log.Info().Msgf("配置文件读取失败%v", err)
			}
		} else {
			log.Info().Msgf("配置文件读取失败%v", err)
		}
	}
	//开启定时任务
	cron := cron.New()
	errTimes := 0
	cron.AddFunc("0 */1 * * * ?", func() {
		//获取license信息
		res, err := http.Get(fmt.Sprintf("http://%s%s", lc.LicenseHost, lc.LicenseUrl))
		defer res.Body.Close()
		body := res.Body
		data, err := io.ReadAll(body)
		if err != nil || res.StatusCode != 200 {
			if errTimes > 10 {
				hasLic = false
			}
			errTimes += 1
			log.Warn().Msgf("读取license信息异常\n%v", err)
		} else {
			var jsonData interface{}
			json.Unmarshal(data, jsonData)
			code, err := jsonpath.JsonPathLookup(jsonData, "$.code")
			if err != nil {
				log.Warn().Msgf("从响应体中读取code异常\n%v", err)
				return
			}
			if code == "200" || code == "0" {
				errTimes = 0
				hasLic = true
			}
		}
	})
}

//Valid 函数则是我们需要在调用方显式查找的symbol
//export Valid
func Valid() error {
	if hasLic {
		return nil
	}
	resp := `
{
	"code": 403,
	"message":"OS未授权,请联系管理员"
}
`
	return errors.New(resp)
}

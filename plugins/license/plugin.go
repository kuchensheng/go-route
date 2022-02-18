// Package license OS鉴权插件
package main

import "C"
import (
	"encoding/json"
	"fmt"
	"github.com/oliveagle/jsonpath"
	"github.com/robfig/cron"
	"github.com/rs/zerolog/log"
	"io"
	"isc-route-service/common"
	"net/http"
)

var lc *LicenseConf
var hasLic = true

type LicenseConf struct {
	LicenseHost string `json:"host"`
	LicenseUrl  string `json:"url"`
}

// init 函数的目的是在插件模块加载的时候自动执行一些我们要做的事情，比如：自动将方法和类型注册到插件平台、输出插件信息等等。
func init() {
	log.Info().Msgf("授权校验插件初始化……")
	lc = &LicenseConf{
		LicenseHost: "isc-license-service:9013",
		LicenseUrl:  "/api/core/license/valid",
	}
	common.ReadJsonToStruct("license.json", lc)

	//开启定时任务
	cron := cron.New()
	errTimes := 0
	cron.AddFunc("0 */1 * * * ?", func() {
		//获取license信息
		licenseUrl := fmt.Sprintf("http://%s%s", lc.LicenseHost, lc.LicenseUrl)
		log.Debug().Msgf("license请求地址:%s", licenseUrl)
		res, err := http.Get(licenseUrl)
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
			json.Unmarshal(data, &jsonData)
			log.Debug().Msgf("响应内容:%v", string(data))
			code, err := jsonpath.JsonPathLookup(jsonData, "$.code")
			if err != nil {
				log.Warn().Msgf("从响应体中读取code异常\n%v", err)
				return
			}
			c := int(code.(float64))
			if c == 200 || c == 0 {
				log.Info().Msgf("返回值code=%v,已被授权", code)
				errTimes = 0
				hasLic = true
			}
		}
	})
	cron.Start()
	log.Info().Msgf("授权校验插件初始化完成")
}

//Valid 函数则是我们需要在调用方显式查找的symbol
//export Valid
func Valid(req *http.Request, target []byte) error {
	if hasLic {
		return nil
	}

	err := &common.BusinessException{
		StatusCode: 403,
		Code:       1040403,
		Message:    "OS未授权，请联系管理员",
	}
	return err
}

func main() {
	//Need a main function to make CGO compile package as C shared library
}

// Package license OS鉴权插件
package main

import "C"
import (
	"encoding/json"
	"fmt"
	"github.com/robfig/cron"
	"github.com/rs/zerolog/log"
	"io"
	plugins "isc-route-service/plugins/common"
	"net/http"
	"time"
)

var lc *LicenseConf
var hasLic = true

type LicenseConf struct {
	LicenseHost string `json:"host"`
	LicenseUrl  string `json:"url"`
}

type LicenseResult struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// init 函数的目的是在插件模块加载的时候自动执行一些我们要做的事情，比如：自动将方法和类型注册到插件平台、输出插件信息等等。
func init() {
	log.Info().Msgf("授权校验插件初始化……")
	lc = &LicenseConf{
		LicenseHost: "isc-license-service:9013",
		LicenseUrl:  "/api/core/license/valid",
	}
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	plugins.ReadJsonToStruct("license/license.json", lc)

	//开启定时任务
	cron := cron.New()
	errTimes := 0
	cron.AddFunc("0 */1 * * * ?", func() {
		//获取license信息
		licenseUrl := fmt.Sprintf("http://%s%s", lc.LicenseHost, lc.LicenseUrl)
		log.Debug().Msgf("license请求地址:%s", licenseUrl)
		res, err := client.Get(licenseUrl)
		if err != nil {
			log.Warn().Msgf("license服务请求异常:%v", err)
			if errTimes > 10 {
				hasLic = false
			}
			errTimes += 1
			return
		}

		body := res.Body
		defer func() {
			if res != nil && body != nil {
				body.Close()
			}
		}()

		data, err := io.ReadAll(body)
		if err != nil || res.StatusCode != 200 {
			if errTimes > 10 {
				hasLic = false
			}
			errTimes += 1
			log.Warn().Msgf("读取license信息异常\n%v", err)
		} else {
			jsonData := LicenseResult{}
			log.Debug().Msgf("响应内容:%v", string(data))
			err = json.Unmarshal(data, &jsonData)
			if err != nil {
				log.Warn().Msgf("从响应体中读取code异常\n%v", err)
				return
			}

			c := jsonData.Code
			if c == 200 || c == 0 {
				log.Info().Msgf("返回值code=%v,已被授权", c)
				errTimes = 0
				hasLic = true
			} else {
				log.Warn().Msgf("授权结果异常：%v", jsonData)
				if errTimes > 10 {
					hasLic = false
				} else {
					errTimes += 1
				}
			}
		}
	})
	cron.Start()
	log.Info().Msgf("授权校验插件初始化完成")
}

//Valid 函数则是我们需要在调用方显式查找的symbol
//export Valid
func Valid() error {
	if hasLic {
		return nil
	}

	err := &plugins.BusinessException{
		StatusCode: 403,
		Code:       1040403,
		Message:    "OS未授权，请联系管理员",
	}
	return err
}

func main() {
	//Need a main function to make CGO compile package as C shared library
}

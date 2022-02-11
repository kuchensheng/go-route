// Package plugins 登陆鉴权
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/oliveagle/jsonpath"
	"github.com/rs/zerolog/log"
	"github.com/wxnacy/wgo/arrays"
	"io"
	"isc-route-service/pkg/domain"
	"isc-route-service/plugins"
	"isc-route-service/utils"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var Req *http.Request
var Params interface{}

var lc *LoginConf

type LoginConf struct {
	LoginUrl   string `json:"login_url"`
	StatusUrl  string `json:"status_url"`
	LogoutUrl  string `json:"logout_url"`
	AuthServer string `json:"auth_server"`
}

// init 函数的目的是在插件模块加载的时候自动执行一些我们要做的事情，比如：自动将方法和类型注册到插件平台、输出插件信息等等。
func init() {
	fmt.Println("鉴权拦截差价")
	pwd, _ := os.Getwd()
	file, err := os.OpenFile(filepath.Join(pwd, "login.json"), os.O_RDWR|os.O_SYNC, 0666)
	lc = &LoginConf{
		LoginUrl:   "/api/permission/auth/login",
		StatusUrl:  "/api/permission/auth/status",
		LogoutUrl:  "/api/permission/auth/logout",
		AuthServer: "isc-permission-service:32100",
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
	} else {
		log.Warn().Msgf("鉴权配置文件读取异常%v", err)
	}
}

//Login 函数则是我们需要在调用方显式查找的symbol
//export Login
func Login(args ...interface{}) error {
	Req := args[0].(*http.Request)
	p := *args[1].(*domain.RouteInfo)
	uri := Req.URL.Path
	if strings.EqualFold(uri, lc.LoginUrl) {
		//登陆uri不进行校验
		return nil
	} else if arrays.Contains(p.ExcludeUrl, uri) > 0 {
		//无需登录校验
		return nil
	} else {
		//路径匹配
		for _, p := range p.ExcludeUrl {
			if utils.Match(uri, p) {
				return nil
			}
		}
	}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s%s", lc.AuthServer, lc.StatusUrl), nil)
	if err != nil {
		log.Error().Msgf("登陆鉴权服务请求异常%v", err)
		return err
	}
	req.Header = Req.Header
	for _, cookie := range Req.Cookies() {
		req.AddCookie(cookie)
	}
	client := &http.Client{
		Timeout: time.Second * 5,
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Error().Msgf("登陆鉴权服务请求异常%v", err)
		return err
	}
	body := resp.Body
	defer resp.Body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	var jsonData interface{}
	json.Unmarshal(data, &jsonData)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		response := Req.Response
		if response != nil {
			response.StatusCode = 401
		}
		return &plugins.PluginError{
			StatusCode: strconv.Itoa(resp.StatusCode),
			Content:    jsonData,
		}
	}

	code, err := jsonpath.JsonPathLookup(jsonData, "$.code")
	if err != nil {
		return err
	}
	c := int(code.(float64))
	if c != 0 || c != 200 {
		return errors.New("鉴权失败")
	}
	return nil
}

func main() {
	//Need a main function to make CGO compile package as C shared library
}

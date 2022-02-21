// Package plugins 登陆鉴权
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	. "isc-route-service/plugins/common"
	"net/http"
	"strings"
	"time"
)

var Params interface{}

var lc *LoginConf

type LoginConf struct {
	LoginUrl   string `json:"login_url"`
	StatusUrl  string `json:"status_url"`
	LogoutUrl  string `json:"logout_url"`
	AuthServer string `json:"auth_server"`
}

type Status struct {
	Code int `json:"code"`
	Data struct {
		UserId    string   `json:"userId"`
		LoginName string   `json:"loginName"`
		RoleId    []string `json:"roleId"`
		NickName  string   `json:"nickname"`
		TenantId  string   `json:"tenantId"`
		UserType  string   `json:"userType"`
	} `json:"data"`
}

// init 函数的目的是在插件模块加载的时候自动执行一些我们要做的事情，比如：自动将方法和类型注册到插件平台、输出插件信息等等。
func init() {
	log.Info().Msgf("初始化登录鉴权插件")
	lc = &LoginConf{
		LoginUrl:   "/api/permission/auth/login",
		StatusUrl:  "/api/permission/auth/status",
		LogoutUrl:  "/api/permission/auth/logout",
		AuthServer: "isc-permission-service:32100",
	}
	ReadJsonToStruct("login/login.json", lc)
}
func contains(excludeUrls []string, uri string) bool {
	for _, item := range excludeUrls {
		if strings.EqualFold(item, uri) {
			return true
		}
	}
	return false
}

//Valid 函数则是我们需要在调用方显式查找的symbol
func Valid(Req *http.Request, target []byte) error {
	p := RouteInfo{}
	err := json.Unmarshal(target, &p)
	if err != nil {
		log.Error().Msgf("传输数据转换为targetRoute异常:%v", err)
		return err
	}
	uri := Req.URL.Path
	if strings.EqualFold(uri, lc.LoginUrl) || contains(p.ExcludeUrl, uri) {
		//登陆uri不进行校验
		return nil
	} else {
		//路径匹配
		for _, p := range p.ExcludeUrl {
			if Match(uri, p) {
				return nil
			}
		}
	}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s%s", lc.AuthServer, lc.StatusUrl), nil)
	if err != nil {
		log.Error().Msgf("登录鉴权服务请求异常%v", err)
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
		log.Error().Msgf("登录鉴权服务请求异常%v", err)
		return err
	}
	body := resp.Body
	defer resp.Body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	jsonData := &Status{}
	json.Unmarshal(data, jsonData)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		response := Req.Response
		if response != nil {
			response.StatusCode = 401
		}
		return &BusinessException{
			StatusCode: 401,
			Code:       1040401,
			Message:    "登录鉴权未通过",
			Data:       jsonData,
		}
	}

	c := jsonData.Code
	if c != 0 && c != 200 {
		return errors.New("鉴权失败")
	}
	//验证通过后，添加请求头
	Req.Header.Set("t-head-userId", jsonData.Data.UserId)
	Req.Header.Set("t-head-userName", jsonData.Data.LoginName)
	return nil
}

func main() {
	//Need a main function to make CGO compile package as C shared library
}
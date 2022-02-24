// Package plugins 登陆鉴权
package main

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	. "isc-route-service/plugins/common"
	"net/http"
	"strconv"
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
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		UserId     string   `json:"userId"`
		LoginName  string   `json:"loginName"`
		RoleId     []string `json:"roleId"`
		NickName   string   `json:"nickname"`
		TenantId   string   `json:"tenantId"`
		UserType   string   `json:"userType"`
		SuperAdmin bool     `json:"superAdmin"`
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
		} else {
			if Match(uri, item) {
				return true
			}
		}
	}
	return false
}

var req *http.Request
var client *http.Client

func initHttpRequest() error {
	if req == nil {
		if r, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s%s", lc.AuthServer, lc.StatusUrl), nil); err != nil {
			log.Error().Msgf("初始化异常%v", err)
			return &BusinessException{
				StatusCode: http.StatusInternalServerError,
				Code:       1040500,
				Message:    "登录鉴权服务请求异常,详情见isc-permission-service",
				Data:       err,
			}
		} else {
			req = r
		}
	}
	if client == nil {
		c := &http.Client{
			Timeout: time.Second * 5,
		}
		client = c
	}
	return nil
}

//Valid 函数则是我们需要在调用方显式查找的symbol
func Valid(Req *http.Request, target []byte) error {
	p := RouteInfo{}
	err := json.Unmarshal(target, &p)
	if err != nil {
		log.Error().Msgf("传输数据转换为targetRoute异常:%v", err)
		return &BusinessException{
			StatusCode: http.StatusInternalServerError,
			Code:       1040500,
			Message:    "传输数据转换为targetRoute异常",
			Data:       err,
		}
	}
	uri := Req.URL.Path
	if strings.EqualFold(uri, lc.LoginUrl) || contains(p.ExcludeUrls, uri) {
		//登陆uri不进行校验
		return nil
	}
	var req *http.Request
	//todo 这里是否需要使用缓存，有待商榷
	err = initHttpRequest()
	if err != nil {
		return err
	}
	req.Header = Req.Header
	req.Header.Set("Connection", "keep-alive")
	for _, cookie := range Req.Cookies() {
		req.AddCookie(cookie)
	}
	var resp *http.Response
	if resp, err = client.Do(req); err != nil {
		log.Error().Msgf("登录鉴权服务请求异常%v", err)
		return &BusinessException{
			StatusCode: http.StatusBadRequest,
			Code:       1040400,
			Message:    "登录鉴权服务请求异常,详情见isc-permission-service",
			Data:       err,
		}
	}

	body := resp.Body
	defer func() {
		if body != nil {
			body.Close()
		}
	}()
	data, err := io.ReadAll(body)
	if err != nil {
		return &BusinessException{
			StatusCode: http.StatusInternalServerError,
			Code:       1040500,
			Message:    "请求结果解析异常,isc-permission-service与isc-route-service解析规则不一致,请修改对应插件并升级服务",
			Data:       err,
		}
	}
	jsonData := &Status{}
	err = json.Unmarshal(data, jsonData)
	if err != nil {
		return &BusinessException{
			StatusCode: http.StatusInternalServerError,
			Code:       1040500,
			Message:    "请求结果解析异常,isc-permission-service与isc-route-service解析规则不一致,请修改对应插件并升级服务",
			Data:       err,
		}
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
			//Data:       jsonData,
		}
	}

	c := jsonData.Code
	m := jsonData.Message
	if m == "" {
		m = "登录鉴权失败"
	}
	if c != 0 && c != 200 {
		return &BusinessException{
			StatusCode: 401,
			Code:       1040400,
			Message:    m,
		}
	}
	//验证通过后，添加请求头
	Req.Header.Set("t-head-userId", jsonData.Data.UserId)
	Req.Header.Set("t-head-userName", jsonData.Data.LoginName)
	Req.Header.Set("isc-tenant-id", jsonData.Data.TenantId)
	Req.Header.Set("isc-tenant-admin", strconv.FormatBool(jsonData.Data.SuperAdmin))
	return nil
}

func main() {
	//Need a main function to make CGO compile package as C shared library
}

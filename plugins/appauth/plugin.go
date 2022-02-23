package main

import (
	"encoding/json"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog/log"
	. "isc-route-service/plugins/common"
	"net/http"
	"time"
)

var c *cache.Cache
var client http.Client
var ac *authConf

type authConf struct {
	Tenant struct {
		Address struct {
			Auth string `yaml:"auth"`
			Url  string `yaml:"url"`
		} `yaml:"address"`
	} `yaml:"tenant"`
}
type AuthOneData struct {
	AppCode     string `json:"appCode"`
	Type        int    `json:"type"`
	RelatedId   string `json:"relatedId"`
	RelatedInfo string `json:"relatedInfo"`
	ParentId    int    `json:"parentId"`
	Granted     bool   `json:"granted"`
	IsDefault   bool   `json:"isDefault"`
}
type AuthOne struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    AuthOneData
}

func init() {
	c = cache.New(5*time.Minute, 10*time.Minute)
	client = http.Client{
		Timeout: 3 * time.Second,
	}
	a := &authConf{
		Tenant: struct {
			Address struct {
				Auth string `yaml:"auth"`
				Url  string `yaml:"url"`
			} `yaml:"address"`
		}{
			Address: struct {
				Auth string `yaml:"auth"`
				Url  string `yaml:"url"`
			}{
				"http://isc-authorization-service:9033",
				"/api/core/authorization/one",
			},
		},
	}
	if err := ReadJsonToStruct("appauth/conf.json", a); err != nil {
		log.Error().Msgf("应用授权初始化失败，将使用默认数据进行初始化，%v", a)
	}
	ac = a
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
	//超级管理员放过
	if isSuperAdmin(Req) {
		return nil
	}
	//从请求头中获取tenantId
	tenantId := Req.Header.Get("isc-tenant-id")
	forbiddenError := &BusinessException{
		StatusCode: 403,
		Code:       1040403,
		Message:    "应用无权限访问",
	}
	if tenantId == "" {
		return forbiddenError
	} else {
		g, found := c.Get(p.AppCode)
		if !found {
			resp, err := client.Get(ac.Tenant.Address.Auth + ac.Tenant.Address.Url)
			if err != nil {
				log.Error().Msgf("请求鉴权服务异常:%v", err)
			} else {
				one := AuthOne{}
				err = ReadResp(resp, one)
				if err != nil {
					log.Error().Msgf("%v", err)
				} else {
					granted := one.Data.Granted
					if !granted {
						c.Set(p.AppCode, granted, time.Minute*1)
						return err
					}
					c.Set(p.AppCode, granted, time.Minute*5)
				}
			}
		} else if g.(bool) {
			return nil
		} else {
			return err
		}
	}
	return nil
}

func isSuperAdmin(Req *http.Request) bool {
	superAdmin := Req.Header.Get("isc-tenant-admin")
	return superAdmin == "" || superAdmin == "true"
}

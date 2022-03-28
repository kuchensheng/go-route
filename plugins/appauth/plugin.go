package main

import "C"
import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog/log"
	. "isc-route-service/plugins/common"
	"net/http"
	"time"
	"unsafe"
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
	if err := ReadYamlToStruct("appauth/conf.yml", a); err != nil {
		log.Error().Msgf("应用授权初始化失败，将使用默认数据进行初始化，%v", a)
	}
	ac = a
}

//Valid 函数则是我们需要在调用方显式查找的symbol
//export Valid
func Valid(r *C.int, t []C.char) error {
	Req := (*http.Request)(unsafe.Pointer(r))
	target := *(*[]byte)(unsafe.Pointer(&t))
	log.Debug().Msg("应用授权校验")
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
		log.Debug().Msg("应用授权校验，是超级管理员直接放过")
		return nil
	}
	//从请求头中获取tenantId
	tenantId := Req.Header.Get("isc-tenant-id")
	forbiddenError := &BusinessException{
		StatusCode: 403,
		Code:       1040403,
		Message:    "应用无权限访问",
	}
	//log.Info().Msgf("租户Id=%s",tenantId)
	if tenantId == "" {
		return forbiddenError
	} else {
		g, found := c.Get(p.AppCode)
		log.Debug().Msgf("从缓存中获取结果:%v", g)
		if !found {
			reqBody := fmt.Sprintf(`{"appCode":"%s","type":1,"relatedId":"%s"}`, p.AppCode, tenantId)
			resp, err := client.Post(ac.Tenant.Address.Auth+ac.Tenant.Address.Url, "application/json", bytes.NewBufferString(reqBody))
			if err != nil {
				log.Error().Msgf("请求鉴权服务异常:%v", err)
			} else {
				one := AuthOne{}
				err = ReadResp(resp, one)
				if err != nil {
					log.Error().Msgf("%v", err)
				} else {
					log.Info().Msgf("从响应体取到结果:%v", one)
					granted := one.Data.Granted
					if !granted {
						c.Set(p.AppCode+"_"+tenantId, granted, time.Minute*1)
						return forbiddenError
					}
					c.Set(p.AppCode+"_"+tenantId, granted, time.Minute*5)
				}
			}
		} else if g.(bool) {
			return nil
		} else {
			return forbiddenError
		}
	}
	return nil
}

func isSuperAdmin(Req *http.Request) bool {
	superAdmin := Req.Header.Get("isc-tenant-admin")
	return superAdmin == "" || superAdmin == "true"
}

func main() {

}

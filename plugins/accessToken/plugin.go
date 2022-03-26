//go:build (linux && cgo) || (darwin && cgo) || (freebsd && cgo)
// +build linux,cgo darwin,cgo freebsd,cgo

package main

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
	plugins "isc-route-service/plugins/common"
	"net/http"
)

var IscAccessTokenKey = "isc-access-token"
var GatewayApiServiceAccessTokenRedisKeyPrefix = "gateway:api:service:access:token:"
var ac = &AccessTokeConf{}

type AccessTokeConf struct {
	AccessToken struct {
		urls []string `yaml:"urls"`
	} `yaml:"access-token"`
}

var RedisClient redis.Client

func init() {
	//这里做初始化操作
	plugins.ReadYamlToStruct("accessToken/conf.yml", ac)
	if len(ac.AccessToken.urls) == 0 {
		ac.AccessToken.urls = []string{"/api/common"}
	}
	RedisClient = *plugins.InitRedisClient("accessToken/conf.yml")
}

//Valid access token 验证
func Valid(req *http.Request, target []byte) error {
	uri := req.URL.Path
	if !plugins.IsInSlice(ac.AccessToken.urls, uri) {
		return nil
	}
	iscAccessToken := req.Header.Get(IscAccessTokenKey)
	log.Info().Msgf("公共服务请求,isc-access-token=%s", iscAccessToken)
	err := &plugins.BusinessException{
		StatusCode: 403,
		Code:       1040403,
		Message:    "应用无权限访问",
	}
	if iscAccessToken == "" {
		return err
	} else {
		//todo 需要连接redis
		value := RedisClient.Get(context.Background(), fmt.Sprintf("%s%s", GatewayApiServiceAccessTokenRedisKeyPrefix, iscAccessToken)).Val()
		if value == "" {
			return err
		}
	}
	return nil

}

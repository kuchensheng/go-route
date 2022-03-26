package main

import (
	"github.com/go-redis/redis/v8"
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

var redisClient redis.Client

func init() {
	//这里做初始化操作
	plugins.ReadYamlToStruct("accessToken/conf.yml", ac)
	if len(ac.AccessToken.urls) == 0 {
		ac.AccessToken.urls = []string{"/api/common"}
	}
	redisClient = *plugins.InitRedisClient("accessToken/conf.yml")
}

type PluginIntf interface {
	valid(req *http.Request, target []byte) error
}

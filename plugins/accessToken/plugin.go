package main

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
	plugins "go.mod/common"
	"net/http"
)

var IscAccessTokenKey = "isc-access-token"
var GatewayApiServiceAccessTokenRedisKeyPrefix = "gateway:api:service:access:token:"
var ac = &accessTokeConf{
	Urls: []string{"/api/common"},
}

type accessTokeConf struct {
	Urls []string `json:"urls"`
}

var redisClient redis.Client

func init() {
	//这里做初始化操作
	plugins.ReadJsonToStruct("license.json", ac)
	plugins.InitRedisClient("conf.yaml")
}

//Valid access token 验证
//export Valid
func Valid(req *http.Request, target []byte) error {
	uri := req.URL.Path
	if !plugins.IsInSlice(ac.Urls, uri) {
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
		value := redisClient.Get(context.Background(), fmt.Sprintf("%s%s", GatewayApiServiceAccessTokenRedisKeyPrefix, iscAccessToken)).Val()
		if value == "" {
			return err
		}
	}
	return nil

}

//go:build windows

package main

import (
	"C"
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
	plugins "isc-route-service/plugins/common"
	"net/http"
	"unsafe"
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

type WindowPlugin struct {
	redis.Client
}

var p = WindowPlugin{}

//Valid access token 验证
//export Valid
func Valid(r *C.int, t []C.char) error {
	req := (*http.Request)(unsafe.Pointer(r))
	fmt.Println(req, t)
	target := *(*[]byte)(unsafe.Pointer(&t))
	return p.Handler(req, target)
}

func main() {
	//h := make(map[string][]string)
	//h["isc-api-version"] = []string{"3.0"}
	//r := http.Request{
	//	Header: h,
	//	URL: &url.URL{
	//		Scheme: "http",
	//		Host:   "www.techcrunch.com",
	//		Path:   "/api/common/test",
	//	},
	//}
	//ptr := &r
	//println("指针地址",ptr,reflect.TypeOf(ptr).String())
	//fmt.Println(*ptr)
	//c := []C.char{'1','2','a'}
	//intPtr := unsafe.Pointer(ptr)
	//addr := (*C.int)(intPtr)
	//println("addr ",addr)
	//Valid(addr,c)
}

func (a WindowPlugin) Handler(req *http.Request, target []byte) error {
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

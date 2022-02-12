package main

import (
	"github.com/rs/zerolog/log"
	"isc-route-service/pkg/exception"
	"isc-route-service/utils"
	"net/http"
	"os"
	"path/filepath"
)

var IscAccessTokenKey = "isc-access-token"

var ac = &accessTokeConf{
	Urls: []string{"/api/common"},
}

type accessTokeConf struct {
	Urls []string `json:"urls"`
}

func init() {
	//这里做初始化操作
	pwd, _ := os.Getwd()
	fp := filepath.Join(pwd, "license.json")
	utils.OpenFileAndUnmarshal(fp, ac)
}

//Valid access token 验证
func Valid(args ...interface{}) error {
	req := args[0].(*http.Request)
	uri := req.URL.Path
	if !utils.IsInSlice(ac.Urls, uri) {
		return nil
	}
	iscAccessToken := req.Header.Get(IscAccessTokenKey)
	log.Info().Msgf("公共服务请求,isc-access-token=%s", iscAccessToken)
	if iscAccessToken == "" {
		return &exception.BusinessException{
			StatusCode: 403,
			Code:       1040403,
			Message:    "应用无权限访问",
		}
	} else {
		//todo 需要连接redis
	}
	return nil

}

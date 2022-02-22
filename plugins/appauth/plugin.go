package main

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
	. "isc-route-service/plugins/common"
	"net/http"
)

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
	//从请求头中获取
	return nil
}

//go:build (linux && cgo) || (darwin && cgo) || (freebsd && cgo)
// +build linux,cgo darwin,cgo freebsd,cgo

package middleware

import (
	"C"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"isc-route-service/pkg/domain"
	"net/http"
	"unsafe"
)

//MiddleWare 全局拦截器
func MiddleWare() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Header.Get("t-header")
	}
}

//PrepareMiddleWare 前置拦截处理
func PrepareMiddleWare(c *gin.Context, plugins []domain.PluginPointer) error {
	//处理器按照order字段已排序
	for _, pp := range plugins {
		var method = pp.Symbol
		//方法执行
		var runtimeError error
		if pp.Args < 0 {
			runtimeError = method.(func() error)()
			if x := recover(); x != nil {
				runtimeError = x.(error)
			}
		} else {
			data, _ := json.Marshal(pp.RouteInfo)
			r := (*C.int)(unsafe.Pointer(c.Request))
			t := ([]C.char)(unsafe.Pointer(&data))
			runtimeError = method.(func(*C.int, []C.char) error)(r, t)
			if x := recover(); x != nil {
				runtimeError = x.(error)
			}
		}

		if runtimeError != nil && runtimeError.Error() != "" {
			log.Warn().Msgf("插件[%s]方法[%s]执行异常,%v\n 请求路径:[%s]", pp.Name, pp.Method, runtimeError, c.Request.URL.String())
			return runtimeError
		}
	}

	defer func() error {
		if x := recover(); x != nil {
			return x.(error)
		}
		return nil
	}()
	return nil
}

// PostMiddleWare 后置拦截处理
func PostMiddleWare() error {
	return nil
}

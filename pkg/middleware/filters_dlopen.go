//go:build (linux && cgo) || (darwin && cgo) || (freebsd && cgo)
// +build linux,cgo darwin,cgo freebsd,cgo

package middleware

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"isc-route-service/pkg/domain"
	"net/http"
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
		//变量赋值
		p := pp.Plugin
		//log.Info().Msgf("执行插件[%s]方法[%s]", pp.Name, pp.Method)
		method, err := p.Lookup(pp.Method)
		if err != nil {
			log.Warn().Msgf("插件[%s]方法[%s]读取异常,%v", pp.Name, pp.Method, err)
			continue
		}
		//方法执行
		var runtimeError error
		if pp.Args < 0 {
			runtimeError = method.(func() error)()
		} else {
			data, _ := json.Marshal(pp.RouteInfo)
			runtimeError = method.(func(*http.Request, []byte) error)(c.Request, data)
		}

		if runtimeError != nil && runtimeError.Error() != "" {
			log.Warn().Msgf("插件[%s]方法[%s]执行异常,%v", pp.Name, pp.Method, runtimeError)
			return runtimeError
		}
	}

	return nil
}

// PostMiddleWare 后置拦截处理
func PostMiddleWare() error {
	return nil
}

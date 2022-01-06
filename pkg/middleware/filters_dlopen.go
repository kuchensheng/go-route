//go:build (linux && cgo) || (darwin && cgo) || (freebsd && cgo)
// +build linux,cgo darwin,cgo freebsd,cgo

package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"isc-route-service/pkg/domain"
	"net/http"
)

// MiddleWare 全局拦截器
func middleWare() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Header.Get("t-header")
	}
}

// PrepareMiddleWare 前置拦截处理
func prepareMiddleWare(c *gin.Context, plugins []*domain.PluginPointer) error {
	//处理器按照order字段已排序
	for _, pp := range plugins {
		//变量赋值
		p := pp.Plugin
		pi := pp.PI
		reqVar, err := p.Lookup("Req")
		if err != nil {
			log.Error().Msgf("插件[%s]变量[%s]读取异常,%v", pi.Name, "Req", err)
			continue
		}
		w, err := p.Lookup("W")
		if err != nil {
			log.Error().Msgf("插件[%s]变量[%s]读取异常,%v", pi.Name, "W", err)
			continue
		}
		*reqVar.(*http.Request) = *c.Request
		*w.(*http.Response) = *c.Request.Response
		method, err := p.Lookup(pi.Method)
		if err != nil {
			log.Error().Msgf("插件[%s]方法[%s]读取异常,%v", pi.Name, pi.Method, err)
			continue
		}
		//方法执行
		runtimeError := method.(func() error)()
		if runtimeError != nil {
			log.Warn().Msgf("插件[%s]方法[%s]执行异常,%v", pi.Name, pi.Method, err)
			return runtimeError
		}
	}

	return nil
}

// PostMiddleWare 后置拦截处理
func postMiddleWare() error {
	return nil
}

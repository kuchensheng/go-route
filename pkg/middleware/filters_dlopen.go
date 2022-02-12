//go:build (linux && cgo) || (darwin && cgo) || (freebsd && cgo)
// +build linux,cgo darwin,cgo freebsd,cgo

package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"isc-route-service/pkg/domain"
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
		//reqVar, err := p.Lookup("Req")
		//if err != nil {
		//	log.Warn().Msgf("插件[%s]变量[%s]读取异常,%v", pp.Name, "Req", err)
		//} else {
		//	*reqVar.(*http.Request) = *c.Request
		//}
		//方法寻找
		//params, err := p.Lookup("Params")
		//if err != nil {
		//	log.Warn().Msgf("插件[%s]变量读取异常,%v", pp.Name, "Params", err)
		//} else {
		//	*params.(*interface{}) = pp.RouteInfo
		//}

		method, err := p.Lookup(pp.Method)
		if err != nil {
			log.Warn().Msgf("插件[%s]方法[%s]读取异常,%v", pp.Name, pp.Method, err)
			continue
		}
		//方法执行
		err = func() error {
			runtimeError := method.(func(args ...interface{}) error)(c.Request, pp.RouteInfo)
			if runtimeError != nil && runtimeError.Error() != "" {
				log.Warn().Msgf("插件[%s]方法[%s]执行异常,%v", pp.Name, pp.Method, err)
				return runtimeError
			}
			return nil
		}()
		if e := recover(); e != nil {
			return err
		}
	}

	return nil
}

// PostMiddleWare 后置拦截处理
func postMiddleWare() error {
	return nil
}

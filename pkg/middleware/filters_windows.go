//go:build windows

package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"isc-route-service/pkg/domain"
	"strings"
	"syscall"
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
		pi := pp.PI
		reqVar := pp.Symbol
		//方法执行
		_, _, runtimeError := syscall.Syscall(reqVar.(uintptr), 0, 0, 0, 0)
		if !strings.Contains(runtimeError.Error(), "success") {
			log.Warn().Msgf("插件[%s]方法[%s]执行异常,%v", pi.Name, pi.Method, runtimeError.Error())
			return runtimeError
		}
	}

	return nil
}

// PostMiddleWare 后置拦截处理
func postMiddleWare() error {
	return nil
}

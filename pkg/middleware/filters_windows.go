//go:build windows

package middleware

import (
	"github.com/gin-gonic/gin"
	"isc-route-service/pkg/domain"
)

//MiddleWare 全局拦截器
func MiddleWare() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Header.Get("t-header")
	}
}

//PrepareMiddleWare 前置拦截处理
func PrepareMiddleWare(c *gin.Context, plugins []domain.PluginPointer) error {
	return nil
}

// PostMiddleWare 后置拦截处理
func PostMiddleWare() error {
	return nil
}

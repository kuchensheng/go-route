package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"plugin"
)

var plugsin []*plugin.Plugin

// MiddleWare 全局拦截器
func MiddleWare() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Header.Get("t-header")
	}
}

// PrepareMiddleWare 前置拦截处理
func PrepareMiddleWare() error {
	//todo 设计处理器链

	return fmt.Errorf("处理异常了")
}

// PostMiddleWare 后置拦截处理
func PostMiddleWare() error {
	return nil
}

package middleware

import (
	"github.com/gin-gonic/gin"
	"isc-route-service/pkg/domain"
)

// MiddleWare 全局拦截器
func MiddleWare() gin.HandlerFunc {
	return middleWare()
}

// PrepareMiddleWare 前置拦截处理
func PrepareMiddleWare(c *gin.Context, plugins []domain.PluginPointer) error {
	//处理器按照order字段已排序
	return prepareMiddleWare(c, plugins)
}

// PostMiddleWare 后置拦截处理
func PostMiddleWare() error {
	return postMiddleWare()
}

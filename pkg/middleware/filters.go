package middleware

import "github.com/gin-gonic/gin"

// MiddleWare todo 这里是个业务请求链
func MiddleWare() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Header.Get("t-header")
	}
}

// MiddleWare todo 这里是个请求前的业务逻辑
func PrepareMiddleWare() gin.HandlerFunc {
	return func(context *gin.Context) {
		context.Abort()
	}
}

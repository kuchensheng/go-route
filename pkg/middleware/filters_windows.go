//go:build windows

package middleware

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"isc-route-service/pkg/domain"
	"syscall"
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
	for _, plugin := range plugins {
		symbol := (plugin.Symbol).(*syscall.LazyProc)
		data, _ := json.Marshal(plugin.RouteInfo)
		ret, _, callErr := symbol.Call(uintptr(unsafe.Pointer(c.Request)), uintptr(unsafe.Pointer(&data)))
		if ret != 0 {
			return *(*error)(unsafe.Pointer(ret))
		}
		if callErr != nil {
			return callErr
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

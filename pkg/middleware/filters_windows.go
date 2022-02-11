//go:build windows

package middleware

import "C"
import (
	"github.com/gin-gonic/gin"
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
	//for _, pp := range plugins {
	//	//reqVar := pp.Symbol
	//	log.Info().Msgf("执行插件[%s]", pp.Name)
	//	//方法执行
	//	proc := (*syscall.Proc)(unsafe.Pointer(pp.Symbol))
	//	r, _, err := proc.Call(uintptr(unsafe.Pointer(c.Request))) //syscall.Syscall(reqVar, 0, 0, 0, 0)
	//	if err != nil &&  err.(syscall.Errno) != 0 {
	//		return errors.New(fmt.Sprintf("插件:[%s]方法:[%s]执行异常,%v",pp.Name,pp.Method,err))
	//	} else {
	//		result := (*C.char)(unsafe.Pointer(r))
	//		s := C.GoString(result)
	//		log.Warn().Msgf("插件[%s]方法[%s]执行异常", pp.Name, pp.Method)
	//		return errors.New(s)
	//	}
	//}

	return nil
}

// PostMiddleWare 后置拦截处理
func postMiddleWare() error {
	return nil
}

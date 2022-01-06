//go:build windows

// Package plugins +build windows
package main

import "C"
import (
	"fmt"
)

//var req *http.Request
//var w *http.Response

// init 函数的目的是在插件模块加载的时候自动执行一些我们要做的事情，比如：自动将方法和类型注册到插件平台、输出插件信息等等。
func init() {
	fmt.Println("word-我是库陈胜")
}

//Hello 函数则是我们需要在调用方显式查找的symbol
//export Hello
func Hello() {
	fmt.Println("result=我是库陈胜")
	//return uintptr(0), uintptr(0), syscall.Errno(0)
}

//func SetReq(r *http.Request) {
//	req = r
//	w = r.Response
//}

func main() {
	//Need a main function to make CGO compile package as C shared library
}

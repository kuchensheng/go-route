//go:build windows

// Package plugins +build windows
package main

import "C"
import (
	"fmt"
)

// init 函数的目的是在插件模块加载的时候自动执行一些我们要做的事情，比如：自动将方法和类型注册到插件平台、输出插件信息等等。
func init() {
	fmt.Println("word-我是库陈胜，我是插件,我初始化了")
}

//Hello 函数则是我们需要在调用方显式查找的symbol,
//返回值 int 0表示成功，否则表示失败
//export Hello
func Hello(req *int) int {
	print("我是酷达舒")
	return 0
}

func main() {
	//Need a main function to make CGO compile package as C shared library
}

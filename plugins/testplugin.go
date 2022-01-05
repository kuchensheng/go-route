//必须是main包
//test plugin.go
package main

import "fmt"

// init 函数的目的是在插件模块加载的时候自动执行一些我们要做的事情，比如：自动将方法和类型注册到插件平台、输出插件信息等等。
func init() {
	fmt.Println("word")
}

//Hello 函数则是我们需要在调用方显式查找的symbol
func Hello() {
	fmt.Println("hello")
}

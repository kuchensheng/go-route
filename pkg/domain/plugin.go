package domain

import (
	"plugin"
)

var Plugins []PluginInfo

var PrePlugins []*PluginPointer
var PostPlugsins []*PluginPointer
var OtherPlugins []*PluginPointer

//PluginConfigPath 插件定义文件地址
var PluginConfigPath string

type PluginInfo struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Order  int    `json:"order"`
	Method string `json:"method"`
	Type   int    `json:"type"`
}

type PluginPointer struct {
	PI     PluginInfo
	Plugin *plugin.Plugin
	Symbol plugin.Symbol
}

const (
	PRE  = iota //0
	POST        //1
)

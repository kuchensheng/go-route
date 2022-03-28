//go:build windows

// Package domain +build windows
package domain

import (
	"syscall"
)

//initPlugins 加载插件
func openPlugin(pluginInfo *PluginInfo) (*PluginPointer, error) {
	//dll, err := syscall.LoadDLL(pluginInfo.Path)
	dll, err := syscall.LoadDLL(pluginInfo.AbsolutePath)
	if err != nil {
		return nil, err
	}
	proc, err := dll.FindProc(pluginInfo.Method)
	if err != nil {
		return nil, err
	}
	pp := &PluginPointer{
		Symbol: proc,
		Type:   1,
	}
	pp.PluginInfo = *pluginInfo
	return pp, nil
}

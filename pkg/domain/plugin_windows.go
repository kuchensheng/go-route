//go:build windows

// Package domain +build windows
package domain

import (
	"errors"
	"strings"
	"syscall"
)

//initPlugins 加载插件
func openPlugin(pluginInfo *PluginInfo) (*PluginPointer, error) {
	path := pluginInfo.Path
	if !strings.HasSuffix(path, "dll") {
		return nil, errors.New("动态链接库必须以dll结尾")
	}
	//dll, err := syscall.LoadDLL(pluginInfo.Path)
	dll := syscall.NewLazyDLL(pluginInfo.Path)
	proc := dll.NewProc(pluginInfo.Method)
	pp := &PluginPointer{
		Symbol: proc,
		Type:   1,
	}
	pp.PluginInfo = pluginInfo
	return pp, nil
}

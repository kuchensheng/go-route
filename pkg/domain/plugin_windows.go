//go:build windows

// Package domain +build windows
package domain

import (
	"errors"
	"strings"
	"syscall"
	"unsafe"
)

//initPlugins 加载插件
func openPlugin(pluginInfo *PluginInfo) (*PluginPointer, error) {
	path := pluginInfo.Path
	if !strings.HasSuffix(path, "dll") {
		return nil, errors.New("动态链接库必须以dll结尾")
	}
	dll, err := syscall.LoadDLL(pluginInfo.Path)
	if err != nil {
		return nil, err
	}
	proc, err := dll.FindProc(pluginInfo.Method)
	if err != nil {
		return nil, err
	}
	pp := &PluginPointer{
		PI:     *pluginInfo,
		Symbol: uintptr(unsafe.Pointer(proc)),
	}
	return pp, nil
}

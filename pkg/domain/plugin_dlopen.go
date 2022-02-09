//go:build (linux && cgo) || (darwin && cgo) || (freebsd && cgo)
// +build linux,cgo darwin,cgo freebsd,cgo

package domain

import (
	"plugin"
)

//initPlugins 加载插件
func openPlugin(pluginInfo *PluginInfo) (*PluginPointer, error) {
	p, err := plugin.Open(pluginInfo.Path)
	if err != nil {
		return nil, err
	}
	symbol, err := p.Lookup(pluginInfo.Method)
	if err != nil {
		return nil, err
	}
	pp := &PluginPointer{
		Plugin: p,
	}
	pp.PluginInfo = pluginInfo
	pp.Symbol = symbol
	return pp, nil
}

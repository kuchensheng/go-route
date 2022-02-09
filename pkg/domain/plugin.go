package domain

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"os"
	"path/filepath"
	"plugin"
	"sort"
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
	Symbol uintptr
	*RouteInfo
}

const (
	PRE  = iota //0
	POST        //1
)

//InitPlugins 加载插件
func InitPlugins() {
	if PluginConfigPath == "" {
		wd, _ := os.Getwd()
		fp := filepath.Join(wd, "resources", "plugins.json")
		if _, err := os.Stat(fp); os.IsNotExist(err) {
			log.Fatal().Msg("路由配置文件不存在")
		} else {
			PluginConfigPath = fp
		}
	}
	pluginData, err := ioutil.ReadFile(PluginConfigPath)
	if err != nil {
		log.Fatal().Msgf("插件文件加载异常", err)
	}
	err = json.Unmarshal(pluginData, &Plugins)
	if err != nil {
		log.Fatal().Msgf("插件文件解析异常", err)
	}
	//todo 需要根据t'y'pe 进行分类处理
	sort.SliceIsSorted(Plugins, func(i, j int) bool {
		return Plugins[i].Order < Plugins[j].Order
	})
	for _, pluginInfo := range Plugins {
		log.Info().Msgf("加载插件[%s]", pluginInfo.Name)
		//判断文件是否存在
		if _, err := os.Stat(pluginInfo.Path); os.IsNotExist(err) {
			log.Warn().Msgf("插件[%s]文件找不到%v", pluginInfo.Name, err)
			continue
		}
		pp, err := openPlugin(&pluginInfo)
		if err != nil {
			log.Error().Msgf("插件[%s]加载异常,%v", pluginInfo.Name, err)
		} else {
			//按照分类放入list，以待执行
			switch pluginInfo.Type {
			case PRE:
				PrePlugins = append(PrePlugins, pp)
			case POST:
				PostPlugsins = append(PostPlugsins, pp)
			default:
				//
				OtherPlugins = append(OtherPlugins, pp)
			}
		}
	}
	//todo 监听插件配置文件/数据变化
}

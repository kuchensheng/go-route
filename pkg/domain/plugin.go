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

//PluginConfigPath 插件定义文件地址
var PluginConfigPath string

type PluginInfo struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Order  int    `json:"order"`
	Method string `json:"method"`
	Type   int    `json:"type"`
}

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
		p, err := plugin.Open(pluginInfo.Path)
		if err != nil {
			log.Error().Msgf("插件[%s]加载异常", "")
		} else {
			p.Lookup(pluginInfo.Method)
		}
	}

	//todo 监听插件配置文件/数据变化
}

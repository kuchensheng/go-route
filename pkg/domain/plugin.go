package domain

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"isc-route-service/watcher"
	"os"
	"path/filepath"
	"plugin"
	"runtime"
	"sort"
)

var Plugins []PluginInfo

var PrePlugins []PluginPointer
var PostPlugsins []PluginPointer
var OtherPlugins []PluginPointer

//PluginConfigPath 动态链接库定义文件地址
var PluginConfigPath string

type PluginInfo struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Order  int    `json:"order"`
	Method string `json:"method"`
	Type   int    `json:"type"`
	Args   int    `json:"args"`
}

type PluginPointer struct {
	*PluginInfo
	Plugin *plugin.Plugin
	Symbol interface{}
	Type   int
	*RouteInfo
}

const (
	PRE  = iota //0
	POST        //1
)

//InitPlugins 加载动态链接库
func InitPlugins() {
	log.Info().Msg("加载动态链接库信息")
	wd, _ := os.Getwd()
	if PluginConfigPath == "" {
		fp := filepath.Join(wd, "resources", "plugins.json")
		if _, err := os.Stat(fp); os.IsNotExist(err) {
			log.Fatal().Msg("动态链接库配置文件不存在")
		} else {
			PluginConfigPath = fp
		}
	}

	handler := func(fp string) {
		pluginData, err := ioutil.ReadFile(fp)
		if err != nil {
			log.Fatal().Msgf("动态链接库文件加载异常", err)
		}
		err = json.Unmarshal(pluginData, &Plugins)
		if err != nil {
			log.Fatal().Msgf("动态链接库文件解析异常", err)
		}
		//todo 需要根据t'y'pe 进行分类处理
		sort.Slice(Plugins, func(i, j int) bool {
			return Plugins[i].Order > Plugins[j].Order
		})
		for _, pluginInfo := range Plugins {
			log.Info().Msgf("加载动态链接库[%s]", pluginInfo.Name)
			pluginPath := pluginInfo.Path
			if runtime.GOOS != "windows" && PluginConfigPath[0] == os.PathSeparator {
				//绝对路径
			} else {
				pluginPath = filepath.Join(wd, "plugins", pluginPath)
			}
			//判断文件是否存在
			if _, err := os.Stat(pluginInfo.Path); os.IsNotExist(err) {
				log.Warn().Msgf("动态链接库[%s]文件找不到%v", pluginInfo.Name, err)
				continue
			}
			pp, err := openPlugin(&pluginInfo)
			if err != nil {
				log.Error().Msgf("动态链接库[%s]加载异常,%v", pluginInfo.Name, err)
			} else {
				//按照分类放入list，以待执行
				switch pluginInfo.Type {
				case PRE:
					PrePlugins = append(PrePlugins, *pp)
				case POST:
					PostPlugsins = append(PostPlugsins, *pp)
				default:
					//
					OtherPlugins = append(OtherPlugins, *pp)
				}
			}
		}
	}
	handler(PluginConfigPath)
	//todo 监听动态链接库配置文件/数据变化
	go func() {
		watcher.AddWatcher(PluginConfigPath, handler)
	}()
}

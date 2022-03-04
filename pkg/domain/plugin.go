package domain

//Package domain's plugin provide the ability to initialize plugins.
//Note: At present,it must run on Linux operating system
//domain's plugin will
import (
	"encoding/json"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"isc-route-service/watcher"
	"os"
	"path/filepath"
	"plugin"
	"sort"
	"strings"
)

//PrePlugins returns all plugins before proxy request
var PrePlugins []PluginPointer

//PostPlugsins returns all plugins after proxy request
var PostPlugsins []PluginPointer

var OtherPlugins []PluginPointer

var AllPlugins []PluginInfo

//PluginConfigPath 动态链接库定义文件地址
var PluginConfigPath string

//PluginInfo 插件配置信息
type PluginInfo struct {
	Name         string `json:"name"`
	Path         string `json:"path"`
	AbsolutePath string
	Order        int    `json:"order"`
	Method       string `json:"method"`
	Type         int    `json:"type"`
	Args         int    `json:"args"`
	Version      string `json:"version"`
}

//PluginPointer 插件信息与路由规则组合
type PluginPointer struct {
	PluginInfo
	Plugin plugin.Plugin
	Symbol interface{}
	Type   int
	RouteInfo
}

const (
	PRE  = iota //0
	POST        //1
)

var alreadyExistsPlugins map[string]PluginInfo

//InitPlugins 加载动态链接库
func InitPlugins() {
	alreadyExistsPlugins = make(map[string]PluginInfo)
	log.Info().Msg("加载动态链接库信息")
	wd, _ := os.Getwd()
	if PluginConfigPath == "" {
		fp := filepath.Join(wd, "data", "resources", "plugins.json")
		if _, err := os.Stat(fp); os.IsNotExist(err) {
			log.Fatal().Msg("动态链接库配置文件不存在")
		} else {
			PluginConfigPath = fp
		}
	}

	handler := func(fp string) {
		var Plugins []PluginInfo
		pluginData, err := ioutil.ReadFile(fp)
		if err != nil {
			log.Error().Msgf("动态链接库文件加载异常", err)
			return
		}
		err = json.Unmarshal(pluginData, &Plugins)
		if err != nil {
			log.Error().Msgf("动态链接库文件解析异常", err)
			return
		}
		//todo 需要根据t'y'pe 进行分类处理
		sort.Slice(Plugins, func(i, j int) bool {
			return Plugins[i].Order < Plugins[j].Order
		})
		newPlugin := make(map[string]PluginInfo)
		for _, item := range Plugins {
			newPlugin[strings.Join([]string{item.Name, item.Version}, "_")] = item
		}
		var newPrePlugin []PluginPointer
		var newPostPlugin []PluginPointer
		var newOtherPlugin []PluginPointer
		for key, pluginInfo := range newPlugin {
			if _, ok := alreadyExistsPlugins[key]; ok {
				//已存在的插件，不再重新加载
				continue
			}
			log.Info().Msgf("加载动态链接库[%s]", pluginInfo.Name)
			pluginPath := filepath.Join(wd, "data", "plugins", pluginInfo.Path)
			//判断文件是否存在
			if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
				log.Warn().Msgf("动态链接库[%s]文件找不到[%s]%v", pluginInfo.Name, pluginPath, err)
				continue
			}
			pluginInfo.AbsolutePath = pluginPath
			pp, err := openPlugin(&pluginInfo)
			if err != nil {
				log.Error().Stack().Msgf("动态链接库[%s]加载异常,%v", pluginInfo.Name, err)
				continue
			} else {
				AllPlugins = append(AllPlugins, pluginInfo)
				//按照分类放入list，以待执行
				switch pluginInfo.Type {
				case PRE:
					newPrePlugin = append(newPrePlugin, *pp)
				case POST:
					newPostPlugin = append(newPostPlugin, *pp)
				default:
					newOtherPlugin = append(newOtherPlugin, *pp)
				}
				alreadyExistsPlugins[key] = pluginInfo
			}
		}
		PrePlugins = newPrePlugin
		PostPlugsins = newPostPlugin
		OtherPlugins = newOtherPlugin
		alreadyExistsPlugins = newPlugin
	}
	handler(PluginConfigPath)

	//监听plugins文件变化
	go func() {
		watcher.AddWatcher("./data/resources/plugins.json", handler)
	}()
}

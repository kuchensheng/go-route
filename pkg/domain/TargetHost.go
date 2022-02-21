package domain

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
	"io"
	"isc-route-service/watcher"
	"os"
	"path/filepath"
)

var RouteInfos []RouteInfo
var ConfigPath string

type Id int
type RouteInfo struct {
	Id
	Path        string   `json:"path"`
	ServiceId   string   `json:"serviceId"`
	Url         string   `json:"url"`
	Protocol    string   `json:"protocol"`
	ExcludeUrl  string   `json:"excludeUrl"`
	ExcludeUrls []string `json:"excludeUrls"`
	SpecialUrl  string   `json:"specialUrl"`
	SpecialUrls []string `json:"specialUrls"`
}

func GetRouteInfoConfigPath() string {
	cp := ConfigPath
	if cp == "" {
		wd, _ := os.Getwd()
		fp := filepath.Join(wd, "resources", "routeInfo.json")
		log.Info().Msgf("路由配置文件:%s", fp)
		if _, err := os.Stat(fp); os.IsNotExist(err) {
			log.Fatal().Msg("路由配置文件不存在")
		} else {
			cp = fp
		}
	}
	return cp
}
func InitRouteInfo() {
	log.Info().Msg("初始加载路由规则")
	cp := GetRouteInfoConfigPath()

	handler := func(filepath string) {
		log.Info().Msgf("读取文件路径%s,并加载路由信息", filepath)
		file, err := os.OpenFile(filepath, os.O_RDWR, 0666)
		if err != nil {
			log.Error().Msgf("配置文件读取异常,%v", err)
		}
		fileContent, err := io.ReadAll(file)
		if err != nil {
			log.Error().Msgf("配置文件读取异常,%v", err)
		}
		err = json.Unmarshal(fileContent, &RouteInfos)
		if err != nil {
			log.Error().Msgf("配置文件读取异常,%v", err)
		}
	}

	handler(cp)
	//todo 监听文件变化
	go func() {
		watcher.AddWatcher(cp, handler)
	}()
}

package domain

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
	"io"
	"os"
	"path/filepath"
)

var RouteInfos []RouteInfo
var ConfigPath string

type Id int
type RouteInfo struct {
	Id
	Path       string   `json:"path"`
	ServiceId  string   `json:"serviceId"`
	Url        string   `json:"url"`
	CreateTime string   `json:"createTime"`
	UpdateTime string   `json:"updateTime"`
	Protocol   string   `json:"protocol"`
	ExcludeUrl []string `json:"excludeUrl"`
	SpecialUrl []string `json:"specialUrl"`
}

func InitRouteInfo() {
	log.Info().Msg("初始加载路由规则")
	cp := ConfigPath
	if cp == "" {
		wd, _ := os.Getwd()
		fp := filepath.Join(wd, "resources", "routeInfo.json")
		if _, err := os.Stat(fp); os.IsNotExist(err) {
			log.Fatal().Msg("路由配置文件不存在")
		} else {
			cp = fp
		}
	}
	log.Info().Msgf("读取文件路径%s", cp)
	file, err := os.OpenFile(cp, os.O_RDWR, 0666)
	if err != nil {
		log.Fatal().Msgf("配置文件读取异常,%v", err)
	}

	fileContent, err := io.ReadAll(file)
	if err != nil {
		log.Fatal().Msgf("配置文件读取异常,%v", err)
	}
	err = json.Unmarshal(fileContent, &RouteInfos)
	if err != nil {
		log.Fatal().Msgf("配置文件读取异常,%v", err)
	}
	//todo 监听文件变化

}

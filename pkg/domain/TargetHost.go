package domain

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"isc-route-service/watcher"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	CreateTime  string   `json:"create_time"`
	UpdateTime  string   `json:"update_time"`
	AppCode     string   `json:"appCode"`
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
		var riMap map[string]RouteInfo
		if len(RouteInfos) > 0 {
			for _, item := range RouteInfos {
				riMap[item.ServiceId] = item
			}
		}
		var ris []RouteInfo
		err = json.Unmarshal(fileContent, &ris)
		if err != nil {
			log.Error().Msgf("配置文件读取异常,%v", err)
		}
		if len(ris) > 0 {
			//获取appCode
			serviceIdCodeMap := getServiceIdCodeMap()
			//合并同类项,以ris为准
			for idx, item := range ris {
				var e = item
				if item.ExcludeUrl != "" {
					e.ExcludeUrls = []string{item.ExcludeUrl}
					if strings.Contains(item.ExcludeUrl, ";") {
						e.ExcludeUrls = strings.Split(item.ExcludeUrl, ";")
					}
				}
				if item.SpecialUrl != "" {
					e.SpecialUrls = []string{item.SpecialUrl}
					if strings.Contains(item.SpecialUrl, ";") {
						e.SpecialUrls = strings.Split(item.SpecialUrl, ";")
					}
				}
				//todo 获取appCode
				if item.AppCode == "" {
					if appCode, ok := serviceIdCodeMap[item.ServiceId]; ok {
						e.AppCode = appCode
					}
				}
				if _, ok := riMap[item.ServiceId]; ok {
					RouteInfos[idx] = e
				} else {
					RouteInfos = append(RouteInfos, e)
					riMap[e.ServiceId] = e
				}

			}
		}

	}

	handler(cp)
	//todo 监听文件变化
	go func() {
		wd, _ := os.Getwd()
		fp := filepath.Join(wd, "data", "routeInfo.json")
		log.Info().Msgf("路由配置文件:%s", fp)
		f, err := os.OpenFile(fp, os.O_CREATE, 0666)
		if err != nil {
			log.Error().Msgf("文件打开|创建失败:%v，将不会进行文件监听", err)
			return
		}
		f.Close()
		watcher.AddWatcher(fp, handler)
	}()
}

type relevanceType struct {
	AppCode  string `json:"appCode"`
	Services []struct {
		ServiceId   string `json:"serviceId"`
		ServiceName string `json:"serviceName"`
	} `json:"services"`
}
type relevanceInfo struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Data    []relevanceType `json:"data"`
}

func getServiceIdCodeMap() map[string]string {
	result := make(map[string]string)
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	rh := ApplicationConfig.Rc.Host
	if rh == "" {
		rh = "http://isc-rc-application-service:34200"
	}
	if strings.HasPrefix(rh, "http") {
		log.Error().Msgf("rc.host must with http:// or https://")
		return result
	}
	relevance := ApplicationConfig.Rc.Relevance
	if relevance == "" {
		relevance = "/api/rc-application/application/service/relevance"
	}
	if strings.HasPrefix(relevance, "/") {
		relevance = "/" + relevance
	}
	resp, err := client.Get(fmt.Sprintf("%s%s", rh, relevance))
	if err != nil {
		log.Error().Stack().Msgf("注册中心服务调用异常:%v", err)
		return result
	}
	body := resp.Body
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		log.Error().Stack().Msgf("读取注册中心响应数据异常：%v", err)
		return result
	}
	var info relevanceInfo
	if err = json.Unmarshal(data, &info); err != nil {
		log.Error().Stack().Msgf("注册中心响应数据解析异常：%v", err)
		return result
	}
	types := info.Data
	for _, item := range types {
		appCode := item.AppCode
		for _, service := range item.Services {
			result[service.ServiceId] = appCode
		}
	}

	return result
}

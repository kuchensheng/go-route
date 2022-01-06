package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"isc-route-service/pkg/domain"
	"isc-route-service/pkg/middleware"
	"isc-route-service/pkg/proxy"
	"os"
	"path/filepath"
	"plugin"
	"sort"
)

func main() {
	port := flag.String("port", "31000", "路由服务启动端口号,默认31000")
	flag.StringVar(&domain.ConfigPath, "conf", "", "路由规则定义文件地址,默认/home/resources/routeInfo.json")
	flag.StringVar(&domain.PluginConfigPath, "plugins", "", "插件信息定位文件地址，默认/home/resources/plugins.json")
	flag.Parse()
	log.Info().Msgf("服务启动占用端口，%s", *port)
	//初始加载路由规则
	domain.InitRouteInfo()
	initPlugins()
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	//todo 拦截器
	router.Any("/*action", Forward)
	router.Run(":" + *port)
}

//InitPlugins 加载插件
func initPlugins() {
	if domain.PluginConfigPath == "" {
		wd, _ := os.Getwd()
		fp := filepath.Join(wd, "resources", "plugins.json")
		if _, err := os.Stat(fp); os.IsNotExist(err) {
			log.Fatal().Msg("路由配置文件不存在")
		} else {
			domain.PluginConfigPath = fp
		}
	}
	pluginData, err := ioutil.ReadFile(domain.PluginConfigPath)
	if err != nil {
		log.Fatal().Msgf("插件文件加载异常", err)
	}
	err = json.Unmarshal(pluginData, &domain.Plugins)
	if err != nil {
		log.Fatal().Msgf("插件文件解析异常", err)
	}
	//todo 需要根据t'y'pe 进行分类处理
	sort.SliceIsSorted(domain.Plugins, func(i, j int) bool {
		return domain.Plugins[i].Order < domain.Plugins[j].Order
	})
	for _, pluginInfo := range domain.Plugins {
		p, err := plugin.Open(pluginInfo.Path)
		if err != nil {
			log.Error().Msgf("插件[%s]加载异常,%v", pluginInfo.Name, err)
		} else {
			symbol, err := p.Lookup(pluginInfo.Method)
			if err != nil {
				log.Error().Msgf("插件[%s]变量[%s]读取异常,%v", pluginInfo.Name, pluginInfo.Method, err)
				continue
			}
			pp := &domain.PluginPointer{
				Plugin: p,
				PI:     pluginInfo,
				Symbol: symbol,
			}
			//按照分类放入list，以待执行
			switch pluginInfo.Type {
			case domain.PRE:
				domain.PrePlugins = append(domain.PrePlugins, pp)
			case domain.POST:
				domain.PostPlugsins = append(domain.PostPlugsins, pp)
			default:
				//
				domain.OtherPlugins = append(domain.OtherPlugins, pp)
			}
		}
	}

	//todo 监听插件配置文件/数据变化
}

func Forward(c *gin.Context) {
	ch := make(chan error)
	defer close(ch)
	go func() {
		//请求转发前的动作
		//ch <- middleware.PrepareMiddleWare()
		err := middleware.PrepareMiddleWare(c, domain.PrePlugins)
		if err != nil {
			c.JSON(400, fmt.Sprintf("前置处理器异常,%v", err))
			ch <- err
		} else {
			uri := c.Request.RequestURI
			targetHost, err := proxy.GetTargetRoute(uri)
			if err != nil {
				c.JSON(404, fmt.Sprintf("目标资源寻找错误，%v", err))
				ch <- err
			} else {
				ch <- proxy.HostReverseProxy(c.Writer, c.Request, *targetHost)
			}
		}
		err = middleware.PostMiddleWare()
		if err != nil {
			c.JSON(400, fmt.Sprintf("后置处理器异常,%v", err))
			ch <- err
		}
		//c.Next()
	}()
	//请求转发后的动作

	log.Debug().Msgf("代理转发完成%v", <-ch)
}

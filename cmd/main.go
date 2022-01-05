package main

import (
	"flag"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"isc-route-service/pkg/domain"
	"isc-route-service/pkg/middleware"
	"isc-route-service/pkg/proxy"
)

func main() {
	port := flag.String("port", "31000", "路由服务启动端口号,默认31000")
	flag.StringVar(&domain.ConfigPath, "conf", "", "路由规则定义文件地址,默认/home/resources/routeInfo.json")
	flag.StringVar(&domain.PluginConfigPath, "plugins", "", "插件信息定位文件地址，默认/home/resources/plugins.json")
	flag.Parse()
	log.Info().Msgf("服务启动占用端口，%s", port)
	//初始加载路由规则
	domain.InitRouteInfo()
	domain.InitPlugins()
	proxy.Router = gin.New()
	proxy.Router.Use(gin.Logger(), gin.Recovery())
	proxy.Router.Use(middleware.MiddleWare())

	//todo 拦截器
	proxy.Router.Any("/*action", proxy.Forward)
	proxy.Router.Run(":" + *port)
}

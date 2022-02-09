package main

import (
	"flag"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"isc-route-service/pkg/domain"
	"isc-route-service/pkg/proxy"
	"net"
)

func main() {
	port := flag.String("port", "31000", "http路由服务启动端口号,默认31000")
	tcpPort := flag.Int("tcpPort", 31080, "tcp路由服务启动端口号,默认31080")
	udpPort := flag.String("updPort", "31053", "tcp路由服务启动端口号,默认31053")
	flag.StringVar(&domain.ConfigPath, "conf", "", "路由规则定义文件地址,默认/home/resources/routeInfo.json")
	flag.StringVar(&domain.PluginConfigPath, "plugins", "", "插件信息定位文件地址，默认/home/resources/plugins.json")
	flag.Parse()
	log.Info().Msgf("服务启动占用端口，%s", *port)
	//初始加载路由规则
	domain.InitRouteInfo()
	domain.InitPlugins()
	go func() {
		log.Info().Msgf("tcp服务监听占用端口:%d", *tcpPort)
		tcpListener, err := net.ListenTCP("tcp", &net.TCPAddr{Port: *tcpPort})
		if err != nil {
			log.Fatal().Msgf("tcp服务监听异常:%v", err)
		}
		for {
			proxyConn, err := tcpListener.AcceptTCP()
			if err != nil {
				log.Error().Msgf("Unable to accept a request,error : %v", err)
				continue
			}
			proxyConn.Write([]byte("收到了"))
			log.Info().Msgf("localAddr : %v", proxyConn.LocalAddr())
			log.Info().Msgf("remoteAddr : %v", proxyConn.RemoteAddr())
			proxy.TCPForward(proxyConn)
		}
	}()
	go func() {
		log.Info().Msgf("upd服务监听占用端口：%s", *udpPort)
	}()
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	//todo 拦截器
	router.Any("/*action", proxy.Forward)
	router.Run(":" + *port)
}

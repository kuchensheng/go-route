package main

import (
	"flag"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"io"
	"isc-route-service/pkg/domain"
	"isc-route-service/pkg/handler"
	"isc-route-service/pkg/proxy"
	"os/exec"
)

func main() {
	port := flag.Int("port", 31000, "http路由服务启动端口号,默认31000")
	//tcpPort := flag.Int("tcpPort", 31080, "tcp路由服务启动端口号,默认31080")
	//udpPort := flag.String("updPort", "31053", "tcp路由服务启动端口号,默认31053")
	flag.StringVar(&domain.ConfigPath, "conf", "", "路由规则定义文件地址,默认/home/isc-route-service/data/resources/routeInfo.json")
	flag.StringVar(&domain.PluginConfigPath, "plugins", "", "插件信息定位文件地址，默认/home/isc-route-service/data/resources/plugins.json")
	flag.StringVar(&domain.Profile, "profiles", "", "指定的配置文件地址，例如dev,表示加载application-dev.yaml信息")

	flag.Parse()
	log.Info().Msgf("拷贝plugins和resources目录到data目录下")
	mvDir := func(dir string) error {
		cmd := exec.Command("cp", "-r", "-n", dir, "data/")
		log.Info().Msgf("执行命令：%s", cmd.String())
		return cmd.Run()

	}
	err := mvDir("resources")
	if err != nil {
		log.Error().Msgf("resource目录拷贝异常%v", err)
		return
	}
	err = mvDir("plugins")
	if err != nil {
		log.Error().Msgf("plugins目录拷贝异常%v", err)
		return
	}

	//读取指定配置文件信息
	domain.ReadProfileYaml()
	//初始加载路由规则
	domain.InitRouteInfo()
	//初始化加载动态库信息
	domain.InitPlugins()
	//go func() {
	//	log.Info().Msgf("tcp服务监听占用端口:%d", *tcpPort)
	//	tcpListener, err := net.ListenTCP("tcp", &net.TCPAddr{Port: *tcpPort})
	//	if err != nil {
	//		log.Fatal().Msgf("tcp服务监听异常:%v", err)
	//	}
	//	for {
	//		proxyConn, err := tcpListener.AcceptTCP()
	//		if err != nil {
	//			log.Error().Msgf("Unable to accept a request,error : %v", err)
	//			continue
	//		}
	//		proxyConn.Write([]byte("收到了"))
	//		log.Info().Msgf("localAddr : %v", proxyConn.LocalAddr())
	//		log.Info().Msgf("remoteAddr : %v", proxyConn.RemoteAddr())
	//		proxy.TCPForward(proxyConn)
	//	}
	//}()
	//go func() {
	//	log.Info().Msgf("upd服务监听占用端口：%s", *udpPort)
	//}()
	gin.DefaultWriter = io.Discard
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), cors.Default())
	//todo 拦截器
	router.Any("/api/*action", proxy.Forward)
	router.POST("/metrics", handler.PromHandler(promhttp.Handler()))
	pr := *port
	p := domain.ApplicationConfig.Server.Port
	if p != 0 {
		pr = p
	}
	log.Info().Msgf("服务启动占用端口，%d", pr)
	err = router.Run(fmt.Sprintf(":%d", pr))
	if err != nil {
		log.Fatal().Msgf("unable to start server due to: %v", err)
	}
}

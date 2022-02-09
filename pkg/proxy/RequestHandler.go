package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"isc-route-service/pkg/domain"
	"isc-route-service/pkg/middleware"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// TCPForward tcp请求转发
func TCPForward(proxyConn *net.TCPConn) {
	// Read a header firstly in case you could have opportunity to check request
	// whether to decline or proceed the request
	defer proxyConn.Close()
	data, err := ioutil.ReadAll(proxyConn)
	if err != nil {
		msf := fmt.Sprintf("Unable to read from input,error : %v", err)
		log.Warn().Msg(msf)
		proxyConn.Write([]byte(msf))
		return
	}
	log.Info().Msgf("接收到信息:%s", string(data))
	//todo 寻找目标
	log.Info().Msgf("localAddr : %v", proxyConn.LocalAddr())
	log.Info().Msgf("remoteAddr : %v", proxyConn.RemoteAddr())
}

//Forward http请求转发
func Forward(c *gin.Context) {
	ch := make(chan error)
	defer close(ch)
	go func() {
		//请求转发前的动作
		//ch <- middleware.PrepareMiddleWare()
		uri := c.Request.RequestURI
		targetHost, err := getTargetRoute(uri)
		if err != nil {
			c.JSON(404, fmt.Sprintf("目标资源寻找错误，%v", err))
			ch <- err
		} else {
			pre := domain.PrePlugins
			for _, p := range pre {
				p.RouteInfo = targetHost
			}
			err = middleware.PrepareMiddleWare(c, pre)
			if err != nil {
				c.JSON(400, fmt.Sprintf("前置处理器异常,%v", err))
				ch <- err
			} else {
				ch <- hostReverseProxy(c.Writer, c.Request, *targetHost)
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
func hostReverseProxy(w http.ResponseWriter, req *http.Request, target domain.RouteInfo) error {
	targetUri := target.Url
	remote, err := url.Parse(targetUri)
	if err != nil {
		msg := fmt.Sprintf("url 解析异常%v", err)
		log.Error().Msgf(msg)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(msg))
		return err
	}
	proxy := httputil.NewSingleHostReverseProxy(remote)
	if target.Protocol != "" && strings.ToUpper(target.Protocol) == "HTTPS" {
		tls, err := getVerTLSConfig("")
		if err != nil {
			msg := fmt.Sprintf("https crt error:%v", err)
			log.Error().Msg(msg)
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(msg))
			return err
		}
		var pTransport http.RoundTripper = &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				c, err := net.DialTimeout(netw, addr, time.Minute*1)
				if err != nil {
					return nil, err
				}
				return c, nil
			},
			ResponseHeaderTimeout: time.Second * 5,
			TLSClientConfig:       tls,
		}
		proxy.Transport = pTransport
	}
	proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, err error) {
		//todo 代理异常处理器
	}
	proxy.ServeHTTP(w, req)
	proxy.ModifyResponse = func(response *http.Response) error {
		//todo 代理响应处理，这里可以作为数据脱敏等处理
		return middleware.PostMiddleWare()
	}
	return nil
}

func getTargetRoute(uri string) (*domain.RouteInfo, error) {
	//todo  根据uri解析到目标路由服务
	for _, route := range domain.RouteInfos {
		path := route.Path
		paths := strings.Split(path, "/")
		uriPaths := strings.Split(uri, "/")
		var match = false
	inner:
		for idx, p := range paths {
			if p == "" {
				continue
			}
			if strings.Contains(p, "*") {
				match = true
				break inner
			}
			if p != uriPaths[idx] {
				match = false
				break inner
			}
		}
		if match {
			return &route, nil
		}
	}
	return nil, fmt.Errorf("路由规则不存在")
}

func getVerTLSConfig(CaPath string) (*tls.Config, error) {
	if CaPath == "" {
		return &tls.Config{
			InsecureSkipVerify: true,
		}, nil
	}
	caData, err := ioutil.ReadFile(CaPath)
	if err != nil {
		log.Error().Msgf("read ca file fail,%v", err)
		return nil, err
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caData)
	return &tls.Config{
		RootCAs: pool,
	}, err
}

package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"isc-route-service/pkg/domain"
	"isc-route-service/pkg/exception"
	"isc-route-service/pkg/handler"
	"isc-route-service/pkg/middleware"
	tracer2 "isc-route-service/pkg/tracer"
	plugins "isc-route-service/plugins/common"
	"isc-route-service/utils"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"reflect"
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
func startTrace(c *http.Request) (*tracer2.Tracer, error) {
	//开启tracer
	tracer, err := tracer2.New()
	if err != nil {
		return nil, err
	}
	tracer.TraceType = tracer2.HTTP
	tracer.RemoteIp = getRemoteIp(c)
	return tracer, nil
}

func getRemoteIp(c *http.Request) string {
	return c.RemoteAddr
}

//Forward http请求转发
func Forward(c *gin.Context) {
	uri := c.Request.RequestURI
	if uri == "/api/route/refreshRoute" {
		handler.UpdateRoute(c)
		return
	}
	if uri == "/api/route/system/status" {
		c.JSON(http.StatusOK, `{}`)
	}
	ch := make(chan error)
	defer close(ch)
	//开启tracer
	tracer, err := startTrace(c.Request)
	if err != nil {
		log.Error().Msgf("链路跟踪服务端开启异常,\n%v", err)
		ch <- err
	}
	//设置当前节点是服务端trace
	tracer.Endpoint = tracer2.SERVER
	//获取remoteIP
	tracer.RemoteIp = c.ClientIP()

	//开启协程转发http请求
	go func() {
		//请求转发前的动作
		//1.查看目标主机信息，clientRecovery
		targetHost, err := getTargetRoute(uri)
		if err != nil {
			c.JSON(404, exception.BusinessException{
				Code:    1040404,
				Message: "路由规则不存在",
			})
			ch <- err
		} else {
			//这里的逻辑有点怪，每个pre插件都需要获取到routeInfo?先这样处理
			pre := domain.PrePlugins
			for idx, _ := range pre {
				pre[idx].RouteInfo = *targetHost
			}
			//执行前置插件，只要有一个插件抛出异常，则终止服务
			err = middleware.PrepareMiddleWare(c, pre)
			if err != nil {
				//异常判断处理，如果是自定义异常，则需要进行相关转化
				pe := &plugins.BusinessException{}
				if reflect.TypeOf(err) == reflect.TypeOf(pe) {
					pe = err.(*plugins.BusinessException)
					statusCode := pe.StatusCode
					c.JSON(statusCode, pe)
				} else {
					c.JSON(400, err)
				}
				ch <- err
			} else {
				ch <- hostReverseProxy(c.Writer, c.Request, *targetHost)
			}
		}
		//c.Next()
	}()
	err = <-ch
	resultChan := make(chan string, 1)
	//请求转发后的动作
	go func(err error) {
		if tracer != nil {
			if err != nil {
				resultChan <- err.Error()
				tracer.EndTrace(tracer2.ERROR, err.Error())
			} else {
				tracer.EndTrace(tracer2.OK, "")
				resultChan <- "trace执行完毕"
			}

		} else {
			resultChan <- "trace为空"
		}
	}(err)
	result := <-resultChan
	log.Debug().Msgf("代理转发完成\n%v", result)

}

//这里用于创建http client，默认5s超时，超时处理暂未实现
var transport = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	MaxIdleConns:          1024,
	MaxIdleConnsPerHost:   512,
	IdleConnTimeout:       time.Duration(30) * time.Second,
	ResponseHeaderTimeout: 5 * time.Second,
}

//hostReverseProxy 真正的转发逻辑，基于httputil.NewSingleHostReverseProxy 进行代理转发
func hostReverseProxy(w http.ResponseWriter, req *http.Request, target domain.RouteInfo) error {
	protocal := "HTTP"
	targetUri := target.Url
	if strings.HasPrefix(targetUri, "ws://") {
		targetUri = strings.ReplaceAll(targetUri, "ws://", "http://")
	}
	if strings.HasPrefix(targetUri, "wss://") {
		targetUri = strings.ReplaceAll(targetUri, "wss://", "https://")
	}
	remote, err := url.Parse(targetUri)
	if err != nil {
		msg := fmt.Sprintf("url 解析异常%v", err)
		log.Error().Msgf(msg)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(msg))
		return err
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	if target.Protocol != "" {
		protocal = strings.ToUpper(target.Protocol)
	}
	var tls *tls.Config
	if protocal == "HTTPS" {
		tls, err = getVerTLSConfig("")
		if err != nil {
			msg := fmt.Sprintf("https crt error:%v", err)
			log.Error().Msg(msg)
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(msg))
			return err
		}
		transport.TLSClientConfig = tls
	}
	if isSpecialReq(req.URL.Path, &target) {
		transport.IdleConnTimeout = 5 * time.Minute
	}
	proxy.Transport = transport
	//traceClient处理,tracer.enter
	trace, err := startTrace(req)
	if err != nil {
		log.Warn().Msgf("链路跟踪客户端初始化异常，将不开启客户端跟踪\n%v", err)
	} else {
		trace.TraceName = fmt.Sprintf("<%s>%s", req.Method, req.URL.Path)
		trace.Endpoint = tracer2.CLIENT
	}
	proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, err error) {
		//异常处理器
		w.WriteHeader(http.StatusOK)
		writeContent := exception.BusinessException{
			Code:    1040502,
			Message: fmt.Sprintf("远程调用异常:%v", err.Error()),
			Data:    err,
		}
		data, _ := json.Marshal(writeContent)
		w.Write(data)
		if err != nil {
			trace.EndTrace(tracer2.WARNING, err.Error())
		}
	}
	proxy.ModifyResponse = func(response *http.Response) error {
		//todo 代理响应处理，这里可以作为数据脱敏等处理
		trace.EndTrace(tracer2.OK, "")
		return middleware.PostMiddleWare()
	}
	proxy.ServeHTTP(w, req)
	return nil
}

//isSpecialReq 判断是否符合特殊处理，若符合则设置超时时间为5分钟
func isSpecialReq(uri string, targetRoute *domain.RouteInfo) bool {
	if len(targetRoute.SpecialUrl) == 0 {
		return false
	}
	for _, item := range targetRoute.SpecialUrls {
		paths := strings.Split(item, "/")
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
		return match
	}
	return false
}

//getTargetRoute 根据uri解析查找目标服务,这里是clientRecovery
func getTargetRoute(uri string) (*domain.RouteInfo, error) {
	// 根据uri解析到目标路由服务
	for _, route := range domain.RouteInfos {
		path := route.Path
		if strings.Contains(path, ";") {
			for _, item := range strings.Split(path, ";") {
				if utils.Match(uri, item) {
					return &route, nil
				}
			}
		} else {
			if utils.Match(uri, path) {
				return &route, nil
			}
		}
	}
	return nil, fmt.Errorf("路由规则不存在")
}

var tlsSkipVerify *tls.Config
var tlsConfig *tls.Config

//getVerTLSConfig 获取证书信息，CaPath表示证书路径，如果获取不到则表示跳过证书
func getVerTLSConfig(CaPath string) (*tls.Config, error) {
	if CaPath == "" {
		if tlsSkipVerify == nil {
			tlsSkipVerify = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
		return tlsSkipVerify, nil
	} else {
		if tlsConfig == nil {
			caData, err := ioutil.ReadFile(CaPath)
			if err != nil {
				log.Error().Msgf("read ca file fail,%v", err)
				return nil, err
			}
			pool := x509.NewCertPool()
			pool.AppendCertsFromPEM(caData)
			tlsConfig = &tls.Config{
				RootCAs: pool,
			}
		}
		return tlsConfig, nil
	}
}

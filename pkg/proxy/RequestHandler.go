package proxy

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"isc-route-service/pkg/domain"
	"isc-route-service/pkg/exception"
	"isc-route-service/pkg/handler"
	"isc-route-service/pkg/middleware"
	"isc-route-service/pkg/ratelimit"
	tracer2 "isc-route-service/pkg/tracer"
	plugins "isc-route-service/plugins/common"
	"net"
	"net/http"
	"reflect"
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
	tracer, err := tracer2.New(c)
	if err != nil {
		return nil, err
	}
	tracer.TraceType = tracer2.HTTP
	tracer.RemoteIp = getRemoteIp(c)
	if c.Header.Get("t-head-traceId") == "" {
		c.Header.Set("t-head-traceId", tracer.TracId)
	}
	return tracer, nil
}

func getRemoteIp(c *http.Request) string {
	return c.RemoteAddr
}

//Forward http请求转发
func Forward(c *gin.Context) {
	if handler.IscRouteHandler(c) {
		return
	}
	if ratelimit.RateLimiterHandler(c) {
		return
	}
	uri := c.Request.RequestURI

	ch := make(chan error)
	defer close(ch)
	var tracer *tracer2.Tracer
	var err error
	if domain.ApplicationConfig.Loki.Enable {
		//开启tracer
		tracer, err = startTrace(c.Request)
		if err != nil {
			log.Error().Msgf("链路跟踪服务端开启异常,\n%v", err)
			ch <- err
		}
		//设置当前节点是服务端trace
		tracer.Endpoint = tracer2.SERVER
		//获取remoteIP
		tracer.RemoteIp = c.ClientIP()
	}

	//开启协程转发http请求
	go func() {
		//请求转发前的动作
		//1.查看目标主机信息，clientRecovery
		targetHost, err := domain.GetTargetRoute(uri)
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

//这里用于创建http client，默认5s超时
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
	//traceClient处理,tracer.enter
	var trace *tracer2.Tracer
	var err error
	if domain.ApplicationConfig.Loki.Enable {
		trace, err = startTrace(req)
		if err != nil {
			log.Warn().Msgf("链路跟踪客户端初始化异常，将不开启客户端跟踪\n%v", err)
		} else {
			trace.TraceName = fmt.Sprintf("<%s>%s", req.Method, req.URL.Path)
			trace.Endpoint = tracer2.CLIENT
		}
	}

	proxy, err := target.GetProxy(w, req)
	if err != nil || proxy == nil {
		return &exception.BusinessException{
			Code:    1040404,
			Message: fmt.Sprintf("代理创建失败:%v", err.Error()),
			Data:    err,
		}
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
		if err != nil && trace != nil {
			trace.EndTrace(tracer2.WARNING, err.Error())
		}
	}
	proxy.ModifyResponse = func(response *http.Response) error {
		//todo 代理响应处理，这里可以作为数据脱敏等处理
		if trace != nil {
			trace.EndTrace(tracer2.OK, "")
		}
		return middleware.PostMiddleWare()
	}
	done := make(chan error)
	go func() {
		defer func() {
			if x := recover(); x != nil {
				done <- x.(error)
			}
			close(done)
		}()
		proxy.ServeHTTP(w, req)
		target.AddProxy(proxy)
	}()
	return <-done
}

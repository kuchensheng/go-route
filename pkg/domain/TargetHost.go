package domain

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
	"io"
	"io/ioutil"
	"isc-route-service/watcher"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var RouteInfos []RouteInfo
var ConfigPath string

const (
	Kernel int = iota
	Center
	Other
)

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
	Predicates  []string `json:"predicates"`
	//Type returns route type,
	Type int `json:"type"`
}

type DBRouteInfo struct {
	Id         *int    `db:"id"`
	Path       *string `db:"path"`
	ServiceId  *string `db:"service_id"`
	Url        *string `db:"url"`
	Protocol   *string `db:"protocol"`
	ExcludeUrl *string `db:"exclude_url"`
	SpecialUrl *string `db:"special_url"`
	CreateTime *string `db:"create_time"`
	UpdateTime *string `db:"update_time"`
}

func B2S(bs []uint8) string {
	var ba []byte
	for _, b := range bs {
		ba = append(ba, b)
	}
	return string(ba)
}

func db2RouteInfo(route DBRouteInfo) (*RouteInfo, error) {
	if route.Url == nil {
		return nil, errors.New("url不能为空")
	}
	if route.Path == nil {
		return nil, errors.New("path为空")
	}
	r := &RouteInfo{}
	data, _ := json.Marshal(route)
	json.Unmarshal(data, r)
	protocol := route.Protocol
	if protocol == nil {
		r.Protocol = "HTTP"
	}

	return r, nil
}

func readDataBase(fp string) {
	//"root:XXXX@tcp(127.0.0.1:3306)/test"
	dataSourceName := fmt.Sprintf("%s:%s@tcp(%s)/%s", ApplicationConfig.Mysql.UserName, ApplicationConfig.Mysql.Password, ApplicationConfig.Mysql.Host, ApplicationConfig.Mysql.DataBase)
	database, err := sqlx.Open("mysql", dataSourceName)
	if err != nil {
		log.Fatal().Msgf("路由配置文件不存在，且数据库连接异常%v", err)
	}
	defer database.Close()
	var rs []DBRouteInfo
	err = database.Select(&rs, "select id,path,service_id,url,protocol,exclude_url,special_url,create_time,update_time  from route_info")
	if err != nil {
		log.Fatal().Msgf("从数据库中读取路由服务信息异常%v", err)
	}
	//转换成routeInfo并写入到routeInfo.json
	var rss []RouteInfo
	for _, route := range rs {
		r, err := db2RouteInfo(route)
		if err != nil {
			log.Warn().Msgf("路由信息转换异常：%v", err)
			continue
		}
		rss = append(rss, *r)
	}
	data, err := json.Marshal(rss)
	if err != nil {
		log.Fatal().Msgf("路由信息序列化异常%v", err)
	}
	file, err := os.OpenFile(fp, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal().Msgf("路由配置文件创建/打开异常%v", err)
	}
	_, err = file.Write(data)
	if err != nil {
		log.Fatal().Msgf("路由配置文件创建/打开异常%v", err)
	}
	log.Info().Msgf("完成从数据库中读取路由服务信息")
}

func GetRouteInfoConfigPath() string {
	cp := ConfigPath
	if cp == "" {
		wd, _ := os.Getwd()
		fp := filepath.Join(wd, "data", "resources", "routeInfo.json")
		log.Info().Msgf("路由配置文件:%s", fp)
		if _, err := os.Stat(fp); os.IsNotExist(err) {
			if !ApplicationConfig.Server.Compatible {
				mvDir := func(dir string) error {
					cmd := exec.Command("cp", "-r", "-n", dir, "data/resources/routeInfo.json")
					log.Info().Msgf("执行命令：%s", cmd.String())
					return cmd.Run()
				}
				err := mvDir("init/routeInfo.json")
				if err != nil {
					log.Error().Msgf("resource目录拷贝异常%v", err)
				}
			} else {
				log.Error().Msg("路由配置文件不存在,从数据库中读取数据")
				readDataBase(fp)
			}
		}
		cp = fp
	}
	return cp
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

//isSpecialReq 判断是否符合特殊处理，若符合则设置超时时间为5分钟
func (target *RouteInfo) isSpecialReq(uri string) bool {
	if len(target.SpecialUrl) == 0 {
		return false
	}
	for _, item := range target.SpecialUrls {
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

//var proxyPool = make(map[string][]*httputil.ReverseProxy)
var proxyPool sync.Map

func (target *RouteInfo) GetProxy(w http.ResponseWriter, req *http.Request) (*httputil.ReverseProxy, error) {
	targetUri := target.getTargetUri()
	remote, err := url.Parse(targetUri)
	if err != nil {
		msg := fmt.Sprintf("url 解析异常%v", err)
		log.Error().Msgf(msg)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(msg))
		return nil, err
	}
	if target.Predicates != nil {
		url := req.URL.Path
		for _, v := range target.Predicates {
			if !strings.Contains(v, "=") {
				log.Debug().Msgf("predicates must contains = ,eq:stripPrefix=2")
				continue
			}
			kv := strings.Split(v, "=")
			key := strings.TrimSpace(kv[0])
			value := kv[1]
			if key == "stripPrefix" {
				v1, err := strconv.ParseInt(value, 10, 0)
				if err != nil {
					continue
				}
				subUrls := strings.Split(url, "/")
				subUrls = subUrls[v1:]
				url = strings.Join(subUrls, "/")
				req.URL.Path = url
			}
		}
	}
	//
	var proxy *httputil.ReverseProxy
	ps, ok := proxyPool.Load(targetUri)
	if !ok || len(ps.([]*httputil.ReverseProxy)) == 0 {
		proxy, err = target.createProxy(w, req, remote)
		if err == nil {
			target.AddProxy(proxy)
		}
	} else {
		proxies := ps.([]*httputil.ReverseProxy)
		proxy = proxies[0]
		proxyPool.Store(targetUri, proxies[1:])
	}
	t := *transport
	if target.isSpecialReq(req.URL.Path) {
		t.IdleConnTimeout = 5 * time.Minute
		t.DialContext = (&net.Dialer{
			Timeout:   5 * time.Minute,
			KeepAlive: 30 * time.Second,
		}).DialContext
		//t.ResponseHeaderTimeout = 5 * time.Minute
	}
	proxy.Transport = &t
	return proxy, nil
}
func (target *RouteInfo) getTargetUri() string {
	targetUri := target.Url
	if strings.HasPrefix(targetUri, "ws://") {
		targetUri = strings.ReplaceAll(targetUri, "ws://", "http://")
	}
	if strings.HasPrefix(targetUri, "wss://") {
		targetUri = strings.ReplaceAll(targetUri, "wss://", "https://")
	}
	return targetUri
}
func (target *RouteInfo) AddProxy(proxy *httputil.ReverseProxy) {
	targetUri := target.getTargetUri()
	var ps []*httputil.ReverseProxy
	proxies, ok := proxyPool.Load(targetUri)
	if !ok || len(proxies.([]*httputil.ReverseProxy)) == 0 {
		ps = append(ps, proxy)
	} else {
		ps = append(proxies.([]*httputil.ReverseProxy), proxy)
	}
	proxyPool.Store(targetUri, ps)
}

//这里用于创建http client，默认5s超时，超时处理暂未实现
var transport = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	MaxIdleConns:        1024,
	MaxIdleConnsPerHost: 512,
	IdleConnTimeout:     time.Duration(5) * time.Second,
	//ResponseHeaderTimeout: 5 * time.Second,
}

func (target *RouteInfo) createProxy(w http.ResponseWriter, req *http.Request, remote *url.URL) (*httputil.ReverseProxy, error) {
	protocal := "HTTP"
	proxy := httputil.NewSingleHostReverseProxy(remote)
	if target.Protocol != "" {
		protocal = strings.ToUpper(target.Protocol)
	}
	//var tls *tls.Config
	if protocal == "HTTPS" {
		tls, err := getVerTLSConfig("")
		if err != nil {
			msg := fmt.Sprintf("https crt error:%v", err)
			log.Error().Msg(msg)
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(msg))
			return nil, err
		}
		transport.TLSClientConfig = tls
	}
	proxy.Transport = transport

	return proxy, nil
}

var R *Router

func InitRouter(routes []RouteInfo) {
	records := []Record{}
	for _, routeInfo := range routes {
		key := routeInfo.Path
		if strings.Contains(key, ";") {
			keys := strings.Split(key, ";")
			for _, k := range keys {
				if k != "" {
					records = append(records, Record{Key: k, Value: routeInfo})
				}
			}
		} else {
			records = append(records, Record{Key: key, Value: routeInfo})
		}
	}
	r := New()
	R = r
	if err := R.Build(records); err != nil {
		log.Panic().Msgf("路由索引生成失败,%v", err)
	}
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
		//获取已有的路由规则信息
		riMap := make(map[string]RouteInfo)
		if len(RouteInfos) > 0 {
			for _, item := range RouteInfos {
				riMap[item.ServiceId+"_"+item.Protocol] = item
			}
		}
		//获取文件中的路由规则信息
		var ris []RouteInfo
		err = json.Unmarshal(fileContent, &ris)
		if err != nil {
			log.Error().Msgf("配置文件读取异常,%v", err)
		}
		if len(ris) > 0 {
			//获取appCode
			serviceIdCodeMap := getServiceIdCodeMap()
			//合并同类项,以ris为准
			for _, item := range ris {
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
				riMap[e.ServiceId+"_"+e.Protocol] = e
			}
			//将路由信息再次转换为list
			var newRouteInfos []RouteInfo
			for _, v := range riMap {
				newRouteInfos = append(newRouteInfos, v)
			}
			RouteInfos = newRouteInfos
			InitRouter(newRouteInfos)
		}

	}

	handler(cp)
	//todo 监听文件变化
	go func() {
		watcher.AddWatcher("./data/resources/routeInfo.json", handler)
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
	if !strings.HasPrefix(rh, "http") {
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

//GetTargetRoute 根据uri解析查找目标服务,这里是clientRecovery
func GetTargetRoute(uri string) (*RouteInfo, error) {
	// 根据uri解析到目标路由服务
	lookup, _, found := R.Lookup(uri)
	if found {
		value := lookup.(RouteInfo)
		return &value, nil
	}
	return nil, fmt.Errorf("路由规则不存在")
}

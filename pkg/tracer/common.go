package tracer

import (
	"fmt"
	idworker "github.com/gitstliu/go-id-worker"
	"github.com/robfig/cron"
	"github.com/rs/zerolog/log"
	"isc-route-service/pkg/domain"
	"net"
	"strconv"
	"strings"
	"time"
)

type TraceTypeEnum int
type EndpointEnum int
type TraceStatusEnum int

const (
	ROOT TraceTypeEnum = iota
	HTTP
	DUBBO
	MYSQL
	ROCKETMQ
	REDIS
	KAFKA
	IDS
	MQTT
	ORACLE
	ELASTIC
	ZOOKEEPER
	HBASE
	HADOOP
	FLINK
	SPARK
	KUDU
	HIVE
	STORM
	CONFIG
)
const (
	CLIENT EndpointEnum = iota
	SERVER
)
const (
	OK TraceStatusEnum = iota
	ERROR
	WARNING
	TIMEOUT
)

type Tracer struct {
	//调用链ID,一旦初始化,不能修改
	tracId       string
	rpcId        string
	TraceType    TraceTypeEnum
	TraceName    string
	Endpoint     EndpointEnum
	status       TraceStatusEnum
	RemoteStatus TraceStatusEnum
	RemoteIp     string
	message      string
	Size         int
	startTime    int64
	endTime      int64
	sampled      bool
	bizData      map[string]interface{}
	Ended        bool
	attrMap      map[string]string
}

var currWorker = &idworker.IdWorker{}
var Client *LokiClient

func InitLokiClient() {
	c, err := CreateClient(domain.ApplicationConfig.Loki.Host, 512, 30*time.Second)
	if err != nil {
		log.Fatal().Msgf("loki客户端初始化失败\n%v", err)
	}
	Client = c
}
func init() {
	log.Info().Msgf("初始化信息跟踪动态库")
	currWorker.InitIdWorker(1500, 1)
	InitLokiClient()
	c := cron.New()
	c.AddFunc("*/1 * * * * ?", func() {
		//上传到loki中
		err := Client.send()
		if err != nil {
			log.Warn().Msgf("日志上传异常%v", err)
		}
		Client.currentMessage.Streams = nil

	})
	c.Start()
}
func New() (*Tracer, error) {
	newId, err := currWorker.NextId()
	if err != nil {
		return nil, err
	}
	return &Tracer{
		tracId:    strconv.FormatInt(newId, 19),
		sampled:   true,
		startTime: time.Now().UnixMilli(),
		rpcId:     "0",
	}, nil
}

func (tracer *Tracer) EndTrace(status TraceStatusEnum, message string) {
	if tracer.Ended {
		return
	}
	if tracer.tracId == "" {
		return
	}
	if tracer.rpcId == "" {
		return
	}
	tracer.Ended = true
	if !tracer.sampled {
		return
	}
	tracer.endTime = time.Now().UnixMilli()
	tracer.status = status
	if message != "" {
		tracer.message = message
	}
	//扔到loki中去
	labels := make(map[string]string)
	labels["app"] = "isc-route-service"
	labels["appName"] = "路由服务"
	labels["time"] = strconv.FormatInt(tracer.endTime, 13)
	Client.AddStream(labels, []Message{tracer.buildLog()})
}
func (tracer *Tracer) buildLog() Message {
	var strItem []string
	result := &Message{
		Time: strconv.FormatInt(tracer.endTime, 10) + "000000",
	}
	strItem = append(strItem, "0", "default", strconv.FormatInt(tracer.startTime, 13), tracer.tracId,
		tracer.rpcId, strconv.Itoa(int(tracer.Endpoint)), strconv.Itoa(int(tracer.TraceType)), tracer.TraceName,
		"isc-route-service", GetLocalIp(), tracer.RemoteIp, strconv.Itoa(int(tracer.status)), strconv.Itoa(tracer.Size),
		strconv.FormatInt(tracer.endTime-tracer.startTime, 13), tracer.message)
	strItem = append(strItem, tracer.tracId)
	result.Message = strings.Join(strItem, "|")
	return *result
}

func GetLocalIp() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Warn().Msgf("获取本地地址异常,%v", err)
		return "127.0.0.1"
	}
	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip := fmt.Sprintf(ipnet.IP.String())
				return ip
			}

		}
	}
	return "127.0.0.1"
}

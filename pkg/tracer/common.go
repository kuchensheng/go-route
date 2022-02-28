package tracer

import (
	"fmt"
	idworker "github.com/gitstliu/go-id-worker"
	"github.com/robfig/cron"
	"github.com/rs/zerolog/log"
	"isc-route-service/pkg/domain"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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
	TracId       string
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
	host := "http://loki-service:3100"
	if domain.ApplicationConfig.Loki.Host != "" {
		host = domain.ApplicationConfig.Loki.Host
	}
	c, err := CreateClient(host, 512, 30*time.Second)
	if err != nil {
		log.Fatal().Msgf("loki客户端初始化失败\n%v", err)
	}
	Client = c
}

var SEQ atomic.Value
var MAX = 8000

func init() {
	log.Info().Msgf("初始化信息跟踪动态库")
	SEQ.Store(1)
	currWorker.InitIdWorker(1500, 1)
	InitLokiClient()
	c := cron.New()
	c.AddFunc("* * * * * ?", func() {
		//上传到loki中
		err := Client.send()
		defer func() {
			Client.currentMessage.Streams = []jsonStream{}
		}()
		if err != nil {
			log.Warn().Msgf("日志上传异常%v", err)
		}

	})
	c.Start()
}

var lock sync.Mutex

//GenerateTraceId 生成唯一traceId值
func GenerateTraceId() string {
	lock.Lock()
	defer lock.Unlock()
	var buffer string
	current := SEQ.Load().(int)
	next := current
	if current > MAX {
		next = 1
	} else {
		next = current + 1
	}
	if SEQ.CompareAndSwap(current, next) {
		buffer += strconv.Itoa(next)
	}
	buffer += strconv.FormatInt(time.Now().UnixMilli(), 10)
	localIp := GetLocalIp()
	addrInt := ipAddrToInt(localIp)
	c := strconv.FormatInt(addrInt, 10)
	buffer += c
	buffer += strconv.Itoa(os.Getpid())
	//var result []byte

	//hex.Decode(result, []byte(buffer))
	return buffer
}
func New(req *http.Request) (*Tracer, error) {
	//newId, err := currWorker.NextId()
	//if err != nil {
	//	return nil, err
	//}
	traceId := req.Header.Get("t-head-traceId")
	if traceId != "" {
		traceId = GenerateTraceId()
	}
	return &Tracer{
		//tracId:    strconv.FormatInt(newId, 19),
		TracId:    traceId,
		sampled:   true,
		startTime: time.Now().UnixMilli(),
		rpcId:     "0",
	}, nil
}

func (tracer *Tracer) EndTrace(status TraceStatusEnum, message string) {
	if tracer.Ended {
		return
	}
	if tracer.TracId == "" {
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
	labels["job"] = "tracelogs"
	Client.AddStream(labels, []Message{tracer.buildLog()})
}
func (tracer *Tracer) buildLog() Message {
	var strItem []string
	result := &Message{
		Time: strconv.FormatInt(tracer.endTime, 10) + "000000",
	}
	strItem = append(strItem, "0", "default", strconv.FormatInt(tracer.startTime, 10), tracer.TracId,
		tracer.rpcId, strconv.Itoa(int(tracer.Endpoint)), strconv.Itoa(int(tracer.TraceType)), tracer.TraceName,
		"isc-route-service", GetLocalIp(), tracer.RemoteIp, strconv.Itoa(int(tracer.status)), strconv.Itoa(tracer.Size),
		strconv.FormatInt(tracer.endTime-tracer.startTime, 10), tracer.message)
	result.Message = strings.Join(strItem, "|")
	return *result
}

type localIp struct {
	LocalIp string
}

var li *localIp

func GetLocalIp() string {
	if li != nil {
		return li.LocalIp
	}
	li = &localIp{
		LocalIp: "127.0.0.1",
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Warn().Msgf("获取本地地址异常,%v", err)
		return li.LocalIp
	}
	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip := fmt.Sprintf(ipnet.IP.String())
				li = &localIp{ip}
				return li.LocalIp
			}

		}
	}
	return li.LocalIp
}

func ipAddrToInt(ipAddr string) int64 {
	bits := strings.Split(ipAddr, ".")
	b0, _ := strconv.Atoi(bits[0])
	b1, _ := strconv.Atoi(bits[1])
	b2, _ := strconv.Atoi(bits[2])
	b3, _ := strconv.Atoi(bits[3])
	var sum int64
	sum += int64(b0) << 24
	sum += int64(b1) << 16
	sum += int64(b2) << 8
	sum += int64(b3)

	return sum
}

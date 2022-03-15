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
		if len(Client.currentMessage.Streams) > 0 {
			err := Client.send()
			defer func() {
				Client.currentMessage.Streams = []jsonStream{}
			}()
			if err != nil {
				log.Warn().Msgf("日志上传异常%v", err)
			}
		}
	})
	c.Start()
}

var lock sync.Mutex

var seq uint64 = 0
var digits = []uint8{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f'}

const max = 8000

//GenerateTraceId 生成唯一traceId值
func GenerateTraceId() string {
	buffer := make([]byte, 16)

	// 计算当前session的咋一序号
	atomic.AddUint64(&seq, 1)
	current := seq
	var next uint64
	if current >= max {
		next = 1
	} else {
		next = current + 1
	}
	seq = next
	bs := shortToBytes(uint16(current))
	putBuffer(&buffer, bs, 0)

	// 计算时间
	t0 := time.Now().UnixMilli()
	bt0 := int64ToBytes(t0)
	putBuffer(&buffer, bt0, 2)

	// 计算IP地址
	addrs, _ := net.InterfaceAddrs()
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip := ipnet.IP.To4()
				putBuffer(&buffer, ip, 10)
				break
			}
		}
	}

	// 计算PID
	pid := os.Getpid()
	bp := shortToBytes(uint16(pid))
	putBuffer(&buffer, bp, 14)

	hex := encodeHex(buffer, digits)

	return string(hex)
}

func putBuffer(buf *[]byte, b []byte, from int) {
	idx := from
	for _, e := range b {
		(*buf)[idx] = e
		idx++
	}
}

func shortToBytes(s uint16) []byte {
	b := make([]byte, 2)
	b[0] = byte(s >> 8)
	b[1] = byte(s)
	return b
}

func int64ToBytes(i int64) []byte {
	b := make([]byte, 8)
	b[0] = byte(i >> 56)
	b[1] = byte(i >> 48)
	b[2] = byte(i >> 40)
	b[3] = byte(i >> 32)
	b[4] = byte(i >> 24)
	b[5] = byte(i >> 16)
	b[6] = byte(i >> 8)
	b[7] = byte(i)
	return b
}

func encodeHex(data []byte, dig []uint8) []uint8 {
	l := len(data)
	out := make([]uint8, l<<1)
	var j int = 0
	for i := 0; i < l; i++ {
		out[j] = dig[(0xf0&data[i])>>4]
		j++
		out[j] = dig[0x0f&data[i]]
		j++
	}
	return out
}
func New(req *http.Request) (*Tracer, error) {
	//newId, err := currWorker.NextId()
	//if err != nil {
	//	return nil, err
	//}
	traceId := req.Header.Get("t-head-traceId")
	if traceId == "" {
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

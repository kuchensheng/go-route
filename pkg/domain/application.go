package domain

import (
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/lestrrat-go/file-rotatelogs"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var Profile string

var ApplicationConfig *AppServerConf
var RedisClient *redis.Client

type ServerConf struct {
	Port    int    `yaml:"port"`
	Name    string `yaml:"name"`
	Module  string `yaml:"api-module"`
	Logging struct {
		Level string `yaml:"level"`
	} `yaml:"logging"`
	Limit   int `yaml:"limit"`
	Profile struct {
		Active string `yaml:"active"`
	} `yaml:"profile"`
}
type AppServerConf struct {
	Server ServerConf `yaml:"server"`
	Loki   struct {
		Host string `yaml:"host"`
	} `yaml:"loki"`
	Rc struct {
		Host      string `yaml:"host"`
		Relevance string `yaml:"relevance"`
	} `yaml:"rc"`
	Mysql struct {
		Host     string `yaml:"host"`
		UserName string `yaml:"user_name"`
		Password string `yaml:"password"`
		DataBase string `yaml:"data_base"`
	} `yaml:"mysql"`
}

func newDefaultConf() *AppServerConf {
	return &AppServerConf{
		Server: ServerConf{
			Port:   31000,
			Name:   "isc-route-service",
			Module: "route",
			Logging: struct {
				Level string `yaml:"level"`
			}(struct{ Level string }{Level: "INFO"}),
			Limit: 512,
		},
		Loki: struct {
			Host string `yaml:"host"`
		}{Host: "http://loki-service:3100"},
		Mysql: struct {
			Host     string `yaml:"host"`
			UserName string `yaml:"user_name"`
			Password string `yaml:"password"`
			DataBase string `yaml:"data_base"`
		}{Host: "mysql-service:3306", UserName: "isyscore", Password: "Isysc0re", DataBase: "isc_service"},
	}
}

func init() {
	//初始化ApplicationConf
	ApplicationConfig = newDefaultConf()
	ApplicationConfig.readApplicationYaml("")
	act := ApplicationConfig.Server.Profile.Active
	if act != "" {
		ApplicationConfig.readApplicationYaml(act)
	}
	InitLog()
}
func ReadProfileYaml() {
	if Profile != "" {
		ApplicationConfig.readApplicationYaml(Profile)
	}
}
func (conf *AppServerConf) readApplicationYaml(act string) {
	pwd, _ := os.Getwd()
	path := "application.yml"
	if act != "" {
		path = fmt.Sprintf("application-%s.yml", act)
	}
	log.Info().Msgf("加载[%s]文件", path)
	fp := filepath.Join(pwd, path)
	data, err := ioutil.ReadFile(fp)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Warn().Msgf("%s文件不存在", fp)
		} else {
			log.Fatal().Msgf("application.yml文件读取异常", err)
		}
	} else {
		err = yaml.Unmarshal(data, conf)
		if err != nil {
			log.Fatal().Msgf("application.yml解析错误", err)
		}
	}
}

var loggerTrace *zerolog.Logger
var loggerInfo *zerolog.Logger
var loggerDebug *zerolog.Logger
var loggerWarn *zerolog.Logger
var loggerError *zerolog.Logger
var loggerOther *zerolog.Logger

func InitLog() {
	initLogDir()
	level := ApplicationConfig.Server.Logging.Level
	l := zerolog.InfoLevel
	if level != "" {
		l1, err := zerolog.ParseLevel(strings.ToLower(level))
		if err != nil {
			log.Warn().Msgf("日志设置异常，将使用默认级别 INFO")
		} else {
			l = l1
		}
		zerolog.SetGlobalLevel(l)
	}
	zerolog.CallerSkipFrameCount = 2
	zerolog.TimeFieldFormat = time.RFC3339
	out := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "2006-01-02 15:04:05.000", NoColor: true}
	out.FormatLevel = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf(" [%s] [%-2s]", ApplicationConfig.Server.Name, i))
	}
	log.Logger = log.Logger.Output(out).With().Caller().Logger()
	//添加hook
	levelInfoHook := zerolog.HookFunc(func(e *zerolog.Event, l zerolog.Level, msg string) {
		//levelName := l.String()
		e1 := e

		switch l {
		case zerolog.DebugLevel:
			e1 = loggerDebug.Debug().Stack()
		case zerolog.InfoLevel:
			e1 = loggerInfo.Info().Stack()
		case zerolog.WarnLevel:
			e1 = loggerWarn.Warn().Stack()
		case zerolog.ErrorLevel:
			e1 = loggerError.Error().Stack()
		case zerolog.TraceLevel:
			e1 = loggerTrace.Trace().Stack()
		default:
			//默认输出到stdError
			//e1 = log.Logger.WithLevel(l).Stack().Caller(2)
		}
		e1.Msg(msg)
	})
	log.Logger = log.Logger.Hook(levelInfoHook)
}

func initLoggerFile(logDir string, fileName string) *zerolog.Logger {
	var l zerolog.Logger
	logFile := filepath.Join(logDir, fileName+"-%Y%m%d.log")
	linkName := filepath.Join(logDir, fileName+".log")
	if file, err := rotatelogs.New(logFile, rotatelogs.WithLinkName(linkName), rotatelogs.WithMaxAge(24*time.Hour), rotatelogs.WithRotationTime(time.Hour)); err == nil {
		l = log.Logger.With().Logger()
		out := zerolog.ConsoleWriter{Out: file, TimeFormat: "2006-01-02 15:04:05.000", NoColor: true}
		out.FormatLevel = func(i interface{}) string {
			return strings.ToUpper(fmt.Sprintf(" [%s] [%-2s]", ApplicationConfig.Server.Name, i))
		}
		l = l.Output(out).With().Caller().Logger()
	}
	return &l
}

func initLogDir() {
	// 创建日志目录
	logDir := filepath.Join(".", "logs")
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		_ = os.Mkdir(logDir, os.ModePerm)
	}
	// 创建日志文件
	loggerInfo = initLoggerFile(logDir, "app-info")
	loggerDebug = initLoggerFile(logDir, "app-debug")
	loggerWarn = initLoggerFile(logDir, "app-warn")
	loggerError = initLoggerFile(logDir, "app-error")
	loggerOther = initLoggerFile(logDir, "app-other")
	loggerTrace = initLoggerFile(logDir, "app-trace")
}

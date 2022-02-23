package domain

import (
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
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
	}
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
		},
		Loki: struct {
			Host string `yaml:"host"`
		}{Host: "http://loki-service:3100"},
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

var loggerInfo zerolog.Logger
var loggerDebug zerolog.Logger
var loggerWarn zerolog.Logger
var loggerError zerolog.Logger

func InitLog() {
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
	zerolog.CallerSkipFrameCount = 3
	zerolog.TimeFieldFormat = time.RFC3339
	out := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05.000"}
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
			e1 = loggerDebug.Debug()
		case zerolog.InfoLevel:
			e1 = loggerInfo.Info()
		case zerolog.WarnLevel:
			e1 = loggerWarn.Warn()
		case zerolog.ErrorLevel:
			e1 = loggerError.Error()
		default:
			//默认输出到stdError
		}
		e1.Msg(msg)
	})
	log.Hook(levelInfoHook)
}

func initLoggerFile(logDir string, fileName string) zerolog.Logger {
	var l zerolog.Logger
	logFile := filepath.Join(logDir, fileName)
	if file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.ModePerm); err == nil {
		l = log.With().Logger()
		l.Output(file)
	}
	return l
}

func init() {
	// 创建日志目录
	logDir := filepath.Join(".", "logs")
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		_ = os.Mkdir(logDir, os.ModePerm)
	}
	// 创建日志文件
	loggerInfo = initLoggerFile(logDir, "app-info.log")
	loggerDebug = initLoggerFile(logDir, "app-debug.log")
	loggerWarn = initLoggerFile(logDir, "app-warn.log")
	loggerError = initLoggerFile(logDir, "app-error.log")

}

package domain

//Package domain's application provides application runtime config,it will read config info from resources/application.yml,
//and overwrite it from resources/application-dev.yaml if Profile equals dev.
import (
	"errors"
	"fmt"
	"github.com/lestrrat-go/file-rotatelogs"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//Profile File to be loaded application-${Profile}.yml
var Profile string

//ApplicationConfig the pointer of whole application config.the summary is as follows
//		type ServerConf struct {
//			Port    int    `yaml:"port"`
//			Name    string `yaml:"name"`
//			Module  string `yaml:"api-module"`
//			Logging struct {
//				Level string `yaml:"level"`
//			} `yaml:"logging"`
//			Limit   int `yaml:"limit"`
//			Profile struct {
//				Active string `yaml:"active"`
//			} `yaml:"profile"`
//		}
//		type AppServerConf struct {
//			Server ServerConf `yaml:"server"`
//			Loki   struct {
//				Host string `yaml:"host"`
//			} `yaml:"loki"`
//			Rc struct {
//				Host      string `yaml:"host"`
//				Relevance string `yaml:"relevance"`
//			} `yaml:"rc"`
//			Mysql struct {
//				Host     string `yaml:"host"`
//				UserName string `yaml:"user_name"`
//				Password string `yaml:"password"`
//				DataBase string `yaml:"data_base"`
//			} `yaml:"mysql"`
//		}
var ApplicationConfig *AppServerConf

type ServerConf struct {
	//Port server port ,default 31000
	Port int `yaml:"port"`
	//Name application's name ,default value isc-route-service
	Name string `yaml:"name"`
	//Module servlet context path ,default value is `route`
	Module string `yaml:"api-module"`
	//Logging server's log config
	Logging struct {
		//Level log's level by all comparable types(debug,info,warn,error,fatal,panic,trace)
		Level string `yaml:"level"`
	} `yaml:"logging"`
	//Limit server's requests per second,default value is 512/s
	Limit   int `yaml:"limit"`
	Profile struct {
		Active string `yaml:"active"`
	} `yaml:"profile"`
	//Compatible default vale is false.if it is true,it will let GetRouteInfoConfigPath read Mysql Database, of isc-route-service 3.x
	Compatible bool `yaml:"compatible"`
}
type AppServerConf struct {
	Server ServerConf `yaml:"server"`
	Loki   struct {
		Host   string `yaml:"host"`
		Enable bool   `yaml:"enable"`
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

//newDefaultConf 初始化默认值
func newDefaultConf() *AppServerConf {
	return &AppServerConf{
		Server: ServerConf{
			Port:   31000,
			Name:   "isc-route-service",
			Module: "route",
			Logging: struct {
				Level string `yaml:"level"`
			}(struct{ Level string }{Level: "INFO"}),
			Limit:      512,
			Compatible: false,
		},
		Loki: struct {
			Host   string `yaml:"host"`
			Enable bool   `yaml:"enable"`
		}{Host: "http://loki-service:3100", Enable: false},
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

//ReadProfileYaml 读取application.yml文件，如果Profile不为空,或者server.profile.active不为空，则再次读取对应的application-${profile}.yml
//如果map相同，则覆盖更新
func ReadProfileYaml() {
	ApplicationConfig.readApplicationYaml("")
	if Profile == "" {
		Profile = os.Getenv("profiles")
	}
	if Profile == "" {
		Profile = "default"
	}
	ApplicationConfig.readApplicationYaml(Profile)
}
func (conf *AppServerConf) readApplicationYaml(act string) {
	pwd, _ := os.Getwd()
	path := "application.yml"
	if act != "" {
		path = fmt.Sprintf("application-%s.yml", act)
	}
	fp := filepath.Join(pwd, path)
	if act == "default" {
		fp = filepath.Join(pwd, "config", path)
	}
	log.Info().Msgf("加载[%s]文件", fp)
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
		if Profile == "" {
			Profile = conf.Server.Profile.Active
		}
	}
}

var loggerTrace *zerolog.Logger
var loggerInfo *zerolog.Logger
var loggerDebug *zerolog.Logger
var loggerWarn *zerolog.Logger
var loggerError *zerolog.Logger
var loggerOther *zerolog.Logger

//InitLog 初始化日志设置
func InitLog() {
	InitWriter()
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
	sw := &syslogWriter{}
	writer := zerolog.MultiLevelWriter(out, zerolog.MultiLevelWriter(zerolog.SyslogLevelWriter(sw)))
	log.Logger = log.Logger.Output(writer).With().Caller().Logger()
}

//getWriter 根据参数logDir,fileName 确定输出流
func getWriter(logDir, fileName string) io.Writer {
	logFile := filepath.Join(logDir, fileName+"-%Y%m%d.log")
	linkName := filepath.Join(logDir, fileName+".log")
	file, err := rotatelogs.New(logFile, rotatelogs.WithLinkName(linkName), rotatelogs.WithMaxAge(24*time.Hour), rotatelogs.WithRotationTime(time.Hour))
	if err != nil {
		return nil
	}
	return file
}

func initLoggerFile(logDir string, fileName string) *zerolog.Logger {
	var l zerolog.Logger
	if file := getWriter(logDir, fileName); file != nil {
		l = log.Logger.With().Logger()
		//out := zerolog.ConsoleWriter{Out: file, TimeFormat: "2006-01-02 15:04:05.000", NoColor: true}
		//out.FormatLevel = func(i interface{}) string {
		//	return strings.ToUpper(fmt.Sprintf(" [%s] [%-2s]", ApplicationConfig.Server.Name, i))
		//}
		l = l.Output(file).With().Logger()
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

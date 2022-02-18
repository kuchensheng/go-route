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
	applicationConfig := &AppServerConf{}
	applicationConfig.readApplicationYaml("")
	act := ApplicationConfig.Server.Profile.Active
	if act != "" {
		ApplicationConfig.readApplicationYaml(act)
	}
	level := ApplicationConfig.Server.Logging.Level
	l := zerolog.InfoLevel
	if level != "" {
		l1, err := zerolog.ParseLevel(level)
		if err != nil {
			log.Warn().Msgf("日志设置异常，将使用默认级别 INFO")
		} else {
			l = l1
		}
		zerolog.SetGlobalLevel(l)
		zerolog.TimeFieldFormat = time.RFC3339
		out := zerolog.ConsoleWriter{Out: os.Stdout}
		out.FormatLevel = func(i interface{}) string {
			return strings.ToUpper(fmt.Sprintf(" [%s] [%-6s] ", ApplicationConfig.Server.Name, i))
		}

		log.Logger = log.Logger.Output(out)
	}
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

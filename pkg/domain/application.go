package domain

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

var Profile string

var ApplicationConfig *AppServerConf
var RedisClient *redis.Client

type AppServerConf struct {
	Server struct {
		Port    int `yaml:"port"`
		Servlet struct {
			ContextPath string `yaml:"context-path"`
		} `yaml:"servlet"`
		Tcp struct {
			Port int `yaml:"port"`
		} `yaml:"tcp"`
		Udp struct {
			Port int `yaml:"port"`
		} `yaml:"udp"`
		Profile struct {
			Active string `yaml:"active"`
		} `yaml:"profile"`
	} `yaml:"server"`
	Redis struct {
		MasterName string   `yaml:"master-name"`
		Addrs      []string `yaml:"addrs"`
		Password   string   `yaml:"password"`
		DB         int      `yaml:"db"`
	} `yaml:"redis"`
	Loki struct {
		Host string `yaml:"host"`
	} `yaml:"loki"`
}
type ApplicationConf struct {
	Conf map[string]interface{}
}

func init() {
	//初始化ApplicationConf
	ApplicationConfig = &AppServerConf{}
	ApplicationConfig.readApplicationYaml("")
	act := ApplicationConfig.Server.Profile.Active
	if act != "" {
		ApplicationConfig.readApplicationYaml(act)
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
		//加载配置
		addr := ApplicationConfig.Redis.Addrs
		if len(addr) > 0 {
			initRedisClient()
		}
	}
}

func initRedisClient() {
	log.Info().Msgf("初始化redis客户端")
	masterName := ApplicationConfig.Redis.MasterName
	//redisValueInterface := ApplicationConfig.Conf["redis"]
	//redisValue ,ok := redisValueInterface.(interface{})
	//if ok {
	//	if redisMap,ok := redisValue.(map[interface{}]interface{});ok {
	//		if m,ok := redisMap["master"];ok {
	//			masterName = m.(string)
	//		}
	//		if addrs,ok := redisMap["addrs"];ok {
	//			sa := sentinelAddrs[1:]
	//			for _,a := range addrs.([]interface{}){
	//				sa = append(sa,a.(string))
	//			}
	//			sentinelAddrs = sa
	//		}
	//		if pwd,ok := redisMap["password"];ok {
	//			password = pwd.(string)
	//		}
	//		if db1, ok := redisMap["db"];ok {
	//			db = db1.(int)
	//		}
	//	}
	//}

	if masterName != "" {
		log.Info().Msgf("初始化哨兵模式客户端")
		sf := &redis.FailoverOptions{
			MasterName:    masterName,
			SentinelAddrs: ApplicationConfig.Redis.Addrs,
			Password:      ApplicationConfig.Redis.Password,
			DB:            ApplicationConfig.Redis.DB,
			IdleTimeout:   100 * time.Millisecond,
			MaxRetries:    -1,
		}
		RedisClient = redis.NewFailoverClient(sf)
	} else {
		log.Info().Msgf("初始化单机模式客户端")
		RedisClient = redis.NewClient(&redis.Options{
			Addr:     ApplicationConfig.Redis.Addrs[0],
			Password: ApplicationConfig.Redis.Password,
			DB:       ApplicationConfig.Redis.DB,
		})
	}
	ctx := context.Background()
	err := RedisClient.Ping(ctx).Err()
	if err != nil {
		log.Fatal().Msgf("redis初始化失败%v", err)
	}

}

package plugins

import (
	"encoding/json"
	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
)

type RedisConf struct {
	Redis struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	} `yaml:"redis"`
}

// InitRedisClient returns a client to the Redis Server specified by configName.
func InitRedisClient(configName string) *redis.Client {
	opt := &RedisConf{}
	err := ReadYamlToStruct(configName, opt)
	if err != nil {
		log.Error().Msgf("redis客户端初始化失败\n%v", err)
	}

	redisOpt := &redis.Options{
		Addr:     opt.Redis.Addr,
		Password: opt.Redis.Password,
		DB:       opt.Redis.DB,
	}

	return redis.NewClient(redisOpt)
}

//InitFailoverClient returns a Redis client that uses Redis Sentinel
// for automatic failover. It's safe for concurrent use by multiple
// goroutines.
func InitFailoverClient(configFileName string) *redis.Client {
	opt := &redis.FailoverOptions{}
	err := ReadYamlToStruct(configFileName, opt)
	if err != nil {
		log.Error().Msgf("redisFailoverClient初始化失败\n%v", err)
	}

	return redis.NewFailoverClient(opt)
}

func InitClusterClient(configFileName string) *redis.ClusterClient {
	opt := &redis.ClusterOptions{}
	err := ReadYamlToStruct(configFileName, opt)
	if err != nil {
		log.Error().Msgf("redisClusterClient初始化失败\n%v", err)
	}
	return redis.NewClusterClient(opt)
}

func InitFailoverClusterClient(configFileName string) *redis.ClusterClient {
	opt := &redis.FailoverOptions{}
	err := ReadYamlToStruct(configFileName, opt)
	if err != nil {
		log.Error().Msgf("redisFailoverClusterClient初始化失败\n%v", err)
	}
	return redis.NewFailoverClusterClient(opt)
}

func ReadYamlToStruct(fileName string, conf interface{}) error {
	return readFileTo(fileName, conf, func(data []byte, conf interface{}) error {
		err := yaml.Unmarshal(data, conf)
		if err != nil {
			log.Error().Msgf("yaml转配置[%v]异常\n%v", conf, err)
			return err
		}
		return nil
	})
}

func ReadJsonToStruct(fileName string, conf interface{}) error {
	return readFileTo(fileName, conf, func(data []byte, conf interface{}) error {
		err := json.Unmarshal(data, conf)
		if err != nil {
			log.Error().Msgf("json转配置[%v]异常\n%v", conf, err)
			return err
		}
		return nil
	})
}

func readFileTo(fileName string, conf interface{}, handler func(data []byte, conf interface{}) error) error {
	pwd, _ := os.Getwd()
	fp := filepath.Join(pwd, "data", "plugins", fileName)
	log.Info().Msgf("读取文件[%s]", fp)
	data, err := ioutil.ReadFile(fp)
	if err != nil {
		log.Error().Msgf("读取文件[%s]异常\n%v", fp, err)
		return err
	}
	return handler(data, conf)
}

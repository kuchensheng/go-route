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

// InitRedisClient returns a client to the Redis Server specified by configName.
func InitRedisClient(configName string) *redis.Client {
	opt := &redis.Options{}
	err := ReadYamlToStruct(configName, opt)
	log.Fatal().Msgf("redis初始化失败\n", err)
	return redis.NewClient(opt)
}

//InitFailoverClient returns a Redis client that uses Redis Sentinel
// for automatic failover. It's safe for concurrent use by multiple
// goroutines.
func InitFailoverClient(configFileName string) *redis.Client {
	opt := &redis.FailoverOptions{}
	err := ReadYamlToStruct(configFileName, opt)
	log.Fatal().Msgf("redis初始化失败\n", err)
	return redis.NewFailoverClient(opt)
}

func InitClusterClient(configFileName string) *redis.ClusterClient {
	opt := &redis.ClusterOptions{}
	err := ReadYamlToStruct(configFileName, opt)
	log.Fatal().Msgf("redis初始化失败\n", err)
	return redis.NewClusterClient(opt)
}

func InitFailoverClusterClient(configFileName string) *redis.ClusterClient {
	opt := &redis.FailoverOptions{}
	err := ReadYamlToStruct(configFileName, opt)
	log.Fatal().Msgf("redis初始化失败\n", err)
	return redis.NewFailoverClusterClient(opt)
}

func ReadYamlToStruct(fileName string, conf any) error {
	return readFileTo(fileName, conf, func(data []byte, conf any) error {
		err := yaml.Unmarshal(data, conf)
		if err != nil {
			log.Error().Msgf("yaml转配置[%v]异常\n%v", conf, err)
			return err
		}
		return nil
	})
}

func ReadJsonToStruct(fileName string, conf any) error {
	return readFileTo(fileName, conf, func(data []byte, conf any) error {
		err := json.Unmarshal(data, conf)
		if err != nil {
			log.Error().Msgf("json转配置[%v]异常\n%v", conf, err)
			return err
		}
		return nil
	})
}

func readFileTo(fileName string, conf any, handler func(data []byte, conf any) error) error {
	pwd, _ := os.Getwd()
	fp := filepath.Join(pwd, fileName)
	log.Info().Msgf("读取文件[%s]", fp)
	data, err := ioutil.ReadFile(fp)
	if err != nil {
		log.Error().Msgf("读取文件[%s]异常\n%v", fp, err)
		return err
	}
	return handler(data, conf)
}

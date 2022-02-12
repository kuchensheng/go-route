package domain

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
)

var Profile string

var ApplicationConfig *ApplicationConf

const ServerProfilesActive = "server.profiles.active"

type ApplicationConf struct {
	Conf map[string]interface{}
}

func init() {
	//初始化ApplicationConf
	ApplicationConfig = &ApplicationConf{}
	readApplicationYaml("")
	if act, ok := ApplicationConfig.Conf[ServerProfilesActive]; ok {
		readApplicationYaml(*act.(*string))
	}
}
func ReadProfileYaml() {
	if Profile != "" {
		readApplicationYaml(Profile)
	}
}
func readApplicationYaml(act string) map[string]interface{} {
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
		result := make(map[string]interface{})
		err = yaml.Unmarshal(data, &result)
		if err != nil {
			log.Fatal().Msgf("application.yml解析错误", err)
		}
		//加载配置
		if result != nil {
			for k, v := range result {
				ApplicationConfig.Conf[k] = v
			}
		}
		return result
	}
	return nil
}

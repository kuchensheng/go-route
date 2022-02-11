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

var Application = make(map[string]interface{})
var Profile string

func init() {
	//初始化ApplicationConf
	pwd, _ := os.Getwd()
	handler := func(path string) map[string]interface{} {
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
			return result
		}
		return nil
	}
	appendItem := func(res map[string]interface{}) {
		if res != nil {
			for k, v := range res {
				Application[k] = v
			}
		}
	}
	appendItem(handler("application.yml"))
	if Profile != "" {
		//文件二次读取
		appendItem(handler(fmt.Sprintf("application-%s.yml", Profile)))
	}

}

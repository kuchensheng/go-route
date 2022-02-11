package utils

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
	"io"
	"os"
)

func OpenFileAndUnmarshal(filePath string, lc interface{}) error {
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_SYNC, 0666)
	if err == nil {
		data, err := io.ReadAll(file)
		if err == nil {
			err = json.Unmarshal(data, &lc)
			if err != nil {
				log.Warn().Msgf("配置文件读取失败%v", err)
				return err
			}
		} else {
			log.Warn().Msgf("配置文件读取失败%v", err)
			return err
		}
	} else {
		log.Warn().Msgf("读取配置文件license.json异常%v", err)
		return err
	}
	return nil
}

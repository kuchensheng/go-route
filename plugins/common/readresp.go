package plugins

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"net/http"
)

//ReadResp 从响应体中读取数据并转换成对应的数据格式
func ReadResp(resp *http.Response, result any) error {
	body := resp.Body
	defer body.Close()
	data, err := ioutil.ReadAll(body)
	if err != nil {
		log.Error().Msgf("读取服务响应体数据异常:%v", err)
		return err
	} else {
		err = json.Unmarshal(data, &result)
		if err != nil {
			log.Error().Msgf("服务响应体数据转化为结构体异常:%v", err)
			return err
		}
	}
	return nil
}

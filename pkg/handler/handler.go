package handler

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"isc-route-service/pkg/domain"
	"isc-route-service/pkg/exception"
	"net/http"
	"os"
	"strings"
)

type RouteInfo struct {
	path        string   `json:"path"`
	serviceId   string   `json:"serviceId"`
	url         string   `json:"url"`
	protocol    string   `json:"protocol"`
	excludeUrl  string   `json:"excludeUrl"`
	excludeUrls []string `json:"ExcludeUrls"`
	specialUrl  string   `json:"specialUrl"`
	specialUrls []string `json:"specialUrls"`
}

func UpdateRoute(c *gin.Context) {
	//获取请求体
	b := c.Request.Body
	defer b.Close()
	data, err := ioutil.ReadAll(b)
	if err != nil {
		log.Warn().Stack().Msgf("请求体读取异常\n%v", err)
		c.JSON(http.StatusBadRequest, exception.BusinessException{
			Code:    1040000,
			Message: fmt.Sprintf("请求体读取异常%v", err),
		})
		return
	}
	ri := &RouteInfo{}
	err = json.Unmarshal(data, ri)
	if err != nil {
		log.Warn().Stack().Msgf("请求体不合法,请求内容[%s],异常信息:\n%v", string(data), err)
		c.JSON(http.StatusBadRequest, err)
		return
	}
	checkUrl(ri, c)
	//添加路由信息
	if saveOrUpdate(*ri) != nil {
		c.JSON(http.StatusInternalServerError, err)
	}
}

func saveOrUpdate(ri RouteInfo) error {
	fp := domain.GetRouteInfoConfigPath()
	file, err := os.OpenFile(fp, os.O_RDWR|os.O_SYNC, 0666)
	if err != nil {
		log.Error().Stack().Msgf("路由配置文件打开异常:%v", err)
		return err
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Error().Stack().Msgf("路由配置信息读取异常:%v", err)
		return err
	}
	var routes []RouteInfo
	err = json.Unmarshal(data, &routes)
	if err != nil {
		log.Error().Stack().Msgf("路由配置信息转换异常:%v", err)
		return err
	}
	idx := func() int {
		for idx, r := range routes {
			if ri.serviceId == r.serviceId {
				return idx
			}
		}
		return -1
	}()
	if idx < 0 {
		//新增路由信息
		routes = append(routes, ri)
	} else {
		//更新路由信息
		routes[idx] = ri
	}

	newData, err := json.Marshal(routes)
	if err != nil {
		log.Error().Stack().Msgf("路由配置信息更新异常:%v", err)
		return err
	}
	_, err = file.WriteString(string(newData))
	if err != nil {
		log.Error().Stack().Msgf("更新路由配置文件异常%v", err)
		return err
	}
	return nil
}

func checkUrl(ri *RouteInfo, c *gin.Context) {
	if ri.url == "" || strings.TrimSpace(ri.url) == "" {
		c.JSON(http.StatusBadRequest, exception.BusinessException{
			Code:    1040001,
			Message: "url不能为空",
		})
		return
	}
	if !(strings.HasPrefix(ri.url, "http") || strings.HasPrefix(ri.url, "ws")) {
		c.JSON(http.StatusBadRequest, exception.BusinessException{
			Code:    1040002,
			Message: "url必须以http/https/ws/wss开头",
		})
		return
	}
	if !(strings.HasSuffix(ri.url, "/*") || strings.HasSuffix(ri.url, "/**")) {
		c.JSON(http.StatusBadRequest, exception.BusinessException{
			Code:    1040002,
			Message: "url必须以/*或/**结尾",
		})
		return
	}
}

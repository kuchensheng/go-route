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
	"sort"
	"strings"
)

const (
	REFRESH_URI    = "/api/route/refreshRoute"
	STATUS_URI     = "/api/route/system/status"
	ROUTE_LIST_URI = "/api/route/list"
)

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
	ri := domain.RouteInfo{}
	err = json.Unmarshal(data, &ri)
	if err != nil {
		log.Warn().Stack().Msgf("请求体不合法,请求内容[%s],异常信息:\n%v", string(data), err)
		c.JSON(http.StatusBadRequest, err)
		return
	}
	checkUrl(ri, c)
	checkPath(ri, c)
	if ri.ExcludeUrl != "" {
		ri.ExcludeUrls = strings.Split(ri.ExcludeUrl, ";")
	}
	if ri.SpecialUrl != "" {
		ri.SpecialUrls = strings.Split(ri.SpecialUrl, ";")
	}
	//添加路由信息
	if saveOrUpdate(ri) != nil {
		c.JSON(http.StatusInternalServerError, err)
	}
}

func saveOrUpdate(ri domain.RouteInfo) error {
	fp := domain.GetRouteInfoConfigPath()
	file, err := os.OpenFile(fp, os.O_RDWR|os.O_SYNC, 0644)
	if err != nil {
		log.Error().Stack().Msgf("路由配置文件打开异常:%v", err)
		return err
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Error().Stack().Msgf("路由配置信息读取异常:%v", err)
		return err
	}
	file.Close()

	var routes []domain.RouteInfo
	err = json.Unmarshal(data, &routes)
	if err != nil {
		log.Error().Stack().Msgf("路由配置信息转换异常:%v", err)
		return err
	}
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].Id < routes[j].Id
	})
	maxId := routes[len(routes)-1].Id

	idx := func() int {
		for idx, r := range routes {
			if ri.ServiceId == r.ServiceId {
				return idx
			}
		}
		return -1
	}()
	if idx < 0 {
		//新增路由信息
		ri.Id = maxId + 1
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

	file1, err := os.OpenFile(fp, os.O_WRONLY|os.O_TRUNC, 0644)
	n, _ := file1.Seek(0, 0)
	_, err = file1.WriteAt(newData, n)
	if err != nil {
		log.Error().Stack().Msgf("更新路由配置文件异常%v", err)
		return err
	}
	defer file1.Close()
	return nil
}

func checkUrl(ri domain.RouteInfo, c *gin.Context) {
	if ri.Url == "" || strings.TrimSpace(ri.Url) == "" {
		c.JSON(http.StatusBadRequest, exception.BusinessException{
			Code:    1040001,
			Message: "url不能为空",
		})
		return
	}
	if !(strings.HasPrefix(ri.Url, "http") || strings.HasPrefix(ri.Url, "ws")) {
		c.JSON(http.StatusBadRequest, exception.BusinessException{
			Code:    1040002,
			Message: "url必须以http/https/ws/wss开头",
		})
		return
	}
}

func checkPath(ri domain.RouteInfo, c *gin.Context) {
	if !(strings.HasSuffix(ri.Path, "/*") || strings.HasSuffix(ri.Path, "/**")) {
		c.JSON(http.StatusBadRequest, exception.BusinessException{
			Code:    1040002,
			Message: "url必须以/*或/**结尾",
		})
		return
	}
}

func RouteList(c *gin.Context) {
	c.JSON(http.StatusOK, domain.RouteInfos)
}

func IscRouteHandler(c *gin.Context) bool {
	uri := c.Request.RequestURI
	switch uri {
	case ROUTE_LIST_URI:
		log.Info().Msgf("获取路由列表")
		RouteList(c)
		return true
	case REFRESH_URI:
		log.Info().Msgf("更新路由列表")
		UpdateRoute(c)
		return true
	case STATUS_URI:
		c.JSON(http.StatusOK, `{}`)
		return true
	default:
		return false
	}
}

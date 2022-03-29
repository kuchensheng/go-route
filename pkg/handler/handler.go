package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"isc-route-service/pkg/domain"
	"isc-route-service/pkg/exception"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	REFRESH_URI    = "/api/route/refreshRoute"
	STATUS_URI     = "/api/route/system/status"
	ROUTE_LIST_URI = "/api/route/list"
	ADD_PLUGINS    = "/api/route/plugins"
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
	updateRoutes := func(routes []domain.RouteInfo, svcId string) []domain.RouteInfo {
		var result []domain.RouteInfo
		for _, item := range routes {
			if item.ServiceId == svcId {
				result = append(result, item)
			}
		}
		return result
	}
	if list := updateRoutes(domain.RouteInfos, ri.ServiceId); len(list) > 0 {
		log.Info().Msgf("更新路由规则，serviceId = %s,requestBody =%v", ri.ServiceId, string(data))
		for _, item := range list {
			if ri.Enabled != nil {
				enabled := *ri.Enabled
				item.Enabled = &enabled
			}

			if ri.Path != "" && len(ri.Path) > 0 {
				log.Info().Msgf("更新了path")
				item.Path = ri.Path
			}
			if ri.Url != "" && len(ri.Url) > 0 {
				log.Info().Msgf("更新了url")
				item.Url = ri.Url
			}
			if ri.ExcludeUrl != "" && len(ri.ExcludeUrl) > 0 {
				log.Info().Msgf("更新了ExcludeUrl")
				item.ExcludeUrl = ri.ExcludeUrl
			}
			if ri.SpecialUrl != "" && len(ri.SpecialUrl) > 0 {
				item.SpecialUrl = ri.SpecialUrl
			}
			if ri.Predicates != nil {
				item.Predicates = ri.Predicates
			}
			if ri.AppCode != "" && len(ri.AppCode) > 0 {
				item.AppCode = ri.AppCode
			}
			item.UpdateTime = time.Now().Format(time.RFC3339)
			checkUrl(item, c)
			checkPath(item, c)
			if item.ExcludeUrl != "" {
				item.ExcludeUrls = strings.Split(ri.ExcludeUrl, ";")
			}
			if item.SpecialUrl != "" {
				item.SpecialUrls = strings.Split(ri.SpecialUrl, ";")
			}
			if err1 := saveOrUpdate(item); err1 != nil {
				c.JSON(http.StatusInternalServerError, err)
				break
			}
		}
	} else {
		log.Info().Msgf("新增路由规则")
		*ri.Enabled = 1
		checkUrl(ri, c)
		checkPath(ri, c)
		if ri.ExcludeUrl != "" {
			ri.ExcludeUrls = strings.Split(ri.ExcludeUrl, ";")
		}
		if ri.SpecialUrl != "" {
			ri.SpecialUrls = strings.Split(ri.SpecialUrl, ";")
		}
		if err1 := saveOrUpdate(ri); err1 != nil {
			c.JSON(http.StatusInternalServerError, err)
		}
	}

}

func saveOrUpdate(ri domain.RouteInfo) error {
	log.Info().Msgf("更新路由配置信息,%v", ri)
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
			Message: "path必须以/*或/**结尾",
		})
		return
	}
}

func RouteList(c *gin.Context) {
	c.JSON(http.StatusOK, domain.RouteInfos)
}

func getFileFromRequest(c *gin.Context, fieldName, dir string) (string, string, error) {
	f, err := c.FormFile(fieldName)
	if err != nil {
		if fieldName == "plugin" {
			return "", dir, err
		}
		return "", dir, nil
	}

	if dir == "" {
		lastIndex := strings.LastIndex(f.Filename, ".")
		dir = f.Filename[:lastIndex]
	}
	fp := filepath.Join(".", "data", "plugins", dir, f.Filename)
	err = c.SaveUploadedFile(f, fp)
	if err != nil {
		return "", dir, err
	}

	path := filepath.Join(dir, f.Filename)
	return path, dir, nil
}

func generalPluginInfo(c *gin.Context) (*domain.PluginInfo, error) {
	pi := &domain.PluginInfo{}
	method, b := c.GetPostForm("method")
	if !b {
		return nil, errors.New("method不能为空,且必须首字母大写")
	}
	pi.Method = method
	order, _ := c.GetPostForm("order")
	lenAllPlugins := len(domain.AllPlugins)
	if !b {
		if lenAllPlugins == 0 {
			order = "0"
		} else {
			pi.Order = domain.AllPlugins[len(domain.AllPlugins)-1].Order + 1
		}
	}
	intOrder, err := strconv.Atoi(order)
	if err != nil {
		intOrder = 0
	}
	pi.Order = intOrder
	pluginType, b := c.GetPostForm("type")
	if !b {
		pluginType = "0"
	}
	intPluginType, err := strconv.Atoi(pluginType)
	if err != nil {
		intPluginType = 0
	}
	pi.Type = intPluginType
	version, _ := c.GetPostForm("version")
	pi.Version = version
	name, _ := c.GetPostForm("name")
	pi.Name = name
	return pi, nil
}

func AddPlugin(c *gin.Context) {
	dir, _ := c.GetPostForm("dir")
	pluginPath, dir, err := getFileFromRequest(c, "plugin", dir)
	if err != nil {
		log.Warn().Stack().Msgf("请求体读取异常\n%v", err)
		c.JSON(http.StatusBadRequest, exception.BusinessException{
			Code:    1040000,
			Message: fmt.Sprintf("请求参数plugin读取异常%v", err),
		})
		return
	}
	_, _, err = getFileFromRequest(c, "config", dir)
	if err != nil {
		log.Warn().Stack().Msgf("请求体读取异常\n%v", err)
		c.JSON(http.StatusBadRequest, exception.BusinessException{
			Code:    1040000,
			Message: fmt.Sprintf("请求参数config读取异常%v", err),
		})
		return
	}

	pi, err := generalPluginInfo(c)
	if err != nil {
		log.Warn().Stack().Msgf("请求体不合法,异常信息:\n%v", err)
		c.JSON(http.StatusBadRequest, err)
		return
	}

	pi.Path = pluginPath
	if pi.Name == "" {
		pi.Name = dir
	}
	distinct := true
	//插件去重
	for _, item := range domain.AllPlugins {
		if item.Path == pi.Path {
			distinct = false
			break
		}
	}
	if !distinct {
		c.JSON(http.StatusBadRequest, exception.BusinessException{
			Code:    1040400,
			Message: "插件已存在，不能重复上传",
		})
		return
	}
	domain.AllPlugins = append(domain.AllPlugins, *pi)
	configContent, err := json.Marshal(domain.AllPlugins)
	if err != nil {
		if err != nil {
			log.Warn().Stack().Msgf("插件配置信息变更异常:\n%v", err)
			c.JSON(http.StatusBadRequest, err)
			return
		}
	}
	file, err := os.OpenFile("./data/resources/plugins.json", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		log.Warn().Stack().Msgf("插件配置文件打开异常:\n%v", err)
		c.JSON(http.StatusBadRequest, err)
		return
	}
	defer file.Close()
	_, err = file.Write(configContent)
	if err != nil {
		log.Warn().Stack().Msgf("插件配置文件更新异常:\n%v", err)
		c.JSON(http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, exception.BusinessException{
		Code:    0,
		Message: "插件安装成功",
	})
}

func IscRouteHandler(c *gin.Context) bool {
	uri := c.Request.RequestURI
	switch uri {
	case ROUTE_LIST_URI:
		log.Debug().Msgf("获取路由列表")
		RouteList(c)
		return true
	case REFRESH_URI:
		log.Info().Msgf("更新路由列表")
		UpdateRoute(c)
		return true
	case STATUS_URI:
		c.JSON(http.StatusOK, `{}`)
		return true
	case ADD_PLUGINS:
		log.Info().Msgf("添加/更新插件信息")
		AddPlugin(c)
		return true
	default:
		return false
	}
}

func PromHandler(handler http.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler.ServeHTTP(c.Writer, c.Request)
	}
}

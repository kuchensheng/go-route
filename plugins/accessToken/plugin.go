//go:build (linux && cgo) || (darwin && cgo) || (freebsd && cgo)
// +build linux,cgo darwin,cgo freebsd,cgo

package main

//Valid access token 验证
func Valid(req *http.Request, target []byte) error {
	uri := req.URL.Path
	if !plugins.IsInSlice(ac.AccessToken.urls, uri) {
		return nil
	}
	iscAccessToken := req.Header.Get(IscAccessTokenKey)
	log.Info().Msgf("公共服务请求,isc-access-token=%s", iscAccessToken)
	err := &plugins.BusinessException{
		StatusCode: 403,
		Code:       1040403,
		Message:    "应用无权限访问",
	}
	if iscAccessToken == "" {
		return err
	} else {
		//todo 需要连接redis
		value := redisClient.Get(context.Background(), fmt.Sprintf("%s%s", GatewayApiServiceAccessTokenRedisKeyPrefix, iscAccessToken)).Val()
		if value == "" {
			return err
		}
	}
	return nil

}

# 路由服务
## 简介
## 原理
利用httputils.ReverseProxy + goroutine + channel 瞬发请求，降低在路由服务的时间损耗
## 模块
- docker 编译运行模块，用于docker构建，包含docker构建命令
- pkg 源码模块
  - domain 领域模型，路由规则加载、应用初始话、插件信息加载等内容
  - exception 异常处理模型
  - handler 路由自身业务模型
  - middleware 中间件加载模型
  - proxy 代理模型
  - ratelimit 限流拦截处理
  - tracer 链路跟踪处理
- plugins 插件信息
  - accessToken 
  - appauth 应用授权鉴权
  - license 授权鉴权
  - login 登录鉴权
  - common 插件的公共内容，慎重改动，改动common需要重新构建所有的相关plugin信息
- utils 一些工具
- watcher 监听器，监听文件变化、消息推送或者配置中心内容变更等
- resources 静态或配置资源

## 版本特性
- 兼容：支持路由规则与Mysql数据同步，同步仅一次
- 轻量：内存占用较少，空载仅10M
- 支持动态加载路由规则和插件信息
- 支持插件上传
- 支持路由规则动态更新
- 支持自定义截取路由规则，自定义Predicate
- 支持License鉴权、登录鉴权、限流等
- 支持多租户处理与应用鉴权
- 支持AccessToken校验
- 暂不支持非http协议转发和http协议转换为其他RPC协议
- 支持自定义登陆鉴权白名单
- 支持自定义超时延迟白名单，延迟到5M
## 运行方式
**注意**
可运行文件的配置信息放置到docker/bin目录下，如果程序无法运行，请自行拷贝对应文件到根目录下
```ssh
# 运行时可以指明配置未知，否则会加载默认值
 -conf /mnt/d/worksapace/go/isc-route-service/resources/routeInfo.json -plugins /mnt/d/worksapace/go/isc-route-service/resources/plugins.json
```
## 插件上传方法
```text
curl --silent --location --request POST 'http://172.21.169.195:31000/api/route/plugins' \
--header 'isc-api-version: 3.0' \
--form 'plugin=@"/D:/worksapace/go/isc-route-service/docker/bin/plugins/appauth/plugin.so"' \
--form 'method="Valid"' \
--form 'dir="appauth"' \
--form 'name="应用授权鉴权"' \
--form 'order="4"' \
--form 'type="0"' \
--form 'version=""'
```
## 路由规则更新方法
```text
curl --silent --location --request POST 'http://10.30.30.78:38080/api/route/refreshRoute' \
--header 'Content-Type: application/json' \
--data-raw '{
            "path": "/api/apix/**",
            "serviceId": "isc-apix-service",
            "url": "http://isc-apix-service:31001",
            "excludeUrl":"/api/apix/**",
            "specialUrl":"/api/apix/**",
            "predicate":["stripPrefix=2"]
}'
```

##错误码
异常码均遵循指令集异常码规范，路由服务的code前缀1040~,例如1040400 = BadRequest,1040502 = BadGateway等
无其他特殊异常码
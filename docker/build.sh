TAG=${1:-latest}
echo "$TAG"
export GO111MODULE ON

echo "删除原有可执行文件 server"
rm server
echo "开始构建新的可执行文件 server"

GOARCH=amd64 GOOS=linux go build -gcflags "all=-N -l" -ldflags "-s -w" -o server ../main.go
# shellcheck disable=SC1009
size() {
  stat -c %s "$1" | tr -d '\n'
}
echo "执行压缩命令upx,压缩前文件大小:" + `size "server"`
upx -o  s1 server
rm server
mv s1 server


echo "压缩后的文件大小" + `size "server"`
echo "构建docker镜像,TAG =$TAG"

#构建镜像前删除本地的data目录和logs目录
rm -rf logs
rm -rf data/plugins
rm -rf data/resources
docker build -t 10.30.30.22:9080/isyscore/isc-route-service:"$TAG" . && docker push 10.30.30.22:9080/isyscore/isc-route-service:"$TAG"
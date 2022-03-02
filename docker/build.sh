TAG=${1:-latest}
echo "$TAG"
export GO111MODULE ON

echo "删除原有可执行文件 bin/server"
rm bin/server
echo "开始构建新的可执行文件 bin/server"

GOARCH=amd64 GOOS=linux go build -gcflags "all=-N -l" -ldflags "-s -w" -o bin/server ../main.go
# shellcheck disable=SC1009
size() {
  stat -c %s "$1" | tr -d '\n'
}
echo "执行压缩命令upx,压缩前文件大小:" + `size "bin/server"`
upx -o  bin/s1 bin/server
rm bin/server
mv bin/s1 bin/server


echo "压缩后的文件大小" + `size "bin/server"`
echo "构建docker镜像,TAG =$TAG"
docker build -t 10.30.30.22:9080/isyscore/isc-route-service:"$TAG" . && docker push 10.30.30.22:9080/isyscore/isc-route-service:"$TAG"
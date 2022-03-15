TAG=${1:-latest}
echo "$TAG"
export GO111MODULE ON \

echo "构建插件"
cd ../plugins \

# shellcheck disable=SC2045
for dir in $(ls ./)
do
  # shellcheck disable=SC2107
  if [ -d $dir ] && [ $dir != "common" ]; then
      ./compile.sh $dir
  fi

done

cd ../docker

echo "删除文件夹files"
rm -rf ./files
echo "删除文件夹ld"
rm -rf ./ld
echo "删除文件application.yml"
rm -f ./application.yml
echo "删除原有可执行文件 isc-route-service"
rm -f ./isc-route-service

echo "拷贝文件夹files"
cp -rf ../files .
echo "拷贝文件夹ld"
cp -rf ../ld .
echo  "拷贝application.yml"
cp -f ../application.yml .


echo "删除原有可执行文件 isc-route-service"
rm isc-route-service
echo "开始构建新的可执行文件 isc-route-service"

GOARCH=amd64 GOOS=linux go build -gcflags "all=-N -l" -ldflags "-s -w" -o isc-route-service ../main.go
# shellcheck disable=SC1009
size() {
  stat -c %s "$1" | tr -d '\n'
}
echo "执行压缩命令upx,压缩前文件大小:" + `size "isc-route-service"`
upx -o  s1 isc-route-service
rm isc-route-service
mv s1 isc-route-service


echo "压缩后的文件大小" + `size "isc-route-service"`
echo "构建docker镜像,TAG =$TAG"

#构建镜像前删除本地的data目录和logs目录
rm -rf logs
rm -rf data/plugins
rm -rf data/resources

docker build -t 10.30.30.22:9080/isyscore/isc-route-service:"$TAG" . && docker push 10.30.30.22:9080/isyscore/isc-route-service:"$TAG"
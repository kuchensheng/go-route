echo "删除../docker/bin/plugins/$1/plugin.so"
rm -f ../docker/bin/plugins/"$1"/plugin.so
echo "构建 $1/plugin.so"

GOARCH=amd64 GOOS=linux go build -gcflags "all=-N -l" -ldflags "-s -w" -o ../docker/bin/plugins/"$1"/plugin.so -buildmode=plugin "$1"/plugin.go

size() {
  stat -c %s "$1" | tr -d '\n'
}
echo "构建完成，动态链接库大小" + `size "../docker/bin/plugins/"$1"/plugin.so"`
#echo "执行压缩命令"
#upx -o  ../docker/bin/plugins/"$1"/plugin1.so ../docker/bin/plugins/"$1"/plugin.so
#
#rm ../docker/bin/plugins/"$1"/plugin.so
#
#mv ../docker/bin/plugins/"$1"/plugin1.so ../docker/bin/plugins/"$1"/plugin.so
#echo "压缩完成，动态链接库大小" + `size "../docker/bin/plugins/"$1"/plugin.so"`
#echo "构建完毕"
echo "删除../ld/plugins/$1/plugin.so"
rm -f ../ld/plugins/"$1"/plugin.so
echo "构建 $1/plugin.so" \

go build -gcflags "all=-N -l" -ldflags "-s -w" -o ../ld/plugins/"$1"/plugin.so -buildmode=plugin "$1"/plugin.go \

size() {
  stat -c %s "$1" | tr -d '\n'
}
echo "构建完成，动态链接库大小" + `size "../ld/plugins/"$1"/plugin.so"`
echo "拷贝配置文件"
# shellcheck disable=SC2045
for file in $(ls "$1"/)
do
  # shellcheck disable=SC2107
  if [ "$file" != "plugin.go" ]; then
      cp -n -r "$1/$file" ../ld/plugins/"$1"/
  fi

done
#echo "执行压缩命令"
#upx -o  ../ld/plugins/"$1"/plugin1.so ../ld/plugins/"$1"/plugin.so
#
#rm ../ld/plugins/"$1"/plugin.so
#
#mv ../ld/plugins/"$1"/plugin1.so ../ld/plugins/"$1"/plugin.so
#echo "压缩完成，动态链接库大小" + `size "../ld/plugins/"$1"/plugin.so"`
#echo "构建完毕"
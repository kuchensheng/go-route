@echo off

echo "删除..\ld\plugins\%1\plugin.dll"
DEL "/Q ..\ld\plugins\%1\plugin.dll"
echo "构建 %1%\plugin.dll"

go build -gcflags "all=-N -l" -ldflags "-s -w" -o ..\ld\plugins\"%1"\plugin.dll -buildmode=c-archive "%1"\plugin_windows.go

echo "拷贝配置文件"

for %%I in (dir "%1"/)
do
  if  %%I == "plugin_windows.go"
      copy /Y "%1%/%%I" destination ../ld/plugins/"%1"/
pause
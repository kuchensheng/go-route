package watcher

import (
	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
)

//AddWatcher 添加文件(夹)监听器
func AddWatcher(watcherPath string, handler func(filePath string)) error {
	log.Info().Msgf("监听文件(夹):%s", watcherPath)
	//创建一个监控对象
	watcher, err := fsnotify.NewWatcher()
	closeFunc := func() {
		watcher.Close()
	}
	if err != nil {
		//todo 是否使用fatal更合适
		log.Error().Msgf("监听对象创建异常%v", err)
		defer closeFunc()
	}
	//添加要监控的对象，文件或文件夹
	err = watcher.Add(watcherPath)
	if err != nil {
		//todo 是否使用fatal更合适
		log.Error().Msgf("文件(夹)监听失败\n%v", err)
		defer closeFunc()
		return err
	}
	//另起goroutine来处理监控对象
	done := make(chan bool)
	go func() {
		for {
			select {
			case ev, ok := <-watcher.Events:
				if !ok {
					return
				}
				//判断事件类型,CREATE(创建),WRITE(写入),REMOVE(删除),RENAME(重命名),CHMOD(修改权限)
				if ev.Op&fsnotify.Create == fsnotify.Create {
					log.Info().Msgf("创建文件 : %s", ev.Name)
				}
				if ev.Op&fsnotify.Write == fsnotify.Write {
					log.Info().Msgf("写入文件 ： %s", ev.Name)
					//todo 读取最新变化内容
					handler(watcherPath)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Error().Msgf("监控异常：%v", err)
			}
		}
	}()
	<-done //主协程不退出
	return nil
}

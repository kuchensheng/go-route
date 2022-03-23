package watcher

import (
	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

type Watch struct {
	Watcher *fsnotify.Watcher
}

//AddWatcher 添加文件(夹)监听器
func AddWatcher(watcherPath string, handler func(filePath string)) {
	log.Info().Msgf("监听文件(夹):%s", watcherPath)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error().Msgf("NewWatcher failed: ", err)
	}
	defer watcher.Close()
	dir, fileName := filepath.Split(watcherPath)
	done := make(chan bool)
	go func() {
		defer close(done)
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Debug().Msgf("%s %s is write:%v\n", event.Name, event.Op, event.Op&fsnotify.Write == fsnotify.Write)
				if event.Op&fsnotify.Write == fsnotify.Write && strings.HasSuffix(event.Name, fileName) {
					log.Info().Msgf("监听到文件[%s]发生了改变,将执行函数[%v]", watcherPath, runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name())
					handler(watcherPath)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Error().Msgf("error:", err)
			}
		}
	}()
	err = watcher.Add(dir)
	if err != nil {
		log.Error().Msgf("Add failed:", err)
	}
	<-done
}

package watcher

import (
	"github.com/fsnotify/fsnotify"
	"log"
	"testing"
)

func TestAddWatcher(t *testing.T) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add("/mnt/d/worksapace/go/isc-route-service/data/resources/routeInfo.json")
	if err != nil {
		log.Fatal(err)
	}
	<-done

}

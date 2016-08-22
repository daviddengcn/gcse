package gcse

import (
	"time"

	"github.com/howeyc/fsnotify"
)

func ClearWatcherEvents(watcher *fsnotify.Watcher) {
	return
	/*
		for {
			select {
			case <-watcher.Event:
			case err := <-watcher.Error:
				log.Printf("Wather.Error: %v", err)
			default:
				break
			}
		}
	*/
}

func WaitForWatcherEvents(watcher *fsnotify.Watcher) {
	time.Sleep(10 * time.Second)
	return
	/*
		for {
			select {
			case <-watcher.Event:
			case err := <-watcher.Error:
				log.Println("Wather.Error: %v", err)
			}
		}
	*/
}

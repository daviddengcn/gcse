package main

import (
	"github.com/daviddengcn/gcse"
	"github.com/howeyc/fsnotify"
	"log"
	"time"
)

func main() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("NewWatcher failed: %v", err)
	}
	for {
		if err := gcse.IndexSegments.ClearUndones(); err != nil {
			log.Printf("ClearUndones failed: %v", err)
		}

		if watcher != nil {
			gcse.ClearWatcherEvents(watcher)
			for {
				dbSegm, err := gcse.DBOutSegments.FindMaxDone()
				if err == nil && dbSegm != nil {
					break
				}
				gcse.WaitForWatcherEvents(watcher)
			}
		} else {
			time.Sleep(1 * time.Minute)
		}

		dbSegm, err := gcse.DBOutSegments.FindMaxDone()
		if err == nil && dbSegm != nil {
			if err := clearOutdatedIndex(); err != nil {
				log.Printf("clearOutdatedIndex failed: %v", err)
			}
			doIndex(dbSegm)
		}
	}
}

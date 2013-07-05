package gcse

import (
	"encoding/json"
	"fmt"
	"github.com/daviddengcn/go-villa"
	"github.com/howeyc/fsnotify"
	"log"
	"runtime"
	"time"
)

func WriteJsonFile(fn villa.Path, data interface{}) error {
	f, err := fn.Create()
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	return enc.Encode(data)
}

func ReadJsonFile(fn villa.Path, data interface{}) error {
	f, err := fn.Open()
	if err != nil {
		return err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	return dec.Decode(data)
}

type Size int64

func (s Size) String() string {
	var unit string
	var base int64
	switch {
	case s < 1024:
		unit, base = "", 1
	case s < 1024*1024:
		unit, base = "K", 1024
	case s < 1024*1024*1024:
		unit, base = "M", 1024*1024
	case s < 1024*1024*1024*1024:
		unit, base = "G", 1024*1024*1024
	case s < 1024*1024*1024*1024*1024:
		unit, base = "T", 1024*1024*1024*1024
	case s < 1024*1024*1024*1024*1024*1024:
		unit, base = "P", 1024*1024*1024*1024*1024
	}

	remain := int64(s) / base
	if remain < 10 {
		return fmt.Sprintf("%.2f%s", float64(s)/float64(base), unit)
	}
	if remain < 100 {
		return fmt.Sprintf("%.1f%s", float64(s)/float64(base), unit)
	}

	return fmt.Sprintf("%d%s", int64(s)/base, unit)
}

func DumpMemStats() {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	log.Printf("[MemStats] Alloc: %v, TotalAlloc: %v, Sys: %v, Go: %d",
		Size(ms.Alloc), Size(ms.TotalAlloc), Size(ms.Sys),
		runtime.NumGoroutine())
}

func ClearWatcherEvents(watcher *fsnotify.Watcher) {
	return
	for {
		select {
		case <-watcher.Event:
		case err := <-watcher.Error:
			log.Println("Wather.Error: %v", err)
		default:
			break
		}
	}
}

func WaitForWatcherEvents(watcher *fsnotify.Watcher) {
	time.Sleep(10 * time.Second)
	return
	for {
		select {
		case <-watcher.Event:
		case err := <-watcher.Error:
			log.Println("Wather.Error: %v", err)
		}
	}
}

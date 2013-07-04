/*
	GCSE Crawler background program.
*/
package main

import (
	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/go-villa"
	"github.com/howeyc/fsnotify"
	"log"
	"time"
	"runtime"
	"fmt"
)

func ReadImports(watcher *fsnotify.Watcher) {
	watcher.Watch(gcse.ImportPath.S())

	for {
		for {
			if err := processImports(); err != nil {
				log.Printf("scanImports failed: %v", err)
			}

			all, _ := gcse.ImportSegments.ListAll()
			if len(all) == 0 {
				break
			}

			time.Sleep(5 * time.Second)
		}

		select {
		case <-watcher.Event:
		//log.Println("Wather.Event: %v", ev)
		case err := <-watcher.Error:
			log.Println("Wather.Error: %v", err)
		}
	}
}

const (
	fnDocDB     = "docdb"
	fnCrawlerDB = "crawler"

	kindDocDB   = "docdb"
	kindImports = "imports"
)

var (
	DocDBPath     villa.Path
	CrawlerDBPath villa.Path
)

func init() {
	DocDBPath = gcse.DataRoot.Join(fnDocDB)
	CrawlerDBPath = gcse.DataRoot.Join(fnCrawlerDB)
}

func syncDatabases() {
	dumpMemStats()
	log.Printf("Synchronizing databases to disk...")
	if err := docDB.Sync(); err != nil {
		log.Printf("docDB.Sync failed: %v", err)
	}
	if err := importsDB.Sync(); err != nil {
		log.Printf("importsDB.Sync failed: %v", err)
	}
	if err := cPackageDB.Sync(); err != nil {
		log.Printf("cPackageDB.Sync failed: %v", err)
	}
	if err := cPersonDB.Sync(); err != nil {
		log.Printf("cPersonDB.Sync failed: %v", err)
	}
	runtime.GC()
	dumpMemStats()
}

func syncLoop(gap time.Duration) {
	for {
		time.Sleep(gap)
		syncDatabases()
	}
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

func dumpMemStats() {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	log.Printf("[MemStats] Alloc: %v, TotalAlloc: %v, Sys: %v, Go: %d", 
		Size(ms.Alloc), Size(ms.TotalAlloc), Size(ms.Sys),
		runtime.NumGoroutine())
}

func dumpingStatusLoop(gap time.Duration) {
	for {
		dumpMemStats()
		time.Sleep(gap)
	}
}

func main() {
	docDB = gcse.NewMemDB(DocDBPath, kindDocDB)
	importsDB = gcse.NewTokenIndexer(DocDBPath, kindImports)

	cPackageDB = gcse.NewMemDB(CrawlerDBPath, kindPackage)
	cPersonDB = gcse.NewMemDB(CrawlerDBPath, kindPerson)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	go ReadImports(watcher)
	go CrawlEnetires()
	go syncLoop(10 * time.Minute)
	go indexLooop(1 * time.Minute)
	go dumpingStatusLoop(10 * time.Minute)

	select {}
}

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
	
	kindDocDB = "docdb"
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
}

func syncLoop(gap time.Duration) {
	for {
		time.Sleep(gap)
		syncDatabases()
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
	go syncLoop(10*time.Minute)
	go indexLooop(1*time.Minute)

	select {}
}

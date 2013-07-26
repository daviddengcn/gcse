/*
	GCSE Crawler background program.
*/
package main

import (
	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/go-villa"
	"log"
	"runtime"
	"sync"
	"time"
)

const (
	fnDocDB     = "docdb"
	fnCrawlerDB = "crawler"
)

var (
	DocDBPath     villa.Path
	CrawlerDBPath villa.Path
	AppStopTime      time.Time
)

func init() {
	DocDBPath = gcse.DataRoot.Join(fnDocDB)
	CrawlerDBPath = gcse.DataRoot.Join(fnCrawlerDB)
}

func syncDatabases() {
	gcse.DumpMemStats()
	log.Printf("Synchronizing databases to disk...")
	if err := docDB.Sync(); err != nil {
		log.Printf("docDB.Sync failed: %v", err)
	}
	if err := cPackageDB.Sync(); err != nil {
		log.Printf("cPackageDB.Sync failed: %v", err)
	}
	if err := cPersonDB.Sync(); err != nil {
		log.Printf("cPersonDB.Sync failed: %v", err)
	}
	gcse.DumpMemStats()
	runtime.GC()
	gcse.DumpMemStats()
}

func syncLoop(wg *sync.WaitGroup) {
	for AppStopTime.Sub(time.Now()) > gcse.CrawlerSyncGap {
		time.Sleep(gcse.CrawlerSyncGap)
		syncDatabases()
	}
	wg.Done()
}

func dumpingStatusLoop() {
	for time.Now().Before(AppStopTime) {
		gcse.DumpMemStats()
		time.Sleep(10 * time.Minute)
	}
}

func main() {
	log.Println("crawler started...")
	
	AppStopTime = time.Now().Add(30 * time.Minute)
	
	
	docDB = gcse.NewMemDB(DocDBPath, gcse.KindDocDB)

	cPackageDB = gcse.NewMemDB(CrawlerDBPath, kindPackage)
	cPersonDB = gcse.NewMemDB(CrawlerDBPath, kindPerson)

	go dumpingStatusLoop()
	
	var wg sync.WaitGroup
	wg.Add(1)
	go syncLoop(&wg)

	crawlEntriesLoop()
	
	// dump docDB
	if err := gcse.DBOutSegments.ClearUndones(); err != nil {
		log.Printf("DBOutSegments.ClearUndones failed: %v", err)
	}

	if err := dumpDB(); err != nil {
		log.Printf("dumpDB failed: %v", err)
	}
	
	wg.Wait()
	log.Println("crawler stopped...")
}

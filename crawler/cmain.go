/*
	GCSE Crawler background program.
*/
package main

import (
	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/go-villa"
	"log"
	"runtime"
	"time"
)

const (
	fnDocDB     = "docdb"
	fnCrawlerDB = "crawler"
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

func syncLoop() {
	for {
		time.Sleep(10 * time.Minute)
		syncDatabases()
	}
}

func dumpingStatusLoop() {
	for {
		gcse.DumpMemStats()
		time.Sleep(10 * time.Minute)
	}
}

func main() {
	docDB = gcse.NewMemDB(DocDBPath, gcse.KindDocDB)

	cPackageDB = gcse.NewMemDB(CrawlerDBPath, kindPackage)
	cPersonDB = gcse.NewMemDB(CrawlerDBPath, kindPerson)

	go importingLoop()
	go dumpingLoop()

	go syncLoop()
	go dumpingStatusLoop()

	crawlEnetiresLoop()
}

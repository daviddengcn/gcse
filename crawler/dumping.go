package main

import (
	"github.com/daviddengcn/gcse"
	"github.com/howeyc/fsnotify"
	"log"
	"time"
)

var (
	lastDumpTime time.Time
)

func hasDonesDBOut() bool {
	dones, err := gcse.DBOutSegments.ListDones()
	if err != nil {
		log.Printf("DBOutSegments.ListDones failed: %v", err)
		return false
	}
	
	return len(dones) > 0
}

func needDump() bool {
	return docDB.LastModified().After(lastDumpTime)
}

func dumpDB() error {
	segm, err := gcse.DBOutSegments.GenMaxSegment()
	if err != nil {
		return err
	}
	log.Printf("Dumping docDB to %v ...", segm)
	if err := docDB.Export(segm.Join(""), gcse.KindDocDB); err != nil {
		return err
	}
	
	if err := segm.Done(); err != nil {
		return err
	}
	
	lastDumpTime = time.Now()
	return nil
}

func dumpingLoop() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("Creating watcher failed: %v", err)
		watcher = nil
	}
	watcher.Watch(gcse.DBOutPath.S())
	for {
		// clear watcher events (most generated for dumping docDB)
		gcse.ClearWatcherEvents(watcher)
		// wait for indexer to consume
		for hasDonesDBOut() {
			gcse.WaitForWatcherEvents(watcher)
		}

		// wait for data change		
		for !needDump() {
			time.Sleep(1 * time.Minute)
		} 
		
		// dump docDB
		if err := gcse.DBOutSegments.ClearUndones(); err != nil {
			log.Printf("DBOutSegments.ClearUndones failed: %v", err)
		}
		
		if err := dumpDB(); err != nil {
			log.Printf("dumpDB failed: %v", err)
			time.Sleep(1 * time.Minute)
		}
	}
}


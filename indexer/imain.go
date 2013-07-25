package main

import (
	"github.com/daviddengcn/gcse"
	"log"
	"time"
)

func main() {
	log.Println("indexer started...")
	
	if err := gcse.IndexSegments.ClearUndones(); err != nil {
		log.Printf("ClearUndones failed: %v", err)
	}

	dbSegm, err := gcse.DBOutSegments.FindMaxDone()
	if err == nil && dbSegm != nil {
		if err := clearOutdatedIndex(); err != nil {
			log.Printf("clearOutdatedIndex failed: %v", err)
		}
		doIndex(dbSegm)
	}
	
	// wait for a minute anyway in case the outer script does not sleep
	log.Println("Sleep for 1 min...")
	time.Sleep(1 * time.Minute)
	
	log.Println("indexer exits...")
}

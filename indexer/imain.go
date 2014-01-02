package main

import (
	"github.com/daviddengcn/gcse"
	"log"
//	"time"
)

func main() {
	log.Println("indexer started...")

	if err := gcse.IndexSegments.ClearUndones(); err != nil {
		log.Printf("ClearUndones failed: %v", err)
	}

	doIndex()

	// wait for a minute anyway in case the outer script does not sleep
//	log.Println("Sleep for 1 min...")
//	time.Sleep(1 * time.Minute)

	log.Println("indexer exits...")
}

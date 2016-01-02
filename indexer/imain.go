package main

import (
	"github.com/daviddengcn/gcse"
	"log"
	//	"time"
)

func main() {
	runtime.GOMAXPROCS(1)
	log.Println("indexer started...")

	if err := gcse.IndexSegments.ClearUndones(); err != nil {
		log.Printf("ClearUndones failed: %v", err)
	}

	if err := clearOutdatedIndex(); err != nil {
		log.Printf("clearOutdatedIndex failed: %v", err)
	}
	doIndex()

	log.Println("indexer exits...")
}

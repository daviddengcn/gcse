package main

import (
	"log"
	"strings"

	"github.com/golangplus/fmt"

	"github.com/daviddengcn/gcse"
)

func main() {
	// Load CrawlerDB
	cDB := gcse.LoadCrawlerDB()
	db := cDB.PackageDB
	var toDelete []string
	if err := db.Iterate(func(id string, val interface{}) error {
		parts := strings.Split(id, "/")
		if len(parts) < 6 || len(parts)%2 != 0 {
			return nil
		}
		l := (len(parts) - 4) / 2
		a := parts[3 : 3+l]
		b := parts[3+l : 3+l+l]
		for i := range a {
			if a[i] != b[i] {
				return nil
			}
		}
		toDelete = append(toDelete, id)
		return nil
	}); err != nil {
		log.Fatalf("Iterate failed: %v", err)
	}
	fmtp.Printfln("Total: %d", len(toDelete))
	for _, id := range toDelete {
		db.Delete(id)
	}
	log.Printf("Synchronizing databases to disk...")
	if err := cDB.Sync(); err != nil {
		log.Fatalf("cdb.Sync() failed: %v", err)
	}
}

package main

import (
	"fmt"
	"log"
	"time"
	
	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/sophie"
)

var (
	cPackageDB *gcse.MemDB
	cPersonDB  *gcse.MemDB
)

func generateCrawlEntries(db *gcse.MemDB, hostFromID func(id string) string,
	  out sophie.KVDirOutput) error {
	now := time.Now()
	groups := make(map[string]sophie.CollectCloser)
	count := 0
	if err := db.Iterate(func(id string, val interface{}) error {
		ent, ok := val.(gcse.CrawlingEntry)
		if !ok {
			log.Printf("Wrong entry: %+v", ent)
			return nil
		}

		if ent.ScheduleTime.After(now) {
			return nil
		}

		host := hostFromID(id)
		c, ok := groups[host]
		if !ok {
			index := len(groups)
			var err error
			c, err = out.Collector(index)
			if err != nil {
				return err
			}
			groups[host] = c
		}
		
		count ++
		return c.Collect(sophie.RawString(id), &ent)
	}); err != nil {
		return err
	}
	
	for _, c := range groups {
		c.Close()
	}
	
	log.Printf("%d entries to crawl for folder %v", count, out.Path)
	return nil
}

func main() {
	log.Println("Running tocrawl tool, to generate crawling list")
	cPackageDB = gcse.NewMemDB(gcse.CrawlerDBPath, gcse.KindPackage)
	cPersonDB = gcse.NewMemDB(gcse.CrawlerDBPath, gcse.KindPerson)
	
	fmt.Printf("Package DB: %d entries\n", cPackageDB.Count())
	fmt.Printf("Person DB: %d entries\n", cPersonDB.Count())
	
	pathToCrawl := gcse.DataRoot.Join(gcse.FnToCrawl)
	
	kvPackage := sophie.KVDirOutput {
		Fs: sophie.LocalFS,
		Path: pathToCrawl.Join(gcse.FnPackage).S(),
	}
	kvPackage.Clean()
	if err := generateCrawlEntries(cPackageDB, gcse.HostOfPackage, kvPackage);
			err != nil {
		log.Fatalf("generateCrawlEntries %v failed: %v", kvPackage.Path, err)
	}
	
	kvPerson := sophie.KVDirOutput {
		Fs: sophie.LocalFS,
		Path: pathToCrawl.Join(gcse.FnPerson).S(),
	}
	kvPerson.Clean()
	if err := generateCrawlEntries(cPersonDB, func(id string) string {
		site, _ := gcse.ParsePersonId(id)
		return site
	}, kvPerson); err != nil {
		log.Fatalf("generateCrawlEntries %v failed: %v", kvPerson.Path, err)
	}
}

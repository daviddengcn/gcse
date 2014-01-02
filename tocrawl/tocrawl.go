package main

import (
	"fmt"
	"log"
	"runtime"
	"time"
	
	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/sophie"
)

var (
	cPackageDB *gcse.MemDB
	cPersonDB  *gcse.MemDB
)

func loadPackageUpdateTimes(fpDocs sophie.FsPath) (map[string]time.Time, error) {
	dir := sophie.KVDirInput(fpDocs)
	cnt, err := dir.PartCount()
	if err != nil {
		return nil, err
	}
	
	pkgUTs := make(map[string]time.Time)
	
	var pkg sophie.RawString
	var info gcse.DocInfo
	for i := 0; i < cnt; i++ {
		it, err := dir.Iterator(i)
		if err != nil {
			return nil, err
		}
		for {
			if err := it.Next(&pkg, &info); err != nil {
				if err == sophie.EOF {
					break
				}
				return nil, err
			}
			
			pkgUTs[string(pkg)] = info.LastUpdated
		}
	}
	return pkgUTs, nil
}

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

func syncDatabases() {
	gcse.DumpMemStats()
	log.Printf("Synchronizing databases to disk...")
	if err := cPackageDB.Sync(); err != nil {
		log.Fatalf("cPackageDB.Sync failed: %v", err)
	}
	if err := cPersonDB.Sync(); err != nil {
		log.Fatalf("cPersonDB.Sync failed: %v", err)
	}
	gcse.DumpMemStats()
	runtime.GC()
	gcse.DumpMemStats()
}

func main() {
	log.Println("Running tocrawl tool, to generate crawling list")
	cPackageDB = gcse.NewMemDB(gcse.CrawlerDBPath, gcse.KindPackage)
	cPersonDB = gcse.NewMemDB(gcse.CrawlerDBPath, gcse.KindPerson)
	
	var err error
	pkgUTs, err = loadPackageUpdateTimes(
		sophie.LocalFsPath(gcse.DocsDBPath.S()))
	if err != nil {
		log.Fatalf("loadPackageUpdateTimes failed: %v", err)
	}
	
	touchByGithubUpdates()
	syncDatabases()
	
	fmt.Printf("Package DB: %d entries\n", cPackageDB.Count())
	fmt.Printf("Person DB: %d entries\n", cPersonDB.Count())
	
	pathToCrawl := gcse.DataRoot.Join(gcse.FnToCrawl)
	
	kvPackage := sophie.KVDirOutput(sophie.LocalFsPath(
		pathToCrawl.Join(gcse.FnPackage).S()))
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

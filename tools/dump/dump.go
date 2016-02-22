package main

import (
	"fmt"
	"log"
	"os"

	"github.com/golangplus/fmt"

	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/gcse/configs"
	"github.com/daviddengcn/go-index"
	"github.com/daviddengcn/sophie"
	"github.com/daviddengcn/sophie/kv"
)

func help() {
	fmt.Fprintln(os.Stderr, `Usage: dump docs|index|crawler [keys...]`)
}

func dumpDocs(keys []string) {
	path := configs.DataRoot.Join(configs.FnDocs).S()
	kvDir := kv.DirInput(sophie.LocalFsPath(path))
	cnt, err := kvDir.PartCount()
	if err != nil {
		log.Fatalf("kvDir.PartCount() failed: %v", err)
	}

	parts := make(map[int]map[string]bool)
	for _, key := range keys {
		part := gcse.CalcPackagePartition(key, gcse.DOCS_PARTS)
		if parts[part] == nil {
			parts[part] = make(map[string]bool)
		}

		parts[part][key] = true
	}

	var key sophie.RawString
	var val gcse.DocInfo
	for part := 0; part < cnt; part++ {
		if len(keys) > 0 && parts[part] == nil {
			continue
		}

		it, err := kvDir.Iterator(part)
		if err != nil {
			log.Fatalf("kvDir.Collector(%d) failed: %v", part, err)
		}

		func() {
			defer it.Close()

			for {
				if err := it.Next(&key, &val); err != nil {
					if err == sophie.EOF {
						break
					}
					log.Fatalf("it.Next failed %v", err)
				}
				pkg := key.String()
				if len(keys) > 0 && !parts[part][pkg] {
					continue
				}
				fmtp.Printfln("%v -> %+v", key, val)
			}

			it.Close()
		}()
	}
}

func dumpIndex(keys []string) {
	segm, err := gcse.IndexSegments.FindMaxDone()
	if segm == nil || err != nil {
		log.Fatalf("gcse.IndexSegments.FindMaxDone() failed: %v", err)
	}

	db := &index.TokenSetSearcher{}
	f, err := segm.Join(gcse.IndexFn).Open()
	if err != nil {
		log.Fatalf("%v.Join(%s).Open() failed: %v", segm, gcse.IndexFn, err)
	}
	defer f.Close()

	if err := db.Load(f); err != nil {
		log.Fatalf("db.Open() failed: %v", err)
	}

	for _, key := range keys {
		db.Search(index.SingleFieldQuery(gcse.IndexPkgField, key),
			func(docID int32, data interface{}) error {
				info, _ := data.(gcse.HitInfo)
				fmtp.Printfln("%s:%s -> %+v", gcse.IndexPkgField, key, info)
				return nil
			})
		db.Search(index.SingleFieldQuery(gcse.IndexTextField, key),
			func(docID int32, data interface{}) error {
				info, _ := data.(gcse.HitInfo)
				fmtp.Printfln("%s:%s -> %+v", gcse.IndexTextField, key, info)
				return nil
			})
	}
}

func dumpCrawler(keys []string) {
	cDB := gcse.LoadCrawlerDB()
	if len(keys) == 0 {
		// Full dump
		log.Printf("Dumping PackageDB...")
		cDB.PackageDB.Iterate(func(k string, v interface{}) error {
			fmtp.Printfln("Package %v: %+v", k, v)
			return nil
		})
		return
	}
	for _, key := range keys {
		var ent gcse.CrawlingEntry
		if cDB.PackageDB.Get(key, &ent) {
			fmtp.Printfln("Package %v: %+v", key, ent)
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		help()
		return
	}

	switch os.Args[1] {
	case "docs":
		dumpDocs(os.Args[2:])
	case "index":
		dumpIndex(os.Args[2:])
	case "crawler":
		dumpCrawler(os.Args[2:])
	default:
		help()
	}
}

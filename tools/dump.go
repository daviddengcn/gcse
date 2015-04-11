package main

import(
	"fmt"
	"log"
	"os"
	
	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/go-index"
	"github.com/daviddengcn/sophie"
	"github.com/daviddengcn/sophie/kv"
	"github.com/golangplus/fmt"
)

func help() {
	fmt.Fprintln(os.Stderr, `Usage: dump docs|index`)
}

func dumpDocs(keys []string) {
	path := "data/docs"
	kvDir := kv.DirInput(sophie.LocalFsPath(path))
	cnt, err := kvDir.PartCount()
	if err != nil {
		log.Fatalf("kvDir.PartCount() failed: %v")
	}
	
	parts := make(map[int]map[string]bool)
	for _, key := range keys {
		part := gcse.CalcPackagePartition(key, gcse.DOCS_PARTS);
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
						break;
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

func main() {
	s := `qrt` + "\xEF\xBB\xBF"
	for i, c := range s {
		fmtp.Printfln("%d: %x", i, c)
	}

	
	if len(os.Args) < 2 {
		help()
		return
	}
	
	switch os.Args[1] {
	case "docs": dumpDocs(os.Args[2:])
	case "index": dumpIndex(os.Args[2:])
	}
}
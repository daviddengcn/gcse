package main

import (
	"fmt"
	"log"
	
	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/go-villa"
	"github.com/daviddengcn/sophie"
)

var (
	DocDBPath villa.Path
)

const (
	fnDocDB     = "packed-docdb"
	fnNewDocDB  = "docs"
)

func init() {
	DocDBPath = gcse.DataRoot.Join(fnDocDB)
}

func main() {
	fmt.Println("Data conversion tool")
	fpRoot := sophie.FsPath {
		Fs: sophie.LocalFS,
		Path: "./data",
	}
	/*
	 * Doc db
	 */
	if DocDBPath.Exists() {
		if DocDBPath.Join(gcse.KindDocDB+".gob").Exists() &&
			!gcse.DataRoot.Join(fnNewDocDB).Exists() {
			src := DocDBPath.Join(gcse.KindDocDB+".gob")
			dst := fpRoot.Join(fnNewDocDB)
			fmt.Println("Convert", src, "to", dst, "...")
			
			srcDB := gcse.PackedDocDB{gcse.NewMemDB(DocDBPath, gcse.KindDocDB)}
			if err := srcDB.Load(); err != nil {
				log.Fatalf("srcDB.Load: %v", err)
			}
			
			fpDocs := fpRoot.Join(fnNewDocDB)
			dstDB := sophie.KVDirOutput(fpDocs)
			c, err := dstDB.Collector(0)
			if err != nil {
				log.Fatalf("dstDB.Collector: %v", err)
			}
			
			count := 0
			if err := srcDB.Iterate(func(key string, val interface{}) error {
				k := sophie.RawString(key)
				v := val.(gcse.DocInfo)
				
				if count < 10 {
					fmt.Printf("  key: %+v, value: %+v\n", k, v)
				}
				
				count ++
				return c.Collect(k, &v)
			}); err != nil {
				fpDocs.Remove()
				log.Fatalf("srcDB.Iterate: %v", err)
			}
			c.Close()
			
			fmt.Printf("Conversion sucess, %d entries collected.\n", count)
		}
	}
}

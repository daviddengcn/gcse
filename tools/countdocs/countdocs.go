package main

import (
	"log"

	"github.com/golangplus/fmt"

	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/sophie"
	"github.com/daviddengcn/sophie/kv"
)

func main() {
	//	path := "data/docs"
	path := "data/docs-updated"
	kvDir := kv.DirInput(sophie.LocalFsPath(path))

	cnt, err := kvDir.PartCount()
	if err != nil {
		log.Fatalf("kvDir.PartCount failed: %v", err)
	}

	totalEntries := 0
	for i := 0; i < cnt; i++ {
		it, err := kvDir.Iterator(i)
		if err != nil {
			log.Fatalf("kvDir.Collector(%d) failed: %v", i, err)
		}

		var key sophie.RawString
		var val gcse.DocInfo
		for {
			if err := it.Next(&key, &val); err != nil {
				if err == sophie.EOF {
					break
				}
				log.Fatalf("it.Next failed %v", err)
			}
			totalEntries++
		}

		it.Close()
	}

	fmtp.Printfln("Total %d files, %d entries.", cnt, totalEntries)
}

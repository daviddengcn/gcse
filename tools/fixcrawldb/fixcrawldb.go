package main

import (
	"io"
	"log"
	"strings"

	"github.com/golangplus/errors"
	"github.com/golangplus/fmt"
	"github.com/golangplus/strings"

	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/gcse/configs"
	"github.com/daviddengcn/gcse/spider"
	"github.com/daviddengcn/sophie"
	"github.com/daviddengcn/sophie/kv"
)

func loadDocsPkgs(in kv.DirInput) (stringsp.Set, error) {
	var pkgs stringsp.Set
	cnt, err := in.PartCount()
	if err != nil {
		return nil, err
	}
	for part := 0; part < cnt; part++ {
		c, err := in.Iterator(part)
		if err != nil {
			return nil, err
		}
		for {
			var key sophie.RawString
			var val gcse.DocInfo
			if err := c.Next(&key, &val); err != nil {
				if errorsp.Cause(err) == io.EOF {
					break
				}
				return nil, err
			}
			pkgs.Add(string(key))
			// value is ignored
		}
	}
	return pkgs, nil
}

func main() {
	dryRun := false
	// Load CrawlerDB
	cDB := gcse.LoadCrawlerDB()
	fpDataRoot := sophie.FsPath{
		Fs:   sophie.LocalFS,
		Path: configs.DataRoot.S(),
	}
	pkgs, err := loadDocsPkgs(kv.DirInput(fpDataRoot.Join(configs.FnDocs)))
	if err != nil {
		log.Fatalf("loadDocsPkgs failed: %v", err)
	}
	db := cDB.PackageDB
	var toDelete []string
	if err := db.Iterate(func(id string, val interface{}) error {
		if pkgs.Contain(id) {
			// If the pacakge is already in docs, do not touch it.
			return nil
		}
		parts := strings.Split(id, "/")
		if len(parts) >= 4 {
			// Check last part.
			// github.com/user/repo/sub
			name := parts[len(parts)-1]
			if !spider.LikeGoSubFolder(name) {
				toDelete = append(toDelete, id)
				return nil
			}
		}
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
	if dryRun {
		return
	}
	for _, id := range toDelete {
		db.Delete(id)
	}
	log.Printf("Synchronizing databases to disk...")
	if err := cDB.Sync(); err != nil {
		log.Fatalf("cdb.Sync() failed: %v", err)
	}
}

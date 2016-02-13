/*
	GCSE Crawler background program.
*/
package main

import (
	"encoding/gob"
	"errors"
	"flag"
	"log"
	"runtime"
	"strings"
	"time"

	"github.com/golangplus/bytes"
	"github.com/golangplus/errors"
	"github.com/golangplus/fmt"
	"github.com/golangplus/strings"

	"github.com/daviddengcn/bolthelper"
	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/gcse/spider/github"
	"github.com/daviddengcn/gddo/doc"
	"github.com/daviddengcn/go-easybi"
	"github.com/daviddengcn/go-villa"
	"github.com/daviddengcn/sophie"
	"github.com/daviddengcn/sophie/kv"
)

var (
	AppStopTime time.Time
	cDB         *gcse.CrawlerDB
)

func init() {
	if gcse.CrawlerGithubClientID != "" {
		log.Printf("Github clientid: %s", gcse.CrawlerGithubClientID)
		log.Printf("Github clientsecret: %s", gcse.CrawlerGithubClientSecret)
		doc.SetGithubCredentials(gcse.CrawlerGithubClientID, gcse.CrawlerGithubClientSecret)
	}
	doc.SetUserAgent("Go-Search(http://go-search.org/)")
}

func syncDatabases() {
	gcse.DumpMemStats()
	log.Printf("Synchronizing databases to disk...")
	if err := cDB.Sync(); err != nil {
		log.Fatalf("cdb.Sync() failed: %v", err)
	}
	gcse.DumpMemStats()
	runtime.GC()
	gcse.DumpMemStats()
}

func loadAllDocsPkgs(in kv.DirInput) error {
	cnt, err := in.PartCount()
	if err != nil {
		return err
	}
	for part := 0; part < cnt; part++ {
		c, err := in.Iterator(part)
		if err != nil {
			return err
		}
		for {
			var key sophie.RawString
			var val gcse.DocInfo
			if err := c.Next(&key, &val); err != nil {
				if err == sophie.EOF {
					break
				}
				return err
			}
			allDocsPkgs.Add(string(key))
			// value is ignored
		}
	}
	return nil
}

type crawlerMapper struct {
}

// Mapper interface
func (crawlerMapper) NewKey() sophie.Sophier {
	return new(sophie.RawString)
}

// Mapper interface
func (crawlerMapper) NewVal() sophie.Sophier {
	return new(gcse.CrawlingEntry)
}

// Mapper interface
func (crawlerMapper) MapEnd(c []sophie.Collector) error {
	return nil
}

func cleanTempDir() {
	tmpFn := villa.Path("/tmp/gddo")
	if err := tmpFn.RemoveAll(); err != nil {
		log.Printf("Delete %v failed: %v", tmpFn, err)
	}
}

type boltFileCache struct {
	bh.DB
}

// Filecache folders:
// s/<path>             - signature of this path
// c/<signature>        - contents of a signagure
// p/<signature>/<path> - list of paths referencing this signature

var (
	cacheSignatureKey = []byte("s")
	cacheContentsKey  = []byte("c")
	cachePathsKey     = []byte("p")
)

func (bc boltFileCache) Get(signature string, contents interface{}) bool {
	found := false
	if err := bc.View(func(tx bh.Tx) error {
		return tx.Value([][]byte{cacheContentsKey, []byte(signature)}, func(v bytesp.Slice) error {
			found = true
			return errorsp.WithStacks(gob.NewDecoder(&v).Decode(contents))
		})
	}); err != nil {
		log.Printf("Reading from file cache DB for %v failed: %v", signature, err)
		bi.AddValue(bi.Sum, "crawler.filecache.get_error", 1)
		return false
	}
	if found {
		bi.AddValue(bi.Sum, "crawler.filecache.hit", 1)
	} else {
		bi.AddValue(bi.Sum, "crawler.filecache.missed", 1)
	}
	return found
}

func (bc boltFileCache) Set(signature string, contents interface{}) {
	if err := bc.Update(func(tx bh.Tx) error {
		var bs bytesp.Slice
		if err := gob.NewEncoder(&bs).Encode(contents); err != nil {
			return errorsp.WithStacks(err)
		}
		return tx.Put([][]byte{cacheContentsKey, []byte(signature)}, bs)
	}); err != nil {
		bi.AddValue(bi.Sum, "crawler.filecache.set_error", 1)
		log.Printf("Updating to file cache DB for %v failed: %v", signature, err)
	}
	bi.AddValue(bi.Sum, "crawler.filecache.sign_saved", 1)
}

var errStop = errors.New("stop")

func (bc boltFileCache) SetFolderSignatures(folder string, nameToSignature map[string]string) {
	if !strings.HasSuffix(folder, "/") {
		folder += "/"
	}
	log.Printf("nameoSignature: %v", nameToSignature)
	if err := bc.Update(func(tx bh.Tx) error {
		// sub path -> current signature
		toDelete := make(map[string]string)
		// sub path -> current signature
		toUpdate := make(map[string]string)
		var unchanged stringsp.Set

		if err := tx.Cursor([][]byte{cacheSignatureKey}, func(c bh.Cursor) error {
			for k, v := c.Seek([]byte(folder)); strings.HasPrefix(string(k), folder); k, v = c.Next() {
				sub := string(k[len(folder):])
				ps := strings.SplitN(sub, "/", 2)
				if len(ps) > 1 {
					// k is a file under a sub folder.
					if s, ok := nameToSignature[ps[0]]; !ok || s != "" {
						// if the sub folder no longer exist or is not a folder (i.e. is a
						// file with signature), delete the current file
						toDelete[sub] = string(v)
					}
				} else if len(ps) == 1 {
					newS := nameToSignature[sub]
					if newS == "" {
						// no longer a file
						toDelete[sub] = string(v)
					} else if newS != string(v) {
						bi.AddValue(bi.Sum, "crawler.filecache.file_changed", 1)
						// signature changed
						toUpdate[sub] = string(v)
					} else {
						unchanged.Add(sub)
					}
				}
			}
			return nil
		}); err != nil {
			return err
		}
		log.Printf("toDelete: %v", toDelete)
		log.Printf("toUpdate: %v", toUpdate)
		log.Printf("unchanged: %v", unchanged)
		// Add new files into toUpdate
		for name, signature := range nameToSignature {
			if signature == "" {
				// folders
				continue
			}
			if _, ok := toUpdate[name]; ok {
				continue
			}
			if unchanged.Contain(name) {
				continue
			}
			bi.AddValue(bi.Sum, "crawler.filecache.file_added", 1)
			toUpdate[name] = ""
		}
		deleteReferenceToSignatureFromPath := func(signature, path string) error {
			if err := tx.Delete([][]byte{cachePathsKey, []byte(signature), []byte(path)}); err != nil {
				return err
			}
			// Check whether the signature is still referenced by any path.
			hasKeys := false
			if err := tx.ForEach([][]byte{cachePathsKey, []byte(signature)}, func(bh.Bucket, bytesp.Slice, bytesp.Slice) error {
				hasKeys = true
				return errStop
			}); err != nil && err != errStop {
				return err
			}
			if !hasKeys {
				// all references to the signature have been deleted, delete the contents of the signature as well
				if err := tx.Delete([][]byte{cacheContentsKey, []byte(signature)}); err != nil {
					return err
				}
				bi.AddValue(bi.Sum, "crawler.filecache.sign_deleted", 1)
			}
			return nil
		}
		for sub, signature := range toDelete {
			path := folder + sub
			if err := deleteReferenceToSignatureFromPath(signature, path); err != nil {
				return nil
			}
			// Delete the signature item of the path
			if err := tx.Delete([][]byte{cacheSignatureKey, []byte(path)}); err != nil {
				return err
			}
		}
		for sub, oldS := range toUpdate {
			path := folder + sub
			if oldS != "" {
				if err := deleteReferenceToSignatureFromPath(oldS, path); err != nil {
					return nil
				}
			}
			// Update the signature item of the path
			newS := nameToSignature[sub]
			if err := tx.Put([][]byte{cacheSignatureKey, []byte(path)}, []byte(newS)); err != nil {
				return err
			}
			// Add reference to new signature from path
			if err := tx.Put([][]byte{cachePathsKey, []byte(newS), []byte(path)}, []byte(newS)); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		log.Printf("SetFolderSignatures folder %v failed: %v", folder, err)
		bi.AddValue(bi.Sum, "crawler.filecache.sign_error", 1)
	}
}

func main() {
	runtime.GOMAXPROCS(2)

	log.Printf("Using personal: %v", gcse.CrawlerGithubPersonal)
	gcse.GithubSpider = github.NewSpiderWithToken(gcse.CrawlerGithubPersonal)

	if db, err := bh.Open(gcse.DataRoot.Join("filecache.bolt").S(), 0644, nil); err == nil {
		log.Print("Using file cache!")
		gcse.GithubSpider.FileCache = boltFileCache{db}
	} else {
		log.Printf("Open file cache failed: %v", err)
	}

	cleanTempDir()
	defer cleanTempDir()

	defer func() {
		bi.Flush()
		bi.Process()
	}()

	singlePackage := flag.String("pkg", "", "Crawling a single package")
	singleETag := flag.String("etag", "", "ETag for the single package crawling")
	singlePerson := flag.String("person", "", "Crawling a single person")

	flag.Parse()

	httpClient := gcse.GenHttpClient("")

	if *singlePerson != "" {
		log.Printf("Crawling single person %s ...", *singlePerson)
		p, err := gcse.CrawlPerson(httpClient, *singlePerson)
		if err != nil {
			fmtp.Printfln("Crawling person %s failed: %v", *singlePerson, err)
		} else {
			fmtp.Printfln("Person %s: %+v", *singlePerson, p)
		}
	}
	if *singlePackage != "" {
		log.Printf("Crawling single package %s ...", *singlePackage)
		p, err := gcse.CrawlPackage(httpClient, *singlePackage, *singleETag)
		if err != nil {
			fmtp.Printfln("Crawling package %s failed: %v", *singlePackage, err)
		} else {
			fmtp.Printfln("Package %s: %+v", *singlePackage, p)
		}
	}
	if *singlePackage != "" || *singlePerson != "" {
		return
	}

	log.Println("crawler started...")

	// Load CrawlerDB
	cDB = gcse.LoadCrawlerDB()

	fpDataRoot := sophie.FsPath{
		Fs:   sophie.LocalFS,
		Path: gcse.DataRoot.S(),
	}

	fpDocs := fpDataRoot.Join(gcse.FnDocs)
	if err := loadAllDocsPkgs(kv.DirInput(fpDocs)); err != nil {
		log.Fatalf("loadAllDocsPkgs: %v", err)
	}
	log.Printf("%d docs loaded!", len(allDocsPkgs))

	AppStopTime = time.Now().Add(gcse.CrawlerDuePerRun)

	//pathToCrawl := gcse.DataRoot.Join(gcse.FnToCrawl)
	fpCrawler := fpDataRoot.Join(gcse.FnCrawlerDB)
	fpToCrawl := fpDataRoot.Join(gcse.FnToCrawl)

	fpNewDocs := fpCrawler.Join(gcse.FnNewDocs)
	fpNewDocs.Remove()

	pkgEnd := make(chan error, 1)
	go crawlPackages(httpClient, fpToCrawl.Join(gcse.FnPackage), fpNewDocs, pkgEnd)

	psnEnd := make(chan error, 1)
	go crawlPersons(httpClient, fpToCrawl.Join(gcse.FnPerson), psnEnd)

	errPkg, errPsn := <-pkgEnd, <-psnEnd
	if errPkg != nil || errPsn != nil {
		log.Fatalf("Some job may failed, package: %v, person: %v", errPkg, errPsn)
	}
	if err := processImports(); err != nil {
		log.Printf("processImports failed: %v", err)
	}
	syncDatabases()
	log.Println("crawler stopped...")
}

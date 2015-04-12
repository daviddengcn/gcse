/*
	GCSE Crawler background program.
*/
package main

import (
	"flag"
	"log"
	"runtime"
	"time"

	"github.com/golangplus/fmt"
	
	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/gddo/doc"
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
			allDocsPkgs.Put(string(key))
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

func main() {
	singlePackge := ""
	singleETag := ""
	flag.StringVar(&singlePackge, "pkg", singlePackge, "Crawling single package")
	flag.StringVar(&singleETag, "etag", singleETag, "ETag for single package crawling")

	flag.Parse()

	httpClient := gcse.GenHttpClient("")

	if singlePackge != "" {
		log.Printf("Crawling single package %s ...", singlePackge)
		p, err := gcse.CrawlPackage(httpClient, singlePackge, singleETag)
		if err != nil {
			fmtp.Printfln("Crawling package %s failured: %v", singlePackge, err)
		} else {
			fmtp.Printfln("Package %s: %+v", singlePackge, p)
		}
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
	go crawlPackages(httpClient, fpToCrawl.Join(gcse.FnPackage), fpNewDocs,
		pkgEnd)

	psnEnd := make(chan error, 1)
	go crawlPersons(httpClient, fpToCrawl.Join(gcse.FnPerson), psnEnd)

	errPkg, errPsn := <-pkgEnd, <-psnEnd
	if errPkg != nil || errPsn != nil {
		log.Fatalf("Some job may failed, package: %v, person: %v",
			errPkg, errPsn)
	}

	if err := processImports(); err != nil {
		log.Printf("processImports failed: %v", err)
	}

	syncDatabases()
	log.Println("crawler stopped...")
}

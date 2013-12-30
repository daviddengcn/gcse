/*
	GCSE Crawler background program.
*/
package main

import (
	"log"
	"net/http"
	"runtime"
	"time"
	
	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/gddo/doc"
	"github.com/daviddengcn/go-villa"
	"github.com/daviddengcn/sophie"
)

const (
	fnCrawlerDB = "crawler"
	fnNewCrawled = "newcrawled"
)

var (
	CrawlerDBPath villa.Path
	AppStopTime   time.Time
	cNewDoc sophie.CollectCloser
)

func init() {
	CrawlerDBPath = gcse.DataRoot.Join(fnCrawlerDB)
}

func syncDatabases() {
	gcse.DumpMemStats()
	log.Printf("Synchronizing databases to disk...")
	if err := cPackageDB.Sync(); err != nil {
		log.Printf("cPackageDB.Sync failed: %v", err)
	}
	if err := cPersonDB.Sync(); err != nil {
		log.Printf("cPersonDB.Sync failed: %v", err)
	}
	gcse.DumpMemStats()
	runtime.GC()
	gcse.DumpMemStats()
}

func dumpingStatusLoop() {
	for time.Now().Before(AppStopTime) {
		gcse.DumpMemStats()
		time.Sleep(10 * time.Minute)
	}
}

type PackageCrawler struct {
	part int
	sophie.EmptyOnlyMapper
	failCount int
	httpClient *http.Client
}

// OnlyMapper.NewKey
func (*PackageCrawler) NewKey() sophie.Sophier {
	return new(sophie.RawString)
}

// OnlyMapper.NewVal
func (*PackageCrawler) NewVal() sophie.Sophier {
	return new(gcse.CrawlingEntry)
}

func packageToDoc(p *gcse.Package) gcse.DocInfo {
	// copy Package as a DocInfo
	d := gcse.DocInfo{
		Package:     p.Package,
		Name:        p.Name,
		Synopsis:    p.Synopsis,
		Description: p.Doc,
		LastUpdated: time.Now(),
		Author:      gcse.AuthorOfPackage(p.Package),
		ProjectURL:  p.ProjectURL,
		StarCount:   p.StarCount,
		ReadmeFn:    p.ReadmeFn,
		ReadmeData:  p.ReadmeData,
		Exported:    p.Exported,
	}

	d.Imports = nil
	for _, imp := range p.Imports {
		if doc.IsValidRemotePath(imp) {
			d.Imports = append(d.Imports, imp)
		}
	}

	return d
}

// OnlyMapper.Map
func (pc *PackageCrawler) Map(key, val sophie.SophieWriter,
		c []sophie.Collector) error {
	if time.Now().After(AppStopTime) {
		log.Printf("Timeout(key = %v), part %d returns EOM", key, pc.part)
		return sophie.EOM
	}
						
	pkg := *key.(*sophie.RawString)
	ent := val.(*gcse.CrawlingEntry)
	log.Printf("Crawling %v\n", pkg)
	p, err := gcse.CrawlPackage(pc.httpClient, pkg.String(), ent.Etag)
	_ = p
	if err != nil && err != gcse.ErrPackageNotModifed {
		log.Printf("Crawling pkg %s failed: %v", pkg, err)
/*	
		if gcse.IsBadPackage(err) {
			// a wrong path
			deletePackage(ent.ID)
			log.Printf("Remove wrong package %s", ent.ID)
		} else {
			failCount++
	
			schedulePackage(ent.ID, time.Now().Add(
				12*time.Hour), ent.Etag)
	
			if failCount >= 10 {
				durToSleep := 10 * time.Minute
				if time.Now().Add(durToSleep).After(AppStopTime) {
					break
				}
	
				log.Printf("Last ten crawling %s packages failed, sleep for a while...",
					host)
				time.Sleep(durToSleep)
				failCount = 0
			}
		}
*/
		return nil
	}
	
	pc.failCount = 0
	if err == gcse.ErrPackageNotModifed {
		log.Printf("Package %s unchanged!", pkg)
//		schedulePackageNextCrawl(ent.ID, ent.Etag)
		return nil
	}
	
	log.Printf("Crawled package %s success!", pkg)
	
	nda := gcse.NewDocAction {
		Action: gcse.NDA_UPDATE,
		DocInfo: packageToDoc(p),
	}
	c[0].Collect(pkg, &nda)
	log.Printf("Package %s saved!", pkg)

	return nil
}

type PackageCrawlerFactory struct {
	httpClient *http.Client
}

func (pcf PackageCrawlerFactory) NewMapper(part int) sophie.OnlyMapper {
	return &PackageCrawler{part: part, httpClient: pcf.httpClient}
}

func main() {
	cPackageDB = gcse.NewMemDB(CrawlerDBPath, kindPackage)
	
	log.Println("crawler started...")

	AppStopTime = time.Now().Add(10 * time.Second)
	
	//pathToCrawl := gcse.DataRoot.Join(gcse.FnToCrawl)
	fpDataRoot := sophie.FsPath {
		Fs: sophie.LocalFS,
		Path: gcse.DataRoot.S(),
	}
	fpToCrawl := fpDataRoot.Join(gcse.FnToCrawl)
	fpCrawler := fpDataRoot.Join(gcse.FnCrawlerDB)
	outNewDocs := sophie.KVDirOutput(fpCrawler.Join(gcse.FnNewDocs))
	outNewDocs.Clean()
	src := sophie.KVDirInput(fpToCrawl.Join(gcse.FnPackage))
	job := sophie.MapOnlyJob{
		MapFactory: PackageCrawlerFactory{
			httpClient: gcse.GenHttpClient(""),
		},
		
		Source: src,
		Dest: []sophie.Output{
			outNewDocs,
		},
	}
	if err := job.Run(); err != nil {
		log.Fatalf("job.Run failed: %v", err)
	}
/*
	cPersonDB = gcse.NewMemDB(CrawlerDBPath, kindPerson)
	
	kvDirNewDoc := sophie.FsPath {
		Fs: sophie.LocalFS,
		Path: gcse.DataRoot.Join(fnNewCrawled).S(),
	}
	var err error
	kvfNewDoc, err = kvDirNewDoc.Collector(0)
	if err != nil {
		log.Fatalf("kvDirNewDoc.Collector(0) failed: %v", err)
	}

	go dumpingStatusLoop()

	crawlEntriesLoop()

	// dump docDB
	if err := gcse.DBOutSegments.ClearUndones(); err != nil {
		log.Printf("DBOutSegments.ClearUndones failed: %v", err)
	}

//	if err := dumpDB(); err != nil {
//		log.Printf("dumpDB failed: %v", err)
//	}
*/
	log.Println("crawler stopped...")
}

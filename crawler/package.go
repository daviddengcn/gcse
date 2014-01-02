package main

import (
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
	
	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/gddo/doc"
	"github.com/daviddengcn/go-villa"
	"github.com/daviddengcn/sophie"
)

const (
	DefaultPackageAge = 10 * 24 * time.Hour
)

var (
	cPackageDB *gcse.MemDB
	allDocsPkgs villa.StrSet
)

func schedulePackage(pkg string, sTime time.Time, etag string) error {
	ent := gcse.CrawlingEntry{
		ScheduleTime: sTime,
		Version:      gcse.CrawlerVersion,
		Etag:         etag,
	}

	cPackageDB.Put(pkg, ent)

	log.Printf("Schedule package %s to %v", pkg, sTime)
	return nil
}

// schedule a package for next crawling cycle, commonly after a successful update.
func schedulePackageNextCrawl(pkg string, etag string) {
	schedulePackage(pkg, time.Now().Add(time.Duration(
		float64(DefaultPackageAge)*(1+(rand.Float64()-0.5)*0.2))), etag)

}

func appendPackage(pkg string) {
	pkg = strings.TrimFunc(strings.TrimSpace(pkg), func(r rune) bool {
		return r > rune(128)
	})
	if !doc.IsValidRemotePath(pkg) {
		//log.Printf("  [appendPackage] Not a valid remote path: %s", pkg)
		return
	}

	var ent gcse.CrawlingEntry
	exists := cPackageDB.Get(pkg, &ent)
	if exists {
		if allDocsPkgs.In(pkg) {
			return
		}
	}

	// if the package doesn't exist in docDB, Etag is discarded
	schedulePackage(pkg, time.Now(), "")
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

	// append new authors
	if strings.HasPrefix(d.Package, "github.com/") {
		appendPerson("github.com", d.Author)
	} else if strings.HasPrefix(d.Package, "bitbucket.org/") {
		appendPerson("bitbucket.org", d.Author)
	}

	for _, imp := range d.Imports {
		appendPackage(imp)
	}
	log.Printf("[pushPackage] References: %v", p.References)
	for _, ref := range p.References {
		appendPackage(ref)
	}

	schedulePackageNextCrawl(d.Package, p.Etag)
	
	return d
}


type PackageCrawler struct {
	crawlerMapper
	
	part int
	failCount int
	httpClient *http.Client
}

// OnlyMapper.Map
func (pc *PackageCrawler) Map(key, val sophie.SophieWriter,
		c []sophie.Collector) error {
	if time.Now().After(AppStopTime) {
		log.Printf("Timeout(key = %v), PackageCrawler part %d returns EOM", key, pc.part)
		return sophie.EOM
	}
						
	pkg := string(*key.(*sophie.RawString))
	ent := val.(*gcse.CrawlingEntry)
	log.Printf("Crawling package %v\n", pkg)
	
	p, err := gcse.CrawlPackage(pc.httpClient, pkg, ent.Etag)
	_ = p
	if err != nil && err != gcse.ErrPackageNotModifed {
		log.Printf("Crawling pkg %s failed: %v", pkg, err)
		if gcse.IsBadPackage(err) {
			// a wrong path
			nda := gcse.NewDocAction {
				Action: gcse.NDA_DEL,
			}
			c[0].Collect(sophie.RawString(pkg), &nda)
			cPackageDB.Delete(pkg)
			log.Printf("Remove wrong package %s", pkg)
		} else {
			pc.failCount++
	
			schedulePackage(pkg, time.Now().Add(12*time.Hour), ent.Etag)
	
			if pc.failCount >= 10 {
				durToSleep := 10 * time.Minute
				if time.Now().Add(durToSleep).After(AppStopTime) {
					log.Printf("Timeout(key = %v), part %d returns EOM", key, pc.part)
					return sophie.EOM
				}
	
				log.Printf("Last ten crawling packages failed, sleep for a while...(current: %s)",
					pkg)
				time.Sleep(durToSleep)
				pc.failCount = 0
			}
		}
		return nil
	}
	
	pc.failCount = 0
	if err == gcse.ErrPackageNotModifed {
		// TODO crawling stars for unchanged project
		log.Printf("Package %s unchanged!", pkg)
		schedulePackageNextCrawl(pkg, ent.Etag)
		return nil
	}
	
	log.Printf("Crawled package %s success!", pkg)
	
	nda := gcse.NewDocAction {
		Action: gcse.NDA_UPDATE,
		DocInfo: packageToDoc(p),
	}
	c[0].Collect(sophie.RawString(pkg), &nda)
	log.Printf("Package %s saved!", pkg)

	return nil
}

// crawl packages, send error back to end
func crawlPackages(httpClient *http.Client, fpToCrawlPkg, fpOutNewDocs sophie.FsPath, end chan error) {
	end <- func() error {
		outNewDocs := sophie.KVDirOutput(fpOutNewDocs)
		outNewDocs.Clean()
		job := sophie.MapOnlyJob{
			Source: []sophie.Input{
				sophie.KVDirInput(fpToCrawlPkg),
			},
			
			MapFactory: sophie.OnlyMapperFactoryFunc(
			func(src, part int) sophie.OnlyMapper {
				return &PackageCrawler{
					part: part,
					httpClient: httpClient,
				}
			}),
			
			Dest: []sophie.Output{
				outNewDocs,
			},
		}
		
		if err := job.Run(); err != nil {
			log.Printf("crawlPackages: job.Run failed: %v", err)
			return err
		}
		return nil
	}()
}

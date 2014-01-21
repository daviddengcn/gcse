package main

import (
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/gddo/doc"
	"github.com/daviddengcn/go-villa"
	"github.com/daviddengcn/sophie"
	"github.com/daviddengcn/sophie/mr"
	"github.com/daviddengcn/sophie/kv"
)

const (
	DefaultPackageAge = 10 * 24 * time.Hour
)

var (
	allDocsPkgs villa.StrSet
)

// Schedule a package for next crawling cycle, commonly after a successful
// update.
func schedulePackageNextCrawl(pkg string, etag string) {
	cDB.SchedulePackage(pkg, time.Now().Add(time.Duration(
		float64(DefaultPackageAge)*(1+(rand.Float64()-0.5)*0.2))), etag)

}

func appendPackage(pkg string) {
	cDB.AppendPackage(pkg, allDocsPkgs.In)
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
	d.TestImports = nil
	for _, imp := range p.TestImports {
		if doc.IsValidRemotePath(imp) {
			d.TestImports = append(d.TestImports, imp)
		}
	}

	// append new authors
	if strings.HasPrefix(d.Package, "github.com/") {
		cDB.AppendPerson("github.com", d.Author)
	} else if strings.HasPrefix(d.Package, "bitbucket.org/") {
		cDB.AppendPerson("bitbucket.org", d.Author)
	}

	for _, imp := range d.Imports {
		appendPackage(imp)
	}
	for _, imp := range d.TestImports {
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

	part       int
	failCount  int
	httpClient doc.HttpClient
}

// OnlyMapper.Map
func (pc *PackageCrawler) Map(key, val sophie.SophieWriter,
	c []sophie.Collector) error {
	if time.Now().After(AppStopTime) {
		log.Printf("Timeout(key = %v), PackageCrawler part %d returns EOM",
			key, pc.part)
		return mr.EOM
	}

	pkg := string(*key.(*sophie.RawString))
	ent := val.(*gcse.CrawlingEntry)
	if ent.Version < gcse.CrawlerVersion {
		// if gcse.CrawlerVersion is larger than Version, Etag is ignored.
		ent.Etag = ""
	}
	log.Printf("Crawling package %v\n", pkg)

	p, err := gcse.CrawlPackage(pc.httpClient, pkg, ent.Etag)
	_ = p
	if err != nil && err != gcse.ErrPackageNotModifed {
		log.Printf("Crawling pkg %s failed: %v", pkg, err)
		if gcse.IsBadPackage(err) {
			// a wrong path
			nda := gcse.NewDocAction{
				Action: gcse.NDA_DEL,
			}
			c[0].Collect(sophie.RawString(pkg), &nda)
			cDB.PackageDB.Delete(pkg)
			log.Printf("Remove wrong package %s", pkg)
		} else {
			pc.failCount++

			cDB.SchedulePackage(pkg, time.Now().Add(12*time.Hour), ent.Etag)

			if pc.failCount >= 10 || strings.Contains(err.Error(), "403") {
				durToSleep := 10 * time.Minute
				if time.Now().Add(durToSleep).After(AppStopTime) {
					log.Printf("Timeout(key = %v), part %d returns EOM",
						key, pc.part)
					return mr.EOM
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

	nda := gcse.NewDocAction{
		Action:  gcse.NDA_UPDATE,
		DocInfo: packageToDoc(p),
	}
	c[0].Collect(sophie.RawString(pkg), &nda)
	log.Printf("Package %s saved!", pkg)

	time.Sleep(10 * time.Second)

	return nil
}

// crawl packages, send error back to end
func crawlPackages(httpClient doc.HttpClient, fpToCrawlPkg,
	fpOutNewDocs sophie.FsPath, end chan error) {
	end <- func() error {
		outNewDocs := kv.DirOutput(fpOutNewDocs)
		outNewDocs.Clean()
		job := mr.MapOnlyJob{
			Source: []mr.Input{
				kv.DirInput(fpToCrawlPkg),
			},

			NewMapperF: func(src, part int) mr.OnlyMapper {
				return &PackageCrawler{
					part:       part,
					httpClient: httpClient,
				}
			},

			Dest: []mr.Output{
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

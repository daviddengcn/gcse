package main

import (
	"errors"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/golangplus/strings"
	"github.com/golangplus/time"

	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/gcse/configs"
	"github.com/daviddengcn/gddo/doc"
	"github.com/daviddengcn/go-easybi"
	"github.com/daviddengcn/sophie"
	"github.com/daviddengcn/sophie/kv"
	"github.com/daviddengcn/sophie/mr"
)

const (
	DefaultPackageAge = 10 * timep.Day
)

var (
	allDocsPkgs stringsp.Set
)

// Schedule a package for next crawling cycle, commonly after a successful
// update.
func schedulePackageNextCrawl(pkg string, etag string) {
	cDB.SchedulePackage(pkg, time.Now().Add(time.Duration(
		float64(DefaultPackageAge)*(1+(rand.Float64()-0.5)*0.2))), etag)

}

func appendPackage(pkg string) {
	cDB.AppendPackage(pkg, allDocsPkgs.Contain)
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
func (pc *PackageCrawler) Map(key, val sophie.SophieWriter, c []sophie.Collector) error {
	if time.Now().After(AppStopTime) {
		log.Printf("[Part %d] Timeout(key = %v), PackageCrawler returns EOM",
			pc.part, key)
		return mr.EOM
	}
	pkg := string(*key.(*sophie.RawString))
	ent := val.(*gcse.CrawlingEntry)
	if ent.Version < gcse.CrawlerVersion {
		// if gcse.CrawlerVersion is larger than Version, Etag is ignored.
		ent.Etag = ""
	}
	log.Printf("[Part %d] Crawling package %v with etag %s\n", pc.part, pkg, ent.Etag)

	p, flds, err := gcse.CrawlPackage(pc.httpClient, pkg, ent.Etag)
	for _, fld := range flds {
		appendPackage(pkg + "/" + fld.Path)
	}
	if err != nil && err != gcse.ErrPackageNotModifed {
		log.Printf("[Part %d] Crawling pkg %s failed: %v", pc.part, pkg, err)
		if gcse.IsBadPackage(err) {
			bi.AddValue(bi.Sum, "crawler.package.wrong-package", 1)
			// a wrong path
			nda := gcse.NewDocAction{
				Action: gcse.NDA_DEL,
			}
			c[0].Collect(sophie.RawString(pkg), &nda)
			cDB.PackageDB.Delete(pkg)
			log.Printf("[Part %d] Remove wrong package %s", pc.part, pkg)
		} else {
			bi.Inc("crawler.package.failed")
			if strings.HasPrefix(pkg, "github.com/") {
				bi.Inc("crawler.package.failed.github")
			}
			pc.failCount++

			cDB.SchedulePackage(pkg, time.Now().Add(12*time.Hour), ent.Etag)

			if pc.failCount >= 10 || strings.Contains(err.Error(), "403") {
				durToSleep := 10 * time.Minute
				if time.Now().Add(durToSleep).After(AppStopTime) {
					log.Printf("[Part %d] Timeout(key = %v), PackageCrawler returns EOM",
						pc.part, key)
					return mr.EOM
				}

				log.Printf("[Part %d] Last ten crawling packages failed, sleep for a while...(current: %s)",
					pc.part, pkg)
				time.Sleep(durToSleep)
				pc.failCount = 0
			}
		}
		return nil
	}
	pc.failCount = 0
	if err == gcse.ErrPackageNotModifed {
		// TODO crawling stars for unchanged project
		log.Printf("[Part %d] Package %s unchanged!", pc.part, pkg)
		schedulePackageNextCrawl(pkg, ent.Etag)
		bi.AddValue(bi.Sum, "crawler.package.not-modified", 1)
		return nil
	}
	bi.AddValue(bi.Sum, "crawler.package.success", 1)
	if strings.HasPrefix(pkg, "github.com/") {
		bi.AddValue(bi.Sum, "crawler.package.success.github", 1)
	}
	log.Printf("[Part %d] Crawled package %s success!", pc.part, pkg)

	nda := gcse.NewDocAction{
		Action:  gcse.NDA_UPDATE,
		DocInfo: packageToDoc(p),
	}
	c[0].Collect(sophie.RawString(pkg), &nda)
	log.Printf("[Part %d] Package %s saved!", pc.part, pkg)

	if !strings.HasPrefix(pkg, "github.com/") {
		// github.com throttling is done within the GithubSpider.
		time.Sleep(10 * time.Second)
	}
	return nil
}

// crawl packages, send error back to end
func crawlPackages(httpClient doc.HttpClient, fpToCrawlPkg,
	fpOutNewDocs sophie.FsPath, end chan error) {

	time.AfterFunc(configs.CrawlerDuePerRun+time.Minute*10, func() {
		end <- errors.New("Crawling packages timeout!")
	})
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

package main

import (
	"encoding/gob"
	"errors"
	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/gddo/doc"
	"github.com/daviddengcn/go-code-crawl"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"
)

var (
	cPackageDB *gcse.MemDB
	cPersonDB  *gcse.MemDB
)

const (
	DefaultPackageAge = 10 * 24 * time.Hour
	DefaultPersonAge  = 10 * 24 * time.Hour

	kindPackage = "package"
	kindPerson  = "person"
)

type CrawlingEntry struct {
	ScheduleTime time.Time
}

func init() {
	gob.Register(CrawlingEntry{})

	doc.SetGithubCredentials("94446b37edb575accd8b",
		"15f55815f0515a3f6ad057aaffa9ea83dceb220b")
	doc.SetUserAgent("Go-Code-Search-Agent")
}

func schedulePackage(pkg string, sTime time.Time) error {
	var ent CrawlingEntry

	ent.ScheduleTime = sTime
	cPackageDB.Put(pkg, ent)

	log.Printf("Schedule package %s to %v", pkg, sTime)
	return nil
}

func appendPackage(pkg string) bool {
	pkg = strings.TrimSpace(pkg)
	if !doc.IsValidRemotePath(pkg) {
		// log.Printf("  [appendPackage] Not a valid remote path: %s", pkg)
		return false
	}

	var ent CrawlingEntry
	exists := cPackageDB.Get(pkg, &ent)
	if exists {
		var di gcse.DocInfo
		exists := docDB.Get(pkg, &di)
		if exists {
			// already scheduled
			log.Printf("  [appendPackage] Package %s was scheduled to %v", pkg, ent.ScheduleTime)
			return false
		}
	}

	return schedulePackage(pkg, time.Now()) == nil
}

func processImports() error {
	segments, err := gcse.ImportSegments.ListDones()
	if err != nil {
		return err
	}

	for _, s := range segments {
		log.Printf("Processing done segment %v ...", s)
		files, err := s.ListFiles()
		if err != nil {
			log.Printf("ListFiles failed: %v", err)
			continue
		}

		for _, fn := range files {
			var pkgs []string
			if err := gcse.ReadJsonFile(fn, &pkgs); err != nil {
				log.Printf("ReadJsonFile failed: %v", err)
				continue
			}
			log.Printf("Importing %d packages ...", len(pkgs))
			for _, pkg := range pkgs {
				pkg = strings.TrimSpace(pkg)
				appendPackage(pkg)
			}
		}

		if err := cPackageDB.Sync(); err != nil {
			log.Printf("crawlerDB.Sync failed: %v", err)
		}

		if err := s.Remove(); err != nil {
			log.Printf("s.Remove failed: %v", err)
		}
	}

	return nil
}

var errStop = errors.New("Stop")

func listCrawlEntries(db *gcse.MemDB, l int) (ids []string) {
	now := time.Now()
	if l > 0 {
		ids = make([]string, 0, l)
	}
	db.Iterate(func(id string, val interface{}) error {
		ent, ok := val.(CrawlingEntry)
		if !ok {
			return nil
		}

		if ent.ScheduleTime.After(now) {
			return nil
		}

		ids = append(ids, id)

		if l > 0 && len(ids) >= l {
			return errStop
		}
		return nil
	})

	return ids
}

func deletePackage(pkg string) {
	cPackageDB.Delete(pkg)
	docDB.Delete(pkg)
}

func schedulePerson(site, username string, sTime time.Time) error {
	id := gcc.IdOfPerson(site, username)
	/*
		err, _ := ddb.Get(id, &ent)
		if err != nil {
			log.Printf("  [scheduledPerson] crawler-person.Get(%s) failed: %v", id,
				err)
		}
	*/

	var ent CrawlingEntry
	ent.ScheduleTime = sTime

	cPersonDB.Put(id, ent)

	log.Printf("Schedule person %s to %v", id, sTime)
	return nil
}

func appendPerson(site, username string) bool {
	id := gcc.IdOfPerson(site, username)

	var ent CrawlingEntry
	exists := cPersonDB.Get(id, &ent)
	if exists {
		// already scheduled
		log.Printf("  [appendPerson] Person %s was scheduled to %v", id, ent.ScheduleTime)
		return false
	}

	return schedulePerson(site, username, time.Now()) == nil
}

func pushPackage(p *gcc.Package) (succ bool) {
	// copy Package as a DocInfo
	d := gcse.DocInfo{
		Name:        p.Name,
		Package:     p.ImportPath,
		Synopsis:    p.Synopsis,
		Description: p.Doc,
		LastUpdated: time.Now(),
		Author:      gcc.AuthorOfPackage(p.ImportPath),
		ProjectURL:  p.ProjectURL,
		StarCount:   p.StarCount,
		ReadmeFn:    p.ReadmeFn,
		ReadmeData:  p.ReadmeData,
	}

	d.Imports = nil
	for _, imp := range p.Imports {
		if doc.IsValidRemotePath(imp) {
			d.Imports = append(d.Imports, imp)
		}
	}

	if err := processDocument(&d); err != nil {
		log.Printf("processDocument %s failed: %v", d.Package, err)
		return false
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

	schedulePackage(d.Package, time.Now().Add(time.Duration(
		float64(DefaultPackageAge)*(1+(rand.Float64()-0.5)*0.2))))

	return true
}

func pushPerson(p *gcc.Person) (hasNewPkg bool) {
	for _, pkg := range p.Packages {
		if appendPackage(pkg) {
			hasNewPkg = true
		}
	}

	site, username := gcc.ParsePersonId(p.Id)

	schedulePerson(site, username, time.Now().Add(time.Duration(
		float64(DefaultPersonAge)*(1+(rand.Float64()-0.5)*0.2))))

	return
}

func CrawlEnetires() {
	httpClient := gcc.GenHttpClient("")

	for {
		didSomething := false
		var wg sync.WaitGroup

		pkgs := listCrawlEntries(cPackageDB, -1)
		if len(pkgs) > 0 {
			didSomething = true

			groups := gcc.GroupPackages(pkgs)
			log.Printf("Crawling %d groups, %d packages: %v", len(groups),
				len(pkgs), groups)

			wg.Add(len(groups))

			for _, pkgs := range groups {
				go func(pkgs []string) {
					failCount := 0
					for _, pkg := range pkgs {
						p, err := gcc.CrawlPackage(httpClient, pkg)
						if err != nil {
							log.Printf("Crawling pkg %s failed: %v", pkg, err)

							if gcc.IsBadPackage(err) {
								// a wrong path
								deletePackage(pkg)
								log.Printf("Remove wrong package %s", pkg)
							} else {
								failCount ++
							}
							continue
						} else {
							failCount = 0
						}

						log.Printf("Crawled package %s success!", pkg)

						pushPackage(p)
						log.Printf("Package %s saved!", pkg)
						
						if failCount >= 10 {
							log.Printf("Last ten crawling failed, sleep for a while...")
							time.Sleep(2 * time.Minute)
							failCount = 0
						}
					}

					wg.Done()
				}(pkgs)
			}
		}

		persons := listCrawlEntries(cPersonDB, -1)
		if len(persons) > 0 {
			didSomething = true

			groups := gcc.GroupPersons(persons)
			log.Printf("persons: %v, %d groups, %d persons", groups,
				len(groups), len(persons))

			wg.Add(len(groups))

			for _, ids := range groups {
				go func(ids []string) {
					failCount := 0
					for _, id := range ids {
						p, err := gcc.CrawlPerson(httpClient, id)
						if err != nil {
							failCount ++
							log.Printf("Crawling person %s failed: %v", id, err)
							continue
						}

						log.Printf("Crawled person %s success!", id)
						pushPerson(p)
						log.Printf("Push person %s success", id)
						
						if failCount >= 10 {
							log.Printf("Last ten crawling failed, sleep for a while...")
							time.Sleep(2 * time.Minute)
							failCount = 0
						}
					}

					wg.Done()
				}(ids)
			}
		}
		wg.Wait()

		syncDatabases()

		if !didSomething {
			log.Printf("Nothing to crawl sleep for a while...")
			time.Sleep(2 * time.Minute)
		}
	}
}

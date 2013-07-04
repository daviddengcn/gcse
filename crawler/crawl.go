package main

import (
	"encoding/gob"
	"encoding/json"
	"errors"
	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/gddo/doc"
	"github.com/daviddengcn/go-code-crawl"
	"net/http"
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

func touchPackage(pkg string) bool {
	pkg = strings.TrimSpace(pkg)
	if !doc.IsValidRemotePath(pkg) {
		// log.Printf("  [appendPackage] Not a valid remote path: %s", pkg)
		return false
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
				//touchPackage(pkg)
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

func listCrawlEntriesByHost(db *gcse.MemDB, hostFromID func(id string) string,
	maxHosts, numPerHost int) (groups map[string][]string) {
	now := time.Now()
	groups = make(map[string][]string)
	fullGroups := 0
	db.Iterate(func(pkg string, val interface{}) error {
		ent, ok := val.(CrawlingEntry)
		if !ok {
			return nil
		}

		if ent.ScheduleTime.After(now) {
			return nil
		}

		host := hostFromID(pkg)
		pkgs := groups[host]
		if maxHosts > 0 {
			// check host limit
			if len(pkgs) == 0 && len(groups) == maxHosts {
				// no quota for new group
				return nil
			}
		}
		if numPerHost > 0 {
			// check per host limit
			if len(pkgs) == numPerHost - 1 {
				// this group is about to be full, count it
				fullGroups ++
			} else if len(pkgs) == numPerHost {
				// no quota for this group
				return nil
			}
		}
		groups[host] = append(pkgs, pkg)
		
		if fullGroups == maxHosts {
			return errStop
		}
		return nil
	})

	return groups
}

func listPackagesByHost(maxHosts, numPerHost int) (groups map[string][]string) {
	return listCrawlEntriesByHost(cPackageDB, gcc.HostOfPackage, maxHosts, numPerHost)
}

func listPersonsByHost(maxHosts, numPerHost int) (groups map[string][]string) {
	return listCrawlEntriesByHost(cPersonDB, func(id string) string {
		site, _ := gcc.ParsePersonId(id)
		return site
	}, maxHosts, numPerHost)
}

func deletePackage(pkg string) {
	cPackageDB.Delete(pkg)
	docDB.Delete(pkg)
}

func schedulePerson(id string, sTime time.Time) error {
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

	return schedulePerson(id, time.Now()) == nil
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

	schedulePerson(p.Id, time.Now().Add(time.Duration(
		float64(DefaultPersonAge)*(1+(rand.Float64()-0.5)*0.2))))

	return
}

const (
	godocApiUrl = "http://api.godoc.org/packages"
	godocCrawlGap = 4 * time.Hour
)
var (
	godocLastCrawled time.Time
)

func processGodoc(httpClient *http.Client) bool {
	if time.Now().Before(godocLastCrawled.Add(godocCrawlGap)) {
		return false
	}
	
	resp, err := httpClient.Get(godocApiUrl)
	if err != nil {
		log.Printf("Get %s failed: %v", godocApiUrl, err)
		return false
	}
	if resp.StatusCode != 200 {
		log.Printf("StatusCode: %d", resp.StatusCode)
		return false
	}
	defer resp.Body.Close()
	
	godocLastCrawled = time.Now()

	var results map[string][]map[string]string
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&results)
	if err != nil {
		log.Printf("Parse results failed: %v", err)
		return false
	}

	for _, res := range results["results"] {
		pkg := res["path"]
		appendPackage(pkg)
	}
	
	return true
}

func CrawlEnetires() {
	httpClient := gcc.GenHttpClient("")

	for {
		didSomething := false
		var wg sync.WaitGroup

		pkgGroups := listPackagesByHost(5, 50)
		if len(pkgGroups) > 0 {
			didSomething = true

			log.Printf("Crawling packages of %d groups", len(pkgGroups))

			wg.Add(len(pkgGroups))

			for host, pkgs := range pkgGroups {
				go func(host string, pkgs []string) {
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
								
								schedulePackage(pkg, time.Now().Add(
									12 * time.Hour))
									
								if failCount >= 10 {
									log.Printf("Last ten crawling %s packages failed, sleep for a while...",
										host)
									time.Sleep(10 * time.Minute)
									failCount = 0
								}
							}
							continue
						} else {
							failCount = 0
						}

						log.Printf("Crawled package %s success!", pkg)

						pushPackage(p)
						log.Printf("Package %s saved!", pkg)
					}

					wg.Done()
				}(host, pkgs)
			}
		}

		personGroups := listPersonsByHost(5, 100)
		if len(personGroups) > 0 {
			didSomething = true

			log.Printf("Crawling persons of %d groups", len(personGroups))

			wg.Add(len(personGroups))

			for host, ids := range personGroups {
				go func(host string, ids []string) {
					failCount := 0
					for _, id := range ids {
						p, err := gcc.CrawlPerson(httpClient, id)
						if err != nil {
							failCount ++
							log.Printf("Crawling person %s failed: %v", id, err)
								
							schedulePerson(id, time.Now().Add(12 * time.Hour))
							
							if failCount >= 10 {
								log.Printf("Last ten crawling %s persons failed, sleep for a while...",
									host)
								time.Sleep(10 * time.Minute)
								failCount = 0
							}
							continue
						}

						log.Printf("Crawled person %s success!", id)
						pushPerson(p)
						log.Printf("Push person %s success", id)
					}

					wg.Done()
				}(host, ids)
			}
		}
		wg.Wait()

		syncDatabases()
		
		if processGodoc(httpClient) {
			didSomething = true
		}

		if !didSomething {
			log.Printf("Nothing to crawl sleep for a while...")
			time.Sleep(2 * time.Minute)
		}
	}
}

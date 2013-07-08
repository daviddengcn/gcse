package main

import (
	"encoding/gob"
	"encoding/json"
	"errors"
	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/gddo/doc"
	"log"
	"math/rand"
	"net/http"
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
	Version      int // if gcse.CrawlerVersion is different from this value, etag is ignored
	Etag         string
}

func init() {
	gob.Register(CrawlingEntry{})

	doc.SetGithubCredentials("94446b37edb575accd8b",
		"15f55815f0515a3f6ad057aaffa9ea83dceb220b")
	doc.SetUserAgent("Go-Code-Search-Agent")
}

func schedulePackage(pkg string, sTime time.Time, etag string) error {
	ent := CrawlingEntry{
		ScheduleTime: sTime,
		Version:      gcse.CrawlerVersion,
		Etag:         etag,
	}

	cPackageDB.Put(pkg, ent)

	log.Printf("Schedule package %s to %v", pkg, sTime)
	return nil
}

func appendPackage(pkg string) bool {
	pkg = strings.TrimFunc(strings.TrimSpace(pkg), func(r rune) bool {
		return r > rune(128)
	})
	if !doc.IsValidRemotePath(pkg) {
		//log.Printf("  [appendPackage] Not a valid remote path: %s", pkg)
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

	// if the package doesn't exist in docDB, Etag is discarded
	return schedulePackage(pkg, time.Now(), "") == nil
}

// reschedule if last crawl time is later than crawledBefore
func touchPackage(pkg string, crawledBefore time.Time) bool {
	pkg = strings.TrimSpace(pkg)
	if !doc.IsValidRemotePath(pkg) {
		//log.Printf("  [touchPackage] Not a valid remote path: %s", pkg)
		return false
	}
	
	var ent gcse.DocInfo 
	if docDB.Get(pkg, &ent) {
		if ent.LastUpdated.After(crawledBefore) {
			//log.Printf("  [touchPackage] no need to update: %s", pkg)
			return false
		}
	}

	// set Etag to "" to force updating
	return schedulePackage(pkg, time.Now(), "") == nil
}

var errStop = errors.New("Stop")

type EntryInfo struct {
	ID   string
	Etag string
}

func listCrawlEntriesByHost(db *gcse.MemDB, hostFromID func(id string) string,
	maxHosts, numPerHost int) (groups map[string][]EntryInfo) {
	now := time.Now()
	groups = make(map[string][]EntryInfo)
	fullGroups := 0
	db.Iterate(func(id string, val interface{}) error {
		ent, ok := val.(CrawlingEntry)
		if !ok {
			return nil
		}

		if ent.ScheduleTime.After(now) {
			return nil
		}

		host := hostFromID(id)
		entryInfos := groups[host]
		if maxHosts > 0 {
			// check host limit
			if len(entryInfos) == 0 && len(groups) == maxHosts {
				// no quota for new group
				return nil
			}
		}
		if numPerHost > 0 {
			// check per host limit
			if len(entryInfos) == numPerHost-1 {
				// this group is about to be full, count it
				fullGroups++
			} else if len(entryInfos) == numPerHost {
				// no quota for this group
				return nil
			}
		}
		
		etag := ent.Etag
		if ent.Version != gcse.CrawlerVersion {
			etag = ""
		}
		groups[host] = append(entryInfos, EntryInfo{
			ID:   id,
			Etag: etag,
		})

		if fullGroups == maxHosts {
			return errStop
		}
		return nil
	})

	return groups
}

func listPackagesByHost(maxHosts, numPerHost int) (groups map[string][]EntryInfo) {
	return listCrawlEntriesByHost(cPackageDB, gcse.HostOfPackage, maxHosts, numPerHost)
}

func listPersonsByHost(maxHosts, numPerHost int) (groups map[string][]EntryInfo) {
	return listCrawlEntriesByHost(cPersonDB, func(id string) string {
		site, _ := gcse.ParsePersonId(id)
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
	id := gcse.IdOfPerson(site, username)

	var ent CrawlingEntry
	exists := cPersonDB.Get(id, &ent)
	if exists {
		// already scheduled
		log.Printf("  [appendPerson] Person %s was scheduled to %v", id, ent.ScheduleTime)
		return false
	}

	return schedulePerson(id, time.Now()) == nil
}

func schedulePackageNextCrawl(pkg string, etag string) {
	schedulePackage(pkg, time.Now().Add(time.Duration(
		float64(DefaultPackageAge)*(1+(rand.Float64()-0.5)*0.2))), etag)

}

// push crawled Package to docDB as DocInfo
func pushPackage(p *gcse.Package) (succ bool) {
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

	schedulePackageNextCrawl(d.Package, p.Etag)

	return true
}

func pushPerson(p *gcse.Person) (hasNewPkg bool) {
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
	godocApiUrl   = "http://api.godoc.org/packages"
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

const (
	githubUpdatesGap = 4 * time.Hour
)

var (
	githubUpdatesCrawled time.Time
)

func touchByGithubUpdates() bool {
	if time.Now().Before(githubUpdatesCrawled.Add(githubUpdatesGap)) {
		return false
	}
	
	updates, err := gcse.GithubUpdates()
	if err != nil {
		log.Printf("GithubUpdates failed: %v", err)
		return false
	}
	
	log.Printf("%d updates found!", len(updates))
	
	res := false
	for pkg, ut := range updates {
		if touchPackage(pkg, ut) {
			res = true
		}
	}
	
	return res
}

func crawlEnetiresLoop() {
	httpClient := gcse.GenHttpClient("")

	for {
		didSomething := false
		var wg sync.WaitGroup

		pkgGroups := listPackagesByHost(5, 50)
		if len(pkgGroups) > 0 {
			didSomething = true

			log.Printf("Crawling packages of %d groups", len(pkgGroups))

			wg.Add(len(pkgGroups))

			for host, ents := range pkgGroups {
				go func(host string, ents []EntryInfo) {
					failCount := 0
					for _, ent := range ents {
						p, err := gcse.CrawlPackage(httpClient, ent.ID, ent.Etag)
						if err == gcse.ErrPackageNotModifed {
							log.Printf("Package %s unchanged!", ent.ID)
							schedulePackageNextCrawl(ent.ID, ent.Etag)
						}
						if err != nil {
							log.Printf("Crawling pkg %s failed: %v", ent.ID, err)

							if gcse.IsBadPackage(err) {
								// a wrong path
								deletePackage(ent.ID)
								log.Printf("Remove wrong package %s", ent.ID)
							} else {
								failCount++

								schedulePackage(ent.ID, time.Now().Add(
									12*time.Hour), ent.Etag)

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

						log.Printf("Crawled package %s success!", ent.ID)

						pushPackage(p)
						log.Printf("Package %s saved!", ent.ID)
					}

					wg.Done()
				}(host, ents)
			}
		}

		personGroups := listPersonsByHost(5, 100)
		if len(personGroups) > 0 {
			didSomething = true

			log.Printf("Crawling persons of %d groups", len(personGroups))

			wg.Add(len(personGroups))

			for host, ents := range personGroups {
				go func(host string, ents []EntryInfo) {
					failCount := 0
					for _, ent := range ents {
						p, err := gcse.CrawlPerson(httpClient, ent.ID)
						if err != nil {
							failCount++
							log.Printf("Crawling person %s failed: %v", ent.ID, err)

							schedulePerson(ent.ID, time.Now().Add(12*time.Hour))

							if failCount >= 10 {
								log.Printf("Last ten crawling %s persons failed, sleep for a while...",
									host)
								time.Sleep(10 * time.Minute)
								failCount = 0
							}
							continue
						}

						log.Printf("Crawled person %s success!", ent.ID)
						pushPerson(p)
						log.Printf("Push person %s success", ent.ID)
					}

					wg.Done()
				}(host, ents)
			}
		}
		wg.Wait()

		syncDatabases()

		if gcse.CrawlByGodocApi {
			if processGodoc(httpClient) {
				didSomething = true
			}
		}
		
		if touchByGithubUpdates() {
			didSomething = true
		}

		if !didSomething {
			log.Printf("Nothing to crawl sleep for a while...")
			time.Sleep(2 * time.Minute)
		}
	}
}

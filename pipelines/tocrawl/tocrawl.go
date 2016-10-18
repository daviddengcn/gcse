package main

import (
	"io"
	"log"
	"math/rand"
	"runtime"
	"strings"
	"time"

	"github.com/golangplus/errors"
	"github.com/golangplus/sort"
	"github.com/golangplus/time"

	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/gcse/configs"
	"github.com/daviddengcn/gcse/spider/github"
	"github.com/daviddengcn/gcse/spider/godocorg"
	"github.com/daviddengcn/gcse/store"
	"github.com/daviddengcn/gcse/utils"
	"github.com/daviddengcn/gddo/doc"
	"github.com/daviddengcn/go-easybi"
	"github.com/daviddengcn/go-villa"
	"github.com/daviddengcn/sophie"
	"github.com/daviddengcn/sophie/kv"

	sppb "github.com/daviddengcn/gcse/proto/spider"
)

var (
	cDB *gcse.CrawlerDB
)

func loadPackageUpdateTimes(fpDocs sophie.FsPath) (map[string]time.Time, error) {
	dir := kv.DirInput(fpDocs)
	cnt, err := dir.PartCount()
	if err != nil {
		return nil, err
	}

	pkgUTs := make(map[string]time.Time)

	var pkg sophie.RawString
	var info gcse.DocInfo
	for i := 0; i < cnt; i++ {
		it, err := dir.Iterator(i)
		if err != nil {
			return nil, err
		}
		for {
			if err := it.Next(&pkg, &info); err != nil {
				if errorsp.Cause(err) == io.EOF {
					break
				}
				return nil, err
			}

			pkgUTs[string(pkg)] = info.LastUpdated
		}
	}
	return pkgUTs, nil
}

func generateCrawlEntries(db *gcse.MemDB, hostFromID func(id string) string, out kv.DirOutput, pkgUTs map[string]time.Time) error {
	now := time.Now()
	type idAndCrawlingEntry struct {
		id  string
		ent *gcse.CrawlingEntry
	}
	groups := make(map[string][]idAndCrawlingEntry)
	count := 0
	type nameAndAges struct {
		maxName string
		maxAge  time.Duration

		sumAgeHours float64
		cnt         int

		// The number of packages not in pkgUTs
		newCnt int
	}
	ages := make(map[string]nameAndAges)
	if err := db.Iterate(func(id string, val interface{}) error {
		ent, ok := val.(gcse.CrawlingEntry)
		if !ok {
			log.Printf("Wrong entry: %+v", ent)
			return nil
		}
		if ent.Version == gcse.CrawlerVersion && ent.ScheduleTime.After(now) {
			return nil
		}
		host := hostFromID(id)

		// check host black list
		if configs.NonCrawlHosts.Contain(host) {
			return nil
		}
		if rand.Intn(10) == 0 {
			// randomly set Etag to empty to fetch stars
			ent.Etag = ""
		}
		groups[host] = append(groups[host], idAndCrawlingEntry{
			id:  id,
			ent: &ent,
		})

		age := now.Sub(ent.ScheduleTime)
		na := ages[host]
		if age > na.maxAge {
			na.maxName, na.maxAge = id, age
		}
		na.sumAgeHours += age.Hours()
		na.cnt++
		if _, ok := pkgUTs[id]; !ok {
			na.newCnt++
		}
		ages[host] = na

		count++
		return nil
	}); err != nil {
		return errorsp.WithStacks(err)
	}
	index := 0
	for _, g := range groups {
		sortp.SortF(len(g), func(i, j int) bool {
			if pkgUTs != nil {
				_, inDocsI := pkgUTs[g[i].id]
				_, inDocsJ := pkgUTs[g[j].id]
				if inDocsI != inDocsJ {
					// The one not in docs should be crawled first.
					// I.e. if g[i] in doc (inDocsI = true), g[j] not in doc (inDocsJ == false), shoud return false
					// vice versa.
					return inDocsJ
				}
			}
			return g[i].ent.ScheduleTime.Before(g[j].ent.ScheduleTime)
		}, func(i, j int) {
			g[i], g[j] = g[j], g[i]
		})
		if err := func(index int, ies []idAndCrawlingEntry) error {
			c, err := out.Collector(index)
			if err != nil {
				return err
			}
			defer c.Close()

			for i, ie := range ies {
				if err := c.Collect(sophie.RawString(ie.id), ie.ent); err != nil {
					return err
				}
				if i < 10 {
					log.Printf("id: %s, ent: %+v", ie.id, *ie.ent)
				}
			}
			return nil
		}(index, g); err != nil {
			log.Printf("Saving ents failed: %v", err)
		}
		index++
	}
	for host, na := range ages {
		aveAge := time.Duration(na.sumAgeHours / float64(na.cnt) * float64(time.Hour))
		log.Printf("%s age: max -> %v(%s), ave -> %v, new -> %v", host, na.maxAge, na.maxName, aveAge, na.newCnt)
		if host == "github.com" && strings.Contains(out.Path, configs.FnPackage) {
			gcse.AddBiValueAndProcess(bi.Average, "crawler.github_max_age.hours", int(na.maxAge.Hours()))
			gcse.AddBiValueAndProcess(bi.Average, "crawler.github_max_age.days", int(na.maxAge/timep.Day))
			gcse.AddBiValueAndProcess(bi.Average, "crawler.github_ave_age.hours", int(aveAge.Hours()))
			gcse.AddBiValueAndProcess(bi.Average, "crawler.github_ave_age.days", int(aveAge/timep.Day))
			gcse.AddBiValueAndProcess(bi.Average, "crawler.github_new_cnt", na.newCnt)
		}
	}
	log.Printf("%d entries to crawl for folder %v", count, out.Path)
	return nil
}

func syncDatabases() {
	utils.DumpMemStats()
	log.Printf("Synchronizing databases to disk...")
	if err := cDB.Sync(); err != nil {
		log.Fatalf("cdb.Sync() failed: %v", err)
	}
	utils.DumpMemStats()
	runtime.GC()
	utils.DumpMemStats()
}

func main() {
	log.Println("Running tocrawl tool, to generate crawling list")
	log.Println("NonCrawlHosts: ", configs.NonCrawlHosts)
	log.Println("CrawlGithubUpdate: ", configs.CrawlGithubUpdate)
	log.Println("CrawlByGodocApi: ", configs.CrawlByGodocApi)

	log.Printf("Using personal: %v", configs.CrawlerGithubPersonal)
	gcse.GithubSpider = github.NewSpiderWithToken(configs.CrawlerGithubPersonal)

	// Load CrawlerDB
	cDB = gcse.LoadCrawlerDB()

	// load pkgUTs
	pkgUTs, err := loadPackageUpdateTimes(sophie.LocalFsPath(configs.DocsDBPath()))
	if err != nil {
		log.Fatalf("loadPackageUpdateTimes failed: %v", err)
	}
	if configs.CrawlGithubUpdate || configs.CrawlByGodocApi {
		if configs.CrawlGithubUpdate {
			touchByGithubUpdates(pkgUTs)
		}

		if configs.CrawlByGodocApi {
			httpClient := gcse.GenHttpClient("")
			pkgs, err := godocorg.FetchAllPackagesInGodoc(httpClient)
			if err != nil {
				log.Fatalf("FetchAllPackagesInGodoc failed: %v", err)
			}
			gcse.AddBiValueAndProcess(bi.Max, "godoc.doc-count", len(pkgs))
			log.Printf("FetchAllPackagesInGodoc returns %d entries", len(pkgs))
			now := time.Now()
			for _, pkg := range pkgs {
				if !doc.IsValidRemotePath(pkg) {
					continue
				}
				cDB.AppendPackage(pkg, func(pkg string) bool {
					_, ok := pkgUTs[pkg]
					return ok
				})
				site, path := utils.SplitPackage(pkg)
				if err := store.AppendPackageEvent(site, path, "godoc", now, sppb.HistoryEvent_Action_None); err != nil {
					log.Printf("UpdatePackageHistory %s %s failed: %v", site, path, err)
				}
			}
		}
		syncDatabases()
	}

	log.Printf("Package DB: %d entries", cDB.PackageDB.Count())
	log.Printf("Person DB: %d entries", cDB.PersonDB.Count())

	pathToCrawl := villa.Path(configs.ToCrawlPath())

	kvPackage := kv.DirOutput(sophie.LocalFsPath(
		pathToCrawl.Join(configs.FnPackage).S()))
	kvPackage.Clean()
	if err := generateCrawlEntries(cDB.PackageDB, gcse.HostOfPackage, kvPackage, pkgUTs); err != nil {
		log.Fatalf("generateCrawlEntries %v failed: %v", kvPackage.Path, err)
	}

	kvPerson := kv.DirOutput(sophie.LocalFsPath(
		pathToCrawl.Join(configs.FnPerson).S()))

	kvPerson.Clean()
	if err := generateCrawlEntries(cDB.PersonDB, func(id string) string {
		site, _ := gcse.ParsePersonId(id)
		return site
	}, kvPerson, nil); err != nil {
		log.Fatalf("generateCrawlEntries %v failed: %v", kvPerson.Path, err)
	}
}

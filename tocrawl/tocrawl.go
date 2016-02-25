package main

import (
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
	"github.com/daviddengcn/go-easybi"
	"github.com/daviddengcn/sophie"
	"github.com/daviddengcn/sophie/kv"
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
				if err == sophie.EOF {
					break
				}
				return nil, err
			}

			pkgUTs[string(pkg)] = info.LastUpdated
		}
	}
	return pkgUTs, nil
}

func generateCrawlEntries(db *gcse.MemDB, hostFromID func(id string) string, out kv.DirOutput) error {
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

		if host != "github.com" {
			// Temporarily only crawl github.com
			return nil
		}

		// check host black list
		if configs.NonCrawlHosts.Contain(host) {
			return nil
		}
		if rand.Intn(10) == 0 {
			// randomly set Etag to empty to fetch stars
			ent.Etag = ""
		}
		groups[host] = append(groups[host], idAndCrawlingEntry{id, &ent})

		age := now.Sub(ent.ScheduleTime)
		na := ages[host]
		if age > na.maxAge {
			na.maxName, na.maxAge = id, age
		}
		na.sumAgeHours += age.Hours()
		na.cnt++
		ages[host] = na

		count++
		return nil
	}); err != nil {
		return errorsp.WithStacks(err)
	}
	index := 0
	for _, g := range groups {
		sortp.SortF(len(g), func(i, j int) bool {
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

			for _, ie := range ies {
				if err := c.Collect(sophie.RawString(ie.id), ie.ent); err != nil {
					return err
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
		log.Printf("%s age: max -> %v(%s), ave -> %v", host, na.maxAge, na.maxName, aveAge)
		if host == "github.com" && strings.Contains(out.Path, configs.FnPackage) {
			gcse.AddBiValueAndProcess(bi.Average, "crawler.github_max_age.hours", int(na.maxAge.Hours()))
			gcse.AddBiValueAndProcess(bi.Average, "crawler.github_max_age.days", int(na.maxAge/timep.Day))
			gcse.AddBiValueAndProcess(bi.Average, "crawler.github_ave_age.hours", int(aveAge.Hours()))
			gcse.AddBiValueAndProcess(bi.Average, "crawler.github_ave_age.days", int(aveAge/timep.Day))
		}
	}
	log.Printf("%d entries to crawl for folder %v", count, out.Path)
	return nil
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

func main() {
	log.Println("Running tocrawl tool, to generate crawling list")
	log.Println("NonCrawlHosts: ", configs.NonCrawlHosts)
	log.Println("CrawlGithubUpdate: ", configs.CrawlGithubUpdate)
	log.Println("CrawlByGodocApi: ", configs.CrawlByGodocApi)

	log.Printf("Using personal: %v", configs.CrawlerGithubPersonal)
	gcse.GithubSpider = github.NewSpiderWithToken(configs.CrawlerGithubPersonal)

	// Load CrawlerDB
	cDB = gcse.LoadCrawlerDB()

	if configs.CrawlGithubUpdate || configs.CrawlByGodocApi {
		// load pkgUTs
		pkgUTs, err := loadPackageUpdateTimes(
			sophie.LocalFsPath(configs.DocsDBPath().S()))
		if err != nil {
			log.Fatalf("loadPackageUpdateTimes failed: %v", err)
		}

		if configs.CrawlGithubUpdate {
			touchByGithubUpdates(pkgUTs)
		}

		if configs.CrawlByGodocApi {
			httpClient := gcse.GenHttpClient("")
			pkgs, err := gcse.FetchAllPackagesInGodoc(httpClient)
			if err != nil {
				log.Fatalf("FetchAllPackagesInGodoc failed: %v", err)
			}
			gcse.AddBiValueAndProcess(bi.Max, "godoc.doc-count", len(pkgs))
			log.Printf("FetchAllPackagesInGodoc returns %d entries", len(pkgs))
			for _, pkg := range pkgs {
				cDB.AppendPackage(pkg, func(pkg string) bool {
					_, ok := pkgUTs[pkg]
					return ok
				})
			}
		}
		syncDatabases()
	}

	log.Printf("Package DB: %d entries", cDB.PackageDB.Count())
	log.Printf("Person DB: %d entries", cDB.PersonDB.Count())

	pathToCrawl := configs.DataRoot.Join(configs.FnToCrawl)

	kvPackage := kv.DirOutput(sophie.LocalFsPath(
		pathToCrawl.Join(configs.FnPackage).S()))
	kvPackage.Clean()
	if err := generateCrawlEntries(cDB.PackageDB, gcse.HostOfPackage, kvPackage); err != nil {
		log.Fatalf("generateCrawlEntries %v failed: %v", kvPackage.Path, err)
	}

	kvPerson := kv.DirOutput(sophie.LocalFsPath(
		pathToCrawl.Join(configs.FnPerson).S()))

	kvPerson.Clean()
	if err := generateCrawlEntries(cDB.PersonDB, func(id string) string {
		site, _ := gcse.ParsePersonId(id)
		return site
	}, kvPerson); err != nil {
		log.Fatalf("generateCrawlEntries %v failed: %v", kvPerson.Path, err)
	}
}

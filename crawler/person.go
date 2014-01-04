package main

import (
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/sophie"
)

const (
	DefaultPersonAge = 10 * 24 * time.Hour
)

var (
	cPersonDB *gcse.MemDB
)

func schedulePerson(id string, sTime time.Time) error {
	var ent gcse.CrawlingEntry
	ent.ScheduleTime = sTime

	cPersonDB.Put(id, ent)

	log.Printf("Schedule person %s to %v", id, sTime)
	return nil
}

func appendPerson(site, username string) bool {
	id := gcse.IdOfPerson(site, username)

	var ent gcse.CrawlingEntry
	exists := cPersonDB.Get(id, &ent)
	if exists {
		// already scheduled
		// log.Printf("  [appendPerson] Person %s was scheduled to %v", id, ent.ScheduleTime)
		return false
	}

	return schedulePerson(id, time.Now()) == nil
}

type PersonCrawler struct {
	crawlerMapper

	part       int
	failCount  int
	httpClient *http.Client
}

func pushPerson(p *gcse.Person) {
	for _, pkg := range p.Packages {
		appendPackage(pkg)
	}

	schedulePerson(p.Id, time.Now().Add(time.Duration(
		float64(DefaultPersonAge)*(1+(rand.Float64()-0.5)*0.2))))
}

// OnlyMapper.Map
func (pc *PersonCrawler) Map(key, val sophie.SophieWriter,
	c []sophie.Collector) error {
	if time.Now().After(AppStopTime) {
		log.Printf("Timeout(key = %v), PersonCrawler part %d returns EOM", key, pc.part)
		return sophie.EOM
	}

	id := string(*key.(*sophie.RawString))
	// ent := val.(*gcse.CrawlingEntry)
	log.Printf("Crawling person %v\n", id)

	p, err := gcse.CrawlPerson(pc.httpClient, id)
	if err != nil {
		pc.failCount++
		log.Printf("Crawling person %s failed: %v", id, err)

		schedulePerson(id, time.Now().Add(12*time.Hour))

		if pc.failCount >= 10 {
			durToSleep := 10 * time.Minute
			if time.Now().Add(durToSleep).After(AppStopTime) {
				log.Printf("Timeout(key = %v), PersonCrawler part %d returns EOM", key, pc.part)
				return sophie.EOM
			}

			log.Printf("Last ten crawling persons failed, sleep for a while...(current: %s)",
				id)
			time.Sleep(durToSleep)
			pc.failCount = 0
		}
		return nil
	}

	log.Printf("Crawled person %s success!", id)
	pushPerson(p)
	log.Printf("Push person %s success", id)
	pc.failCount = 0
	return nil
}

type PeresonCrawlerFactory struct {
	httpClient *http.Client
}

func (pcf PeresonCrawlerFactory) NewMapper(part int) sophie.OnlyMapper {
	return &PersonCrawler{part: part, httpClient: pcf.httpClient}
}

// crawl packages, send error back to end
func crawlPersons(httpClient *http.Client, fpToCrawlPsn sophie.FsPath, end chan error) {
	end <- func() error {
		job := sophie.MapOnlyJob{
			Source: []sophie.Input{
				sophie.KVDirInput(fpToCrawlPsn),
			},

			MapFactory: sophie.OnlyMapperFactoryFunc(
				func(src, part int) sophie.OnlyMapper {
					return &PersonCrawler{
						part:       part,
						httpClient: httpClient,
					}
				}),
		}

		if err := job.Run(); err != nil {
			log.Printf("crawlPersons: job.Run failed: %v", err)
			return err
		}
		return nil
	}()
}

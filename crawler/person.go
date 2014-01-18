package main

import (
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/daviddengcn/gddo/doc"
	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/sophie"
)

const (
	DefaultPersonAge = 10 * 24 * time.Hour
)

type PersonCrawler struct {
	crawlerMapper

	part       int
	failCount  int
	httpClient doc.HttpClient
}

func pushPerson(p *gcse.Person) {
	for _, pkg := range p.Packages {
		appendPackage(pkg)
	}

	cDB.SchedulePerson(p.Id, time.Now().Add(time.Duration(
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

		cDB.SchedulePerson(id, time.Now().Add(12*time.Hour))

		if pc.failCount >= 10 || strings.Contains(err.Error(), "403") {
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

	time.Sleep(10 * time.Second)

	return nil
}

type PeresonCrawlerFactory struct {
	httpClient doc.HttpClient
}

func (pcf PeresonCrawlerFactory) NewMapper(part int) sophie.OnlyMapper {
	return &PersonCrawler{part: part, httpClient: pcf.httpClient}
}

// crawl packages, send error back to end
func crawlPersons(httpClient doc.HttpClient, fpToCrawlPsn sophie.FsPath, end chan error) {
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

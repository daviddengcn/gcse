package main

import  (
	"log"
	"strings"
	"time"
	
	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/gddo/doc"
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

func touchPackage(pkg string, crawledBefore time.Time,
		pkgUTs map[string]time.Time) {
	pkg = strings.TrimSpace(pkg)
	if !doc.IsValidRemotePath(pkg) {
		//log.Printf("  [touchPackage] Not a valid remote path: %s", pkg)
		return
	}

	ut, ok := pkgUTs[pkg]
	if ok && ut.After(crawledBefore) {
		return
	}

	// set Etag to "" to force updating
	schedulePackage(pkg, time.Now(), "")
}

func touchByGithubUpdates(pkgUTs map[string]time.Time) {
	log.Printf("touchByGithubUpdates ...")
	
	updates, err := gcse.GithubUpdates()
	if err != nil {
		log.Printf("GithubUpdates failed: %v", err)
	}
	
	log.Printf("%d updates found!", len(updates))
	
	for pkg, ut := range updates {
		touchPackage(pkg, ut, pkgUTs)
	}
}

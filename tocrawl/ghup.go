package main

import (
	"log"
	"strings"
	"time"

	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/gddo/doc"
)

// touchPackage forces a package to update if it was not crawled before a
// specific time.
func touchPackage(pkg string, crawledBefore time.Time, pkgUTs map[string]time.Time) {
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
	cDB.SchedulePackage(pkg, time.Now(), "")
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

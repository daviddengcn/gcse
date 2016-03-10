package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/gcse/store"
	"github.com/daviddengcn/gddo/doc"
	"github.com/golangplus/strings"

	sppb "github.com/daviddengcn/gcse/proto/spider"
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
	cDB.PushToCrawlPackage(pkg)
}

func touchByGithubUpdates(pkgUTs map[string]time.Time) {
	log.Printf("touchByGithubUpdates ...")

	rs, err := gcse.GithubSpider.SearchRepositories("")
	if err != nil {
		log.Printf("SearchRepositories failed: %v", err)
		return
	}
	count := 0
	now := time.Now()
	for _, r := range rs {
		if r.Owner == nil || r.UpdatedAt == nil {
			continue
		}
		user := stringsp.Get(r.Owner.Name)
		path := stringsp.Get(r.Name)
		if user == "" || path == "" {
			continue
		}
		touchPackage(fmt.Sprintf("github.com/%s/%s", user, path), r.UpdatedAt.Time, pkgUTs)
		if err := store.AppendPackageEvent("github.com", user+"/"+path, "githubhupdate", now, sppb.HistoryEvent_Action_None); err != nil {
			log.Printf("UpdatePackageHistory %s %s failed: %v", "github.com", user+"/"+path, err)
		}
		count++
	}
	log.Printf("%d updates found!", count)
}

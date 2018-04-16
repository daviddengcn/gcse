package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/gcse/store"
	"github.com/daviddengcn/gddo/doc"
	"github.com/golang/glog"

	gpb "github.com/daviddengcn/gcse/shared/proto"
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

func touchByGithubUpdates(ctx context.Context, pkgUTs map[string]time.Time) {
	log.Printf("touchByGithubUpdates ...")

	rs, err := gcse.GithubSpider.SearchRepositories(ctx, "")
	if err != nil {
		log.Printf("SearchRepositories failed: %v", err)
		return
	}
	count := 0
	now := time.Now()
	emptyOwnerOrUpdatedAt, emptyUserOrPath := 0, 0
	for _, r := range rs {
		if r.Owner == nil || r.UpdatedAt == nil {
			emptyOwnerOrUpdatedAt++
			continue
		}
		user := r.Owner.GetName()
		if user == "" {
			user = r.Owner.GetLogin()
		}
		path := r.GetName()
		if user == "" || path == "" {
			emptyUserOrPath++
			continue
		}
		touchPackage(fmt.Sprintf("github.com/%s/%s", user, path), r.UpdatedAt.Time, pkgUTs)
		if err := store.AppendPackageEvent("github.com", user+"/"+path, "githubhupdate", now, gpb.HistoryEvent_Action_None); err != nil {
			log.Printf("UpdatePackageHistory %s %s failed: %v", "github.com", user+"/"+path, err)
		}
		count++
	}
	glog.Infof("%d updates found!", count)
	glog.Infof("Total: %d, emptyOwnerOrUpdatedAt: %d, emptyUserOrPath: %d", len(rs), emptyOwnerOrUpdatedAt, emptyUserOrPath)
}

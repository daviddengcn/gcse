package main

import (
//	"log"
)

const (
	godocApiUrl   = "http://api.godoc.org/packages"
	godocCrawlGap = 4 * time.Hour
)
/*
func touchByGithubUpdates() bool {
	log.Printf("touchByGithubUpdates ...")

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
*/
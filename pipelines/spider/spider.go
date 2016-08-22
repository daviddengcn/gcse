package main

import (
	"log"
	"time"

	"github.com/golangplus/container/heap"
	"github.com/golangplus/errors"
	"github.com/golangplus/time"

	"github.com/daviddengcn/gcse/configs"
	"github.com/daviddengcn/gcse/proto/spider"
	"github.com/daviddengcn/gcse/proto/store"
	"github.com/daviddengcn/gcse/spider/github"
	"github.com/daviddengcn/gcse/store"
)

type RepositoryInfo struct {
	*stpb.Repository

	User string
	Name string
}

func needCrawl(r *stpb.Repository) bool {
	if r.CrawlingInfo == nil {
		return true
	}
	return r.CrawlingInfo.CrawlingTimeAsTime().Before(time.Now().Add(-timep.Day))
}

func shouldCrawlLater(a, b *RepositoryInfo) bool {
	if a.CrawlingInfo == nil {
		if b.CrawlingInfo == nil {
			return a.Name+a.User < b.Name+b.User
		}
		return false
	}
	if b.CrawlingInfo == nil {
		return true
	}
	return a.CrawlingInfo.CrawlingTimeAsTime().After(b.CrawlingInfo.CrawlingTimeAsTime())
}

func selectRepos(site string, maxCrawl int) ([]*RepositoryInfo, error) {
	repos := heap.NewInterfaces(func(x, y interface{}) bool {
		return shouldCrawlLater(x.(*RepositoryInfo), y.(*RepositoryInfo))
	}, maxCrawl)
	if err := store.ForEachRepositoryOfSite(site, func(user, name string, doc *stpb.Repository) error {
		if !needCrawl(doc) {
			return nil
		}
		ri := &RepositoryInfo{
			User:       user,
			Name:       name,
			Repository: doc,
		}
		repos.TopNPush(ri)
		return nil
	}); err != nil {
		return nil, err
	}
	res := make([]*RepositoryInfo, 0, repos.Len())
	for _, r := range repos.PopAll() {
		res = append(res, r.(*RepositoryInfo))
	}
	return res, nil
}

var githubSpider *github.Spider
var now timep.NowFunc = time.Now

func crawlRepo(site string, repo *RepositoryInfo) error {
	if site != "github.com" {
		return errorsp.NewWithStacks("Cannot crawl the repository in %v", site)
	}
	repo.CrawlingInfo = &sppb.CrawlingInfo{}
	repo.CrawlingInfo.SetCrawlingTime(now())

	sha, err := githubSpider.RepoBranchSHA(repo.User, repo.Name, repo.Branch)
	if err != nil {
		return err
	}
	if repo.Signature == sha {
		return nil
	}
	repo.Signature = sha

	repo.Packages = make(map[string]*sppb.Package)
	if err := githubSpider.ReadRepo(repo.User, repo.Name, repo.Signature, func(path string, doc *sppb.Package) error {
		log.Printf("Package: %v", doc)
		repo.Packages[path] = doc
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func crawlAndSaveRepo(site string, repo *RepositoryInfo) error {
	if err := crawlRepo(site, repo); err != nil {
		if errorsp.Cause(err) == github.ErrInvalidRepository {
			// Remove the repo entry.
			return store.DeleteRepository(site, repo.User, repo.Name)
		}
		return err
	}
	return store.UpdateRepository(site, repo.User, repo.Name, func(doc *stpb.Repository) error {
		*doc = *repo.Repository
		return nil
	})
}

func crawl(site string, out chan error, maxCrawl int, dur time.Duration) {
	repos, err := selectRepos(site, maxCrawl)
	if err != nil {
		out <- err
		return
	}
	log.Printf("%d repos selected", len(repos))
	var anyErr error
	for _, repo := range repos {
		if err := crawlAndSaveRepo(site, repo); err != nil {
			anyErr = err
			log.Printf("crawlAndSaveRepo %v %v %v failed: %v", site, repo.User, repo.Name, err)
		}
	}
	out <- anyErr
}

func exec(maxCrawl int, dur time.Duration) error {
	out := make(chan error)
	n := 0
	anyErr := store.ForEachRepositorySite(func(site string) error {
		n++
		go crawl(site, out, maxCrawl, dur)
		return nil
	})
	if anyErr != nil {
		log.Printf("ForEachRepositorySite failed: %v", anyErr)
	}
	log.Printf("Waiting for %d site(s)...", n)
	for ; n > 0; n-- {
		if e := <-out; e != nil {
			log.Printf("Error from out: %v", e)
			anyErr = e
		}
	}
	return anyErr
}

func main() {
	log.Printf("Using Github personal token: %v", configs.CrawlerGithubPersonal)
	githubSpider = github.NewSpiderWithToken(configs.CrawlerGithubPersonal)

	if err := exec(1000, configs.CrawlerDuePerRun); err != nil {
		log.Fatalf("exec failed: %v", err)
	}
}

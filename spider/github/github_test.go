package github

import (
	"testing"

	"github.com/daviddengcn/gcse"
	"github.com/golangplus/testing/assert"
)

func TestReadUser(t *testing.T) {
	s := NewSpiderWithToken(gcse.CrawlerGithubPersonal)
	assert.Should(t, s != nil, "s == nil")

	da, err := s.ReadUser("daviddengcn")
	assert.NoErrorOrDie(t, err)
	assert.ValueShould(t, "len(da.Repos)", len(da.Repos), len(da.Repos) > 0, "== 0")
}

func TestReadRepository(t *testing.T) {
	s := NewSpiderWithToken(gcse.CrawlerGithubPersonal)
	assert.Should(t, s != nil, "s == nil")

	repo, err := s.ReadRepository("daviddengcn", "gosl", true)
	assert.NoErrorOrDie(t, err)
	assert.ValueShould(t, "len(repo.Packages)", len(repo.Packages), len(repo.Packages) > 0, "== 0")
}

func TestReadPackage(t *testing.T) {
	s := NewSpiderWithToken(gcse.CrawlerGithubPersonal)
	assert.Should(t, s != nil, "s == nil")

	pkg, err := s.ReadPackage("daviddengcn", "gosl", "builtin")
	assert.NoErrorOrDie(t, err)
	assert.ValueShould(t, "len(pkg.Imports)", len(pkg.Imports), len(pkg.Imports) > 0, "== 0")
}

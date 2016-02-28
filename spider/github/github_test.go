package github

import (
	"testing"

	"github.com/golangplus/testing/assert"
)

func TestReadUser(t *testing.T) {
	s := NewSpiderWithToken("")
	assert.Should(t, s != nil, "s == nil")

	da, err := s.ReadUser("daviddengcn")
	assert.NoErrorOrDie(t, err)
	assert.ValueShould(t, "len(da.Repos)", len(da.Repos), len(da.Repos) > 0, "> 0")
}

func TestReadRepository(t *testing.T) {
	s := NewSpiderWithToken("")
	assert.Should(t, s != nil, "s == nil")

	repo, err := s.ReadRepository("daviddengcn", "gosl", true)
	assert.NoErrorOrDie(t, err)
	assert.ValueShould(t, "len(repo.Packages)", len(repo.Packages), len(repo.Packages) > 0, "> 0")
}

func TestReadPackage(t *testing.T) {
	s := NewSpiderWithToken("")
	assert.Should(t, s != nil, "s == nil")

	pkg, folders, err := s.ReadPackage("daviddengcn", "gcse", "spider/github/testdata")
	assert.NoErrorOrDie(t, err)
	assert.Equal(t, "pkg.Name", pkg.Name, "pkg")
	assert.Equal(t, "pkg.Imports", pkg.Imports, []string{
		"github.com/daviddengcn/gcse/spider/github",
		"github.com/golangplus/strings",
	})
	assert.Equal(t, "pkg.TestImports", pkg.TestImports, []string{"github.com/golangplus/testing/assert"})
	assert.Equal(t, "len(folders)", len(folders), 1)
	assert.Equal(t, "folders[0].Name", folders[0].Name, "sub")
	assert.Equal(t, "folders[0].Path", folders[0].Path, "spider/github/testdata/sub")
}

func TestSearchRepositories(t *testing.T) {
	s := NewSpiderWithToken("")
	assert.Should(t, s != nil, "s == nil")

	rs, err := s.SearchRepositories("")
	assert.NoErrorOrDie(t, err)
	assert.ValueShould(t, "len(rs)", len(rs), len(rs) > 0, "> 0")
}

func TestParseGoFile(t *testing.T) {
	assert.Equal(t, "parseGoFile", parseGoFile("g.go", []byte(`
package main
`+`// +build ignore
	`)), GoFileInfo{Status: ShouldIgnored})
}

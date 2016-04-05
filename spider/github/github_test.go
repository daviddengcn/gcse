package github

import (
	"testing"

	"github.com/golangplus/errors"
	"github.com/golangplus/testing/assert"

	sppb "github.com/daviddengcn/gcse/proto/spider"
)

//func TestReadUser(t *testing.T) {
//	s := NewSpiderWithToken("")
//	assert.Should(t, s != nil, "s == nil")

//	da, err := s.ReadUser("daviddengcn")
//	assert.NoErrorOrDie(t, err)
//	assert.ValueShould(t, "len(da.Repos)", len(da.Repos), len(da.Repos) > 0, "> 0")
//}

//func TestReadRepository(t *testing.T) {
//	s := NewSpiderWithToken("")
//	assert.Should(t, s != nil, "s == nil")

//	repo, err := s.ReadRepository("daviddengcn", "gosl")
//	assert.NoErrorOrDie(t, err)
//	assert.ValueShould(t, "repo.Stars", repo.Stars, repo.Stars > 0, "> 0")
//}

//func TestReadPackage(t *testing.T) {
//	s := NewSpiderWithToken("")
//	assert.Should(t, s != nil, "s == nil")

//	pkg, folders, err := s.ReadPackage("daviddengcn", "gcse", "spider/github/testdata")
//	assert.NoErrorOrDie(t, err)
//	assert.Equal(t, "pkg.Name", pkg.Name, "pkg")
//	sort.Strings(pkg.Imports)
//	assert.Equal(t, "pkg.Imports", pkg.Imports, []string{
//		"github.com/daviddengcn/gcse/spider/github",
//		"github.com/golangplus/strings",
//	})
//	assert.Equal(t, "pkg.TestImports", pkg.TestImports, []string{"github.com/golangplus/testing/assert"})
//	assert.Equal(t, "len(folders)", len(folders), 1)
//	assert.Equal(t, "folders[0].Name", folders[0].Name, "sub")
//	assert.Equal(t, "folders[0].Path", folders[0].Path, "spider/github/testdata/sub")
//}

//func TestSearchRepositories(t *testing.T) {
//	s := NewSpiderWithToken("")
//	assert.Should(t, s != nil, "s == nil")

//	rs, err := s.SearchRepositories("")
//	assert.NoErrorOrDie(t, err)
//	assert.ValueShould(t, "len(rs)", len(rs), len(rs) > 0, "> 0")
//}

func TestParseGoFile(t *testing.T) {
	fi := &sppb.GoFileInfo{}
	parseGoFile("g.go", []byte(`
package main
`+`// +build ignore
	`), fi)
	assert.Equal(t, "fi", fi, &sppb.GoFileInfo{Status: sppb.GoFileInfo_ShouldIgnore})
}

func TestRepoBranchSHA(t *testing.T) {
	s := NewSpiderWithContents(map[string]string{
		"/repos/daviddengcn/repo-branch-sha/branches/master": `
			{
			  "name": "master",
			  "commit": {
    		    "sha": "sha-1"
			  }
			}
		`,
	})
	sha, err := s.RepoBranchSHA("daviddengcn", "repo-branch-sha", "master")
	assert.NoError(t, err)
	assert.Equal(t, "sha", sha, "sha-1")
}

func TestRepoBranchSHA_NotFound(t *testing.T) {
	s := NewSpiderWithContents(map[string]string{})
	_, err := s.RepoBranchSHA("noone", "nothing", "master")
	assert.Equal(t, "err", errorsp.Cause(err), ErrInvalidRepository)
}

func TestReadRepo(t *testing.T) {
	s := NewSpiderWithContents(map[string]string{
		"/repos/daviddengcn/readrepo/branches/master": `
			{
			  "name": "master",
			  "commit": {
    		    "sha": "sha-1"
			  }
			}
		`,
		"/repos/daviddengcn/readrepo/git/trees/sha-1?recursive=1": `
			{
				"sha": "sha-1",
				"tree": [
					{
						"path": "a.go",
						"type": "blob",
						"sha": "sha-2"
					},
					{
						"path": "sub/a.go",
						"type": "blob",
						"sha": "sha-2"
					}
				],
				"truncated": false
			}`,
		"/repos/daviddengcn/readrepo/contents/a.go": `
			{
				"name": "bi.go",
				"path": "bi.go",
				"sha": "sha-2",
				"content": "cGFja2FnZSBnY3NlCgppbXBvcnQgKAoJImdpdGh1Yi5jb20vZGF2aWRkZW5n\nY24vZ28tZWFzeWJpIgopCgpmdW5jIEFkZEJpVmFsdWVBbmRQcm9jZXNzKGFn\nZ3IgYmkuQWdncmVnYXRlTWV0aG9kLCBuYW1lIHN0cmluZywgdmFsdWUgaW50\nKSB7CgliaS5BZGRWYWx1ZShhZ2dyLCBuYW1lLCB2YWx1ZSkKCWJpLkZsdXNo\nKCkKCWJpLlByb2Nlc3MoKQp9Cg==\n",
				"encoding": "base64",
				"type": "file"
			}
		`,
		"/repos/daviddengcn/readrepo/contents/sub/a.go": `
			{
				"name": "bi.go",
				"path": "bi.go",
				"sha": "sha-2",
				"content": "cGFja2FnZSBnY3NlCgppbXBvcnQgKAoJImdpdGh1Yi5jb20vZGF2aWRkZW5n\nY24vZ28tZWFzeWJpIgopCgpmdW5jIEFkZEJpVmFsdWVBbmRQcm9jZXNzKGFn\nZ3IgYmkuQWdncmVnYXRlTWV0aG9kLCBuYW1lIHN0cmluZywgdmFsdWUgaW50\nKSB7CgliaS5BZGRWYWx1ZShhZ2dyLCBuYW1lLCB2YWx1ZSkKCWJpLkZsdXNo\nKCkKCWJpLlByb2Nlc3MoKQp9Cg==\n",
				"encoding": "base64",
				"type": "file"
			}
		`,
	})
	pkgs := make(map[string]*sppb.Package)
	assert.NoError(t, s.ReadRepo("daviddengcn", "readrepo", "sha-1", func(path string, pkg *sppb.Package) error {
		pkgs[path] = pkg
		return nil
	}))
	assert.Equal(t, "pkgs", pkgs, map[string]*sppb.Package{
		"": &sppb.Package{
			Name:        "gcse",
			Path:        "",
			Imports:     []string{"github.com/daviddengcn/go-easybi"},
			TestImports: []string{},
		},
		"/sub": &sppb.Package{
			Name:        "gcse",
			Path:        "/sub",
			Imports:     []string{"github.com/daviddengcn/go-easybi"},
			TestImports: []string{},
		},
	})
}

func TestReadRepo_NotFound(t *testing.T) {
	s := NewSpiderWithContents(map[string]string{})
	assert.Equal(t, "err", errorsp.Cause(s.ReadRepo("noone", "nothing", "sha-1", nil)), ErrInvalidRepository)
}

package github

import (
	"testing"

	"github.com/golangplus/testing/assert"
)

func TestReadUser(t *testing.T) {
	s := NewSpiderWithToken("4c02234f457197af55e37346d7fb599a27d8d421")
	assert.Should(t, s != nil, "s == nil")

	da, err := s.ReadUser("daviddengcn")
	assert.NoErrorOrDie(t, err)
	assert.ValueShould(t, "len(da.Repos)", len(da.Repos), len(da.Repos) > 0, "== 0")
}

func TestReadRepositry(t *testing.T) {
	s := NewSpiderWithToken("4c02234f457197af55e37346d7fb599a27d8d421")
	assert.Should(t, s != nil, "s == nil")

	repo, err := s.ReadRepositry("daviddengcn", "gosl")
	assert.NoErrorOrDie(t, err)
	assert.ValueShould(t, "len(repo.Packages)", len(repo.Packages), len(repo.Packages) > 0, "== 0")
}

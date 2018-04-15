package store

import (
	"testing"

	"github.com/golangplus/testing/assert"

	gpb "github.com/daviddengcn/gcse/shared/proto"
)

func TestUpdateReadDeleteRepository(t *testing.T) {
	const (
		site = "TestUpdateReadDeleteRepository.com"
		user = "daviddengcn"
		repo = "gcse"
	)
	assert.NoError(t, UpdateRepository(site, user, repo, func(doc *gpb.Repository) error {
		assert.Equal(t, "doc", doc, &gpb.Repository{})
		doc.Stars = 10
		return nil
	}))
	r, err := ReadRepository(site, user, repo)
	assert.NoError(t, err)
	assert.Equal(t, "r", r, &gpb.Repository{Stars: 10})

	assert.NoError(t, DeleteRepository(site, user, repo))

	r, err = ReadRepository(site, user, repo)
	assert.NoError(t, err)
	assert.Equal(t, "r", r, &gpb.Repository{})
}

func TestForEachRepositorySite(t *testing.T) {
	cleanDatabase(t)

	const (
		site = "TestForEachRepositorySite.com"
		user = "daviddengcn"
		repo = "gcse"
	)
	assert.NoError(t, UpdateRepository(site, user, repo, func(doc *gpb.Repository) error {
		return nil
	}))
	var sites []string
	assert.NoError(t, ForEachRepositorySite(func(site string) error {
		sites = append(sites, site)
		return nil
	}))
	assert.Equal(t, "sites", sites, []string{site})
}

func TestForEachRepositoryOfSite(t *testing.T) {
	const (
		site = "TestForEachRepositoryOfSite.com"
		user = "daviddengcn"
		repo = "gcse"
	)
	assert.NoError(t, UpdateRepository(site, user, repo, func(doc *gpb.Repository) error {
		doc.ReadmeData = "hello"
		return nil
	}))
	assert.NoError(t, ForEachRepositoryOfSite(site, func(u, r string, doc *gpb.Repository) error {
		assert.Equal(t, "user", u, user)
		assert.Equal(t, "repo", r, repo)
		assert.Equal(t, "doc", doc, &gpb.Repository{ReadmeData: "hello"})
		return nil
	}))
}

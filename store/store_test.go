package store

import (
	"log"
	"os"
	"testing"

	"github.com/golangplus/testing/assert"

	"github.com/daviddengcn/gcse/configs"
	"github.com/daviddengcn/go-villa"

	spb "github.com/daviddengcn/gcse/proto"
)

func init() {
	configs.DataRoot = villa.Path(os.TempDir()).Join("gcse_testing")
	configs.DataRoot.RemoveAll()
	configs.DataRoot.MkdirAll(0755)
	log.Printf("DataRoot: %v", configs.DataRoot)
}

func TestStoreDeleteRepoInfo(t *testing.T) {
	const (
		site = "TestStoreDeleteRepoInfo.com"
		user = "anonymous"
		path = "fake"
	)

	assert.NoError(t, SaveRepoInfo(site, user, path, &spb.RepoInfo{Stars: 123, Description: "hello"}))
	r, err := FetchRepoInfo(site, user, path)
	assert.NoError(t, err)
	assert.Equal(t, "r", r, &spb.RepoInfo{Stars: 123, Description: "hello"})

	assert.NoError(t, DeleteRepoInfo(site, user, path))
	r, err = FetchRepoInfo("example.com", user, path)
	assert.NoError(t, err)
	assert.Equal(t, "r", r, spb.RepoInfo{})
}

func TestForEachReposInSite(t *testing.T) {
	const (
		site  = "TestForEachReposInSite.com"
		user1 = "first"
		user2 = "second"
	)

	assert.NoError(t, SaveRepoInfo(site, user1, "one", &spb.RepoInfo{Stars: 123, Description: "hello 1"}))
	assert.NoError(t, SaveRepoInfo(site, user1, "two", &spb.RepoInfo{Stars: 456, Description: "hello 2"}))
	assert.NoError(t, SaveRepoInfo(site, user2, "three", &spb.RepoInfo{Stars: 789, Description: "hello 3"}))

	type all_info struct {
		user string
		path string
		info *spb.RepoInfo
	}
	var collected []all_info
	assert.NoError(t, ForEachReposInSite(site, func(user, path string, info *spb.RepoInfo) error {
		collected = append(collected, all_info{
			user: user,
			path: path,
			info: info,
		})
		return nil
	}))
	assert.Equal(t, "collected", collected, []all_info{{
		user: user1,
		path: "one",
		info: &spb.RepoInfo{Stars: 123, Description: "hello 1"},
	}, {
		user: user1,
		path: "two",
		info: &spb.RepoInfo{Stars: 456, Description: "hello 2"},
	}, {
		user: user2,
		path: "three",
		info: &spb.RepoInfo{Stars: 789, Description: "hello 3"},
	}})
}

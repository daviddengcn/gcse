package store

import (
	"encoding/gob"
	"time"

	"github.com/golangplus/bytes"

	"github.com/daviddengcn/bolthelper"
	"github.com/daviddengcn/gcse/configs"
)

var (
	// repo
	//   - <repo-path> -> RepoInfo
	repoRoot = []byte("repo")
)

type RepoInfo struct {
	LastUpdated time.Time

	RepoUpdated time.Time
	Stars       int
	Description string
}

func (r RepoInfo) Age() time.Duration {
	return time.Now().Sub(r.LastUpdated)
}

func init() {
	gob.Register(RepoInfo{})
}

var box = bh.RefCountBox{
	DataPath: func() string {
		return configs.DataRoot.Join("store.bolt").S()
	},
}

func FetchRepoInfo(site, user, path string) (*RepoInfo, error) {
	var r RepoInfo
	if err := box.View(func(tx bh.Tx) error {
		return tx.GobValue([][]byte{repoRoot, []byte(site), []byte(user), []byte(path)}, func(v interface{}) error {
			r, _ = v.(RepoInfo)
			return nil
		})
	}); err != nil {
		return nil, err
	}
	return &r, nil
}

func ForEachReposInSite(site string, f func(user, path string, info RepoInfo) error) error {
	return box.View(func(tx bh.Tx) error {
		return tx.ForEach([][]byte{repoRoot, []byte(site)}, func(b bh.Bucket, user, v bytesp.Slice) error {
			if v != nil {
				return nil
			}
			return b.ForEachGob([][]byte{user}, func(path bytesp.Slice, v interface{}) error {
				info, _ := v.(RepoInfo)
				return f(string(user), string(path), info)
			})
		})
	})
}

func SaveRepoInfo(site, user, path string, r RepoInfo) error {
	return box.Update(func(tx bh.Tx) error {
		return tx.PutGob([][]byte{repoRoot, []byte(site), []byte(user), []byte(path)}, &r)
	})
}

func DeleteRepoInfo(site, user, path string) error {
	return box.Update(func(tx bh.Tx) error {
		return tx.Delete([][]byte{repoRoot, []byte(site), []byte(user), []byte(path)})
	})
}

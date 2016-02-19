package gcse

import (
	"encoding/gob"
	"time"

	"github.com/daviddengcn/bolthelper"
)

var (
	// repo
	//   - <repo-path> -> RepoInfo
	repoRoot = []byte("repo")
)

type RepoInfo struct {
	LastUpdated time.Time

	Stars       int
	Description string
}

func (r RepoInfo) Age() time.Duration {
	return time.Now().Sub(r.LastUpdated)
}

func init() {
	gob.Register(RepoInfo{})
}

var store = bh.RefCountBox{
	DataPath: func() string {
		return DataRoot.Join("store.bolt").S()
	},
}

func FetchRepoInfo(site, user, path string) (*RepoInfo, error) {
	var r RepoInfo
	if err := store.View(func(tx bh.Tx) error {
		return tx.GobValue([][]byte{repoRoot, []byte(site), []byte(user), []byte(path)}, func(v interface{}) error {
			r, _ = v.(RepoInfo)
			return nil
		})
	}); err != nil {
		return nil, err
	}
	return &r, nil
}

func SaveRepoInfo(site, user, path string, r RepoInfo) error {
	return store.Update(func(tx bh.Tx) error {
		return tx.PutGob([][]byte{repoRoot, []byte(site), []byte(user), []byte(path)}, &r)
	})
}

func DeleteRepoInfo(site, user, path string) error {
	return store.Update(func(tx bh.Tx) error {
		return tx.Delete([][]byte{repoRoot, []byte(site), []byte(user), []byte(path)})
	})
}

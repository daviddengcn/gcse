package store

import (
	"log"

	"github.com/golangplus/bytes"
	"github.com/golangplus/errors"

	"github.com/daviddengcn/bolthelper"
	"github.com/daviddengcn/gcse/configs"
	"github.com/golang/protobuf/proto"

	spb "github.com/daviddengcn/gcse/proto"
)

var (
	// repo
	//   - <repo-path> -> RepoInfo
	repoRoot = []byte("repo")
)

var box = bh.RefCountBox{
	DataPath: func() string {
		return configs.DataRoot.Join("store.bolt").S()
	},
}

func FetchRepoInfo(site, user, path string) (*spb.RepoInfo, error) {
	var r *spb.RepoInfo
	if err := box.View(func(tx bh.Tx) error {
		return tx.Value([][]byte{repoRoot, []byte(site), []byte(user), []byte(path)}, func(v bytesp.Slice) error {
			r = &spb.RepoInfo{}
			return errorsp.WithStacksAndMessage(proto.Unmarshal(v, r), "len = %d", len(v))
		})
	}); err != nil {
		return nil, err
	}
	return r, nil
}

func ForEachReposInSite(site string, f func(user, path string, info *spb.RepoInfo) error) error {
	return box.View(func(tx bh.Tx) error {
		return tx.ForEach([][]byte{repoRoot, []byte(site)}, func(b bh.Bucket, user, v bytesp.Slice) error {
			if v != nil {
				return nil
			}
			return b.ForEach([][]byte{user}, func(path bytesp.Slice, v bytesp.Slice) error {
				r := &spb.RepoInfo{}
				if err := errorsp.WithStacksAndMessage(proto.Unmarshal(v, r), "len = %d", len(v)); err != nil {
					log.Printf("Unmarshal RepoInfo for %s failed: %v, ignored", string(path), err)
					return nil
				}
				return f(string(user), string(path), r)
			})
		})
	})
}

func SaveRepoInfo(site, user, path string, r *spb.RepoInfo) error {
	return box.Update(func(tx bh.Tx) error {
		bs, err := proto.Marshal(r)
		if err != nil {
			return errorsp.WithStacksAndMessage(err, "marshal %v", r)
		}
		return tx.Put([][]byte{repoRoot, []byte(site), []byte(user), []byte(path)}, bs)
	})
}

func DeleteRepoInfo(site, user, path string) error {
	return box.Update(func(tx bh.Tx) error {
		return tx.Delete([][]byte{repoRoot, []byte(site), []byte(user), []byte(path)})
	})
}

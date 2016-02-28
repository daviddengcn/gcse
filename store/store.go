package store

import (
	"log"
	"time"

	"github.com/golangplus/bytes"
	"github.com/golangplus/errors"

	"github.com/daviddengcn/bolthelper"
	"github.com/daviddengcn/gcse/configs"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"

	stpb "github.com/daviddengcn/gcse/proto/store"
	tspb "github.com/golang/protobuf/ptypes/timestamp"
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

func RepoInfoAge(r *stpb.RepoInfo) time.Duration {
	t, _ := ptypes.Timestamp(r.LastCrawled)
	return time.Now().Sub(t)
}

func TimestampProto(t time.Time) *tspb.Timestamp {
	ts, _ := ptypes.TimestampProto(t)
	return ts
}

// return nil without error if cache not found.
func FetchRepoInfo(site, user, path string) (*stpb.RepoInfo, error) {
	var r *stpb.RepoInfo
	if err := box.View(func(tx bh.Tx) error {
		return tx.Value([][]byte{repoRoot, []byte(site), []byte(user), []byte(path)}, func(v bytesp.Slice) error {
			r = &stpb.RepoInfo{}
			return errorsp.WithStacksAndMessage(proto.Unmarshal(v, r), "len = %d", len(v))
		})
	}); err != nil {
		return nil, err
	}
	return r, nil
}

func ForEachReposInSite(site string, f func(user, path string, info *stpb.RepoInfo) error) error {
	return box.View(func(tx bh.Tx) error {
		return tx.ForEach([][]byte{repoRoot, []byte(site)}, func(b bh.Bucket, user, v bytesp.Slice) error {
			if v != nil {
				return nil
			}
			return b.ForEach([][]byte{user}, func(path bytesp.Slice, v bytesp.Slice) error {
				r := &stpb.RepoInfo{}
				if err := errorsp.WithStacksAndMessage(proto.Unmarshal(v, r), "len = %d", len(v)); err != nil {
					log.Printf("Unmarshal RepoInfo for %s failed: %v, ignored", string(path), err)
					return nil
				}
				return f(string(user), string(path), r)
			})
		})
	})
}

func SaveRepoInfo(site, user, path string, r *stpb.RepoInfo) error {
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

package store

import (
	"log"

	"github.com/golang/protobuf/proto"
	"github.com/golangplus/bytes"
	"github.com/golangplus/errors"

	"github.com/daviddengcn/bolthelper"
	"github.com/daviddengcn/gcse/proto/store"
)

// Returns an empty (non-nil) PackageInfo if not found.
func ReadRepository(site, user, repo string) (*stpb.Repository, error) {
	doc := &stpb.Repository{}
	if err := box.View(func(tx bh.Tx) error {
		return tx.Value([][]byte{reposRoot, []byte(site), []byte(user), []byte(repo)}, func(bs bytesp.Slice) error {
			if err := errorsp.WithStacksAndMessage(proto.Unmarshal(bs, doc), "Unmarshal %d bytes failed", len(bs)); err != nil {
				log.Printf("Unmarshal failed: %v", err)
				*doc = stpb.Repository{}
			}
			return nil
		})
	}); err != nil {
		return nil, err
	}
	return doc, nil
}

func UpdateRepository(site, user, repo string, f func(doc *stpb.Repository) error) error {
	return box.Update(func(tx bh.Tx) error {
		b, err := tx.CreateBucketIfNotExists([][]byte{reposRoot, []byte(site), []byte(user)})
		if err != nil {
			return err
		}
		doc := &stpb.Repository{}
		if err := b.Value([][]byte{[]byte(repo)}, func(bs bytesp.Slice) error {
			if err := errorsp.WithStacksAndMessage(proto.Unmarshal(bs, doc), "Unmarshal %d bytes", len(bs)); err != nil {
				log.Printf("Unmarshaling failed: %v", err)
				*doc = stpb.Repository{}
			}
			return nil
		}); err != nil {
			return err
		}
		if err := errorsp.WithStacks(f(doc)); err != nil {
			return err
		}
		bs, err := proto.Marshal(doc)
		if err != nil {
			return errorsp.WithStacksAndMessage(err, "marshaling %v failed: %v", doc, err)
		}
		return b.Put([][]byte{[]byte(repo)}, bs)
	})
}

func DeleteRepository(site, user, repo string) error {
	return box.Update(func(tx bh.Tx) error {
		return tx.Delete([][]byte{reposRoot, []byte(site), []byte(user), []byte(repo)})
	})
}

func ForEachRepositorySite(f func(string) error) error {
	return box.View(func(tx bh.Tx) error {
		return tx.ForEach([][]byte{reposRoot}, func(_ bh.Bucket, k, v bytesp.Slice) error {
			if v != nil {
				log.Printf("Unexpected value %q for key %q, ignored", string(v), string(k))
				return nil
			}
			return errorsp.WithStacks(f(string(k)))
		})
	})
}

func ForEachRepositoryOfSite(site string, f func(user, name string, doc *stpb.Repository) error) error {
	return box.View(func(tx bh.Tx) error {
		return tx.ForEach([][]byte{reposRoot, []byte(site)}, func(b bh.Bucket, user, v bytesp.Slice) error {
			if v != nil {
				log.Printf("Unexpected value %q for key %q, ignored", string(v), string(user))
				return nil
			}
			return b.ForEach([][]byte{user}, func(name, bs bytesp.Slice) error {
				if bs == nil {
					log.Printf("Unexpected nil value for key %q, ignored", string(name))
					return nil
				}
				doc := &stpb.Repository{}
				if err := errorsp.WithStacksAndMessage(proto.Unmarshal(bs, doc), "Unmarshal %d bytes", len(bs)); err != nil {
					log.Printf("Unmarshaling value for %v failed, ignored: %v", name, err)
					return nil
				}
				return errorsp.WithStacks(f(string(user), string(name), doc))
			})
		})
	})
}

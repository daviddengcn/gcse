package spider

import (
	"errors"
	"log"
	"strings"

	"github.com/golangplus/bytes"
	"github.com/golangplus/errors"
	"github.com/golangplus/strings"

	"github.com/daviddengcn/bolthelper"
	"github.com/golang/protobuf/proto"
)

type FileCache interface {
	Get(signature string, contents proto.Message) bool
	Set(signature string, contents proto.Message)
	// nameToSignature: folder map to ""
	SetFolderSignatures(folder string, nameToSignature map[string]string)
}

type NullFileCache struct{}

func (NullFileCache) Get(string, proto.Message) bool                { return false }
func (NullFileCache) Set(string, proto.Message)                     {}
func (NullFileCache) SetFolderSignatures(string, map[string]string) {}

var _ FileCache = NullFileCache{}

type BoltFileCache struct {
	bh.DB
	IncCounter func(string)
}

var _ FileCache = BoltFileCache{}

// Filecache folders:
// s/<path>             - signature of this path
// c/<signature>        - contents of a signagure
// p/<signature>/<path> - list of paths referencing this signature

var (
	cacheSignatureKey = []byte("s")
	cacheContentsKey  = []byte("c")
	cachePathsKey     = []byte("p")
)

func (bc BoltFileCache) inc(name string) {
	if bc.IncCounter == nil {
		return
	}
	bc.IncCounter(name)
}

func (bc BoltFileCache) Get(sign string, contents proto.Message) bool {
	found := false
	if err := bc.View(func(tx bh.Tx) error {
		return tx.Value([][]byte{cacheContentsKey, []byte(sign)}, func(v bytesp.Slice) error {
			found = true
			return errorsp.WithStacks(proto.Unmarshal(v, contents))
		})
	}); err != nil {
		log.Printf("Reading from file cache DB for %v failed: %v", sign, err)
		bc.inc("crawler.filecache.get_error")
		return false
	}
	if found {
		bc.inc("crawler.filecache.hit")
	} else {
		bc.inc("crawler.filecache.missed")
	}
	return found
}

func (bc BoltFileCache) Set(signature string, contents proto.Message) {
	if err := bc.Update(func(tx bh.Tx) error {
		bs, err := proto.Marshal(contents)
		if err != nil {
			return errorsp.WithStacksAndMessage(err, "Marshal %v failed", contents)
		}
		return tx.Put([][]byte{cacheContentsKey, []byte(signature)}, bs)
	}); err != nil {
		bc.inc("crawler.filecache.set_error")
		log.Printf("Updating to file cache DB for %v failed: %v", signature, err)
	}
	bc.inc("crawler.filecache.sign_saved")
}

var errStop = errors.New("stop")

func (bc BoltFileCache) SetFolderSignatures(folder string, nameToSignature map[string]string) {
	if !strings.HasSuffix(folder, "/") {
		folder += "/"
	}
	if err := bc.Update(func(tx bh.Tx) error {
		// sub path -> current signature
		toDelete := make(map[string]string)
		// sub path -> current signature
		toUpdate := make(map[string]string)
		var unchanged stringsp.Set

		if err := tx.Cursor([][]byte{cacheSignatureKey}, func(c bh.Cursor) error {
			for k, v := c.Seek([]byte(folder)); strings.HasPrefix(string(k), folder); k, v = c.Next() {
				sub := string(k[len(folder):])
				ps := strings.SplitN(sub, "/", 2)
				if len(ps) > 1 {
					// k is a file under a sub folder.
					if s, ok := nameToSignature[ps[0]]; !ok || s != "" {
						// if the sub folder no longer exist or is not a folder (i.e. is a
						// file with signature), delete the current file
						toDelete[sub] = string(v)
					}
				} else if len(ps) == 1 {
					newS := nameToSignature[sub]
					if newS == "" {
						// no longer a file
						toDelete[sub] = string(v)
					} else if newS != string(v) {
						bc.inc("crawler.filecache.file_changed")
						// signature changed
						toUpdate[sub] = string(v)
					} else {
						unchanged.Add(sub)
					}
				}
			}
			return nil
		}); err != nil {
			return err
		}
		log.Printf("toDelete: %v", toDelete)
		log.Printf("toUpdate: %v", toUpdate)
		log.Printf("unchanged: %v", unchanged)
		// Add new files into toUpdate
		for name, signature := range nameToSignature {
			if signature == "" {
				// folders
				continue
			}
			if _, ok := toUpdate[name]; ok {
				continue
			}
			if unchanged.Contain(name) {
				continue
			}
			bc.inc("crawler.filecache.file_added")
			toUpdate[name] = ""
		}
		deleteReferenceToSignatureFromPath := func(signature, path string) error {
			if err := tx.Delete([][]byte{cachePathsKey, []byte(signature), []byte(path)}); err != nil {
				return err
			}
			// Check whether the signature is still referenced by any path.
			hasKeys := false
			if err := tx.ForEach([][]byte{cachePathsKey, []byte(signature)}, func(bh.Bucket, bytesp.Slice, bytesp.Slice) error {
				hasKeys = true
				return errStop
			}); err != nil && err != errStop {
				return err
			}
			if !hasKeys {
				// all references to the signature have been deleted, delete the contents of the signature as well
				if err := tx.Delete([][]byte{cacheContentsKey, []byte(signature)}); err != nil {
					return err
				}
				bc.inc("crawler.filecache.sign_deleted")
			}
			return nil
		}
		for sub, signature := range toDelete {
			path := folder + sub
			if err := deleteReferenceToSignatureFromPath(signature, path); err != nil {
				log.Printf("deleteReferenceToSignatureFromPath failed: %v", err)
				return nil
			}
			bc.inc("crawler.filecache.file_deleted")
			// Delete the signature item of the path
			if err := tx.Delete([][]byte{cacheSignatureKey, []byte(path)}); err != nil {
				return err
			}
		}
		for sub, oldS := range toUpdate {
			path := folder + sub
			if oldS != "" {
				if err := deleteReferenceToSignatureFromPath(oldS, path); err != nil {
					return nil
				}
			}
			// Update the signature item of the path
			newS := nameToSignature[sub]
			if err := tx.Put([][]byte{cacheSignatureKey, []byte(path)}, []byte(newS)); err != nil {
				return err
			}
			// Add reference to new signature from path
			if err := tx.Put([][]byte{cachePathsKey, []byte(newS), []byte(path)}, []byte(newS)); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		log.Printf("SetFolderSignatures folder %v failed: %v", folder, err)
		bc.inc("crawler.filecache.sign_error")
	}
}

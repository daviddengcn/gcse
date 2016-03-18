package spider

import (
	"log"

	"github.com/golangplus/bytes"
	"github.com/golangplus/errors"

	"github.com/daviddengcn/bolthelper"
	"github.com/golang/protobuf/proto"
)

type FileCache interface {
	Get(signature string, contents proto.Message) bool
	Set(signature string, contents proto.Message)
}

type NullFileCache struct{}

func (NullFileCache) Get(string, proto.Message) bool { return false }
func (NullFileCache) Set(string, proto.Message)      {}

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

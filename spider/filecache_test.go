package spider

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/golangplus/testing/assert"

	"github.com/daviddengcn/bolthelper"

	sppb "github.com/daviddengcn/gcse/proto/spider"
)

func TestNullFileCache(t *testing.T) {
	c := NullFileCache{}
	c.Set("", nil)
	assert.False(t, "c.Get", c.Get("", nil))
}

func TestBoltFileCache(t *testing.T) {
	fn := filepath.Join(os.TempDir(), "TestBoltFileCache.bolt")
	assert.NoErrorOrDie(t, os.RemoveAll(fn))

	db, err := bh.Open(fn, 0755, nil)
	assert.NoErrorOrDie(t, err)

	counter := make(map[string]int)
	c := BoltFileCache{
		DB: db,
		IncCounter: func(name string) {
			counter[name] = counter[name] + 1
		},
	}
	const (
		sign1      = "abc"
		sign2      = "def"
		sign3      = "ghi"
		gofile     = "file.go"
		rootfolder = "root"
		sub        = "sub"
		subfolder  = "root/sub"
	)
	fi := &sppb.GoFileInfo{}

	//////////////////////////////////////////////////////////////
	// New file found.
	//////////////////////////////////////////////////////////////
	// Get before set, should return false
	assert.False(t, "c.Get", c.Get(sign1, fi))
	assert.Equal(t, "counter", counter, map[string]int{
		"crawler.filecache.missed": 1,
	})
	// Set the info.
	c.Set(sign1, &sppb.GoFileInfo{Status: sppb.GoFileInfo_ShouldIgnore})
	assert.Equal(t, "counter", counter, map[string]int{
		"crawler.filecache.missed":     1,
		"crawler.filecache.sign_saved": 1,
	})
	// Now, should fetch the cache
	assert.True(t, "c.Get", c.Get(sign1, fi))
	assert.Equal(t, "fi", fi, &sppb.GoFileInfo{Status: sppb.GoFileInfo_ShouldIgnore})
	assert.Equal(t, "counter", counter, map[string]int{
		"crawler.filecache.missed":     1,
		"crawler.filecache.sign_saved": 1,
		"crawler.filecache.hit":        1,
	})
}

package spider

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/golangplus/testing/assert"

	sppb "github.com/daviddengcn/gcse/proto/spider"

	"github.com/daviddengcn/bolthelper"
)

func TestNullFileCache(t *testing.T) {
	c := NullFileCache{}
	c.Set("", nil)
	assert.False(t, "c.Get", c.Get("", nil))
	c.SetFolderSignatures("", nil)
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
	// SetFolderSignatures
	c.SetFolderSignatures(rootfolder, map[string]string{
		gofile: sign1,
	})
	assert.Equal(t, "counter", counter, map[string]int{
		"crawler.filecache.missed":     1,
		"crawler.filecache.sign_saved": 1,
		"crawler.filecache.hit":        1,
		"crawler.filecache.file_added": 1,
	})
	// Should still fetch the cache
	assert.True(t, "c.Get", c.Get(sign1, fi))
	assert.Equal(t, "counter", counter, map[string]int{
		"crawler.filecache.missed":     1,
		"crawler.filecache.sign_saved": 1,
		"crawler.filecache.hit":        2,
		"crawler.filecache.file_added": 1,
	})

	//////////////////////////////////////////////////////////////
	// File changed.
	//////////////////////////////////////////////////////////////
	// Should not fetch with new sign
	assert.False(t, "c.Get", c.Get(sign3, fi))
	assert.Equal(t, "counter", counter, map[string]int{
		"crawler.filecache.missed":     2,
		"crawler.filecache.sign_saved": 1,
		"crawler.filecache.hit":        2,
		"crawler.filecache.file_added": 1,
	})
	// Set the new info.
	c.Set(sign3, &sppb.GoFileInfo{Status: sppb.GoFileInfo_ParseSuccess})
	assert.Equal(t, "counter", counter, map[string]int{
		"crawler.filecache.missed":     2,
		"crawler.filecache.sign_saved": 2,
		"crawler.filecache.hit":        2,
		"crawler.filecache.file_added": 1,
	})
	// SetFolderSignatures, should add sign3, delete sign1
	c.SetFolderSignatures(rootfolder, map[string]string{
		gofile: sign3,
	})
	assert.Equal(t, "counter", counter, map[string]int{
		"crawler.filecache.missed":       2,
		"crawler.filecache.sign_saved":   2,
		"crawler.filecache.hit":          2,
		"crawler.filecache.file_added":   1,
		"crawler.filecache.file_changed": 1,
		"crawler.filecache.sign_deleted": 1,
	})
	// Should still fetch the new sign
	assert.True(t, "c.Get", c.Get(sign3, fi))
	assert.Equal(t, "counter", counter, map[string]int{
		"crawler.filecache.missed":       2,
		"crawler.filecache.sign_saved":   2,
		"crawler.filecache.hit":          3,
		"crawler.filecache.file_added":   1,
		"crawler.filecache.file_changed": 1,
		"crawler.filecache.sign_deleted": 1,
	})
	// Should not fetch the old sign
	assert.False(t, "c.Get", c.Get(sign1, fi))
	assert.Equal(t, "counter", counter, map[string]int{
		"crawler.filecache.missed":       3,
		"crawler.filecache.sign_saved":   2,
		"crawler.filecache.hit":          3,
		"crawler.filecache.file_added":   1,
		"crawler.filecache.file_changed": 1,
		"crawler.filecache.sign_deleted": 1,
	})
	//////////////////////////////////////////////////////////////
	// File deleted.
	//////////////////////////////////////////////////////////////
	// SetFolderSignatures again after the file was deleted
	c.SetFolderSignatures(rootfolder, map[string]string{})
	assert.Equal(t, "counter", counter, map[string]int{
		"crawler.filecache.missed":       3,
		"crawler.filecache.sign_saved":   2,
		"crawler.filecache.hit":          3,
		"crawler.filecache.file_added":   1,
		"crawler.filecache.file_deleted": 1,
		"crawler.filecache.file_changed": 1,
		"crawler.filecache.sign_deleted": 2,
	})
	// Should not fetch because the cache has been deleted
	assert.False(t, "c.Get", c.Get(sign3, fi))
	assert.Equal(t, "counter", counter, map[string]int{
		"crawler.filecache.missed":       4,
		"crawler.filecache.sign_saved":   2,
		"crawler.filecache.hit":          3,
		"crawler.filecache.file_added":   1,
		"crawler.filecache.file_deleted": 1,
		"crawler.filecache.file_changed": 1,
		"crawler.filecache.sign_deleted": 2,
	})
	//////////////////////////////////////////////////////////////
	// Update the root folder.
	//////////////////////////////////////////////////////////////
	// Set the info again
	c.Set(sign2, &sppb.GoFileInfo{Status: sppb.GoFileInfo_ParseFailed})
	assert.Equal(t, "counter", counter, map[string]int{
		"crawler.filecache.missed":       4,
		"crawler.filecache.sign_saved":   3,
		"crawler.filecache.hit":          3,
		"crawler.filecache.file_added":   1,
		"crawler.filecache.file_deleted": 1,
		"crawler.filecache.file_changed": 1,
		"crawler.filecache.sign_deleted": 2,
	})
	// SetFolderSignatures as a sub folder.
	c.SetFolderSignatures(subfolder, map[string]string{
		gofile: sign2,
	})
	assert.Equal(t, "counter", counter, map[string]int{
		"crawler.filecache.missed":       4,
		"crawler.filecache.sign_saved":   3,
		"crawler.filecache.hit":          3,
		"crawler.filecache.file_added":   2,
		"crawler.filecache.file_deleted": 1,
		"crawler.filecache.file_changed": 1,
		"crawler.filecache.sign_deleted": 2,
	})
	// Should still fetch the cache
	assert.True(t, "c.Get", c.Get(sign2, fi))
	assert.Equal(t, "counter", counter, map[string]int{
		"crawler.filecache.missed":       4,
		"crawler.filecache.sign_saved":   3,
		"crawler.filecache.hit":          4,
		"crawler.filecache.file_added":   2,
		"crawler.filecache.file_deleted": 1,
		"crawler.filecache.file_changed": 1,
		"crawler.filecache.sign_deleted": 2,
	})
	// SetFolderSignatures to the root, sub should remain unchanged
	c.SetFolderSignatures(rootfolder, map[string]string{
		sub: "",
	})
	assert.Equal(t, "counter", counter, map[string]int{
		"crawler.filecache.missed":       4,
		"crawler.filecache.sign_saved":   3,
		"crawler.filecache.hit":          4,
		"crawler.filecache.file_added":   2,
		"crawler.filecache.file_deleted": 1,
		"crawler.filecache.file_changed": 1,
		"crawler.filecache.sign_deleted": 2,
	})
	// SetFolderSignatures to clean the root, sub should be cleared as well.
	c.SetFolderSignatures(rootfolder, map[string]string{})
	assert.Equal(t, "counter", counter, map[string]int{
		"crawler.filecache.missed":       4,
		"crawler.filecache.sign_saved":   3,
		"crawler.filecache.hit":          4,
		"crawler.filecache.file_added":   2,
		"crawler.filecache.file_deleted": 2,
		"crawler.filecache.file_changed": 1,
		"crawler.filecache.sign_deleted": 3,
	})
	// Should not fetch because the cache has been deleted
	assert.False(t, "c.Get", c.Get(sign1, fi))
	assert.Equal(t, "fi", fi, &sppb.GoFileInfo{Status: sppb.GoFileInfo_ParseFailed})
	assert.Equal(t, "counter", counter, map[string]int{
		"crawler.filecache.missed":       5,
		"crawler.filecache.sign_saved":   3,
		"crawler.filecache.hit":          4,
		"crawler.filecache.file_added":   2,
		"crawler.filecache.file_deleted": 2,
		"crawler.filecache.file_changed": 1,
		"crawler.filecache.sign_deleted": 3,
	})
}

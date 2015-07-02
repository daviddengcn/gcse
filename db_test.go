package gcse

import (
	"testing"

	"github.com/golangplus/testing/assert"

	"github.com/daviddengcn/go-villa"
)

func TestMemDB_Bug_Sync(t *testing.T) {
	path := villa.Path(".").Join("testmemdb.gob")
	if path.Exists() {
		path.Remove()
	}

	db := NewMemDB(".", "testmemdb")
	db.Put("s", 1)
	err := db.Sync()
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, "Exists", path.Exists(), true)
	if err := path.Remove(); err != nil {
		t.Error(err)
	}
	assert.Equal(t, "Exists", path.Exists(), false)

	//if err := db.Load(); err != nil {
	//	t.Error(err)
	//}
}

func TestMemDB_Recover(t *testing.T) {
	path := villa.Path(".").Join("testmemdb.gob")
	if path.Exists() {
		path.Remove()
	}

	db := NewMemDB(".", "testmemdb")
	db.Put("s", 1)
	if err := db.Sync(); err != nil {
		t.Error(err)
		return
	}

	if err := path.Rename(path + ".new"); err != nil {
		t.Error(err)
		return
	}
	// Now in the status of fn.new exists, fn not exist

	if err := db.Load(); err != nil {
		t.Error(err)
		return
	}
	var vl int
	if ok := db.Get("s", &vl); !ok {
		t.Error("Recover failed!")
		return
	}
	assert.Equal(t, "vl", vl, 1)
}

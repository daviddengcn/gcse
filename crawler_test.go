package gcse

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/daviddengcn/gddo/doc"
	"github.com/daviddengcn/go-assert"
	"github.com/daviddengcn/go-villa"
)

func TestGithubUpdates(t *testing.T) {
	_, err := GithubUpdates()
	if err != nil {
		t.Error(err)
	}
	//log.Printf("Updates: %v", updates)
}

func TestReadmeToText(t *testing.T) {
	text := strings.TrimSpace(ReadmeToText("a.md", "#abc"))
	assert.Equals(t, "text", text, "abc")
}

func TestReadmeToText_Panic(t *testing.T) {
	ReadmeToText("a.md", "* [[t]](/t)")
}

func TestPlusone(t *testing.T) {
	url := "http://www.google.com/"
	cnt, err := Plusone(http.DefaultClient, url)
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("Plusone of %s: %d", url, cnt)
	if cnt <= 0 {
		t.Errorf("Zero Plusone count for %s", url)
	}
}

func TestLikeButton(t *testing.T) {
	url := "http://www.google.com/"
	cnt, err := LikeButton(http.DefaultClient, url)
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("LikeButton of %s: %d", url, cnt)
	if cnt <= 0 {
		t.Errorf("Zero LikeButton count for %s", url)
	}
}

func TestGddo(t *testing.T) {
	doc.SetGithubCredentials("94446b37edb575accd8b",
		"15f55815f0515a3f6ad057aaffa9ea83dceb220b")
	doc.SetUserAgent("Go-Code-Search-Agent")

	pkg := "github.com/daviddengcn/gcse"
	p, err := CrawlPackage(http.DefaultClient, pkg, "")
	if err != nil {
		t.Error(err)
	} else {
		t.Logf("p: %+v", p.Exported)
	}
	//	t.Error(nil)
}

func TestDocDB(t *testing.T) {
	var db DocDB = PackedDocDB{NewMemDB("", "")}

	info := DocInfo{
		Name: "github.com/daviddengcn/gcse",
	}
	db.Put("hello", info)
	var info2 DocInfo
	if ok := db.Get("hello", &info2); !ok {
		t.Error("db.Get failed!")
		return
	}
	assert.StringEquals(t, "hello", info2, info)

	if err := db.Iterate(func(key string, val interface{}) error {
		info3, ok := val.(DocInfo)
		if !ok {
			return errors.New("errNotDocInfo")
		}

		assert.StringEquals(t, key, info3, info)
		return nil
	}); err != nil {
		t.Errorf("db.Iterate failed: %v", err)
	}
}

func TestDocDB_Export(t *testing.T) {
	var db DocDB = PackedDocDB{NewMemDB("", "")}

	info := DocInfo{
		Name: "github.com/daviddengcn/gcse",
	}

	db.Put("go", info)

	if err := db.Export(villa.Path("."), "testexport_db"); err != nil {
		t.Errorf("db.Export failed: %v", err)
		return
	}

	var newDB DocDB = PackedDocDB{NewMemDB(villa.Path("."), "testexport_db")}
	count := 0
	if err := newDB.Iterate(func(key string, val interface{}) error {
		info, ok := val.(DocInfo)
		if !ok {
			return errors.New("Not a DocInfo object")
		}
		assert.StringEquals(t, "info.Name", info.Name,
			"github.com/daviddengcn/gcse")
		count++
		return nil
	}); err != nil {
		t.Errorf("newDB.Iterate failed: %v", err)
	}

	assert.Equals(t, "count", count, 1)
}

func TestCrawlingEntry(t *testing.T) {
	src := CrawlingEntry {
		ScheduleTime: time.Now(),
		Version:      19,
		Etag:         "Hello",
	}
	
	var buf villa.ByteSlice
	assert.NoErrorf(t, "src.WriteTo failed: %v", src.WriteTo(&buf))
	
	var dst CrawlingEntry
	assert.NoErrorf(t, "dst.ReadFrom failed: %v", dst.ReadFrom(&buf, -1))
	
	assert.StringEquals(t, "dst", dst, src)
}

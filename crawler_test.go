package gcse

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golangplus/bytes"
	"github.com/golangplus/testing/assert"

	"github.com/daviddengcn/gcse/configs"
	"github.com/daviddengcn/gddo/doc"
	"github.com/daviddengcn/go-villa"
)

func TestReadmeToText(t *testing.T) {
	text := strings.TrimSpace(ReadmeToText("a.md", "#abc"))
	assert.Equal(t, "text", text, "abc")
}

func TestReadmeToText_Panic(t *testing.T) {
	ReadmeToText("a.md", "* [[t]](/t)")
}

func TestPlusone(t *testing.T) {
	url := "http://www.google.com/"
	cnt, err := Plusone(http.DefaultClient, url)
	assert.NoError(t, err)
	t.Logf("Plusone of %s: %d", url, cnt)
	if cnt <= 0 {
		t.Errorf("Zero Plusone count for %s", url)
	}
}

func TestLikeButton(t *testing.T) {
	url := "http://www.facebook.com/"
	cnt, err := LikeButton(http.DefaultClient, url)
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("LikeButton of %s: %d", url, cnt)
	if cnt <= 0 {
		//		t.Errorf("Zero LikeButton count for %s", url)
	}
}

func TestCrawlPackage(t *testing.T) {
	if configs.CrawlerGithubClientID != "" {
		t.Logf("Github clientid: %s", configs.CrawlerGithubClientID)
		t.Logf("Github clientsecret: %s", configs.CrawlerGithubClientSecret)
		doc.SetGithubCredentials(configs.CrawlerGithubClientID, configs.CrawlerGithubClientSecret)
	}

	pkg := "github.com/daviddengcn/gcse"
	httpClient := GenHttpClient("")
	p, _, err := CrawlPackage(httpClient, pkg, "")
	if err != nil {
		if strings.Index(err.Error(), "403") == -1 {
			t.Error(err)
		}
	} else {
		assert.Equal(t, "pkg", p.Package, pkg)
	}

	//	pkg = "git.gitorious.org/go-pkg/epubgo.git"
	//	p, err = CrawlPackage(httpClient, pkg, "")
	//	if err != nil {
	//		if strings.Index(err.Error(), "403") == -1 {
	//			t.Error(err)
	//		}
	//	} else {
	//		assert.Equal(t, "pkg", p.Package, pkg)
	//	}

	pkg = "thezombie.net/libgojira"
	p, _, err = CrawlPackage(httpClient, pkg, "")
	if err != nil {
		if !IsBadPackage(err) {
			t.Errorf("%s should be an invalid package", pkg)
		}
	} else {
		t.Errorf("%s should be an invalid package", pkg)
	}
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
	assert.StringEqual(t, "hello", info2, info)

	if err := db.Iterate(func(key string, val interface{}) error {
		info3, ok := val.(DocInfo)
		if !ok {
			return errors.New("errNotDocInfo")
		}

		assert.StringEqual(t, key, info3, info)
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
		assert.StringEqual(t, "info.Name", info.Name,
			"github.com/daviddengcn/gcse")
		count++
		return nil
	}); err != nil {
		t.Errorf("newDB.Iterate failed: %v", err)
	}

	assert.Equal(t, "count", count, 1)
}

func TestCrawlingEntry(t *testing.T) {
	src := CrawlingEntry{
		ScheduleTime: time.Now(),
		Version:      19,
		Etag:         "Hello",
	}

	var buf bytesp.Slice
	assert.NoError(t, src.WriteTo(&buf))

	var dst CrawlingEntry
	assert.NoError(t, dst.ReadFrom(&buf, -1))

	assert.StringEqual(t, "dst", dst, src)
}

func TestFullProjectOfPackage(t *testing.T) {
	DATA := []string{
		"github.com/daviddengcn/gcse", "github.com/daviddengcn/gcse",
		"github.com/daviddengcn/gcse/index", "github.com/daviddengcn/gcse",
		"code.google.com/p/go.net/websocket", "code.google.com/p/go.net",
	}

	for i := 0; i < len(DATA); i += 2 {
		pkg, prj := DATA[i], DATA[i+1]
		assert.Equal(t, "FullProjectOfPackage "+pkg, FullProjectOfPackage(pkg), prj)
	}
}

package gcse

import (
	"github.com/daviddengcn/go-villa"
	"net/http"
	"strings"
	"testing"
)

func TestGithubUpdates(t *testing.T) {
	_, err := GithubUpdates()
	if err != nil {
		t.Error(err)
	}
	// log.Printf("Updates: %v", updates)
}

func TestReadmeToText(t *testing.T) {
	text := strings.TrimSpace(ReadmeToText("a.md", "#abc"))
	villa.AssertEquals(t, "text", text, "abc")
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
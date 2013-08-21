package gcse

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	
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
	AssertEquals(t, "text", text, "abc")
}

func _TestPlusone(t *testing.T) {
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

func _TestLikeButton(t *testing.T) {
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

/*
	AssertEquals shows error message when act and exp don't equal
*/
func AssertEquals(t *testing.T, name string, act, exp interface{}) {
	if act != exp {
		t.Errorf("%s is expected to be %v, but got %v", name, exp, act)
	}
}

/*
	AssertEquals shows error message when strings forms of act and exp don't
	equal
*/
func AssertStringEquals(t *testing.T, name string, act, exp interface{}) {
	if fmt.Sprintf("%v", act) != fmt.Sprintf("%v", exp) {
		t.Errorf("%s is expected to be %v, but got %v", name, exp, act)
	} // if
}

/*
	AssertStrSetEquals shows error message when act and exp are equal string
	sets.
*/
func AssertStrSetEquals(t *testing.T, name string, act, exp villa.StrSet) {
	if !act.Equals(exp) {
		t.Errorf("%s is expected to be %v, but got %v", name, exp, act)
	}
}

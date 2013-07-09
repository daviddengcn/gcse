package gcse

import (
	"github.com/daviddengcn/go-villa"
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

package utils

import (
	"testing"

	"github.com/golangplus/testing/assert"
)

func TestSplitPackage(t *testing.T) {
	for _, c := range []struct {
		pkg  string
		site string
		path string
	}{
		{"github.com/daviddengcn", "github.com", "daviddengcn"},
		{"github.com", "github.com", ""},
		{"", "", ""},
	} {
		site, path := SplitPackage(c.pkg)
		assert.Equal(t, "site", site, c.site)
		assert.Equal(t, "path", path, c.path)
	}
}

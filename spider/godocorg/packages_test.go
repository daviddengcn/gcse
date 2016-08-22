package godocorg

import (
	"net/http"
	"testing"

	"github.com/golangplus/testing/assert"
)

func TestFetchAllPackagesInGodoc(t *testing.T) {
	pkgs, err := FetchAllPackagesInGodoc(http.DefaultClient)
	assert.NoError(t, err)

	if len(pkgs) == 0 {
		t.Errorf("No packages returned!")
	}
}

package gcse

import (
	"os"
	"testing"

	"github.com/golangplus/testing/assert"

	"github.com/daviddengcn/go-villa"
)

func setTestingDataPath() {
	DataRoot = villa.Path(os.TempDir()).Join("gcse_testing")
	DataRoot.MkdirAll(0755)
}

func TestStoreRepoInfo(t *testing.T) {
	assert.NoError(t, SaveRepoInfo("example.com", "anonymous", "fake", RepoInfo{Stars: 123, Description: "hello"}))
	r, err := FetchRepoInfo("example.com", "anonymous", "fake")
	assert.NoError(t, err)
	assert.Equal(t, "r", *r, RepoInfo{Stars: 123, Description: "hello"})
}

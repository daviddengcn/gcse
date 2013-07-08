package gcse

import(
	"testing"
)

func TestGithubUpdates(t *testing.T) {
	_, err := GithubUpdates()
	if err != nil {
		t.Error(err)
	}
	// log.Printf("Updates: %v", updates)
}
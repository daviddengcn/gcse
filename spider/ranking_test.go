package spider

import (
	"fmt"
	"testing"

	"github.com/golangplus/testing/assert"
)

func TestLikeGoSubFolder(t *testing.T) {
	pos_cases := []string{
		"go", "v8", "v-8",
	}
	for _, c := range pos_cases {
		assert.True(t, fmt.Sprintf("LikeGoSubFolder %v", c), LikeGoSubFolder(c))
	}

	neg_cases := []string{
		"js", "1234", "1234-5678", "1234_5678",
	}
	for _, c := range neg_cases {
		assert.False(t, fmt.Sprintf("LikeGoSubFolder %v", c), LikeGoSubFolder(c))
	}
}

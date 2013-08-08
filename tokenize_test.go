package gcse

import (
	"github.com/daviddengcn/go-villa"
	"testing"
)

func TestTokenize(t *testing.T) {
	text := []byte("abc 3d 中文输入")
	tokens := AppendTokens(nil, text)
	AssertStrSetEquals(t, "tokens", tokens,
		villa.NewStrSet("abc", "3", "d", "3-d", "中", "文", "输", "入", "中文",
			"文输", "输入"))
}

func TestTokenize2(t *testing.T) {
	text := []byte("PubSubHub")
	tokens := AppendTokens(nil, text)
	AssertStrSetEquals(t, "tokens", tokens,
		villa.NewStrSet("pub", "sub", "hub", "pubsub", "subhub",
			"pubsubhub"))
}

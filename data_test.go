package gcse

import (
	"testing"
	"time"

	"github.com/daviddengcn/go-assert"
	"github.com/daviddengcn/go-index"
	"github.com/daviddengcn/go-villa"
)

func TestDocInfo(t *testing.T) {
	src := DocInfo{
		Name:        "gcse",
		Package:     "github.com/daviddengcn/gcse",
		Author:      "github.com/daviddengcn",
		LastUpdated: time.Now(),
		StarCount:   10,
		Synopsis:    "Go Package Search Engine",
		Description: "More details about GCSE",
		ProjectURL:  "http://github.com/daviddengcn/gcse",
		ReadmeFn:    "readme.txt",
		ReadmeData:  "Just read me",
		Imports: []string{
			"github.com/daviddengcn/go-villa",
			"github.com/daviddengcn/sophie",
		},
		TestImports: []string{
			"github.com/daviddengcn/go-check",
		},
		Exported: []string{
			"DocInfo", "CheckRuneType",
		},
	}
	var buf villa.ByteSlice
	assert.NoErrorf(t, "src.WriteTo failed: %v", src.WriteTo(&buf))

	var dst DocInfo
	assert.NoErrorf(t, "dst.ReadFrom failed: %v", dst.ReadFrom(&buf, -1))

	assert.StringEquals(t, "dst", dst, src)

	// checking the bug introduced by reusing slice
	dst2 := dst
	assert.StringEquals(t, "dst2.Imports[0]", dst2.Imports[0],
		"github.com/daviddengcn/go-villa")

	src.Imports[0] = "github.com/daviddengcn/go-assert"
	buf = nil
	assert.NoErrorf(t, "src.WriteTo failed: %v", src.WriteTo(&buf))
	assert.NoErrorf(t, "dst.ReadFrom failed: %v", dst.ReadFrom(&buf, -1))
	assert.StringEquals(t, "dst", dst, src)

	assert.StringEquals(t, "dst2.Imports[0]", dst2.Imports[0],
		"github.com/daviddengcn/go-villa")
}


func TestCheckRuneType_BOM(t *testing.T) {
	tp := CheckRuneType('A', 0xfeff)
	assert.Equals(t, "CheckRuneType(0, 0xfeff)", tp, index.TokenSep)
}

package gcse

import (
	"testing"
	"time"

	"github.com/golangplus/bytes"
	"github.com/golangplus/testing/assert"

	"github.com/daviddengcn/go-index"
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
	var buf bytesp.Slice
	assert.NoError(t, src.WriteTo(&buf))

	var dst DocInfo
	assert.NoError(t, dst.ReadFrom(&buf, -1))

	assert.StringEqual(t, "dst", dst, src)

	// checking the bug introduced by reusing slice
	dst2 := dst
	assert.StringEqual(t, "dst2.Imports[0]", dst2.Imports[0],
		"github.com/daviddengcn/go-villa")

	src.Imports[0] = "github.com/daviddengcn/go-assert"
	buf = nil
	assert.NoError(t, src.WriteTo(&buf))
	assert.NoError(t, dst.ReadFrom(&buf, -1))
	assert.StringEqual(t, "dst", dst, src)

	assert.StringEqual(t, "dst2.Imports[0]", dst2.Imports[0],
		"github.com/daviddengcn/go-villa")
}

func TestCheckRuneType_BOM(t *testing.T) {
	tp := CheckRuneType('A', 0xfeff)
	assert.Equal(t, "CheckRuneType(A, 0xfeff)", tp, index.TokenSep)
}

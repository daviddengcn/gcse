package gcse

import (
	"testing"
	"time"
	
	"github.com/daviddengcn/go-villa"
	"github.com/daviddengcn/go-assert"
)

func TestDocInfo(t *testing.T) {
	src := DocInfo {
		Name: "gcse",
		Package: "github.com/daviddengcn/gcse",
		Author: "github.com/daviddengcn",
		LastUpdated: time.Now(),
		StarCount: 10,
		Synopsis: "Go Package Search Engine",
		Description: "More details about GCSE",
		ProjectURL: "http://github.com/daviddengcn/gcse",
		ReadmeFn: "readme.txt",
		ReadmeData: "Just read me",
		Imports: []string {
			"github.com/daviddengcn/go-villa",
			"github.com/daviddengcn/sophie",
		},
		Exported: []string {
			"DocInfo", "CheckRuneType",
		},
	}
	var buf villa.ByteSlice
	assert.NoErrorf(t, "src.WriteTo failed: %v", src.WriteTo(&buf))
	
	var dst DocInfo
	assert.NoErrorf(t, "dst.ReadFrom failed: %v", dst.ReadFrom(&buf, -1))
	
	assert.StringEquals(t, "dst", dst, src)
}

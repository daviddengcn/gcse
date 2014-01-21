package gcse

import (
	"fmt"
	"testing"

	"github.com/daviddengcn/go-assert"
	"github.com/daviddengcn/go-villa"
	"github.com/daviddengcn/sophie"
	"github.com/daviddengcn/sophie/mr"
)

func TestIndex(t *testing.T) {
	docs := []DocInfo{
		{
			Package: "github.com/daviddengcn/gcse",
			Name:    "gcse",
		}, {
			Package: "github.com/daviddengcn/gcse/indexer",
			Name:    "main",
			Imports: []string{"github.com/daviddengcn/gcse"},
		},
	}
	ts, err := Index(&mr.InputStruct{
		PartCountF: func() (int, error) {
			return 1, nil
		},
		IteratorF: func(int) (sophie.IterateCloser, error) {
			index := 0
			return &sophie.IterateCloserStruct{
				NextF: func(key, val sophie.SophieReader) error {
					if index >= len(docs) {
						return sophie.EOF
					}
					*key.(*sophie.RawString) = sophie.RawString(
						docs[index].Package)
					*val.(*DocInfo) = docs[index]

					index++
					return nil
				},
			}, nil
		},
	})
	if err != nil {
		t.Error(err)
		return
	}

	numDocs := ts.DocCount()
	assert.Equals(t, "DocCount", numDocs, 2)

	var pkgs []string
	if err := ts.Search(map[string]villa.StrSet{IndexTextField: nil},
		func(docID int32, data interface{}) error {
			hit := data.(HitInfo)
			pkgs = append(pkgs, hit.Package)
			t.Logf("%s: %v", hit.Package, hit)
			return nil
		},
	); err != nil {
		t.Error(err)
		return
	}
	assert.StringEquals(t, "all", pkgs,
		"[github.com/daviddengcn/gcse github.com/daviddengcn/gcse/indexer]")

	var gcseInfo HitInfo
	if err := ts.Search(map[string]villa.StrSet{
		IndexPkgField: villa.NewStrSet("github.com/daviddengcn/gcse"),
	}, func(docID int32, data interface{}) error {
		gcseInfo = data.(HitInfo)
		return nil
	}); err != nil {
		t.Errorf("ts.Search: %v", err)
		return
	}
	assert.StringEquals(t, "gcseInfo.Imported",
		fmt.Sprintf("%+v", gcseInfo.Imported),
		"[github.com/daviddengcn/gcse/indexer]")

	var indexerInfo HitInfo
	if err := ts.Search(map[string]villa.StrSet{
		IndexPkgField: villa.NewStrSet("github.com/daviddengcn/gcse/indexer"),
	}, func(docID int32, data interface{}) error {
		gcseInfo = data.(HitInfo)
		return nil
	}); err != nil {
		t.Errorf("ts.Search: %v", err)
		return
	}
	assert.StringEquals(t, "indexerInfo.Imported",
		fmt.Sprintf("%+v", indexerInfo.Imported),
		"[]")
}

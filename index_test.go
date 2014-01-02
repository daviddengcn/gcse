package gcse

import (
	"testing"

	"github.com/daviddengcn/go-assert"
	"github.com/daviddengcn/go-villa"
	"github.com/daviddengcn/sophie"
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
	ts, err := Index(&sophie.InputStruct{
		PartCountFunc: func() (int, error) {
			return 1, nil
		},
		IteratorFunc: func(int) (sophie.IterateCloser, error) {
			index := 0
			return &sophie.IterateCloserStruct{
				NextFunc: func(key, val sophie.SophieReader) error {
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
}

package gcse

import (
	"github.com/daviddengcn/go-villa"
	"testing"
)

func TestIndex(t *testing.T) {
	docDB := NewMemDB("", "")
	docDB.Put("github.com/daviddengcn/gcse", DocInfo{
		Package: "github.com/daviddengcn/gcse",
		Name:    "gcse",
	})
	docDB.Put("github.com/daviddengcn/gcse/indexer", DocInfo{
		Package: "github.com/daviddengcn/gcse/indexer",
		Name:    "main",
		Imports: []string{"github.com/daviddengcn/gcse"},
	})
	ts, err := Index(docDB)
	if err != nil {
		t.Error(err)
		return
	}

	numDocs := ts.DocCount()
	AssertEquals(t, "DocCount", numDocs, 2)

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
	AssertStringEquals(t, "all", pkgs,
		"[github.com/daviddengcn/gcse github.com/daviddengcn/gcse/indexer]")
}

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
			TestImports: []string{
				"github.com/daviddengcn/go-villa",
				"github.com/daviddengcn/gcse",
			},
		}, {
			Package: "github.com/daviddengcn/gcse/indexer",
			Name:    "main",
			Imports: []string{
				"github.com/daviddengcn/gcse",
				"github.com/daviddengcn/go-villa",
				"github.com/daviddengcn/gcse/indexer",
			},
		}, {
			Package: "github.com/daviddengcn/go-villa",
			Name:    "villa",
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
					val.(*DocInfo).Imports = append([]string{}, docs[index].Imports...)
					val.(*DocInfo).TestImports = append([]string{}, docs[index].TestImports...)

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
	assert.Equals(t, "DocCount", numDocs, 3)

	var pkgs []string
	if err := ts.Search(map[string]villa.StrSet{IndexTextField: nil},
		func(docID int32, data interface{}) error {
			hit := data.(HitInfo)
			pkgs = append(pkgs, hit.Package)
			return nil
		},
	); err != nil {
		t.Error(err)
		return
	}
	assert.LinesEqual(t, "all", pkgs,
		[]string{
			"github.com/daviddengcn/gcse",
			"github.com/daviddengcn/go-villa",
			"github.com/daviddengcn/gcse/indexer",
		})

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
	assert.LinesEqual(t, "gcseInfo.Imported",
		gcseInfo.Imported,
		[]string{
			"github.com/daviddengcn/gcse/indexer",
		})
	assert.LinesEqual(t, "gcseInfo.TestImports",
		gcseInfo.TestImports,
		[]string{
			"github.com/daviddengcn/go-villa",
		})

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
	assert.LinesEqual(t, "indexerInfo.Imported",
		indexerInfo.Imported, nil)
	assert.LinesEqual(t, "indexerInfo.Imports",
		indexerInfo.Imports, nil)

	if err := ts.Search(map[string]villa.StrSet{
		IndexPkgField: villa.NewStrSet("github.com/daviddengcn/go-villa"),
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
	assert.LinesEqual(t, "gcseInfo.TestImported",
		gcseInfo.TestImported,
		[]string{"github.com/daviddengcn/gcse"})
}

func TestAppendTokens_filter(t *testing.T) {
	SRC_DST := []interface{} {
		"My address is http://go-search.org", []string{"my", "address", "is"},
		"Hello david_deng-cn.123@gmail-yahoo.com", []string{"hello"},
	}
	
	for i := 0; i < len(SRC_DST); i += 2 {
		SRC := SRC_DST[i].(string);
		DST := villa.NewStrSet(SRC_DST[i + 1].([]string)...);
		
		assert.StrSetEquals(t, "Tokens of " + SRC, AppendTokens(nil, []byte(SRC)), DST)
	}
}
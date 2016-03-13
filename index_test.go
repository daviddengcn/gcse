package gcse

import (
	"io"
	"path"
	"testing"

	"github.com/golangplus/strings"
	"github.com/golangplus/testing/assert"

	"github.com/daviddengcn/go-index"
	"github.com/daviddengcn/sophie"
	"github.com/daviddengcn/sophie/mr"
)

func TestIndex(t *testing.T) {
	const (
		package0 = "github.com/daviddengcn/gcse"
		package1 = "github.com/daviddengcn/gcse/indexer"
		package2 = "github.com/daviddengcn/go-villa"
	)

	docs := []DocInfo{
		{
			Package: package0,
			Name:    "gcse",
			TestImports: []string{
				package2, package0,
			},
		}, {
			Package: package1,
			Name:    "main",
			Imports: []string{
				package0,
				package2,
				package1,
			},
		}, {
			Package: package2,
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
						return io.EOF
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
	}, "./tmp")
	assert.NoErrorOrDie(t, err)

	hitsArr, err := index.OpenConstArray(path.Join("./tmp", HitsArrFn))
	for _, doc := range docs {
		idx := -1
		ts.Search(index.SingleFieldQuery(IndexPkgField, doc.Package), func(docID int32, data interface{}) error {
			idx = int(docID)
			return nil
		})
		d, err := hitsArr.GetGob(idx)
		assert.NoError(t, err)
		assert.Equal(t, "d.Package", d.(HitInfo).Package, doc.Package)
	}
	numDocs := ts.DocCount()
	assert.Equal(t, "DocCount", numDocs, 3)

	var pkgs []string
	if err := ts.Search(map[string]stringsp.Set{IndexTextField: nil},
		func(docID int32, data interface{}) error {
			hit := data.(HitInfo)
			pkgs = append(pkgs, hit.Package)
			return nil
		},
	); err != nil {
		t.Error(err)
		return
	}
	assert.StringEqual(t, "all", pkgs,
		[]string{
			"github.com/daviddengcn/gcse",
			"github.com/daviddengcn/go-villa",
			"github.com/daviddengcn/gcse/indexer",
		})

	var gcseInfo HitInfo
	if err := ts.Search(map[string]stringsp.Set{
		IndexPkgField: stringsp.NewSet("github.com/daviddengcn/gcse"),
	}, func(docID int32, data interface{}) error {
		gcseInfo = data.(HitInfo)
		return nil
	}); err != nil {
		t.Errorf("ts.Search: %v", err)
		return
	}
	assert.Equal(t, "gcseInfo.Imported", gcseInfo.Imported, []string(nil))
	assert.Equal(t, "gcseInfo.ImportedLen", gcseInfo.ImportedLen, 1)
	assert.Equal(t, "gcseInfo.TestImports", gcseInfo.TestImports, []string{"github.com/daviddengcn/go-villa"})

	var indexerInfo HitInfo
	if err := ts.Search(map[string]stringsp.Set{
		IndexPkgField: stringsp.NewSet("github.com/daviddengcn/gcse/indexer"),
	}, func(docID int32, data interface{}) error {
		gcseInfo = data.(HitInfo)
		return nil
	}); err != nil {
		t.Errorf("ts.Search: %v", err)
		return
	}
	assert.StringEqual(t, "indexerInfo.Imported",
		indexerInfo.Imported, []string{})
	assert.StringEqual(t, "indexerInfo.Imports",
		indexerInfo.Imports, []string{})

	if err := ts.Search(map[string]stringsp.Set{
		IndexPkgField: stringsp.NewSet("github.com/daviddengcn/go-villa"),
	}, func(docID int32, data interface{}) error {
		gcseInfo = data.(HitInfo)
		return nil
	}); err != nil {
		t.Errorf("ts.Search: %v", err)
		return
	}
	assert.Equal(t, "indexerInfo.Imported", indexerInfo.Imported, []string(nil))
	assert.Equal(t, "gcseInfo.TestImportedLen", gcseInfo.TestImportedLen, 1)
	assert.Equal(t, "gcseInfo.TestImported", gcseInfo.TestImported, []string(nil))
}

func TestAppendTokens_filter(t *testing.T) {
	SRC_DST := []interface{}{
		"My address is http://go-search.org", []string{"my", "address", "is"},
		"Hello david_deng-cn.123@gmail-yahoo.com", []string{"hello"},
	}

	for i := 0; i < len(SRC_DST); i += 2 {
		SRC := SRC_DST[i].(string)
		DST := stringsp.NewSet(SRC_DST[i+1].([]string)...)

		assert.Equal(t, "Tokens of "+SRC, AppendTokens(nil, []byte(SRC)), DST)
	}
}

func search(ts *index.TokenSetSearcher, field string, text string) ([]HitInfo, error) {
	var hits []HitInfo
	err := ts.Search(map[string]stringsp.Set{field: AppendTokens(nil, []byte(text))}, func(_ int32, data interface{}) error {
		hits = append(hits, data.(HitInfo))
		return nil
	})
	return hits, err
}

func TestIndex_DescNotIndexedBug(t *testing.T) {
	const (
		description = "description"
		readme      = "readme"
	)
	hits := []HitInfo{{
		DocInfo: DocInfo{
			Package:     "github.com/daviddengcn/gcse",
			Name:        "gcse",
			Description: description,
			ReadmeData:  readme,
		},
	}}
	idxs := []int{0}
	fullHitSaved := 0
	ts := &index.TokenSetSearcher{}
	assert.NoError(t, indexAndSaveHits(ts, hits, idxs, func(hit *HitInfo) error {
		fullHitSaved++
		assert.Equal(t, "Description", hit.Description, description)
		assert.Equal(t, "Readme", hit.ReadmeData, readme)
		return nil
	}))
	assert.Equal(t, "fullHitSaved", fullHitSaved, 1)
	results, err := search(ts, IndexTextField, description)
	assert.NoError(t, err)
	assert.Equal(t, "results", results, hits)

	results, err = search(ts, IndexTextField, readme)
	assert.NoError(t, err)
	assert.Equal(t, "results", results, hits)
}

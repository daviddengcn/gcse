package gcse

import (
	"encoding/gob"
	"os"
	"testing"

	"github.com/golangplus/bytes"
	"github.com/golangplus/strings"
	"github.com/golangplus/testing/assert"

	"github.com/boltdb/bolt"
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
	if !assert.NoError(t, os.RemoveAll("/tmp/gcse-TestIndex.bolt")) {
		return
	}
	wholeInfoDb, err := bolt.Open("/tmp/gcse-TestIndex.bolt", 0644, nil)
	if !assert.NoError(t, err) {
		return
	}
	defer wholeInfoDb.Close()

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
	}, wholeInfoDb)
	assert.NoErrorOrDie(t, err)

	assert.NoError(t, wholeInfoDb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(IndexHitsBucket))
		if !assert.Should(t, b != nil, "Bucket not found!") {
			return nil
		}
		for _, doc := range docs {
			bs := bytesp.Slice(b.Get([]byte(doc.Package)))
			if assert.Should(t, bs != nil, "Get "+doc.Package+" returns nil") {
				var info HitInfo
				if assert.NoError(t, gob.NewDecoder(&bs).Decode(&info)) {
					assert.Equal(t, "package", info.Package, doc.Package)
				}
			}
		}
		return nil
	}))

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

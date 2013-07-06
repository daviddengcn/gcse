package gcse

import (
	"errors"
	"github.com/daviddengcn/go-index"
	"github.com/daviddengcn/go-villa"
	"log"
)

var errNotDocInfo = errors.New("Value is not DocInfo")

func Index(docDB *MemDB) (*index.TokenSetSearcher, error) {
	DumpMemStats()
	log.Printf("Generating importsDB ...")
	importsDB := NewTokenIndexer("", "")
	// generate importsDB
	if err := docDB.Iterate(func(pkg string, val interface{}) error {
		docInfo, ok := val.(DocInfo)
		if !ok {
			return errNotDocInfo
		}
		importsDB.Put(pkg, villa.NewStrSet(docInfo.Imports...))
		return nil
	}); err != nil {
		return nil, err
	}

	DumpMemStats()
	log.Printf("Making TokenSetSearcher ...")

	ts := &index.TokenSetSearcher{}
	if err := docDB.Iterate(func(key string, val interface{}) error {
		var hitInfo HitInfo

		var ok bool
		hitInfo.DocInfo, ok = val.(DocInfo)
		if !ok {
			return errNotDocInfo
		}
		hitInfo.Imported = importsDB.IdsOfToken(hitInfo.Package)
		// StaticScore is calculated after setting all other fields of hitInfo
		hitInfo.StaticScore = CalcStaticScore(&hitInfo)

		var tokens villa.StrSet
		tokens = AppendTokens(tokens, []byte(hitInfo.Name))
		tokens = AppendTokens(tokens, []byte(hitInfo.Package))
		tokens = AppendTokens(tokens, []byte(hitInfo.Description))
		tokens = AppendTokens(tokens, []byte(hitInfo.ReadmeData))
		tokens = AppendTokens(tokens, []byte(hitInfo.Author))

		ts.AddDoc(map[string]villa.StrSet{
			"text": tokens,
			"pkg":  villa.NewStrSet(hitInfo.Package),
		}, hitInfo)
		return nil
	}); err != nil {
		return nil, err
	}
	importsDB = nil
	return ts, nil
}

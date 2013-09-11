package gcse

import (
	"errors"
	"github.com/daviddengcn/go-index"
	"github.com/daviddengcn/go-villa"
	"log"
)

const (
	IndexTextField = "text"
	IndexPkgField  = "pkg"
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

	var hits []HitInfo
	if err := docDB.Iterate(func(key string, val interface{}) error {
		var hitInfo HitInfo

		var ok bool
		hitInfo.DocInfo, ok = val.(DocInfo)
		if !ok {
			return errNotDocInfo
		}
		hitInfo.Imported = importsDB.IdsOfToken(hitInfo.Package)

		readme := ReadmeToText(hitInfo.ReadmeFn, hitInfo.ReadmeData)

		hitInfo.ImportantSentences = ChooseImportantSentenses(readme, hitInfo.Name, hitInfo.Package)
		// StaticScore is calculated after setting all other fields of hitInfo
		hitInfo.StaticScore = CalcStaticScore(&hitInfo)

		hits = append(hits, hitInfo)
		return nil
	}); err != nil {
		return nil, err
	}

	log.Printf("%d hits collected, sorting static-scores in descending order",
		len(hits))
	idxs := make([]int, len(hits))
	for i := range idxs {
		idxs[i] = i
	}
	villa.SortF(len(idxs), func(i, j int) bool {
		return hits[idxs[i]].StaticScore > hits[idxs[j]].StaticScore
	}, func(i, j int) {
		idxs[i], idxs[j] = idxs[j], idxs[i]
	})
	ts := &index.TokenSetSearcher{}

	log.Printf("Indexing to TokenSetSearcher ...")
	rank := 0
	for i := range idxs {
		hit := &hits[idxs[i]]
		if i > 0 && hit.StaticScore < hits[idxs[i-1]].StaticScore {
			rank = i
		}
		hit.StaticRank = rank

		var tokens villa.StrSet
		tokens = AppendTokens(tokens, []byte(hit.Name))
		tokens = AppendTokens(tokens, []byte(hit.Package))
		tokens = AppendTokens(tokens, []byte(hit.Description))
		tokens = AppendTokens(tokens, []byte(hit.ReadmeData))
		tokens = AppendTokens(tokens, []byte(hit.Author))

		ts.AddDoc(map[string]villa.StrSet{
			IndexTextField: tokens,
			IndexPkgField:  villa.NewStrSet(hit.Package),
		}, *hit)
	}

	importsDB = nil
	return ts, nil
}

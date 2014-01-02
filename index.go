package gcse

import (
	"errors"
	"log"

	"github.com/daviddengcn/go-index"
	"github.com/daviddengcn/go-villa"
	"github.com/daviddengcn/sophie"
)

const (
	IndexTextField = "text"
	IndexPkgField  = "pkg"
)

var errNotDocInfo = errors.New("Value is not DocInfo")

func Index(docDB sophie.Input) (*index.TokenSetSearcher, error) {
	DumpMemStats()

	docPartCnt, err := docDB.PartCount()
	if err != nil {
		return nil, err
	}
	docCount := 0

	log.Printf("Generating importsDB ...")
	importsDB := NewTokenIndexer("", "")
	// generate importsDB
	for i := 0; i < docPartCnt; i++ {
		it, err := docDB.Iterator(i)
		if err != nil {
			return nil, err
		}

		var pkg sophie.RawString
		var docInfo DocInfo
		for {
			if err := it.Next(&pkg, &docInfo); err != nil {
				if err == sophie.EOF {
					break
				}
				it.Close()
				return nil, err
			}
			importsDB.Put(string(pkg), villa.NewStrSet(docInfo.Imports...))
			docCount++
		}

		it.Close()
	}

	DumpMemStats()
	log.Printf("Making HitInfos ...")
	hits := make([]HitInfo, 0, docCount)
	for i := 0; i < docPartCnt; i++ {
		it, err := docDB.Iterator(i)
		if err != nil {
			return nil, err
		}

		var pkg sophie.RawString
		var hitInfo HitInfo
		for {
			if err := it.Next(&pkg, &hitInfo.DocInfo); err != nil {
				if err == sophie.EOF {
					break
				}
				it.Close()
				return nil, err
			}

			hitInfo.Imported = importsDB.IdsOfToken(hitInfo.Package)

			readme := ReadmeToText(hitInfo.ReadmeFn, hitInfo.ReadmeData)

			hitInfo.ImportantSentences = ChooseImportantSentenses(readme,
				hitInfo.Name, hitInfo.Package)
			// StaticScore is calculated after setting all other fields of
			// hitInfo
			hitInfo.StaticScore = CalcStaticScore(&hitInfo)

			hits = append(hits, hitInfo)
		}

		it.Close()
	}

	DumpMemStats()
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

	DumpMemStats()
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
		for _, word := range hit.Exported {
			AppendTokens(tokens, []byte(word))
		}

		ts.AddDoc(map[string]villa.StrSet{
			IndexTextField: tokens,
			IndexPkgField:  villa.NewStrSet(hit.Package),
		}, *hit)
	}

	DumpMemStats()
	importsDB = nil
	return ts, nil
}

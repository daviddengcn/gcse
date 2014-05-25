package gcse

import (
	"errors"
	"log"
	"time"

	"github.com/daviddengcn/go-index"
	"github.com/daviddengcn/go-villa"
	"github.com/daviddengcn/sophie"
	"github.com/daviddengcn/sophie/mr"
)

const (
	IndexTextField = "text"
	IndexNameField = "name"
	IndexPkgField  = "pkg"
)

var errNotDocInfo = errors.New("Value is not DocInfo")

func Index(docDB mr.Input) (*index.TokenSetSearcher, error) {
	DumpMemStats()

	docPartCnt, err := docDB.PartCount()
	if err != nil {
		return nil, err
	}
	docCount := 0

	log.Printf("Generating importsDB ...")
	importsDB := NewTokenIndexer("", "")
	testImportsDB := NewTokenIndexer("", "")
	// per project imported by projects
	prjImportsDB := NewTokenIndexer("", "")
	prjStars := make(map[string]struct {
		StarCount   int
		LastUpdated time.Time
	})
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
			testImportsDB.Put(string(pkg),
				villa.NewStrSet(docInfo.TestImports...))

			var projects villa.StrSet
			for _, imp := range docInfo.Imports {
				projects.Put(FullProjectOfPackage(imp))
			}
			for _, imp := range docInfo.TestImports {
				projects.Put(FullProjectOfPackage(imp))
			}
			prj := FullProjectOfPackage(string(pkg))
			orgProjects := prjImportsDB.TokensOfId(prj)
			projects.Put(orgProjects...)
			prjImportsDB.Put(prj, projects)

			// update stars
			if cur, ok := prjStars[prj]; !ok ||
				docInfo.LastUpdated.After(cur.LastUpdated) {
				prjStars[prj] = struct {
					StarCount   int
					LastUpdated time.Time
				}{
					docInfo.StarCount,
					docInfo.LastUpdated,
				}
			}

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
			hitInfo.TestImported = testImportsDB.IdsOfToken(hitInfo.Package)

			prj := FullProjectOfPackage(hitInfo.Package)
			impPrjsCnt := len(prjImportsDB.IdsOfToken(prj))
			var assignedStarCount = float64(prjStars[prj].StarCount)
			if prj != hitInfo.Package {
				if impPrjsCnt == 0 {
					assignedStarCount = 0
				} else {
					perStarCount :=
						float64(prjStars[prj].StarCount) / float64(impPrjsCnt)

					var projects villa.StrSet
					for _, imp := range hitInfo.Imported {
						projects.Put(FullProjectOfPackage(imp))
					}
					for _, imp := range hitInfo.TestImported {
						projects.Put(FullProjectOfPackage(imp))
					}
					assignedStarCount = perStarCount * float64(len(projects))
				}
			}
			hitInfo.AssignedStarCount = assignedStarCount

			readme := ReadmeToText(hitInfo.ReadmeFn, hitInfo.ReadmeData)

			hitInfo.ImportantSentences = ChooseImportantSentenses(readme,
				hitInfo.Name, hitInfo.Package)
			// StaticScore is calculated after setting all other fields of
			// hitInfo
			hitInfo.StaticScore = CalcStaticScore(&hitInfo)
			hitInfo.TestStaticScore = CalcTestStaticScore(&hitInfo)
			hits = append(hits, hitInfo)
		}

		it.Close()
	}

	DumpMemStats()
	importsDB = nil
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

		var nameTokens villa.StrSet
		nameTokens = AppendTokens(nameTokens, []byte(hit.Name))

		var tokens villa.StrSet
		tokens.Put(nameTokens.Elements()...)
		tokens = AppendTokens(tokens, []byte(hit.Package))
		tokens = AppendTokens(tokens, []byte(hit.Description))
		tokens = AppendTokens(tokens, []byte(hit.ReadmeData))
		tokens = AppendTokens(tokens, []byte(hit.Author))
		for _, word := range hit.Exported {
			AppendTokens(tokens, []byte(word))
		}

		ts.AddDoc(map[string]villa.StrSet{
			IndexTextField: tokens,
			IndexNameField: nameTokens,
			IndexPkgField:  villa.NewStrSet(hit.Package),
		}, *hit)
	}

	DumpMemStats()
	return ts, nil
}

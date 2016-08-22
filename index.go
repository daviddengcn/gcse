package gcse

import (
	"errors"
	"io"
	"log"
	"os"
	"path"
	"time"

	"github.com/golangplus/errors"
	"github.com/golangplus/sort"
	"github.com/golangplus/strings"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/cheggaaa/pb"
	"github.com/daviddengcn/gcse/utils"
	"github.com/daviddengcn/go-index"
	"github.com/daviddengcn/sophie"
	"github.com/daviddengcn/sophie/mr"
)

const (
	HitsArrFn = "hits"

	IndexTextField = "text"
	IndexNameField = "name"
	IndexPkgField  = "pkg"
)

var errNotDocInfo = errors.New("Value is not DocInfo")

// Excludes packages in src which has same full-project with any elements in excl.
func excludeImports(src, excl []string) (dst []string) {
	exclPrjsSets := stringsp.NewSet()
	for _, pkg := range excl {
		exclPrjsSets.Add(FullProjectOfPackage(pkg))
	}

	for _, pkg := range src {
		prj := FullProjectOfPackage(string(pkg))
		if !exclPrjsSets.Contain(prj) {
			dst = append(dst, pkg)
		}
	}
	return dst
}

func filterDocInfo(docInfo *DocInfo) {
	for i, imp := range docInfo.Imports {
		if imp == docInfo.Package {
			docInfo.Imports = stringsp.SliceRemove(docInfo.Imports, i)
			break
		}
	}
	for i, imp := range docInfo.TestImports {
		if imp == docInfo.Package {
			docInfo.TestImports = stringsp.SliceRemove(docInfo.TestImports, i)
			break
		}
	}
}

func indexAndSaveHits(ts *index.TokenSetSearcher, hits []HitInfo, idxs []int, saveFullHit func(*HitInfo) error) error {
	rank := 0
	var bar *pb.ProgressBar
	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		bar = pb.New(len(idxs))
		bar.Start()
	}
	for i := range idxs {
		hit := &hits[idxs[i]]
		if i > 0 && hit.StaticScore < hits[idxs[i-1]].StaticScore {
			rank = i
		}
		hit.StaticRank = rank

		if err := saveFullHit(hit); err != nil {
			return err
		}

		var desc, readme string
		desc, hit.Description = hit.Description, ""
		readme, hit.ReadmeData = hit.ReadmeData, ""
		hit.Imported = nil
		hit.TestImported = nil

		var nameTokens stringsp.Set
		nameTokens = AppendTokens(nameTokens, []byte(hit.Name))

		var tokens stringsp.Set
		tokens.Add(nameTokens.Elements()...)
		tokens = AppendTokens(tokens, []byte(hit.Package))
		tokens = AppendTokens(tokens, []byte(desc))
		tokens = AppendTokens(tokens, []byte(readme))
		tokens = AppendTokens(tokens, []byte(hit.Author))
		for _, word := range hit.Exported {
			AppendTokens(tokens, []byte(word))
		}
		ts.AddDoc(map[string]stringsp.Set{
			IndexTextField: tokens,
			IndexNameField: nameTokens,
			IndexPkgField:  stringsp.NewSet(hit.Package),
		}, *hit)
		if bar != nil {
			bar.Increment()
		}
	}
	if bar != nil {
		bar.FinishPrint("Indexing finished!")
	}
	utils.DumpMemStats()
	return nil
}

func Index(docDB mr.Input, outDir string) (*index.TokenSetSearcher, error) {
	utils.DumpMemStats()

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
	type projectStart struct {
		StarCount   int
		LastUpdated time.Time
	}
	prjStars := make(map[string]projectStart)

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
				if errorsp.Cause(err) == io.EOF {
					break
				}
				it.Close()
				return nil, err
			}
			filterDocInfo(&docInfo)

			importsDB.PutTokens(string(pkg), stringsp.NewSet(docInfo.Imports...))
			testImportsDB.PutTokens(string(pkg), stringsp.NewSet(docInfo.TestImports...))

			var projects stringsp.Set
			for _, imp := range docInfo.Imports {
				projects.Add(FullProjectOfPackage(imp))
			}
			for _, imp := range docInfo.TestImports {
				projects.Add(FullProjectOfPackage(imp))
			}
			prj := FullProjectOfPackage(string(pkg))
			orgProjects := prjImportsDB.TokensOfId(prj)
			projects.Add(orgProjects...)
			prjImportsDB.PutTokens(prj, projects)

			// update stars
			if cur, ok := prjStars[prj]; !ok ||
				docInfo.LastUpdated.After(cur.LastUpdated) {
				prjStars[prj] = projectStart{
					StarCount:   docInfo.StarCount,
					LastUpdated: docInfo.LastUpdated,
				}
			}
			docCount++
		}
		it.Close()
	}

	utils.DumpMemStats()
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
				if errorsp.Cause(err) == io.EOF {
					break
				}
				it.Close()
				return nil, err
			}
			filterDocInfo(&hitInfo.DocInfo)

			hitInfo.Imported = importsDB.IdsOfToken(hitInfo.Package)
			hitInfo.ImportedLen = len(hitInfo.Imported)
			hitInfo.TestImported = testImportsDB.IdsOfToken(hitInfo.Package)
			hitInfo.TestImportedLen = len(hitInfo.TestImported)
			realTestImported := excludeImports(testImportsDB.IdsOfToken(hitInfo.Package), hitInfo.Imported)

			prj := FullProjectOfPackage(hitInfo.Package)
			impPrjsCnt := len(prjImportsDB.IdsOfToken(prj))
			var assignedStarCount = float64(prjStars[prj].StarCount)
			if prj != hitInfo.Package {
				if impPrjsCnt == 0 {
					assignedStarCount = 0
				} else {
					perStarCount :=
						float64(prjStars[prj].StarCount) / float64(impPrjsCnt)

					var projects stringsp.Set
					for _, imp := range hitInfo.Imported {
						projects.Add(FullProjectOfPackage(imp))
					}
					for _, imp := range hitInfo.TestImported {
						projects.Add(FullProjectOfPackage(imp))
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
			hitInfo.TestStaticScore = CalcTestStaticScore(&hitInfo, realTestImported)
			hits = append(hits, hitInfo)
		}
		it.Close()
	}

	utils.DumpMemStats()
	importsDB = nil
	testImportsDB = nil
	utils.DumpMemStats()
	log.Printf("%d hits collected, sorting static-scores in descending order", len(hits))

	idxs := sortp.IndexSortF(len(hits), func(i, j int) bool {
		return hits[i].StaticScore > hits[j].StaticScore
	})

	ts := &index.TokenSetSearcher{}
	utils.DumpMemStats()
	log.Printf("Indexing %d packages to TokenSetSearcher ...", len(idxs))
	hitsArr, err := index.CreateConstArray(path.Join(outDir, HitsArrFn))
	if err != nil {
		return nil, err
	}
	defer hitsArr.Close()

	if err := indexAndSaveHits(ts, hits, idxs, func(hit *HitInfo) error {
		_, err := hitsArr.AppendGob(*hit)
		return err
	}); err != nil {
		return nil, err
	}

	return ts, nil
}

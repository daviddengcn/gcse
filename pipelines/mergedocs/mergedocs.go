// Input
//   FnDocs
//   FnNewDocs
package main

import (
	//	"fmt"
	"io"
	"log"
	"regexp"
	"sync/atomic"

	"github.com/golangplus/errors"
	"github.com/golangplus/strings"

	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/gcse/configs"
	"github.com/daviddengcn/go-villa"
	"github.com/daviddengcn/sophie"
	"github.com/daviddengcn/sophie/kv"
	"github.com/daviddengcn/sophie/mr"
)

func main() {
	log.Println("Merging new crawled docs back...")

	var nonStorePackage *regexp.Regexp
	if len(configs.NonStorePackageRegexps) > 0 {
		nonStorePackage = regexp.MustCompile(
			stringsp.FullJoin(configs.NonStorePackageRegexps, "(", ")|(", ")"))
	}

	fpDataRoot := sophie.LocalFsPath(configs.DataRoot.S())

	fpCrawler := configs.CrawlerDBFsPath()
	outDocsUpdated := kv.DirOutput(fpDataRoot.Join("docs-updated"))
	outDocsUpdated.Clean()

	var cntDeleted, cntUpdated, cntNew, cntUnchanged int64

	job := mr.MrJob{
		Source: []mr.Input{
			kv.DirInput(fpDataRoot.Join(configs.FnDocs)),   // 0
			kv.DirInput(fpCrawler.Join(configs.FnNewDocs)), // 1
		},

		NewMapperF: func(src, part int) mr.Mapper {
			if src == 0 {
				// Mapper for docs
				return &mr.MapperStruct{
					NewKeyF: sophie.NewRawString,
					NewValF: gcse.NewDocInfo,
					MapF: func(key, val sophie.SophieWriter, c mr.PartCollector) error {
						pkg := key.(*sophie.RawString).String()
						di := val.(*gcse.DocInfo)
						act := gcse.NewDocAction{
							Action:  gcse.NDA_ORIGINAL,
							DocInfo: *di,
						}
						part := gcse.CalcPackagePartition(pkg, gcse.DOCS_PARTS)
						return c.CollectTo(part, key, &act)
					},
				}
			}
			// Mapper for new docs
			return &mr.MapperStruct{
				NewKeyF: sophie.NewRawString,
				NewValF: gcse.NewNewDocAction,
				MapF: func(key, val sophie.SophieWriter, c mr.PartCollector) error {
					pkg := string(*key.(*sophie.RawString))
					part := gcse.CalcPackagePartition(pkg, gcse.DOCS_PARTS)
					return c.CollectTo(part, key, val)
				},
			}
		},

		Sorter: mr.NewFileSorter(fpDataRoot.Join("tmp")),

		NewReducerF: func(part int) mr.Reducer {
			return &mr.ReducerStruct{
				NewKeyF: sophie.NewRawString,
				NewValF: gcse.NewNewDocAction,
				ReduceF: func(key sophie.SophieWriter,
					nextVal mr.SophierIterator, c []sophie.Collector) error {

					if nonStorePackage != nil {
						pkg := string(*key.(*sophie.RawString))
						if nonStorePackage.MatchString(pkg) {
							log.Printf("Ignoring non-store pkg: %s", pkg)
							return nil
						}
					}

					var act gcse.DocInfo
					isSet := false
					isUpdated := false
					hasOriginal := false
					for {
						val, err := nextVal()
						if errorsp.Cause(err) == io.EOF {
							break
						}
						if err != nil {
							return err
						}

						cur := val.(*gcse.NewDocAction)
						switch cur.Action {
						case gcse.NDA_DEL:
							// not collect out to delete it
							atomic.AddInt64(&cntDeleted, 1)
							return nil

						case gcse.NDA_ORIGINAL:
							hasOriginal = true
						}

						if !isSet {
							isSet = true
							act = cur.DocInfo
						} else {
							if cur.LastUpdated.After(act.LastUpdated) {
								isUpdated = true
								act = cur.DocInfo
							}
						}
					}

					if isSet {
						if isUpdated {
							atomic.AddInt64(&cntUpdated, 1)
						} else if hasOriginal {
							atomic.AddInt64(&cntUnchanged, 1)
						} else {
							atomic.AddInt64(&cntNew, 1)
						}
						return c[0].Collect(key, &act)
					} else {
						return nil
					}
				},
			}
		},

		Dest: []mr.Output{
			outDocsUpdated,
		},
	}

	if err := job.Run(); err != nil {
		log.Fatalf("job.Run failed: %v", err)
	}

	log.Printf("Deleted:   %v", cntDeleted)
	log.Printf("Updated:   %v", cntUpdated)
	log.Printf("New:       %v", cntNew)
	log.Printf("Unchanged: %v", cntUnchanged)

	pDocs := villa.Path(configs.DocsDBPath())
	pUpdated := configs.DataRoot.Join("docs-updated")
	pTmp := configs.DataRoot.Join("docs-tmp")

	pTmp.RemoveAll()
	if err := pDocs.Rename(pTmp); err != nil {
		log.Fatalf("rename %v to %v failed: %v", pDocs, pTmp, err)
	}
	if err := pUpdated.Rename(pDocs); err != nil {
		log.Fatalf("rename %v to %v failed: %v", pUpdated, pDocs, err)
	}

	log.Println("Merging success...")
}

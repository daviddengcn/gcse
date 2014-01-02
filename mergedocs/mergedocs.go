package main

import (
//	"fmt"
	"log"
	"sync/atomic"
	
	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/sophie"
)

type RawStringKey struct {}

func (RawStringKey) NewKey() sophie.Sophier {
	return new(sophie.RawString)
}

type docsMapper struct {
	sophie.EmptyMapper
	RawStringKey
}

func (*docsMapper) NewVal() sophie.Sophier {
	return new(gcse.DocInfo)
}

func (*docsMapper) Map(key, val sophie.SophieWriter, c sophie.PartCollector) error {
	pkg := string(*key.(*sophie.RawString))
	di := val.(*gcse.DocInfo)
	act := gcse.NewDocAction {
		Action: gcse.NDA_UPDATE,
		DocInfo: *di,
	}
	
	part := gcse.CalcPackagePartition(pkg, gcse.DOCS_PARTS)
	return c.CollectTo(part, key, &act)
}

type newdocsMapper struct {
	sophie.EmptyMapper
	RawStringKey
}

func (*newdocsMapper) NewVal() sophie.Sophier {
	return new(gcse.NewDocAction)
}

func (*newdocsMapper) Map(key, val sophie.SophieWriter, c sophie.PartCollector) error {
	pkg := string(*key.(*sophie.RawString))
	part := gcse.CalcPackagePartition(pkg, gcse.DOCS_PARTS)
	return c.CollectTo(part, key, val)
}

type mergedocsReducer struct {
	sophie.EmptyReducer
	RawStringKey
}

func (mergedocsReducer) NewVal() sophie.Sophier {
	return new(gcse.NewDocAction)
}

var (
	cntDeleted     int64
	cntUpdated     int64
	cntNewUnchange int64
)

func (mergedocsReducer) Reduce(key sophie.SophieWriter,
		nextVal sophie.SophierIterator, c []sophie.Collector) error {
	var act gcse.DocInfo
	isSet := false
	isUpdated := false
	for {
		val, err := nextVal()
		if err == sophie.EOF {
			break;
		}
		if err != nil {
			return err;
		}
		
		cur := val.(*gcse.NewDocAction)
		if cur.Action == gcse.NDA_DEL {
			// not collect out to delete it
			atomic.AddInt64(&cntDeleted, 1)
			return nil
		}
		if !isSet {
			isSet = true
			act = cur.DocInfo
//			fmt.Printf("First %v - %v vs %v, %p\n", key, cur.LastUpdated, act.LastUpdated, cur)
		} else {
//			fmt.Printf("Later %v - %v vs %v, %p\n", key, cur.LastUpdated, act.LastUpdated, cur)
			if cur.LastUpdated.After(act.LastUpdated) {
				isUpdated = true
				act = cur.DocInfo
			}
		}
	}
	
	if isSet {
		if isUpdated {
			atomic.AddInt64(&cntUpdated, 1)
		} else {
			atomic.AddInt64(&cntNewUnchange, 1)
		}
		return c[0].Collect(key, &act)
	} else {
		return nil
	}
}

func main() {
	log.Println("Merging new crawled docs back...")
	
	fpDataRoot := sophie.FsPath {
		Fs: sophie.LocalFS,
		Path: gcse.DataRoot.S(),
	}
	
	fpCrawler := fpDataRoot.Join(gcse.FnCrawlerDB)
	outDocsUpdated := sophie.KVDirOutput(fpDataRoot.Join("docs-updated"))
	outDocsUpdated.Clean()
	
	job := sophie.MrJob {
		Source: []sophie.Input{
			sophie.KVDirInput(fpDataRoot.Join(gcse.FnDocs)),		// 0
			sophie.KVDirInput(fpCrawler.Join(gcse.FnNewDocs)),		// 1
		},
		
		MapFactory: sophie.MapperFactoryFunc(
		func(src, part int) sophie.Mapper {
			if src == 0 {
				return &docsMapper{}
			}
			
			return &newdocsMapper{}
		}),
		
		Sorter: sophie.NewFileSorter(fpDataRoot.Join("tmp")),
		
		RedFactory: sophie.ReducerFactoryFunc(func(part int) sophie.Reducer {
			return mergedocsReducer{}
		}),
		
		Dest: []sophie.Output{
			outDocsUpdated,
		},
	}
	
	if err := job.Run(); err != nil {
		log.Fatalf("job.Run failed: %v", err)
	}
	
	log.Printf("Deleted: %v", cntDeleted)
	log.Printf("Updated: %v", cntUpdated)
	log.Printf("NewUnchange: %v", cntNewUnchange)
	
	pDocs := gcse.DataRoot.Join(gcse.FnDocs)
	pUpdated := gcse.DataRoot.Join("docs-updated")
	pTmp := gcse.DataRoot.Join("docs-tmp")
	
	pTmp.RemoveAll()
	if err := pDocs.Rename(pTmp); err != nil {
		log.Fatalf("rename %v to %v failed: %v", pDocs, pTmp, err)
	}
	if err := pUpdated.Rename(pDocs); err != nil {
		log.Fatalf("rename %v to %v failed: %v", pUpdated, pDocs, err)
	}

	log.Println("Merging success...")
}

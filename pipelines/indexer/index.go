package main

import (
	"log"
	"os"
	"runtime"

	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/gcse/configs"
	"github.com/daviddengcn/gcse/store"
	"github.com/daviddengcn/gcse/utils"
	"github.com/daviddengcn/go-easybi"
	"github.com/daviddengcn/sophie/kv"
)

func clearOutdatedIndex() error {
	segm, err := configs.IndexSegments().FindMaxDone()
	if err != nil {
		return err
	}
	all, err := configs.IndexSegments().ListAll()
	if err != nil {
		return err
	}
	for _, s := range all {
		if s == segm {
			continue
		}
		err := s.Remove()
		if err != nil {
			return err
		}
		log.Printf("Outdated segment %v removed!", s)
	}
	return nil
}

func doIndex() bool {
	idxSegm, err := configs.IndexSegments().GenMaxSegment()
	if err != nil {
		log.Printf("GenMaxSegment failed: %v", err)
		return false
	}

	runtime.GC()
	utils.DumpMemStats()

	log.Printf("Indexing to %v ...", idxSegm)

	fpDocDB := configs.DocsDBFsPath()
	ts, err := gcse.Index(kv.DirInput(fpDocDB), string(idxSegm))
	if err != nil {
		log.Printf("Indexing failed: %v", err)
		return false
	}

	if !func() bool {
		f, err := os.Create(idxSegm.Join(gcse.IndexFn))
		if err != nil {
			log.Printf("Create index file failed: %v", err)
			return false
		}
		defer f.Close()

		log.Printf("Saving index to %v ...", idxSegm)
		if err := ts.Save(f); err != nil {
			log.Printf("ts.Save failed: %v", err)
			return false
		}
		return true
	}() {
		return false
	}
	runtime.GC()
	utils.DumpMemStats()

	storePath := idxSegm.Join(configs.FnStore)
	log.Printf("Saving store snapshot to %v", storePath)
	if err := store.SaveSnapshot(storePath); err != nil {
		log.Printf("SaveSnapshot %v failed: %v", storePath, err)
	}

	if err := idxSegm.Done(); err != nil {
		log.Printf("segm.Done failed: %v", err)
		return false
	}

	log.Printf("Indexing success: %s (%d)", idxSegm, ts.DocCount())
	gcse.AddBiValueAndProcess(bi.Average, "index.doc-count", ts.DocCount())

	ts = nil
	utils.DumpMemStats()
	runtime.GC()
	utils.DumpMemStats()

	return true
}

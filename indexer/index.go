package main

import (
	"github.com/daviddengcn/gcse"
	"log"
	"runtime"
)

func clearOutdatedIndex() error {
	segm, err := gcse.IndexSegments.FindMaxDone()
	if err != nil {
		return err
	}
	all, err := gcse.IndexSegments.ListAll()
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
		log.Printf("Segment %v deleted", s)
	}

	return nil
}

func doIndex(dbSegm gcse.Segment) bool {
	idxSegm, err := gcse.IndexSegments.GenMaxSegment()
	if err != nil {
		log.Printf("GenMaxSegment failed: %v", err)
		return false
	}

	runtime.GC()
	gcse.DumpMemStats()
	log.Printf("Reading docDB from %v ...", dbSegm)
	// read docDB
	docDB := gcse.PackedDocDB{gcse.NewMemDB(dbSegm.Join(""), gcse.KindDocDB)}

	log.Printf("Indexing to %v ...", idxSegm)

	ts, err := gcse.Index(docDB)
	if err != nil {
		log.Printf("Indexing failed: %v", err)
		return false
	}

	f, err := idxSegm.Join(gcse.IndexFn).Create()
	if err != nil {
		log.Printf("Create index file failed: %v", err)
		return false
	}
	defer f.Close()
	if err := ts.Save(f); err != nil {
		log.Printf("ts.Save failed: %v", err)
		return false
	}

	if err := idxSegm.Done(); err != nil {
		log.Printf("segm.Done failed: %v", err)
		return false
	}

	log.Printf("Indexing success: %s (%d)", idxSegm, ts.DocCount())

	docDB.MemDB, ts = nil, nil
	gcse.DumpMemStats()
	runtime.GC()
	gcse.DumpMemStats()

	if err := dbSegm.Remove(); err != nil {
		log.Printf("Delete segment %v failed: %v", dbSegm, err)
	}

	return true
}

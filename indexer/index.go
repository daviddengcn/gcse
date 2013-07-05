package main

import (
	"errors"
	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/go-index"
	"github.com/daviddengcn/go-villa"
	"log"
	"runtime"
	"time"
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

var errNotDocInfo = errors.New("Value is not DocInfo")

func doIndex(dbSegm gcse.Segment) {
	idxSegm, err := gcse.IndexSegments.GenMaxSegment()
	if err != nil {
		log.Printf("GenMaxSegment failed: %v", err)
		return
	}
	
	runtime.GC()
	gcse.DumpMemStats()
	log.Printf("Reading docDB from %v ...", dbSegm)
	// read docDB
	docDB := gcse.NewMemDB(dbSegm.Join(""), gcse.KindDocDB)
	
	gcse.DumpMemStats()
	log.Printf("Generating importsDB ...")
	importsDB := gcse.NewTokenIndexer("", "")
	// generate importsDB
	if err := docDB.Iterate(func(pkg string, val interface{}) error {
		docInfo, ok := val.(gcse.DocInfo)
		if !ok {
			return errNotDocInfo
		}
		importsDB.Put(pkg, villa.NewStrSet(docInfo.Imports...))
		return nil
	}); err != nil {
		log.Printf("Making importsDB failed: %v", err)
		return
	}
	
	gcse.DumpMemStats()
	log.Printf("Indexing to %v ...", idxSegm)

	ts := &index.TokenSetSearcher{}
	if err := docDB.Iterate(func(key string, val interface{}) error {
		var hitInfo gcse.HitInfo

		var ok bool
		hitInfo.DocInfo, ok = val.(gcse.DocInfo)
		if !ok {
			return errNotDocInfo
		}
		hitInfo.Imported = importsDB.IdsOfToken(hitInfo.Package)

		hitInfo.StaticScore = gcse.CalcStaticScore(&hitInfo)

		var tokens villa.StrSet
		tokens = gcse.AppendTokens(tokens, hitInfo.Name)
		tokens = gcse.AppendTokens(tokens, hitInfo.Package)
		tokens = gcse.AppendTokens(tokens, hitInfo.Description)
		tokens = gcse.AppendTokens(tokens, hitInfo.ReadmeData)
		tokens = gcse.AppendTokens(tokens, hitInfo.Author)

		ts.AddDoc(map[string]villa.StrSet{
			"text": tokens,
			"pkg":  villa.NewStrSet(hitInfo.Package),
		}, hitInfo)
		return nil
	}); err != nil {
		log.Printf("Iterate failed: %v", err)
		return
	}

	f, err := idxSegm.Join(gcse.IndexFn).Create()
	if err != nil {
		log.Printf("Create index file failed: %v", err)
		return
	}
	defer f.Close()
	if err := ts.Save(f); err != nil {
		log.Printf("ts.Save failed: %v", err)
		return
	}

	if err := idxSegm.Done(); err != nil {
		log.Printf("segm.Done failed: %v", err)
		return
	}

	log.Printf("Indexing success: %s (%d)", idxSegm, ts.DocCount())
	
	docDB, importsDB, ts = nil, nil, nil
	gcse.DumpMemStats()
	runtime.GC()
	gcse.DumpMemStats()
	
	if err := dbSegm.Remove(); err != nil {
		log.Printf("Delete segment %v failed: %v", dbSegm, err)
	}
}

func indexLoop(gap time.Duration) {
	for {
		if err := gcse.IndexSegments.ClearUndones(); err != nil {
			log.Printf("ClearUndones failed: %v", err)
		}
		dbSegm, err := gcse.DBOutSegments.FindMaxDone()
		if err == nil && dbSegm != nil {
			if err := clearOutdatedIndex(); err != nil {
				log.Printf("clearOutdatedIndex failed: %v", err)
			}
			doIndex(dbSegm)
		}

		time.Sleep(gap)
	}
}

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

func needIndex() bool {
	dones, err := gcse.IndexSegments.ListDones()
	if err != nil {
		log.Printf("ListDones failed: %v", err)
		return false
	}

	if len(dones) == 0 {
		log.Printf("To generate first index...")
		return true
	}

	maxDone, err := gcse.IndexSegments.FindMaxDone()
	if err != nil {
		log.Printf("FindMaxDone failed: %v", err)
		return false
	}

	fn := maxDone.Join(gcse.IndexFn)
	fi, err := fn.Stat()
	if err != nil {
		log.Printf("fn.Stat failed: %v", err)
		return true
	}
	_ = fi.ModTime()

	return true
}

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

func doIndex() {
	dumpMemStats()
	segm, err := gcse.IndexSegments.GenMaxSegment()
	if err != nil {
		log.Printf("GenMaxSegment failed: %v", err)
		return
	}
	log.Printf("Indexing to %v ...", segm)
	runtime.GC()
	dumpMemStats()

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

	f, err := segm.Join(gcse.IndexFn).Create()
	if err != nil {
		log.Printf("Create index file failed: %v", err)
		return
	}
	defer f.Close()
	if err := ts.Save(f); err != nil {
		log.Printf("ts.Save failed: %v", err)
		return
	}

	ts = nil

	if err := segm.Done(); err != nil {
		log.Printf("segm.Done failed: %v", err)
		return
	}

	log.Printf("Indexing success: %s", segm)
	dumpMemStats()
}

func indexLooop(gap time.Duration) {
	for {
		if err := gcse.IndexSegments.ClearUndones(); err != nil {
			log.Printf("ClearUndones failed: %v", err)
		}
		if needIndex() {
			if err := clearOutdatedIndex(); err != nil {
				log.Printf("clearOutdatedIndex failed: %v", err)
			}
			syncDatabases()
			doIndex()
		}

		time.Sleep(gap)
	}
}

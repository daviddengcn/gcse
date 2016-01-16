package main

import (
	"encoding/gob"
	"errors"
	"log"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/golangplus/bytes"
	"github.com/golangplus/strings"

	"github.com/boltdb/bolt"
	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/go-index"
)

var (
	databaseValue atomic.Value
	indexSegment  gcse.Segment
)

type database interface {
	PackageCount() int
	ProjectCount() int
	IndexUpdated() time.Time
	Close()

	FindFullPackage(id string) (hit gcse.HitInfo, found bool)
	ForEachFullPackage(func(gcse.HitInfo) error) error
	PackageCountOfToken(field, token string) int
	Search(q map[string]stringsp.Set, out func(docID int32, data interface{}) error) error
}

type searcherDB struct {
	ts   index.TokenSetSearcher
	hits *bolt.DB

	projectCount int
	indexUpdated time.Time
}

func (db *searcherDB) PackageCount() int {
	if db == nil {
		return 0
	}
	return db.ts.DocCount()
}

func (db *searcherDB) ProjectCount() int {
	if db == nil {
		return 0
	}
	return db.projectCount
}

func (db *searcherDB) IndexUpdated() time.Time {
	if db == nil {
		return time.Now()
	}
	return db.indexUpdated
}

func (db *searcherDB) Close() {
	if db == nil {
		return
	}
	db.hits.Close()
}

var notFoundInHits = errors.New("Not found in hits")

func (db *searcherDB) FindFullPackage(id string) (gcse.HitInfo, bool) {
	if db == nil {
		return gcse.HitInfo{}, false
	}
	var hit gcse.HitInfo
	if err := db.hits.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(gcse.IndexHitsBucket))
		if b == nil {
			return notFoundInHits
		}
		bs := bytesp.Slice(b.Get([]byte(id)))
		if bs == nil {
			return notFoundInHits
		}
		return gob.NewDecoder(&bs).Decode(&hit)
	}); err != nil {
		return gcse.HitInfo{}, false
	}
	return hit, true
}

func (db *searcherDB) ForEachFullPackage(out func(gcse.HitInfo) error) error {
	if db == nil {
		return nil
	}
	return db.hits.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(gcse.IndexHitsBucket))
		if b == nil {
			return nil
		}
		return b.ForEach(func(_, v []byte) error {
			if v == nil {
				// Skip sub bucket if any.
				return nil
			}
			bs := bytesp.Slice(v)
			var hit gcse.HitInfo
			if err := gob.NewDecoder(&bs).Decode(&hit); err != nil {
				return err
			}
			return out(hit)
		})
	})
}

func (db *searcherDB) PackageCountOfToken(field, token string) int {
	if db == nil {
		return 0
	}
	return len(db.ts.TokenDocList(field, token))
}

func (db *searcherDB) Search(q map[string]stringsp.Set, out func(docID int32, data interface{}) error) error {
	if db == nil {
		return nil
	}
	return db.ts.Search(q, out)
}

func getDatabase() database {
	db, ok := databaseValue.Load().(database)
	if !ok {
		return (*searcherDB)(nil)
	}
	return db
}

func loadIndex() error {
	segm, err := gcse.IndexSegments.FindMaxDone()
	if segm == nil || err != nil {
		return err
	}

	if indexSegment != nil && !gcse.SegmentLess(indexSegment, segm) {
		// no new index
		return nil
	}

	db := &searcherDB{}
	if err := func() error {
		f, err := segm.Join(gcse.IndexFn).Open()
		if err != nil {
			return err
		}
		defer f.Close()

		return db.ts.Load(f)
	}(); err != nil {
		return err
	}
	wholeInfoPath := segm.Join(gcse.WholeInfoFn)
	if db.hits, err = bolt.Open(wholeInfoPath.S(), 0666, &bolt.Options{ReadOnly: true}); err != nil {
		log.Printf("bolt.Open %v failed: %v", wholeInfoPath, err)
		return err
	}
	// Calculate db.projectCount
	var projects stringsp.Set
	db.ts.Search(nil, func(docID int32, data interface{}) error {
		hit := data.(gcse.HitInfo)
		projects.Add(hit.ProjectURL)
		return nil
	})
	db.projectCount = len(projects)

	// Update db.indexUpdated
	db.indexUpdated = time.Now()
	if st, err := segm.Join(gcse.IndexFn).Stat(); err == nil {
		db.indexUpdated = st.ModTime()
	}

	indexSegment = segm
	log.Printf("Load index from %v (%d packages)", segm, db.PackageCount())

	// Exchange new/old database and close the old one.
	oldDB := getDatabase()
	databaseValue.Store(db)
	oldDB.Close()
	oldDB = nil
	gcse.DumpMemStats()

	runtime.GC()
	gcse.DumpMemStats()

	return nil
}

func loadIndexLoop() {
	for {
		time.Sleep(30 * time.Second)

		if err := loadIndex(); err != nil {
			log.Printf("loadIndex failed: %v", err)
		}
	}
}

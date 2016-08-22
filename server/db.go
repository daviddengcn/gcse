package main

import (
	"errors"
	"log"
	"os"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/golangplus/strings"

	"github.com/daviddengcn/bolthelper"
	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/gcse/configs"
	"github.com/daviddengcn/gcse/store"
	"github.com/daviddengcn/gcse/utils"
	"github.com/daviddengcn/go-easybi"
	"github.com/daviddengcn/go-index"

	sppb "github.com/daviddengcn/gcse/proto/spider"
)

var (
	databaseValue atomic.Value
	indexSegment  utils.Segment
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
	PackageCrawlHistory(pkg string) *sppb.HistoryInfo
}

type searcherDB struct {
	ts   index.TokenSetSearcher
	hits *index.ConstArrayReader

	projectCount int
	indexUpdated time.Time

	storeDB *bh.RefCountBox
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
		log.Print("Database not loaded!")
		return gcse.HitInfo{}, false
	}
	var hit gcse.HitInfo
	found := false
	if err := db.ts.Search(index.SingleFieldQuery(gcse.IndexPkgField, id), func(docID int32, _ interface{}) error {
		h, err := db.hits.GetGob(int(docID))
		if err != nil {
			return err
		}
		hit = h.(gcse.HitInfo)
		found = true
		return nil
	}); err != nil {
		return gcse.HitInfo{}, false
	}
	if !found {
		return gcse.HitInfo{}, false
	}
	return hit, true
}

func (db *searcherDB) ForEachFullPackage(out func(gcse.HitInfo) error) error {
	if db == nil {
		return nil
	}
	return db.hits.ForEachGob(func(_ int, hit interface{}) error {
		return out(hit.(gcse.HitInfo))
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

func (db *searcherDB) PackageCrawlHistory(pkg string) *sppb.HistoryInfo {
	site, path := utils.SplitPackage(pkg)
	info, err := store.ReadPackageHistoryOf(db.storeDB, site, path)
	if err != nil {
		log.Printf("ReadPackageHistoryOf %s %s failed: %v", site, path, err)
		return nil
	}
	return info
}

func getDatabase() database {
	db, ok := databaseValue.Load().(database)
	if !ok {
		return (*searcherDB)(nil)
	}
	return db
}

func loadIndex() error {
	segm, err := configs.IndexSegments().FindMaxDone()
	if segm == "" || err != nil {
		return err
	}
	if indexSegment != "" && !utils.SegmentLess(indexSegment, segm) {
		// no new index
		return nil
	}
	db := &searcherDB{}
	if err := func() error {
		f, err := os.Open(segm.Join(gcse.IndexFn))
		if err != nil {
			return err
		}
		defer f.Close()

		return db.ts.Load(f)
	}(); err != nil {
		return err
	}
	db.storeDB = &bh.RefCountBox{
		DataPath: func() string {
			return segm.Join(configs.FnStore)
		},
	}
	hitsPath := segm.Join(gcse.HitsArrFn)
	if db.hits, err = index.OpenConstArray(hitsPath); err != nil {
		log.Printf("OpenConstArray %v failed: %v", hitsPath, err)
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
	gcse.AddBiValueAndProcess(bi.Max, "index.proj-count", db.projectCount)

	// Update db.indexUpdated
	db.indexUpdated = time.Now()
	if st, err := os.Stat(segm.Join(gcse.IndexFn)); err == nil {
		db.indexUpdated = st.ModTime()
	}
	indexSegment = segm
	log.Printf("Load index from %v (%d packages)", segm, db.PackageCount())

	// Exchange new/old database and close the old one.
	oldDB := getDatabase()
	databaseValue.Store(db)
	oldDB.Close()
	oldDB = nil
	utils.DumpMemStats()

	runtime.GC()
	utils.DumpMemStats()

	return nil
}

func loadIndexLoop() {
	for {
		time.Sleep(30 * time.Second)

		if err := loadIndex(); err != nil {
			log.Printf("loadIndex failed: %v", err)
		}
		bi.AddValue(bi.Max, "search.age_in_hours", int(time.Now().Sub(getDatabase().IndexUpdated()).Hours()))
		bi.AddValue(bi.Max, "search.age_in_mins", int(time.Now().Sub(getDatabase().IndexUpdated()).Minutes()))
	}
}

func processBi() {
	for {
		bi.Process()
		time.Sleep(time.Minute)
	}
}

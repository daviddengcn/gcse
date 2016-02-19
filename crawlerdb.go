package gcse

import (
	"log"
	"strings"
	"time"

	"github.com/daviddengcn/gddo/doc"
)

/*
 * CrawlerDB including all crawler entires database.
 */
type CrawlerDB struct {
	PackageDB *MemDB
	PersonDB  *MemDB
}

// LoadCrawlerDB loads PackageDB and PersonDB and returns a new *CrawlerDB
func LoadCrawlerDB() *CrawlerDB {
	CrawlerDBPath := DataRoot.Join(FnCrawlerDB)

	return &CrawlerDB{
		PackageDB: NewMemDB(CrawlerDBPath, KindPackage),
		PersonDB:  NewMemDB(CrawlerDBPath, KindPerson),
	}
}

// Sync syncs both PackageDB and PersonDB. Returns error if any of the sync
// failed.
func (cdb *CrawlerDB) Sync() error {
	if err := cdb.PackageDB.Sync(); err != nil {
		log.Printf("cdb.PackageDB.Sync failed: %v", err)
		return err
	}
	if err := cdb.PersonDB.Sync(); err != nil {
		log.Printf("cdb.PersonDB.Sync failed: %v", err)
		return err
	}

	return nil
}

// SchedulePackage schedules a package to be crawled at a specific time.
func (cdb *CrawlerDB) SchedulePackage(pkg string, sTime time.Time, etag string) error {
	ent := CrawlingEntry{
		ScheduleTime: sTime,
		Version:      CrawlerVersion,
		Etag:         etag,
	}

	cdb.PackageDB.Put(pkg, ent)

	//	log.Printf("Schedule package %s to %v", pkg, sTime)
	return nil
}

func TrimPackageName(pkg string) string {
	return strings.TrimFunc(strings.TrimSpace(pkg), func(r rune) bool {
		return r > rune(128)
	})
}

// AppendPackage appends a package. If the package did not exist in either
// PackageDB or Docs, schedule it (immediately).
func (cdb *CrawlerDB) AppendPackage(pkg string, inDocs func(pkg string) bool) {
	pkg = TrimPackageName(pkg)
	if !doc.IsValidRemotePath(pkg) {
		return
	}
	var ent CrawlingEntry
	exists := cdb.PackageDB.Get(pkg, &ent)
	if exists {
		if ent.ScheduleTime.After(time.Now()) || inDocs(pkg) {
			return
		}
		// if the docs is missing in Docs, schedule it earlier
		log.Printf("Scheduling a package with missing docs: %v", pkg)
	} else {
		log.Printf("Scheduling new package: %v", pkg)
	}
	cdb.SchedulePackage(pkg, time.Now(), "")
}

// SchedulePerson schedules a person to be crawled at a specific time.
func (cdb *CrawlerDB) SchedulePerson(id string, sTime time.Time) error {
	ent := CrawlingEntry{
		ScheduleTime: sTime,
		Version:      CrawlerVersion,
	}

	cdb.PersonDB.Put(id, ent)

	log.Printf("Schedule person %s to %v", id, sTime)
	return nil
}

// AppendPerson appends a person to the PersonDB, schedules to crawl
// immediately for a new person
func (cdb *CrawlerDB) AppendPerson(site, username string) bool {
	id := IdOfPerson(site, username)

	var ent CrawlingEntry
	exists := cdb.PersonDB.Get(id, &ent)
	if exists {
		// already scheduled
		return false
	}

	return cdb.SchedulePerson(id, time.Now()) == nil
}

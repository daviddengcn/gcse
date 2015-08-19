package gcse

import (
	"encoding/gob"
	"io"
	"log"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/golangplus/strings"

	"github.com/daviddengcn/go-index"
	"github.com/daviddengcn/go-villa"
)

type MemDB struct {
	db map[string]interface{}
	fn villa.Path
	sync.RWMutex
	syncMutex    sync.Mutex // if lock both mutexes, lock RWMutex first
	lastModified time.Time
	modified     bool
}

func NewMemDB(root villa.Path, kind string) *MemDB {
	mdb := &MemDB{
		db: make(map[string]interface{}),
	}

	if root != "" {
		if err := root.MkdirAll(0755); err != nil {
			log.Printf("MkdirAll failed: %v", err)
		}

		mdb.fn = root.Join(kind + ".gob")

		if err := mdb.Load(); err != nil {
			log.Printf("Load MemDB %s failed: %v", kind, err)
		}
	}

	return mdb
}

func (mdb *MemDB) Modified() bool {
	return mdb.modified
}

func (mdb *MemDB) LastModified() time.Time {
	return mdb.lastModified
}

func (mdb *MemDB) Load() error {
	if mdb.fn == "" {
		return nil
	}
	mdb.Lock()
	defer mdb.Unlock()

	lastModified := time.Now()
	if st, err := mdb.fn.Stat(); err == nil {
		lastModified = st.ModTime()
	}

	if f, err := mdb.fn.Open(); err == nil {
		defer f.Close()

		dec := gob.NewDecoder(f)
		if err := dec.Decode(&mdb.db); err != nil {
			return err
		}
	} else if os.IsNotExist(err) {
		// try recover from fn.new
		if f, err := (mdb.fn + ".new").Open(); err == nil {
			defer f.Close()

			dec := gob.NewDecoder(f)
			if err := dec.Decode(&mdb.db); err != nil {
				return err
			}
		} else if os.IsNotExist(err) {
			// just an empty db
			mdb.db = make(map[string]interface{})
		} else {
			return err
		}
	} else {
		return err
	}

	mdb.lastModified = lastModified
	mdb.modified = false
	return nil
}

// 1) save to fn.new; 2) remove fn; 3) rename fn.new to fn.
func safeSave(fn villa.Path, doSave func(w io.Writer) error) error {
	tmpFn := fn + ".new"
	if err := func() error {
		f, err := tmpFn.Create()
		if err != nil {
			return err
		}
		defer f.Close()

		return doSave(f)
	}(); err != nil {
		return err
	}

	if err := fn.Remove(); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := tmpFn.Rename(fn); err != nil {
		return err
	}

	return nil
}

func (mdb *MemDB) Sync() error {
	if mdb.fn == "" {
		// this db is not for syncing
		return nil
	}

	mdb.RLock()
	defer mdb.RUnlock()

	if !mdb.modified {
		return nil
	}

	mdb.syncMutex.Lock()
	defer mdb.syncMutex.Unlock()

	if err := safeSave(mdb.fn, func(w io.Writer) error {
		enc := gob.NewEncoder(w)
		return enc.Encode(mdb.db)
	}); err != nil {
		return err
	}

	mdb.modified = false
	return nil
}

/*
	Export saves the data to some space, but not affecting the modified property.
*/
func (mdb *MemDB) Export(root villa.Path, kind string) error {
	mdb.RLock()
	defer mdb.RUnlock()

	fn := root.Join(kind + ".gob")

	f, err := fn.Create()
	if err != nil {
		return err
	}
	defer f.Close()

	enc := gob.NewEncoder(f)
	return enc.Encode(mdb.db)
}

// Get fetches an entry of specified key. data is a pointer. Return false if not exists
func (mdb *MemDB) Get(key string, data interface{}) bool {
	mdb.RLock()
	defer mdb.RUnlock()

	vl, ok := mdb.db[key]
	if !ok {
		return false
	}
	reflect.ValueOf(data).Elem().Set(reflect.ValueOf(vl))
	return true
}

func (mdb *MemDB) Put(key string, data interface{}) {
	mdb.Lock()
	defer mdb.Unlock()

	mdb.db[key] = data
	mdb.lastModified = time.Now()
	mdb.modified = true
}

func (mdb *MemDB) Delete(key string) {
	mdb.Lock()
	defer mdb.Unlock()

	delete(mdb.db, key)
	mdb.lastModified = time.Now()
	mdb.modified = true
}

func (mdb *MemDB) Iterate(output func(key string, val interface{}) error) error {
	mdb.RLock()
	defer mdb.RUnlock()

	for k, v := range mdb.db {
		if err := output(k, v); err != nil {
			return err
		}
	}

	return nil
}

// Count returns the number of entries in the DB
func (mdb *MemDB) Count() int {
	mdb.RLock()
	defer mdb.RUnlock()

	return len(mdb.db)
}

// TokenIndexer is thread-safe.
type TokenIndexer struct {
	index.TokenIndexer
	fn villa.Path
	sync.RWMutex
	syncMutex    sync.Mutex
	lastModified time.Time
	modified     bool
}

func NewTokenIndexer(root villa.Path, kind string) *TokenIndexer {
	ti := &TokenIndexer{}

	if root != "" {
		if err := root.MkdirAll(0755); err != nil {
			log.Printf("MkdirAll failed: %v", err)
		}
		ti.fn = root.Join(kind + ".gob")
		if err := ti.Load(); err != nil {
			log.Printf("Load MemDB failed: %v", err)
			// its ok loading failed
		}
	}
	return ti
}

func (ti *TokenIndexer) Modified() bool {
	return ti.modified
}

func (ti *TokenIndexer) LastModified() time.Time {
	return ti.lastModified
}

func (ti *TokenIndexer) Load() error {
	ti.Lock()
	defer ti.Unlock()

	lastModified := time.Now()
	st, err := ti.fn.Stat()
	if err == nil {
		lastModified = st.ModTime()
	}

	f, err := ti.fn.Open()
	if err == os.ErrNotExist {
		return nil
	}

	if err != nil {
		return err
	}
	defer f.Close()

	if err := ti.TokenIndexer.Load(f); err != nil {
		return err
	}

	ti.lastModified = lastModified
	ti.modified = false
	return nil
}

func (ti *TokenIndexer) Sync() error {
	ti.RLock()
	defer ti.RUnlock()

	ti.syncMutex.Lock()
	defer ti.syncMutex.Unlock()

	if !ti.modified {
		return nil
	}

	if err := safeSave(ti.fn, ti.TokenIndexer.Save); err != nil {
		return err
	}

	ti.modified = false
	return nil
}

func (ti *TokenIndexer) Export(root villa.Path, kind string) error {
	fn := root.Join(kind + ".gob")

	ti.RLock()
	defer ti.RUnlock()

	f, err := fn.Create()
	if err != nil {
		return err
	}
	defer f.Close()

	return ti.TokenIndexer.Save(f)
}

func (ti *TokenIndexer) Put(id string, tokens stringsp.Set) {
	ti.Lock()
	defer ti.Unlock()

	ti.TokenIndexer.PutTokens(id, tokens)
	ti.lastModified = time.Now()
	ti.modified = true
}

func (ti *TokenIndexer) IdsOfToken(token string) []string {
	ti.RLock()
	defer ti.RUnlock()

	return ti.TokenIndexer.IdsOfToken(token)
}

func (ti *TokenIndexer) TokensOfId(id string) []string {
	ti.RLock()
	defer ti.RUnlock()

	return ti.TokenIndexer.TokensOfId(id)
}

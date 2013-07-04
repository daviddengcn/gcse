package gcse

import (
	"encoding/gob"
	"github.com/daviddengcn/go-index"
	"github.com/daviddengcn/go-villa"
	"io"
	"log"
	"os"
	"reflect"
	"sync"
)

type MemDB struct {
	db map[string]interface{}
	fn villa.Path
	sync.RWMutex
	syncMutex sync.Mutex // if lock both mutexes, lock RWMutex first
	modified  bool
}

func NewMemDB(root villa.Path, kind string) *MemDB {
	mdb := &MemDB{
		db: make(map[string]interface{}),
	}

	if len(kind) != 0 {
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

func (mdb *MemDB) Load() error {
	if mdb.fn == "" {
		return nil
	}
	mdb.Lock()
	defer mdb.Unlock()

	f, err := mdb.fn.Open()
	if err == os.ErrNotExist {
		return nil
	}

	if err != nil {
		return err
	}
	defer f.Close()

	dec := gob.NewDecoder(f)
	if err := dec.Decode(&mdb.db); err != nil {
		return err
	}

	mdb.modified = false
	return nil
}

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

	if err := fn.Remove(); err != nil {
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

func (mdb *MemDB) Export(fn villa.Path) error {
	mdb.RLock()
	defer mdb.RUnlock()

	f, err := fn.Create()
	if err != nil {
		return err
	}
	defer f.Close()

	enc := gob.NewEncoder(f)
	return enc.Encode(mdb.db)
}

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
	mdb.modified = true
}

func (mdb *MemDB) Delete(key string) {
	mdb.Lock()
	defer mdb.Unlock()

	delete(mdb.db, key)
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

type TokenIndexer struct {
	index.TokenIndexer
	fn villa.Path
	sync.RWMutex
	syncMutex sync.Mutex
	modified  bool
}

func NewTokenIndexer(root villa.Path, kind string) *TokenIndexer {
	if err := root.MkdirAll(0755); err != nil {
		log.Printf("MkdirAll failed: %v", err)
	}

	ti := &TokenIndexer{
		fn: root.Join(kind + ".gob"),
	}

	if err := ti.Load(); err != nil {
		log.Printf("Load MemDB failed: %v", err)
	}
	return ti
}

func (ti *TokenIndexer) Load() error {
	ti.Lock()
	defer ti.Unlock()

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

func (ti *TokenIndexer) Export(fn villa.Path) error {
	ti.RLock()
	defer ti.RUnlock()

	f, err := fn.Create()
	if err != nil {
		return err
	}
	defer f.Close()

	return ti.TokenIndexer.Save(f)
}

func (ti *TokenIndexer) Put(id string, tokens villa.StrSet) {
	ti.Lock()
	defer ti.Unlock()

	ti.TokenIndexer.Put(id, tokens)
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

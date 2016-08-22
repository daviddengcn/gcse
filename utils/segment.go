package utils

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/golangplus/errors"
	"github.com/golangplus/strings"
)

const (
	fnDone = ".done"
)

type Segment string

func (s Segment) Make() error {
	return os.MkdirAll(string(s), 0755)
}

func (s Segment) Name() string {
	return filepath.Base(string(s))
}

func (s Segment) Join(name string) string {
	if name == "" {
		return string(s)
	}
	return filepath.Join(string(s), name)
}

func (s Segment) IsDone() bool {
	_, err := os.Stat(s.Join(fnDone))
	return err == nil
}

func (s Segment) Done() error {
	if err := os.MkdirAll(string(s), 0755); err != nil {
		return err
	}
	f, err := os.Create(s.Join(fnDone))
	if err != nil {
		return errorsp.WithStacks(err)
	}
	return errorsp.WithStacks(f.Close())
}

func (s Segment) ListFiles() ([]string, error) {
	files, err := ioutil.ReadDir(string(s))
	if err != nil {
		return nil, errorsp.WithStacks(err)
	}
	list := make([]string, 0, len(files))
	for _, f := range files {
		if f.Name() == fnDone {
			continue
		}
		list = append(list, filepath.Join(string(s), f.Name()))
	}
	return list, nil
}

func (s Segment) Remove() error {
	return errorsp.WithStacks(os.RemoveAll(string(s)))
}

type Segments string

func (ss Segments) Join(sub string) Segment {
	return Segment(filepath.Join(string(ss), sub))
}

func (ss Segments) ListAll() ([]Segment, error) {
	files, err := ioutil.ReadDir(string(ss))
	if err != nil {
		if os.IsNotExist(errorsp.Cause(err)) {
			// Returns empty slice if the folder does not exist.
			return nil, nil
		}
		return nil, errorsp.WithStacks(err)
	}
	segms := make([]Segment, 0, len(files))
	for _, f := range files {
		if !f.IsDir() {
			// A segment is always a folder.
			continue
		}
		segms = append(segms, ss.Join(f.Name()))
	}
	return segms, nil
}

func (ss Segments) ListDones() ([]Segment, error) {
	segms, err := ss.ListAll()
	if err != nil {
		return nil, err
	}
	dones := make([]Segment, 0, len(segms))
	for _, s := range segms {
		if s.IsDone() {
			dones = append(dones, s)
		}
	}
	return dones, nil
}

func SegmentLess(a, b Segment) bool {
	numA, errA := strconv.Atoi(a.Name())
	numB, errB := strconv.Atoi(b.Name())

	if errA != nil {
		if errB != nil {
			// both non-numbers
			return a.Name() < b.Name()
		}
		// non < number
		return true
	}
	if errB != nil {
		// number > non
		return false
	}
	// number comparison
	return numA < numB
}

func (ss Segments) FindMaxDone() (Segment, error) {
	var maxS Segment
	dones, err := ss.ListDones()
	if err != nil {
		return "", errorsp.WithStacks(err)
	}
	for _, s := range dones {
		if maxS == "" || SegmentLess(maxS, s) {
			maxS = s
		}
	}
	return maxS, nil
}

func makeSegment(s Segment) (Segment, error) {
	return s, os.MkdirAll(string(s), 0755)
}

func (ss Segments) GenNewSegment() (Segment, error) {
	curSs, err := ss.ListAll()
	if err != nil {
		return "", errorsp.WithStacks(err)
	}

	var nset stringsp.Set
	for _, s := range curSs {
		nset.Add(s.Name())
	}

	for i := 0; ; i++ {
		fn := strconv.Itoa(i)
		if nset.Contain(fn) {
			continue
		}
		return makeSegment(ss.Join(fn))
	}
}

func (ss Segments) GenMaxSegment() (Segment, error) {
	var maxS Segment
	dones, err := ss.ListAll()
	if err != nil {
		return "", errorsp.WithStacks(err)
	}
	for _, s := range dones {
		if maxS == "" || SegmentLess(maxS, s) {
			maxS = s
		}
	}
	if maxS == "" {
		return makeSegment(ss.Join("0"))
	}
	num, err := strconv.Atoi(maxS.Name())
	if err != nil {
		return makeSegment(ss.Join("0"))
	}
	return makeSegment(ss.Join(strconv.Itoa(num + 1)))
}

func (ss Segments) ClearUndones() error {
	segms, err := ss.ListAll()
	if err != nil {
		return err
	}
	for _, segm := range segms {
		if !segm.IsDone() {
			if err := segm.Remove(); err != nil {
				return err
			}
			log.Printf("Undone segment %v is removed!", segm)
		}
	}
	return nil
}

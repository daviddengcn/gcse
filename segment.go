package gcse

import (
	"fmt"
	"github.com/daviddengcn/go-villa"
	"strconv"
	"github.com/howeyc/fsnotify"
)

const (
	fnDone = ".done"
)

type Segment interface {
	Name() string
	Join(name string) villa.Path
	IsDone() bool
	Done() error
	ListFiles() ([]villa.Path, error)
	Remove() error
}

type segment villa.Path

func (s segment) Name() string {
	return string(villa.Path(s).Base())
}

func (s segment) Join(name string) villa.Path {
	if name == "" {
		return villa.Path(s)
	}
	return villa.Path(s).Join(name)
}

func (s segment) IsDone() bool {
	return villa.Path(s).Join(fnDone).Exists()
}

func (s segment) Done() error {
	f, err := villa.Path(s).Join(fnDone).Create()
	if err != nil {
		return err
	}
	return f.Close()
}

func (s segment) ListFiles() ([]villa.Path, error) {
	files, err := villa.Path(s).ReadDir()
	if err != nil {
		return nil, err
	}

	list := make([]villa.Path, 0, len(files))
	for _, f := range files {
		if f.Name() == fnDone {
			continue
		}

		list = append(list, villa.Path(s).Join(f.Name()))
	}

	return list, nil
}

func (s segment) Remove() error {
	return villa.Path(s).RemoveAll()
}

type Segments interface {
	Watch(watcher *fsnotify.Watcher) error
	ListAll() ([]Segment, error)
	// all done
	ListDones() ([]Segment, error)
	// max done
	FindMaxDone() (Segment, error)
	// generates an arbitrary new segment
	GenNewSegment() (Segment, error)
	// generates a segment greated than all existence
	GenMaxSegment() (Segment, error)
	// clear
	ClearUndones() error
}

type segments villa.Path

func newSegment(path villa.Path) segment {
	path.MkdirAll(0755)
	return segment(path)
}

func (s segments) Watch(watcher *fsnotify.Watcher) error {
	return watcher.Watch(string(s))
}

func (s segments) ListAll() ([]Segment, error) {
	files, err := villa.Path(s).ReadDir()
	if err != nil {
		return nil, err
	}

	segments := make([]Segment, 0, len(files))
	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		segments = append(segments, segment(villa.Path(s).Join(f.Name())))
	}

	return segments, nil
}

func (s segments) ListDones() ([]Segment, error) {
	segments, err := s.ListAll()
	if err != nil {
		return nil, err
	}
	dones := make([]Segment, 0, len(segments))
	for _, s := range segments {
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

func (s segments) FindMaxDone() (Segment, error) {
	var maxS Segment
	dones, err := s.ListDones()
	if err != nil {
		return nil, err
	}
	for _, s := range dones {
		if maxS == nil || SegmentLess(maxS, s) {
			maxS = s
		}
	}

	return maxS, nil
}

func (s segments) GenNewSegment() (Segment, error) {
	curSs, err := s.ListAll()
	if err != nil {
		return nil, err
	}

	var nset villa.StrSet
	for _, s := range curSs {
		nset.Put(s.Name())
	}

	for i := 0; ; i++ {
		fn := fmt.Sprintf("%d", i)
		if nset.In(fn) {
			continue
		}

		path := ImportPath.Join(fn)
		path.MkdirAll(0755)
		return newSegment(path), nil
	}
	// won't happend
	return nil, nil
}

func (s segments) GenMaxSegment() (Segment, error) {
	var maxS Segment
	dones, err := s.ListDones()
	if err != nil {
		return nil, err
	}
	for _, s := range dones {
		if maxS == nil || SegmentLess(maxS, s) {
			maxS = s
		}
	}

	if maxS == nil {
		return newSegment(villa.Path(s).Join("0")), nil
	}

	num, err := strconv.Atoi(maxS.Name())
	if err != nil {
		return newSegment(villa.Path(s).Join("0")), nil
	}

	return newSegment(villa.Path(s).Join(strconv.Itoa(num + 1))), nil
}

func (s segments) ClearUndones() error {
	segms, err := s.ListAll()
	if err != nil {
		return err
	}
	for _, s := range segms {
		if !s.IsDone() {
			if err := s.Remove(); err != nil {
				return err
			}
		}
	}

	return nil
}

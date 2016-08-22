package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/golangplus/testing/assert"
)

func TestSegment(t *testing.T) {
	const (
		name   = "1"
		dataFn = "data.txt"
		subDir = "sub"
	)

	path := filepath.Join(os.TempDir(), name)
	assert.NoErrorOrDie(t, os.RemoveAll(path))

	s := Segment(path)
	assert.Equal(t, "name", s.Name(), name)
	assert.Equal(t, "join", s.Join(""), path)
	assert.Equal(t, "join", s.Join(dataFn), filepath.Join(path, dataFn))
	assert.False(t, "is-done", s.IsDone())
	assert.NoError(t, s.Done())
	assert.True(t, "is-done", s.IsDone())

	// Check ListFiles returns sub directories.
	assert.NoError(t, Segment(s.Join(subDir)).Make())
	files, err := s.ListFiles()
	assert.NoError(t, err)
	assert.Equal(t, "files", files, []string{s.Join(subDir)})
}

func TestSegments(t *testing.T) {
	path := filepath.Join(os.TempDir(), "TestSegments")
	assert.NoErrorOrDie(t, os.RemoveAll(path))

	ss := Segments(path)
	s0, err := ss.GenNewSegment()
	assert.NoError(t, err)
	assert.Equal(t, "s0", s0, Segment(filepath.Join(path, "0")))
	assert.NoError(t, s0.Done())

	s1, err := ss.GenNewSegment()
	assert.NoError(t, err)
	assert.Equal(t, "s1", s1, Segment(filepath.Join(path, "1")))

	// Create a file under path, should not be returned by ListAll()
	f, err := os.Create(filepath.Join(path, "a.txt"))
	assert.NoError(t, err)
	assert.NoError(t, f.Close())
	sa, err := ss.ListAll()
	assert.NoError(t, err)
	assert.Equal(t, "sa", sa, []Segment{s0, s1})

	sa, err = ss.ListDones()
	assert.NoError(t, err)
	assert.Equal(t, "sa", sa, []Segment{s0})

	s2, err := ss.GenMaxSegment()
	assert.NoError(t, err)
	assert.Equal(t, "s2", s2, ss.Join("2"))

	ms, err := ss.FindMaxDone()
	assert.NoError(t, err)
	assert.Equal(t, "ms", ms, ss.Join("0"))

	assert.NoError(t, s2.Done())
	ms, err = ss.FindMaxDone()
	assert.NoError(t, err)
	assert.Equal(t, "ms", ms, ss.Join("2"))

	assert.NoError(t, ss.ClearUndones())
	sa, err = ss.ListAll()
	assert.NoError(t, err)
	assert.Equal(t, "sa", sa, []Segment{s0, s2})
}

func TestSegments_GenMaxSegment(t *testing.T) {
	path := filepath.Join(os.TempDir(), "TestSegments_GenMaxSegment")
	assert.NoErrorOrDie(t, os.RemoveAll(path))
	assert.NoErrorOrDie(t, os.MkdirAll(path, 0755))

	ss := Segments(path)

	s, err := ss.GenMaxSegment()
	assert.NoError(t, err)
	assert.Equal(t, "s", s, ss.Join("0"))
	assert.NoError(t, s.Remove())

	assert.NoError(t, os.MkdirAll(filepath.Join(path, "word"), 0755))
	s, err = ss.GenMaxSegment()
	assert.NoError(t, err)
	assert.Equal(t, "s", s, ss.Join("0"))
	assert.NoError(t, s.Remove())
}

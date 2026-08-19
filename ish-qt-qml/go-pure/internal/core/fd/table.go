// Package fd provides the guest file-descriptor table used by the Go kernel.
// It intentionally stores interfaces rather than OS handles, so the same table
// works with fakefs files, pipes, bytes.Buffer values, and a host PTY bridge.
package fd

import (
	"errors"
	"io"
	"sync"
)

var (
	ErrBadFD    = errors.New("fd: bad file descriptor")
	ErrNotSeek  = errors.New("fd: descriptor is not seekable")
	ErrOccupied = errors.New("fd: descriptor is already occupied")
)

type File struct {
	Reader io.Reader
	Writer io.Writer
	Closer io.Closer
	Seeker io.Seeker
}

func (f *File) Close() error {
	if f == nil {
		return ErrBadFD
	}
	if f.Closer == nil {
		return nil
	}
	return f.Closer.Close()
}

func (f *File) Read(p []byte) (int, error) {
	if f == nil || f.Reader == nil {
		return 0, ErrBadFD
	}
	return f.Reader.Read(p)
}

func (f *File) Write(p []byte) (int, error) {
	if f == nil || f.Writer == nil {
		return 0, ErrBadFD
	}
	return f.Writer.Write(p)
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	if f == nil || f.Seeker == nil {
		return 0, ErrNotSeek
	}
	return f.Seeker.Seek(offset, whence)
}

type Table struct {
	mu      sync.RWMutex
	entries map[int32]*File
	next    int32
}

func New() *Table {
	return &Table{entries: make(map[int32]*File), next: 3}
}

func (t *Table) InstallAt(fd int32, file *File, replace bool) error {
	if t == nil || fd < 0 || file == nil {
		return ErrBadFD
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.entries == nil {
		t.entries = make(map[int32]*File)
	}
	if _, exists := t.entries[fd]; exists && !replace {
		return ErrOccupied
	}
	t.entries[fd] = file
	if fd >= t.next {
		t.next = fd + 1
	}
	return nil
}

func (t *Table) Open(file *File) (int32, error) {
	if t == nil || file == nil {
		return -1, ErrBadFD
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.entries == nil {
		t.entries = make(map[int32]*File)
	}
	for {
		if _, exists := t.entries[t.next]; !exists {
			fd := t.next
			t.entries[fd] = file
			t.next++
			return fd, nil
		}
		t.next++
	}
}

func (t *Table) Get(fd int32) (*File, error) {
	if t == nil || fd < 0 {
		return nil, ErrBadFD
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	file := t.entries[fd]
	if file == nil {
		return nil, ErrBadFD
	}
	return file, nil
}

func (t *Table) Close(fd int32) error {
	if t == nil || fd < 0 {
		return ErrBadFD
	}
	t.mu.Lock()
	file := t.entries[fd]
	if file == nil {
		t.mu.Unlock()
		return ErrBadFD
	}
	delete(t.entries, fd)
	t.mu.Unlock()
	return file.Close()
}

func (t *Table) Dup(fd int32) (int32, error) {
	file, err := t.Get(fd)
	if err != nil {
		return -1, err
	}
	return t.Open(file)
}

func (t *Table) Dup2(oldfd, newfd int32) (int32, error) {
	file, err := t.Get(oldfd)
	if err != nil || newfd < 0 {
		return -1, ErrBadFD
	}
	if oldfd == newfd {
		return newfd, nil
	}
	if existing, getErr := t.Get(newfd); getErr == nil {
		_ = existing.Close()
	}
	if err := t.InstallAt(newfd, file, true); err != nil {
		return -1, err
	}
	return newfd, nil
}

func (t *Table) Count() int {
	if t == nil {
		return 0
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.entries)
}

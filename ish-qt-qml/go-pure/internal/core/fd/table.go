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
	Reader  io.Reader
	Writer  io.Writer
	Closer  io.Closer
	Seeker  io.Seeker
	Poll    func(events uint16) uint16
	Cloexec bool

	refMu  sync.Mutex
	refs   int
	closed bool

	// Path is the guest absolute path for descriptors opened through fakefs.
	// It is empty for pipes, console streams, and other anonymous handles.
	Path string

	// DirPos is the guest directory-stream cookie used by getdents64.
	DirPos int
}

func (f *File) retain() {
	if f == nil {
		return
	}
	f.refMu.Lock()
	if !f.closed {
		f.refs++
	}
	f.refMu.Unlock()
}

func (f *File) Close() error {
	if f == nil {
		return ErrBadFD
	}
	f.refMu.Lock()
	if f.refs > 0 {
		f.refs--
	}
	if f.refs != 0 || f.closed {
		f.refMu.Unlock()
		return nil
	}
	f.closed = true
	closer := f.Closer
	f.refMu.Unlock()
	if closer == nil {
		return nil
	}
	return closer.Close()
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
	if t.entries == nil {
		t.entries = make(map[int32]*File)
	}
	old := t.entries[fd]
	if old != nil && !replace {
		t.mu.Unlock()
		return ErrOccupied
	}
	file.retain()
	t.entries[fd] = file
	if fd >= t.next {
		t.next = fd + 1
	}
	t.mu.Unlock()
	if old != nil {
		_ = old.Close()
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
			file.retain()
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
	if err := t.InstallAt(newfd, file, true); err != nil {
		return -1, err
	}
	return newfd, nil
}

// CloseOnExec removes and closes descriptors marked close-on-exec. It returns
// the guest descriptor numbers removed from the table.
func (t *Table) CloseOnExec() []int32 {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	removed := make([]int32, 0)
	files := make([]*File, 0)
	for fd, file := range t.entries {
		if file == nil || !file.Cloexec {
			continue
		}
		delete(t.entries, fd)
		removed = append(removed, fd)
		files = append(files, file)
	}
	t.mu.Unlock()
	for _, file := range files {
		_ = file.Close()
	}
	return removed
}

func (t *Table) Count() int {
	if t == nil {
		return 0
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.entries)
}

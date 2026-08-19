package fd

import (
	"bytes"
	"io"
	"testing"
)

func TestTableLifecycleAndDup(t *testing.T) {
	input := bytes.NewBufferString("abc")
	output := new(bytes.Buffer)
	table := New()
	in, err := table.Open(&File{Reader: input})
	if err != nil {
		t.Fatal(err)
	}
	out, err := table.Open(&File{Writer: output})
	if err != nil {
		t.Fatal(err)
	}
	if in != 3 || out != 4 {
		t.Fatalf("allocated descriptors %d, %d; want 3, 4", in, out)
	}
	dup, err := table.Dup(out)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := table.Get(dup); err != nil {
		t.Fatal(err)
	}
	if _, err := table.Get(99); err != ErrBadFD {
		t.Fatalf("Get(99) = %v, want ErrBadFD", err)
	}
	if _, err := table.Get(in); err != nil {
		t.Fatal(err)
	}
	if err := table.Close(in); err != nil {
		t.Fatal(err)
	}
	if _, err := table.Get(in); err != ErrBadFD {
		t.Fatalf("closed fd lookup = %v", err)
	}
}

func TestTableDup2AndSeek(t *testing.T) {
	file := bytes.NewReader([]byte("hello"))
	table := New()
	oldfd, err := table.Open(&File{Reader: file, Seeker: file})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := table.Dup2(oldfd, 10); err != nil || got != 10 {
		t.Fatalf("Dup2 = %d, %v", got, err)
	}
	entry, err := table.Get(10)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Seek(2, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 2)
	if _, err := entry.Read(buf); err != nil {
		t.Fatal(err)
	}
	if string(buf) != "ll" {
		t.Fatalf("read after seek = %q", buf)
	}
}

type countingCloser struct{ calls int }

func (c *countingCloser) Close() error {
	c.calls++
	return nil
}

func TestTableDupKeepsCloserAlive(t *testing.T) {
	closer := &countingCloser{}
	table := New()
	fd, err := table.Open(&File{Closer: closer})
	if err != nil {
		t.Fatal(err)
	}
	dup, err := table.Dup(fd)
	if err != nil {
		t.Fatal(err)
	}
	if err := table.Close(fd); err != nil {
		t.Fatal(err)
	}
	if closer.calls != 0 {
		t.Fatalf("closer calls after first close = %d", closer.calls)
	}
	if err := table.Close(dup); err != nil {
		t.Fatal(err)
	}
	if closer.calls != 1 {
		t.Fatalf("closer calls after final close = %d", closer.calls)
	}
}

func TestTableCloseOnExec(t *testing.T) {
	cloexecCloser := &countingCloser{}
	regularCloser := &countingCloser{}
	table := New()
	cloexecFD, err := table.Open(&File{Closer: cloexecCloser, Cloexec: true})
	if err != nil {
		t.Fatal(err)
	}
	regularFD, err := table.Open(&File{Closer: regularCloser})
	if err != nil {
		t.Fatal(err)
	}
	removed := table.CloseOnExec()
	if len(removed) != 1 || removed[0] != cloexecFD {
		t.Fatalf("removed = %#v, want [%d]", removed, cloexecFD)
	}
	if _, err := table.Get(cloexecFD); err != ErrBadFD {
		t.Fatalf("CLOEXEC fd lookup = %v", err)
	}
	if _, err := table.Get(regularFD); err != nil {
		t.Fatalf("regular fd lookup = %v", err)
	}
	if cloexecCloser.calls != 1 || regularCloser.calls != 0 {
		t.Fatalf("closer calls = %d/%d", cloexecCloser.calls, regularCloser.calls)
	}
}

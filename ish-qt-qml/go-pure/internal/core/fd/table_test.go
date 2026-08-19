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

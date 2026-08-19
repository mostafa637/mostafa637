package syscall

import (
	"bytes"
	"encoding/binary"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
)

func TestDispatcher64ReadvWritev(t *testing.T) {
	memory := corecpu.NewMemory64()
	const base corecpu.Address64 = 0x1000
	if err := memory.Map(base, 5*corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	ctx := NewContext64(memory)
	input := bytes.NewBufferString("hello world")
	if err := ctx.InstallFile(9, &corefd.File{Reader: input}); err != nil {
		t.Fatal(err)
	}
	output := new(bytes.Buffer)
	if err := ctx.InstallFile(10, &corefd.File{Writer: output}); err != nil {
		t.Fatal(err)
	}
	dispatcher := NewDispatcher64(ctx)
	state := corecpu.NewMachineState64(memory)
	const iovAddress corecpu.Address64 = 0x1800
	const firstBuffer corecpu.Address64 = 0x2000
	const secondBuffer corecpu.Address64 = 0x3000
	writeIOVec := func(index int, buffer corecpu.Address64, length uint64) {
		t.Helper()
		raw := make([]byte, iovec64Size)
		binary.LittleEndian.PutUint64(raw[0:8], uint64(buffer))
		binary.LittleEndian.PutUint64(raw[8:16], length)
		if err := memory.Write(iovAddress+corecpu.Address64(index*iovec64Size), raw); err != nil {
			t.Fatal(err)
		}
	}
	writeIOVec(0, firstBuffer, 5)
	writeIOVec(1, secondBuffer, 6)

	dispatch := func(number Number64, fd uint64, iov uint64, count uint64) int64 {
		t.Helper()
		state.Set(corecpu.RAX, uint64(number))
		state.Set(corecpu.RDI, fd)
		state.Set(corecpu.RSI, iov)
		state.Set(corecpu.RDX, count)
		if resume, err := dispatcher.Dispatch(state); err != nil || !resume {
			t.Fatalf("dispatch %d: resume=%v err=%v", number, resume, err)
		}
		return int64(state.Get(corecpu.RAX))
	}

	if got := dispatch(Sys64Readv, 9, uint64(iovAddress), 2); got != 11 {
		t.Fatalf("readv64 = %d, want 11", got)
	}
	first := make([]byte, 5)
	second := make([]byte, 6)
	if err := memory.Read(firstBuffer, first); err != nil {
		t.Fatal(err)
	}
	if err := memory.Read(secondBuffer, second); err != nil {
		t.Fatal(err)
	}
	if string(first) != "hello" || string(second) != " world" {
		t.Fatalf("readv64 buffers = %q %q", first, second)
	}

	if err := memory.Write(firstBuffer, []byte("pure!")); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(secondBuffer, []byte(" go!!!")); err != nil {
		t.Fatal(err)
	}
	if got := dispatch(Sys64Writev, 10, uint64(iovAddress), 2); got != 11 {
		t.Fatalf("writev64 = %d, want 11", got)
	}
	if output.String() != "pure! go!!!" {
		t.Fatalf("writev64 output = %q", output.String())
	}

	if got := dispatch(Sys64Readv, 9, uint64(base+5*corecpu.Address64(corecpu.Page64Size)), 2); got != int64(EFAULT) {
		t.Fatalf("bad iovec address = %d, want %d", got, EFAULT)
	}
	writeIOVec(0, firstBuffer, maxIOBytes64)
	writeIOVec(1, secondBuffer, 1)
	if got := dispatch(Sys64Writev, 10, uint64(iovAddress), 2); got != int64(EINVAL) {
		t.Fatalf("iovec total overflow = %d, want %d", got, EINVAL)
	}
}

package syscall

import (
	"bytes"
	"encoding/binary"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
)

func TestDispatcher64Pipe2NonblockAndFlags(t *testing.T) {
	memory := corecpu.NewMemory64()
	const area corecpu.Address64 = 0xb000
	if err := memory.Map(area, 2*corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	ctx := NewContext64(memory)
	dispatcher := NewDispatcher64(ctx)
	state := corecpu.NewMachineState64(memory)

	state.Set(corecpu.RAX, uint64(Sys64Pipe2))
	state.Set(corecpu.RDI, uint64(area))
	state.Set(corecpu.RSI, uint64(pipe2Nonblock|pipe2Cloexec))
	resume, err := dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("pipe2: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	var pair [8]byte
	if err := memory.Read(area, pair[:]); err != nil {
		t.Fatal(err)
	}
	readFD := binary.LittleEndian.Uint32(pair[:4])
	writeFD := binary.LittleEndian.Uint32(pair[4:])
	readFile, err := ctx.GetFile(uint64(readFD))
	if err != nil || !readFile.Cloexec || readFile.StatusFlags&uint64(guestOpenNonblock) == 0 {
		t.Fatalf("read pipe flags: err=%v file=%#v", err, readFile)
	}
	writeFile, err := ctx.GetFile(uint64(writeFD))
	if err != nil || !writeFile.Cloexec || writeFile.StatusFlags&uint64(guestOpenNonblock) == 0 {
		t.Fatalf("write pipe flags: err=%v file=%#v", err, writeFile)
	}

	state.Set(corecpu.RAX, uint64(Sys64Fcntl))
	state.Set(corecpu.RDI, uint64(readFD))
	state.Set(corecpu.RSI, fcntlGetFL)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || state.Get(corecpu.RAX)&uint64(guestOpenNonblock) == 0 {
		t.Fatalf("pipe fcntl getfl: resume=%v err=%v rax=%d", resume, err, state.Get(corecpu.RAX))
	}

	state.Set(corecpu.RAX, uint64(Sys64Read))
	state.Set(corecpu.RDI, uint64(readFD))
	state.Set(corecpu.RSI, uint64(area+0x100))
	state.Set(corecpu.RDX, 4)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != int64(EAGAIN) {
		t.Fatalf("empty nonblock pipe read: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}

	if err := memory.Write(area+0x200, []byte("pipe")); err != nil {
		t.Fatal(err)
	}
	state.Set(corecpu.RAX, uint64(Sys64Write))
	state.Set(corecpu.RDI, uint64(writeFD))
	state.Set(corecpu.RSI, uint64(area+0x200))
	state.Set(corecpu.RDX, 4)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || state.Get(corecpu.RAX) != 4 {
		t.Fatalf("pipe write: resume=%v err=%v rax=%d", resume, err, state.Get(corecpu.RAX))
	}
	state.Set(corecpu.RAX, uint64(Sys64Read))
	state.Set(corecpu.RDI, uint64(readFD))
	state.Set(corecpu.RSI, uint64(area+0x100))
	state.Set(corecpu.RDX, 4)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || state.Get(corecpu.RAX) != 4 {
		t.Fatalf("pipe read: resume=%v err=%v rax=%d", resume, err, state.Get(corecpu.RAX))
	}
	var data [4]byte
	if err := memory.Read(area+0x100, data[:]); err != nil {
		t.Fatal(err)
	}
	if string(data[:]) != "pipe" {
		t.Fatalf("pipe data = %q", data[:])
	}

	state.Set(corecpu.RAX, uint64(Sys64Fcntl))
	state.Set(corecpu.RDI, uint64(readFD))
	state.Set(corecpu.RSI, fcntlSetFL)
	state.Set(corecpu.RDX, 0)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("pipe fcntl setfl: resume=%v err=%v rax=%d", resume, err, state.Get(corecpu.RAX))
	}
	readFile, err = ctx.GetFile(uint64(readFD))
	if err != nil || readFile.StatusFlags&uint64(guestOpenNonblock) != 0 {
		t.Fatalf("pipe setfl did not clear nonblock: err=%v file=%#v", err, readFile)
	}
}

func TestDispatcher64SendfileAndSplice(t *testing.T) {
	memory := corecpu.NewMemory64()
	const area corecpu.Address64 = 0xc000
	if err := memory.Map(area, corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	ctx := NewContext64(memory)
	dispatcher := NewDispatcher64(ctx)
	state := corecpu.NewMachineState64(memory)

	source := bytes.NewReader([]byte("sendfile-data"))
	var destination bytes.Buffer
	if err := ctx.InstallFile(10, &corefd.File{Reader: source, Seeker: source}); err != nil {
		t.Fatal(err)
	}
	if err := ctx.InstallFile(11, &corefd.File{Writer: &destination}); err != nil {
		t.Fatal(err)
	}
	var offset [8]byte
	binary.LittleEndian.PutUint64(offset[:], 0)
	if err := memory.Write(area, offset[:]); err != nil {
		t.Fatal(err)
	}
	state.Set(corecpu.RAX, uint64(Sys64Sendfile))
	state.Set(corecpu.RDI, 11)
	state.Set(corecpu.RSI, 10)
	state.Set(corecpu.RDX, uint64(area))
	state.Set(corecpu.R10, uint64(len("sendfile-data")))
	resume, err := dispatcher.Dispatch(state)
	if err != nil || !resume || state.Get(corecpu.RAX) != uint64(len("sendfile-data")) {
		t.Fatalf("sendfile: resume=%v err=%v rax=%d", resume, err, state.Get(corecpu.RAX))
	}
	if destination.String() != "sendfile-data" {
		t.Fatalf("sendfile destination = %q", destination.String())
	}
	if err := memory.Read(area, offset[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint64(offset[:]); got != uint64(len("sendfile-data")) {
		t.Fatalf("sendfile offset = %d", got)
	}

	state.Set(corecpu.RAX, uint64(Sys64Pipe2))
	state.Set(corecpu.RDI, uint64(area+0x100))
	state.Set(corecpu.RSI, 0)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("splice pipe2: resume=%v err=%v rax=%d", resume, err, state.Get(corecpu.RAX))
	}
	var pair [8]byte
	if err := memory.Read(area+0x100, pair[:]); err != nil {
		t.Fatal(err)
	}
	readFD := binary.LittleEndian.Uint32(pair[:4])
	writeFD := binary.LittleEndian.Uint32(pair[4:])
	if err := memory.Write(area+0x200, []byte("splice-data")); err != nil {
		t.Fatal(err)
	}
	state.Set(corecpu.RAX, uint64(Sys64Write))
	state.Set(corecpu.RDI, uint64(writeFD))
	state.Set(corecpu.RSI, uint64(area+0x200))
	state.Set(corecpu.RDX, uint64(len("splice-data")))
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || state.Get(corecpu.RAX) != uint64(len("splice-data")) {
		t.Fatalf("splice source write: resume=%v err=%v rax=%d", resume, err, state.Get(corecpu.RAX))
	}
	var spliceDestination bytes.Buffer
	const spliceDestinationFD uint64 = 20
	if err := ctx.InstallFile(spliceDestinationFD, &corefd.File{Writer: &spliceDestination}); err != nil {
		t.Fatal(err)
	}
	state.Set(corecpu.RAX, uint64(Sys64Splice))
	state.Set(corecpu.RDI, uint64(readFD))
	state.Set(corecpu.RSI, 0)
	state.Set(corecpu.RDX, spliceDestinationFD)
	state.Set(corecpu.R10, 0)
	state.Set(corecpu.R8, uint64(len("splice-data")))
	state.Set(corecpu.R9, 0)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || state.Get(corecpu.RAX) != uint64(len("splice-data")) {
		t.Fatalf("splice: resume=%v err=%v rax=%d", resume, err, state.Get(corecpu.RAX))
	}
	if spliceDestination.String() != "splice-data" {
		t.Fatalf("splice destination = %q", spliceDestination.String())
	}
}

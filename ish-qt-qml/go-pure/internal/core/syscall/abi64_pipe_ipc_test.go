package syscall

import (
	"bytes"
	"encoding/binary"
	"io"
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

func TestDispatcher64SendfileExplicitOffsetPreservesSourcePosition(t *testing.T) {
	memory := corecpu.NewMemory64()
	const area corecpu.Address64 = 0xd000
	if err := memory.Map(area, corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	ctx := NewContext64(memory)
	dispatcher := NewDispatcher64(ctx)
	state := corecpu.NewMachineState64(memory)

	source := bytes.NewReader([]byte("0123456789"))
	if _, err := source.Seek(7, 0); err != nil {
		t.Fatal(err)
	}
	var destination bytes.Buffer
	const sourceFD uint64 = 30
	const destinationFD uint64 = 31
	if err := ctx.InstallFile(sourceFD, &corefd.File{Reader: source, Seeker: source}); err != nil {
		t.Fatal(err)
	}
	if err := ctx.InstallFile(destinationFD, &corefd.File{Writer: &destination}); err != nil {
		t.Fatal(err)
	}
	var raw [8]byte
	binary.LittleEndian.PutUint64(raw[:], 2)
	if err := memory.Write(area, raw[:]); err != nil {
		t.Fatal(err)
	}

	state.Set(corecpu.RAX, uint64(Sys64Sendfile))
	state.Set(corecpu.RDI, destinationFD)
	state.Set(corecpu.RSI, sourceFD)
	state.Set(corecpu.RDX, uint64(area))
	state.Set(corecpu.R10, 3)
	resume, err := dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 3 {
		t.Fatalf("sendfile explicit offset: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	if destination.String() != "234" {
		t.Fatalf("sendfile explicit destination = %q, want %q", destination.String(), "234")
	}
	if got, err := source.Seek(0, 1); err != nil || got != 7 {
		t.Fatalf("sendfile changed source position: got=%d err=%v, want 7", got, err)
	}
	if err := memory.Read(area, raw[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint64(raw[:]); got != 5 {
		t.Fatalf("sendfile explicit offset = %d, want 5", got)
	}
}

func TestDispatcher64SpliceOffsetsAndValidation(t *testing.T) {
	memory := corecpu.NewMemory64()
	const area corecpu.Address64 = 0xe000
	if err := memory.Map(area, corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	ctx := NewContext64(memory)
	dispatcher := NewDispatcher64(ctx)
	state := corecpu.NewMachineState64(memory)

	source := bytes.NewReader([]byte("abcdef"))
	if _, err := source.Seek(5, 0); err != nil {
		t.Fatal(err)
	}
	destination := &seekBuffer{}
	const sourceFD uint64 = 32
	const destinationFD uint64 = 33
	if err := ctx.InstallFile(sourceFD, &corefd.File{Reader: source, Seeker: source}); err != nil {
		t.Fatal(err)
	}
	if err := ctx.InstallFile(destinationFD, &corefd.File{Writer: destination, Seeker: destination}); err != nil {
		t.Fatal(err)
	}
	var inputOffset [8]byte
	var outputOffset [8]byte
	binary.LittleEndian.PutUint64(inputOffset[:], 1)
	binary.LittleEndian.PutUint64(outputOffset[:], 2)
	if err := memory.Write(area, inputOffset[:]); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(area+8, outputOffset[:]); err != nil {
		t.Fatal(err)
	}

	state.Set(corecpu.RAX, uint64(Sys64Splice))
	state.Set(corecpu.RDI, sourceFD)
	state.Set(corecpu.RSI, uint64(area))
	state.Set(corecpu.RDX, destinationFD)
	state.Set(corecpu.R10, uint64(area+8))
	state.Set(corecpu.R8, 2)
	state.Set(corecpu.R9, 0)
	resume, err := dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 2 {
		t.Fatalf("splice explicit offsets: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	if got := string(destination.data); got != "\x00\x00bc" {
		t.Fatalf("splice destination = %q, want leading offset and %q", got, "bc")
	}
	if got, err := source.Seek(0, 1); err != nil || got != 5 {
		t.Fatalf("splice changed source position: got=%d err=%v, want 5", got, err)
	}
	if got := destination.pos; got != 0 {
		t.Fatalf("splice changed destination position: got=%d, want origin 0", got)
	}
	if err := memory.Read(area, inputOffset[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint64(inputOffset[:]); got != 3 {
		t.Fatalf("splice input offset = %d, want 3", got)
	}
	if err := memory.Read(area+8, outputOffset[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint64(outputOffset[:]); got != 4 {
		t.Fatalf("splice output offset = %d, want 4", got)
	}

	binary.LittleEndian.PutUint64(inputOffset[:], ^uint64(0))
	if err := memory.Write(area, inputOffset[:]); err != nil {
		t.Fatal(err)
	}
	state.Set(corecpu.RAX, uint64(Sys64Splice))
	state.Set(corecpu.RDI, sourceFD)
	state.Set(corecpu.RSI, uint64(area))
	state.Set(corecpu.RDX, destinationFD)
	state.Set(corecpu.R10, 0)
	state.Set(corecpu.R8, 1)
	state.Set(corecpu.R9, 0)
	_, err = dispatcher.Dispatch(state)
	if err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("negative splice offset: err=%v rax=%d, want EINVAL", err, int64(state.Get(corecpu.RAX)))
	}

	state.Set(corecpu.RAX, uint64(Sys64Sendfile))
	state.Set(corecpu.RDI, sourceFD)
	state.Set(corecpu.RSI, sourceFD)
	state.Set(corecpu.RDX, 0)
	state.Set(corecpu.R10, 0)
	_, err = dispatcher.Dispatch(state)
	if err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("same-fd sendfile: err=%v rax=%d, want EINVAL", err, int64(state.Get(corecpu.RAX)))
	}

	state.Set(corecpu.RAX, uint64(Sys64Splice))
	state.Set(corecpu.RDI, sourceFD)
	state.Set(corecpu.RSI, 0)
	state.Set(corecpu.RDX, sourceFD)
	state.Set(corecpu.R10, 0)
	state.Set(corecpu.R8, 1)
	state.Set(corecpu.R9, 0)
	_, err = dispatcher.Dispatch(state)
	if err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("same-fd splice: err=%v rax=%d, want EINVAL", err, int64(state.Get(corecpu.RAX)))
	}
}

func TestDispatcher64SplicePipeToPipeAndZeroCount(t *testing.T) {
	memory := corecpu.NewMemory64()
	const area corecpu.Address64 = 0xf000
	if err := memory.Map(area, 2*corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	ctx := NewContext64(memory)
	dispatcher := NewDispatcher64(ctx)
	state := corecpu.NewMachineState64(memory)

	makePipe := func(address corecpu.Address64) (uint64, uint64) {
		state.Set(corecpu.RAX, uint64(Sys64Pipe2))
		state.Set(corecpu.RDI, uint64(address))
		state.Set(corecpu.RSI, 0)
		resume, err := dispatcher.Dispatch(state)
		if err != nil || !resume || state.Get(corecpu.RAX) != 0 {
			t.Fatalf("pipe2: resume=%v err=%v rax=%d", resume, err, state.Get(corecpu.RAX))
		}
		var pair [8]byte
		if err := memory.Read(address, pair[:]); err != nil {
			t.Fatal(err)
		}
		return uint64(binary.LittleEndian.Uint32(pair[:4])), uint64(binary.LittleEndian.Uint32(pair[4:]))
	}
	readIn, writeIn := makePipe(area)
	readOut, writeOut := makePipe(area + 0x100)
	if err := memory.Write(area+0x200, []byte("tee-like")); err != nil {
		t.Fatal(err)
	}
	state.Set(corecpu.RAX, uint64(Sys64Write))
	state.Set(corecpu.RDI, writeIn)
	state.Set(corecpu.RSI, uint64(area+0x200))
	state.Set(corecpu.RDX, 8)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 8 {
		t.Fatalf("pipe source write: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}

	state.Set(corecpu.RAX, uint64(Sys64Splice))
	state.Set(corecpu.RDI, readIn)
	state.Set(corecpu.RSI, 0)
	state.Set(corecpu.RDX, writeOut)
	state.Set(corecpu.R10, 0)
	state.Set(corecpu.R8, 0)
	state.Set(corecpu.R9, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("zero-count splice: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}
	state.Set(corecpu.RAX, uint64(Sys64Splice))
	state.Set(corecpu.R8, 8)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 8 {
		t.Fatalf("pipe-to-pipe splice: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}

	state.Set(corecpu.RAX, uint64(Sys64Read))
	state.Set(corecpu.RDI, readOut)
	state.Set(corecpu.RSI, uint64(area+0x300))
	state.Set(corecpu.RDX, 8)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 8 {
		t.Fatalf("pipe destination read: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}
	var got [8]byte
	if err := memory.Read(area+0x300, got[:]); err != nil {
		t.Fatal(err)
	}
	if string(got[:]) != "tee-like" {
		t.Fatalf("pipe-to-pipe data = %q, want %q", got[:], "tee-like")
	}
}

type seekBuffer struct {
	data []byte
	pos  int64
}

func (b *seekBuffer) Write(p []byte) (int, error) {
	end := b.pos + int64(len(p))
	if end > int64(len(b.data)) {
		b.data = append(b.data, make([]byte, end-int64(len(b.data)))...)
	}
	copy(b.data[b.pos:end], p)
	b.pos = end
	return len(p), nil
}

func (b *seekBuffer) Seek(offset int64, whence int) (int64, error) {
	var base int64
	switch whence {
	case 0:
		base = 0
	case 1:
		base = b.pos
	case 2:
		base = int64(len(b.data))
	default:
		return 0, io.ErrUnexpectedEOF
	}
	position := base + offset
	if position < 0 {
		return 0, io.ErrUnexpectedEOF
	}
	b.pos = position
	return position, nil
}

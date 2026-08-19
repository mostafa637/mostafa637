package syscall

import (
	"encoding/binary"
	"fmt"
	"net"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func TestDispatcher64Socketpair(t *testing.T) {
	memory := corecpu.NewMemory64()
	const area corecpu.Address64 = 0x9000
	if err := memory.Map(area, corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	context64 := NewContext64(memory)
	dispatcher := NewDispatcher64(context64)
	state := corecpu.NewMachineState64(memory)

	state.Set(corecpu.RAX, uint64(Sys64Socketpair))
	state.Set(corecpu.RDI, socketAFUnix)
	state.Set(corecpu.RSI, socketStream)
	state.Set(corecpu.RDX, 0)
	state.Set(corecpu.R10, uint64(area))
	resume, err := dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("socketpair: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	var pair [8]byte
	if err := memory.Read(area, pair[:]); err != nil {
		t.Fatal(err)
	}
	leftFD := binary.LittleEndian.Uint32(pair[:4])
	rightFD := binary.LittleEndian.Uint32(pair[4:])
	if leftFD < 3 || rightFD < 3 || leftFD == rightFD {
		t.Fatalf("socketpair fds = %d,%d", leftFD, rightFD)
	}

	if err := memory.Write(area+0x100, []byte("ping")); err != nil {
		t.Fatal(err)
	}
	writeState := corecpu.NewMachineState64(memory)
	writeState.Set(corecpu.RAX, uint64(Sys64Write))
	writeState.Set(corecpu.RDI, uint64(leftFD))
	writeState.Set(corecpu.RSI, uint64(area+0x100))
	writeState.Set(corecpu.RDX, 4)
	writeDone := make(chan struct{})
	var writeResume bool
	var writeErr error
	go func() {
		writeResume, writeErr = dispatcher.Dispatch(writeState)
		close(writeDone)
	}()

	state.Set(corecpu.RAX, uint64(Sys64Read))
	state.Set(corecpu.RDI, uint64(rightFD))
	state.Set(corecpu.RSI, uint64(area+0x200))
	state.Set(corecpu.RDX, 4)
	resume, err = dispatcher.Dispatch(state)
	<-writeDone
	if writeErr != nil || !writeResume || writeState.Get(corecpu.RAX) != 4 {
		t.Fatalf("socketpair write: resume=%v err=%v rax=%d", writeResume, writeErr, writeState.Get(corecpu.RAX))
	}
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 4 {
		t.Fatalf("socketpair read: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	var data [4]byte
	if err := memory.Read(area+0x200, data[:]); err != nil {
		t.Fatal(err)
	}
	if string(data[:]) != "ping" {
		t.Fatalf("socketpair data = %q", data[:])
	}

	for _, fd := range []uint32{leftFD, rightFD} {
		state.Set(corecpu.RAX, uint64(Sys64Close))
		state.Set(corecpu.RDI, uint64(fd))
		resume, err = dispatcher.Dispatch(state)
		if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 0 {
			t.Fatalf("socketpair close fd=%d: resume=%v err=%v rax=%d", fd, resume, err, int64(state.Get(corecpu.RAX)))
		}
	}
}

func TestDispatcher64TCPAccept4(t *testing.T) {
	memory := corecpu.NewMemory64()
	const area corecpu.Address64 = 0xa000
	if err := memory.Map(area, corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	context64 := NewContext64(memory)
	dispatcher := NewDispatcher64(context64)
	state := corecpu.NewMachineState64(memory)

	state.Set(corecpu.RAX, uint64(Sys64Socket))
	state.Set(corecpu.RDI, socketAFInet)
	state.Set(corecpu.RSI, socketStream)
	state.Set(corecpu.RDX, 0)
	resume, err := dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) < 3 {
		t.Fatalf("socket: resume=%v err=%v fd=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	serverFD := uint64(state.Get(corecpu.RAX))

	var sockaddr [16]byte
	binary.LittleEndian.PutUint16(sockaddr[0:2], socketAFInet)
	binary.BigEndian.PutUint16(sockaddr[2:4], 0)
	sockaddr[4] = 127
	sockaddr[5] = 0
	sockaddr[6] = 0
	sockaddr[7] = 1
	if err := memory.Write(area+0x100, sockaddr[:]); err != nil {
		t.Fatal(err)
	}
	state.Set(corecpu.RAX, uint64(Sys64Bind))
	state.Set(corecpu.RDI, serverFD)
	state.Set(corecpu.RSI, uint64(area+0x100))
	state.Set(corecpu.RDX, uint64(len(sockaddr)))
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("bind: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}

	state.Set(corecpu.RAX, uint64(Sys64Listen))
	state.Set(corecpu.RDI, serverFD)
	state.Set(corecpu.RSI, 8)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("listen: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}

	var addrLen [4]byte
	binary.LittleEndian.PutUint32(addrLen[:], uint32(len(sockaddr)))
	if err := memory.Write(area+0x200, addrLen[:]); err != nil {
		t.Fatal(err)
	}
	state.Set(corecpu.RAX, uint64(Sys64Getsockname))
	state.Set(corecpu.RDI, serverFD)
	state.Set(corecpu.RSI, uint64(area+0x100))
	state.Set(corecpu.RDX, uint64(area+0x200))
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("getsockname: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	if err := memory.Read(area+0x100, sockaddr[:]); err != nil {
		t.Fatal(err)
	}
	port := binary.BigEndian.Uint16(sockaddr[2:4])
	if port == 0 {
		t.Fatal("getsockname returned port zero")
	}

	client, err := net.Dial("tcp4", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := client.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}

	binary.LittleEndian.PutUint32(addrLen[:], uint32(len(sockaddr)))
	if err := memory.Write(area+0x200, addrLen[:]); err != nil {
		t.Fatal(err)
	}
	state.Set(corecpu.RAX, uint64(Sys64Accept4))
	state.Set(corecpu.RDI, serverFD)
	state.Set(corecpu.RSI, uint64(area+0x100))
	state.Set(corecpu.RDX, uint64(area+0x200))
	state.Set(corecpu.R10, socketCloexec)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) < 3 {
		t.Fatalf("accept4: resume=%v err=%v fd=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	acceptedFD := uint64(state.Get(corecpu.RAX))
	file, err := context64.GetFile(acceptedFD)
	if err != nil || !file.Cloexec {
		t.Fatalf("accept4 cloexec: err=%v file=%#v", err, file)
	}

	state.Set(corecpu.RAX, uint64(Sys64Read))
	state.Set(corecpu.RDI, acceptedFD)
	state.Set(corecpu.RSI, uint64(area+0x300))
	state.Set(corecpu.RDX, 5)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 5 {
		t.Fatalf("accepted read: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	var data [5]byte
	if err := memory.Read(area+0x300, data[:]); err != nil {
		t.Fatal(err)
	}
	if string(data[:]) != "hello" {
		t.Fatalf("accepted data = %q", data[:])
	}

	for _, fd := range []uint64{acceptedFD, serverFD} {
		state.Set(corecpu.RAX, uint64(Sys64Close))
		state.Set(corecpu.RDI, fd)
		resume, err = dispatcher.Dispatch(state)
		if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 0 {
			t.Fatalf("tcp close fd=%d: resume=%v err=%v rax=%d", fd, resume, err, int64(state.Get(corecpu.RAX)))
		}
	}
}

package syscall

import (
	"encoding/binary"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func TestDispatcher64SocketOptionsAndMessages(t *testing.T) {
	memory := corecpu.NewMemory64()
	const area corecpu.Address64 = 0x11000
	if err := memory.Map(area, 2*corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	context64 := NewContext64(memory)
	dispatcher := NewDispatcher64(context64)
	state := corecpu.NewMachineState64(memory)

	pairAddress := area + 0x100
	set64Syscall(state, Sys64Socketpair, socketAFUnix, socketStream, 0, uint64(pairAddress))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("socketpair: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	var pair [8]byte
	if err := memory.Read(pairAddress, pair[:]); err != nil {
		t.Fatal(err)
	}
	leftFD := uint64(binary.LittleEndian.Uint32(pair[0:4]))
	rightFD := uint64(binary.LittleEndian.Uint32(pair[4:8]))
	if leftFD < 3 || rightFD < 3 || leftFD == rightFD {
		t.Fatalf("socketpair fds = %d,%d", leftFD, rightFD)
	}

	optValue := area + 0x200
	optLength := area + 0x210
	var four [4]byte
	binary.LittleEndian.PutUint32(four[:], 1)
	if err := memory.Write(optValue, four[:]); err != nil {
		t.Fatal(err)
	}
	binary.LittleEndian.PutUint32(four[:], 4)
	if err := memory.Write(optLength, four[:]); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64Setsockopt, leftFD, socketSOLSocket64, socketSOReuseAddr64, uint64(optValue), 4)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("setsockopt: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	binary.LittleEndian.PutUint32(four[:], 0)
	if err := memory.Write(optValue, four[:]); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64Getsockopt, leftFD, socketSOLSocket64, socketSOReuseAddr64, uint64(optValue), uint64(optLength))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("getsockopt: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if err := memory.Read(optValue, four[:]); err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint32(four[:]) != 1 {
		t.Fatalf("getsockopt value = %d, want 1", binary.LittleEndian.Uint32(four[:]))
	}
	if err := memory.Read(optLength, four[:]); err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint32(four[:]) != 4 {
		t.Fatalf("getsockopt length = %d, want 4", binary.LittleEndian.Uint32(four[:]))
	}

	const (
		messageAddress corecpu.Address64 = area + 0x300
		iovecAddress   corecpu.Address64 = area + 0x400
		inputAddress   corecpu.Address64 = area + 0x500
		outputAddress  corecpu.Address64 = area + 0x600
	)
	if err := memory.Write(inputAddress, []byte("hello")); err != nil {
		t.Fatal(err)
	}
	var iovec [16]byte
	binary.LittleEndian.PutUint64(iovec[0:8], uint64(inputAddress))
	binary.LittleEndian.PutUint64(iovec[8:16], 5)
	if err := memory.Write(iovecAddress, iovec[:]); err != nil {
		t.Fatal(err)
	}
	var header [msghdr64Size]byte
	binary.LittleEndian.PutUint64(header[msghdr64IOVOffset:msghdr64IOVOffset+8], uint64(iovecAddress))
	binary.LittleEndian.PutUint64(header[msghdr64IOVLenOff:msghdr64IOVLenOff+8], 1)
	if err := memory.Write(messageAddress, header[:]); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64Sendmsg, leftFD, uint64(messageAddress), 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 5 {
		t.Fatalf("sendmsg: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	binary.LittleEndian.PutUint64(iovec[0:8], uint64(outputAddress))
	if err := memory.Write(iovecAddress, iovec[:]); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(messageAddress, header[:]); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64Recvmsg, rightFD, uint64(messageAddress), 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 5 {
		t.Fatalf("recvmsg: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	var output [5]byte
	if err := memory.Read(outputAddress, output[:]); err != nil {
		t.Fatal(err)
	}
	if string(output[:]) != "hello" {
		t.Fatalf("recvmsg output = %q", output[:])
	}

	for _, fd := range []uint64{leftFD, rightFD} {
		set64Syscall(state, Sys64Close, fd)
		if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
			t.Fatalf("close fd=%d: err=%v rax=%d", fd, err, int64(state.Get(corecpu.RAX)))
		}
	}
}

func TestDispatcher64SocketOptionValidation(t *testing.T) {
	memory := corecpu.NewMemory64()
	const area corecpu.Address64 = 0x13000
	if err := memory.Map(area, corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	context64 := NewContext64(memory)
	dispatcher := NewDispatcher64(context64)
	state := corecpu.NewMachineState64(memory)
	set64Syscall(state, Sys64Socket, socketAFInet, socketStream, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) < 3 {
		t.Fatalf("socket: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	fd := state.Get(corecpu.RAX)
	set64Syscall(state, Sys64Setsockopt, fd, socketSOLSocket64, socketSOReuseAddr64, uint64(area+corecpu.Address64(corecpu.Page64Size)), 4)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EFAULT) {
		t.Fatalf("bad optval address: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Close, fd)
	if _, err := dispatcher.Dispatch(state); err != nil {
		t.Fatal(err)
	}
}

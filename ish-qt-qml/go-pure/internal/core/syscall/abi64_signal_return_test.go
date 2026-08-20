package syscall

import (
	"encoding/binary"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func TestABI64RtSigreturnRestoresMachineState(t *testing.T) {
	_, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	const frameAddress corecpu.Address64 = 0x10800
	state.Set(corecpu.RSP, uint64(frameAddress))
	state.RIP = 0x12345678
	state.RFLAGS = corecpu.Flag64IF | corecpu.Flag64CF | corecpu.Flag64ZF
	state.FSBase = 0x11000
	state.GSBase = 0x12000
	for reg := corecpu.Reg64(0); reg < corecpu.Reg64Count; reg++ {
		state.Set(reg, 0x1000+uint64(reg)*0x111)
	}
	wantMask := uint64(0xfeed) | signalFrame64ReservedMask
	if !writeSignalFrame64(memory, frameAddress, state, wantMask) {
		t.Fatal("could not write signal frame")
	}
	wantRIP := state.RIP
	wantRSP := state.Get(corecpu.RSP)
	wantRFLAGS := state.RFLAGS | signalFrame64RequiredFlags
	wantRegs := state.Regs
	wantFS, wantGS := state.FSBase, state.GSBase
	state.RIP = 0
	state.Set(corecpu.RSP, uint64(frameAddress))
	state.Set(corecpu.RAX, 0)
	state.FSBase, state.GSBase = 0, 0
	ctx.SignalMask = 0
	set64Syscall(state, Sys64RtSigreturn)
	if resume, err := dispatcher.Dispatch(state); err != nil || !resume {
		t.Fatalf("sigreturn: resume=%v err=%v", resume, err)
	}
	if got := state.RIP; got != wantRIP {
		t.Fatalf("RIP=%#x want %#x", got, wantRIP)
	}
	if got := state.Get(corecpu.RSP); got != wantRSP {
		t.Fatalf("RSP=%#x want %#x", got, wantRSP)
	}
	if got := state.RFLAGS; got != wantRFLAGS|corecpu.Flag64IF {
		t.Fatalf("RFLAGS=%#x want %#x", got, wantRFLAGS|corecpu.Flag64IF)
	}
	if state.FSBase != wantFS || state.GSBase != wantGS || ctx.FSBase != wantFS || ctx.GSBase != wantGS {
		t.Fatalf("FS/GS state=%#x/%#x context=%#x/%#x want %#x/%#x", state.FSBase, state.GSBase, ctx.FSBase, ctx.GSBase, wantFS, wantGS)
	}
	for reg := corecpu.Reg64(0); reg < corecpu.Reg64Count; reg++ {
		if got := state.Get(reg); got != wantRegs[reg] {
			t.Fatalf("reg %s=%#x want %#x", reg, got, wantRegs[reg])
		}
	}
	if ctx.SignalMask != wantMask&^signalFrame64ReservedMask {
		t.Fatalf("signal mask=%#x", ctx.SignalMask)
	}
	if ctx.SignalRestored {
		t.Fatal("dispatcher did not clear transient SignalRestored marker")
	}
}

func TestABI64RtSigreturnValidation(t *testing.T) {
	_, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	const frameAddress corecpu.Address64 = 0x10800
	state.Set(corecpu.RSP, uint64(frameAddress))
	state.RIP = 0x400000
	state.Set(corecpu.RAX, 0x600d)
	state.RFLAGS = corecpu.Flag64IF
	if !writeSignalFrame64(memory, frameAddress, state, 0) {
		t.Fatal("could not write signal frame")
	}
	set64Syscall(state, Sys64RtSigreturn)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 0x600d {
		t.Fatalf("valid frame: err=%v rax=%#x", err, state.Get(corecpu.RAX))
	}

	state.Set(corecpu.RSP, 0)
	set64Syscall(state, Sys64RtSigreturn)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EFAULT) {
		t.Fatalf("null frame: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	state.Set(corecpu.RSP, uint64(frameAddress))
	var raw [signalFrame64Size]byte
	if err := memory.Read(frameAddress, raw[:]); err != nil {
		t.Fatal(err)
	}
	binary.LittleEndian.PutUint64(raw[0:8], 0)
	if err := memory.Write(frameAddress, raw[:]); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64RtSigreturn)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("bad magic: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	_ = ctx
}

func TestABI64RtSigreturnRejectsPrivilegedFlagsAndNonCanonicalRIP(t *testing.T) {
	_, memory, _, dispatcher, state := newABI64FilesystemTest(t)
	const frameAddress corecpu.Address64 = 0x10800
	state.Set(corecpu.RSP, uint64(frameAddress))
	state.RIP = 0x400000
	state.Set(corecpu.RAX, 0x600d)
	state.RFLAGS = corecpu.Flag64IF
	if !writeSignalFrame64(memory, frameAddress, state, 0) {
		t.Fatal("could not write signal frame")
	}
	var raw [signalFrame64Size]byte
	if err := memory.Read(frameAddress, raw[:]); err != nil {
		t.Fatal(err)
	}
	binary.LittleEndian.PutUint64(raw[signalFrame64RFLAGSOffset:signalFrame64RFLAGSOffset+8], signalFrame64RequiredFlags|(3<<12))
	if err := memory.Write(frameAddress, raw[:]); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64RtSigreturn)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("privileged flags: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	if err := memory.Read(frameAddress, raw[:]); err != nil {
		t.Fatal(err)
	}
	binary.LittleEndian.PutUint64(raw[signalFrame64RFLAGSOffset:signalFrame64RFLAGSOffset+8], signalFrame64RequiredFlags)
	binary.LittleEndian.PutUint64(raw[signalFrame64RIPOffset:signalFrame64RIPOffset+8], 0x0001000000000000)
	if err := memory.Write(frameAddress, raw[:]); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64RtSigreturn)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("noncanonical RIP: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
}

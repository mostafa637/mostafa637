package syscall

import (
	"encoding/binary"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

const (
	signalFrame64Magic   uint64 = 0x4953483634534947
	signalFrame64Version uint64 = 1
	signalFrame64Size           = 64 + int(corecpu.Reg64Count)*8

	signalFrame64MaskOffset    = 16
	signalFrame64FSBaseOffset  = 24
	signalFrame64GSBaseOffset  = 32
	signalFrame64RIPOffset     = 40
	signalFrame64RFLAGSOffset  = 48
	signalFrame64RSPOffset     = 56
	signalFrame64RegsOffset    = 64
	signalFrame64ReservedMask  = (uint64(1) << (9 - 1)) | (uint64(1) << (19 - 1))
	signalFrame64RequiredFlags = uint64(1) << 1
)

func signalFrame64Canonical(value uint64) bool {
	upper := value >> 48
	return upper == 0 || upper == 0xffff
}

func readSignalFrame64(memory *corecpu.Memory64, address corecpu.Address64) ([signalFrame64Size]byte, bool) {
	var raw [signalFrame64Size]byte
	if memory == nil || address == 0 {
		return raw, false
	}
	if err := memory.Read(address, raw[:]); err != nil {
		return raw, false
	}
	return raw, true
}

func writeSignalFrame64(memory *corecpu.Memory64, address corecpu.Address64, state *corecpu.MachineState64, mask uint64) bool {
	if memory == nil || state == nil || address == 0 {
		return false
	}
	var raw [signalFrame64Size]byte
	binary.LittleEndian.PutUint64(raw[0:8], signalFrame64Magic)
	binary.LittleEndian.PutUint64(raw[8:16], signalFrame64Version)
	binary.LittleEndian.PutUint64(raw[signalFrame64MaskOffset:signalFrame64MaskOffset+8], mask&^signalFrame64ReservedMask)
	binary.LittleEndian.PutUint64(raw[signalFrame64FSBaseOffset:signalFrame64FSBaseOffset+8], state.FSBase)
	binary.LittleEndian.PutUint64(raw[signalFrame64GSBaseOffset:signalFrame64GSBaseOffset+8], state.GSBase)
	binary.LittleEndian.PutUint64(raw[signalFrame64RIPOffset:signalFrame64RIPOffset+8], state.RIP)
	binary.LittleEndian.PutUint64(raw[signalFrame64RFLAGSOffset:signalFrame64RFLAGSOffset+8], state.RFLAGS|signalFrame64RequiredFlags)
	binary.LittleEndian.PutUint64(raw[signalFrame64RSPOffset:signalFrame64RSPOffset+8], state.Get(corecpu.RSP))
	for reg := corecpu.Reg64(0); reg < corecpu.Reg64Count; reg++ {
		offset := signalFrame64RegsOffset + int(reg)*8
		binary.LittleEndian.PutUint64(raw[offset:offset+8], state.Get(reg))
	}
	return memory.Write(address, raw[:]) == nil
}

func rtSigreturn64(ctx *Context64, _ [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || ctx.Machine == nil {
		return int64(EFAULT)
	}
	frameAddress := corecpu.Address64(ctx.Machine.Get(corecpu.RSP))
	raw, ok := readSignalFrame64(ctx.Memory, frameAddress)
	if !ok {
		return int64(EFAULT)
	}
	if binary.LittleEndian.Uint64(raw[0:8]) != signalFrame64Magic || binary.LittleEndian.Uint64(raw[8:16]) != signalFrame64Version {
		return int64(EINVAL)
	}
	mask := binary.LittleEndian.Uint64(raw[signalFrame64MaskOffset : signalFrame64MaskOffset+8])
	fsBase := binary.LittleEndian.Uint64(raw[signalFrame64FSBaseOffset : signalFrame64FSBaseOffset+8])
	gsBase := binary.LittleEndian.Uint64(raw[signalFrame64GSBaseOffset : signalFrame64GSBaseOffset+8])
	rip := binary.LittleEndian.Uint64(raw[signalFrame64RIPOffset : signalFrame64RIPOffset+8])
	rflags := binary.LittleEndian.Uint64(raw[signalFrame64RFLAGSOffset : signalFrame64RFLAGSOffset+8])
	rsp := binary.LittleEndian.Uint64(raw[signalFrame64RSPOffset : signalFrame64RSPOffset+8])
	if !signalFrame64Canonical(fsBase) || !signalFrame64Canonical(gsBase) || !signalFrame64Canonical(rip) || !signalFrame64Canonical(rsp) {
		return int64(EINVAL)
	}
	if rflags&signalFrame64RequiredFlags == 0 || rflags&(3<<12) != 0 {
		return int64(EINVAL)
	}

	state := ctx.Machine
	state.SetRFLAGS(rflags)
	state.RIP = rip
	state.Set(corecpu.RSP, rsp)
	state.FSBase = fsBase
	state.GSBase = gsBase
	for reg := corecpu.Reg64(0); reg < corecpu.Reg64Count; reg++ {
		offset := signalFrame64RegsOffset + int(reg)*8
		state.Set(reg, binary.LittleEndian.Uint64(raw[offset:offset+8]))
	}
	state.Halted = false
	ctx.FSBase = fsBase
	ctx.GSBase = gsBase
	ctx.SignalMask = mask &^ signalFrame64ReservedMask
	ctx.SignalRestored = true
	return 0
}

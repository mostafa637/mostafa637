package wasmjit

import (
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"
	"golang.org/x/arch/x86/x86asm"
)

type memoryRef struct {
	base, index int16
	scale       uint8
	disp        int64
	rip         bool
	nextPC      uint64
}

func decodeMem(arg x86asm.Arg, nextPC uint64) (memoryRef, bool) {
	mem, ok := arg.(x86asm.Mem)
	if !ok || mem.Segment != 0 {
		return memoryRef{}, false
	}
	ref := memoryRef{base: -1, index: -1, scale: uint8(mem.Scale), disp: mem.Disp, nextPC: nextPC}
	if ref.scale == 0 {
		ref.scale = 1
	}
	if ref.scale != 1 && ref.scale != 2 && ref.scale != 4 && ref.scale != 8 {
		return memoryRef{}, false
	}
	if mem.Base == x86asm.RIP {
		ref.rip = true
	} else if mem.Base != 0 {
		var valid bool
		ref.base, valid = decodeReg(mem.Base)
		if !valid {
			return memoryRef{}, false
		}
	}
	if mem.Index != 0 {
		var valid bool
		ref.index, valid = decodeReg(mem.Index)
		if !valid {
			return memoryRef{}, false
		}
	}
	if ref.base < 0 && ref.index < 0 && !ref.rip {
		return memoryRef{}, false
	}
	return ref, true
}

func memoryInstruction(op machinecode.Op, dst, src int16, ref memoryRef) machinecode.Instruction {
	return machinecode.Instruction{Op: op, Dst: dst, Src: src, Imm: ref.disp,
		MemBase: ref.base, MemIndex: ref.index, MemScale: ref.scale,
		MemRIP: ref.rip, NextPC: ref.nextPC}
}

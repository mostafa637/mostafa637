package wasmjit

import (
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"
	"golang.org/x/arch/x86/x86asm"
)

func decodeExtend(inst x86asm.Inst, address uint64, signed bool) (machinecode.Instruction, error) {
	dst, ok := decodeReg(inst.Args[0])
	if !ok {
		return machinecode.Instruction{}, ErrUnsupported
	}
	dstWidth := registerWidth(inst.Args[0])
	if src, width, ok := extendRegSource(inst); ok {
		return machinecode.Instruction{Op: machinecode.OpExtend64, Dst: dst, Src: src,
			Width: width, DstWidth: dstWidth, Signed: signed}, nil
	}
	ref, width, ok := extendMemSource(inst, address)
	if !ok {
		return machinecode.Instruction{}, ErrUnsupported
	}
	item := memoryInstruction(machinecode.OpExtend64, dst, -1, ref)
	item.Width, item.DstWidth, item.Signed = width, dstWidth, signed
	return item, nil
}

func extendRegSource(inst x86asm.Inst) (int16, uint8, bool) {
	src, ok := decodeReg(inst.Args[1])
	if !ok {
		return 0, 0, false
	}
	return src, registerWidth(inst.Args[1]), validExtendWidth(registerWidth(inst.Args[1]))
}

func extendMemSource(inst x86asm.Inst, address uint64) (memoryRef, uint8, bool) {
	width := uint8(inst.MemBytes)
	if !validExtendWidth(width) {
		return memoryRef{}, 0, false
	}
	ref, ok := decodeMem(inst.Args[1], address+uint64(inst.Len))
	return ref, width, ok
}

func validExtendWidth(width uint8) bool {
	return width == 1 || width == 2 || width == 4
}

func registerWidth(arg x86asm.Arg) uint8 {
	reg, ok := arg.(x86asm.Reg)
	if !ok {
		return 0
	}
	switch {
	case reg >= x86asm.AL && reg <= x86asm.R15B:
		return 1
	case reg >= x86asm.AX && reg <= x86asm.R15W:
		return 2
	case reg >= x86asm.EAX && reg <= x86asm.R15L:
		return 4
	case reg >= x86asm.RAX && reg <= x86asm.R15:
		return 8
	default:
		return 0
	}
}

func decodeLEA(inst x86asm.Inst, address uint64) (machinecode.Instruction, error) {
	dst, ok := decodeReg(inst.Args[0])
	if !ok {
		return machinecode.Instruction{}, ErrUnsupported
	}
	ref, ok := decodeMem(inst.Args[1], address+uint64(inst.Len))
	if !ok {
		return machinecode.Instruction{}, ErrUnsupported
	}
	return memoryInstruction(machinecode.OpLEA64, dst, -1, ref), nil
}

package wasmjit

import (
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"
	"golang.org/x/arch/x86/x86asm"
)

func decodeShift(inst x86asm.Inst) (machinecode.Instruction, error) {
	dst, ok := decodeReg(inst.Args[0])
	if !ok {
		return machinecode.Instruction{}, ErrUnsupported
	}
	kind, ok := shiftKind(inst.Op)
	if !ok {
		return machinecode.Instruction{}, ErrUnsupported
	}
	out := machinecode.Instruction{Op: machinecode.OpShift64, Dst: dst, Src: -1}
	reg, _ := inst.Args[0].(x86asm.Reg)
	out.Width = uint8(regWidth(reg))
	out.ShiftKind = kind
	return decodeShiftCount(out, inst.Args[1])
}

func decodeShiftCount(out machinecode.Instruction, arg x86asm.Arg) (machinecode.Instruction, error) {
	if imm, ok := decodeImm(arg); ok {
		out.Imm = imm
		return out, nil
	}
	reg, ok := arg.(x86asm.Reg)
	if !ok || reg != x86asm.CL {
		return machinecode.Instruction{}, ErrUnsupported
	}
	out.Src, _ = decodeReg(reg)
	return out, nil
}

func shiftKind(op x86asm.Op) (uint8, bool) {
	switch op {
	case x86asm.SHL:
		return machinecode.ShiftSHL, true
	case x86asm.SHR:
		return machinecode.ShiftSHR, true
	case x86asm.SAR:
		return machinecode.ShiftSAR, true
	case x86asm.ROL:
		return machinecode.ShiftROL, true
	case x86asm.ROR:
		return machinecode.ShiftROR, true
	case x86asm.RCL:
		return machinecode.ShiftRCL, true
	case x86asm.RCR:
		return machinecode.ShiftRCR, true
	default:
		return 0, false
	}
}

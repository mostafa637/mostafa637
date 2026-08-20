package llvmir

import (
	"fmt"

	"golang.org/x/arch/arm64/arm64asm"
)

var ErrUnsupportedARM64 = fmt.Errorf("llvmir: unsupported ARM64 instruction")

func FromARM64(inst arm64asm.Inst) (Program, error) {
	switch inst.Op {
	case arm64asm.NOP:
		return Program{Name: "arm64_nop"}, nil
	case arm64asm.ADD, arm64asm.ADDS:
		return arm64Add(inst, false)
	case arm64asm.SUB, arm64asm.SUBS:
		return arm64Add(inst, true)
	case arm64asm.MOVZ:
		return arm64Move(inst), nil
	default:
		return Program{}, ErrUnsupportedARM64
	}
}

func arm64Add(inst arm64asm.Inst, subtract bool) (Program, error) {
	if _, ok := inst.Args[2].(arm64asm.ImmShift); !ok {
		return Program{}, ErrUnsupportedARM64
	}
	kind := OpAdd
	if subtract {
		kind = OpSub
	}
	return Program{Name: "arm64_alu", Ops: []Op{{Kind: kind, Value: addSubImm(inst.Enc)}}}, nil
}

func arm64Move(inst arm64asm.Inst) Program {
	value := uint64((inst.Enc >> 5) & 0xffff)
	value <<= 16 * ((inst.Enc >> 21) & 3)
	return Program{Name: "arm64_movz", Ops: []Op{{Kind: OpConst, Value: value}}}
}

func addSubImm(enc uint32) uint64 {
	value := uint64((enc >> 10) & 0xfff)
	if enc&(1<<22) != 0 {
		return value << 12
	}
	return value
}

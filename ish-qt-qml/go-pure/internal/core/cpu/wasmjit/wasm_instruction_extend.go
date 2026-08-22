package wasmjit

import "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"

func emitExtend(inst machinecode.Instruction) []byte {
	out := extendSource(inst)
	out = append(out, localCode(17)...)
	out = append(out, constCode(widthMask(inst.Width))...)
	out = append(out, WasmOpI64And, WasmOpLocalSet, 17)
	if inst.Signed {
		out = append(out, signExtend(17, inst.Width)...)
	}
	if inst.DstWidth == 4 {
		out = append(out, localCode(17)...)
		out = append(out, constCode(0xffffffff)...)
		out = append(out, WasmOpI64And, WasmOpLocalSet, 17)
	}
	out = append(out, localCode(17)...)
	return append(out, WasmOpLocalSet, byte(inst.Dst))
}

func extendSource(inst machinecode.Instruction) []byte {
	if inst.Src >= 0 {
		out := append(localCode(inst.Src), WasmOpLocalSet, 17)
		return out
	}
	load := inst
	load.Op, load.Dst = machinecode.OpLoad64, 17
	return emitMemoryLoad(load)
}

func signExtend(local int16, width uint8) []byte {
	shift := int64(64 - uint64(width)*8)
	out := append(localCode(local), constCode(shift)...)
	out = append(out, WasmOpI64Shl)
	out = append(out, constCode(shift)...)
	out = append(out, WasmOpI64ShrS)
	return append(out, WasmOpLocalSet, byte(local))
}

func widthMask(width uint8) int64 {
	switch width {
	case 1:
		return 0xff
	case 2:
		return 0xffff
	case 4:
		return 0xffffffff
	default:
		return -1
	}
}

package wasmjit

const (
	WasmOpLocalGet      byte = 0x20
	WasmOpLocalSet      byte = 0x21
	WasmOpI32Const      byte = 0x41
	WasmOpI64Const      byte = 0x42
	WasmOpI64Add        byte = 0x7c
	WasmOpI64Sub        byte = 0x7d
	WasmOpI64Mul        byte = 0x7e
	WasmOpI64DivS       byte = 0x7f
	WasmOpI64DivU       byte = 0x80
	WasmOpI64RemS       byte = 0x81
	WasmOpI64RemU       byte = 0x82
	WasmOpI64And        byte = 0x83
	WasmOpI64Or         byte = 0x84
	WasmOpI64Xor        byte = 0x85
	WasmOpI64Shl        byte = 0x86
	WasmOpI64ShrS       byte = 0x87
	WasmOpI64ShrU       byte = 0x88
	WasmOpI64Eq         byte = 0x51
	WasmOpI64Ne         byte = 0x52
	WasmOpI64LtS        byte = 0x53
	WasmOpI64LtU        byte = 0x54
	WasmOpI64GtS        byte = 0x55
	WasmOpI64GtU        byte = 0x56
	WasmOpI64LeS        byte = 0x57
	WasmOpI64LeU        byte = 0x58
	WasmOpI64GeS        byte = 0x59
	WasmOpI64GeU        byte = 0x5a
	WasmOpI64Eqz        byte = 0x50
	WasmOpI32Eqz        byte = 0x45
	WasmOpI32And        byte = 0x71
	WasmOpI32Or         byte = 0x72
	WasmOpI32Xor        byte = 0x73
	WasmOpI32WrapI64    byte = 0xa7
	WasmOpI64ExtendI32U byte = 0xad
	WasmOpIf            byte = 0x04
	WasmOpElse          byte = 0x05
	WasmOpEnd           byte = 0x0b
	WasmOpReturn        byte = 0x0f
	WasmOpLoop          byte = 0x03
	WasmOpBr            byte = 0x0c
	WasmOpBrIf          byte = 0x0d
	WasmOpBlock         byte = 0x02
)

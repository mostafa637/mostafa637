package wasmjit

import "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"

func EmitBlock(block GuestBlock) ([]byte, error) {
	if block.Arch != "" && block.Arch != "amd64" {
		return nil, ErrUnsupported
	}
	insts, err := decodeX86(block.Bytes, block.PC)
	if err != nil {
		return nil, err
	}
	return emitModule(insts), nil
}

func emitModule(insts []machinecode.Instruction) []byte {
	body := emitBody(insts)
	module := []byte{0x00, 0x61, 0x73, 0x6d, 1, 0, 0, 0}
	module = append(module, wasmSection(1, wasmType())...)
	module = append(module, wasmSection(2, wasmImport())...)
	module = append(module, wasmSection(3, []byte{1, 0})...)
	module = append(module, wasmSection(5, []byte{1, 0, 1})...)
	module = append(module, wasmSection(7, wasmExport())...)
	return append(module, wasmSection(10, wasmCode(body))...)
}

func wasmType() []byte {
	out := []byte{4} // 4 types
	// Type 0: (17 i64) -> (27 i64) for run
	out = append(out, 0x60)
	out = appendULEB(out, 17)
	for i := 0; i < 17; i++ {
		out = append(out, 0x7e)
	}
	out = appendULEB(out, 27)
	for i := 0; i < 27; i++ {
		out = append(out, 0x7e)
	}
	// Type 1: (7 i64) -> (1 i64) for syscall64
	out = append(out, 0x60, 7)
	for i := 0; i < 7; i++ {
		out = append(out, 0x7e)
	}
	out = append(out, 1, 0x7e)
	// Type 2: (1 i64) -> (1 i64) for load64
	out = append(out, 0x60, 1, 0x7e, 1, 0x7e)
	// Type 3: (2 i64) -> (0) for store64
	out = append(out, 0x60, 2, 0x7e, 0x7e, 0)
	return out
}

func wasmImport() []byte {
	out := []byte{3}
	out = appendImport(out, "syscall64", 1)
	out = appendImport(out, "load64", 2)
	return appendImport(out, "store64", 3)
}

func appendImport(out []byte, name string, typ byte) []byte {
	out = append(out, 3, 'e', 'n', 'v', byte(len(name)))
	out = append(out, []byte(name)...)
	return append(out, 0, typ)
}

func wasmExport() []byte {
	// Function run is index 3 (after 3 imports), memory is index 0
	return []byte{2, 3, 'r', 'u', 'n', 0, 3, 6, 'm', 'e', 'm', 'o', 'r', 'y', 2, 0}
}

func wasmCode(body []byte) []byte {
	out := append([]byte{1}, appendULEB(nil, uint32(len(body)))...)
	return append(out, body...)
}

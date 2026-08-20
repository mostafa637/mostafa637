package wasmjit

import "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"

func EmitBlock(block GuestBlock) ([]byte, error) {
	if block.Arch != "" && block.Arch != "amd64" {
		return nil, ErrUnsupported
	}
	insts, err := decodeX86(block.Bytes)
	if err != nil {
		return nil, err
	}
	return emitModule(insts), nil
}

func emitModule(insts []machinecode.Instruction) []byte {
	body := emitBody(insts)
	imported := hasSyscall(insts)
	module := []byte{0x00, 0x61, 0x73, 0x6d, 1, 0, 0, 0}
	module = append(module, wasmSection(1, wasmType(imported))...)
	if imported {
		module = append(module, wasmSection(2, wasmImport())...)
	}
	module = append(module, wasmSection(3, []byte{1, 0})...)
	module = append(module, wasmSection(5, []byte{1, 0, 1})...)
	module = append(module, wasmSection(7, wasmExport(imported))...)
	return append(module, wasmSection(10, wasmCode(body))...)
}

func hasSyscall(insts []machinecode.Instruction) bool {
	for _, inst := range insts {
		if inst.Op == machinecode.OpSyscall {
			return true
		}
	}
	return false
}

func wasmType(imported bool) []byte {
	count := byte(1)
	if imported {
		count = 2
	}
	out := []byte{count, 0x60, 16}
	for i := 0; i < 16; i++ {
		out = append(out, 0x7e)
	}
	out = append(out, 16)
	for i := 0; i < 16; i++ {
		out = append(out, 0x7e)
	}
	if !imported {
		return out
	}
	out = append(out, 0x60, 7)
	for i := 0; i < 7; i++ {
		out = append(out, 0x7e)
	}
	return append(out, 1, 0x7e)
}

func wasmImport() []byte {
	return []byte{1, 3, 'e', 'n', 'v', 9, 's', 'y', 's', 'c', 'a', 'l', 'l', '6', '4', 0, 1}
}

func wasmExport(imported bool) []byte {
	index := byte(0)
	if imported {
		index = 1
	}
	return []byte{2, 3, 'r', 'u', 'n', 0, index, 6, 'm', 'e', 'm', 'o', 'r', 'y', 2, 0}
}

func wasmCode(body []byte) []byte {
	out := append([]byte{1}, appendULEB(nil, uint32(len(body)))...)
	return append(out, body...)
}

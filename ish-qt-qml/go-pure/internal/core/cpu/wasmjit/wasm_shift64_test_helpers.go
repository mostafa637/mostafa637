package wasmjit

import (
	"context"
	"encoding/binary"
	"testing"
)

func runRaw64(t *testing.T, code []byte, flags uint64) ([16]uint64, uint64) {
	t.Helper()
	host, err := CompileBlock(context.Background(), GuestBlock{Bytes: code, Arch: "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close(context.Background())
	regs, got, err := host.RunRegsFlags(context.Background(), [16]uint64{}, flags)
	if err != nil {
		t.Fatal(err)
	}
	return regs, got
}

func codeImm(op1, op2, count byte, value uint64) []byte {
	code := movRegImm(0, value)
	return append(code, 0x48, op1, op2, count, 0xc3)
}

func movRegImm(reg byte, value uint64) []byte {
	code := []byte{0x48, 0xb8 + reg, 0, 0, 0, 0, 0, 0, 0, 0}
	binary.LittleEndian.PutUint64(code[2:], value)
	return code
}

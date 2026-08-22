package wasmjit

import (
	"context"
	"encoding/binary"
	"math"
	"testing"

	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"
)

func mulDivCode(bytes []byte) []byte {
	return append(bytes, 0xc3)
}

func codeMUL(srcReg byte, value uint64) []byte {
	code := movRegImm(0, value)
	code = append(code, 0x48, 0xf7)
	return mulDivCode(append(code, 0xe0+srcReg))
}

func codeIMUL2(dst, src byte, value uint64) []byte {
	code := movRegImm(dst, value)
	code = append(code, 0x48, 0xf7)
	return mulDivCode(append(code, 0xe8+src))
}

func codeIMUL3(dst, src byte, value uint64, imm int64) []byte {
	code := movRegImm(dst, value)
	code = append(code, movRegImm(src, 0)...)
	immBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(immBytes, uint32(imm))
	code = append(code, 0x48, 0x69)
	code = append(code, 0xc0+dst, immBytes[0], immBytes[1], immBytes[2], immBytes[3])
	return mulDivCode(code)
}

func codeDIV(srcReg byte, rax, rdx uint64) []byte {
	code := movRegImm(0, rax)
	code = append(code, movRegImm(2, rdx)...)
	code = append(code, 0x48, 0xf7)
	return mulDivCode(append(code, 0xf0+srcReg))
}

func codeIDIV(srcReg byte, rax, rdx uint64) []byte {
	code := movRegImm(0, rax)
	code = append(code, movRegImm(2, rdx)...)
	code = append(code, 0x48, 0xf7)
	return mulDivCode(append(code, 0xf8+srcReg))
}

func TestWasmMUL64(t *testing.T) {
	regs, got := runRaw64(t, codeMUL(0, 0x123456789abcdef0), 0)
	if regs[0] != 0 || regs[2] != 0 {
		t.Fatalf("MUL small: RAX=%d RDX=%d", regs[0], regs[2])
	}
	_ = got
}

func TestWasmMUL64Wide(t *testing.T) {
	regs, _ := runRaw64(t, codeMUL(1, math.MaxUint64), 0)
	var wantRAX uint64 = math.MaxUint64
	var wantRDX uint64 = math.MaxUint64 - 1
	if regs[0] != wantRAX || regs[2] != wantRDX {
		t.Fatalf("MUL wide: RAX=%x RDX=%x want RAX=%x RDX=%x",
			regs[0], regs[2], wantRAX, wantRDX)
	}
}

func TestWasmMUL64Flag(t *testing.T) {
	regs, got := runRaw64(t, codeMUL(0, math.MaxUint64), 0)
	var wantRDX uint64 = math.MaxUint64 - 1
	if regs[0] != 0 || regs[2] != wantRDX {
		t.Fatalf("MUL flag result: RAX=%x RDX=%x", regs[0], regs[2])
	}
	if got&flagCF == 0 {
		t.Fatalf("MUL wide: CF must be set, flags=%x", got)
	}
}

func TestWasmIMUL64Flag(t *testing.T) {
	regs, got := runRaw64(t, codeIMUL2(1, 0, 0x8000000000000000), 0)
	if regs[0] != 0 {
		t.Fatalf("IMUL result: RAX=%x", regs[0])
	}
	if regs[2] != 0x8000000000000000 {
		t.Fatalf("IMUL sign extension: RDX=%x want 0x8000000000000000", regs[2])
	}
	if got&flagCF == 0 {
		t.Fatalf("IMUL sign overflow: CF must be set, flags=%x", got)
	}
}

func TestWasmIMUL64Small(t *testing.T) {
	regs, got := runRaw64(t, codeIMUL2(0, 0, 0x123456789abcdef0), 0)
	if regs[0] != 0 || regs[2] != 0 {
		t.Fatalf("IMUL small: RAX=%x RDX=%x", regs[0], regs[2])
	}
	if got&flagCF != 0 {
		t.Fatalf("IMUL small: CF must be clear, flags=%x", got)
	}
}

func TestWasmDIV64(t *testing.T) {
	regs, _ := runRaw64(t, codeDIV(1, 100, 0), 0)
	if regs[0] != 100 || regs[2] != 0 {
		t.Fatalf("DIV: RAX=%d RDX=%d", regs[0], regs[2])
	}
}

func TestWasmDIV64Remainder(t *testing.T) {
	regs, _ := runRaw64(t, codeDIV(1, 103, 0), 0)
	if regs[0] != 34 || regs[2] != 1 {
		t.Fatalf("DIV remainder: RAX=%d RDX=%d", regs[0], regs[2])
	}
}

func TestWasmDIV64Wide(t *testing.T) {
	regs, _ := runRaw64(t, codeDIV(1, math.MaxUint64, 1), 0)
	if regs[0] != 0x5555555555555555 || regs[2] != 0 {
		t.Fatalf("DIV wide: RAX=%x RDX=%x", regs[0], regs[2])
	}
}

func TestWasmIDIV64(t *testing.T) {
	regs, _ := runRaw64(t, codeIDIV(1, 100, 0), 0)
	if regs[0] != 100 || regs[2] != 0 {
		t.Fatalf("IDIV positive: RAX=%d RDX=%d", regs[0], regs[2])
	}
}

func TestWasmIDIV64Sign(t *testing.T) {
	regs, _ := runRaw64(t, codeIDIV(1, math.MaxUint64, 0xffffffffffffffff), 0)
	if regs[0] != 1 || regs[2] != 0 {
		t.Fatalf("IDIV sign: RAX=%d RDX=%d", regs[0], regs[2])
	}
}

func TestWasmIMUL364(t *testing.T) {
	regs, got := runRaw64(t, codeIMUL3(3, 1, 100, 25), 0)
	if regs[3] != 2500 {
		t.Fatalf("IMUL3: RDX=%d want 2500", regs[3])
	}
	if got&flagCF != 0 {
		t.Fatalf("IMUL3: CF must be clear, flags=%x", got)
	}
}

func TestWasmMULDecode(t *testing.T) {
	code := codeMUL(0, 5)
	host, err := CompileBlock(context.Background(), GuestBlock{Bytes: code, Arch: "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close(context.Background())
	if host.flow.Op != machinecode.OpMUL64 {
		t.Fatalf("flow op = %v want OpMUL64", host.flow.Op)
	}
}

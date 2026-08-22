package wasmjit

import (
	"context"
	"encoding/binary"
	"math"
	"math/bits"
	"testing"

	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"
)

func mulDivCode(code []byte) []byte { return append(code, 0xc3) }

func codeMul(op byte, a, b uint64) []byte {
	code := append(movRegImm(0, a), movRegImm(1, b)...)
	return mulDivCode(append(code, 0x48, 0xf7, op))
}

func codeDiv(op byte, rax, rdx, divisor uint64) []byte {
	code := append(movRegImm(0, rax), movRegImm(2, rdx)...)
	code = append(code, movRegImm(1, divisor)...)
	return mulDivCode(append(code, 0x48, 0xf7, op))
}

func codeIMUL3(a, b uint64, imm int32) []byte {
	code := append(movRegImm(0, a), movRegImm(1, b)...)
	var raw [4]byte
	binary.LittleEndian.PutUint32(raw[:], uint32(imm))
	code = append(code, 0x48, 0x69, 0xc1)
	return mulDivCode(append(code, raw[:]...))
}

func TestWasmMUL64(t *testing.T) {
	a, b := uint64(0x123456789abcdef0), uint64(0xfedcba9876543210)
	regs, _ := runRaw64(t, codeMul(0xe1, a, b), 0)
	hi, lo := bits.Mul64(a, b)
	if regs[0] != lo || regs[2] != hi {
		t.Fatalf("MUL=%x:%x want %x:%x", regs[2], regs[0], hi, lo)
	}
}

func TestWasmMUL64Flag(t *testing.T) {
	regs, flags := runRaw64(t, codeMul(0xe1, math.MaxUint64, 2), 0)
	if regs[0] != math.MaxUint64-1 || regs[2] != 1 || flags&flagCF == 0 {
		t.Fatalf("MUL result=%x:%x flags=%x", regs[2], regs[0], flags)
	}
}

func TestWasmIMUL64(t *testing.T) {
	regs, flags := runRaw64(t, codeMul(0xe9, ^uint64(1), 3), 0)
	if regs[0] != ^uint64(5) || regs[2] != math.MaxUint64 || flags&flagCF != 0 {
		t.Fatalf("IMUL=%x:%x flags=%x", regs[2], regs[0], flags)
	}
}

func TestWasmDIV64(t *testing.T) {
	regs, _ := runRaw64(t, codeDiv(0xf1, 103, 0, 3), 0)
	if regs[0] != 34 || regs[2] != 1 {
		t.Fatalf("DIV=%d:%d", regs[0], regs[2])
	}
}

func TestWasmDIV64Wide(t *testing.T) {
	regs, _ := runRaw64(t, codeDiv(0xf1, math.MaxUint64, 1, 3), 0)
	if regs[0] != 0xaaaaaaaaaaaaaaaa || regs[2] != 1 {
		t.Fatalf("DIV wide=%x:%x", regs[0], regs[2])
	}
}

func TestWasmIDIV64(t *testing.T) {
	regs, _ := runRaw64(t, codeDiv(0xf9, ^uint64(102), 0, ^uint64(2)), 0)
	if regs[0] != 34 || regs[2] != ^uint64(0) {
		t.Fatalf("IDIV=%x:%x", regs[0], regs[2])
	}
}

func TestWasmIMUL3(t *testing.T) {
	regs, flags := runRaw64(t, codeIMUL3(100, 4, 25), 0)
	if regs[0] != 100 || flags&flagCF != 0 {
		t.Fatalf("IMUL3=%x flags=%x", regs[0], flags)
	}
}

func TestWasmMULDecode(t *testing.T) {
	host, err := CompileBlock(context.Background(), GuestBlock{Bytes: codeMul(0xe1, 2, 3), Arch: "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close(context.Background())
	if host.flow.Op != machinecode.OpMUL64 {
		t.Fatalf("flow op=%v", host.flow.Op)
	}
}

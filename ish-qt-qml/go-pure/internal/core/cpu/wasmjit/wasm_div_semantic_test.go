package wasmjit

import "testing"

func codeDIVReady(rax, rdx, divisor uint64, signed bool) []byte {
	code := movRegImm(0, rax)
	code = append(code, movRegImm(2, rdx)...)
	code = append(code, movRegImm(1, divisor)...)
	code = append(code, 0x48, 0xf7)
	op := byte(0xf0)
	if signed {
		op = 0xf8
	}
	return append(code, op+1, 0xc3)
}

func TestWasmDIVSemantic(t *testing.T) {
	regs, _ := runRaw64(t, codeDIVReady(103, 0, 3, false), 0)
	if regs[0] != 34 || regs[2] != 1 {
		t.Fatalf("quotient=%d remainder=%d", regs[0], regs[2])
	}
}

func TestWasmDIVWideSemantic(t *testing.T) {
	regs, _ := runRaw64(t, codeDIVReady(^uint64(0), 1, 3, false), 0)
	if regs[0] != 0xaaaaaaaaaaaaaaaa || regs[2] != 1 {
		t.Fatalf("quotient=%x remainder=%x", regs[0], regs[2])
	}
}

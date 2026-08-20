package wasmjit

import "testing"

func TestWasmSHL64(t *testing.T) {
	regs, flags := runRaw64(t, codeImm(0xc1, 0xe0, 1, 0x8000000000000000), 0)
	want := flagCF | flagOF | flagZF | flagPF
	if regs[0] != 0 || flags&want != want || flags&flagSF != 0 {
		t.Fatalf("rax=%#x flags=%#x", regs[0], flags)
	}
}

func TestWasmSHR64(t *testing.T) {
	regs, flags := runRaw64(t, codeImm(0xc1, 0xe8, 1, 0x8000000000000000), 0)
	if regs[0] != 0x4000000000000000 || flags&flagCF != 0 || flags&(flagOF|flagPF) != (flagOF|flagPF) {
		t.Fatalf("rax=%#x flags=%#x", regs[0], flags)
	}
}

func TestWasmSAR64(t *testing.T) {
	regs, flags := runRaw64(t, codeImm(0xc1, 0xf8, 1, 0x8000000000000000), 0)
	if regs[0] != 0xc000000000000000 || flags&flagSF == 0 || flags&flagOF != 0 {
		t.Fatalf("rax=%#x flags=%#x", regs[0], flags)
	}
}

func TestWasmSHLCL64(t *testing.T) {
	code := append(movRegImm(0, 1), movRegImm(1, 3)...)
	code = append(code, 0x48, 0xd3, 0xe0, 0xc3)
	regs, _ := runRaw64(t, code, 0)
	if regs[0] != 8 {
		t.Fatalf("rax=%#x, want 8", regs[0])
	}
}

func TestWasmROL64(t *testing.T) {
	regs, flags := runRaw64(t, codeImm(0xc1, 0xc0, 1, 0x8000000000000000), 0)
	if regs[0] != 1 || flags&flagCF == 0 || flags&flagOF == 0 {
		t.Fatalf("rax=%#x flags=%#x", regs[0], flags)
	}
}

func TestWasmROR64(t *testing.T) {
	regs, flags := runRaw64(t, codeImm(0xc1, 0xc8, 1, 1), 0)
	if regs[0] != 0x8000000000000000 || flags&flagCF == 0 || flags&flagOF == 0 {
		t.Fatalf("rax=%#x flags=%#x", regs[0], flags)
	}
}

func TestWasmRCL64(t *testing.T) {
	code := codeImm(0xc1, 0xd0, 1, 0x8000000000000000)
	regs, flags := runRaw64(t, code, flagCF)
	if regs[0] != 1 || flags&(flagCF|flagOF) != (flagCF|flagOF) {
		t.Fatalf("rax=%#x flags=%#x", regs[0], flags)
	}
}

func TestWasmRCR64(t *testing.T) {
	code := codeImm(0xc1, 0xd8, 1, 1)
	regs, flags := runRaw64(t, code, flagCF)
	if regs[0] != 0x8000000000000000 || flags&(flagCF|flagOF) != (flagCF|flagOF) {
		t.Fatalf("rax=%#x flags=%#x", regs[0], flags)
	}
}

func TestWasmShiftCountZeroPreservesFlags(t *testing.T) {
	flags := flagCF | flagZF | flagOF
	code := append(movRegImm(0, 1), 0x48, 0xd3, 0xe0, 0xc3)
	regs, got := runRaw64(t, code, flags)
	if regs[0] != 1 || got != flags {
		t.Fatalf("rax=%#x flags=%#x, want %#x", regs[0], got, flags)
	}
}

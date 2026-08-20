package machinecode

import "testing"

func TestEmitX86(t *testing.T) {
	code, err := EmitX86(sampleProgram())
	if err != nil {
		t.Fatal(err)
	}
	if len(code) == 0 {
		t.Fatal("x86 emitter returned empty code")
	}
}

func TestEmitARM64(t *testing.T) {
	code, err := EmitARM64(sampleProgram())
	if err != nil {
		t.Fatal(err)
	}
	if len(code) == 0 || len(code)%4 != 0 {
		t.Fatalf("invalid ARM64 code size: %d", len(code))
	}
}

func TestRejectsUnknown(t *testing.T) {
	if _, err := EmitX86([]Instruction{{Op: 99}}); err != ErrUnsupported {
		t.Fatalf("unexpected error: %v", err)
	}
}

func sampleProgram() []Instruction {
	return []Instruction{
		{Op: OpMOVImm, Dst: 0, Imm: 7},
		{Op: OpADDImm, Dst: 0, Src: 1, Imm: 2},
		{Op: OpSUBImm, Dst: 0, Src: 1, Imm: 1},
		{Op: OpRET},
	}
}

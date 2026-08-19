package cpu

import "testing"

func TestDecodeX86CMOVccRegister(t *testing.T) {
	memory, _ := mappedCode(t, []byte{0x0F, 0x44, 0xCB}) // cmove ecx, ebx
	instruction, err := Decode(memory, PageSize)
	if err != nil {
		t.Fatal(err)
	}
	if instruction.Op != OpCMOVcc {
		t.Fatalf("op = %v, want OpCMOVcc", instruction.Op)
	}
	if instruction.Len != 3 {
		t.Fatalf("length = %d, want 3", instruction.Len)
	}
	if instruction.Group != 2 {
		t.Fatalf("condition group = %d, want ZF/E", instruction.Group)
	}
	if instruction.Dst.IsMem || instruction.Dst.Reg != ECX || instruction.Dst.Width != 4 {
		t.Fatalf("unexpected destination: %#v", instruction.Dst)
	}
	if instruction.Src.IsMem || instruction.Src.Reg != EBX || instruction.Src.Width != 4 {
		t.Fatalf("unexpected source: %#v", instruction.Src)
	}
}

func TestDecodeX86CMOVccMemorySource(t *testing.T) {
	memory, _ := mappedCode(t, []byte{0x0F, 0x45, 0x0B}) // cmovne ecx, dword ptr [ebx]
	instruction, err := Decode(memory, PageSize)
	if err != nil {
		t.Fatal(err)
	}
	if instruction.Op != OpCMOVcc {
		t.Fatalf("op = %v, want OpCMOVcc", instruction.Op)
	}
	if instruction.Group != 3 {
		t.Fatalf("condition group = %d, want !ZF/NE", instruction.Group)
	}
	if instruction.Dst.IsMem || instruction.Dst.Reg != ECX || instruction.Dst.Width != 4 {
		t.Fatalf("unexpected destination: %#v", instruction.Dst)
	}
	if !instruction.Src.IsMem || instruction.Src.Memory.Base != EBX || !instruction.Src.Memory.HasBase || instruction.Src.Width != 4 {
		t.Fatalf("unexpected memory source: %#v", instruction.Src)
	}
}

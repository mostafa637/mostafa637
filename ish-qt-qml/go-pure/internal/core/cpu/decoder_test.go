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

func TestDecodeXchgRegister(t *testing.T) {
	memory, _ := mappedCode(t, []byte{0x87, 0xC3}) // xchg eax, ebx
	instruction, err := Decode(memory, PageSize)
	if err != nil {
		t.Fatal(err)
	}
	if instruction.Op != OpXchg || instruction.Len != 2 {
		t.Fatalf("instruction = %#v, want OpXchg length 2", instruction)
	}
	if instruction.Dst.IsMem || instruction.Dst.Reg != EBX || instruction.Dst.Width != 4 {
		t.Fatalf("unexpected XCHG destination: %#v", instruction.Dst)
	}
	if instruction.Src.IsMem || instruction.Src.Reg != EAX || instruction.Src.Width != 4 {
		t.Fatalf("unexpected XCHG source: %#v", instruction.Src)
	}
}

func TestDecodeXchgMemory(t *testing.T) {
	memory, _ := mappedCode(t, []byte{0x87, 0x03}) // xchg dword ptr [ebx], eax
	instruction, err := Decode(memory, PageSize)
	if err != nil {
		t.Fatal(err)
	}
	if instruction.Op != OpXchg {
		t.Fatalf("op = %v, want OpXchg", instruction.Op)
	}
	if !instruction.Dst.IsMem || instruction.Dst.Memory.Base != EBX || !instruction.Dst.Memory.HasBase || instruction.Dst.Width != 4 {
		t.Fatalf("unexpected memory destination: %#v", instruction.Dst)
	}
	if instruction.Src.IsMem || instruction.Src.Reg != EAX || instruction.Src.Width != 4 {
		t.Fatalf("unexpected register source: %#v", instruction.Src)
	}
}

func TestDecodeXchgByteRegisters(t *testing.T) {
	memory, _ := mappedCode(t, []byte{0x86, 0xDC}) // xchg ah, bl
	instruction, err := Decode(memory, PageSize)
	if err != nil {
		t.Fatal(err)
	}
	if instruction.Op != OpXchg {
		t.Fatalf("op = %v, want OpXchg", instruction.Op)
	}
	if instruction.Dst.IsMem || instruction.Dst.Reg != EAX || instruction.Dst.Width != 1 || instruction.Dst.ByteOffset != 1 {
		t.Fatalf("unexpected byte destination: %#v", instruction.Dst)
	}
	if instruction.Src.IsMem || instruction.Src.Reg != EBX || instruction.Src.Width != 1 || instruction.Src.ByteOffset != 0 {
		t.Fatalf("unexpected byte source: %#v", instruction.Src)
	}
}

func TestDecodeX86AddSubMemoryOperands(t *testing.T) {
	tests := []struct {
		name string
		code []byte
		op   Op
		dst  Operand
		src  Operand
	}{
		{
			name: "add memory destination",
			code: []byte{0x01, 0x03}, // add dword ptr [ebx], eax
			op:   OpAddOperands,
			dst:  Operand{IsMem: true, Width: 4, Memory: MemoryOperand{Base: EBX, HasBase: true, Scale: 1}},
			src:  regOperand(EAX),
		},
		{
			name: "sub memory destination",
			code: []byte{0x29, 0x03}, // sub dword ptr [ebx], eax
			op:   OpSubOperands,
			dst:  Operand{IsMem: true, Width: 4, Memory: MemoryOperand{Base: EBX, HasBase: true, Scale: 1}},
			src:  regOperand(EAX),
		},
		{
			name: "add register destination",
			code: []byte{0x03, 0x03}, // add eax, dword ptr [ebx]
			op:   OpAddOperands,
			dst:  regOperand(EAX),
			src:  Operand{IsMem: true, Width: 4, Memory: MemoryOperand{Base: EBX, HasBase: true, Scale: 1}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			memory, _ := mappedCode(t, test.code)
			instruction, err := Decode(memory, PageSize)
			if err != nil {
				t.Fatal(err)
			}
			if instruction.Op != test.op || instruction.Len != uint32(len(test.code)) {
				t.Fatalf("instruction = %#v, want op %v len %d", instruction, test.op, len(test.code))
			}
			if instruction.Dst != test.dst {
				t.Fatalf("destination = %#v, want %#v", instruction.Dst, test.dst)
			}
			if instruction.Src != test.src {
				t.Fatalf("source = %#v, want %#v", instruction.Src, test.src)
			}
		})
	}
}

func TestDecodeX86AddSubMemoryImmediate(t *testing.T) {
	for _, test := range []struct {
		name string
		code []byte
		op   Op
		imm  int32
	}{
		{name: "add", code: []byte{0x83, 0x03, 0x05}, op: OpAddOperandImm, imm: 5},
		{name: "sub", code: []byte{0x83, 0x2B, 0x05}, op: OpSubOperandImm, imm: 5},
	} {
		t.Run(test.name, func(t *testing.T) {
			memory, _ := mappedCode(t, test.code)
			instruction, err := Decode(memory, PageSize)
			if err != nil {
				t.Fatal(err)
			}
			if instruction.Op != test.op || instruction.Imm != test.imm || !instruction.Dst.IsMem || instruction.Dst.Width != 4 {
				t.Fatalf("instruction = %#v, want op %v imm %d memory destination", instruction, test.op, test.imm)
			}
		})
	}
}

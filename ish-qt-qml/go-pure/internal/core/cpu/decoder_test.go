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

func TestDecodeX86MulDiv(t *testing.T) {
	tests := []struct {
		name string
		code []byte
		op   Op
		dst  Reg32
		src  Reg32
		imm  int32
		grp  uint8
	}{
		{name: "mul implicit", code: []byte{0xF7, 0xE3}, op: OpMulImplicit, src: EBX},
		{name: "imul implicit", code: []byte{0xF7, 0xEB}, op: OpIMulImplicit, src: EBX},
		{name: "div implicit", code: []byte{0xF7, 0xF3}, op: OpDivImplicit, src: EBX},
		{name: "idiv implicit", code: []byte{0xF7, 0xFB}, op: OpIDivImplicit, src: EBX},
		{name: "imul two operands", code: []byte{0x0F, 0xAF, 0xC3}, op: OpIMulOperands, dst: EAX, src: EBX},
		{name: "imul imm8", code: []byte{0x6B, 0xC3, 0x05}, op: OpIMulOperands, dst: EAX, src: EBX, imm: 5, grp: 1},
		{name: "imul imm32", code: []byte{0x69, 0xC3, 0x05, 0x00, 0x00, 0x00}, op: OpIMulOperands, dst: EAX, src: EBX, imm: 5, grp: 1},
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
			if test.op == OpIMulOperands {
				if instruction.Dst.IsMem || instruction.Dst.Reg != test.dst || instruction.Src.IsMem || instruction.Src.Reg != test.src || instruction.Dst.Width != 4 || instruction.Src.Width != 4 {
					t.Fatalf("operands = dst %#v src %#v", instruction.Dst, instruction.Src)
				}
				if instruction.Imm != test.imm || instruction.Group != test.grp {
					t.Fatalf("IMUL metadata = imm %d group %d, want imm %d group %d", instruction.Imm, instruction.Group, test.imm, test.grp)
				}
			} else if instruction.Src.IsMem || instruction.Src.Reg != test.src || instruction.Src.Width != 4 {
				t.Fatalf("source = %#v, want register %v width 4", instruction.Src, test.src)
			}
		})
	}
}

func TestDecodeX86StringInstructions(t *testing.T) {
	tests := []struct {
		name  string
		code  []byte
		op    Op
		width int32
		rep   uint8
	}{
		{name: "movsb", code: []byte{0xA4}, op: OpMovs, width: 1},
		{name: "movsw", code: []byte{0x66, 0xA5}, op: OpMovs, width: 2},
		{name: "rep movsw", code: []byte{0xF3, 0x66, 0xA5}, op: OpMovs, width: 2, rep: 1},
		{name: "rep movsd", code: []byte{0xF3, 0xA5}, op: OpMovs, width: 4, rep: 1},
		{name: "stosb", code: []byte{0xAA}, op: OpStos, width: 1},
		{name: "stosw", code: []byte{0x66, 0xAB}, op: OpStos, width: 2},
		{name: "lodsw", code: []byte{0x66, 0xAD}, op: OpLods, width: 2},
		{name: "lodsd", code: []byte{0xAD}, op: OpLods, width: 4},
		{name: "repne scasw", code: []byte{0xF2, 0x66, 0xAF}, op: OpScas, width: 2, rep: 2},
		{name: "repne scasb", code: []byte{0xF2, 0xAE}, op: OpScas, width: 1, rep: 2},
		{name: "repe cmpsw", code: []byte{0xF3, 0x66, 0xA7}, op: OpCmps, width: 2, rep: 1},
		{name: "repe cmpsd", code: []byte{0xF3, 0xA7}, op: OpCmps, width: 4, rep: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			memory, _ := mappedCode(t, test.code)
			instruction, err := Decode(memory, PageSize)
			if err != nil {
				t.Fatal(err)
			}
			if instruction.Op != test.op || instruction.Len != uint32(len(test.code)) || instruction.Imm != test.width || instruction.Group != test.rep {
				t.Fatalf("instruction = %#v, want op %v len %d width %d repeat %d", instruction, test.op, len(test.code), test.width, test.rep)
			}
		})
	}
}

func TestDecodeX86StackAndControlFlow(t *testing.T) {
	tests := []struct {
		name string
		code []byte
		op   Op
		reg  Reg32
		imm  int32
		rel  int32
		src  Operand
		dst  Operand
	}{
		{name: "push register", code: []byte{0x50}, op: OpPushReg, reg: EAX},
		{name: "pop register", code: []byte{0x58}, op: OpPopReg, reg: EAX},
		{name: "push immediate", code: []byte{0x6A, 0xF9}, op: OpPushImm, imm: -7},
		{name: "push memory", code: []byte{0xFF, 0x33}, op: OpPushMem, src: Operand{IsMem: true, Width: 4, Memory: MemoryOperand{Base: EBX, HasBase: true, Scale: 1}}},
		{name: "pop memory", code: []byte{0x8F, 0x43, 0x04}, op: OpPopMem, dst: Operand{IsMem: true, Width: 4, Memory: MemoryOperand{Base: EBX, HasBase: true, Scale: 1, Disp: 4}}},
		{name: "call relative", code: []byte{0xE8, 0x05, 0x00, 0x00, 0x00}, op: OpCallRel, rel: 5},
		{name: "call register", code: []byte{0xFF, 0xD0}, op: OpCallOperand, src: regOperand(EAX)},
		{name: "ret", code: []byte{0xC3}, op: OpRet},
		{name: "ret immediate", code: []byte{0xC2, 0x08, 0x00}, op: OpRetImm, imm: 8},
		{name: "push flags", code: []byte{0x9C}, op: OpPushFlags},
		{name: "pop flags", code: []byte{0x9D}, op: OpPopFlags},
		{name: "push all", code: []byte{0x60}, op: OpPushAll},
		{name: "pop all", code: []byte{0x61}, op: OpPopAll},
		{name: "leave", code: []byte{0xC9}, op: OpLeave},
		{name: "enter frame", code: []byte{0xC8, 0x20, 0x01, 0x03}, op: OpEnter, imm: 0x120, rel: 0},
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
			if test.reg != 0 && instruction.Reg != test.reg {
				t.Fatalf("register = %v, want %v", instruction.Reg, test.reg)
			}
			if instruction.Imm != test.imm || instruction.Rel != test.rel {
				t.Fatalf("immediate/relative = %d/%d, want %d/%d", instruction.Imm, instruction.Rel, test.imm, test.rel)
			}
			if test.op == OpEnter && instruction.Group != 3 {
				t.Fatalf("ENTER nesting level = %d, want 3", instruction.Group)
			}

			if test.src != (Operand{}) && instruction.Src != test.src {
				t.Fatalf("source = %#v, want %#v", instruction.Src, test.src)
			}
			if test.dst != (Operand{}) && instruction.Dst != test.dst {
				t.Fatalf("destination = %#v, want %#v", instruction.Dst, test.dst)
			}
		})
	}
}

func TestDecodeX86JccConditions(t *testing.T) {
	tests := []struct {
		name      string
		opcode    byte
		condition uint8
	}{
		{name: "jo", opcode: 0x70, condition: 14},
		{name: "jno", opcode: 0x71, condition: 15},
		{name: "jb", opcode: 0x72, condition: 0},
		{name: "jae", opcode: 0x73, condition: 1},
		{name: "je", opcode: 0x74, condition: 2},
		{name: "jne", opcode: 0x75, condition: 3},
		{name: "jbe", opcode: 0x76, condition: 4},
		{name: "ja", opcode: 0x77, condition: 5},
		{name: "js", opcode: 0x78, condition: 6},
		{name: "jns", opcode: 0x79, condition: 7},
		{name: "jp", opcode: 0x7A, condition: 8},
		{name: "jnp", opcode: 0x7B, condition: 9},
		{name: "jl", opcode: 0x7C, condition: 10},
		{name: "jge", opcode: 0x7D, condition: 11},
		{name: "jle", opcode: 0x7E, condition: 12},
		{name: "jg", opcode: 0x7F, condition: 13},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			memory, _ := mappedCode(t, []byte{test.opcode, 0x05})
			instruction, err := Decode(memory, PageSize)
			if err != nil {
				t.Fatal(err)
			}
			if instruction.Op != OpJcc || instruction.Len != 2 || instruction.Rel != 5 || instruction.Group != test.condition {
				t.Fatalf("instruction = %#v, want OpJcc len 2 rel 5 group %d", instruction, test.condition)
			}
		})
	}
}

func TestDecodeX86JccNear(t *testing.T) {
	memory, _ := mappedCode(t, []byte{0x0F, 0x8C, 0x05, 0x00, 0x00, 0x00}) // jl near +5
	instruction, err := Decode(memory, PageSize)
	if err != nil {
		t.Fatal(err)
	}
	if instruction.Op != OpJcc || instruction.Len != 6 || instruction.Rel != 5 || instruction.Group != 10 {
		t.Fatalf("instruction = %#v, want near JL group 10 rel 5", instruction)
	}
}

func TestDecodeX86LoopInstructions(t *testing.T) {
	tests := []struct {
		name      string
		code      []byte
		length    uint32
		condition uint8
	}{
		{name: "loop", code: []byte{0xE2, 0xFE}, length: 2, condition: 0},
		{name: "loope", code: []byte{0xE1, 0xFE}, length: 2, condition: 1},
		{name: "loopne", code: []byte{0xE0, 0xFE}, length: 2, condition: 2},
		{name: "jecxz", code: []byte{0xE3, 0xFE}, length: 2, condition: 3},
		{name: "jcxz", code: []byte{0x67, 0xE3, 0xFE}, length: 3, condition: 4},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			memory, _ := mappedCode(t, test.code)
			instruction, err := Decode(memory, PageSize)
			if err != nil {
				t.Fatal(err)
			}
			if instruction.Op != OpLoop || instruction.Len != test.length || instruction.Rel != -2 || instruction.Group != test.condition {
				t.Fatalf("instruction = %#v, want OpLoop len %d rel -2 group %d", instruction, test.length, test.condition)
			}
		})
	}
}

func TestDecodeX86RegisterExtensionsAndTransforms(t *testing.T) {
	tests := []struct {
		name     string
		code     []byte
		op       Op
		srcWidth uint8
		srcIsMem bool
		dst      Reg32
		length   uint32
	}{
		{name: "movzx byte register", code: []byte{0x0F, 0xB6, 0xC0}, op: OpMovzxRegOperand, srcWidth: 1, dst: EAX, length: 3},
		{name: "movzx word register", code: []byte{0x0F, 0xB7, 0xC1}, op: OpMovzxRegOperand, srcWidth: 2, dst: EAX, length: 3},
		{name: "movsx byte memory", code: []byte{0x0F, 0xBE, 0x43, 0x02}, op: OpMovsxRegOperand, srcWidth: 1, srcIsMem: true, dst: EAX, length: 4},
		{name: "movsx word memory", code: []byte{0x0F, 0xBF, 0x4B, 0x04}, op: OpMovsxRegOperand, srcWidth: 2, srcIsMem: true, dst: ECX, length: 4},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			memory, _ := mappedCode(t, test.code)
			instruction, err := Decode(memory, PageSize)
			if err != nil {
				t.Fatal(err)
			}
			if instruction.Op != test.op || instruction.Len != test.length || instruction.Dst.Reg != test.dst || instruction.Src.Width != test.srcWidth || instruction.Src.IsMem != test.srcIsMem {
				t.Fatalf("instruction = %#v, want op %v len %d dst %v src width %d mem %v", instruction, test.op, test.length, test.dst, test.srcWidth, test.srcIsMem)
			}
		})
	}

	for _, test := range []struct {
		name string
		code []byte
		op   Op
	}{
		{name: "lea", code: []byte{0x8D, 0x54, 0x8B, 0x10}, op: OpLeaRegMem},
		{name: "cwde", code: []byte{0x98}, op: OpCWDE},
		{name: "cdq", code: []byte{0x99}, op: OpCDQ},
	} {
		t.Run(test.name, func(t *testing.T) {
			memory, _ := mappedCode(t, test.code)
			instruction, err := Decode(memory, PageSize)
			if err != nil {
				t.Fatal(err)
			}
			if instruction.Op != test.op || instruction.Len != uint32(len(test.code)) {
				t.Fatalf("instruction = %#v, want op %v len %d", instruction, test.op, len(test.code))
			}
		})
	}
}

func TestDecodeX86AtomicInstructions(t *testing.T) {
	tests := []struct {
		name   string
		code   []byte
		op     Op
		dst    Operand
		src    Operand
		length uint32
	}{
		{
			name:   "cmpxchg register",
			code:   []byte{0x0F, 0xB1, 0xC8}, // cmpxchg eax, ecx
			op:     OpCmpxchg,
			dst:    regOperand(EAX),
			src:    regOperand(ECX),
			length: 3,
		},
		{
			name: "cmpxchg memory",
			code: []byte{0x0F, 0xB1, 0x4B, 0x04}, // cmpxchg [ebx+4], ecx
			op:   OpCmpxchg,
			dst:  Operand{IsMem: true, Width: 4, Memory: MemoryOperand{Base: EBX, HasBase: true, Scale: 1, Disp: 4}},
			src:  regOperand(ECX), length: 4,
		},
		{
			name:   "cmpxchg byte",
			code:   []byte{0x0F, 0xB0, 0xC8}, // cmpxchg al, cl
			op:     OpCmpxchg,
			dst:    Operand{Reg: EAX, Width: 1},
			src:    Operand{Reg: ECX, Width: 1},
			length: 3,
		},
		{
			name:   "xadd register",
			code:   []byte{0x0F, 0xC1, 0xC8}, // xadd eax, ecx
			op:     OpXadd,
			dst:    regOperand(EAX),
			src:    regOperand(ECX),
			length: 3,
		},
		{
			name: "xadd memory",
			code: []byte{0x0F, 0xC1, 0x4B, 0x04}, // xadd [ebx+4], ecx
			op:   OpXadd,
			dst:  Operand{IsMem: true, Width: 4, Memory: MemoryOperand{Base: EBX, HasBase: true, Scale: 1, Disp: 4}},
			src:  regOperand(ECX), length: 4,
		},
		{
			name:   "xadd byte",
			code:   []byte{0x0F, 0xC0, 0xC8}, // xadd al, cl
			op:     OpXadd,
			dst:    Operand{Reg: EAX, Width: 1},
			src:    Operand{Reg: ECX, Width: 1},
			length: 3,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			memory, _ := mappedCode(t, test.code)
			instruction, err := Decode(memory, PageSize)
			if err != nil {
				t.Fatal(err)
			}
			if instruction.Op != test.op || instruction.Len != test.length {
				t.Fatalf("instruction = %#v, want op %v len %d", instruction, test.op, test.length)
			}
			if instruction.Dst != test.dst || instruction.Src != test.src {
				t.Fatalf("operands = dst %#v src %#v, want dst %#v src %#v", instruction.Dst, instruction.Src, test.dst, test.src)
			}
		})
	}
}

func TestDecodeX86CarryAndFlagTransfer(t *testing.T) {
	tests := []struct {
		name  string
		code  []byte
		op    Op
		width uint8
	}{
		{name: "adc register", code: []byte{0x11, 0xD8}, op: OpAdcOperands, width: 4},         // adc eax, ebx
		{name: "sbb memory", code: []byte{0x19, 0x03}, op: OpSbbOperands, width: 4},           // sbb [ebx], eax
		{name: "adc byte immediate", code: []byte{0x14, 0x01}, op: OpAdcImm, width: 1},        // adc al, 1
		{name: "sbb dword immediate", code: []byte{0x83, 0xD8, 0x01}, op: OpSbbImm, width: 4}, // sbb eax, 1
		{name: "lahf", code: []byte{0x9F}, op: OpLahf},
		{name: "sahf", code: []byte{0x9E}, op: OpSahf},
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
			if test.width != 0 && instruction.Dst.Width != test.width {
				t.Fatalf("destination width = %d, want %d", instruction.Dst.Width, test.width)
			}
		})
	}
}

func TestDecodeX86CarryMemorySource(t *testing.T) {
	memory, _ := mappedCode(t, []byte{0x13, 0x03}) // adc eax, dword ptr [ebx]
	instruction, err := Decode(memory, PageSize)
	if err != nil {
		t.Fatal(err)
	}
	if instruction.Op != OpAdcOperands || !instruction.Dst.IsMem && instruction.Dst.Reg != EAX {
		t.Fatalf("instruction = %#v, want ADC register destination", instruction)
	}
	if !instruction.Src.IsMem || instruction.Src.Memory.Base != EBX || instruction.Src.Width != 4 {
		t.Fatalf("source = %#v, want dword ptr [ebx]", instruction.Src)
	}
}

func TestDecodeX86BitOperations(t *testing.T) {
	tests := []struct {
		name  string
		code  []byte
		op    Op
		group uint8
		dst   Operand
		src   Operand
		imm   int32
	}{
		{name: "bswap", code: []byte{0x0F, 0xC8}, op: OpBswap, dst: regOperand(EAX)},
		{name: "bound", code: []byte{0x62, 0x44, 0x8B, 0x04}, op: OpBound, dst: regOperand(EAX), src: Operand{IsMem: true, Width: 4, Memory: MemoryOperand{Base: EBX, Index: ECX, Scale: 4, Disp: 4, HasBase: true, HasIndex: true}}},
		{name: "bt register", code: []byte{0x0F, 0xA3, 0xCB}, op: OpBitTest, dst: regOperand(EBX), src: regOperand(ECX)},
		{name: "bts memory", code: []byte{0x0F, 0xAB, 0x4B, 0x04}, op: OpBitTest, group: 1, dst: Operand{IsMem: true, Width: 4, Memory: MemoryOperand{Base: EBX, HasBase: true, Scale: 1, Disp: 4}}, src: regOperand(ECX)},
		{name: "btr memory", code: []byte{0x0F, 0xB3, 0x53, 0x08}, op: OpBitTest, group: 2, dst: Operand{IsMem: true, Width: 4, Memory: MemoryOperand{Base: EBX, HasBase: true, Scale: 1, Disp: 8}}, src: regOperand(EDX)},
		{name: "btc immediate", code: []byte{0x0F, 0xBA, 0x7B, 0x0C, 0x1F}, op: OpBitTest, group: 3, dst: Operand{IsMem: true, Width: 4, Memory: MemoryOperand{Base: EBX, HasBase: true, Scale: 1, Disp: 12}}, imm: 31},
		{name: "bsf", code: []byte{0x0F, 0xBC, 0xC3}, op: OpBitScan, dst: regOperand(EAX), src: regOperand(EBX)},
		{name: "bsr memory", code: []byte{0x0F, 0xBD, 0x4B, 0x04}, op: OpBitScan, group: 1, dst: regOperand(ECX), src: Operand{IsMem: true, Width: 4, Memory: MemoryOperand{Base: EBX, HasBase: true, Scale: 1, Disp: 4}}},
		{name: "popcnt memory", code: []byte{0xF3, 0x0F, 0xB8, 0x43, 0x04}, op: OpPopcnt, dst: regOperand(EAX), src: Operand{IsMem: true, Width: 4, Memory: MemoryOperand{Base: EBX, HasBase: true, Scale: 1, Disp: 4}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			memory, _ := mappedCode(t, test.code)
			instruction, err := Decode(memory, PageSize)
			if err != nil {
				t.Fatal(err)
			}
			if instruction.Op != test.op || instruction.Len != uint32(len(test.code)) || instruction.Group != test.group || instruction.Imm != test.imm {
				t.Fatalf("instruction = %#v, want op %v len %d group %d imm %d", instruction, test.op, len(test.code), test.group, test.imm)
			}
			if instruction.Dst != test.dst || instruction.Src != test.src {
				t.Fatalf("operands = dst %#v src %#v, want dst %#v src %#v", instruction.Dst, instruction.Src, test.dst, test.src)
			}
		})
	}
}

func TestDecodeX86RotateInstructions(t *testing.T) {
	tests := []struct {
		name  string
		code  []byte
		group uint8
		dst   Operand
		src   Operand
		imm   int32
	}{
		{name: "rol one", code: []byte{0xD1, 0xC0}, group: 0, dst: regOperand(EAX), imm: 1},
		{name: "ror immediate", code: []byte{0xC1, 0xC8, 0x07}, group: 1, dst: regOperand(EAX), imm: 7},
		{name: "rcl cl", code: []byte{0xD3, 0xD0}, group: 2, dst: regOperand(EAX), src: Operand{Reg: ECX, Width: 1}},
		{name: "rcr memory", code: []byte{0xC1, 0x18, 0x01}, group: 3, dst: Operand{IsMem: true, Width: 4, Memory: MemoryOperand{Base: EAX, HasBase: true, Scale: 1}}, imm: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			memory, _ := mappedCode(t, test.code)
			instruction, err := Decode(memory, PageSize)
			if err != nil {
				t.Fatal(err)
			}
			if instruction.Op != OpRotate || instruction.Len != uint32(len(test.code)) || instruction.Group != test.group || instruction.Imm != test.imm {
				t.Fatalf("instruction = %#v, want rotate len %d group %d imm %d", instruction, len(test.code), test.group, test.imm)
			}
			if instruction.Dst != test.dst || instruction.Src != test.src {
				t.Fatalf("operands = dst %#v src %#v, want dst %#v src %#v", instruction.Dst, instruction.Src, test.dst, test.src)
			}
		})
	}
}

func TestDecodeX86MovByteImmediate(t *testing.T) {
	tests := []struct {
		name       string
		code       []byte
		reg        Reg32
		byteOffset uint8
		imm        int32
	}{
		{name: "al", code: []byte{0xB0, 0x12}, reg: EAX, imm: 0x12},
		{name: "cl", code: []byte{0xB1, 0x34}, reg: ECX, imm: 0x34},
		{name: "dl", code: []byte{0xB2, 0x56}, reg: EDX, imm: 0x56},
		{name: "bl", code: []byte{0xB3, 0x78}, reg: EBX, imm: 0x78},
		{name: "ah", code: []byte{0xB4, 0x9A}, reg: EAX, byteOffset: 1, imm: 0x9A},
		{name: "ch", code: []byte{0xB5, 0xBC}, reg: ECX, byteOffset: 1, imm: 0xBC},
		{name: "dh", code: []byte{0xB6, 0xDE}, reg: EDX, byteOffset: 1, imm: 0xDE},
		{name: "bh", code: []byte{0xB7, 0xF0}, reg: EBX, byteOffset: 1, imm: 0xF0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			memory, _ := mappedCode(t, test.code)
			instruction, err := Decode(memory, PageSize)
			if err != nil {
				t.Fatal(err)
			}
			if instruction.Op != OpMovByteImm || instruction.Len != 2 || instruction.Dst.Reg != test.reg || instruction.Dst.Width != 1 || instruction.Dst.ByteOffset != test.byteOffset || instruction.Imm != test.imm {
				t.Fatalf("instruction = %#v, want byte MOV reg=%v offset=%d imm=%#x", instruction, test.reg, test.byteOffset, test.imm)
			}
		})
	}
}

func TestDecodeX86Movbe(t *testing.T) {
	tests := []struct {
		name string
		code []byte
		dst  Operand
		src  Operand
	}{
		{
			name: "load register memory",
			code: []byte{0x0F, 0x38, 0xF0, 0x03},
			dst:  regOperand(EAX),
			src:  Operand{IsMem: true, Width: 4, Memory: MemoryOperand{Base: EBX, HasBase: true, Scale: 1}},
		},
		{
			name: "store memory register with SIB",
			code: []byte{0x0F, 0x38, 0xF1, 0x6C, 0xB3, 0x04},
			dst:  Operand{IsMem: true, Width: 4, Memory: MemoryOperand{Base: EBX, Index: ESI, Scale: 4, Disp: 4, HasBase: true, HasIndex: true}},
			src:  regOperand(EBP),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			memory, _ := mappedCode(t, test.code)
			instruction, err := Decode(memory, PageSize)
			if err != nil {
				t.Fatal(err)
			}
			if instruction.Op != OpMovbe || instruction.Len != uint32(len(test.code)) || instruction.Dst != test.dst || instruction.Src != test.src {
				t.Fatalf("instruction = %#v, want MOVBE len %d dst %#v src %#v", instruction, len(test.code), test.dst, test.src)
			}
		})
	}
}

func TestDecodeX86FPUStack(t *testing.T) {
	tests := []struct {
		name  string
		code  []byte
		op    Op
		group uint8
		reg   uint8
		reg2  uint8
	}{
		{name: "fld1", code: []byte{0xD9, 0xE8}, op: OpFPUConst, group: 1},
		{name: "fldz", code: []byte{0xD9, 0xEE}, op: OpFPUConst, group: 0},
		{name: "fchs", code: []byte{0xD9, 0xE0}, op: OpFPUUnary, group: 0},
		{name: "fabs", code: []byte{0xD9, 0xE1}, op: OpFPUUnary, group: 1},
		{name: "fxch st1", code: []byte{0xD9, 0xC9}, op: OpFPUStack, group: fpuStackXchg, reg: 1},
		{name: "fld st1", code: []byte{0xD9, 0xC1}, op: OpFPUStack, group: fpuStackLoad, reg: 1},
		{name: "fstp st1", code: []byte{0xDD, 0xD9}, op: OpFPUStack, group: fpuStackStorePop, reg: 1},
		{name: "fincstp", code: []byte{0xD9, 0xF7}, op: OpFPUStack, group: fpuStackIncTop},
		{name: "fdecstp", code: []byte{0xD9, 0xF6}, op: OpFPUStack, group: fpuStackDecTop},
		{name: "fadd st0, st1", code: []byte{0xD8, 0xC1}, op: OpFPUBinary, group: fpuBinaryAdd, reg: 0, reg2: 1},
		{name: "faddp st1, st0", code: []byte{0xDE, 0xC1}, op: OpFPUBinary, group: fpuBinaryAdd | 0x80, reg: 1, reg2: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			memory, _ := mappedCode(t, test.code)
			instruction, err := Decode(memory, PageSize)
			if err != nil {
				t.Fatal(err)
			}
			if instruction.Op != test.op || instruction.Len != uint32(len(test.code)) || instruction.Group != test.group || instruction.FPUReg != test.reg || instruction.FPUReg2 != test.reg2 {
				t.Fatalf("instruction = %#v, want op=%v len=%d group=%d fpu=%d/%d", instruction, test.op, len(test.code), test.group, test.reg, test.reg2)
			}
		})
	}
}

func TestDecodeX86FPUMemory(t *testing.T) {
	tests := []struct {
		name  string
		code  []byte
		group uint8
		width uint8
		base  Reg32
		index Reg32
		disp  int32
	}{
		{name: "fld m32 absolute", code: []byte{0xD9, 0x05, 0x00, 0x20, 0x00, 0x00}, group: fpuStackLoad, width: 4, base: RegNone, index: RegNone, disp: 0x2000},
		{name: "fst m32 absolute", code: []byte{0xD9, 0x15, 0x00, 0x20, 0x00, 0x00}, group: fpuStackStore, width: 4, base: RegNone, index: RegNone, disp: 0x2000},
		{name: "fstp m64 absolute", code: []byte{0xDD, 0x1D, 0x00, 0x20, 0x00, 0x00}, group: fpuStackStorePop, width: 8, base: RegNone, index: RegNone, disp: 0x2000},
		{name: "fld m32 sib", code: []byte{0xD9, 0x44, 0xB3, 0x04}, group: fpuStackLoad, width: 4, base: EBX, index: ESI, disp: 4},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			memory, _ := mappedCode(t, test.code)
			instruction, err := Decode(memory, PageSize)
			if err != nil {
				t.Fatal(err)
			}
			if instruction.Op != OpFPUStack || instruction.Group != test.group || instruction.FPUMemWidth != test.width || instruction.Len != uint32(len(test.code)) {
				t.Fatalf("instruction = %#v, want FPU memory group=%d width=%d len=%d", instruction, test.group, test.width, len(test.code))
			}
			if !instruction.Dst.IsMem || instruction.Dst.Memory.Disp != test.disp {
				t.Fatalf("destination memory = %#v, want disp %#x", instruction.Dst, test.disp)
			}
			if test.base != RegNone && (!instruction.Dst.Memory.HasBase || instruction.Dst.Memory.Base != test.base) {
				t.Fatalf("memory base = %#v, want %s", instruction.Dst.Memory, test.base)
			}
			if test.index != RegNone && (!instruction.Dst.Memory.HasIndex || instruction.Dst.Memory.Index != test.index || instruction.Dst.Memory.Scale != 4) {
				t.Fatalf("memory index = %#v, want %s*4", instruction.Dst.Memory, test.index)
			}
		})
	}
}

func TestDecodeX87IntegerMemory(t *testing.T) {
	tests := []struct {
		name  string
		code  []byte
		group uint8
		width uint8
		base  Reg32
		index Reg32
		disp  int32
	}{
		{name: "fild m16", code: []byte{0xDF, 0x05, 0x00, 0x20, 0x00, 0x00}, group: fpuStackLoadInt, width: 2, base: RegNone, index: RegNone, disp: 0x2000},
		{name: "fild m32", code: []byte{0xDB, 0x05, 0x00, 0x20, 0x00, 0x00}, group: fpuStackLoadInt, width: 4, base: RegNone, index: RegNone, disp: 0x2000},
		{name: "fild m64", code: []byte{0xDF, 0x2D, 0x00, 0x20, 0x00, 0x00}, group: fpuStackLoadInt, width: 8, base: RegNone, index: RegNone, disp: 0x2000},
		{name: "fist m16", code: []byte{0xDF, 0x15, 0x00, 0x20, 0x00, 0x00}, group: fpuStackStoreInt, width: 2, base: RegNone, index: RegNone, disp: 0x2000},
		{name: "fistp m32", code: []byte{0xDB, 0x1D, 0x00, 0x20, 0x00, 0x00}, group: fpuStackStoreIntPop, width: 4, base: RegNone, index: RegNone, disp: 0x2000},
		{name: "fistp m64", code: []byte{0xDF, 0x3D, 0x00, 0x20, 0x00, 0x00}, group: fpuStackStoreIntPop, width: 8, base: RegNone, index: RegNone, disp: 0x2000},
		{name: "fisttp m16", code: []byte{0xDF, 0x0D, 0x00, 0x20, 0x00, 0x00}, group: fpuStackStoreIntTruncPop, width: 2, base: RegNone, index: RegNone, disp: 0x2000},
		{name: "fild m32 sib", code: []byte{0xDB, 0x44, 0xB3, 0x04}, group: fpuStackLoadInt, width: 4, base: EBX, index: ESI, disp: 4},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			memory, _ := mappedCode(t, test.code)
			instruction, err := Decode(memory, PageSize)
			if err != nil {
				t.Fatal(err)
			}
			if instruction.Op != OpFPUStack || instruction.Group != test.group || instruction.FPUMemWidth != test.width || instruction.Len != uint32(len(test.code)) {
				t.Fatalf("instruction = %#v, want FPU integer group=%d width=%d len=%d", instruction, test.group, test.width, len(test.code))
			}
			if !instruction.Dst.IsMem || instruction.Dst.Memory.Disp != test.disp {
				t.Fatalf("destination memory = %#v, want disp %#x", instruction.Dst, test.disp)
			}
			if test.base != RegNone && (!instruction.Dst.Memory.HasBase || instruction.Dst.Memory.Base != test.base) {
				t.Fatalf("memory base = %#v, want %s", instruction.Dst.Memory, test.base)
			}
			if test.index != RegNone && (!instruction.Dst.Memory.HasIndex || instruction.Dst.Memory.Index != test.index || instruction.Dst.Memory.Scale != 4) {
				t.Fatalf("memory index = %#v, want %s*4", instruction.Dst.Memory, test.index)
			}
		})
	}
}

func TestDecodeX87CompareAndStatus(t *testing.T) {
	tests := []struct {
		name   string
		code   []byte
		op     Op
		group  uint8
		reg    uint8
		reg2   uint8
		width  uint8
		memory bool
	}{
		{name: "fcom st1", code: []byte{0xd8, 0xd1}, op: OpFPUCompare, group: fpuCompareStatus, reg: 1},
		{name: "fcomp st1", code: []byte{0xd8, 0xd9}, op: OpFPUCompare, group: fpuCompareStatus | fpuComparePopSingle, reg: 1},
		{name: "fcompp", code: []byte{0xde, 0xd9}, op: OpFPUCompare, group: fpuCompareStatus | fpuComparePopDouble, reg: 1},
		{name: "fucom st2", code: []byte{0xdd, 0xe2}, op: OpFPUCompare, group: fpuCompareStatus, reg: 2},
		{name: "fcomi st0 st1", code: []byte{0xdb, 0xf1}, op: OpFPUCompare, group: fpuCompareEFlags, reg: 0, reg2: 1},
		{name: "fcomip st0 st1", code: []byte{0xdf, 0xf1}, op: OpFPUCompare, group: fpuCompareEFlags | fpuComparePopSingle, reg: 0, reg2: 1},
		{name: "fcom m32", code: []byte{0xd8, 0x11}, op: OpFPUCompare, group: fpuCompareStatus, width: 4, memory: true},
		{name: "fcom m64", code: []byte{0xdc, 0x11}, op: OpFPUCompare, group: fpuCompareStatus, width: 8, memory: true},
		{name: "fnstsw ax", code: []byte{0xdf, 0xe0}, op: OpFNSTSW},
		{name: "fnstsw m16", code: []byte{0xdd, 0x38}, op: OpFNSTSW, width: 2, memory: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			memory, _ := mappedCode(t, test.code)
			instruction, err := Decode(memory, PageSize)
			if err != nil {
				t.Fatal(err)
			}
			if instruction.Op != test.op || instruction.Len != uint32(len(test.code)) || instruction.Group != test.group || instruction.FPUReg != test.reg || instruction.FPUReg2 != test.reg2 || instruction.FPUMemWidth != test.width {
				t.Fatalf("instruction = %#v, want op=%v len=%d group=%d fpu=%d/%d width=%d", instruction, test.op, len(test.code), test.group, test.reg, test.reg2, test.width)
			}
			if instruction.Dst.IsMem != test.memory {
				t.Fatalf("memory destination = %v, want %v", instruction.Dst.IsMem, test.memory)
			}
		})
	}
}

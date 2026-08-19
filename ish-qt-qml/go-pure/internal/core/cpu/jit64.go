package cpu

import (
	"errors"
	"fmt"
	"math/bits"
	"sync"

	"golang.org/x/arch/x86/x86asm"
)

var (
	ErrUnsupported64  = errors.New("cpu64: unsupported instruction")
	ErrInvalid64Block = errors.New("cpu64: invalid translated block")
)

type Flow64 uint8

const (
	Flow64Continue Flow64 = iota
	Flow64Stop
	Flow64Branch
	Flow64Interrupt
)

type operand64Kind uint8

const (
	operand64Invalid operand64Kind = iota
	operand64Reg
	operand64Mem
	operand64XMM
	operand64Imm
	operand64Rel
)

type memoryOperand64 struct {
	Base        Reg64
	Index       Reg64
	Scale       uint8
	Disp        int64
	HasBase     bool
	HasIndex    bool
	RIPRelative bool
	Segment     x86asm.Reg
}

type operand64 struct {
	Kind       operand64Kind
	Reg        Reg64
	XMM        uint8
	ByteOffset uint8
	Width      uint8
	Mem        memoryOperand64
	Imm        uint64
	Rel        int64
}

type microOp64 struct {
	Address uint64
	Size    uint8
	Run     func(*MachineState64, uint64) (Flow64, error)
}

// CompiledBlock64 is the Pure-Go equivalent of an iSH asbestos fiber_block.
// Its operations are already decoded and validated; no opcode parser exists
// outside x86asm.
type CompiledBlock64 struct {
	Start       uint64
	End         uint64
	Ops         []microOp64
	Pages       []Page64
	Generations map[Page64]uint64
	Invalid     bool
}

func (b *CompiledBlock64) Valid(memory *Memory64) bool {
	if b == nil || b.Invalid || memory == nil {
		return false
	}
	for page, generation := range b.Generations {
		current, mapped := memory.PageGeneration(page)
		if !mapped || current != generation {
			return false
		}
	}
	return true
}

// Run executes one compiled block. Branching instructions terminate the block
// and leave the next guest RIP in state.RIP, matching fiber_ret_chain's contract.
func (b *CompiledBlock64) Run(state *MachineState64) (Flow64, error) {
	if b == nil || state == nil || b.Invalid {
		return Flow64Stop, ErrInvalid64Block
	}
	for _, op := range b.Ops {
		flow, err := op.Run(state, op.Address+uint64(op.Size))
		if err != nil || flow != Flow64Continue {
			return flow, err
		}
	}
	return Flow64Stop, nil
}

type BlockCache64 struct {
	memory *Memory64
	mu     sync.RWMutex
	blocks map[uint64]*CompiledBlock64
	pages  map[Page64]map[uint64]struct{}
}

func NewBlockCache64(memory *Memory64) *BlockCache64 {
	return &BlockCache64{memory: memory, blocks: make(map[uint64]*CompiledBlock64), pages: make(map[Page64]map[uint64]struct{})}
}

func (c *BlockCache64) Get(start uint64) (*CompiledBlock64, bool) {
	if c == nil || c.memory == nil {
		return nil, false
	}
	c.mu.RLock()
	block := c.blocks[start]
	c.mu.RUnlock()
	if !block.Valid(c.memory) {
		if block != nil {
			c.InvalidateBlock(start)
		}
		return nil, false
	}
	return block, true
}

func (c *BlockCache64) Put(block *CompiledBlock64) error {
	if c == nil || c.memory == nil || block == nil || block.Start == 0 && block.End == 0 {
		return ErrInvalid64Block
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if old := c.blocks[block.Start]; old != nil {
		c.removeLocked(old)
	}
	c.blocks[block.Start] = block
	for _, page := range block.Pages {
		set := c.pages[page]
		if set == nil {
			set = make(map[uint64]struct{})
			c.pages[page] = set
		}
		set[block.Start] = struct{}{}
	}
	return nil
}

func (c *BlockCache64) removeLocked(block *CompiledBlock64) {
	if block == nil {
		return
	}
	delete(c.blocks, block.Start)
	for _, page := range block.Pages {
		set := c.pages[page]
		delete(set, block.Start)
		if len(set) == 0 {
			delete(c.pages, page)
		}
	}
	block.Invalid = true
}

func (c *BlockCache64) InvalidateBlock(start uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.removeLocked(c.blocks[start])
}

func (c *BlockCache64) InvalidateRange(start, end Address64) {
	if c == nil || end <= start {
		return
	}
	first := Page64(uint64(start) >> Page64Bits)
	last := Page64((uint64(end) - 1) >> Page64Bits)
	c.mu.Lock()
	defer c.mu.Unlock()
	seen := make(map[uint64]struct{})
	for page := first; page <= last; page++ {
		for blockStart := range c.pages[page] {
			seen[blockStart] = struct{}{}
		}
	}
	for blockStart := range seen {
		c.removeLocked(c.blocks[blockStart])
	}
}

func (c *BlockCache64) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.blocks)
}

func CompileBlock64(memory *Memory64, start Address64, maxBytes uint64) (*CompiledBlock64, error) {
	if memory == nil || !canonicalAddress64(start) {
		return nil, ErrInvalid64Block
	}
	if maxBytes == 0 || maxBytes > Page64Size {
		maxBytes = Page64Size
	}
	block := &CompiledBlock64{Start: uint64(start), Generations: make(map[Page64]uint64)}
	var offset uint64
	for offset < maxBytes {
		address := uint64(start) + offset
		readSize := uint64(15)
		if remaining := maxBytes - offset; remaining < readSize {
			readSize = remaining
		}
		code := make([]byte, readSize)
		if err := memory.Read(Address64(address), code); err != nil {
			return nil, fmt.Errorf("%w at %#x: %v", ErrInvalid64Block, address, err)
		}
		inst, err := x86asm.Decode(code, 64)
		if err != nil || inst.Len == 0 {
			if err == nil {
				err = errors.New("zero-length instruction")
			}
			return nil, fmt.Errorf("%w at %#x: %v", ErrUnsupported64, address, err)
		}
		if offset+uint64(inst.Len) > maxBytes {
			return nil, fmt.Errorf("%w: instruction crosses block limit at %#x", ErrInvalid64Block, address)
		}
		op, terminates, err := compileInstruction64(inst, address)
		if err != nil {
			return nil, fmt.Errorf("%w at %#x (%s): %v", ErrUnsupported64, address, inst, err)
		}
		block.Ops = append(block.Ops, op)
		firstPage := Page64(address >> Page64Bits)
		lastPage := Page64((address + uint64(inst.Len) - 1) >> Page64Bits)
		for page := firstPage; page <= lastPage; page++ {
			if _, exists := block.Generations[page]; !exists {
				generation, mapped := memory.PageGeneration(page)
				if !mapped {
					return nil, fmt.Errorf("%w: unmapped code page %#x", ErrInvalid64Block, page)
				}
				block.Generations[page] = generation
				block.Pages = append(block.Pages, page)
			}
		}
		offset += uint64(inst.Len)
		if terminates {
			break
		}
	}
	if len(block.Ops) == 0 {
		return nil, ErrInvalid64Block
	}
	block.End = block.Start + offset - 1
	return block, nil
}

func reg64FromX86(reg x86asm.Reg) (Reg64, uint8, uint8, bool) {
	switch reg {
	case x86asm.AL:
		return RAX, 0, 1, true
	case x86asm.CL:
		return RCX, 0, 1, true
	case x86asm.DL:
		return RDX, 0, 1, true
	case x86asm.BL:
		return RBX, 0, 1, true
	case x86asm.AH:
		return RAX, 1, 1, true
	case x86asm.CH:
		return RCX, 1, 1, true
	case x86asm.DH:
		return RDX, 1, 1, true
	case x86asm.BH:
		return RBX, 1, 1, true
	case x86asm.AX:
		return RAX, 0, 2, true
	case x86asm.CX:
		return RCX, 0, 2, true
	case x86asm.DX:
		return RDX, 0, 2, true
	case x86asm.BX:
		return RBX, 0, 2, true
	case x86asm.SP:
		return RSP, 0, 2, true
	case x86asm.BP:
		return RBP, 0, 2, true
	case x86asm.SI:
		return RSI, 0, 2, true
	case x86asm.DI:
		return RDI, 0, 2, true
	case x86asm.EAX:
		return RAX, 0, 4, true
	case x86asm.ECX:
		return RCX, 0, 4, true
	case x86asm.EDX:
		return RDX, 0, 4, true
	case x86asm.EBX:
		return RBX, 0, 4, true
	case x86asm.ESP:
		return RSP, 0, 4, true
	case x86asm.EBP:
		return RBP, 0, 4, true
	case x86asm.ESI:
		return RSI, 0, 4, true
	case x86asm.EDI:
		return RDI, 0, 4, true
	case x86asm.R8B, x86asm.R9B, x86asm.R10B, x86asm.R11B, x86asm.R12B, x86asm.R13B, x86asm.R14B, x86asm.R15B:
		return Reg64(reg-x86asm.R8B) + R8, 0, 1, true
	case x86asm.R8W, x86asm.R9W, x86asm.R10W, x86asm.R11W, x86asm.R12W, x86asm.R13W, x86asm.R14W, x86asm.R15W:
		return Reg64(reg-x86asm.R8W) + R8, 0, 2, true
	case x86asm.R8L, x86asm.R9L, x86asm.R10L, x86asm.R11L, x86asm.R12L, x86asm.R13L, x86asm.R14L, x86asm.R15L:
		return Reg64(reg-x86asm.R8L) + R8, 0, 4, true
	case x86asm.RAX, x86asm.RCX, x86asm.RDX, x86asm.RBX, x86asm.RSP, x86asm.RBP, x86asm.RSI, x86asm.RDI:
		return Reg64(reg - x86asm.RAX), 0, 8, true
	case x86asm.R8, x86asm.R9, x86asm.R10, x86asm.R11, x86asm.R12, x86asm.R13, x86asm.R14, x86asm.R15:
		return Reg64(reg-x86asm.R8) + R8, 0, 8, true
	default:
		return 0, 0, 0, false
	}
}

func operand64FromArg(arg x86asm.Arg, width uint8) (operand64, error) {
	switch value := arg.(type) {
	case x86asm.Reg:
		if value >= x86asm.X0 && value <= x86asm.X15 {
			if width != 0 && width != 16 {
				return operand64{}, fmt.Errorf("XMM width %d", width)
			}
			return operand64{Kind: operand64XMM, XMM: uint8(value - x86asm.X0), Width: 16}, nil
		}
		reg, offset, regWidth, ok := reg64FromX86(value)
		if !ok {
			return operand64{}, fmt.Errorf("register %v", value)
		}
		if width == 0 {
			width = regWidth
		}
		if width != regWidth && !(regWidth == 8 && (width == 4 || width == 2 || width == 1)) {
			return operand64{}, fmt.Errorf("register width %d/%d", regWidth, width)
		}
		return operand64{Kind: operand64Reg, Reg: reg, ByteOffset: offset, Width: width}, nil
	case x86asm.Mem:
		mem := memoryOperand64{Disp: value.Disp, Scale: value.Scale, Segment: value.Segment}
		if value.Base == x86asm.RIP {
			mem.RIPRelative = true
		} else if value.Base != 0 {
			reg, _, _, ok := reg64FromX86(value.Base)
			if !ok {
				return operand64{}, fmt.Errorf("memory base %v", value.Base)
			}
			mem.Base, mem.HasBase = reg, true
		}
		if value.Index != 0 {
			reg, _, _, ok := reg64FromX86(value.Index)
			if !ok {
				return operand64{}, fmt.Errorf("memory index %v", value.Index)
			}
			mem.Index, mem.HasIndex = reg, true
		}
		if mem.Scale == 0 {
			mem.Scale = 1
		}
		if width == 0 {
			width = 8
		}
		return operand64{Kind: operand64Mem, Mem: mem, Width: width}, nil
	case x86asm.Imm:
		return operand64{Kind: operand64Imm, Imm: uint64(value), Width: width}, nil
	case x86asm.Rel:
		return operand64{Kind: operand64Rel, Rel: int64(value), Width: width}, nil
	default:
		return operand64{}, fmt.Errorf("operand %T", arg)
	}
}

func instructionWidth64(inst x86asm.Inst, first, second x86asm.Arg) uint8 {
	for _, arg := range []x86asm.Arg{first, second} {
		if reg, ok := arg.(x86asm.Reg); ok {
			if _, _, width, ok := reg64FromX86(reg); ok {
				return width
			}
		}
	}
	if inst.DataSize == 64 {
		return 8
	}
	if inst.DataSize == 16 {
		return 2
	}
	return 4
}

func compileInstruction64(inst x86asm.Inst, address uint64) (microOp64, bool, error) {
	arg := func(index int) x86asm.Arg {
		if index < 0 || index >= len(inst.Args) {
			return nil
		}
		return inst.Args[index]
	}
	width := instructionWidth64(inst, arg(0), arg(1))
	two := func() (operand64, operand64, error) {
		left, err := operand64FromArg(arg(0), width)
		if err != nil {
			return operand64{}, operand64{}, err
		}
		right, err := operand64FromArg(arg(1), width)
		return left, right, err
	}
	switch inst.Op {
	case x86asm.NOP:
		return microOp64{Address: address, Size: uint8(inst.Len), Run: func(state *MachineState64, next uint64) (Flow64, error) {
			state.RIP = next
			return Flow64Continue, nil
		}}, false, nil
	case x86asm.MOV:
		left, right, err := two()
		if err != nil || left.Kind == operand64Imm || left.Kind == operand64Rel {
			return microOp64{}, false, fmt.Errorf("MOV destination: %v", err)
		}
		return makeMove64(address, uint8(inst.Len), left, right), false, nil
	case x86asm.MOVDQA, x86asm.MOVDQU:
		left, err := operand64FromArg(arg(0), 16)
		if err != nil || (left.Kind != operand64XMM && left.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("SSE move destination: %v", err)
		}
		right, err := operand64FromArg(arg(1), 16)
		if err != nil || (right.Kind != operand64XMM && right.Kind != operand64Mem) || (left.Kind == operand64Mem && right.Kind == operand64Mem) {
			return microOp64{}, false, fmt.Errorf("SSE move source: %v", err)
		}
		return makeSSEMove64(address, uint8(inst.Len), left, right), false, nil
	case x86asm.PXOR, x86asm.PAND, x86asm.POR:
		left, err := operand64FromArg(arg(0), 16)
		if err != nil || left.Kind != operand64XMM {
			return microOp64{}, false, fmt.Errorf("SSE ALU destination: %v", err)
		}
		right, err := operand64FromArg(arg(1), 16)
		if err != nil || (right.Kind != operand64XMM && right.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("SSE ALU source: %v", err)
		}
		return makeSSEBinary64(address, uint8(inst.Len), inst.Op, left, right), false, nil

	case x86asm.LEA:
		left, err := operand64FromArg(arg(0), width)
		if err != nil || left.Kind != operand64Reg {
			return microOp64{}, false, fmt.Errorf("LEA destination: %v", err)
		}
		right, err := operand64FromArg(arg(1), 8)
		if err != nil || right.Kind != operand64Mem {
			return microOp64{}, false, fmt.Errorf("LEA source: %v", err)
		}
		return microOp64{Address: address, Size: uint8(inst.Len), Run: func(state *MachineState64, next uint64) (Flow64, error) {
			value, err := effectiveAddress64(state, right.Mem, next)
			if err != nil {
				return Flow64Stop, err
			}
			writeReg64(state, left, uint64(value))
			state.RIP = next
			return Flow64Continue, nil
		}}, false, nil
	case x86asm.ADD, x86asm.SUB, x86asm.XOR, x86asm.AND, x86asm.OR, x86asm.CMP, x86asm.TEST:
		left, right, err := two()
		if err != nil {
			return microOp64{}, false, err
		}
		return makeBinary64(address, uint8(inst.Len), inst.Op, left, right), false, nil
	case x86asm.XCHG:
		left, right, err := two()
		if err != nil || (left.Kind == operand64Mem && right.Kind == operand64Mem) || left.Kind == operand64Imm || right.Kind == operand64Imm {
			return microOp64{}, false, fmt.Errorf("XCHG requires register and register/memory operands: %v", err)
		}
		return makeXchg64(address, uint8(inst.Len), left, right), false, nil
	case x86asm.XADD:
		left, right, err := two()
		if err != nil || left.Kind == operand64Imm || right.Kind != operand64Reg {
			return microOp64{}, false, fmt.Errorf("XADD requires r/m destination and register source: %v", err)
		}
		return makeXadd64(address, uint8(inst.Len), left, right), false, nil
	case x86asm.CMPXCHG:
		left, right, err := two()
		if err != nil || left.Kind == operand64Imm || right.Kind != operand64Reg {
			return microOp64{}, false, fmt.Errorf("CMPXCHG requires r/m destination and register source: %v", err)
		}
		return makeCmpxchg64(address, uint8(inst.Len), left, right), false, nil

	case x86asm.INC, x86asm.DEC, x86asm.NEG, x86asm.NOT:
		value, err := operand64FromArg(arg(0), width)
		if err != nil || value.Kind == operand64Imm || value.Kind == operand64Rel {
			return microOp64{}, false, err
		}
		return makeUnary64(address, uint8(inst.Len), inst.Op, value), false, nil
	case x86asm.PUSH:
		value, err := operand64FromArg(arg(0), 8)
		if err != nil {
			return microOp64{}, false, err
		}
		return makePush64(address, uint8(inst.Len), value), false, nil
	case x86asm.POP:
		value, err := operand64FromArg(arg(0), 8)
		if err != nil || value.Kind == operand64Imm || value.Kind == operand64Rel {
			return microOp64{}, false, err
		}
		return makePop64(address, uint8(inst.Len), value), false, nil
	case x86asm.JMP:
		value, err := operand64FromArg(arg(0), 8)
		if err != nil {
			return microOp64{}, false, err
		}
		return makeJmp64(address, uint8(inst.Len), value), true, nil
	case x86asm.CALL:
		value, err := operand64FromArg(arg(0), 8)
		if err != nil {
			return microOp64{}, false, err
		}
		return makeCall64(address, uint8(inst.Len), value), true, nil
	case x86asm.RET:
		return makeRet64(address, uint8(inst.Len)), true, nil
	case x86asm.SYSCALL:
		return microOp64{Address: address, Size: uint8(inst.Len), Run: func(state *MachineState64, next uint64) (Flow64, error) {
			state.RIP = next
			state.TrapNo = 0x80
			return Flow64Interrupt, nil
		}}, true, nil
	case x86asm.INT:
		vector, ok := arg(0).(x86asm.Imm)
		if !ok {
			return microOp64{}, false, errors.New("INT requires immediate vector")
		}
		return microOp64{Address: address, Size: uint8(inst.Len), Run: func(state *MachineState64, next uint64) (Flow64, error) {
			state.RIP = next
			state.TrapNo = uint64(uint8(vector))
			return Flow64Interrupt, nil
		}}, true, nil
	case x86asm.CLD, x86asm.STD:
		set := inst.Op == x86asm.STD
		return microOp64{Address: address, Size: uint8(inst.Len), Run: func(state *MachineState64, next uint64) (Flow64, error) {
			if set {
				state.RFLAGS |= Flag64DF
			} else {
				state.RFLAGS &^= Flag64DF
			}
			state.RIP = next
			return Flow64Continue, nil
		}}, false, nil
	default:
		if condition, ok := decodeCondition64(inst.Op); ok {
			value, err := operand64FromArg(arg(0), 8)
			if err != nil {
				return microOp64{}, false, err
			}
			return makeJcc64(address, uint8(inst.Len), condition, value), true, nil
		}
		return microOp64{}, false, fmt.Errorf("%s", inst.Op)
	}
}

func readReg64(state *MachineState64, operand operand64) uint64 {
	value := state.Get(operand.Reg)
	if operand.Width == 1 {
		return (value >> (8 * operand.ByteOffset)) & 0xff
	}
	if operand.Width == 2 {
		return value & 0xffff
	}
	if operand.Width == 4 {
		return value & 0xffffffff
	}
	return value
}

func writeReg64(state *MachineState64, operand operand64, value uint64) {
	current := state.Get(operand.Reg)
	switch operand.Width {
	case 1:
		mask := uint64(0xff) << (8 * operand.ByteOffset)
		state.Set(operand.Reg, (current&^mask)|((value&0xff)<<(8*operand.ByteOffset)))
	case 2:
		state.Set(operand.Reg, (current&^0xffff)|(value&0xffff))
	case 4:
		state.Set(operand.Reg, value&0xffffffff)
	default:
		state.Set(operand.Reg, value)
	}
}

func mask64Width(width uint8) uint64 {
	switch width {
	case 1:
		return 0xff
	case 2:
		return 0xffff
	case 4:
		return 0xffffffff
	default:
		return ^uint64(0)
	}
}

func effectiveAddress64(state *MachineState64, mem memoryOperand64, next uint64) (Address64, error) {
	address := uint64(0)
	if mem.RIPRelative {
		address += next
	}
	if mem.HasBase {
		address += state.Get(mem.Base)
	}
	if mem.HasIndex {
		address += state.Get(mem.Index) * uint64(mem.Scale)
	}
	address += uint64(mem.Disp)
	switch mem.Segment {
	case x86asm.FS:
		address += state.FSBase
	case x86asm.GS:
		address += state.GSBase
	}
	if !canonicalAddress64(Address64(address)) {
		return 0, ErrRange
	}
	return Address64(address), nil
}

func readOperand64(state *MachineState64, operand operand64, next uint64) (uint64, error) {
	switch operand.Kind {
	case operand64Reg:
		return readReg64(state, operand), nil
	case operand64Imm:
		return operand.Imm & mask64Width(operand.Width), nil
	case operand64Mem:
		address, err := effectiveAddress64(state, operand.Mem, next)
		if err != nil {
			return 0, err
		}
		var raw [8]byte
		if operand.Width == 0 || operand.Width > 8 {
			return 0, ErrUnsupportedAddressing
		}
		if err := state.Memory.Read(address, raw[:operand.Width]); err != nil {
			return 0, err
		}
		switch operand.Width {
		case 1:
			return uint64(raw[0]), nil
		case 2:
			return uint64(raw[0]) | uint64(raw[1])<<8, nil
		case 4:
			return uint64(raw[0]) | uint64(raw[1])<<8 | uint64(raw[2])<<16 | uint64(raw[3])<<24, nil
		default:
			return uint64(raw[0]) | uint64(raw[1])<<8 | uint64(raw[2])<<16 | uint64(raw[3])<<24 | uint64(raw[4])<<32 | uint64(raw[5])<<40 | uint64(raw[6])<<48 | uint64(raw[7])<<56, nil
		}
	default:
		return 0, ErrUnsupportedAddressing
	}
}

func writeOperand64(state *MachineState64, operand operand64, next, value uint64) error {
	if operand.Kind == operand64Reg {
		writeReg64(state, operand, value)
		return nil
	}
	if operand.Kind != operand64Mem || operand.Width == 0 || operand.Width > 8 {
		return ErrUnsupportedAddressing
	}
	address, err := effectiveAddress64(state, operand.Mem, next)
	if err != nil {
		return err
	}
	var raw [8]byte
	for i := uint8(0); i < operand.Width; i++ {
		raw[i] = byte(value >> (8 * i))
	}
	return state.Memory.Write(address, raw[:operand.Width])
}

func readVector64(state *MachineState64, operand operand64, next uint64) ([16]byte, error) {
	var value [16]byte
	switch operand.Kind {
	case operand64XMM:
		if operand.XMM >= uint8(len(state.XMM)) {
			return value, ErrUnsupported64
		}
		copy(value[:], state.XMM[operand.XMM][:])
		return value, nil
	case operand64Mem:
		address, err := effectiveAddress64(state, operand.Mem, next)
		if err != nil {
			return value, err
		}
		if err := state.Memory.Read(address, value[:]); err != nil {
			return value, err
		}
		return value, nil
	default:
		return value, ErrUnsupportedAddressing
	}
}

func writeVector64(state *MachineState64, operand operand64, next uint64, value [16]byte) error {
	switch operand.Kind {
	case operand64XMM:
		if operand.XMM >= uint8(len(state.XMM)) {
			return ErrUnsupported64
		}
		copy(state.XMM[operand.XMM][:], value[:])
		return nil
	case operand64Mem:
		address, err := effectiveAddress64(state, operand.Mem, next)
		if err != nil {
			return err
		}
		return state.Memory.Write(address, value[:])
	default:
		return ErrUnsupportedAddressing
	}
}

func makeSSEMove64(address uint64, size uint8, dst, src operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		value, err := readVector64(state, src, next)
		if err != nil {
			return Flow64Stop, err
		}
		if err := writeVector64(state, dst, next, value); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeSSEBinary64(address uint64, size uint8, op x86asm.Op, dst, src operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		left, err := readVector64(state, dst, next)
		if err != nil {
			return Flow64Stop, err
		}
		right, err := readVector64(state, src, next)
		if err != nil {
			return Flow64Stop, err
		}
		for i := range left {
			switch op {
			case x86asm.PXOR:
				left[i] ^= right[i]
			case x86asm.PAND:
				left[i] &= right[i]
			case x86asm.POR:
				left[i] |= right[i]
			default:
				return Flow64Stop, ErrUnsupported64
			}
		}
		if err := writeVector64(state, dst, next, left); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeMove64(address uint64, size uint8, dst, src operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		value, err := readOperand64(state, src, next)
		if err != nil {
			return Flow64Stop, err
		}
		if err := writeOperand64(state, dst, next, value); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeBinary64(address uint64, size uint8, op x86asm.Op, dst, src operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		left, err := readOperand64(state, dst, next)
		if err != nil {
			return Flow64Stop, err
		}
		right, err := readOperand64(state, src, next)
		if err != nil {
			return Flow64Stop, err
		}
		mask := mask64Width(dst.Width)
		left &= mask
		right &= mask
		var result uint64
		subtraction := false
		switch op {
		case x86asm.ADD:
			var carry uint64
			result, carry = bits.Add64(left, right, 0)
			result &= mask
			state.SetLazyArithmetic(left, right, result, carry != 0, ((^(left ^ right))&(left^result)&(mask^(mask>>1))) != 0, true)
		case x86asm.SUB, x86asm.CMP:
			var borrow uint64
			result, borrow = bits.Sub64(left, right, 0)
			result &= mask
			subtraction = true
			state.SetLazyArithmetic(left, right, result, borrow != 0, (((left ^ right) & (left ^ result) & (mask ^ (mask >> 1))) != 0), true)
		case x86asm.XOR:
			result = (left ^ right) & mask
			state.SetLazyArithmetic(left, right, result, false, false, false)
		case x86asm.AND, x86asm.TEST:
			result = (left & right) & mask
			state.SetLazyArithmetic(left, right, result, false, false, false)
		case x86asm.OR:
			result = (left | right) & mask
			state.SetLazyArithmetic(left, right, result, false, false, false)
		default:
			return Flow64Stop, ErrUnsupported64
		}
		if op != x86asm.CMP && op != x86asm.TEST {
			if err := writeOperand64(state, dst, next, result); err != nil {
				return Flow64Stop, err
			}
		}
		_ = subtraction
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeXchg64(address uint64, size uint8, left, right operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		leftValue, err := readOperand64(state, left, next)
		if err != nil {
			return Flow64Stop, err
		}
		rightValue, err := readOperand64(state, right, next)
		if err != nil {
			return Flow64Stop, err
		}
		if left.Kind == operand64Mem {
			address, addressErr := effectiveAddress64(state, left.Mem, next)
			if addressErr != nil {
				return Flow64Stop, addressErr
			}
			old, atomicErr := state.Memory.AtomicRMW(address, left.Width, func(uint64) uint64 { return rightValue })
			if atomicErr != nil {
				return Flow64Stop, atomicErr
			}
			leftValue = old
			if err := writeOperand64(state, right, next, leftValue); err != nil {
				return Flow64Stop, err
			}
		} else {
			if err := writeOperand64(state, left, next, rightValue); err != nil {
				return Flow64Stop, err
			}
			if err := writeOperand64(state, right, next, leftValue); err != nil {
				return Flow64Stop, err
			}
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeXadd64(address uint64, size uint8, dst, src operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		srcValue, err := readOperand64(state, src, next)
		if err != nil {
			return Flow64Stop, err
		}
		var old uint64
		if dst.Kind == operand64Mem {
			memoryAddress, addressErr := effectiveAddress64(state, dst.Mem, next)
			if addressErr != nil {
				return Flow64Stop, addressErr
			}
			old, err = state.Memory.AtomicRMW(memoryAddress, dst.Width, func(value uint64) uint64 {
				result, _ := bits.Add64(value&mask64Width(dst.Width), srcValue&mask64Width(dst.Width), 0)
				return result
			})
		} else {
			old, err = readOperand64(state, dst, next)
			if err == nil {
				result, _ := bits.Add64(old&mask64Width(dst.Width), srcValue&mask64Width(dst.Width), 0)
				err = writeOperand64(state, dst, next, result)
			}
		}
		if err != nil {
			return Flow64Stop, err
		}
		mask := mask64Width(dst.Width)
		old &= mask
		srcValue &= mask
		result, carry := bits.Add64(old, srcValue, 0)
		result &= mask
		state.SetLazyArithmetic(old, srcValue, result, carry != 0, ((^(old ^ srcValue))&(old^result)&(mask^(mask>>1))) != 0, true)
		if err := writeOperand64(state, src, next, old); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeCmpxchg64(address uint64, size uint8, dst, src operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		accumulator := operand64{Kind: operand64Reg, Reg: RAX, Width: dst.Width}
		accValue := readReg64(state, accumulator)
		srcValue, err := readOperand64(state, src, next)
		if err != nil {
			return Flow64Stop, err
		}
		var observed uint64
		mask := mask64Width(dst.Width)
		accValue &= mask
		srcValue &= mask
		if dst.Kind == operand64Mem {
			memoryAddress, addressErr := effectiveAddress64(state, dst.Mem, next)
			if addressErr != nil {
				return Flow64Stop, addressErr
			}
			observed, err = state.Memory.AtomicCompareExchange(memoryAddress, dst.Width, accValue, srcValue)
		} else {
			observed, err = readOperand64(state, dst, next)
			if err == nil && observed == accValue {
				err = writeOperand64(state, dst, next, srcValue)
			}
		}
		if err != nil {
			return Flow64Stop, err
		}
		observed &= mask
		result, borrow := bits.Sub64(accValue, observed, 0)
		result &= mask
		state.SetLazyArithmetic(accValue, observed, result, borrow != 0, (((accValue ^ observed) & (accValue ^ result) & (mask ^ (mask >> 1))) != 0), true)
		if accValue != observed {
			writeReg64(state, accumulator, observed)
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeUnary64(address uint64, size uint8, op x86asm.Op, dst operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		value, err := readOperand64(state, dst, next)
		if err != nil {
			return Flow64Stop, err
		}
		mask := mask64Width(dst.Width)
		value &= mask
		result := value
		switch op {
		case x86asm.INC:
			result = (value + 1) & mask
		case x86asm.DEC:
			result = (value - 1) & mask
		case x86asm.NEG:
			result = (-value) & mask
		case x86asm.NOT:
			result = (^value) & mask
		default:
			return Flow64Stop, ErrUnsupported64
		}
		if op != x86asm.NOT {
			state.SetLazyArithmetic(value, 1, result, op == x86asm.DEC || op == x86asm.NEG, false, true)
		}
		if err := writeOperand64(state, dst, next, result); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makePush64(address uint64, size uint8, src operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		value, err := readOperand64(state, src, next)
		if err != nil {
			return Flow64Stop, err
		}
		state.Regs[RSP] -= 8
		if err := state.Memory.Write(Address64(state.Regs[RSP]), []byte{byte(value), byte(value >> 8), byte(value >> 16), byte(value >> 24), byte(value >> 32), byte(value >> 40), byte(value >> 48), byte(value >> 56)}); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makePop64(address uint64, size uint8, dst operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		value, err := state.Memory.ReadUint64(Address64(state.Regs[RSP]))
		if err != nil {
			return Flow64Stop, err
		}
		state.Regs[RSP] += 8
		if err := writeOperand64(state, dst, next, value); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func relativeTarget64(value operand64, next uint64) uint64 {
	if value.Kind == operand64Rel {
		return uint64(int64(next) + value.Rel)
	}
	return value.Imm
}

func makeJmp64(address uint64, size uint8, target operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		if target.Kind == operand64Mem {
			value, err := readOperand64(state, target, next)
			if err != nil {
				return Flow64Stop, err
			}
			state.RIP = value
		} else {
			state.RIP = relativeTarget64(target, next)
		}
		return Flow64Branch, nil
	}}
}

func makeCall64(address uint64, size uint8, target operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		state.Regs[RSP] -= 8
		if err := state.Memory.WriteUint64(Address64(state.Regs[RSP]), next); err != nil {
			return Flow64Stop, err
		}
		if target.Kind == operand64Mem {
			value, err := readOperand64(state, target, next)
			if err != nil {
				return Flow64Stop, err
			}
			state.RIP = value
		} else {
			state.RIP = relativeTarget64(target, next)
		}
		return Flow64Branch, nil
	}}
}

func makeRet64(address uint64, size uint8) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		value, err := state.Memory.ReadUint64(Address64(state.Regs[RSP]))
		if err != nil {
			return Flow64Stop, err
		}
		state.Regs[RSP] += 8
		state.RIP = value
		return Flow64Branch, nil
	}}
}

type conditionCode64 uint8

const (
	condition64O conditionCode64 = iota
	condition64B
	condition64E
	condition64BE
	condition64S
	condition64P
	condition64L
	condition64LE
)

func decodeCondition64(op x86asm.Op) (conditionCode64, bool) {
	switch op {
	case x86asm.JO:
		return condition64O, true
	case x86asm.JNO:
		return condition64O | 0x80, true
	case x86asm.JB:
		return condition64B, true
	case x86asm.JAE:
		return condition64B | 0x80, true
	case x86asm.JE:
		return condition64E, true
	case x86asm.JNE:
		return condition64E | 0x80, true
	case x86asm.JBE:
		return condition64BE, true
	case x86asm.JA:
		return condition64BE | 0x80, true
	case x86asm.JS:
		return condition64S, true
	case x86asm.JNS:
		return condition64S | 0x80, true
	case x86asm.JP:
		return condition64P, true
	case x86asm.JNP:
		return condition64P | 0x80, true
	case x86asm.JL:
		return condition64L, true
	case x86asm.JGE:
		return condition64L | 0x80, true
	case x86asm.JLE:
		return condition64LE, true
	case x86asm.JG:
		return condition64LE | 0x80, true
	default:
		return 0, false
	}
}

func conditionValue64(state *MachineState64, condition conditionCode64) bool {
	invert := condition&0x80 != 0
	base := condition &^ 0x80
	cf, zf, sf, of, pf := state.Flag(Flag64CF), state.Flag(Flag64ZF), state.Flag(Flag64SF), state.Flag(Flag64OF), state.Flag(Flag64PF)
	value := false
	switch base {
	case condition64O:
		value = of
	case condition64B:
		value = cf
	case condition64E:
		value = zf
	case condition64BE:
		value = cf || zf
	case condition64S:
		value = sf
	case condition64P:
		value = pf
	case condition64L:
		value = sf != of
	case condition64LE:
		value = zf || sf != of
	}
	if invert {
		return !value
	}
	return value
}

func makeJcc64(address uint64, size uint8, condition conditionCode64, target operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		if conditionValue64(state, condition) {
			state.RIP = relativeTarget64(target, next)
		} else {
			state.RIP = next
		}
		return Flow64Branch, nil
	}}
}

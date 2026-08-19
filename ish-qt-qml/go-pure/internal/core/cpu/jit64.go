package cpu

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/bits"
	"sync"

	"github.com/mostafa637/mostafa637/go-pure/internal/core/emu/fpu"
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
	case x86asm.RDFSBASE, x86asm.RDGSBASE, x86asm.WRFSBASE, x86asm.WRGSBASE:
		baseOperand, err := operand64FromArg(arg(0), width)
		if err != nil || baseOperand.Kind != operand64Reg {
			return microOp64{}, false, fmt.Errorf("%s operand: %v", inst.Op, err)
		}
		return makeFSBase64(address, uint8(inst.Len), inst.Op, baseOperand), false, nil
	case x86asm.XGETBV, x86asm.XSETBV:
		return makeXCR64(address, uint8(inst.Len), inst.Op), false, nil
	case x86asm.FLD1, x86asm.FLDZ:
		return makeFPUConst64(address, uint8(inst.Len), inst.Op), false, nil
	case x86asm.FCHS, x86asm.FABS:
		return makeFPUUnary64(address, uint8(inst.Len), inst.Op), false, nil
	case x86asm.FINCSTP, x86asm.FDECSTP:
		return makeFPUTopMove64(address, uint8(inst.Len), inst.Op), false, nil
	case x86asm.FXCH:
		index, ok := x87Index64(arg(0))
		if !ok {
			return microOp64{}, false, fmt.Errorf("FXCH operand %v", arg(0))
		}
		return makeFPUFXCH64(address, uint8(inst.Len), index), false, nil
	case x86asm.FLD:
		if index, ok := x87Index64(arg(0)); ok {
			return makeFPULoadReg64(address, uint8(inst.Len), index), false, nil
		}
		if mem, ok := arg(0).(x86asm.Mem); ok && (inst.MemBytes == 4 || inst.MemBytes == 8) {
			operand, err := operand64FromArg(mem, uint8(inst.MemBytes))
			if err != nil {
				return microOp64{}, false, err
			}
			return makeFPULoadMem64(address, uint8(inst.Len), operand), false, nil
		}
		return microOp64{}, false, fmt.Errorf("FLD operand %v", arg(0))
	case x86asm.FST, x86asm.FSTP:
		if index, ok := x87Index64(arg(0)); ok {
			return makeFPUStoreReg64(address, uint8(inst.Len), inst.Op, index), false, nil
		}
		if mem, ok := arg(0).(x86asm.Mem); ok && (inst.MemBytes == 4 || inst.MemBytes == 8) {
			operand, err := operand64FromArg(mem, uint8(inst.MemBytes))
			if err != nil {
				return microOp64{}, false, err
			}
			return makeFPUStoreMem64(address, uint8(inst.Len), inst.Op, operand), false, nil
		}
		return microOp64{}, false, fmt.Errorf("FST operand %v", arg(0))
	case x86asm.FADD, x86asm.FADDP, x86asm.FSUB, x86asm.FSUBP, x86asm.FSUBRP, x86asm.FMUL, x86asm.FMULP, x86asm.FDIV, x86asm.FDIVP, x86asm.FDIVRP:
		if mem, ok := arg(0).(x86asm.Mem); ok && (inst.MemBytes == 4 || inst.MemBytes == 8) {
			operand, err := operand64FromArg(mem, uint8(inst.MemBytes))
			if err != nil {
				return microOp64{}, false, err
			}
			return makeFPUArithmeticMem64(address, uint8(inst.Len), inst.Op, operand), false, nil
		}
		left, leftOK := x87Index64(arg(0))
		right, rightOK := x87Index64(arg(1))
		if !leftOK || !rightOK {
			return microOp64{}, false, fmt.Errorf("FPU arithmetic operands %v", inst.Args)
		}
		return makeFPUArithmeticReg64(address, uint8(inst.Len), inst.Op, left, right), false, nil

	case x86asm.MOV:
		left, right, err := two()
		if err != nil || left.Kind == operand64Imm || left.Kind == operand64Rel {
			return microOp64{}, false, fmt.Errorf("MOV destination: %v", err)
		}
		return makeMove64(address, uint8(inst.Len), left, right), false, nil
	case x86asm.MOVD, x86asm.MOVQ:
		scalarWidth := uint8(4)
		if inst.Op == x86asm.MOVQ {
			scalarWidth = 8
		}
		leftWidth, rightWidth := scalarWidth, scalarWidth
		if reg, ok := arg(0).(x86asm.Reg); ok && reg >= x86asm.X0 && reg <= x86asm.X15 {
			leftWidth = 16
		}
		if reg, ok := arg(1).(x86asm.Reg); ok && reg >= x86asm.X0 && reg <= x86asm.X15 {
			rightWidth = 16
		}
		left, err := operand64FromArg(arg(0), leftWidth)
		if err != nil {
			return microOp64{}, false, fmt.Errorf("%s destination: %v", inst.Op, err)
		}
		right, err := operand64FromArg(arg(1), rightWidth)
		if err != nil {
			return microOp64{}, false, fmt.Errorf("%s source: %v", inst.Op, err)
		}
		if (left.Kind != operand64XMM && left.Kind != operand64Reg && left.Kind != operand64Mem) || (right.Kind != operand64XMM && right.Kind != operand64Reg && right.Kind != operand64Mem) || (left.Kind == operand64Mem && right.Kind == operand64Mem) {
			return microOp64{}, false, fmt.Errorf("%s unsupported operands", inst.Op)
		}
		if left.Kind != operand64XMM && right.Kind != operand64XMM {
			return microOp64{}, false, fmt.Errorf("%s requires an XMM operand", inst.Op)
		}
		return makeMOVScalar64(address, uint8(inst.Len), scalarWidth, left, right), false, nil
	case x86asm.BSWAP:
		destination, err := operand64FromArg(arg(0), 0)
		if err != nil || destination.Kind != operand64Reg {
			return microOp64{}, false, fmt.Errorf("BSWAP destination: %v", err)
		}
		return makeBSWAP64(address, uint8(inst.Len), destination), false, nil
	case x86asm.LZCNT, x86asm.TZCNT:
		destination, err := operand64FromArg(arg(0), 0)
		if err != nil || destination.Kind != operand64Reg {
			return microOp64{}, false, fmt.Errorf("%s destination: %v", inst.Op, err)
		}
		source, err := operand64FromArg(arg(1), destination.Width)
		if err != nil || (source.Kind != operand64Reg && source.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("%s source: %v", inst.Op, err)
		}
		return makeCountZeros64(address, uint8(inst.Len), inst.Op, destination, source), false, nil
	case x86asm.BSF, x86asm.BSR:
		destination, err := operand64FromArg(arg(0), 0)
		if err != nil || destination.Kind != operand64Reg {
			return microOp64{}, false, fmt.Errorf("%s destination: %v", inst.Op, err)
		}
		source, err := operand64FromArg(arg(1), destination.Width)
		if err != nil || (source.Kind != operand64Reg && source.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("%s source: %v", inst.Op, err)
		}
		return makeBitScan64(address, uint8(inst.Len), inst.Op, destination, source), false, nil
	case x86asm.POPCNT:
		destination, err := operand64FromArg(arg(0), 0)
		if err != nil || destination.Kind != operand64Reg {
			return microOp64{}, false, fmt.Errorf("POPCNT destination: %v", err)
		}
		source, err := operand64FromArg(arg(1), destination.Width)
		if err != nil || (source.Kind != operand64Reg && source.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("POPCNT source: %v", err)
		}
		return makePOPCNT64(address, uint8(inst.Len), destination, source), false, nil
	case x86asm.BT, x86asm.BTS, x86asm.BTR, x86asm.BTC:
		destination, err := operand64FromArg(arg(0), width)
		if err != nil || (destination.Kind != operand64Reg && destination.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("%s destination: %v", inst.Op, err)
		}
		index, err := operand64FromArg(arg(1), width)
		if err != nil || (index.Kind != operand64Reg && index.Kind != operand64Imm) {
			return microOp64{}, false, fmt.Errorf("%s bit index: %v", inst.Op, err)
		}
		return makeBitTest64(address, uint8(inst.Len), inst.Op, destination, index), false, nil
	case x86asm.PUSHF, x86asm.PUSHFQ:
		return makePushFlags64(address, uint8(inst.Len)), false, nil
	case x86asm.POPF, x86asm.POPFQ:
		return makePopFlags64(address, uint8(inst.Len)), false, nil
	case x86asm.MOVZX, x86asm.MOVSX:
		destination, err := operand64FromArg(arg(0), 0)
		if err != nil || destination.Kind != operand64Reg {
			return microOp64{}, false, fmt.Errorf("extend destination: %v", err)
		}
		sourceWidth := uint8(0)
		if reg, ok := arg(1).(x86asm.Reg); ok {
			_, _, sourceWidth, ok = reg64FromX86(reg)
			if !ok {
				return microOp64{}, false, fmt.Errorf("extend source register %v", reg)
			}
		} else if _, ok := arg(1).(x86asm.Mem); ok {
			sourceWidth = uint8(inst.MemBytes)
		}
		if sourceWidth != 1 && sourceWidth != 2 && sourceWidth != 4 {
			return microOp64{}, false, fmt.Errorf("extend source width %d", sourceWidth)
		}
		source, err := operand64FromArg(arg(1), sourceWidth)
		if err != nil || (source.Kind != operand64Reg && source.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("extend source: %v", err)
		}
		return makeExtend64(address, uint8(inst.Len), inst.Op == x86asm.MOVSX, destination, source), false, nil
	case x86asm.MOVSXD:
		destination, err := operand64FromArg(arg(0), 8)
		if err != nil || destination.Kind != operand64Reg {
			return microOp64{}, false, fmt.Errorf("MOVSXD destination: %v", err)
		}
		source, err := operand64FromArg(arg(1), 4)
		if err != nil || (source.Kind != operand64Reg && source.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("MOVSXD source: %v", err)
		}
		return makeExtend64(address, uint8(inst.Len), true, destination, source), false, nil
	case x86asm.MOVBE:
		left, err := operand64FromArg(arg(0), width)
		if err != nil {
			return microOp64{}, false, err
		}
		right, err := operand64FromArg(arg(1), width)
		if err != nil || (left.Kind == operand64Mem && right.Kind == operand64Mem) || (left.Kind != operand64Mem && right.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("MOVBE requires one register and one memory operand: %v", err)
		}
		return makeMOVBE64(address, uint8(inst.Len), left, right), false, nil
	case x86asm.CBW, x86asm.CWDE, x86asm.CDQ, x86asm.CDQE, x86asm.CWD, x86asm.CQO:
		return makeConvertAccumulator64(address, uint8(inst.Len), inst.Op), false, nil
	case x86asm.SHL, x86asm.SHR, x86asm.SAR, x86asm.ROL, x86asm.ROR, x86asm.RCL, x86asm.RCR:
		destination, err := operand64FromArg(arg(0), width)
		if err != nil || (destination.Kind != operand64Reg && destination.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("%s destination: %v", inst.Op, err)
		}
		count, err := operand64FromArg(arg(1), 1)
		if err != nil || (count.Kind != operand64Reg && count.Kind != operand64Imm) {
			return microOp64{}, false, fmt.Errorf("%s count: %v", inst.Op, err)
		}
		return makeShift64(address, uint8(inst.Len), inst.Op, destination, count), false, nil
	case x86asm.CMPXCHG8B, x86asm.CMPXCHG16B:
		memoryArg, ok := arg(0).(x86asm.Mem)
		if !ok {
			return microOp64{}, false, fmt.Errorf("%s requires a memory operand", inst.Op)
		}
		memoryWidth := uint8(8)
		if inst.Op == x86asm.CMPXCHG16B {
			memoryWidth = 16
		}
		destination, err := operand64FromArg(memoryArg, memoryWidth)
		if err != nil {
			return microOp64{}, false, fmt.Errorf("%s destination: %v", inst.Op, err)
		}
		return makeCmpxchgB64(address, uint8(inst.Len), inst.Op, destination), false, nil
	case x86asm.MUL, x86asm.DIV, x86asm.IDIV:
		if arg(0) == nil || arg(1) != nil {
			return microOp64{}, false, fmt.Errorf("%s requires one operand", inst.Op)
		}
		source, err := operand64FromArg(arg(0), width)
		if err != nil || (source.Kind != operand64Reg && source.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("%s source: %v", inst.Op, err)
		}
		return makeImplicitArithmetic64(address, uint8(inst.Len), inst.Op, source, width), false, nil
	case x86asm.IMUL:
		if arg(0) == nil {
			return microOp64{}, false, errors.New("IMUL requires an operand")
		}
		if arg(1) == nil {
			source, err := operand64FromArg(arg(0), width)
			if err != nil || (source.Kind != operand64Reg && source.Kind != operand64Mem) {
				return microOp64{}, false, fmt.Errorf("IMUL source: %v", err)
			}
			return makeImplicitArithmetic64(address, uint8(inst.Len), inst.Op, source, width), false, nil
		}
		destination, err := operand64FromArg(arg(0), width)
		if err != nil || destination.Kind != operand64Reg {
			return microOp64{}, false, fmt.Errorf("IMUL destination: %v", err)
		}
		source, err := operand64FromArg(arg(1), width)
		if err != nil || (source.Kind != operand64Reg && source.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("IMUL source: %v", err)
		}
		if arg(2) == nil {
			return makeExplicitIMul64(address, uint8(inst.Len), destination, destination, source, width), false, nil
		}
		immediate, ok := arg(2).(x86asm.Imm)
		if !ok {
			return microOp64{}, false, errors.New("IMUL third operand must be immediate")
		}
		multiplier := operand64{Kind: operand64Imm, Imm: uint64(immediate), Width: width}
		return makeExplicitIMul64(address, uint8(inst.Len), destination, source, multiplier, width), false, nil
	case x86asm.ADD, x86asm.SUB, x86asm.XOR, x86asm.AND, x86asm.OR, x86asm.CMP, x86asm.TEST:
		left, right, err := two()
		if err != nil {
			return microOp64{}, false, err
		}
		return makeBinary64(address, uint8(inst.Len), inst.Op, left, right), false, nil
	case x86asm.ADC, x86asm.SBB:
		left, right, err := two()
		if err != nil || left.Kind == operand64Imm {
			return microOp64{}, false, fmt.Errorf("%s operands: %v", inst.Op, err)
		}
		return makeCarryBinary64(address, uint8(inst.Len), inst.Op, left, right), false, nil
	case x86asm.LAHF, x86asm.SAHF:
		return makeLAHFSAHF64(address, uint8(inst.Len), inst.Op), false, nil
	case x86asm.MOVSS, x86asm.MOVSD_XMM:
		width := uint8(4)
		if inst.Op == x86asm.MOVSD_XMM {
			width = 8
		}

		left, err := operand64ScalarSSEFromArg(arg(0), width)
		if err != nil || (left.Kind != operand64XMM && left.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("scalar SSE move destination: %v", err)
		}
		right, err := operand64ScalarSSEFromArg(arg(1), width)
		if err != nil || (right.Kind != operand64XMM && right.Kind != operand64Mem) || (left.Kind == operand64Mem && right.Kind == operand64Mem) {
			return microOp64{}, false, fmt.Errorf("scalar SSE move source: %v", err)
		}
		return makeSSEScalarMove64(address, uint8(inst.Len), width, left, right), false, nil
	case x86asm.ADDSS, x86asm.ADDSD, x86asm.SUBSS, x86asm.SUBSD,
		x86asm.MULSS, x86asm.MULSD, x86asm.DIVSS, x86asm.DIVSD,
		x86asm.MINSS, x86asm.MINSD, x86asm.MAXSS, x86asm.MAXSD,
		x86asm.SQRTSS, x86asm.SQRTSD:
		width := scalarSSEWidth64(inst.Op)
		destination, err := operand64ScalarSSEFromArg(arg(0), width)
		if err != nil || destination.Kind != operand64XMM {
			return microOp64{}, false, fmt.Errorf("scalar SSE destination: %v", err)
		}
		if inst.Op == x86asm.SQRTSS || inst.Op == x86asm.SQRTSD {
			source, sourceErr := operand64ScalarSSEFromArg(arg(1), width)

			if sourceErr != nil {
				return microOp64{}, false, fmt.Errorf("scalar SSE sqrt source: %v", sourceErr)
			}
			return makeSSEScalarUnary64(address, uint8(inst.Len), width, inst.Op, destination, source), false, nil
		}
		source, sourceErr := operand64ScalarSSEFromArg(arg(1), width)
		if sourceErr != nil || (source.Kind != operand64XMM && source.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("scalar SSE source: %v", sourceErr)
		}
		return makeSSEScalarBinary64(address, uint8(inst.Len), width, inst.Op, destination, source), false, nil
	case x86asm.COMISS, x86asm.COMISD, x86asm.UCOMISS, x86asm.UCOMISD:
		width := scalarSSEWidth64(inst.Op)
		left, err := operand64ScalarSSEFromArg(arg(0), width)
		if err != nil || (left.Kind != operand64XMM && left.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("scalar SSE compare left: %v", err)
		}
		right, rightErr := operand64ScalarSSEFromArg(arg(1), width)
		if rightErr != nil || (right.Kind != operand64XMM && right.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("scalar SSE compare right: %v", rightErr)
		}
		return makeSSECompare64(address, uint8(inst.Len), width, left, right), false, nil
	case x86asm.CVTSI2SS, x86asm.CVTSI2SD:
		destination, err := operand64FromArg(arg(0), 16)
		if err != nil || destination.Kind != operand64XMM {
			return microOp64{}, false, fmt.Errorf("%s destination: %v", inst.Op, err)
		}
		sourceWidth := instructionDataWidth64(inst)
		source, sourceErr := operand64FromArg(arg(1), sourceWidth)
		if sourceErr != nil || (source.Kind != operand64Reg && source.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("%s source: %v", inst.Op, sourceErr)
		}
		return makeCVTScalar64(address, uint8(inst.Len), inst.Op, destination, source, sourceWidth), false, nil
	case x86asm.CVTSS2SD, x86asm.CVTSD2SS:
		destination, err := operand64FromArg(arg(0), 16)
		if err != nil || destination.Kind != operand64XMM {
			return microOp64{}, false, fmt.Errorf("%s destination: %v", inst.Op, err)
		}
		sourceWidth := uint8(4)
		if inst.Op == x86asm.CVTSD2SS {
			sourceWidth = 8
		}
		source, sourceErr := operand64ScalarSSEFromArg(arg(1), sourceWidth)
		if sourceErr != nil || (source.Kind != operand64XMM && source.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("%s source: %v", inst.Op, sourceErr)
		}
		return makeCVTScalar64(address, uint8(inst.Len), inst.Op, destination, source, sourceWidth), false, nil
	case x86asm.CVTSS2SI, x86asm.CVTSD2SI, x86asm.CVTTSS2SI, x86asm.CVTTSD2SI:
		destinationWidth := instructionDataWidth64(inst)
		destination, err := operand64FromArg(arg(0), destinationWidth)
		if err != nil || destination.Kind != operand64Reg {
			return microOp64{}, false, fmt.Errorf("%s destination: %v", inst.Op, err)
		}
		sourceWidth := uint8(4)
		if inst.Op == x86asm.CVTSD2SI || inst.Op == x86asm.CVTTSD2SI {
			sourceWidth = 8
		}
		source, sourceErr := operand64ScalarSSEFromArg(arg(1), sourceWidth)
		if sourceErr != nil || (source.Kind != operand64XMM && source.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("%s source: %v", inst.Op, sourceErr)
		}
		return makeCVTScalar64(address, uint8(inst.Len), inst.Op, destination, source, sourceWidth), false, nil
	case x86asm.CVTDQ2PS, x86asm.CVTPS2DQ, x86asm.CVTTPS2DQ, x86asm.CVTDQ2PD, x86asm.CVTPD2DQ:
		destination, err := operand64FromArg(arg(0), 16)
		if err != nil || destination.Kind != operand64XMM {
			return microOp64{}, false, fmt.Errorf("%s destination: %v", inst.Op, err)
		}
		source, err := operand64FromArg(arg(1), 16)
		if err != nil || (source.Kind != operand64XMM && source.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("%s source: %v", inst.Op, err)
		}
		return makeSSEPackedConvert64(address, uint8(inst.Len), inst.Op, destination, source), false, nil
	case x86asm.MOVDQA, x86asm.MOVDQU, x86asm.MOVAPS, x86asm.MOVUPS, x86asm.MOVAPD, x86asm.MOVUPD:

		left, err := operand64FromArg(arg(0), 16)
		if err != nil || (left.Kind != operand64XMM && left.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("SSE move destination: %v", err)
		}
		right, err := operand64FromArg(arg(1), 16)
		if err != nil || (right.Kind != operand64XMM && right.Kind != operand64Mem) || (left.Kind == operand64Mem && right.Kind == operand64Mem) {
			return microOp64{}, false, fmt.Errorf("SSE move source: %v", err)
		}
		return makeSSEMove64(address, uint8(inst.Len), left, right), false, nil
	case x86asm.PINSRB, x86asm.PINSRW, x86asm.PINSRD, x86asm.PINSRQ:
		width, ok := packedInsertWidth64(inst.Op)
		if !ok {
			return microOp64{}, false, ErrUnsupported64
		}
		destination, err := operand64FromArg(arg(0), 16)
		if err != nil || destination.Kind != operand64XMM {
			return microOp64{}, false, fmt.Errorf("%s destination: %v", inst.Op, err)
		}
		source, err := operand64FromArg(arg(1), width)
		if err != nil && inst.Op != x86asm.PINSRQ {
			source, err = operand64FromArg(arg(1), 4)
			if err != nil {
				source, err = operand64FromArg(arg(1), 8)
			}
		}
		if err != nil || (source.Kind != operand64Reg && source.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("%s source: %v", inst.Op, err)
		}
		immediate, ok := arg(2).(x86asm.Imm)
		if !ok {
			return microOp64{}, false, fmt.Errorf("%s requires an immediate index", inst.Op)
		}
		return makeSSEInsert64(address, uint8(inst.Len), inst.Op, destination, source, uint8(immediate)), false, nil
	case x86asm.PEXTRB, x86asm.PEXTRW, x86asm.PEXTRD, x86asm.PEXTRQ:
		width, ok := packedExtractWidth64(inst.Op)
		if !ok {
			return microOp64{}, false, ErrUnsupported64
		}
		destination, err := operand64FromArg(arg(0), width)
		if err != nil && inst.Op != x86asm.PEXTRQ {
			destination, err = operand64FromArg(arg(0), 4)
			if err != nil {
				destination, err = operand64FromArg(arg(0), 8)
			}
		}
		if err != nil || (destination.Kind != operand64Reg && destination.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("%s destination: %v", inst.Op, err)
		}
		source, err := operand64FromArg(arg(1), 16)
		if err != nil || source.Kind != operand64XMM {
			return microOp64{}, false, fmt.Errorf("%s source: %v", inst.Op, err)
		}
		immediate, ok := arg(2).(x86asm.Imm)
		if !ok {
			return microOp64{}, false, fmt.Errorf("%s requires an immediate index", inst.Op)
		}
		return makeSSEExtract64(address, uint8(inst.Len), inst.Op, destination, source, uint8(immediate)), false, nil
	case x86asm.PMULUDQ, x86asm.PMULHUW, x86asm.PMULLW, x86asm.PMULHW, x86asm.PSADBW,
		x86asm.PMADDWD, x86asm.PMADDUBSW, x86asm.PMULHRSW,
		x86asm.PHADDW, x86asm.PHADDSW, x86asm.PHADDD,
		x86asm.PHSUBW, x86asm.PHSUBSW, x86asm.PHSUBD,
		x86asm.PACKSSWB, x86asm.PACKSSDW, x86asm.PACKUSWB,
		x86asm.PADDUSB, x86asm.PADDUSW, x86asm.PSUBUSB, x86asm.PSUBUSW,
		x86asm.PADDSB, x86asm.PADDSW, x86asm.PSUBSB, x86asm.PSUBSW:

		left, err := operand64FromArg(arg(0), 16)
		if err != nil || left.Kind != operand64XMM {
			return microOp64{}, false, fmt.Errorf("SSE special destination: %v", err)
		}
		right, err := operand64FromArg(arg(1), 16)
		if err != nil || (right.Kind != operand64XMM && right.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("SSE special source: %v", err)
		}
		return makeSSESpecialBinary64(address, uint8(inst.Len), inst.Op, left, right), false, nil
	case x86asm.PXOR, x86asm.PAND, x86asm.POR, x86asm.PANDN,
		x86asm.PADDB, x86asm.PADDW, x86asm.PADDD, x86asm.PADDQ,
		x86asm.PSUBB, x86asm.PSUBW, x86asm.PSUBD, x86asm.PSUBQ,
		x86asm.PCMPEQB, x86asm.PCMPEQW, x86asm.PCMPEQD, x86asm.PCMPEQQ,
		x86asm.PCMPGTB, x86asm.PCMPGTW, x86asm.PCMPGTD,

		x86asm.PAVGB, x86asm.PAVGW,
		x86asm.PMINUB, x86asm.PMAXUB, x86asm.PMINUW, x86asm.PMAXUW,
		x86asm.PMINUD, x86asm.PMAXUD, x86asm.PMINSB, x86asm.PMAXSB,
		x86asm.PMINSW, x86asm.PMAXSW, x86asm.PMINSD, x86asm.PMAXSD:

		left, err := operand64FromArg(arg(0), 16)
		if err != nil || left.Kind != operand64XMM {
			return microOp64{}, false, fmt.Errorf("SSE ALU destination: %v", err)
		}
		right, err := operand64FromArg(arg(1), 16)
		if err != nil || (right.Kind != operand64XMM && right.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("SSE ALU source: %v", err)
		}
		return makeSSEBinary64(address, uint8(inst.Len), inst.Op, left, right), false, nil
	case x86asm.PALIGNR:
		destination, err := operand64FromArg(arg(0), 16)
		if err != nil || destination.Kind != operand64XMM {
			return microOp64{}, false, fmt.Errorf("PALIGNR destination: %v", err)
		}
		source, err := operand64FromArg(arg(1), 16)
		if err != nil || (source.Kind != operand64XMM && source.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("PALIGNR source: %v", err)
		}
		immediate, ok := arg(2).(x86asm.Imm)
		if !ok {
			return microOp64{}, false, errors.New("PALIGNR requires an immediate count")
		}
		return makeSSEAlignRight64(address, uint8(inst.Len), destination, source, uint8(immediate)), false, nil
	case x86asm.PBLENDW:
		destination, err := operand64FromArg(arg(0), 16)
		if err != nil || destination.Kind != operand64XMM {
			return microOp64{}, false, fmt.Errorf("PBLENDW destination: %v", err)
		}
		source, err := operand64FromArg(arg(1), 16)
		if err != nil || (source.Kind != operand64XMM && source.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("PBLENDW source: %v", err)
		}
		immediate, ok := arg(2).(x86asm.Imm)
		if !ok {
			return microOp64{}, false, errors.New("PBLENDW requires an immediate mask")
		}
		return makeSSEBlendW64(address, uint8(inst.Len), destination, source, uint8(immediate)), false, nil
	case x86asm.PTEST:
		destination, err := operand64FromArg(arg(0), 16)
		if err != nil || destination.Kind != operand64XMM {
			return microOp64{}, false, fmt.Errorf("PTEST destination: %v", err)
		}
		source, err := operand64FromArg(arg(1), 16)
		if err != nil || (source.Kind != operand64XMM && source.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("PTEST source: %v", err)
		}
		return makeSSETest64(address, uint8(inst.Len), destination, source), false, nil
	case x86asm.MOVDDUP:
		destination, err := operand64FromArg(arg(0), 16)
		if err != nil || destination.Kind != operand64XMM {
			return microOp64{}, false, fmt.Errorf("MOVDDUP destination: %v", err)
		}
		source, err := operand64ScalarSSEFromArg(arg(1), 8)
		if err != nil || (source.Kind != operand64XMM && source.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("MOVDDUP source: %v", err)
		}
		return makeMOVDDUP64(address, uint8(inst.Len), destination, source), false, nil

	case x86asm.PSLLW, x86asm.PSLLD, x86asm.PSLLQ,

		x86asm.PSRLW, x86asm.PSRLD, x86asm.PSRLQ,
		x86asm.PSRAW, x86asm.PSRAD, x86asm.PSLLDQ, x86asm.PSRLDQ:
		destination, err := operand64FromArg(arg(0), 16)
		if err != nil || destination.Kind != operand64XMM {
			return microOp64{}, false, fmt.Errorf("SSE shift destination: %v", err)
		}
		immediate, ok := arg(1).(x86asm.Imm)
		if !ok {
			return microOp64{}, false, errors.New("SSE shift requires an immediate count")
		}
		return makeSSEShift64(address, uint8(inst.Len), inst.Op, destination, uint8(immediate)), false, nil
	case x86asm.PUNPCKLBW, x86asm.PUNPCKHBW,
		x86asm.PUNPCKLWD, x86asm.PUNPCKHWD,
		x86asm.PUNPCKLDQ, x86asm.PUNPCKHDQ,
		x86asm.PUNPCKLQDQ, x86asm.PUNPCKHQDQ:
		left, err := operand64FromArg(arg(0), 16)
		if err != nil || left.Kind != operand64XMM {
			return microOp64{}, false, fmt.Errorf("PUNPCK destination: %v", err)
		}
		right, err := operand64FromArg(arg(1), 16)
		if err != nil || (right.Kind != operand64XMM && right.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("PUNPCK source: %v", err)
		}
		return makeSSEUnpack64(address, uint8(inst.Len), inst.Op, left, right), false, nil
	case x86asm.PSHUFB:
		destination, err := operand64FromArg(arg(0), 16)
		if err != nil || destination.Kind != operand64XMM {
			return microOp64{}, false, fmt.Errorf("PSHUFB destination: %v", err)
		}
		source, err := operand64FromArg(arg(1), 16)
		if err != nil || (source.Kind != operand64XMM && source.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("PSHUFB source: %v", err)
		}
		return makeSSEShuffleBytes64(address, uint8(inst.Len), destination, source), false, nil
	case x86asm.PSHUFD, x86asm.PSHUFLW, x86asm.PSHUFHW:
		destination, err := operand64FromArg(arg(0), 16)
		if err != nil || destination.Kind != operand64XMM {
			return microOp64{}, false, fmt.Errorf("PSHUFD destination: %v", err)
		}
		source, err := operand64FromArg(arg(1), 16)
		if err != nil || (source.Kind != operand64XMM && source.Kind != operand64Mem) {
			return microOp64{}, false, fmt.Errorf("PSHUFD source: %v", err)
		}
		immediate, ok := arg(2).(x86asm.Imm)
		if !ok {
			return microOp64{}, false, errors.New("PSHUFD requires an immediate selector")
		}
		if inst.Op == x86asm.PSHUFD {
			return makeSSEShuffleD64(address, uint8(inst.Len), destination, source, uint8(immediate)), false, nil
		}
		return makeSSEShuffleW64(address, uint8(inst.Len), inst.Op, destination, source, uint8(immediate)), false, nil
	case x86asm.PMOVMSKB:
		destination, err := operand64FromArg(arg(0), 4)
		if err != nil || destination.Kind != operand64Reg {
			return microOp64{}, false, fmt.Errorf("PMOVMSKB destination: %v", err)
		}
		source, err := operand64FromArg(arg(1), 16)
		if err != nil || source.Kind != operand64XMM {
			return microOp64{}, false, fmt.Errorf("PMOVMSKB source: %v", err)
		}
		return makeSSEMovemask64(address, uint8(inst.Len), destination, source), false, nil
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
	case x86asm.LEAVE:
		stackWidth := uint8(8)
		if inst.DataSize == 16 {
			// In long mode 0x66 selects the 32-bit LEAVE form.
			stackWidth = 4
		}
		return makeLeave64(address, uint8(inst.Len), stackWidth), false, nil
	case x86asm.LFENCE, x86asm.MFENCE, x86asm.SFENCE:
		return makeFence64(address, uint8(inst.Len)), false, nil
	case x86asm.UD2:
		return microOp64{Address: address, Size: uint8(inst.Len), Run: func(state *MachineState64, next uint64) (Flow64, error) {
			state.RIP = next
			state.TrapNo = Trap64InvalidOpcode
			return Flow64Interrupt, nil
		}}, true, nil
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
	case x86asm.MOVSB, x86asm.MOVSW, x86asm.MOVSD, x86asm.MOVSQ,

		x86asm.STOSB, x86asm.STOSW, x86asm.STOSD, x86asm.STOSQ,
		x86asm.LODSB, x86asm.LODSW, x86asm.LODSD, x86asm.LODSQ,
		x86asm.CMPSB, x86asm.CMPSW, x86asm.CMPSD, x86asm.CMPSQ,
		x86asm.SCASB, x86asm.SCASW, x86asm.SCASD, x86asm.SCASQ:
		stringWidth, ok := stringWidth64(inst.Op)
		if !ok {
			return microOp64{}, false, ErrUnsupported64
		}
		repeat := stringRepeatMode64(inst.Prefix)
		return makeString64(address, uint8(inst.Len), inst.Op, stringWidth, uint8(inst.AddrSize), repeat), false, nil
	default:
		if condition, ok := decodeSETcc64(inst.Op); ok {
			destination, err := operand64FromArg(arg(0), 1)
			if err != nil || (destination.Kind != operand64Reg && destination.Kind != operand64Mem) {
				return microOp64{}, false, fmt.Errorf("SETcc destination: %v", err)
			}
			return makeSETcc64(address, uint8(inst.Len), condition, destination), false, nil
		}
		if condition, ok := decodeCMOVcc64(inst.Op); ok {
			destination, err := operand64FromArg(arg(0), width)
			if err != nil || destination.Kind != operand64Reg {
				return microOp64{}, false, fmt.Errorf("CMOVcc destination: %v", err)
			}
			source, err := operand64FromArg(arg(1), width)
			if err != nil || (source.Kind != operand64Reg && source.Kind != operand64Mem) {
				return microOp64{}, false, fmt.Errorf("CMOVcc source: %v", err)
			}
			return makeCMOVcc64(address, uint8(inst.Len), condition, destination, source), false, nil
		}
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

func normalizeArithmeticWidth64(width uint8) uint8 {
	switch width {
	case 1, 2, 4, 8:
		return width
	default:
		return 8
	}
}

func signExtend64Width(value uint64, width uint8) int64 {
	width = normalizeArithmeticWidth64(width)
	mask := mask64Width(width)
	value &= mask
	sign := uint64(1) << (uint(width)*8 - 1)
	if value&sign != 0 {
		value |= ^mask
	}
	return int64(value)
}

func negate12864(hi, lo uint64) (uint64, uint64) {
	lo = ^lo + 1
	hi = ^hi
	if lo == 0 {
		hi++
	}
	return hi, lo
}

func mulWide64(left, right uint64, width uint8, signed bool) (hi, lo uint64, overflow bool) {
	width = normalizeArithmeticWidth64(width)
	mask := mask64Width(width)
	left &= mask
	right &= mask
	if !signed {
		hi, lo = bits.Mul64(left, right)
	} else {
		leftSigned := signExtend64Width(left, width)
		rightSigned := signExtend64Width(right, width)
		negative := (leftSigned < 0) != (rightSigned < 0)
		leftMagnitude := uint64(leftSigned)
		rightMagnitude := uint64(rightSigned)
		if leftSigned < 0 {
			leftMagnitude = ^leftMagnitude + 1
		}
		if rightSigned < 0 {
			rightMagnitude = ^rightMagnitude + 1
		}
		hi, lo = bits.Mul64(leftMagnitude, rightMagnitude)
		if negative {
			hi, lo = negate12864(hi, lo)
		}
	}
	bitsInWidth := uint(width) * 8
	if width < 8 {
		high := (lo >> bitsInWidth) | (hi << (64 - bitsInWidth))
		lo &= mask
		hi = high & mask
	}
	if signed {
		expected := uint64(0)
		if lo&(uint64(1)<<(bitsInWidth-1)) != 0 {
			expected = mask
		}
		overflow = hi != expected
	} else {
		overflow = hi != 0
	}
	return hi, lo, overflow
}

func setMultiplyFlags64(state *MachineState64, overflow bool) {
	state.CF = boolByte64(overflow)
	state.OF = boolByte64(overflow)
	state.Lazy = 0
}

func makeShift64(address uint64, size uint8, op x86asm.Op, destination, countOperand operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		value, err := readOperand64(state, destination, next)
		if err != nil {
			return Flow64Stop, err
		}
		count, err := readOperand64(state, countOperand, next)
		if err != nil {
			return Flow64Stop, err
		}
		width := normalizeArithmeticWidth64(destination.Width)
		bitsInWidth := uint64(width) * 8
		countMask := uint64(0x1f)
		if width == 8 {
			countMask = 0x3f
		}
		count &= countMask
		throughCarry := op == x86asm.RCL || op == x86asm.RCR
		if op == x86asm.ROL || op == x86asm.ROR {
			count %= uint64(bitsInWidth)
		} else if throughCarry {
			count %= uint64(bitsInWidth + 1)
		}
		if count == 0 {
			state.RIP = next
			return Flow64Continue, nil
		}
		mask := mask64Width(width)
		value &= mask
		var result uint64
		var carry, overflow bool
		switch op {
		case x86asm.SHL:
			result = (value << count) & mask
			carry = ((value >> (bitsInWidth - count)) & 1) != 0
			if count == 1 {
				overflow = ((result ^ value) & (uint64(1) << (bitsInWidth - 1))) != 0
			}
		case x86asm.SHR:
			result = value >> count
			carry = ((value >> (count - 1)) & 1) != 0
			if count == 1 {
				overflow = (value & (uint64(1) << (bitsInWidth - 1))) != 0
			}
		case x86asm.SAR:
			result = uint64(signExtend64Width(value, width)>>count) & mask
			carry = ((value >> (count - 1)) & 1) != 0
		case x86asm.ROL:
			result = ((value << count) | (value >> (bitsInWidth - count))) & mask
			carry = (result & 1) != 0
			if count == 1 {
				overflow = ((result >> (bitsInWidth - 1)) & 1) != boolToUint64(carry)
			}
		case x86asm.ROR:
			result = ((value >> count) | (value << (bitsInWidth - count))) & mask
			carry = (result>>(bitsInWidth-1))&1 != 0
			if count == 1 {
				overflow = ((result >> (bitsInWidth - 1)) & 1) != ((result >> (bitsInWidth - 2)) & 1)
			}
		case x86asm.RCL, x86asm.RCR:
			carryIn := state.Flag(Flag64CF)
			for i := uint64(0); i < count; i++ {
				if op == x86asm.RCL {
					carry = (value & (uint64(1) << (bitsInWidth - 1))) != 0
					value = ((value << 1) & mask) | boolToUint64(carryIn)
				} else {
					carry = (value & 1) != 0
					value = (value >> 1) | (boolToUint64(carryIn) << (bitsInWidth - 1))
				}
				carryIn = carry
			}
			result = value
			if count == 1 {
				if op == x86asm.RCL {
					overflow = ((result >> (bitsInWidth - 1)) & 1) != boolToUint64(carry)
				} else {
					overflow = ((result >> (bitsInWidth - 1)) & 1) != ((result >> (bitsInWidth - 2)) & 1)
				}
			}
		default:
			return Flow64Stop, ErrUnsupported64
		}
		if op == x86asm.ROL || op == x86asm.ROR || op == x86asm.RCL || op == x86asm.RCR {
			state.CF = boolByte64(carry)
			state.OF = boolByte64(overflow)
		} else {
			state.SetLazyArithmeticWidth(value, count, result, carry, overflow, false, width)
		}
		if err := writeOperand64(state, destination, next, result); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func boolToUint64(value bool) uint64 {
	if value {
		return 1
	}
	return 0
}

func makeImplicitArithmetic64(address uint64, size uint8, op x86asm.Op, source operand64, width uint8) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		value, err := readOperand64(state, source, next)
		if err != nil {
			return Flow64Stop, err
		}
		width = normalizeArithmeticWidth64(width)
		if op == x86asm.MUL || op == x86asm.IMUL {
			hi, lo, overflow := mulWide64(readReg64(state, operand64{Kind: operand64Reg, Reg: RAX, Width: width}), value, width, op == x86asm.IMUL)
			writeReg64(state, operand64{Kind: operand64Reg, Reg: RAX, Width: width}, lo)
			writeReg64(state, operand64{Kind: operand64Reg, Reg: RDX, Width: width}, hi)
			setMultiplyFlags64(state, overflow)
			state.RIP = next
			return Flow64Continue, nil
		}
		var quotient, remainder uint64
		if op == x86asm.DIV {
			quotient, remainder, err = divideUnsigned64(state, value, width)
		} else {
			quotient, remainder, err = divideSigned64(state, value, width)
		}
		if err != nil {
			return Flow64Stop, err
		}
		writeReg64(state, operand64{Kind: operand64Reg, Reg: RAX, Width: width}, quotient)
		writeReg64(state, operand64{Kind: operand64Reg, Reg: RDX, Width: width}, remainder)
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeExplicitIMul64(address uint64, size uint8, destination, leftOperand, rightOperand operand64, width uint8) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		left, err := readOperand64(state, leftOperand, next)
		if err != nil {
			return Flow64Stop, err
		}
		right, err := readOperand64(state, rightOperand, next)
		if err != nil {
			return Flow64Stop, err
		}
		_, result, overflow := mulWide64(left, right, width, true)
		if err := writeOperand64(state, destination, next, result); err != nil {
			return Flow64Stop, err
		}
		setMultiplyFlags64(state, overflow)
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func divideUnsigned64(state *MachineState64, divisor uint64, width uint8) (uint64, uint64, error) {
	width = normalizeArithmeticWidth64(width)
	mask := mask64Width(width)
	divisor &= mask
	if divisor == 0 {
		return 0, 0, ErrDivisionByZero
	}
	if width < 8 {
		bitsInWidth := uint(width) * 8
		dividend := ((state.Get(RDX) & mask) << bitsInWidth) | (state.Get(RAX) & mask)
		quotient := dividend / divisor
		if quotient > mask {
			return 0, 0, ErrDivisionOverflow
		}
		return quotient, dividend % divisor, nil
	}
	hi, lo := state.Get(RDX), state.Get(RAX)
	if hi >= divisor {
		return 0, 0, ErrDivisionOverflow
	}
	quotient, remainder := bits.Div64(hi, lo, divisor)
	return quotient, remainder, nil
}

func divideSigned64(state *MachineState64, divisor uint64, width uint8) (uint64, uint64, error) {
	width = normalizeArithmeticWidth64(width)
	if width < 8 {
		bitsInWidth := uint(width) * 8
		dividend := (signExtend64Width(state.Get(RDX), width) << bitsInWidth) | int64(state.Get(RAX)&mask64Width(width))
		divisorSigned := signExtend64Width(divisor, width)
		if divisorSigned == 0 {
			return 0, 0, ErrDivisionByZero
		}
		minimum := int64(-1) << (bitsInWidth - 1)
		maximum := int64(1)<<(bitsInWidth-1) - 1
		if dividend == minimum && divisorSigned == -1 {
			return 0, 0, ErrDivisionOverflow
		}
		quotient, remainder := dividend/divisorSigned, dividend%divisorSigned
		if quotient < minimum || quotient > maximum {
			return 0, 0, ErrDivisionOverflow
		}
		return uint64(quotient) & mask64Width(width), uint64(remainder) & mask64Width(width), nil
	}
	divisorSigned := int64(divisor)
	if divisorSigned == 0 {
		return 0, 0, ErrDivisionByZero
	}
	hi, lo := state.Get(RDX), state.Get(RAX)
	negativeDividend := hi>>63 != 0
	magnitudeHi, magnitudeLo := hi, lo
	if negativeDividend {
		magnitudeHi, magnitudeLo = negate12864(magnitudeHi, magnitudeLo)
	}
	negativeDivisor := divisorSigned < 0
	magnitudeDivisor := divisor
	if negativeDivisor {
		magnitudeDivisor = ^magnitudeDivisor + 1
	}
	if magnitudeHi >= magnitudeDivisor {
		return 0, 0, ErrDivisionOverflow
	}
	quotientMagnitude, remainderMagnitude := bits.Div64(magnitudeHi, magnitudeLo, magnitudeDivisor)
	negativeQuotient := negativeDividend != negativeDivisor
	limit := ^uint64(0) >> 1
	if negativeQuotient {
		limit++
	}
	if quotientMagnitude > limit {
		return 0, 0, ErrDivisionOverflow
	}
	quotient := quotientMagnitude
	if negativeQuotient {
		quotient = ^quotient + 1
	}
	remainder := remainderMagnitude
	if negativeDividend {
		remainder = ^remainder + 1
	}
	return quotient, remainder, nil
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

func x87Index64(arg x86asm.Arg) (uint8, bool) {
	reg, ok := arg.(x86asm.Reg)
	if !ok || reg < x86asm.F0 || reg > x86asm.F7 {
		return 0, false
	}
	return uint8(reg - x86asm.F0), true
}

func makeFPUConst64(address uint64, size uint8, op x86asm.Op) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		value := fpu.FromFloat64(0)
		if op == x86asm.FLD1 {
			value = fpu.FromFloat64(1)
		}
		state.PushFP(value)
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeFPUUnary64(address uint64, size uint8, op x86asm.Op) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		value := state.FPAt(0)
		if op == x86asm.FCHS {
			value = value.Neg()
		} else {
			value = value.Abs()
		}
		state.SetFPAt(0, value)
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeFPUTopMove64(address uint64, size uint8, op x86asm.Op) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		if op == x86asm.FINCSTP {
			state.MoveFPUTop(1)
		} else {
			state.MoveFPUTop(-1)
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeFPUFXCH64(address uint64, size uint8, index uint8) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		first := state.FPAt(0)
		second := state.FPAt(index)
		state.SetFPAt(0, second)
		state.SetFPAt(index, first)
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeFPULoadReg64(address uint64, size uint8, index uint8) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		state.PushFP(state.FPAt(index))
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeFPUStoreReg64(address uint64, size uint8, op x86asm.Op, index uint8) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		state.SetFPAt(index, state.FPAt(0))
		if op == x86asm.FSTP {
			state.PopFP()
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeFPULoadMem64(address uint64, size uint8, operand operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		raw, err := readOperand64(state, operand, next)
		if err != nil {
			return Flow64Stop, err
		}
		var value float64
		if operand.Width == 4 {
			value = float64(math.Float32frombits(uint32(raw)))
		} else {
			value = math.Float64frombits(raw)
		}
		state.PushFP(fpu.FromFloat64(value))
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeFPUStoreMem64(address uint64, size uint8, op x86asm.Op, operand operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		value := state.FPAt(0).ToFloat64()
		var raw uint64
		if operand.Width == 4 {
			raw = uint64(math.Float32bits(float32(value)))
		} else {
			raw = math.Float64bits(value)
		}
		if err := writeOperand64(state, operand, next, raw); err != nil {
			return Flow64Stop, err
		}
		if op == x86asm.FSTP {
			state.PopFP()
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func fpu64Arithmetic(op x86asm.Op, left, right fpu.Value) fpu.Value {
	switch op {
	case x86asm.FADD, x86asm.FADDP:
		return left.Add(right)
	case x86asm.FSUB, x86asm.FSUBP:
		return left.Sub(right)
	case x86asm.FSUBRP:
		return right.Sub(left)
	case x86asm.FMUL, x86asm.FMULP:
		return left.Mul(right)
	case x86asm.FDIV, x86asm.FDIVP:
		return left.Div(right)
	case x86asm.FDIVRP:
		return right.Div(left)
	default:
		return left
	}
}

func makeFPUArithmeticReg64(address uint64, size uint8, op x86asm.Op, leftIndex, rightIndex uint8) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		left := state.FPAt(leftIndex)
		right := state.FPAt(rightIndex)
		state.SetFPAt(leftIndex, fpu64Arithmetic(op, left, right))
		if op == x86asm.FADDP || op == x86asm.FSUBP || op == x86asm.FSUBRP || op == x86asm.FMULP || op == x86asm.FDIVP || op == x86asm.FDIVRP {
			state.PopFP()
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeFPUArithmeticMem64(address uint64, size uint8, op x86asm.Op, operand operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		raw, err := readOperand64(state, operand, next)
		if err != nil {
			return Flow64Stop, err
		}
		var value float64
		if operand.Width == 4 {
			value = float64(math.Float32frombits(uint32(raw)))
		} else {
			value = math.Float64frombits(raw)
		}
		state.SetFPAt(0, fpu64Arithmetic(op, state.FPAt(0), fpu.FromFloat64(value)))
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func readVector64(state *MachineState64, operand operand64, next uint64) ([16]byte, error) {
	return readVectorWidth64(state, operand, next, 16)
}

func readVectorWidth64(state *MachineState64, operand operand64, next uint64, width uint8) ([16]byte, error) {
	var value [16]byte
	if width > 16 {
		return value, ErrUnsupported64
	}
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
		if err := state.Memory.Read(address, value[:width]); err != nil {
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

func operand64ScalarSSEFromArg(arg x86asm.Arg, width uint8) (operand64, error) {
	if reg, ok := arg.(x86asm.Reg); ok && reg >= x86asm.X0 && reg <= x86asm.X15 {
		return operand64{Kind: operand64XMM, XMM: uint8(reg - x86asm.X0), Width: 16}, nil
	}
	return operand64FromArg(arg, width)
}

func makeSSEScalarMove64(address uint64, size, width uint8, dst, src operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		var value uint64
		if src.Kind == operand64XMM {
			for i := uint8(0); i < width; i++ {
				value |= uint64(state.XMM[src.XMM][i]) << (8 * i)
			}
		} else {
			var err error
			value, err = readOperand64(state, src, next)
			if err != nil {
				return Flow64Stop, err
			}
		}
		if dst.Kind == operand64XMM {
			for i := uint8(0); i < width; i++ {
				state.XMM[dst.XMM][i] = byte(value >> (8 * i))
			}
			if src.Kind == operand64Mem {
				for i := width; i < 16; i++ {
					state.XMM[dst.XMM][i] = 0
				}
			}
		} else if err := writeOperand64(state, dst, next, value); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func scalarSSEWidth64(op x86asm.Op) uint8 {
	if op == x86asm.ADDSS || op == x86asm.SUBSS || op == x86asm.MULSS || op == x86asm.DIVSS ||
		op == x86asm.MINSS || op == x86asm.MAXSS || op == x86asm.SQRTSS ||
		op == x86asm.COMISS || op == x86asm.UCOMISS {
		return 4
	}
	return 8
}

func instructionDataWidth64(inst x86asm.Inst) uint8 {
	if inst.DataSize == 64 {
		return 8
	}
	return 4
}

func cvtFloatToInt64(value float64, width uint8, truncate bool) uint64 {
	limit := math.Ldexp(1, int(width)*8-1)
	if !truncate {
		value = math.RoundToEven(value)
	}
	if math.IsNaN(value) || math.IsInf(value, 0) || value < -limit || value >= limit {
		return uint64(1) << (uint(width)*8 - 1)
	}
	return uint64(int64(value)) & mask64Width(width)
}

func packedConversionSourceWidth64(op x86asm.Op) uint8 {
	if op == x86asm.CVTDQ2PD {
		return 8
	}
	return 16
}

func makeSSEPackedConvert64(address uint64, size uint8, op x86asm.Op, dst, src operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		source, err := readVectorWidth64(state, src, next, packedConversionSourceWidth64(op))
		if err != nil {
			return Flow64Stop, err
		}
		var result [16]byte
		switch op {
		case x86asm.CVTDQ2PS:
			for lane := 0; lane < 4; lane++ {
				value := int32(binary.LittleEndian.Uint32(source[lane*4:]))
				binary.LittleEndian.PutUint32(result[lane*4:], math.Float32bits(float32(value)))
			}
		case x86asm.CVTPS2DQ, x86asm.CVTTPS2DQ:
			truncate := op == x86asm.CVTTPS2DQ
			for lane := 0; lane < 4; lane++ {
				value := float64(math.Float32frombits(binary.LittleEndian.Uint32(source[lane*4:])))
				raw := cvtFloatToInt64(value, 4, truncate)
				binary.LittleEndian.PutUint32(result[lane*4:], uint32(raw))
			}
		case x86asm.CVTDQ2PD:
			for lane := 0; lane < 2; lane++ {
				value := int32(binary.LittleEndian.Uint32(source[lane*4:]))
				binary.LittleEndian.PutUint64(result[lane*8:], math.Float64bits(float64(value)))
			}
		case x86asm.CVTPD2DQ:
			for lane := 0; lane < 2; lane++ {
				value := math.Float64frombits(binary.LittleEndian.Uint64(source[lane*8:]))
				raw := cvtFloatToInt64(value, 4, false)
				binary.LittleEndian.PutUint32(result[lane*4:], uint32(raw))
			}
		default:
			return Flow64Stop, ErrUnsupported64
		}
		if err := writeVector64(state, dst, next, result); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeCVTScalar64(address uint64, size uint8, op x86asm.Op, dst, src operand64, srcWidth uint8) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		switch op {
		case x86asm.CVTSI2SS, x86asm.CVTSI2SD:
			raw, err := readOperand64(state, src, next)
			if err != nil {
				return Flow64Stop, err
			}
			integer := signExtend64Width(raw, srcWidth)
			value := float64(integer)
			destinationWidth := uint8(4)
			if op == x86asm.CVTSI2SD {
				destinationWidth = 8
			}
			if err := writeScalarSSE64(state, dst, destinationWidth, next, value); err != nil {
				return Flow64Stop, err
			}
		case x86asm.CVTSS2SD, x86asm.CVTSD2SS:
			value, err := readScalarSSE64(state, src, srcWidth, next)
			if err != nil {
				return Flow64Stop, err
			}
			destinationWidth := uint8(8)
			if op == x86asm.CVTSD2SS {
				destinationWidth = 4
			}
			if err := writeScalarSSE64(state, dst, destinationWidth, next, value); err != nil {
				return Flow64Stop, err
			}
		case x86asm.CVTSS2SI, x86asm.CVTSD2SI, x86asm.CVTTSS2SI, x86asm.CVTTSD2SI:
			value, err := readScalarSSE64(state, src, srcWidth, next)
			if err != nil {
				return Flow64Stop, err
			}
			truncate := op == x86asm.CVTTSS2SI || op == x86asm.CVTTSD2SI
			if err := writeOperand64(state, dst, next, cvtFloatToInt64(value, dst.Width, truncate)); err != nil {
				return Flow64Stop, err
			}
		default:
			return Flow64Stop, ErrUnsupported64
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func readScalarSSE64(state *MachineState64, operand operand64, width uint8, next uint64) (float64, error) {
	var raw uint64
	if operand.Kind == operand64XMM {
		for i := uint8(0); i < width; i++ {
			raw |= uint64(state.XMM[operand.XMM][i]) << (8 * i)
		}
	} else {
		var err error
		raw, err = readOperand64(state, operand, next)
		if err != nil {
			return 0, err
		}
	}
	if width == 4 {
		return float64(math.Float32frombits(uint32(raw))), nil
	}
	return math.Float64frombits(raw), nil
}

func writeScalarSSE64(state *MachineState64, operand operand64, width uint8, next uint64, value float64) error {
	var raw uint64
	if width == 4 {
		raw = uint64(math.Float32bits(float32(value)))
	} else {
		raw = math.Float64bits(value)
	}
	if operand.Kind == operand64XMM {
		for i := uint8(0); i < width; i++ {
			state.XMM[operand.XMM][i] = byte(raw >> (8 * i))
		}
		return nil
	}
	return writeOperand64(state, operand, next, raw)
}

func scalarSSEBinaryValue64(op x86asm.Op, left, right float64, width uint8) float64 {
	if width == 4 {
		l, r := float32(left), float32(right)
		switch op {
		case x86asm.ADDSS:
			return float64(l + r)
		case x86asm.SUBSS:
			return float64(l - r)
		case x86asm.MULSS:
			return float64(l * r)
		case x86asm.DIVSS:
			return float64(l / r)
		case x86asm.MINSS:
			return float64(scalarSSEMinMax32(l, r, false))
		case x86asm.MAXSS:
			return float64(scalarSSEMinMax32(l, r, true))
		}
	}
	switch op {
	case x86asm.ADDSD:
		return left + right
	case x86asm.SUBSD:
		return left - right
	case x86asm.MULSD:
		return left * right
	case x86asm.DIVSD:
		return left / right
	case x86asm.MINSD:
		return scalarSSEMinMax64(left, right, false)
	case x86asm.MAXSD:
		return scalarSSEMinMax64(left, right, true)
	}
	return left
}

func scalarSSEMinMax32(left, right float32, maximum bool) float32 {
	if math.IsNaN(float64(left)) || math.IsNaN(float64(right)) || left == right {
		return right
	}
	if maximum {
		if left > right {
			return left
		}
		return right
	}
	if left < right {
		return left
	}
	return right
}

func scalarSSEMinMax64(left, right float64, maximum bool) float64 {
	if math.IsNaN(left) || math.IsNaN(right) || left == right {
		return right
	}
	if maximum {
		if left > right {
			return left
		}
		return right
	}
	if left < right {
		return left
	}
	return right
}

func makeSSEScalarBinary64(address uint64, size, width uint8, op x86asm.Op, dst, src operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		left, err := readScalarSSE64(state, dst, width, next)
		if err != nil {
			return Flow64Stop, err
		}
		right, err := readScalarSSE64(state, src, width, next)
		if err != nil {
			return Flow64Stop, err
		}
		if err := writeScalarSSE64(state, dst, width, next, scalarSSEBinaryValue64(op, left, right, width)); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeSSEScalarUnary64(address uint64, size, width uint8, op x86asm.Op, dst, src operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		value, err := readScalarSSE64(state, src, width, next)
		if err != nil {
			return Flow64Stop, err
		}
		if err := writeScalarSSE64(state, dst, width, next, math.Sqrt(value)); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeSSECompare64(address uint64, size, width uint8, left, right operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		lv, err := readScalarSSE64(state, left, width, next)
		if err != nil {
			return Flow64Stop, err
		}
		rv, err := readScalarSSE64(state, right, width, next)
		if err != nil {
			return Flow64Stop, err
		}
		unordered := math.IsNaN(lv) || math.IsNaN(rv)
		var flags uint64
		if unordered {
			flags = Flag64CF | Flag64PF | Flag64ZF
		} else if lv == rv {
			flags = Flag64ZF
		} else if lv < rv {
			flags = Flag64CF
		}
		state.RFLAGS = (state.RFLAGS &^ (Flag64CF | Flag64PF | Flag64AF | Flag64ZF | Flag64SF | Flag64OF)) | flags
		state.CF = boolByte64(flags&Flag64CF != 0)
		state.OF = 0
		state.Lazy = 0
		state.LazyWidth = 0
		state.RIP = next
		return Flow64Continue, nil
	}}
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

func makeSSEAlignRight64(address uint64, size uint8, destination, source operand64, immediate uint8) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		left, err := readVector64(state, destination, next)
		if err != nil {
			return Flow64Stop, err
		}
		right, err := readVector64(state, source, next)
		if err != nil {
			return Flow64Stop, err
		}
		var combined [32]byte
		copy(combined[:16], right[:])
		copy(combined[16:], left[:])
		var result [16]byte
		count := int(immediate)
		if count < len(combined) {
			copy(result[:], combined[count:count+len(result)])
		}
		if err := writeVector64(state, destination, next, result); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeSSEBlendW64(address uint64, size uint8, destination, source operand64, immediate uint8) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		left, err := readVector64(state, destination, next)
		if err != nil {
			return Flow64Stop, err
		}
		right, err := readVector64(state, source, next)
		if err != nil {
			return Flow64Stop, err
		}
		var result [16]byte
		copy(result[:], left[:])
		for lane := 0; lane < 8; lane++ {
			if immediate&(1<<uint(lane)) != 0 {
				copy(result[lane*2:lane*2+2], right[lane*2:lane*2+2])
			}
		}
		if err := writeVector64(state, destination, next, result); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeSSETest64(address uint64, size uint8, destination, source operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		left, err := readVector64(state, destination, next)
		if err != nil {
			return Flow64Stop, err
		}
		right, err := readVector64(state, source, next)
		if err != nil {
			return Flow64Stop, err
		}
		andZero := true
		andNotZero := true
		for i := range left {
			if left[i]&right[i] != 0 {
				andZero = false
			}
			if (^left[i])&right[i] != 0 {
				andNotZero = false
			}
		}
		state.CollapseFlags()
		var flags uint64
		if andNotZero {
			flags |= Flag64CF
		}
		if andZero {
			flags |= Flag64ZF
		}
		state.RFLAGS = (state.RFLAGS &^ (Flag64CF | Flag64PF | Flag64AF | Flag64ZF | Flag64SF | Flag64OF)) | flags
		state.CF = boolByte64(flags&Flag64CF != 0)
		state.OF = 0
		state.Lazy = 0
		state.LazyWidth = 0
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeMOVDDUP64(address uint64, size uint8, destination, source operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		value, err := readScalarSSE64(state, source, 8, next)
		if err != nil {
			return Flow64Stop, err
		}
		raw := math.Float64bits(value)
		var result [16]byte
		for i := 0; i < 8; i++ {
			result[i] = byte(raw >> (8 * i))
			result[i+8] = result[i]
		}
		if err := writeVector64(state, destination, next, result); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func sseLaneWidth64(op x86asm.Op) (int, bool) {
	switch op {
	case x86asm.PADDB, x86asm.PSUBB, x86asm.PCMPEQB, x86asm.PCMPGTB,
		x86asm.PAVGB, x86asm.PMINUB, x86asm.PMAXUB,
		x86asm.PMINSB, x86asm.PMAXSB:
		return 1, true
	case x86asm.PADDW, x86asm.PSUBW, x86asm.PCMPEQW, x86asm.PCMPGTW,
		x86asm.PAVGW, x86asm.PMINUW, x86asm.PMAXUW,
		x86asm.PMINSW, x86asm.PMAXSW:
		return 2, true
	case x86asm.PADDD, x86asm.PSUBD, x86asm.PCMPEQD, x86asm.PCMPGTD,
		x86asm.PMINUD, x86asm.PMAXUD, x86asm.PMINSD, x86asm.PMAXSD:
		return 4, true
	case x86asm.PADDQ, x86asm.PSUBQ, x86asm.PCMPEQQ:
		return 8, true
	default:
		return 0, false
	}
}

func packedInsertWidth64(op x86asm.Op) (uint8, bool) {
	switch op {
	case x86asm.PINSRB:
		return 1, true
	case x86asm.PINSRW:
		return 2, true
	case x86asm.PINSRD:
		return 4, true
	case x86asm.PINSRQ:
		return 8, true
	default:
		return 0, false
	}
}

func packedExtractWidth64(op x86asm.Op) (uint8, bool) {
	switch op {
	case x86asm.PEXTRB:
		return 1, true
	case x86asm.PEXTRW:
		return 2, true
	case x86asm.PEXTRD:
		return 4, true
	case x86asm.PEXTRQ:
		return 8, true
	default:
		return 0, false
	}
}

func makeSSEInsert64(address uint64, size uint8, op x86asm.Op, destination, source operand64, immediate uint8) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		value, err := readOperand64(state, source, next)
		if err != nil {
			return Flow64Stop, err
		}
		result, err := readVector64(state, destination, next)
		if err != nil {
			return Flow64Stop, err
		}
		width, _ := packedInsertWidth64(op)
		indexMask := uint8(16/width - 1)
		index := int(immediate & indexMask)
		offset := index * int(width)
		for i := uint8(0); i < width; i++ {
			result[offset+int(i)] = byte(value >> (8 * i))
		}
		if err := writeVector64(state, destination, next, result); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeSSEExtract64(address uint64, size uint8, op x86asm.Op, destination, source operand64, immediate uint8) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		value, err := readVector64(state, source, next)
		if err != nil {
			return Flow64Stop, err
		}
		width, _ := packedExtractWidth64(op)
		indexMask := uint8(16/width - 1)
		index := int(immediate & indexMask)
		offset := index * int(width)
		var result uint64
		for i := uint8(0); i < width; i++ {
			result |= uint64(value[offset+int(i)]) << (8 * i)
		}
		if err := writeOperand64(state, destination, next, result); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeSSESpecialBinary64(address uint64, size uint8, op x86asm.Op, dst, src operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		left, err := readVector64(state, dst, next)
		if err != nil {
			return Flow64Stop, err
		}
		right, err := readVector64(state, src, next)
		if err != nil {
			return Flow64Stop, err
		}
		var result [16]byte
		switch op {
		case x86asm.PMULUDQ:
			for _, offset := range []int{0, 8} {
				l := uint64(binary.LittleEndian.Uint32(left[offset:]))
				r := uint64(binary.LittleEndian.Uint32(right[offset:]))
				binary.LittleEndian.PutUint64(result[offset:], l*r)
			}
		case x86asm.PMULHUW:
			for offset := 0; offset < 16; offset += 2 {
				leftValue := uint64(binary.LittleEndian.Uint16(left[offset:]))
				rightValue := uint64(binary.LittleEndian.Uint16(right[offset:]))
				binary.LittleEndian.PutUint16(result[offset:], uint16((leftValue*rightValue)>>16))
			}
		case x86asm.PMULLW, x86asm.PMULHW:
			for offset := 0; offset < 16; offset += 2 {
				l := int64(int16(binary.LittleEndian.Uint16(left[offset:])))
				r := int64(int16(binary.LittleEndian.Uint16(right[offset:])))
				product := l * r
				if op == x86asm.PMULHW {
					product >>= 16
				}
				binary.LittleEndian.PutUint16(result[offset:], uint16(product))
			}
		case x86asm.PMADDWD:
			for offset := 0; offset < 16; offset += 4 {
				left0 := int64(int16(binary.LittleEndian.Uint16(left[offset:])))
				left1 := int64(int16(binary.LittleEndian.Uint16(left[offset+2:])))
				right0 := int64(int16(binary.LittleEndian.Uint16(right[offset:])))
				right1 := int64(int16(binary.LittleEndian.Uint16(right[offset+2:])))
				sum := left0*right0 + left1*right1
				binary.LittleEndian.PutUint32(result[offset:], uint32(sum))
			}
		case x86asm.PMADDUBSW:
			for offset := 0; offset < 16; offset += 2 {
				sum := int64(left[offset])*int64(int8(right[offset])) + int64(left[offset+1])*int64(int8(right[offset+1]))
				binary.LittleEndian.PutUint16(result[offset:], uint16(int16(clampSigned64(sum, -32768, 32767))))
			}
		case x86asm.PMULHRSW:
			for offset := 0; offset < 16; offset += 2 {
				leftValue := int64(int16(binary.LittleEndian.Uint16(left[offset:])))
				rightValue := int64(int16(binary.LittleEndian.Uint16(right[offset:])))
				product := leftValue * rightValue
				rounded := (product + 0x4000) >> 15
				binary.LittleEndian.PutUint16(result[offset:], uint16(int16(clampSigned64(rounded, -32768, 32767))))
			}
		case x86asm.PHADDW, x86asm.PHADDSW, x86asm.PHADDD,
			x86asm.PHSUBW, x86asm.PHSUBSW, x86asm.PHSUBD:
			subtract := op == x86asm.PHSUBW || op == x86asm.PHSUBSW || op == x86asm.PHSUBD
			saturate := op == x86asm.PHADDSW || op == x86asm.PHSUBSW
			if op == x86asm.PHADDD || op == x86asm.PHSUBD {
				for lane := 0; lane < 4; lane++ {
					vector := left
					pair := lane
					if lane >= 2 {
						vector = right
						pair = lane - 2
					}
					base := pair * 8
					first := int64(int32(binary.LittleEndian.Uint32(vector[base:])))
					second := int64(int32(binary.LittleEndian.Uint32(vector[base+4:])))
					value := first + second
					if subtract {
						value = first - second
					}
					binary.LittleEndian.PutUint32(result[lane*4:], uint32(int32(value)))
				}
			} else {
				for lane := 0; lane < 8; lane++ {
					vector := left
					pair := lane
					if lane >= 4 {
						vector = right
						pair = lane - 4
					}
					base := pair * 4
					first := int64(int16(binary.LittleEndian.Uint16(vector[base:])))
					second := int64(int16(binary.LittleEndian.Uint16(vector[base+2:])))
					value := first + second
					if subtract {
						value = first - second
					}
					if saturate {
						value = clampSigned64(value, -32768, 32767)
					}
					binary.LittleEndian.PutUint16(result[lane*2:], uint16(int16(value)))
				}
			}
		case x86asm.PSADBW:

			for _, base := range []int{0, 8} {
				var sum uint16
				for i := 0; i < 8; i++ {
					l := int(left[base+i])
					r := int(right[base+i])
					if l >= r {
						sum += uint16(l - r)
					} else {
						sum += uint16(r - l)
					}
				}
				binary.LittleEndian.PutUint16(result[base:], sum)
			}
		case x86asm.PACKSSWB:
			for i := 0; i < 8; i++ {
				value := int64(int16(binary.LittleEndian.Uint16(left[i*2:])))
				result[i] = byte(int8(clampSigned64(value, -128, 127)))
				value = int64(int16(binary.LittleEndian.Uint16(right[i*2:])))
				result[i+8] = byte(int8(clampSigned64(value, -128, 127)))
			}
		case x86asm.PACKSSDW:
			for i := 0; i < 4; i++ {
				value := int64(int32(binary.LittleEndian.Uint32(left[i*4:])))
				binary.LittleEndian.PutUint16(result[i*2:], uint16(int16(clampSigned64(value, -32768, 32767))))
				value = int64(int32(binary.LittleEndian.Uint32(right[i*4:])))
				binary.LittleEndian.PutUint16(result[(i+4)*2:], uint16(int16(clampSigned64(value, -32768, 32767))))
			}
		case x86asm.PACKUSWB:
			for i := 0; i < 8; i++ {
				value := int64(int16(binary.LittleEndian.Uint16(left[i*2:])))
				result[i] = byte(clampSigned64(value, 0, 255))
				value = int64(int16(binary.LittleEndian.Uint16(right[i*2:])))
				result[i+8] = byte(clampSigned64(value, 0, 255))
			}
		case x86asm.PADDUSB, x86asm.PSUBUSB:
			for i := 0; i < 16; i++ {
				if op == x86asm.PADDUSB {
					result[i] = byte(clampSigned64(int64(left[i])+int64(right[i]), 0, 255))
				} else {
					result[i] = byte(clampSigned64(int64(left[i])-int64(right[i]), 0, 255))
				}
			}
		case x86asm.PADDSB, x86asm.PSUBSB:
			for i := 0; i < 16; i++ {
				leftValue := int64(int8(left[i]))
				rightValue := int64(int8(right[i]))
				if op == x86asm.PADDSB {
					result[i] = byte(int8(clampSigned64(leftValue+rightValue, -128, 127)))
				} else {
					result[i] = byte(int8(clampSigned64(leftValue-rightValue, -128, 127)))
				}
			}
		case x86asm.PADDSW, x86asm.PSUBSW:
			for offset := 0; offset < 16; offset += 2 {
				leftValue := int64(int16(binary.LittleEndian.Uint16(left[offset:])))
				rightValue := int64(int16(binary.LittleEndian.Uint16(right[offset:])))
				var value int64
				if op == x86asm.PADDSW {
					value = leftValue + rightValue
				} else {
					value = leftValue - rightValue
				}
				binary.LittleEndian.PutUint16(result[offset:], uint16(int16(clampSigned64(value, -32768, 32767))))
			}
		case x86asm.PADDUSW, x86asm.PSUBUSW:
			for offset := 0; offset < 16; offset += 2 {
				l := int64(binary.LittleEndian.Uint16(left[offset:]))
				r := int64(binary.LittleEndian.Uint16(right[offset:]))
				if op == x86asm.PADDUSW {
					binary.LittleEndian.PutUint16(result[offset:], uint16(clampSigned64(l+r, 0, 65535)))
				} else {
					binary.LittleEndian.PutUint16(result[offset:], uint16(clampSigned64(l-r, 0, 65535)))
				}
			}
		default:
			return Flow64Stop, ErrUnsupported64
		}
		if err := writeVector64(state, dst, next, result); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func clampSigned64(value, min, max int64) int64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
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
		if op == x86asm.PXOR || op == x86asm.PAND || op == x86asm.POR || op == x86asm.PANDN {
			for i := range left {
				switch op {
				case x86asm.PXOR:
					left[i] ^= right[i]
				case x86asm.PAND:
					left[i] &= right[i]
				case x86asm.POR:
					left[i] |= right[i]
				case x86asm.PANDN:
					left[i] = (^left[i]) & right[i]
				}
			}
		} else {
			lane, ok := sseLaneWidth64(op)
			if !ok {
				return Flow64Stop, ErrUnsupported64
			}
			for offset := 0; offset < len(left); offset += lane {
				var l, r uint64
				for i := 0; i < lane; i++ {
					l |= uint64(left[offset+i]) << (8 * i)
					r |= uint64(right[offset+i]) << (8 * i)
				}
				var result uint64
				switch op {
				case x86asm.PADDB, x86asm.PADDW, x86asm.PADDD, x86asm.PADDQ:
					result = l + r
				case x86asm.PSUBB, x86asm.PSUBW, x86asm.PSUBD, x86asm.PSUBQ:
					result = l - r
				case x86asm.PCMPEQB, x86asm.PCMPEQW, x86asm.PCMPEQD, x86asm.PCMPEQQ:
					if l == r {
						result = (uint64(1) << (8 * lane)) - 1
					}
				case x86asm.PCMPGTB, x86asm.PCMPGTW, x86asm.PCMPGTD:
					if signExtendLane64(l, lane) > signExtendLane64(r, lane) {
						result = (uint64(1) << (8 * lane)) - 1
					}
				case x86asm.PAVGB, x86asm.PAVGW:

					result = (l + r + 1) / 2
				case x86asm.PMINUB, x86asm.PMAXUB, x86asm.PMINUW, x86asm.PMAXUW,
					x86asm.PMINUD, x86asm.PMAXUD:
					if op == x86asm.PMINUB || op == x86asm.PMINUW || op == x86asm.PMINUD {
						if l < r {
							result = l
						} else {
							result = r
						}
					} else if l > r {
						result = l
					} else {
						result = r
					}
				case x86asm.PMINSB, x86asm.PMAXSB, x86asm.PMINSW, x86asm.PMAXSW,
					x86asm.PMINSD, x86asm.PMAXSD:
					leftSigned := signExtendLane64(l, lane)
					rightSigned := signExtendLane64(r, lane)
					if op == x86asm.PMINSB || op == x86asm.PMINSW || op == x86asm.PMINSD {
						if leftSigned < rightSigned {
							result = l
						} else {
							result = r
						}
					} else if leftSigned > rightSigned {
						result = l
					} else {
						result = r
					}
				}

				for i := 0; i < lane; i++ {
					left[offset+i] = byte(result >> (8 * i))
				}
			}
		}
		if err := writeVector64(state, dst, next, left); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeExtend64(address uint64, size uint8, signed bool, dst, src operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		value, err := readOperand64(state, src, next)
		if err != nil {
			return Flow64Stop, err
		}
		value &= mask64Width(src.Width)
		if signed {
			sign := uint64(1) << (uint(src.Width)*8 - 1)
			if value&sign != 0 {
				value |= ^mask64Width(src.Width)
			}
		}
		if err := writeOperand64(state, dst, next, value); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func addWithCarry64(left, right, carryIn uint64, width uint8) (uint64, bool) {
	mask := mask64Width(width)
	left &= mask
	right &= mask
	total := left + right + carryIn
	return total & mask, total > mask
}

func subWithBorrow64(left, right, borrowIn uint64, width uint8) (uint64, bool) {
	mask := mask64Width(width)
	left &= mask
	right &= mask
	subtrahend := right + borrowIn
	return (left - subtrahend) & mask, left < subtrahend
}

func makeCarryBinary64(address uint64, size uint8, op x86asm.Op, dst, src operand64) microOp64 {
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
		carryIn := uint64(0)
		if state.Flag(Flag64CF) {
			carryIn = 1
		}
		var result uint64
		var carry bool
		var overflow bool
		if op == x86asm.ADC {
			result, carry = addWithCarry64(left, right, carryIn, dst.Width)
			sign := uint64(1) << (uint(dst.Width)*8 - 1)
			overflow = ((^(left ^ right)) & (left ^ result) & sign) != 0
		} else {
			result, carry = subWithBorrow64(left, right, carryIn, dst.Width)
			sign := uint64(1) << (uint(dst.Width)*8 - 1)
			overflow = ((left ^ right) & (left ^ result) & sign) != 0
		}
		state.SetLazyArithmeticWidth(left, right, result, carry, overflow, true, dst.Width)
		if err := writeOperand64(state, dst, next, result); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeLAHFSAHF64(address uint64, size uint8, op x86asm.Op) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		if op == x86asm.LAHF {
			state.CollapseFlags()
			var ah uint64 = 1 << 1
			for _, pair := range []struct {
				flag uint64
				bit  uint
			}{{Flag64CF, 0}, {Flag64PF, 2}, {Flag64AF, 4}, {Flag64ZF, 6}, {Flag64SF, 7}} {
				if state.RFLAGS&pair.flag != 0 {
					ah |= uint64(1) << pair.bit
				}
			}
			state.Set(RAX, (state.Get(RAX)&^uint64(0xff00))|(ah<<8))
		} else {
			state.CollapseFlags()
			ah := (state.Get(RAX) >> 8) & 0xff
			var flags uint64
			for _, pair := range []struct {
				mask uint64
				bit  uint
			}{{Flag64CF, 0}, {Flag64PF, 2}, {Flag64AF, 4}, {Flag64ZF, 6}, {Flag64SF, 7}} {
				if ah&(uint64(1)<<pair.bit) != 0 {
					flags |= pair.mask
				}
			}
			state.RFLAGS = (state.RFLAGS &^ (Flag64CF | Flag64PF | Flag64AF | Flag64ZF | Flag64SF)) | flags | Flag64IF
			state.ExpandFlags()
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
			var carry bool
			result, carry = addWithCarry64(left, right, 0, dst.Width)
			sign := uint64(1) << (uint(dst.Width)*8 - 1)
			state.SetLazyArithmeticWidth(left, right, result, carry, ((^(left ^ right))&(left^result)&sign) != 0, true, dst.Width)
		case x86asm.SUB, x86asm.CMP:
			var borrow bool
			result, borrow = subWithBorrow64(left, right, 0, dst.Width)
			sign := uint64(1) << (uint(dst.Width)*8 - 1)
			result &= mask
			subtraction = true
			state.SetLazyArithmeticWidth(left, right, result, borrow, (((left ^ right) & (left ^ result) & sign) != 0), true, dst.Width)
		case x86asm.XOR:
			result = (left ^ right) & mask
			state.SetLazyArithmeticWidth(left, right, result, false, false, false, dst.Width)
		case x86asm.AND, x86asm.TEST:
			result = (left & right) & mask
			state.SetLazyArithmeticWidth(left, right, result, false, false, false, dst.Width)
		case x86asm.OR:
			result = (left | right) & mask
			state.SetLazyArithmeticWidth(left, right, result, false, false, false, dst.Width)
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
		state.SetLazyArithmeticWidth(old, srcValue, result, carry != 0, ((^(old ^ srcValue))&(old^result)&(mask^(mask>>1))) != 0, true, src.Width)
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
		state.SetLazyArithmeticWidth(accValue, observed, result, borrow != 0, (((accValue ^ observed) & (accValue ^ result) & (mask ^ (mask >> 1))) != 0), true, src.Width)
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
			state.SetLazyArithmeticWidth(value, 1, result, op == x86asm.DEC || op == x86asm.NEG, false, true, dst.Width)
		}
		if err := writeOperand64(state, dst, next, result); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeMOVScalar64(address uint64, size, scalarWidth uint8, destination, source operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		var value uint64
		if source.Kind == operand64XMM {
			if source.XMM >= uint8(len(state.XMM)) {
				return Flow64Stop, ErrUnsupported64
			}
			for i := uint8(0); i < scalarWidth; i++ {
				value |= uint64(state.XMM[source.XMM][i]) << (8 * i)
			}
		} else {
			var err error
			value, err = readOperand64(state, source, next)
			if err != nil {
				return Flow64Stop, err
			}
		}

		if destination.Kind == operand64XMM {
			if destination.XMM >= uint8(len(state.XMM)) {
				return Flow64Stop, ErrUnsupported64
			}
			state.XMM[destination.XMM] = [16]byte{}
			for i := uint8(0); i < scalarWidth; i++ {
				state.XMM[destination.XMM][i] = byte(value >> (8 * i))
			}
		} else if err := writeOperand64(state, destination, next, value); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeBSWAP64(address uint64, size uint8, destination operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		value := readReg64(state, destination)
		switch destination.Width {
		case 2:
			value = uint64(bits.ReverseBytes16(uint16(value)))
		case 4:
			value = uint64(bits.ReverseBytes32(uint32(value)))
		case 8:
			value = bits.ReverseBytes64(value)
		default:
			return Flow64Stop, ErrUnsupported64
		}
		writeReg64(state, destination, value)
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeCountZeros64(address uint64, size uint8, op x86asm.Op, destination, source operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		value, err := readOperand64(state, source, next)
		if err != nil {
			return Flow64Stop, err
		}
		var count uint
		switch destination.Width {
		case 8:
			if op == x86asm.LZCNT {
				count = uint(bits.LeadingZeros64(value))
			} else {
				count = uint(bits.TrailingZeros64(value))
			}
		case 4:
			value = uint64(uint32(value))
			if op == x86asm.LZCNT {
				count = uint(bits.LeadingZeros32(uint32(value)))
			} else {
				count = uint(bits.TrailingZeros32(uint32(value)))
			}
		case 2:
			value = uint64(uint16(value))
			if op == x86asm.LZCNT {
				count = uint(bits.LeadingZeros16(uint16(value)))
			} else {
				count = uint(bits.TrailingZeros16(uint16(value)))
			}
		default:
			return Flow64Stop, ErrUnsupported64
		}
		writeReg64(state, destination, uint64(count))
		state.CollapseFlags()
		const arithmetic = Flag64CF | Flag64PF | Flag64AF | Flag64ZF | Flag64SF | Flag64OF
		state.RFLAGS &^= arithmetic
		if value == 0 {
			state.RFLAGS |= Flag64CF
		}
		if count == 0 {
			state.RFLAGS |= Flag64ZF
		}
		state.ExpandFlags()
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeBitScan64(address uint64, size uint8, op x86asm.Op, destination, source operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		value, err := readOperand64(state, source, next)
		if err != nil {
			return Flow64Stop, err
		}

		if value == 0 {
			state.CollapseFlags()
			state.RFLAGS |= Flag64ZF
			state.ExpandFlags()
			state.RIP = next
			return Flow64Continue, nil
		}
		var index uint
		if op == x86asm.BSF {
			if destination.Width == 8 {
				index = uint(bits.TrailingZeros64(value))
			} else {
				index = uint(bits.TrailingZeros32(uint32(value)))
			}
		} else {
			if destination.Width == 8 {
				index = uint(63 - bits.LeadingZeros64(value))
			} else {
				index = uint(31 - bits.LeadingZeros32(uint32(value)))
			}
		}
		writeReg64(state, destination, uint64(index))

		state.CollapseFlags()
		state.RFLAGS &^= Flag64ZF
		state.ExpandFlags()
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makePOPCNT64(address uint64, size uint8, destination, source operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		value, err := readOperand64(state, source, next)
		if err != nil {
			return Flow64Stop, err
		}
		var count uint64
		if destination.Width == 8 {
			count = uint64(bits.OnesCount64(value))
		} else if destination.Width == 4 {
			count = uint64(bits.OnesCount32(uint32(value)))
		} else {
			count = uint64(bits.OnesCount16(uint16(value)))
		}
		writeReg64(state, destination, count)
		state.CollapseFlags()
		state.RFLAGS &^= Flag64CF | Flag64PF | Flag64AF | Flag64SF | Flag64OF | Flag64ZF
		if count == 0 {
			state.RFLAGS |= Flag64ZF
		}
		state.ExpandFlags()
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func bitTestTarget64(state *MachineState64, destination, index operand64, next uint64) (operand64, uint64, error) {
	bitWidth := uint64(destination.Width) * 8
	if bitWidth == 0 {
		return operand64{}, 0, ErrUnsupported64
	}
	bitIndex, err := readOperand64(state, index, next)
	if err != nil {
		return operand64{}, 0, err
	}
	if destination.Kind != operand64Mem {
		return destination, bitIndex & (bitWidth - 1), nil
	}
	element := int64(bitIndex) / int64(bitWidth)
	bit := bitIndex & (bitWidth - 1)
	target := destination
	target.Mem.Disp += element * int64(destination.Width)
	return target, bit, nil
}

func makeBitTest64(address uint64, size uint8, op x86asm.Op, destination, index operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		target, bitIndex, err := bitTestTarget64(state, destination, index, next)
		if err != nil {
			return Flow64Stop, err
		}
		value, err := readOperand64(state, target, next)
		if err != nil {
			return Flow64Stop, err
		}
		bit := uint64(1) << bitIndex
		oldSet := value&bit != 0
		state.CollapseFlags()
		if oldSet {
			state.RFLAGS |= Flag64CF
		} else {
			state.RFLAGS &^= Flag64CF
		}
		state.ExpandFlags()
		switch op {
		case x86asm.BTS:
			value |= bit
		case x86asm.BTR:
			value &^= bit
		case x86asm.BTC:
			value ^= bit
		case x86asm.BT:
			return setBitTestResult64(state, next), nil
		default:
			return Flow64Stop, ErrUnsupported64
		}
		if err := writeOperand64(state, target, next, value); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func setBitTestResult64(state *MachineState64, next uint64) Flow64 {
	state.RIP = next
	return Flow64Continue
}

func makePushFlags64(address uint64, size uint8) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		state.CollapseFlags()
		state.Regs[RSP] -= 8
		if err := state.Memory.WriteUint64(Address64(state.Regs[RSP]), state.RFLAGS); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makePopFlags64(address uint64, size uint8) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		value, err := state.Memory.ReadUint64(Address64(state.Regs[RSP]))
		if err != nil {
			return Flow64Stop, err
		}
		state.Regs[RSP] += 8
		state.SetRFLAGS(value)
		state.RFLAGS |= Flag64IF
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

func decodeSETcc64(op x86asm.Op) (conditionCode64, bool) {
	switch op {
	case x86asm.SETO:
		return condition64O, true
	case x86asm.SETNO:
		return condition64O | 0x80, true
	case x86asm.SETB:
		return condition64B, true
	case x86asm.SETAE:
		return condition64B | 0x80, true
	case x86asm.SETE:
		return condition64E, true
	case x86asm.SETNE:
		return condition64E | 0x80, true
	case x86asm.SETBE:
		return condition64BE, true
	case x86asm.SETA:
		return condition64BE | 0x80, true
	case x86asm.SETS:
		return condition64S, true
	case x86asm.SETNS:
		return condition64S | 0x80, true
	case x86asm.SETP:
		return condition64P, true
	case x86asm.SETNP:
		return condition64P | 0x80, true
	case x86asm.SETL:
		return condition64L, true
	case x86asm.SETGE:
		return condition64L | 0x80, true
	case x86asm.SETLE:
		return condition64LE, true
	case x86asm.SETG:
		return condition64LE | 0x80, true
	default:
		return 0, false
	}
}

func decodeCMOVcc64(op x86asm.Op) (conditionCode64, bool) {
	switch op {
	case x86asm.CMOVO:
		return condition64O, true
	case x86asm.CMOVNO:
		return condition64O | 0x80, true
	case x86asm.CMOVB:
		return condition64B, true
	case x86asm.CMOVAE:
		return condition64B | 0x80, true
	case x86asm.CMOVE:
		return condition64E, true
	case x86asm.CMOVNE:
		return condition64E | 0x80, true
	case x86asm.CMOVBE:
		return condition64BE, true
	case x86asm.CMOVA:
		return condition64BE | 0x80, true
	case x86asm.CMOVS:
		return condition64S, true
	case x86asm.CMOVNS:
		return condition64S | 0x80, true
	case x86asm.CMOVP:
		return condition64P, true
	case x86asm.CMOVNP:
		return condition64P | 0x80, true
	case x86asm.CMOVL:
		return condition64L, true
	case x86asm.CMOVGE:
		return condition64L | 0x80, true
	case x86asm.CMOVLE:
		return condition64LE, true
	case x86asm.CMOVG:
		return condition64LE | 0x80, true
	default:
		return 0, false
	}
}

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

func makeSETcc64(address uint64, size uint8, condition conditionCode64, destination operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		value := uint64(0)
		if conditionValue64(state, condition) {
			value = 1
		}
		if err := writeOperand64(state, destination, next, value); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeCMOVcc64(address uint64, size uint8, condition conditionCode64, destination, source operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		if conditionValue64(state, condition) {
			value, err := readOperand64(state, source, next)
			if err != nil {
				return Flow64Stop, err
			}
			if err := writeOperand64(state, destination, next, value); err != nil {
				return Flow64Stop, err
			}
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
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

func stringWidth64(op x86asm.Op) (uint8, bool) {
	switch op {
	case x86asm.MOVSB, x86asm.STOSB, x86asm.LODSB, x86asm.CMPSB, x86asm.SCASB:
		return 1, true
	case x86asm.MOVSW, x86asm.STOSW, x86asm.LODSW, x86asm.CMPSW, x86asm.SCASW:
		return 2, true
	case x86asm.MOVSD, x86asm.STOSD, x86asm.LODSD, x86asm.CMPSD, x86asm.SCASD:
		return 4, true
	case x86asm.MOVSQ, x86asm.STOSQ, x86asm.LODSQ, x86asm.CMPSQ, x86asm.SCASQ:
		return 8, true
	default:
		return 0, false
	}
}

func stringRepeatMode64(prefixes x86asm.Prefixes) uint8 {
	for _, prefix := range prefixes {
		if prefix == 0 {
			break
		}
		switch prefix & 0xff {
		case x86asm.PrefixREP & 0xff:
			return 1 // REP/REPE.
		case x86asm.PrefixREPN & 0xff:
			return 2 // REPNE.
		}
	}
	return 0
}

func stringIndex64(state *MachineState64, reg Reg64, addressSize uint8) uint64 {
	value := state.Get(reg)
	if addressSize == 32 {
		return uint64(uint32(value))
	}
	return value
}

func advanceStringIndex64(state *MachineState64, reg Reg64, addressSize, width uint8, decrement bool) {
	value := stringIndex64(state, reg, addressSize)
	if decrement {
		value -= uint64(width)
	} else {
		value += uint64(width)
	}
	if addressSize == 32 {
		state.Set(reg, uint64(uint32(value)))
	} else {
		state.Set(reg, value)
	}
}

func stringCount64(state *MachineState64, addressSize uint8) uint64 {
	if addressSize == 32 {
		return uint64(uint32(state.Get(RCX)))
	}
	return state.Get(RCX)
}

func decrementStringCount64(state *MachineState64, addressSize uint8, count uint64) {
	if addressSize == 32 {
		state.Set(RCX, uint64(uint32(count-1)))
	} else {
		state.Set(RCX, count-1)
	}
}

func readString64(state *MachineState64, address uint64, width uint8) (uint64, error) {
	if width == 0 || width > 8 {
		return 0, ErrUnsupportedAddressing
	}
	var raw [8]byte
	if err := state.Memory.Read(Address64(address), raw[:width]); err != nil {
		return 0, err
	}
	var value uint64
	for i := uint8(0); i < width; i++ {
		value |= uint64(raw[i]) << (8 * i)
	}
	return value, nil
}

func writeString64(state *MachineState64, address uint64, width uint8, value uint64) error {
	if width == 0 || width > 8 {
		return ErrUnsupportedAddressing
	}
	var raw [8]byte
	for i := uint8(0); i < width; i++ {
		raw[i] = byte(value >> (8 * i))
	}
	return state.Memory.Write(Address64(address), raw[:width])
}

func setStringCompareFlags64(state *MachineState64, left, right uint64, width uint8) {
	mask := mask64Width(width)
	left &= mask
	right &= mask
	result := (left - right) & mask
	sign := uint64(1) << (uint(width)*8 - 1)
	overflow := ((left ^ right) & (left ^ result) & sign) != 0
	state.SetLazyArithmeticWidth(left, right, result, left < right, overflow, true, width)
}

func makeString64(address uint64, size uint8, op x86asm.Op, width, addressSize, repeat uint8) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		if addressSize != 32 {
			addressSize = 64
		}
		count := uint64(1)
		if repeat != 0 {
			count = stringCount64(state, addressSize)
			if count == 0 {
				state.RIP = next
				return Flow64Continue, nil
			}
		}
		decrement := state.RFLAGS&Flag64DF != 0
		compare := op == x86asm.CMPSB || op == x86asm.CMPSW || op == x86asm.CMPSD || op == x86asm.CMPSQ ||
			op == x86asm.SCASB || op == x86asm.SCASW || op == x86asm.SCASD || op == x86asm.SCASQ
		for count != 0 {
			switch op {
			case x86asm.MOVSB, x86asm.MOVSW, x86asm.MOVSD, x86asm.MOVSQ:
				value, err := readString64(state, stringIndex64(state, RSI, addressSize), width)
				if err != nil {
					return Flow64Stop, err
				}
				if err := writeString64(state, stringIndex64(state, RDI, addressSize), width, value); err != nil {
					return Flow64Stop, err
				}
				advanceStringIndex64(state, RSI, addressSize, width, decrement)
				advanceStringIndex64(state, RDI, addressSize, width, decrement)
			case x86asm.STOSB, x86asm.STOSW, x86asm.STOSD, x86asm.STOSQ:
				value := readReg64(state, operand64{Kind: operand64Reg, Reg: RAX, Width: width})
				if err := writeString64(state, stringIndex64(state, RDI, addressSize), width, value); err != nil {
					return Flow64Stop, err
				}
				advanceStringIndex64(state, RDI, addressSize, width, decrement)
			case x86asm.LODSB, x86asm.LODSW, x86asm.LODSD, x86asm.LODSQ:
				value, err := readString64(state, stringIndex64(state, RSI, addressSize), width)
				if err != nil {
					return Flow64Stop, err
				}
				writeReg64(state, operand64{Kind: operand64Reg, Reg: RAX, Width: width}, value)
				advanceStringIndex64(state, RSI, addressSize, width, decrement)
			case x86asm.CMPSB, x86asm.CMPSW, x86asm.CMPSD, x86asm.CMPSQ:
				left, err := readString64(state, stringIndex64(state, RSI, addressSize), width)
				if err != nil {
					return Flow64Stop, err
				}
				right, err := readString64(state, stringIndex64(state, RDI, addressSize), width)
				if err != nil {
					return Flow64Stop, err
				}
				setStringCompareFlags64(state, left, right, width)
				advanceStringIndex64(state, RSI, addressSize, width, decrement)
				advanceStringIndex64(state, RDI, addressSize, width, decrement)
			case x86asm.SCASB, x86asm.SCASW, x86asm.SCASD, x86asm.SCASQ:
				left := readReg64(state, operand64{Kind: operand64Reg, Reg: RAX, Width: width})
				right, err := readString64(state, stringIndex64(state, RDI, addressSize), width)
				if err != nil {
					return Flow64Stop, err
				}
				setStringCompareFlags64(state, left, right, width)
				advanceStringIndex64(state, RDI, addressSize, width, decrement)
			default:
				return Flow64Stop, ErrUnsupported64
			}
			if repeat == 0 {
				break
			}
			decrementStringCount64(state, addressSize, count)
			count--
			if count == 0 {
				break
			}
			if compare {
				if repeat == 1 && !state.Flag(Flag64ZF) {
					break
				}
				if repeat == 2 && state.Flag(Flag64ZF) {
					break
				}
			}
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func byteSwap64Width(value uint64, width uint8) uint64 {
	switch width {
	case 2:
		return uint64(bits.ReverseBytes16(uint16(value)))
	case 4:
		return uint64(bits.ReverseBytes32(uint32(value)))
	case 8:
		return bits.ReverseBytes64(value)
	default:
		return value
	}
}

func makeMOVBE64(address uint64, size uint8, dst, src operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		value, err := readOperand64(state, src, next)
		if err != nil {
			return Flow64Stop, err
		}
		value = byteSwap64Width(value, src.Width)
		if err := writeOperand64(state, dst, next, value); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeConvertAccumulator64(address uint64, size uint8, op x86asm.Op) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		switch op {
		case x86asm.CBW:
			value := signExtend64Width(state.Get(RAX), 1)
			writeReg64(state, operand64{Kind: operand64Reg, Reg: RAX, Width: 2}, uint64(value))
		case x86asm.CWDE:
			value := signExtend64Width(state.Get(RAX), 2)
			writeReg64(state, operand64{Kind: operand64Reg, Reg: RAX, Width: 4}, uint64(value))
		case x86asm.CDQ:
			if state.Get(RAX)&0x80000000 != 0 {
				writeReg64(state, operand64{Kind: operand64Reg, Reg: RDX, Width: 4}, 0xffffffff)
			} else {
				writeReg64(state, operand64{Kind: operand64Reg, Reg: RDX, Width: 4}, 0)
			}
		case x86asm.CDQE:
			value := signExtend64Width(state.Get(RAX), 4)
			writeReg64(state, operand64{Kind: operand64Reg, Reg: RAX, Width: 8}, uint64(value))
		case x86asm.CWD:
			if state.Get(RAX)&0x8000 != 0 {
				writeReg64(state, operand64{Kind: operand64Reg, Reg: RDX, Width: 2}, 0xffff)
			} else {
				writeReg64(state, operand64{Kind: operand64Reg, Reg: RDX, Width: 2}, 0)
			}
		case x86asm.CQO:
			if state.Get(RAX)&(uint64(1)<<63) != 0 {
				state.Set(RDX, ^uint64(0))
			} else {
				state.Set(RDX, 0)
			}
		default:
			return Flow64Stop, ErrUnsupported64
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeLeave64(address uint64, size, stackWidth uint8) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		stackPointer := state.Get(RBP)
		if stackWidth == 4 {
			stackPointer = uint64(uint32(stackPointer))
		}
		value, err := readString64(state, stackPointer, stackWidth)
		if err != nil {
			return Flow64Stop, err
		}
		if stackWidth == 4 {
			state.Set(RSP, uint64(uint32(stackPointer+4)))
		} else {
			state.Set(RSP, stackPointer+8)
		}
		writeReg64(state, operand64{Kind: operand64Reg, Reg: RBP, Width: stackWidth}, value)
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeFence64(address uint64, size uint8) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		// Guest memory operations are sequenced by the interpreter/JIT loop. The
		// fence has no additional observable effect in this single-guest model.
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeFSBase64(address uint64, size uint8, op x86asm.Op, operand operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		if op == x86asm.RDFSBASE || op == x86asm.RDGSBASE {
			value := state.FSBase
			if op == x86asm.RDGSBASE {
				value = state.GSBase
			}
			writeReg64(state, operand, value)
		} else {
			value, err := readOperand64(state, operand, next)
			if err != nil {
				return Flow64Stop, err
			}
			if operand.Width == 4 {
				value = uint64(uint32(value))
			}
			if op == x86asm.WRFSBASE {
				state.FSBase = value
			} else {
				state.GSBase = value
			}
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeXCR64(address uint64, size uint8, op x86asm.Op) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		if state.Get(RCX) != 0 {
			state.RIP = next
			state.TrapNo = Trap64GeneralFault
			return Flow64Interrupt, nil
		}
		if op == x86asm.XGETBV {
			value := state.XCR0
			state.Set(RAX, uint64(uint32(value)))
			state.Set(RDX, uint64(uint32(value>>32)))
		} else {
			value := (state.Get(RDX) << 32) | (state.Get(RAX) & 0xffffffff)
			if value&1 == 0 || value&^uint64(0x3) != 0 {
				state.RIP = next
				state.TrapNo = Trap64GeneralFault
				return Flow64Interrupt, nil
			}
			state.XCR0 = value
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeCmpxchgB64(address uint64, size uint8, op x86asm.Op, destination operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		memoryAddress, err := effectiveAddress64(state, destination.Mem, next)
		if err != nil {
			return Flow64Stop, err
		}
		var oldLo, oldHi uint64
		var exchanged bool
		if op == x86asm.CMPXCHG8B {
			expected := uint64(uint32(state.Get(RAX))) | uint64(uint32(state.Get(RDX)))<<32
			replacement := uint64(uint32(state.Get(RBX))) | uint64(uint32(state.Get(RCX)))<<32
			old, atomicErr := state.Memory.AtomicCompareExchange(memoryAddress, 8, expected, replacement)
			if atomicErr != nil {
				return Flow64Stop, atomicErr
			}
			oldLo, oldHi = uint64(uint32(old)), uint64(uint32(old>>32))
			exchanged = old == expected
		} else {
			oldLo, oldHi, exchanged, err = state.Memory.AtomicCompareExchange128(memoryAddress, state.Get(RAX), state.Get(RDX), state.Get(RBX), state.Get(RCX))
			if err != nil {
				return Flow64Stop, err
			}
		}
		if exchanged {
			state.RFLAGS |= Flag64ZF
		} else {
			state.RFLAGS &^= Flag64ZF
			state.Set(RAX, oldLo)
			state.Set(RDX, oldHi)
		}
		state.ExpandFlags()
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func sseShiftLane64(op x86asm.Op) (lane int, right bool, arithmetic bool, ok bool) {
	switch op {
	case x86asm.PSLLW:
		return 2, false, false, true
	case x86asm.PSLLD:
		return 4, false, false, true
	case x86asm.PSLLQ:
		return 8, false, false, true
	case x86asm.PSRLW:
		return 2, true, false, true
	case x86asm.PSRLD:
		return 4, true, false, true
	case x86asm.PSRLQ:
		return 8, true, false, true
	case x86asm.PSRAW:
		return 2, true, true, true
	case x86asm.PSRAD:
		return 4, true, true, true
	default:
		return 0, false, false, false
	}
}

func makeSSEShift64(address uint64, size uint8, op x86asm.Op, destination operand64, immediate uint8) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		value, err := readVector64(state, destination, next)
		if err != nil {
			return Flow64Stop, err
		}
		if op == x86asm.PSLLDQ || op == x86asm.PSRLDQ {
			count := int(immediate)
			var result [16]byte
			if count < len(value) {
				if op == x86asm.PSLLDQ {
					copy(result[count:], value[:len(value)-count])
				} else {
					copy(result[:len(value)-count], value[count:])
				}
			}
			if err := writeVector64(state, destination, next, result); err != nil {
				return Flow64Stop, err
			}
			state.RIP = next
			return Flow64Continue, nil
		}
		lane, right, arithmetic, ok := sseShiftLane64(op)
		if !ok {
			return Flow64Stop, ErrUnsupported64
		}
		count := int(immediate)
		bits := lane * 8
		mask := (uint64(1) << bits) - 1
		for offset := 0; offset < len(value); offset += lane {
			var source uint64
			for i := 0; i < lane; i++ {
				source |= uint64(value[offset+i]) << (8 * i)
			}
			var result uint64
			if count >= bits {
				if arithmetic && source&(uint64(1)<<(bits-1)) != 0 {
					result = mask
				}
			} else if right {
				if arithmetic {
					result = uint64(int64(signExtendLane64(source, lane)) >> count)
				} else {
					result = source >> count
				}
			} else {
				result = source << count
			}
			result &= mask
			for i := 0; i < lane; i++ {
				value[offset+i] = byte(result >> (8 * i))
			}
		}
		if err := writeVector64(state, destination, next, value); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func signExtendLane64(value uint64, lane int) int64 {
	bits := uint(lane * 8)
	value &= (uint64(1) << bits) - 1
	sign := uint64(1) << (bits - 1)
	if value&sign != 0 {
		value |= ^((uint64(1) << bits) - 1)
	}
	return int64(value)
}

func makeSSEShuffleD64(address uint64, size uint8, destination, source operand64, immediate uint8) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		input, err := readVector64(state, source, next)
		if err != nil {
			return Flow64Stop, err
		}
		var output [16]byte
		for lane := 0; lane < 4; lane++ {
			selected := int((immediate >> (2 * lane)) & 3)
			copy(output[lane*4:lane*4+4], input[selected*4:selected*4+4])
		}
		if err := writeVector64(state, destination, next, output); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeSSEMovemask64(address uint64, size uint8, destination, source operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		value, err := readVector64(state, source, next)
		if err != nil {
			return Flow64Stop, err
		}
		var mask uint64
		for i, byteValue := range value {
			mask |= uint64(byteValue>>7) << i
		}
		if err := writeOperand64(state, destination, next, mask); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func sseUnpackLane64(op x86asm.Op) (lane int, high bool, ok bool) {
	switch op {
	case x86asm.PUNPCKLBW:
		return 1, false, true
	case x86asm.PUNPCKHBW:
		return 1, true, true
	case x86asm.PUNPCKLWD:
		return 2, false, true
	case x86asm.PUNPCKHWD:
		return 2, true, true
	case x86asm.PUNPCKLDQ:
		return 4, false, true
	case x86asm.PUNPCKHDQ:
		return 4, true, true
	case x86asm.PUNPCKLQDQ:
		return 8, false, true
	case x86asm.PUNPCKHQDQ:
		return 8, true, true
	default:
		return 0, false, false
	}
}

func makeSSEUnpack64(address uint64, size uint8, op x86asm.Op, destination, source operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		left, err := readVector64(state, destination, next)
		if err != nil {
			return Flow64Stop, err
		}
		right, err := readVector64(state, source, next)
		if err != nil {
			return Flow64Stop, err
		}
		lane, high, ok := sseUnpackLane64(op)
		if !ok {
			return Flow64Stop, ErrUnsupported64
		}
		start := 0
		if high {
			start = 8 / lane
		}
		var result [16]byte
		for i := 0; i < 8/lane; i++ {
			leftOffset := (start + i) * lane
			rightOffset := (start + i) * lane
			outputOffset := i * 2 * lane
			copy(result[outputOffset:outputOffset+lane], left[leftOffset:leftOffset+lane])
			copy(result[outputOffset+lane:outputOffset+2*lane], right[rightOffset:rightOffset+lane])
		}
		if err := writeVector64(state, destination, next, result); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeSSEShuffleW64(address uint64, size uint8, op x86asm.Op, destination, source operand64, immediate uint8) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		input, err := readVector64(state, source, next)
		if err != nil {
			return Flow64Stop, err
		}
		output := input
		start := 0
		if op == x86asm.PSHUFHW {
			start = 4
		}
		for lane := 0; lane < 4; lane++ {
			selected := int((immediate >> (2 * lane)) & 3)
			copy(output[(start+lane)*2:(start+lane)*2+2], input[(start+selected)*2:(start+selected)*2+2])
		}
		if err := writeVector64(state, destination, next, output); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

func makeSSEShuffleBytes64(address uint64, size uint8, destination, source operand64) microOp64 {
	return microOp64{Address: address, Size: size, Run: func(state *MachineState64, next uint64) (Flow64, error) {
		input, err := readVector64(state, destination, next)
		if err != nil {
			return Flow64Stop, err
		}
		control, err := readVector64(state, source, next)
		if err != nil {
			return Flow64Stop, err
		}
		var output [16]byte
		for i, selector := range control {
			if selector&0x80 == 0 {
				output[i] = input[selector&0x0f]
			}
		}
		if err := writeVector64(state, destination, next, output); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	}}
}

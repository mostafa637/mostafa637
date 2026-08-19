package cpu

import (
	"errors"
	"fmt"
	"sync/atomic"

	"golang.org/x/arch/x86/x86asm"
)

// Trap values use the architectural exception numbers where one exists. The
// syscall and timer values are returned to the guest lifecycle instead of being
// swallowed by the JIT loop.
const (
	Trap64None          uint64 = 0
	Trap64Debug         uint64 = 1
	Trap64InvalidOpcode uint64 = 6
	Trap64GeneralFault  uint64 = 13
	Trap64PageFault     uint64 = 14
	Trap64Syscall       uint64 = 0x80
	Trap64Timer         uint64 = 0x100
	Trap64Exit          uint64 = 0x101
)

const (
	blockCache64Size = 1024
	default64Budget  = 1 << 10
)

// JIT64 owns the asbestos-like execution loop for one guest address space. It
// intentionally keeps a global translated-block cache and a per-run direct
// mapped cache, matching iSH's two-level lookup without exposing host pointers.
type JIT64 struct {
	Memory *Memory64
	Cache  *BlockCache64
	Budget uint64

	// OnSyscall64 is supplied by the guest kernel layer. Keeping it as a
	// callback avoids an import cycle between cpu and syscall packages.
	OnSyscall64 func(*MachineState64) (resume bool, err error)
	poked       uint32
}

func NewJIT64(memory *Memory64) *JIT64 {
	return &JIT64{Memory: memory, Cache: NewBlockCache64(memory), Budget: default64Budget}
}

func (j *JIT64) Poke() {
	if j != nil {
		atomic.StoreUint32(&j.poked, 1)
	}
}

func (j *JIT64) consumePoke(state *MachineState64) bool {
	if j != nil && atomic.SwapUint32(&j.poked, 0) != 0 {
		return true
	}
	return state != nil && atomic.SwapUint32(&state.Poked, 0) != 0
}

func blockCache64Hash(ip uint64) uint64 {
	return (ip ^ (ip >> 12)) % blockCache64Size
}

func (j *JIT64) blockAt(ip uint64, local *[blockCache64Size]*CompiledBlock64) (*CompiledBlock64, error) {
	if j == nil || j.Memory == nil || j.Cache == nil {
		return nil, ErrInvalid64Block
	}
	index := blockCache64Hash(ip)
	if block := local[index]; block != nil && block.Start == ip && block.Valid(j.Memory) {
		return block, nil
	}
	if block, ok := j.Cache.Get(ip); ok {
		local[index] = block
		return block, nil
	}
	block, err := CompileBlock64(j.Memory, Address64(ip), Page64Size)
	if err != nil {
		return nil, err
	}
	if err := j.Cache.Put(block); err != nil {
		return nil, err
	}
	local[index] = block
	return block, nil
}

func (j *JIT64) fault(state *MachineState64, err error) uint64 {
	if errors.Is(err, ErrUnmapped) || errors.Is(err, ErrProtection) || errors.Is(err, ErrRange) {
		state.TrapNo = Trap64PageFault
	} else if errors.Is(err, ErrUnsupported64) {
		state.TrapNo = Trap64InvalidOpcode
	} else {
		state.TrapNo = Trap64GeneralFault
	}
	return state.TrapNo
}

// RunToInterrupt executes translated blocks until a guest interrupt, a timer
// budget, a poke, or a memory/instruction fault occurs. The returned state is
// the guest state, not Go's call stack, so callers can suspend and resume it.
func (j *JIT64) RunToInterrupt(state *MachineState64) uint64 {
	if state == nil {
		return Trap64GeneralFault
	}
	if j == nil || j.Memory == nil {
		state.TrapNo = Trap64GeneralFault
		return state.TrapNo
	}
	state.Memory = j.Memory
	state.TrapNo = Trap64None
	state.FaultAt = 0
	state.FaultWrite = false
	budget := j.Budget
	if budget == 0 {
		budget = default64Budget
	}
	var local [blockCache64Size]*CompiledBlock64
	for executed := uint64(0); ; {
		if j.consumePoke(state) {
			state.TrapNo = Trap64Timer
			return state.TrapNo
		}
		if executed >= budget {
			state.TrapNo = Trap64Timer
			return state.TrapNo
		}
		block, err := j.blockAt(state.RIP, &local)
		if err != nil {
			// Compile failure may be caused by one instruction that the block
			// compiler does not yet lower. Decode and execute that instruction
			// through the conservative fallback path before declaring #UD.
			flow, fallbackErr := j.interpretOne64(state)
			if fallbackErr != nil {
				return j.fault(state, fallbackErr)
			}
			executed++
			state.InstructionCount++
			if flow == Flow64Interrupt {
				return state.TrapNo
			}
			continue
		}
		flow, runErr := block.Run(state)
		executed += uint64(len(block.Ops))
		state.InstructionCount += uint64(len(block.Ops))
		if runErr != nil {
			return j.fault(state, runErr)
		}
		if flow == Flow64Interrupt {
			if state.TrapNo == Trap64Syscall && j.OnSyscall64 != nil {
				resume, syscallErr := j.OnSyscall64(state)
				if syscallErr != nil {
					return j.fault(state, syscallErr)
				}
				if resume && !state.Halted {
					state.TrapNo = Trap64None
					continue
				}
				state.TrapNo = Trap64Exit
			}
			return state.TrapNo
		}
	}
}

func (j *JIT64) interpretOne64(state *MachineState64) (Flow64, error) {
	code := make([]byte, 15)
	if err := j.Memory.Read(Address64(state.RIP), code); err != nil {
		return Flow64Stop, err
	}
	inst, err := x86asm.Decode(code, 64)
	if err != nil || inst.Len == 0 {
		if err == nil {
			err = fmt.Errorf("zero-length instruction")
		}
		return Flow64Stop, fmt.Errorf("%w at %#x: %v", ErrUnsupported64, state.RIP, err)
	}
	op, _, err := compileInstruction64(inst, state.RIP)
	if err == nil {
		return op.Run(state, state.RIP+uint64(inst.Len))
	}
	return fallbackInstruction64(state, inst)
}

func fallbackInstruction64(state *MachineState64, inst x86asm.Inst) (Flow64, error) {
	next := state.RIP + uint64(inst.Len)
	switch inst.Op {
	case x86asm.PAUSE:
		state.RIP = next
		return Flow64Continue, nil
	case x86asm.XCHG:
		width := instructionWidth64(inst, inst.Args[0], inst.Args[1])
		left, leftErr := operand64FromArg(inst.Args[0], width)
		right, rightErr := operand64FromArg(inst.Args[1], width)
		if leftErr != nil || rightErr != nil {
			return Flow64Stop, fmt.Errorf("%w: XCHG operands", ErrUnsupported64)
		}
		leftValue, err := readOperand64(state, left, next)
		if err != nil {
			return Flow64Stop, err
		}
		rightValue, err := readOperand64(state, right, next)
		if err != nil {
			return Flow64Stop, err
		}
		if err := writeOperand64(state, left, next, rightValue); err != nil {
			return Flow64Stop, err
		}
		if err := writeOperand64(state, right, next, leftValue); err != nil {
			return Flow64Stop, err
		}
		state.RIP = next
		return Flow64Continue, nil
	case x86asm.RDTSC:
		value := state.Cycle
		state.Set(RAX, uint64(uint32(value)))
		state.Set(RDX, uint64(uint32(value>>32)))
		state.RIP = next
		return Flow64Continue, nil
	case x86asm.CPUID:
		// Keep the contract deterministic and side-effect free until the
		// host feature surface is modelled explicitly.
		leaf := state.Get(RAX)
		switch leaf {
		case 0:
			state.Set(RAX, 1)
			state.Set(RBX, 0x756e6547)
			state.Set(RCX, 0x6c65746e)
			state.Set(RDX, 0x49656e69)
		default:
			state.Set(RAX, 0)
			state.Set(RBX, 0)
			state.Set(RCX, 0)
			state.Set(RDX, 0)
		}
		state.RIP = next
		return Flow64Continue, nil
	case x86asm.HLT:
		state.RIP = next
		state.Halted = true
		state.TrapNo = Trap64Timer
		return Flow64Interrupt, nil
	default:
		return Flow64Stop, fmt.Errorf("%w at %#x (%s)", ErrUnsupported64, state.RIP, inst)
	}
}

func (j *JIT64) InvalidateRange(start, end Address64) {
	if j != nil && j.Cache != nil {
		j.Cache.InvalidateRange(start, end)
	}
}

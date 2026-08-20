package cpu

import (
	"context"
	"fmt"
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/wasmjit"
)

type WasmBlock64 struct {
	Host        *wasmjit.HostBlock
	Memory      *Memory64
	Start, End  uint64
	Pages       []Page64
	Generations map[Page64]uint64
}

func (j *WasmJIT) CompileBlock64(ctx context.Context, memory *Memory64, start Address64, maxBytes uint64) (*WasmBlock64, error) {
	block, err := CompileBlock64(memory, start, maxBytes)
	if err != nil {
		return nil, err
	}
	bytes, err := readBlock64(memory, block)
	if err != nil {
		return nil, err
	}
	host, err := j.Compile(ctx, block.Start, bytes)
	if err != nil {
		return nil, err
	}
	return &WasmBlock64{Host: host, Memory: memory, Start: block.Start, End: block.End, Pages: block.Pages, Generations: block.Generations}, nil
}

func readBlock64(memory *Memory64, block *CompiledBlock64) ([]byte, error) {
	if memory == nil || block == nil || block.End < block.Start {
		return nil, ErrInvalid64Block
	}
	length := block.End - block.Start + 1
	if length > uint64(^uint(0)>>1) {
		return nil, fmt.Errorf("%w: block too large", ErrInvalid64Block)
	}
	bytes := make([]byte, int(length))
	if err := memory.Read(Address64(block.Start), bytes); err != nil {
		return nil, err
	}
	return bytes, nil
}

func (b *WasmBlock64) Run(ctx context.Context, state *MachineState64) (Flow64, error) {
	if b == nil || b.Host == nil || state == nil {
		return Flow64Stop, ErrInvalid64Block
	}
	regs, flags, err := b.Host.RunRegsFlags(ctx, state.Regs, state.RFLAGS)
	state.Regs = regs
	applyPackedFlags(state, flags)
	flow, hasFlow := b.Host.Flow()
	return runWasmFlow(b.Memory, state, flow, hasFlow, err)
}

func runWasmFlow(memory *Memory64, state *MachineState64, flow machinecode.Instruction, hasFlow bool, err error) (Flow64, error) {
	if err != nil {
		return Flow64Stop, err
	}
	if !hasFlow {
		return Flow64Stop, nil
	}
	switch flow.Op {
	case machinecode.OpCall:
		if err := pushCall64(memory, state, flow.Fallthrough); err != nil {
			return Flow64Stop, err
		}
		state.RIP = flow.Target
	case machinecode.OpRET:
		if state.CallDepth == 0 {
			return Flow64Stop, nil
		}
		target, err := popReturn64(memory, state, uint64(flow.Imm))
		if err != nil {
			return Flow64Stop, err
		}
		state.RIP = target
	case machinecode.OpJcc:
		state.RIP = flow.Target
		if !conditionValue64(state, conditionCode64(flow.Cond)) {
			state.RIP = flow.Fallthrough
		}
	default:
		state.RIP = flow.Target
	}
	return Flow64Branch, nil
}

func applyPackedFlags(state *MachineState64, packed uint64) {
	const mask = Flag64CF | Flag64PF | Flag64AF | Flag64ZF | Flag64SF | Flag64OF
	state.RFLAGS = (state.RFLAGS &^ mask) | (packed & mask) | Flag64IF
	state.ExpandFlags()
}

func (b *WasmBlock64) Close(ctx context.Context) error {
	if b == nil || b.Host == nil {
		return nil
	}
	return b.Host.Close(ctx)
}

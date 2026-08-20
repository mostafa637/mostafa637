package cpu

import (
	"context"
	"fmt"
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/wasmjit"
)

type WasmBlock64 struct {
	Host        *wasmjit.HostBlock
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
	return &WasmBlock64{Host: host, Start: block.Start, End: block.End, Pages: block.Pages, Generations: block.Generations}, nil
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
	regs, flags, err := b.Host.RunRegsFlags(ctx, state.Regs)
	state.Regs = regs
	applyZeroFlag(state, flags)
	flow, hasFlow := b.Host.Flow()
	return runWasmFlow(state, flow, hasFlow, err)
}

func runWasmFlow(state *MachineState64, flow machinecode.Instruction, hasFlow bool, err error) (Flow64, error) {
	if err != nil {
		return Flow64Stop, err
	}
	if !hasFlow {
		return Flow64Stop, nil
	}
	if flow.Op == machinecode.OpJcc && !conditionValue64(state, conditionCode64(flow.Cond)) {
		state.RIP = flow.Fallthrough
	} else {
		state.RIP = flow.Target
	}
	return Flow64Branch, nil
}

func applyZeroFlag(state *MachineState64, flags uint64) {
	if flags != 0 {
		state.RFLAGS |= Flag64ZF
		return
	}
	state.RFLAGS &^= Flag64ZF
}

func (b *WasmBlock64) Close(ctx context.Context) error {
	if b == nil || b.Host == nil {
		return nil
	}
	return b.Host.Close(ctx)
}

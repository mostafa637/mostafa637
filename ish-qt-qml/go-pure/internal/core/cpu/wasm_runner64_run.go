package cpu

import (
	"context"
	"sync/atomic"

	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"
)

func (r *WasmRunner64) RunToInterrupt(ctx context.Context, state *MachineState64) uint64 {
	if state == nil || r == nil || r.chain == nil {
		return Trap64GeneralFault
	}
	state.Memory = r.chain.memory
	state.TrapNo = Trap64None
	for count := uint64(0); count < r.limit(); count++ {
		if r.interrupted(state) {
			state.TrapNo = Trap64Timer
			return state.TrapNo
		}
		flow, block, err := r.runBlock(ctx, state)
		if err != nil {
			return wasmFault(state, err)
		}
		if flow == Flow64Interrupt {
			if trap := r.handleSyscall(state); trap != Trap64None {
				return trap
			}
			continue
		}
		if r.stopped(block, state) {
			state.TrapNo = Trap64Exit
			return state.TrapNo
		}
	}
	state.TrapNo = Trap64Timer
	return state.TrapNo
}

func (r *WasmRunner64) limit() uint64 {
	if r.budget == 0 {
		return defaultWasmBudget64
	}
	return r.budget
}

func (r *WasmRunner64) interrupted(state *MachineState64) bool {
	return atomic.SwapUint32(&r.poked, 0) != 0 || atomic.SwapUint32(&state.Poked, 0) != 0
}

func (r *WasmRunner64) runBlock(ctx context.Context, state *MachineState64) (Flow64, *WasmBlock64, error) {
	block, err := r.chain.getOrCompile(ctx, state.RIP)
	if err != nil {
		return Flow64Stop, nil, err
	}
	flow, err := block.Run(ctx, state)
	return flow, block, err
}

func (r *WasmRunner64) stopped(block *WasmBlock64, state *MachineState64) bool {
	if block == nil || block.Host == nil {
		return false
	}
	flow, hasFlow := block.Host.Flow()
	return hasFlow && flow.Op == machinecode.OpRET && state.CallDepth == 0
}

func (r *WasmRunner64) handleSyscall(state *MachineState64) uint64 {
	if r.syscall == nil {
		state.TrapNo = Trap64Exit
		return state.TrapNo
	}
	resume, err := r.syscall(state)
	if err != nil {
		return wasmFault(state, err)
	}
	if resume && !state.Halted {
		state.TrapNo = Trap64None
		return Trap64None
	}
	state.TrapNo = Trap64Exit
	return state.TrapNo
}

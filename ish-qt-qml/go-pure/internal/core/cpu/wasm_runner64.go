package cpu

import (
	"context"
	"sync/atomic"
)

type WasmRunner64 struct {
	jit     *WasmJIT
	chain   *WasmChain64
	budget  uint64
	syscall func(*MachineState64) (bool, error)
	poked   uint32
}

const defaultWasmBudget64 = 1 << 10

func NewWasmRunner64(jit *WasmJIT, memory *Memory64, maxBytes uint64) *WasmRunner64 {
	return &WasmRunner64{jit: jit, chain: NewWasmChain64(jit, memory, maxBytes), budget: defaultWasmBudget64}
}

func (r *WasmRunner64) SetSyscall(handler func(*MachineState64) (bool, error)) {
	if r != nil {
		r.syscall = handler
	}
}

func (r *WasmRunner64) Poke() {
	if r != nil {
		atomic.StoreUint32(&r.poked, 1)
	}
}

func (r *WasmRunner64) Close(ctx context.Context) error {
	if r == nil {
		return nil
	}
	first := r.chain.Close(ctx)
	if r.jit != nil {
		if err := r.jit.Close(ctx); first == nil {
			first = err
		}
	}
	return first
}

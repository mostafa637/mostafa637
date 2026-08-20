package cpu

import (
	"context"
	"sync"
)

type WasmChain64 struct {
	jit      *WasmJIT
	memory   *Memory64
	maxBytes uint64
	mu       sync.RWMutex
	blocks   map[uint64]*WasmBlock64
}

func NewWasmChain64(jit *WasmJIT, memory *Memory64, maxBytes uint64) *WasmChain64 {
	return &WasmChain64{jit: jit, memory: memory, maxBytes: maxBytes, blocks: make(map[uint64]*WasmBlock64)}
}

func (c *WasmChain64) Get(pc uint64) (*WasmBlock64, bool) {
	c.mu.RLock()
	block := c.blocks[pc]
	c.mu.RUnlock()
	if block == nil || !block.Valid(c.memory) {
		if block != nil {
			c.InvalidateBlock(pc)
		}
		return nil, false
	}
	return block, true
}

func (c *WasmChain64) getOrCompile(ctx context.Context, pc uint64) (*WasmBlock64, error) {
	if block, ok := c.Get(pc); ok {
		return block, nil
	}
	block, err := c.jit.CompileBlock64(ctx, c.memory, Address64(pc), c.maxBytes)
	if err != nil {
		return nil, err
	}
	c.Put(block)
	return block, nil
}

func (c *WasmChain64) Run(ctx context.Context, state *MachineState64, maxBlocks uint64) (Flow64, error) {
	if c == nil || c.jit == nil || state == nil {
		return Flow64Stop, ErrInvalid64Block
	}
	if maxBlocks == 0 {
		maxBlocks = 1
	}
	for count := uint64(0); count < maxBlocks; count++ {
		block, err := c.getOrCompile(ctx, state.RIP)
		if err != nil {
			return Flow64Stop, err
		}
		flow, err := block.Run(ctx, state)
		if err != nil || flow != Flow64Branch {
			return flow, err
		}
	}
	return Flow64Branch, nil
}

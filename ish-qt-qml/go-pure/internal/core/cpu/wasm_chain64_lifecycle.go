package cpu

import "context"

func (c *WasmChain64) Put(block *WasmBlock64) {
	if c == nil || block == nil {
		return
	}
	c.mu.Lock()
	old := c.blocks[block.Start]
	c.blocks[block.Start] = block
	c.mu.Unlock()
	if old != nil {
		_ = old.Close(context.Background())
	}
}

func (c *WasmChain64) InvalidateBlock(pc uint64) {
	c.mu.Lock()
	block := c.blocks[pc]
	delete(c.blocks, pc)
	c.mu.Unlock()
	if block != nil {
		_ = block.Close(context.Background())
	}
}

func (c *WasmChain64) InvalidateRange(start, end Address64) {
	c.mu.RLock()
	var starts []uint64
	for pc, block := range c.blocks {
		if blockOverlaps(block, start, end) {
			starts = append(starts, pc)
		}
	}
	c.mu.RUnlock()
	for _, pc := range starts {
		c.InvalidateBlock(pc)
	}
}

func blockOverlaps(block *WasmBlock64, start, end Address64) bool {
	return block != nil && end > Address64(block.Start) && start <= Address64(block.End)
}

func (c *WasmChain64) Close(ctx context.Context) error {
	c.mu.Lock()
	blocks := make([]*WasmBlock64, 0, len(c.blocks))
	for pc, block := range c.blocks {
		blocks = append(blocks, block)
		delete(c.blocks, pc)
	}
	c.mu.Unlock()
	for _, block := range blocks {
		if err := block.Close(ctx); err != nil {
			return err
		}
	}
	return nil
}

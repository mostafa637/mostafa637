// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Ported to Go from src/jit/jit.c — the block cache and the entry into
// compiled code.
//
// Blocks are compiled per guest PC, on the second time the interpreter reaches
// it, and cached until the guest changes the mapping under them (munmap,
// mprotect, or a store into a page that holds compiled code — the last is not
// detected, which is the one way a guest can outrun this: self-modifying code
// must be excluded with a Flush).

package jit

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"unsafe"
)

// Fetch reads one instruction word from the guest, the way the interpreter's
// fetch does. The JIT only ever reads through it, at compile time.
type Fetch func(pc uint64) (uint32, bool)

// State is the guest CPU state a compiled block sees: the general-purpose
// registers, PSTATE's condition bits, PC, and the instruction count. The
// layout is the backend's to know (see backend_x86_64.go).
type State struct {
	X      [32]uint64
	NZCV   uint64
	PC     uint64
	ICount uint64
}

// Block is one compiled basic block.
//
// The fn value is a pointer to `entry`, the way a Go func value is a pointer to
// a funcval whose first word is the code pointer: `entry` has to live in the
// (heap-allocated) Block, not on the stack of whatever made the func value, or
// the func would point at a dead frame.
type Block struct {
	entry uintptr
	PC    uint64
	Insns int // guest instructions it covers
}

// Engine owns the compiled code and the cache.
type Engine struct {
	mu         sync.Mutex
	blocks     map[uint64]*Block
	hot        map[uint64]int
	chunks     [][]byte
	used       int
	Hot        int // executions before a PC is compiled (0 -> the default)
	Blocks     int
	Exits      int
	hotDefault int
}

const (
	defaultHot    = 24
	maxBlockInsns = 64
	chunkSize     = 256 * 1024
)

// New returns an engine with an empty cache.
func New() *Engine {
	return &Engine{
		blocks:     map[uint64]*Block{},
		hot:        map[uint64]int{},
		hotDefault: defaultHot,
	}
}

// Close releases the executable pages.
func (j *Engine) Close() {
	j.mu.Lock()
	defer j.mu.Unlock()
	for _, c := range j.chunks {
		_ = syscall.Munmap(c)
	}
	j.chunks, j.blocks, j.hot, j.used = nil, nil, nil, 0
}

// Flush throws every compiled block away: the guest has changed the mappings
// they were translated from, so the code they hold no longer describes what
// the guest will execute.
func (j *Engine) Flush() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.blocks = map[uint64]*Block{}
	j.hot = map[uint64]int{}
	j.used = 0
	for _, c := range j.chunks {
		for i := range c {
			c[i] = 0xcc // int3: a jump into a stale chunk traps loudly
		}
	}
}

// alloc reserves `n` bytes of executable memory.
func (j *Engine) alloc(n int) ([]byte, error) {
	if j.used+n <= chunkSize && len(j.chunks) > 0 {
		c := j.chunks[len(j.chunks)-1]
		out := c[j.used : j.used+n]
		j.used += n
		return out, nil
	}
	c, err := syscall.Mmap(-1, 0, chunkSize,
		syscall.PROT_READ|syscall.PROT_WRITE|syscall.PROT_EXEC,
		syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		return nil, err
	}
	j.chunks = append(j.chunks, c)
	j.used = n
	return c[:n], nil
}

// uintptrOf is the address of a byte slice's first element, for the tests that
// compile a block without going through Compile's bookkeeping.
func uintptrOf(b []byte) unsafe.Pointer { return unsafe.Pointer(&b[0]) }

// call runs the block against st, through the assembly shim.
func (b *Block) call(st *State) uint32 { return callBlock(b.entry, st) }

// Lookup returns the block compiled for pc, if there is one.
func (j *Engine) Lookup(pc uint64) *Block {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.blocks[pc]
}

// Compile lifts and translates the straight-line code starting at pc, up to a
// branch, an instruction the lifter does not know, or the block limit.
func (j *Engine) Compile(pc uint64, fetch Fetch) *Block {
	var block []Insn
	p := pc
	for i := 0; i < maxBlockInsns; i++ {
		insn, ok := fetch(p)
		if !ok {
			return nil
		}
		ir, lifted := Lift(p, insn)
		if !lifted {
			block = append(block, Insn{Op: OpUnsup, PC: p})
			break
		}
		block = append(block, ir)
		p += 4
		switch ir.Op {
		case OpBr, OpBrCond:
			i = maxBlockInsns // a branch ends the block
		}
	}
	if len(block) == 0 {
		return nil
	}
	// A block has to end with a terminator: the loop above always appends one
	// (an unlifted instruction), but a block that filled up without reaching
	// one needs the fallthrough made explicit.
	last := block[len(block)-1]
	switch last.Op {
	case OpUnsup, OpBr, OpBrCond:
	default:
		block = append(block, Insn{Op: OpEnd, PC: p, Fallthrough: p})
	}

	j.mu.Lock()
	code, err := j.alloc(len(block) * 96) // upper bound: ~96 bytes per IR insn
	if err != nil {
		j.mu.Unlock()
		return nil
	}
	emit := Compile(block)
	if len(emit) > len(code) {
		// The estimate was low: this should not happen, and the safe answer
		// is to refuse the block rather than write past the chunk.
		j.mu.Unlock()
		return nil
	}
	copy(code, emit)
	if os.Getenv("A64_JIT_DUMP") != "" {
		fmt.Fprintf(os.Stderr, "jit: block at 0x%x, %d ir insns, %d bytes:\n", pc, len(block), len(emit))
		for _, ir := range block {
			fmt.Fprintf(os.Stderr, "  0x%08x %+v\n", ir.PC, ir)
		}
		fmt.Fprintf(os.Stderr, "  % x\n", emit)
	}
	blk := &Block{PC: pc, Insns: len(block), entry: uintptr(unsafe.Pointer(&code[0]))}
	j.blocks[pc] = blk
	j.Blocks++
	j.mu.Unlock()
	return blk
}

// Execute runs compiled code from st.PC until an instruction the JIT does not
// have (it returns ExitUnsup with PC on that instruction) or the caller stops
// it. It reports how many guest instructions it retired, which is the block
// lengths — a block that branches early retires fewer, so the count is an
// upper bound and only ever feeds the instruction counter.
func (j *Engine) Execute(st *State, fetch Fetch) (uint32, uint64) {
	threshold := j.hotDefault
	if j.Hot > 0 {
		threshold = j.Hot
	}
	var insns uint64
	trace := os.Getenv("A64_JIT_TRACE") != ""
	for {
		if st.PC == 0 {
			return ExitUnsup, insns
		}
		j.mu.Lock()
		blk := j.blocks[st.PC]
		if blk == nil {
			n := j.hot[st.PC] + 1
			j.hot[st.PC] = n
			j.mu.Unlock()
			if n < threshold {
				return ExitUnsup, insns
			}
			blk = j.Compile(st.PC, fetch)
			if blk == nil {
				return ExitUnsup, insns
			}
		} else {
			j.mu.Unlock()
		}
		code := blk.call(st)
		insns += uint64(blk.Insns)
		j.Exits++
		if trace {
			fmt.Fprintf(os.Stderr, "jit: exit %d pc=0x%x code=%d x0=%d x1=%d nzcv=0x%x\n",
				j.Exits, st.PC, code, st.X[0], st.X[1], st.NZCV)
		}
		if code != ExitContinue {
			return code, insns
		}
	}
}

// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre

package jit

import (
	"testing"
)

// lift turns raw AArch64 words into a block, the way the engine does when it
// compiles from guest memory.
func liftWords(pc uint64, words ...uint32) (*Block, *Engine) {
	j := New()
	var block []Insn
	p := pc
	for _, w := range words {
		ir, ok := Lift(p, w)
		if !ok {
			block = append(block, Insn{Op: OpUnsup, PC: p})
			break
		}
		block = append(block, ir)
		p += 4
		if ir.Op == OpBr || ir.Op == OpBrCond {
			break
		}
	}
	block = append(block, Insn{Op: OpEnd, PC: p, Fallthrough: p})
	return j.emit(pc, block), j
}

// emit compiles a block into the engine's code region.
func (j *Engine) emit(pc uint64, block []Insn) *Block {
	code, err := j.alloc(len(block) * 96)
	if err != nil {
		return nil
	}
	src := Compile(block)
	copy(code, src)
	blk := &Block{PC: pc, Insns: len(block), entry: uintptr(uintptrOf(code))}
	j.blocks[pc] = blk
	return blk
}

func TestBlockAdd(t *testing.T) {
	// movz x0, #5 ; movz x1, #7 ; add x0, x0, x1
	blk, j := liftWords(0x1000, 0xd28000a0, 0xd28000e1, 0x8b010000)
	defer j.Close()
	if blk == nil {
		t.Fatal("no block")
	}
	st := &State{}
	st.PC = 0x1000
	if code := blk.call(st); code != ExitContinue {
		t.Fatalf("exit %d, want %d", code, ExitContinue)
	}
	if st.X[0] != 12 {
		t.Fatalf("X[0] = %d, want 12", st.X[0])
	}
	if st.PC != 0x100c {
		t.Fatalf("PC = 0x%x, want 0x100c", st.PC)
	}
}

func TestBlockSubsFlags(t *testing.T) {
	// subs x1, x1, #1  (0xF1000421)
	blk, j := liftWords(0x1000, 0xf1000421)
	defer j.Close()
	st := &State{}
	st.PC = 0x1000
	st.X[1] = 5
	blk.call(st)
	if st.X[1] != 4 {
		t.Fatalf("X[1] = %d, want 4", st.X[1])
	}
	// 5 - 4: N=0 Z=0 C=1 V=0
	if st.NZCV != 0x20000000 {
		t.Fatalf("NZCV = 0x%x, want 0x20000000", st.NZCV)
	}
	st.X[1] = 1
	blk.call(st)
	if st.X[1] != 0 || st.NZCV != 0x60000000 {
		t.Fatalf("1-1: X[1]=%d NZCV=0x%x, want 0 and 0x60000000", st.X[1], st.NZCV)
	}
}

func TestBlockBranch(t *testing.T) {
	// subs x1, x1, #1 ; b.ne -8   (loop while x1 != 0)
	blk, j := liftWords(0x1000, 0xf1000421, 0x54ffffc1)
	defer j.Close()
	st := &State{}
	st.PC = 0x1000
	st.X[1] = 3
	blk.call(st)
	if st.PC != 0xffc { // taken: the branch at 0x1004 targets 0x1004-8
		t.Fatalf("NE with x1=2: PC = 0x%x, want 0xffc", st.PC)
	}
	st.X[1] = 1
	blk.call(st)
	if st.PC != 0x1008 { // not taken: fall through
		t.Fatalf("NE with x1=0: PC = 0x%x, want 0x1008", st.PC)
	}
}

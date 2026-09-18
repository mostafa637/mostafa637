// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Ported to Go from src/jit/ir.h and src/jit/frontend.c — the IR and the
// AArch64 -> IR lifter.
//
// The C JIT has a full IR and three host backends (aarch64, x86-64, arm32,
// i686); this port carries the IR and one backend (x86-64) and lifts the
// integer subset a hot loop is made of. Anything it does not lift — memory
// accesses, system registers, multiplication of the wider kinds, FP/SIMD —
// ends the block with OpUnsup, and the interpreter runs that one instruction
// before the JIT is entered again.

package jit

import "github.com/mostafa637/mostafa637/core"

// Op is an IR operation.
type Op uint8

// The IR: one operation per guest instruction, in the block's order.
const (
	OpUnsup    Op = iota // not lifted: leave the block and interpret at PC
	OpSetImm             // X[D] = Imm
	OpMov                // X[D] = X[A]
	OpAdd                // X[D] = X[A] + X[B] (or - X[B] when Sub)
	OpAddImm             // X[D] = X[A] + Imm   (or - Imm when Sub)
	OpLogic              // X[D] = X[A] <Kind> X[B]
	OpLogicImm           // X[D] = X[A] <Kind> Imm
	OpShift              // X[D] = X[A] shifted by X[B] & (bits-1)
	OpShiftImm           // X[D] = X[A] shifted by Imm
	OpMul                // X[D] = X[A] * X[B]
	OpMovK               // MOVK: X[D] = (X[A] & ^(0xffff << (16*Kind))) | Imm
	OpCsel               // X[D] = cond ? X[A] : X[B]
	OpBr                 // PC = Target
	OpBrCond             // PC = Target when cond holds, else fall through
	OpEnd                // PC = Fallthrough: the block ran out
)

// Logic kinds for OpLogic / OpLogicImm.
const (
	LAnd = iota
	LOrr
	LEor
	LBic // A & ~B
	LOrn // A | ~B
)

// Shift kinds for OpShift / OpShiftImm.
const (
	SLSL = iota
	SLSR
	SASR
	SROR
)

// Insn is one IR instruction. Fields are only meaningful for the ops that use
// them; every Insn carries the guest PC it came from, so a block that leaves
// early can say where.
type Insn struct {
	Op          Op
	D, A, B     uint8
	Kind        uint8  // logic or shift kind
	Cond        uint8  // ARM condition code (0..15) for OpBrCond / OpCsel
	Sub         bool   // OpAdd / OpAddImm: subtract
	Is64        bool   // 64-bit operation (else the result is truncated to 32 bits)
	SetFlags    bool   // write NZCV as well as the result
	Imm         uint64 // OpSetImm / OpAddImm / OpLogicImm / OpShiftImm
	Target      uint64 // OpBr / OpBrCond
	Fallthrough uint64 // OpEnd / OpBrCond: the next guest PC
	PC          uint64 // the guest instruction this came from
}

// ---- the lifter -----------------------------------------------------------

func bits32(v uint32, hi, lo uint) uint32 {
	return (v >> lo) & ((1 << (hi - lo + 1)) - 1)
}
func bit32(v uint32, n uint) uint32 { return (v >> n) & 1 }

// Lift translates one AArch64 instruction into IR, reporting whether it could.
// An instruction it cannot lift becomes OpUnsup, which is not a failure: the
// block ends there and the interpreter takes that one instruction.
func Lift(pc uint64, insn uint32) (Insn, bool) {
	in := Insn{PC: pc, Is64: bit32(insn, 31) != 0}
	rd := uint8(bits32(insn, 4, 0))
	rn := uint8(bits32(insn, 9, 5))

	switch (insn >> 25) & 0xf {
	case 0x8, 0x9: // data processing - immediate
		return liftDPImm(in, insn, rd, rn)
	case 0x5, 0xd: // data processing - register
		return liftDPReg(in, insn, rd, rn)
	case 0xa, 0xb: // branches
		return liftBranch(in, insn)
	}
	return Insn{Op: OpUnsup, PC: pc}, false
}

func liftDPImm(in Insn, insn uint32, rd, rn uint8) (Insn, bool) {
	sf := in.Is64
	in.D, in.A = rd, rn
	switch bits32(insn, 28, 23) {
	case 0x22: // add/sub (immediate)
		sh := bit32(insn, 22) != 0
		imm := uint64(bits32(insn, 21, 10))
		if sh {
			imm <<= 12
		}
		// Rd == 31 with S is CMP (flags only); Rd == 31 without S is CMN.
		in.Op = OpAddImm
		in.Sub = bit32(insn, 30) != 0
		in.SetFlags = bit32(insn, 29) != 0
		in.Imm = imm
		if rd == 31 {
			if !in.SetFlags {
				return Insn{Op: OpUnsup, PC: in.PC}, false
			}
			in.D = 31 // CMP: no result, flags only
		}
		return in, true
	case 0x24: // logical (immediate)
		opc := bits32(insn, 30, 29)
		immN, immr, imms := bit32(insn, 22), bits32(insn, 21, 16), bits32(insn, 15, 10)
		wmask, _, ok := core.DecodeBitmasks(immN, imms, immr, sf)
		if !ok {
			return Insn{Op: OpUnsup, PC: in.PC}, false
		}
		switch opc {
		case 0: // AND
			in.Kind, in.SetFlags = LAnd, false
		case 1: // ORR
			in.Kind = LOrr
		case 2: // EOR
			in.Kind = LEor
		default: // ANDS
			in.Kind, in.SetFlags = LAnd, true
		}
		in.Op, in.Imm = OpLogicImm, wmask
		return in, true
	case 0x25: // move wide (immediate)
		opc := bits32(insn, 30, 29)
		hw := bits32(insn, 22, 21)
		imm := uint64(bits32(insn, 20, 5)) << (16 * hw)
		if !sf && hw >= 2 {
			return Insn{Op: OpUnsup, PC: in.PC}, false // unallocated
		}
		switch opc {
		case 0: // MOVN: the inverted immediate
			if sf {
				imm = ^imm
			} else {
				imm = uint64(uint32(^imm))
			}
			in.Op, in.Imm = OpSetImm, imm
			return in, true
		case 2: // MOVZ
			in.Op, in.Imm = OpSetImm, imm
			return in, true
		case 3: // MOVK: keep the rest of the register, replace one 16-bit lane
			in.Op, in.A, in.Kind, in.Imm = OpMovK, rd, uint8(hw), imm
			return in, true
		}
	}
	return Insn{Op: OpUnsup, PC: in.PC}, false
}

func liftDPReg(in Insn, insn uint32, rd, rn uint8) (Insn, bool) {
	rm := uint8(bits32(insn, 20, 16))
	in.D, in.A, in.B = rd, rn, rm
	op24 := bits32(insn, 28, 24)
	switch op24 {
	case 0x0a: // logical (shifted register)
		opc := bits32(insn, 30, 29)
		shift := bits32(insn, 23, 22)
		if shift != 0 || bit32(insn, 21) != 0 {
			return Insn{Op: OpUnsup, PC: in.PC}, false // no shifted operand
		}
		switch opc {
		case 0:
			in.Kind = LAnd
		case 1:
			in.Kind = LOrr
		case 2:
			in.Kind = LEor
		default:
			return Insn{Op: OpUnsup, PC: in.PC}, false // ANDS/BICS: flags
		}
		in.Op = OpLogic
		return in, true
	case 0x0b: // add/sub (shifted register)
		shift := bits32(insn, 23, 22)
		if shift != 0 || bit32(insn, 21) != 0 {
			return Insn{Op: OpUnsup, PC: in.PC}, false
		}
		imm6 := bits32(insn, 15, 10)
		if imm6 != 0 {
			return Insn{Op: OpUnsup, PC: in.PC}, false
		}
		in.Op = OpAdd
		in.Sub = bit32(insn, 30) != 0
		in.SetFlags = bit32(insn, 29) != 0
		if rd == 31 {
			if !in.SetFlags {
				return Insn{Op: OpUnsup, PC: in.PC}, false
			}
			in.D = 31 // CMP
		}
		return in, true
	case 0x1b: // data processing (3 source): MADD / MSUB
		op31, o0, ra := bits32(insn, 23, 21), bit32(insn, 15), bits32(insn, 14, 10)
		if op31 != 0 || ra != 31 || !in.Is64 {
			return Insn{Op: OpUnsup, PC: in.PC}, false
		}
		if o0 != 0 {
			return Insn{Op: OpUnsup, PC: in.PC}, false // MSUB
		}
		in.Op = OpMul
		return in, true
	case 0x1a:
		op21 := bits32(insn, 28, 21)
		switch op21 {
		case 0xd4: // conditional select
			if bit32(insn, 29) != 0 || bit32(insn, 11) != 0 {
				return Insn{Op: OpUnsup, PC: in.PC}, false
			}
			if bits32(insn, 30, 29) != 0 || bit32(insn, 10) != 0 {
				return Insn{Op: OpUnsup, PC: in.PC}, false // CSINC/CSINV/CSNEG
			}
			in.Op = OpCsel
			in.Cond = uint8(bits32(insn, 15, 12))
			return in, true
		case 0xd6: // data processing (2 source): variable shifts
			if bits32(insn, 30, 29) != 0 {
				return Insn{Op: OpUnsup, PC: in.PC}, false
			}
			opcode := bits32(insn, 15, 10)
			switch opcode {
			case 0x08: // LSLV
				in.Kind = SLSL
			case 0x09: // LSRV
				in.Kind = SLSR
			case 0x0a: // ASRV
				in.Kind = SASR
			case 0x0b: // RORV
				in.Kind = SROR
			default:
				return Insn{Op: OpUnsup, PC: in.PC}, false
			}
			in.Op = OpShift
			return in, true
		}
	}
	return Insn{Op: OpUnsup, PC: in.PC}, false
}

func liftBranch(in Insn, insn uint32) (Insn, bool) {
	switch bits32(insn, 31, 29) {
	case 0x5: // unconditional branch (immediate): B / BL
		if bit32(insn, 31) != 0 {
			return Insn{Op: OpUnsup, PC: in.PC}, false // BL: link
		}
		imm26 := signExtend(uint64(bits32(insn, 25, 0)), 25)
		in.Op = OpBr
		in.Target = uint64(int64(in.PC) + imm26*4)
		return in, true
	case 0x2: // conditional branch (immediate)
		if bit32(insn, 24) != 0 || bit32(insn, 4) != 0 {
			return Insn{Op: OpUnsup, PC: in.PC}, false
		}
		imm19 := signExtend(uint64(bits32(insn, 23, 5)), 18)
		in.Op = OpBrCond
		in.Cond = uint8(bits32(insn, 3, 0))
		in.Target = uint64(int64(in.PC) + imm19*4)
		in.Fallthrough = in.PC + 4
		return in, true
	}
	return Insn{Op: OpUnsup, PC: in.PC}, false
}

// signExtend widens a `bits`-wide two's-complement field to a signed value.
func signExtend(v uint64, bits uint) int64 {
	if v&(1<<bits) != 0 {
		v |= ^uint64(0) << bits
	}
	return int64(v)
}

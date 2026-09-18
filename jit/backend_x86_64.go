// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Ported to Go from src/jit/backend_x86_64.c — the x86-64 host backend.
//
// The C backend is a register allocator and a peephole pass over ~4k lines.
// This one is deliberately smaller and slower: every IR operation is a
// load/compute/store against the guest register file in memory, with two
// scratch registers, and no value is kept across IR instructions. What it
// buys is a short, obviously-correct encoder — the same block semantics the
// interpreter has, without a second implementation of the AArch64 flag rules
// to disagree with it.
//
// Calling convention: rdi = *State, and r15 is the state base inside the block
// (every callee-saved register the Go ABI cares about is pushed and popped,
// r14 included, which is the goroutine pointer).
//
// NZCV lives in the host flags only for the duration of one operation: it is
// captured with LAHF (SF/ZF/CF in one byte) plus SETO, which is why a
// conditional branch rebuilds the condition from the stored word rather than
// expecting the host flags to still hold it.

package jit

import "encoding/binary"

// x86-64 register numbers.
const (
	rAX = iota
	rCX
	rDX
	rBX
	rSP
	rBP
	rSI
	rDI
	r8
	r9
	r10
	r11
	r12
	r13
	r14
	r15
)

// State offsets (see State in jit.go).
const (
	offX     = 0
	offNZCV  = 32 * 8
	offPC    = offNZCV + 8
	offCount = offPC + 8
)

// Exit codes a compiled block returns.
const (
	ExitContinue = iota // the block ran out; PC says where to go next
	ExitUnsup           // an unlifted instruction; interpret one at PC
)

// enc is the machine-code emitter for one block.
type enc struct {
	code []byte
}

func (e *enc) u8(b byte)       { e.code = append(e.code, b) }
func (e *enc) u16(v uint16)    { e.code = append(e.code, byte(v), byte(v>>8)) }
func (e *enc) u32(v uint32)    { e.code = binary.LittleEndian.AppendUint32(e.code, v) }
func (e *enc) u64(v uint64)    { e.code = binary.LittleEndian.AppendUint64(e.code, v) }
func (e *enc) bytes(b ...byte) { e.code = append(e.code, b...) }

// rex emits a REX prefix. w = 64-bit operand size, r/x/b extend reg, index,
// and base/r-m fields.
func (e *enc) rex(w, r, x, b bool) {
	v := byte(0x40)
	if w {
		v |= 0x08
	}
	if r {
		v |= 0x04
	}
	if x {
		v |= 0x02
	}
	if b {
		v |= 0x01
	}
	e.u8(v)
}

// load emits `mov reg, [r15+off]`.
func (e *enc) load(reg int, off int32) {
	e.rex(true, reg >= 8, false, true)
	e.u8(0x8b)
	e.u8(byte(0x80 | (reg&7)<<3 | 7))
	e.u32(uint32(off))
}

// load32 emits `mov reg, [r15+off]` with a 32-bit destination (the upper half
// of the register is zeroed by the hardware, which is the truncation the
// AArch64 32-bit form wants).
func (e *enc) load32(reg int, off int32) {
	e.rex(false, reg >= 8, false, true)
	e.u8(0x8b)
	e.u8(byte(0x80 | (reg&7)<<3 | 7))
	e.u32(uint32(off))
}

// store emits `mov [r15+off], reg`.
func (e *enc) store(off int32, reg int) {
	e.rex(true, reg >= 8, false, true)
	e.u8(0x89)
	e.u8(byte(0x80 | (reg&7)<<3 | 7))
	e.u32(uint32(off))
}

// moveImm puts a 64-bit constant into a register.
func (e *enc) moveImm(reg int, v uint64) {
	if v == 0 {
		e.rex(true, reg >= 8, false, false)
		e.u8(0x31) // xor reg, reg
		e.u8(byte(0xc0 | (reg&7)<<3 | (reg & 7)))
		return
	}
	if uint64(int32(int64(v))) == v || v>>32 == 0 {
		if v>>32 == 0 {
			e.rex(false, false, false, reg >= 8)
			e.u8(0xb8 + byte(reg&7)) // mov reg, imm32 (zero-extends)
			e.u32(uint32(v))
			return
		}
		e.rex(true, false, false, reg >= 8)
		e.u8(0xc7) // mov r/m64, imm32 (sign-extended)
		e.u8(byte(0xc0 | (reg & 7)))
		e.u32(uint32(int32(int64(v))))
		return
	}
	e.rex(true, false, false, reg >= 8)
	e.u8(0xb8 + byte(reg&7)) // movabs reg, imm64
	e.u64(v)
}

// alu emits `op r/m, reg` (op: 0x01 add, 0x29 sub, 0x21 and, 0x09 or, 0x31 xor,
// 0x39 cmp) with the destination in `dst` and the source in `src`.
func (e *enc) alu(op byte, dst, src int, is64 bool) {
	e.rex(is64, src >= 8, false, dst >= 8)
	e.u8(op)
	e.u8(byte(0xc0 | (src&7)<<3 | (dst & 7)))
}

// aluImm emits `op r/m, imm32` with an 0x81-group opcode extended by `ext`.
func (e *enc) aluImm(dst, ext int, imm uint32, is64 bool) {
	e.rex(is64, false, false, dst >= 8)
	e.u8(0x81)
	e.u8(byte(0xc0 | (ext&7)<<3 | (dst & 7)))
	e.u32(imm)
}

// aluImm64 is aluImm for a constant that does not fit in a signed 32 bits: it
// goes through a scratch register.
func (e *enc) aluImm64(dst, ext, scratch int, imm uint64, is64 bool) {
	e.moveImm(scratch, imm)
	e.alu(0x01+byte(ext)*8, dst, scratch, is64)
}

// shift emits `shl/shr/sar/ror reg, cl` (kind 4/5/7/1 in the /r extension).
func (e *enc) shift(reg, kind int, is64 bool) {
	e.rex(is64, false, false, reg >= 8)
	e.u8(0xd3)
	e.u8(byte(0xc0 | (kind&7)<<3 | (reg & 7)))
}

// shiftImm emits `shl/shr/sar/ror reg, imm8`.
func (e *enc) shiftImm(reg, kind int, imm uint8, is64 bool) {
	e.rex(is64, false, false, reg >= 8)
	e.u8(0xc1)
	e.u8(byte(0xc0 | (kind&7)<<3 | (reg & 7)))
	e.u8(imm)
}

// notReg emits `not reg`.
func (e *enc) notReg(reg int, is64 bool) {
	e.rex(is64, false, false, reg >= 8)
	e.u8(0xf7)
	e.u8(byte(0xc0 | (2)<<3 | (reg & 7)))
}

// imul emits `imul dst, src`.
func (e *enc) imul(dst, src int, is64 bool) {
	e.rex(is64, dst >= 8, false, src >= 8)
	e.u8(0x0f)
	e.u8(0xaf)
	e.u8(byte(0xc0 | (dst&7)<<3 | (src & 7)))
}

// captureFlags stores the host flags as an AArch64 NZCV word. For a
// subtraction the carry is complemented first: an ARM C is "no borrow",
// the x86 CF after a SUB is "borrow".
func (e *enc) captureFlags(sub bool) {
	if sub {
		e.u8(0xf5) // cmc: complement CF
	}
	e.u8(0x9f)                // lahf: ah = SF ZF - AF - PF 1 CF
	e.bytes(0x0f, 0x90, 0xc1) // seto cl                       (V)
	// The carry comes out of AH first: the shift below moves the flags byte
	// out of AH, so anything read from it afterwards is not the flags.
	e.bytes(0x0f, 0xb6, 0xd4) // movzx edx, ah
	e.bytes(0x83, 0xe2, 0x01) // and edx, 1                    (CF)
	e.bytes(0xc1, 0xe2, 0x1d) // shl edx, 29                   -> C at bit 29
	e.bytes(0x0f, 0xb6, 0xc4) // movzx eax, ah
	e.u8(0x25)
	e.u32(0xc0)               // and eax, 0xc0                    (SF, ZF)
	e.bytes(0xc1, 0xe0, 0x18) // shl eax, 24                   -> N at 31, Z at 30
	e.bytes(0x09, 0xd0)       // or eax, edx
	e.bytes(0x0f, 0xb6, 0xc9) // movzx ecx, cl
	e.bytes(0xc1, 0xe1, 0x1c) // shl ecx, 28                   -> V at bit 28
	e.bytes(0x09, 0xc8)       // or eax, ecx
	e.rex(false, false, false, true)
	e.u8(0x89)           // mov [r15+offNZCV], eax (32-bit: the word's
	e.u8(byte(0x80 | 7)) // upper half is zeroed)
	e.u32(offNZCV)
}

// condIntoAL evaluates an AArch64 condition against the stored NZCV word,
// leaving 1 in AL when it holds.
func (e *enc) condIntoAL(cond uint8) {
	e.rex(false, false, false, true)
	e.u8(0x8b)
	e.u8(byte(0x80 | (rCX&7)<<3 | 7))
	e.u32(offNZCV) // mov ecx, [r15+offNZCV]
	switch cond {
	case 0: // EQ: Z
		e.u8(0xf7)
		e.u8(0xc1)
		e.u32(0x40000000)         // test ecx, Z
		e.bytes(0x0f, 0x95, 0xc0) // setne al
	case 1: // NE
		e.u8(0xf7)
		e.u8(0xc1)
		e.u32(0x40000000)
		e.bytes(0x0f, 0x94, 0xc0) // sete al
	case 2: // CS/HS: C
		e.u8(0xf7)
		e.u8(0xc1)
		e.u32(0x20000000)
		e.bytes(0x0f, 0x95, 0xc0)
	case 3: // CC/LO
		e.u8(0xf7)
		e.u8(0xc1)
		e.u32(0x20000000)
		e.bytes(0x0f, 0x94, 0xc0)
	case 4: // MI: N
		e.u8(0xf7)
		e.u8(0xc1)
		e.u32(0x80000000)
		e.bytes(0x0f, 0x95, 0xc0)
	case 5: // PL
		e.u8(0xf7)
		e.u8(0xc1)
		e.u32(0x80000000)
		e.bytes(0x0f, 0x94, 0xc0)
	case 6: // VS: V
		e.u8(0xf7)
		e.u8(0xc1)
		e.u32(0x10000000)
		e.bytes(0x0f, 0x95, 0xc0)
	case 7: // VC
		e.u8(0xf7)
		e.u8(0xc1)
		e.u32(0x10000000)
		e.bytes(0x0f, 0x94, 0xc0)
	case 8, 9: // HI: C && !Z, LS: !(C && !Z)
		e.bytes(0x89, 0xc8) // mov eax, ecx
		e.u8(0x25)
		e.u32(0x20000000)         // and eax, C
		e.bytes(0x0f, 0x95, 0xc0) // setne al
		e.bytes(0x89, 0xca)       // mov edx, ecx
		e.u8(0x81)
		e.u8(0xe2)
		e.u32(0x40000000)         // and edx, Z
		e.bytes(0x0f, 0x94, 0xc2) // sete dl
		e.bytes(0x21, 0xd0)       // and eax, edx
		if cond == 9 {
			e.bytes(0x0f, 0x94, 0xc0) // sete al
		}
	case 10, 11: // GE: N == V, LT: N != V
		e.bytes(0x89, 0xc8) // mov eax, ecx
		e.u8(0xc1)
		e.u8(0xe8)
		e.u8(31)            // shr eax, 31        (N)
		e.bytes(0x89, 0xca) // mov edx, ecx
		e.u8(0xc1)
		e.u8(0xea)
		e.u8(28) // shr edx, 28
		e.u8(0x83)
		e.u8(0xe2)
		e.u8(1)             // and edx, 1          (V)
		e.bytes(0x39, 0xd0) // cmp eax, edx
		if cond == 10 {
			e.bytes(0x0f, 0x94, 0xc0) // sete al
		} else {
			e.bytes(0x0f, 0x95, 0xc0) // setne al
		}
	case 12, 13: // GT: !Z && N == V, LE: !(!Z && N == V)
		e.bytes(0x89, 0xc8)
		e.u8(0xc1)
		e.u8(0xe8)
		e.u8(31) // eax = N
		e.bytes(0x89, 0xca)
		e.u8(0xc1)
		e.u8(0xea)
		e.u8(28)
		e.u8(0x83)
		e.u8(0xe2)
		e.u8(1)                   // edx = V
		e.bytes(0x39, 0xd0)       // cmp eax, edx
		e.bytes(0x0f, 0x94, 0xc0) // sete al        (N == V)
		e.bytes(0x89, 0xca)       // mov edx, ecx
		e.u8(0x81)
		e.u8(0xe2)
		e.u32(0x40000000)         // and edx, Z
		e.bytes(0x0f, 0x94, 0xc2) // sete dl        (!Z)
		e.bytes(0x21, 0xd0)       // and eax, edx
		if cond == 13 {
			e.bytes(0x0f, 0x94, 0xc0) // sete al
		}
	default: // AL / NV: always
		e.bytes(0xb0, 0x01) // mov al, 1
	}
}

// ---- block compilation ----------------------------------------------------

// Compile turns one lifted block into machine code.
func Compile(block []Insn) []byte {
	e := &enc{}
	// Prologue: the callee-saved registers the Go ABI depends on (r14 is the
	// goroutine pointer), then r15 = the state base from rdi.
	for _, r := range []int{rBX, rBP, r12, r13, r14, r15} {
		e.rex(false, false, false, r >= 8)
		e.u8(0x50 + byte(r&7))
	}
	// mov r15, rdi: REX.W + REX.B (r/m is r15), opcode 89 /r, modrm c0|7<<3|7.
	//
	// Blocks are entered through call_amd64.s, which puts the state pointer in
	// rdi (the convention the C backend uses too), not through a Go func
	// value -- the runtime validates what those point at.
	e.rex(true, false, false, true)
	e.u8(0x89)
	e.u8(0xff)

	for _, in := range block {
		switch in.Op {
		case OpSetImm:
			e.moveImm(rAX, trunc(in.Imm, in.Is64))
			if in.D != 31 {
				e.store(xOff(in.D), rAX)
			}
		case OpMov:
			e.load(rAX, xOff(in.A))
			e.store(xOff(in.D), rAX)
		case OpAdd:
			e.load(rAX, xOff(in.A))
			e.load(rCX, xOff(in.B))
			if in.Sub {
				e.alu(0x29, rAX, rCX, in.Is64)
			} else {
				e.alu(0x01, rAX, rCX, in.Is64)
			}
			// The result has to be stored before the flags are captured:
			// captureFlags is free to use every scratch register, rax
			// included, and a store leaves the host flags alone.
			if in.D != 31 {
				e.store(xOff(in.D), rAX)
			}
			if in.SetFlags {
				e.captureFlags(in.Sub)
			}
		case OpAddImm:
			e.load(rAX, xOff(in.A))
			ext := 0 // /0 add
			if in.Sub {
				ext = 5 // /5 sub
			}
			imm := trunc(in.Imm, in.Is64)
			if imm>>31 == 0 || imm>>32 == 0 && imm&0x80000000 == 0 {
				e.aluImm(rAX, ext, uint32(imm), in.Is64)
			} else {
				e.aluImm64(rAX, ext, rCX, imm, in.Is64)
			}
			if in.D != 31 {
				e.store(xOff(in.D), rAX)
			}
			if in.SetFlags {
				e.captureFlags(in.Sub)
			}
		case OpLogic:
			e.load(rAX, xOff(in.A))
			e.load(rCX, xOff(in.B))
			switch in.Kind {
			case LAnd:
				e.alu(0x21, rAX, rCX, in.Is64)
			case LOrr:
				e.alu(0x09, rAX, rCX, in.Is64)
			case LEor:
				e.alu(0x31, rAX, rCX, in.Is64)
			case LBic:
				e.notReg(rCX, in.Is64)
				e.alu(0x21, rAX, rCX, in.Is64)
			case LOrn:
				e.notReg(rCX, in.Is64)
				e.alu(0x09, rAX, rCX, in.Is64)
			}
			if in.D != 31 {
				e.store(xOff(in.D), rAX)
			}
		case OpLogicImm:
			e.load(rAX, xOff(in.A))
			imm := trunc(in.Imm, in.Is64)
			switch in.Kind {
			case LAnd:
				e.logicImm(rAX, 4, imm, in.Is64) // /4 and
			case LOrr:
				e.logicImm(rAX, 1, imm, in.Is64) // /1 or
			case LEor:
				e.logicImm(rAX, 6, imm, in.Is64) // /6 xor
			}
			if in.D != 31 {
				e.store(xOff(in.D), rAX)
			}
			if in.SetFlags {
				// ANDS: N and Z from the result, C and V clear -- a logical
				// operation leaves the host CF alone, so it is cleared first.
				e.u8(0xf8) // clc
				e.captureFlags(false)
			}
		case OpMovK:
			e.load(rAX, xOff(in.A))
			mask := ^uint64(0xffff << (16 * uint(in.Kind)))
			if in.Is64 {
				e.moveImm(rCX, mask)
				e.alu(0x21, rAX, rCX, true)
				e.aluImm64(rAX, 1, rCX, in.Imm, true)
			} else {
				e.moveImm(rCX, uint64(uint32(mask)))
				e.alu(0x23, rAX, rCX, false) // 32-bit and
				e.aluImm(rAX, 1, uint32(in.Imm), false)
			}
			if in.D != 31 {
				e.store(xOff(in.D), rAX)
			}
		case OpShift:
			e.load(rAX, xOff(in.A))
			e.load(rCX, xOff(in.B))
			bits := uint8(63)
			if !in.Is64 {
				bits = 31
			}
			e.aluImm(rCX, 4, uint32(bits), in.Is64) // and rcx, bits
			switch in.Kind {
			case SLSL:
				e.shift(rAX, 4, in.Is64)
			case SLSR:
				e.shift(rAX, 5, in.Is64)
			case SASR:
				e.shift(rAX, 7, in.Is64)
			case SROR:
				e.shift(rAX, 1, in.Is64)
			}
			if !in.Is64 {
				e.moveImm(rCX, 0xffffffff)
				e.alu(0x21, rAX, rCX, true) // zero-extend the 32-bit result
			}
			if in.D != 31 {
				e.store(xOff(in.D), rAX)
			}
		case OpShiftImm:
			e.load(rAX, xOff(in.A))
			switch in.Kind {
			case SLSL:
				e.shiftImm(rAX, 4, uint8(in.Imm&63), in.Is64)
			case SLSR:
				e.shiftImm(rAX, 5, uint8(in.Imm&63), in.Is64)
			case SASR:
				e.shiftImm(rAX, 7, uint8(in.Imm&63), in.Is64)
			case SROR:
				e.shiftImm(rAX, 1, uint8(in.Imm&63), in.Is64)
			}
			if !in.Is64 {
				e.moveImm(rCX, 0xffffffff)
				e.alu(0x21, rAX, rCX, true)
			}
			if in.D != 31 {
				e.store(xOff(in.D), rAX)
			}
		case OpMul:
			e.load(rAX, xOff(in.A))
			e.load(rCX, xOff(in.B))
			e.imul(rAX, rCX, in.Is64)
			if in.D != 31 {
				e.store(xOff(in.D), rAX)
			}
		case OpCsel:
			e.condIntoAL(in.Cond)
			e.load(rAX, xOff(in.A))
			e.load(rCX, xOff(in.B))
			e.bytes(0x84, 0xc0) // test al, al
			e.rex(true, rAX >= 8, false, rCX >= 8)
			e.bytes(0x0f, 0x44, 0xc1) // cmove rax, rcx
			if in.D != 31 {
				e.store(xOff(in.D), rAX)
			}
		case OpEnd, OpUnsup:
			e.moveImm(rAX, in.PC)
			e.store(offPC, rAX)
			code := ExitContinue
			if in.Op == OpUnsup {
				code = ExitUnsup
			}
			e.moveImm(rAX, uint64(code))
			e.epilogue()
			return e.code
		case OpBr:
			e.moveImm(rAX, in.Target)
			e.store(offPC, rAX)
			e.moveImm(rAX, ExitContinue)
			e.epilogue()
			return e.code
		case OpBrCond:
			e.condIntoAL(in.Cond)
			e.bytes(0x84, 0xc0) // test al, al
			e.bytes(0x74, 0x00) // jz rel8 -> not taken (patched below)
			jzDisp := len(e.code) - 1
			arm := len(e.code) // the taken arm: target, exit
			e.moveImm(rAX, in.Target)
			e.store(offPC, rAX)
			e.moveImm(rAX, ExitContinue)
			e.epilogue()
			// The displacement is the length of the arm the branch skips.
			if taken := len(e.code) - arm; taken <= 127 {
				e.code[jzDisp] = byte(taken)
			}
			e.moveImm(rAX, in.Fallthrough)
			e.store(offPC, rAX)
			e.moveImm(rAX, ExitContinue)
			e.epilogue()
			return e.code
		}
	}
	// A block that ends without a terminator (should not happen: the lifter
	// always appends one) leaves PC where it is and reports "unsupported".
	e.moveImm(rAX, ExitUnsup)
	e.epilogue()
	return e.code
}

// logicImm emits an and/or/xor with an immediate, through a scratch register
// when the constant does not fit in a sign-extended 32 bits.
func (e *enc) logicImm(dst, ext int, imm uint64, is64 bool) {
	if is64 && imm>>31 != 0 {
		e.moveImm(rCX, imm)
		e.alu(0x01+byte(ext)*8, dst, rCX, is64)
		return
	}
	e.aluImm(dst, ext, uint32(imm), is64)
}

// epilogue restores the callee-saved registers and returns.
func (e *enc) epilogue() {
	for i := len([]int{rBX, rBP, r12, r13, r14, r15}) - 1; i >= 0; i-- {
		r := []int{rBX, rBP, r12, r13, r14, r15}[i]
		e.rex(false, false, false, r >= 8)
		e.u8(0x58 + byte(r&7))
	}
	e.u8(0xc3) // ret
}

func xOff(reg uint8) int32 { return int32(offX + 8*int(reg)) }

func trunc(v uint64, is64 bool) uint64 {
	if is64 {
		return v
	}
	return uint64(uint32(v))
}

// aluImm's `ext` for the 0x81 group, and the matching reg-field opcode used
// when the operation has to go through a register.
//
//	add /0 = 0x01, or /1 = 0x09, and /4 = 0x21, sub /5 = 0x29, xor /6 = 0x31,
//	cmp /7 = 0x39
var _ = []int{0x01, 0x09, 0x21, 0x29, 0x31, 0x39}

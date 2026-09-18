// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Advanced SIMD: the integer groups — the Go port of the vector half of
// src/core/exec_fpsimd.c.
//
// This is the second slice of the FP/SIMD unit, after core/fpsimd.go's scalar
// floating point. It covers the groups a real aarch64 libc leans on for its
// string and memory functions and for anything vectorized:
//
//	three-same      ADD SUB MUL MLA MLS CMEQ CMTST CMGE CMGT CMHI CMHS
//	                SMAX SMIN UMAX UMIN SHADD UHADD SRHADD URHADD SHSUB
//	                UHSUB SQADD UQADD SQSUB UQSUB SSHL USHL SRSHL URSHL
//	                SQSHL UQSHL SQRSHL UQRSHL SABD UABD SABA UABA PMUL
//	                SQDMULH SQRDMULH ADDP SMAXP SMINP UMAXP UMINP
//	                AND BIC ORR ORN EOR BSL BIT BIF
//	modified imm    MOVI MVNI ORR BIC FMOV (vector, immediate)
//	copy            DUP (element and general) INS UMOV SMOV
//	shift by imm    SHL SSHR USHR SLI SRI SSRA USRA SRSHR URSHR SRSRA
//	                URSRA SQSHL UQSHL SQSHLU SSHLL USHLL
//
// The element size and lane count come straight from the encoding (size and
// Q), and every group is a loop over lanes with the operation selected by
// (U, opcode) — the shape the C uses, and the one that keeps the port
// diffable against it.
//
// Still UNDEFINED: the FP three-same and FP16 groups, the narrowing shifts,
// the by-element and table/permute groups, and the crypto extensions.

package core

// fpsrQC is FPSR's cumulative saturation bit, set by every saturating
// instruction that clamped (ARM D13.2.59).
const fpsrQC = 1 << 27

// ---- element access -------------------------------------------------------

// velemGet reads lane `idx` of a vector as an unsigned value of `size`'s width:
// size 0 is a byte, 1 a halfword, 2 a word, 3 a doubleword.
func velemGet(v *V128, size, idx uint32) uint64 {
	switch size {
	case 0:
		return uint64(v.U8(int(idx)))
	case 1:
		return uint64(v.U16(int(idx)))
	case 2:
		return uint64(v.U32(int(idx)))
	}
	return v.U64(int(idx))
}

func velemSet(v *V128, size, idx uint32, val uint64) {
	switch size {
	case 0:
		v.SetU8(int(idx), uint8(val))
	case 1:
		v.SetU16(int(idx), uint16(val))
	case 2:
		v.SetU32(int(idx), uint32(val))
	default:
		v.SetU64(int(idx), val)
	}
}

// elemMask is the element's value mask: all ones for 64-bit lanes, and
// 2^esize-1 below that (a shift by 64 is defined in Go and gives 0).
func elemMask(esize uint32) uint64 {
	if esize >= 64 {
		return ^uint64(0)
	}
	return 1<<esize - 1
}

// elemSx sign-extends an element of `esize` bits to an int64.
func elemSx(v uint64, esize uint32) int64 {
	if esize >= 64 {
		return int64(v)
	}
	return int64(v) << (64 - esize) >> (64 - esize)
}

// ---- saturation -----------------------------------------------------------

// satS clamps a signed value to a signed element, setting QC when it moved.
func (e *fpEnv) satS(v int64, esize uint32) uint64 {
	max, min := intMaxMin(esize)
	if v > max {
		v = max
		e.exc |= fpsrQC
	} else if v < min {
		v = min
		e.exc |= fpsrQC
	}
	return uint64(v)
}

func (e *fpEnv) satU(v int64, esize uint32) uint64 {
	var max uint64 = ^uint64(0)
	if esize < 64 {
		max = 1<<esize - 1
	}
	if v < 0 {
		e.exc |= fpsrQC
		return 0
	}
	if uint64(v) > max {
		e.exc |= fpsrQC
		return max
	}
	return uint64(v)
}

func (e *fpEnv) satAddS(a, b int64, esize uint32) uint64 {
	if esize < 64 {
		return e.satS(a+b, esize)
	}
	r := a + b
	if (a^r)&(b^r) < 0 { // signed overflow
		e.exc |= fpsrQC
		if a < 0 {
			return 1 << 63
		}
		return 1<<63 - 1
	}
	return uint64(r)
}

func (e *fpEnv) satSubS(a, b int64, esize uint32) uint64 {
	if esize < 64 {
		return e.satS(a-b, esize)
	}
	r := a - b
	if (a^b)&(a^r) < 0 {
		e.exc |= fpsrQC
		if a < 0 {
			return 1 << 63
		}
		return 1<<63 - 1
	}
	return uint64(r)
}

func (e *fpEnv) satAddU(a, b uint64, esize uint32) uint64 {
	r := a + b
	if esize < 64 {
		m := uint64(1)<<esize - 1
		if r > m {
			e.exc |= fpsrQC
			return m
		}
		return r
	}
	if r < a { // carry out
		e.exc |= fpsrQC
		return ^uint64(0)
	}
	return r
}

func (e *fpEnv) satSubU(a, b uint64) uint64 {
	if a < b {
		e.exc |= fpsrQC
		return 0
	}
	return a - b
}

// ---- shifts ---------------------------------------------------------------

// vregShift is the shift kernel shared by the register and immediate forms.
// sh is the amount (negative is a right shift), sgn selects arithmetic or
// logical, round adds the bit that was shifted out, and sat clamps.
func (e *fpEnv) vregShift(val uint64, sh int, esize uint32, sgn, round, sat bool) uint64 {
	emask := elemMask(esize)
	if sh >= 0 { // left
		if !sat {
			return val << uint(sh) & emask
		}
		if sgn {
			sv := elemSx(val, esize)
			max, min := intMaxMin(esize)
			if sv == 0 {
				return 0
			}
			// A shift by at least the element width overflows for every
			// nonzero value, and the `min >> sh` bound below cannot say so:
			// min is held at 64 bits, so below 64 it only decays to -1.
			if sh >= int(esize) || sh >= 64 {
				e.exc |= fpsrQC
				if sv > 0 {
					return uint64(max) & emask
				}
				return uint64(min) & emask
			}
			if sv > 0 {
				if sv > max>>uint(sh) {
					e.exc |= fpsrQC
					return uint64(max)
				}
				return uint64(sv) << uint(sh) & emask
			}
			if sv < min>>uint(sh) {
				e.exc |= fpsrQC
				return uint64(min) & emask
			}
			return uint64(sv) << uint(sh) & emask
		}
		uv := val & emask
		if uv == 0 {
			return 0
		}
		if sh >= 64 {
			e.exc |= fpsrQC
			return emask
		}
		if uv > emask>>uint(sh) {
			e.exc |= fpsrQC
			return emask
		}
		return uv << uint(sh) & emask
	}
	rs := uint(-sh)
	if sgn {
		sv := elemSx(val, esize)
		if rs >= 64 {
			if round {
				return 0
			}
			if sv < 0 {
				return ^uint64(0) & emask
			}
			return 0
		}
		w := sv >> rs
		if round && rs >= 1 {
			w += sv >> (rs - 1) & 1
		}
		return uint64(w) & emask
	}
	uv := val & emask
	if rs >= 64 {
		if !round {
			return 0
		}
		if rs == 64 {
			return uv >> 63 & 1 & emask
		}
		return 0
	}
	w := uv >> rs
	if round && rs >= 1 {
		w += uv >> (rs - 1) & 1
	}
	return w & emask
}

// ---- three-same -----------------------------------------------------------

func execSIMDThreeSame(e *fpEnv, insn uint32) {
	c := e.c
	q := bit(insn, 30)
	u := bit(insn, 29)
	size := bits(insn, 23, 22)
	rm := bits(insn, 20, 16)
	opc := bits(insn, 15, 11)
	rn := bits(insn, 9, 5)
	rd := bits(insn, 4, 0)

	if opc >= 0x18 {
		undefined(c, insn) // FP three-same: on demand
		return
	}

	if opc == 0x03 { // logical: the operation is on whole 64-bit halves
		a0, a1 := c.V[rn].U64(0), c.V[rn].U64(1)
		b0, b1 := c.V[rm].U64(0), c.V[rm].U64(1)
		d0, d1 := c.V[rd].U64(0), c.V[rd].U64(1)
		var r0, r1 uint64
		if !u {
			switch size {
			case 0:
				r0, r1 = a0&b0, a1&b1 // AND
			case 1:
				r0, r1 = a0&^b0, a1&^b1 // BIC
			case 2:
				r0, r1 = a0|b0, a1|b1 // ORR
			default:
				r0, r1 = a0|^b0, a1|^b1 // ORN
			}
		} else {
			switch size {
			case 0:
				r0, r1 = a0^b0, a1^b1 // EOR
			case 1:
				r0, r1 = b0^(b0^a0)&d0, b1^(b1^a1)&d1 // BSL
			case 2:
				r0, r1 = d0^(d0^a0)&b0, d1^(d1^a1)&b1 // BIT
			default:
				r0, r1 = d0^(d0^a0)&^b0, d1^(d1^a1)&^b1 // BIF
			}
		}
		c.V[rd].SetU64(0, r0)
		if q {
			c.V[rd].SetU64(1, r1)
		} else {
			c.V[rd].SetU64(1, 0)
		}
		return
	}

	esize := uint32(8) << size
	lanes := uint32(16 >> size)
	if !q {
		lanes >>= 1
	}
	emask := elemMask(esize)

	if u && opc == 0x13 && size != 0 {
		undefined(c, insn) // PMUL is .8b/.16b only
		return
	}

	var r V128

	// Pairwise: the low half folds pairs of Vn, the high half pairs of Vm.
	if opc == 0x17 && !u || opc == 0x14 || opc == 0x15 {
		for i := uint32(0); i < lanes/2; i++ {
			n0 := velemGet(&c.V[rn], size, 2*i)
			n1 := velemGet(&c.V[rn], size, 2*i+1)
			m0 := velemGet(&c.V[rm], size, 2*i)
			m1 := velemGet(&c.V[rm], size, 2*i+1)
			var lo, hi uint64
			switch {
			case opc == 0x17: // ADDP
				lo, hi = n0+n1, m0+m1
			case opc == 0x14: // MAXP
				if u {
					lo, hi = maxU(n0, n1), maxU(m0, m1)
				} else {
					lo, hi = maxS(n0, n1, esize), maxS(m0, m1, esize)
				}
			default: // MINP
				if u {
					lo, hi = minU(n0, n1), minU(m0, m1)
				} else {
					lo, hi = minS(n0, n1, esize), minS(m0, m1, esize)
				}
			}
			velemSet(&r, size, i, lo&emask)
			velemSet(&r, size, lanes/2+i, hi&emask)
		}
		c.V[rd] = r
		return
	}

	for i := uint32(0); i < lanes; i++ {
		a := velemGet(&c.V[rn], size, i)
		b := velemGet(&c.V[rm], size, i)
		var v uint64
		key := opc
		if u {
			key |= 1 << 5
		}
		switch key {
		case 0x10: // ADD
			v = a + b
		case 0x11: // CMTST
			if a&b != 0 {
				v = emask
			}
		case 0x06: // CMGT
			if elemSx(a, esize) > elemSx(b, esize) {
				v = emask
			}
		case 0x07: // CMGE
			if elemSx(a, esize) >= elemSx(b, esize) {
				v = emask
			}
		case 0x0c: // SMAX
			v = maxS(a, b, esize)
		case 0x0d: // SMIN
			v = minS(a, b, esize)
		case 0x13: // MUL
			v = a * b
		case 0x12: // MLA
			v = velemGet(&c.V[rd], size, i) + a*b
		case 0x00: // SHADD
			v = uint64(elemSx(a, esize)+elemSx(b, esize)) >> 1
		case 0x02: // SRHADD
			v = uint64(elemSx(a, esize)+elemSx(b, esize)+1) >> 1
		case 0x04: // SHSUB
			v = uint64(elemSx(a, esize)-elemSx(b, esize)) >> 1
		case 0x01: // SQADD
			v = e.satAddS(elemSx(a, esize), elemSx(b, esize), esize)
		case 0x05: // SQSUB
			v = e.satSubS(elemSx(a, esize), elemSx(b, esize), esize)
		case 0x08: // SSHL
			v = e.vregShift(a, int(int8(b&0xff)), esize, true, false, false)
		case 0x0a: // SRSHL
			v = e.vregShift(a, int(int8(b&0xff)), esize, true, true, false)
		case 0x09: // SQSHL
			v = e.vregShift(a, int(int8(b&0xff)), esize, true, false, true)
		case 0x0b: // SQRSHL
			v = e.vregShift(a, int(int8(b&0xff)), esize, true, true, true)
		case 0x0e: // SABD
			d := elemSx(a, esize) - elemSx(b, esize)
			if d < 0 {
				v = uint64(-d)
			} else {
				v = uint64(d)
			}
		case 0x0f: // SABA
			d := elemSx(a, esize) - elemSx(b, esize)
			if d < 0 {
				v = velemGet(&c.V[rd], size, i) + uint64(-d)
			} else {
				v = velemGet(&c.V[rd], size, i) + uint64(d)
			}
		case 0x16: // SQDMULH
			p := elemSx(a, esize) * elemSx(b, esize)
			v = e.satS(p>>(esize-1), esize)

		case 0x30: // SUB
			v = a - b
		case 0x31: // CMEQ
			if a == b {
				v = emask
			}
		case 0x26: // CMHI
			if a > b {
				v = emask
			}
		case 0x27: // CMHS
			if a >= b {
				v = emask
			}
		case 0x2c: // UMAX
			v = maxU(a, b)
		case 0x2d: // UMIN
			v = minU(a, b)
		case 0x33: // PMUL
			v = uint64(pmull8(uint8(a), uint8(b))) & 0xff
		case 0x32: // MLS
			v = velemGet(&c.V[rd], size, i) - a*b
		case 0x20: // UHADD
			v = (a + b) >> 1
		case 0x22: // URHADD
			v = (a + b + 1) >> 1
		case 0x24: // UHSUB
			v = (a - b) >> 1
		case 0x21: // UQADD
			v = e.satAddU(a&emask, b&emask, esize)
		case 0x25: // UQSUB
			v = e.satSubU(a&emask, b&emask)
		case 0x28: // USHL
			v = e.vregShift(a, int(int8(b&0xff)), esize, false, false, false)
		case 0x2a: // URSHL
			v = e.vregShift(a, int(int8(b&0xff)), esize, false, true, false)
		case 0x29: // UQSHL
			v = e.vregShift(a, int(int8(b&0xff)), esize, false, false, true)
		case 0x2b: // UQRSHL
			v = e.vregShift(a, int(int8(b&0xff)), esize, false, true, true)
		case 0x2e: // UABD
			ua, ub := a&emask, b&emask
			if ua > ub {
				v = ua - ub
			} else {
				v = ub - ua
			}
		case 0x2f: // UABA
			ua, ub := a&emask, b&emask
			if ua > ub {
				v = velemGet(&c.V[rd], size, i) + ua - ub
			} else {
				v = velemGet(&c.V[rd], size, i) + ub - ua
			}
		case 0x36: // SQRDMULH
			p := elemSx(a, esize)*elemSx(b, esize) + int64(1)<<(esize-2)
			v = e.satS(p>>(esize-1), esize)

		default:
			undefined(c, insn)
			return
		}
		velemSet(&r, size, i, v&emask)
	}
	c.V[rd] = r
}

func maxS(a, b uint64, esize uint32) uint64 {
	if elemSx(a, esize) > elemSx(b, esize) {
		return a
	}
	return b
}

func minS(a, b uint64, esize uint32) uint64 {
	if elemSx(a, esize) < elemSx(b, esize) {
		return a
	}
	return b
}

func maxU(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

func minU(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

// pmull8 is a carry-less (polynomial) multiply of two bytes, the kernel of
// PMUL/PMULL: every set bit of b contributes a shifted copy of a, XORed.
func pmull8(a, b uint8) uint16 {
	var r uint16
	for i := 0; i < 8; i++ {
		if b>>i&1 != 0 {
			r ^= uint16(a) << i
		}
	}
	return r
}

// ---- modified immediate ---------------------------------------------------

func execSIMDModifiedImm(c *CPU, insn uint32) {
	q := bit(insn, 30)
	cmode := bits(insn, 15, 12)
	op := uint32(0)
	if bit(insn, 29) {
		op = 1
	}
	rd := bits(insn, 4, 0)
	imm8 := bits(insn, 18, 16)<<5 | bits(insn, 9, 5)

	hi, lo := (cmode>>1)&7, cmode&1
	if lo == 1 && hi <= 5 { // ORR/BIC (vector, immediate): they keep Vd's bits
		v := expandImm(0, cmode, imm8)
		if op == 0 {
			c.V[rd].SetU64(0, c.V[rd].U64(0)|v)
			if q {
				c.V[rd].SetU64(1, c.V[rd].U64(1)|v)
			}
		} else {
			c.V[rd].SetU64(0, c.V[rd].U64(0)&^v)
			if q {
				c.V[rd].SetU64(1, c.V[rd].U64(1)&^v)
			}
		}
		if !q {
			c.V[rd].SetU64(1, 0)
		}
		return
	}
	v := expandImm(op, cmode, imm8)
	if op != 0 && hi != 7 { // MVNI inverts, except the two MOVI/FMOV forms
		v = ^v
	}
	c.V[rd].SetU64(0, v)
	if q {
		c.V[rd].SetU64(1, v)
	} else {
		c.V[rd].SetU64(1, 0)
	}
}

// expandImm builds the 64-bit element pattern of MOVI/MVNI/ORR/BIC/FMOV
// (vector, immediate) from op, cmode and imm8.
func expandImm(op, cmode, imm8 uint32) uint64 {
	rep8 := func(b uint64) uint64 { return (b & 0xff) * 0x0101010101010101 }
	rep16 := func(h uint64) uint64 { return (h & 0xffff) * 0x0001000100010001 }
	rep32 := func(w uint64) uint64 { return (w & 0xffffffff) | (w&0xffffffff)<<32 }

	hi, lo := (cmode>>1)&7, cmode&1
	switch hi {
	case 0:
		return rep32(uint64(imm8))
	case 1:
		return rep32(uint64(imm8) << 8)
	case 2:
		return rep32(uint64(imm8) << 16)
	case 3:
		return rep32(uint64(imm8) << 24)
	case 4:
		return rep16(uint64(imm8))
	case 5:
		return rep16(uint64(imm8) << 8)
	case 6:
		if lo != 0 {
			return rep32(uint64(imm8)<<16 | 0xffff)
		}
		return rep32(uint64(imm8)<<8 | 0xff)
	}
	if lo == 0 {
		if op == 0 {
			return rep8(uint64(imm8))
		}
		var v uint64 // MOVI 64-bit: each imm8 bit becomes a byte
		for i := 0; i < 8; i++ {
			if imm8>>i&1 != 0 {
				v |= 0xff << (i * 8)
			}
		}
		return v
	}
	if op == 0 {
		return rep32(uint64(vfpImm32(imm8))) // FMOV .4S
	}
	return vfpImm64(imm8) // FMOV .2D
}

// ---- copy: DUP / INS / UMOV / SMOV ----------------------------------------

func simdCopyEncValid(op, imm4, imm5 uint32, q bool) bool {
	if imm5&0xf == 0 {
		return false // no allocated element size
	}
	size := elemSize(imm5)
	if op != 0 {
		return true // INS (element): any size
	}
	switch imm4 {
	case 0x0, 0x1:
		return !(size == 3 && !q) // DUP: the .d form needs Q
	case 0x3:
		return true // INS (general)
	case 0x5:
		if q {
			return size <= 2
		}
		return size <= 1 // SMOV: Wd{B,H} / Xd{B,H,S}
	case 0x7:
		if q {
			return size == 3
		}
		return size <= 2 // UMOV: Wd{B,H,S} / Xd{D}
	}
	return false
}

// elemSize is the element size of an AdvSIMD-copy encoding: the position of
// the lowest set bit of imm5.
func elemSize(imm5 uint32) uint32 {
	switch {
	case imm5&1 != 0:
		return 0
	case imm5&2 != 0:
		return 1
	case imm5&4 != 0:
		return 2
	}
	return 3
}

func execSIMDCopy(c *CPU, insn uint32) {
	q := bit(insn, 30)
	op := uint32(0)
	if bit(insn, 29) {
		op = 1
	}
	imm5 := bits(insn, 20, 16)
	imm4 := bits(insn, 14, 11)
	rn := bits(insn, 9, 5)
	rd := bits(insn, 4, 0)

	if !simdCopyEncValid(op, imm4, imm5, q) {
		undefined(c, insn)
		return
	}
	size := elemSize(imm5)
	index := imm5 >> (size + 1)

	if op != 0 { // INS (element): Vd[index] = Vn[idx2]
		idx2 := imm4 >> size
		velemSet(&c.V[rd], size, index, velemGet(&c.V[rn], size, idx2))
		return
	}
	switch imm4 {
	case 0x0, 0x1: // DUP (element) / DUP (general)
		var e uint64
		if imm4 == 0 {
			e = velemGet(&c.V[rn], size, index)
		} else {
			e = c.RegX(rn)
		}
		var r V128
		lanes := uint32(16 >> size)
		if !q {
			lanes >>= 1
		}
		for i := uint32(0); i < lanes; i++ {
			velemSet(&r, size, i, e)
		}
		c.V[rd] = r
	case 0x3: // INS (general): Vd[index] = Xn
		velemSet(&c.V[rd], size, index, c.RegX(rn))
	case 0x5: // SMOV
		e := velemGet(&c.V[rn], size, index)
		v := uint64(elemSx(e, 8<<size))
		if !q {
			c.SetX(rd, uint64(uint32(v)))
		} else {
			c.SetX(rd, v)
		}
	case 0x7: // UMOV
		e := velemGet(&c.V[rn], size, index)
		if !q {
			e = uint64(uint32(e))
		}
		c.SetX(rd, e)
	default:
		undefined(c, insn)
	}
}

// execSIMDScalarCopy is the one-member scalar group: DUP (element) to a
// scalar, i.e. MOV Dd, Vn.<T>[index]. It writes one element and zeroes the
// rest.
func execSIMDScalarCopy(c *CPU, insn uint32) {
	imm5 := bits(insn, 20, 16)
	rn := bits(insn, 9, 5)
	rd := bits(insn, 4, 0)
	size := elemSize(imm5)
	index := imm5 >> (size + 1)
	var r V128
	velemSet(&r, size, 0, velemGet(&c.V[rn], size, index))
	c.V[rd] = r
}

// ---- shift by immediate ---------------------------------------------------

func execSIMDShiftImm(e *fpEnv, insn uint32) {
	c := e.c
	q := bit(insn, 30)
	u := bit(insn, 29)
	immh := bits(insn, 22, 19)
	immb := bits(insn, 18, 16)
	opc := bits(insn, 15, 11)
	rn := bits(insn, 9, 5)
	rd := bits(insn, 4, 0)

	immhb := immh<<3 | immb
	size := uint32(0)
	switch {
	case immh&8 != 0:
		size = 3
	case immh&4 != 0:
		size = 2
	case immh&2 != 0:
		size = 1
	}
	esize := uint32(8) << size
	emask := elemMask(esize)

	if opc == 0x14 { // SSHLL/USHLL (widening)
		shift := immhb - esize
		n := uint32(64 / esize)
		base := uint32(0)
		if q {
			base = n
		}
		var r V128
		for i := uint32(0); i < n; i++ {
			s := velemGet(&c.V[rn], size, base+i)
			var w uint64
			if u {
				w = s << shift
			} else {
				w = uint64(elemSx(s, esize)) << shift
			}
			velemSet(&r, size+1, i, w)
		}
		c.V[rd] = r
		return
	}

	if opc >= 0x10 && opc <= 0x13 || opc == 0x1c || opc == 0x1f {
		undefined(c, insn) // narrowing shifts and fixed-point converts: on demand
		return
	}

	lanes := uint32(16 >> size)
	if !q {
		lanes >>= 1
	}
	var r V128
	for i := uint32(0); i < lanes; i++ {
		a := velemGet(&c.V[rn], size, i)
		var v uint64
		key := opc
		if u {
			key |= 1 << 5
		}
		switch key {
		case 0x0a: // SHL
			v = a << (immhb - esize)
		case 0x00: // SSHR
			v = e.vregShift(a, -int(2*esize-immhb), esize, true, false, false)
		case 0x20: // USHR
			v = e.vregShift(a, -int(2*esize-immhb), esize, false, false, false)
		case 0x2a: // SLI: keep the low bits of Vd
			sh := immhb - esize
			v = a<<sh | velemGet(&c.V[rd], size, i)&(1<<sh-1)
		case 0x28: // SRI: keep the high bits of Vd
			sh := 2*esize - immhb
			var ins uint64
			if sh < esize {
				ins = emask >> sh
			}
			var shifted uint64
			if sh < esize {
				shifted = (a & emask) >> sh
			}
			v = shifted | velemGet(&c.V[rd], size, i)&^ins
		case 0x02: // SSRA
			v = velemGet(&c.V[rd], size, i) + e.vregShift(a, -int(2*esize-immhb), esize, true, false, false)
		case 0x22: // USRA
			v = velemGet(&c.V[rd], size, i) + e.vregShift(a, -int(2*esize-immhb), esize, false, false, false)
		case 0x04: // SRSHR
			v = e.vregShift(a, -int(2*esize-immhb), esize, true, true, false)
		case 0x24: // URSHR
			v = e.vregShift(a, -int(2*esize-immhb), esize, false, true, false)
		case 0x06: // SRSRA
			v = velemGet(&c.V[rd], size, i) + e.vregShift(a, -int(2*esize-immhb), esize, true, true, false)
		case 0x26: // URSRA
			v = velemGet(&c.V[rd], size, i) + e.vregShift(a, -int(2*esize-immhb), esize, false, true, false)
		case 0x0e: // SQSHL
			v = e.vregShift(a, int(immhb-esize), esize, true, false, true)
		case 0x2e: // UQSHL
			v = e.vregShift(a, int(immhb-esize), esize, false, false, true)
		case 0x2c: // SQSHLU (signed -> unsigned)
			v = sqshlu(e, elemSx(a, esize), int(immhb-esize), emask)
		default:
			undefined(c, insn)
			return
		}
		velemSet(&r, size, i, v&emask)
	}
	c.V[rd] = r
}

// sqshlu is SQSHLU: saturating left shift of a signed value into an unsigned
// element, so anything below zero clamps to zero.
func sqshlu(e *fpEnv, v int64, sh int, emask uint64) uint64 {
	if v <= 0 {
		if v < 0 {
			e.exc |= fpsrQC
		}
		return 0
	}
	if sh >= 64 || uint64(v) > emask>>uint(sh) {
		e.exc |= fpsrQC
		return emask
	}
	return uint64(v) << uint(sh) & emask
}

// vfpImm32/vfpImm64 are the bit patterns of the FP immediate that FMOV
// (vector) broadcasts, and that the scalar FMOV expands in fpsimd.go.
func vfpImm32(imm8 uint32) uint32 {
	s := (imm8 >> 7) & 1
	b6 := (imm8 >> 6) & 1
	e := (imm8 >> 4) & 3
	r := (1 - b6) << 7
	if b6 != 0 {
		r |= 0x1f << 2
	}
	r |= e
	return uint32(s)<<31 | uint32(r)<<23 | uint32(imm8&0xf)<<19
}

func vfpImm64(imm8 uint32) uint64 {
	s := (imm8 >> 7) & 1
	b6 := (imm8 >> 6) & 1
	e := (imm8 >> 4) & 3
	r := (1 - b6) << 10
	if b6 != 0 {
		r |= 0xff << 2
	}
	r |= e
	return uint64(s)<<63 | uint64(r)<<52 | uint64(imm8&0xf)<<48
}

// intMaxMin is the signed range of an element: -2^(e-1) .. 2^(e-1)-1, written
// without a literal that would overflow uint64.
func intMaxMin(esize uint32) (max, min int64) {
	if esize >= 64 {
		return int64(^uint64(0) >> 1), int64(-1) << 63
	}
	h := int64(1) << (esize - 1)
	return h - 1, -h
}

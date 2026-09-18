// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Half-precision (FP16) — the Go port of the f16 half of
// src/core/exec_fpsimd.c.
//
// Half-precision values are held in the low 16 bits of a SIMD register and are
// never computed in: every arithmetic operation widens to double, computes, and
// narrows once. That is exact because double's 53-bit fraction exceeds
// 2*11 + 2, so there is no double rounding for any of the operations here —
// which is why the C can do it, and why this file does not need a 16-bit
// arithmetic unit.
//
// Two details decide whether the result is right:
//
//	Widening goes half -> double directly, not half -> float -> double. A
//	(float) cast quiets a signaling NaN, after which the NaN ranking, the
//	comparisons' signaling detection and the fused multiply-add's quiet-
//	addend rule all see the wrong class of NaN.
//
//	Widening for arithmetic (f16ToF64) applies FPCR.FZ16 and is the only
//	place that does; the pure conversion (fcvtH2S) does not, because
//	FPUnpackCV and FPRoundCV both clear FZ16 before doing anything.
//	FZ16 also raises no Input Denormal: the architecture carves the
//	half-precision flush out of IDC.

package core

import "math"

// f16ToF64Raw is the pure unpack: exact for every half value, and NaNs keep
// both their payload and their signaling-ness (the half fraction lands on
// double bits 51:42, so the half quiet bit maps onto the double quiet bit).
func f16ToF64Raw(h uint16) float64 {
	sign := uint64(h&0x8000) << 48
	exp := uint64(h>>10) & 0x1f
	mant := uint64(h) & 0x3ff
	var bits uint64
	switch {
	case exp == 0x1f: // Inf / NaN: the payload shifts up 42
		bits = sign | 0x7ff0000000000000 | mant<<42
	case exp == 0:
		if mant == 0 {
			bits = sign // +/- zero
		} else { // subnormal half -> normal double
			e := uint64(1023 - 15 + 1)
			for mant&0x400 == 0 {
				mant <<= 1
				e--
			}
			bits = sign | e<<52 | (mant&0x3ff)<<42
		}
	default: // normal: rebias the exponent by +1008
		bits = sign | (exp-15+1023)<<52 | mant<<42
	}
	return math.Float64frombits(bits)
}

// f16ToF64 is the unpacking form, the one that reads a half as a number and
// therefore honours FPCR.FZ16.
func (e *fpEnv) f16ToF64(h uint16) float64 {
	if e.fz16 && h&0x7c00 == 0 && h&0x3ff != 0 {
		h &= 0x8000
	}
	return f16ToF64Raw(h)
}

// rdH reads a half-precision operand without flushing it: the sign-rewriting
// operations (FMOV, FABS, FNEG) unpack nothing.
func (e *fpEnv) rdH(n uint32) uint16 { return e.c.V[n].U16(0) }

func (e *fpEnv) wrH(d uint32, h uint16) {
	e.c.V[d].SetU64(0, uint64(h))
	e.c.V[d].SetU64(1, 0)
}

// f16ToF32Raw is the same unpack one level down, for FCVT to single.
func f16ToF32Raw(h uint16) uint32 {
	sign := uint32(h&0x8000) << 16
	exp := uint32(h>>10) & 0x1f
	mant := uint32(h) & 0x3ff
	var bits uint32
	switch {
	case exp == 0x1f: // Inf / NaN: the payload shifts up 13
		bits = sign | 0x7f800000 | mant<<13
	case exp == 0:
		if mant == 0 {
			bits = sign
		} else {
			e := uint32(127 - 15 + 1)
			for mant&0x400 == 0 {
				mant <<= 1
				e--
			}
			bits = sign | e<<23 | (mant&0x3ff)<<13
		}
	default:
		bits = sign | (exp-15+127)<<23 | mant<<13
	}
	return bits
}

// fcvtH2S is FCVT from half as a *conversion*: a signaling NaN raises Invalid
// Operation and the result NaN is quieted (FPConvertNaN). It takes the raw
// widen — flush-to-zero does not reach the half side of a precision change.
func (e *fpEnv) fcvtH2S(h uint16) float32 {
	b := f16ToF32Raw(h)
	if h&0x7c00 == 0x7c00 && h&0x3ff != 0 {
		if h&0x200 == 0 {
			e.exc |= fpsrIOC
		}
		b |= 0x400000
	}
	return math.Float32frombits(b)
}

// f64ToF16 is binary64 -> binary16 with round-to-nearest-even and the
// conversion's flags: signaling NaN -> IOC, overflow -> OFC|IXC (including a
// round-up into infinity), a tiny or inexact result -> UFC|IXC, plain inexact
// -> IXC. `fz` is FPCR.FZ16, and only UFC is raised when it flushes: the
// discarded rounding reports nothing.
func (e *fpEnv) f64ToF16(x float64) uint16    { return e.f64ToF16Round(x, e.fz16) }
func (e *fpEnv) f64ToF16Raw(x float64) uint16 { return e.f64ToF16Round(x, false) }

func (e *fpEnv) f64ToF16Round(x float64, fz bool) uint16 {
	b := math.Float64bits(x)
	sign := uint16(b>>48) & 0x8000
	mag := b & 0x7fffffffffffffff

	if mag >= 0x7ff0000000000000 {
		if mag > 0x7ff0000000000000 { // NaN: FPConvertNaN
			if mag&0x8000000000000 == 0 {
				e.exc |= fpsrIOC
			}
			// The top payload bits carry over (frac 50:42 -> half 8:0) with
			// the quiet bit forced, so a propagated NaN operand comes back
			// unchanged instead of flattened to the default.
			return sign | 0x7e00 | uint16(mag>>42)&0x1ff
		}
		return sign | 0x7c00
	}
	if mag == 0 {
		return sign
	}

	he := int(mag>>52) - 1023 + 15 // tentative half exponent
	sig := mag&0xfffffffffffff | 0x10000000000000

	if he >= 0x1f { // overflow -> infinity
		e.exc |= fpsrOFC | fpsrIXC
		return sign | 0x7c00
	}
	if fz && he <= 0 { // FPCR.FZ16
		e.exc |= fpsrUFC
		return sign
	}

	shift := 42 // 52 - 10 fraction bits
	if he <= 0 {
		shift = 43 - he  // subnormal: an extra right shift
		if shift >= 54 { // below half an ULP -> +/-0
			e.exc |= fpsrUFC | fpsrIXC
			return sign
		}
	}
	rounded := sig >> shift
	rem := sig & (1<<shift - 1)
	half := uint64(1) << (shift - 1)
	if rem > half || rem == half && rounded&1 != 0 {
		rounded++ // nearest, ties to even
	}
	if rem != 0 {
		if he > 0 {
			e.exc |= fpsrIXC
		} else {
			e.exc |= fpsrUFC | fpsrIXC
		}
	}
	var res uint16
	if he > 0 { // normal: a carry bumps the exponent
		// `rounded` carries the leading fraction bit at bit 10, so ADDING it
		// to the exponent field -- not OR-ing it in -- is what lets a fraction
		// that rounded up carry into the exponent.
		res = sign | uint16((he-1)<<10) + uint16(rounded)
	} else {
		res = sign | uint16(rounded) // subnormal (a carry -> normal)
	}
	if res&0x7c00 == 0x7c00 { // the round-up crossed into infinity
		e.exc |= fpsrOFC | fpsrIXC
	}
	return res
}

// vfpImm16 is the half-precision form of the FP immediate (FMOV Hd, #imm).
func vfpImm16(imm8 uint32) uint16 {
	s := (imm8 >> 7) & 1
	b6 := (imm8 >> 6) & 1
	e := (imm8 >> 4) & 3
	r := (1 - b6) << 4
	if b6 != 0 {
		r |= 3 << 2
	}
	r |= e
	return uint16(s)<<15 | uint16(r)<<10 | uint16(imm8&0xf)<<6
}

// execFPScalarH is the half-precision (ftype=3) half of exec_fp_scalar: the
// convert, 1-source, 2-source, compare, select and immediate groups, all
// computing in double and narrowing once. It reports whether it handled the
// instruction.
func execFPScalarH(e *fpEnv, insn uint32) bool {
	c := e.c
	rn := bits(insn, 9, 5)
	rd := bits(insn, 4, 0)

	// FCVT widening from half and narrowing to half. This encoding page is
	// shared with the half-precision 1-source group (FSQRT, FRINT*, FMOV):
	// bits(14,10)==0x10 selects the page, and only these three opcodes on it
	// are converts, so anything else falls through to the group below.
	if bit(insn, 21) && bits(insn, 14, 10) == 0x10 {
		switch opc := bits(insn, 20, 15); {
		case opc == 0x4 && bits(insn, 23, 22) == 3: // FCVT Sd, Hn
			e.wrS(rd, float64(e.fcvtH2S(e.rdH(rn))))
			return true
		case opc == 0x5 && bits(insn, 23, 22) == 3: // FCVT Dd, Hn
			e.wrD(rd, float64(e.fcvtH2S(e.rdH(rn))))
			return true
		case opc == 0x7 && bits(insn, 23, 22) == 0: // FCVT Hd, Sn
			e.wrH(rd, e.f64ToF16Raw(float64(math.Float32frombits(c.V[rn].U32(0)))))
			return true
		case opc == 0x7 && bits(insn, 23, 22) == 1: // FCVT Hd, Dn
			e.wrH(rd, e.f64ToF16Raw(e.flushD(math.Float64frombits(c.V[rn].U64(0)))))
			return true
		}
	}

	if bits(insn, 23, 22) != 3 {
		// Everything below is half-only. The FCVT block above is the one
		// piece of this page that the single and double forms share, which is
		// why it is checked before this.
		return false
	}

	// FMOV between a general register and a half register: a raw 16-bit move,
	// in the FP<->integer space (bits(15,10)==0).
	if bit(insn, 21) && bits(insn, 15, 10) == 0 && bits(insn, 20, 19) == 0 {
		switch bits(insn, 18, 16) {
		case 6: // FMOV Wd, Hn
			c.SetX(rd, uint64(c.V[rn].U16(0)))
			return true
		case 7: // FMOV Hd, Wn
			e.wrH(rd, uint16(c.RegX(rn)))
			return true
		}
	}

	// The data-processing page: immediate, compare, 1-source, 2-source,
	// FCSEL and FCCMP.
	if bit(insn, 21) && bits(insn, 15, 10) != 0 {
		switch o2 := bits(insn, 11, 10); o2 {
		case 0:
			switch {
			case bit(insn, 12): // FMOV Hd, #imm
				e.wrH(rd, vfpImm16(bits(insn, 20, 13)))
				return true
			case bit(insn, 13): // FCMP / FCMPE (with 0.0)
				rm := bits(insn, 20, 16)
				a := e.f16ToF64(e.rdH(rn))
				b := float64(0)
				if !bit(insn, 3) {
					b = e.f16ToF64(e.rdH(rm))
				}
				c.fpSetFlags(e.compare(a, b, bit(insn, 4)))
				return true
			case bit(insn, 14): // 1 source
				hn := e.rdH(rn)
				var r float64
				switch opc := bits(insn, 20, 15); opc {
				// FMOV/FABS/FNEG rewrite a sign bit: they unpack nothing, do
				// not round, do not flush, and leave a signaling NaN
				// signaling.
				case 0x0:
					e.wrH(rd, hn)
					return true
				case 0x1:
					e.wrH(rd, hn&0x7fff)
					return true
				case 0x2:
					e.wrH(rd, hn^0x8000)
					return true
				case 0x3:
					x := e.f16ToF64(hn)
					if x < 0 && x != 0 {
						e.exc |= fpsrIOC
					}
					r = e.propagateNaN(math.Sqrt(x), x, x)
				case 0x8, 0x9, 0xa, 0xb, 0xc:
					r = e.frint(e.f16ToF64(hn), int(opc-0x8), false)
				case 0xe, 0xf: // FRINTX / FRINTI use FPCR.RMode
					r = e.frint(e.f16ToF64(hn), c.fpRoundMode(), opc == 0xe)
				default:
					return false
				}
				e.wrH(rd, e.f64ToF16(r))
				return true
			}
		case 2: // 2 source
			rm := bits(insn, 20, 16)
			opc := bits(insn, 15, 12)
			if opc > 0x8 {
				return false
			}
			a := e.f16ToF64(e.rdH(rn))
			b := e.f16ToF64(e.rdH(rm))
			var r float64
			switch opc {
			case 0x0:
				r = e.propagateNaN(a*b, a, b) // FMUL
			case 0x1:
				r = e.propagateNaN(a/b, a, b) // FDIV
			case 0x2:
				r = e.propagateNaN(a+b, a, b) // FADD
			case 0x3:
				r = e.propagateNaN(a-b, a, b) // FSUB
			case 0x4:
				r = fopMaxMin(e, 0, a, b) // FMAX
			case 0x5:
				r = fopMaxMin(e, 1, a, b) // FMIN
			case 0x6:
				r = fopMaxMinNM(e, 0, a, b) // FMAXNM
			case 0x7:
				r = fopMaxMinNM(e, 1, a, b) // FMINNM
			case 0x8:
				r = -e.propagateNaN(a*b, a, b) // FNMUL
			}
			e.wrH(rd, e.f64ToF16(r))
			return true
		case 3: // FCSEL
			rm := bits(insn, 20, 16)
			src := rn
			if !c.CondHolds(bits(insn, 15, 12)) {
				src = rm
			}
			e.wrH(rd, e.rdH(src))
			return true
		case 1: // FCCMP / FCCMPE
			rm := bits(insn, 20, 16)
			if c.CondHolds(bits(insn, 15, 12)) {
				c.fpSetFlags(e.compare(e.f16ToF64(e.rdH(rn)), e.f16ToF64(e.rdH(rm)), bit(insn, 4)))
			} else {
				nzcv := bits(insn, 3, 0)
				var f uint32
				if nzcv&8 != 0 {
					f |= PSN
				}
				if nzcv&4 != 0 {
					f |= PSZ
				}
				if nzcv&2 != 0 {
					f |= PSC
				}
				if nzcv&1 != 0 {
					f |= PSV
				}
				c.NZCV = f
			}
			return true
		}
		return false
	}

	// Half-precision FP<->integer: SCVTF/UCVTF and FCVT{N,P,M,Z}{S,U}.
	if bit(insn, 21) && bits(insn, 15, 10) == 0 && bits(insn, 28, 24) == 0x1e {
		sf := bit(insn, 31)
		rmode := bits(insn, 20, 19)
		opcode := bits(insn, 18, 16)
		switch {
		case opcode == 2 || opcode == 3: // SCVTF / UCVTF: int -> half
			v := c.RegX(rn)
			var iv float64
			if opcode == 2 {
				if sf {
					iv = float64(int64(v))
				} else {
					iv = float64(int32(v))
				}
			} else if sf {
				iv = float64(v)
			} else {
				iv = float64(uint32(v))
			}
			e.wrH(rd, e.f64ToF16(iv))
			return true
		case opcode <= 1 || opcode == 4 || opcode == 5: // FCVT to int
			x0 := e.f16ToF64(e.rdH(rn))
			rm := int(rmode)
			if opcode >= 4 {
				rm = 3 // FCVTZ
			}
			c.SetX(rd, e.cvtToInt(x0, rm, opcode&1 == 0, sf))
			return true
		}
	}
	return false
}

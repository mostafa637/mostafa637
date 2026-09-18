// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// FP/SIMD execution — the Go port of src/core/exec_fpsimd.c.
//
// The C file is 4500 lines and owns the whole FP/Advanced-SIMD/crypto
// instruction space. It is being ported a family at a time: this file holds
// the scalar floating-point data-processing instructions (single and double
// precision), which is the part every FP-using program actually executes.
// Half-precision (ftype=3), the Advanced SIMD vector groups and the crypto
// extensions stay UNDEFINED for now, and the ID registers in sysreg.go keep
// advertising only what is here.
//
// One structural difference from the C is worth knowing before reading the
// exception handling: C reads the host FPU's sticky flags with fetestexcept
// to decide Inexact, Underflow and Overflow, and mirrors FPCR.FZ into the
// host's own control word. Go has no fenv. So this file detects what it can
// from the values themselves — Overflow and Division-by-Zero from the
// operands and result, Inexact for single-precision results (computed in
// double, where the exact answer fits) — and leaves Inexact for
// double-precision results unreported rather than guessing. FPSR is
// observationally correct for the common cases, not bit-exact.

package core

import "math"

// FPSR cumulative exception bits (ARM D13.2.59).
const (
	fpsrIOC = 1 << 0 // invalid operation
	fpsrDZC = 1 << 1 // divide by zero
	fpsrOFC = 1 << 2 // overflow
	fpsrUFC = 1 << 3 // underflow
	fpsrIXC = 1 << 4 // inexact
	fpsrIDC = 1 << 7 // input denormal
)

// FPCR fields this file reads (ARM D13.2.58).
const (
	fpcrFZ    = 1 << 24 // flush denormals to zero
	fpcrRMode = 3 << 22 // rounding mode
	fpcrFZ16  = 1 << 19 // flush denormal half-precision to zero
)

// fpEnv is the state of the instruction in flight. The C keeps FZ, FZ16 and
// the pending exception word in thread-locals latched by exec_fpsimd; here it
// is a value passed along, which is also what keeps the emulator's FP state
// per-Task without any globals.
type fpEnv struct {
	c    *CPU
	fz   bool
	fz16 bool
	exc  uint32
}

// sync folds the instruction's exceptions into FPSR, the way fpsr_sync does.
func (e *fpEnv) sync() {
	e.c.FPSR |= e.exc
	e.exc = 0
}

// ---- operands and results -------------------------------------------------

// rdS/rdD read a scalar register as a double, flushing a denormal operand to
// zero when FPCR.FZ is set (FPUnpack, raising IDC). Every arithmetic operand
// goes through one of these; FMOV, FABS, FNEG and FCSEL do not, because they
// never unpack their operand.
func (e *fpEnv) rdS(n uint32) float64 {
	return float64(e.flushS(math.Float32frombits(e.c.V[n].U32(0))))
}

func (e *fpEnv) rdD(n uint32) float64 {
	return e.flushD(math.Float64frombits(e.c.V[n].U64(0)))
}

// wrS/wrD write a scalar result, clearing the register's upper 64 bits the way
// the architecture's "scalar" accessors do, and flushing a tiny result to zero
// (FPRoundBase, raising UFC — and only UFC: a value replaced by zero does not
// report the inexactness of the rounding that no longer happens).
func (e *fpEnv) wrS(d uint32, x float64) {
	e.c.V[d].SetU64(0, uint64(math.Float32bits(e.outS(float32(x)))))
	e.c.V[d].SetU64(1, 0)
}

func (e *fpEnv) wrD(d uint32, x float64) {
	e.c.V[d].SetU64(0, math.Float64bits(e.outD(x)))
	e.c.V[d].SetU64(1, 0)
}

func (e *fpEnv) flushS(x float32) float32 {
	b := math.Float32bits(x)
	if !e.fz || b&0x7f800000 != 0 || b&0x007fffff == 0 {
		return x
	}
	e.exc |= fpsrIDC
	return math.Float32frombits(b & 0x80000000)
}

func (e *fpEnv) flushD(x float64) float64 {
	b := math.Float64bits(x)
	if !e.fz || b&0x7ff0000000000000 != 0 || b&0x000fffffffffffff == 0 {
		return x
	}
	e.exc |= fpsrIDC
	return math.Float64frombits(b & 0x8000000000000000)
}

// outS/outD decide whether the result is *tiny*: a value whose magnitude is
// below the smallest normal and that is not that boundary itself. A result
// that rounded UP to the boundary is not tiny and is left alone.
func (e *fpEnv) outS(x float32) float32 {
	b := math.Float32bits(x)
	mag := b & 0x7fffffff
	if !e.fz || mag == 0 || mag >= 0x00800000 {
		return x
	}
	e.exc |= fpsrUFC
	return math.Float32frombits(b & 0x80000000)
}

func (e *fpEnv) outD(x float64) float64 {
	b := math.Float64bits(x)
	mag := b & 0x7fffffffffffffff
	if !e.fz || mag == 0 || mag >= 0x0010000000000000 {
		return x
	}
	e.exc |= fpsrUFC
	return math.Float64frombits(b & 0x8000000000000000)
}

// ---- NaN handling ---------------------------------------------------------

func isSNaN64(x float64) bool {
	b := math.Float64bits(x)
	return b&0x7ff0000000000000 == 0x7ff0000000000000 && b&0x000fffffffffffff != 0 &&
		b&0x0008000000000000 == 0
}

func quiet64(x float64) float64 {
	return math.Float64frombits(math.Float64bits(x) | 0x0008000000000000)
}

// propagateNaN is FPProcessNaNs: if the computed result is a NaN, the NaN that
// came from an operand wins — the signaling one first, then the quiet one —
// and a signaling input raises Invalid Operation. With no NaN input the result
// is the default NaN.
func (e *fpEnv) propagateNaN(r, a, b float64) float64 {
	if !math.IsNaN(r) {
		return r
	}
	if isSNaN64(a) || isSNaN64(b) {
		e.exc |= fpsrIOC
	}
	if isSNaN64(a) {
		return quiet64(a)
	}
	if isSNaN64(b) {
		return quiet64(b)
	}
	if math.IsNaN(a) {
		return a
	}
	if math.IsNaN(b) {
		return b
	}
	return math.NaN()
}

func (e *fpEnv) propagateNaN3(r, a, n, m float64) float64 {
	if !math.IsNaN(r) {
		return r
	}
	for _, x := range [...]float64{a, n, m} {
		if isSNaN64(x) {
			e.exc |= fpsrIOC
			return quiet64(x)
		}
	}
	for _, x := range [...]float64{a, n, m} {
		if math.IsNaN(x) {
			return x
		}
	}
	return math.NaN()
}

// ---- arithmetic -----------------------------------------------------------

// raise records the exceptions the operation can be seen to raise: Divide by
// Zero, Overflow, and the Invalid Operation of 0*inf, inf-inf and 0/0. Inexact
// is raised where it can be observed, which is single precision: the value is
// computed in double, where the exact answer fits.
func (e *fpEnv) raise(op int, a, b, r float64) {
	switch op {
	case opDiv:
		if b == 0 && a != 0 && !math.IsNaN(a) && !math.IsInf(a, 0) {
			e.exc |= fpsrDZC
		}
		if a == 0 && b == 0 || math.IsInf(a, 0) && math.IsInf(b, 0) {
			e.exc |= fpsrIOC
		}
	case opMul:
		if a == 0 && math.IsInf(b, 0) || math.IsInf(a, 0) && b == 0 {
			e.exc |= fpsrIOC
		}
	case opAdd, opSub:
		if math.IsInf(a, 0) && math.IsInf(b, 0) && (a > 0) != (b > 0) == (op == opAdd) {
			e.exc |= fpsrIOC
		}
	}
	if math.IsInf(r, 0) && !math.IsInf(a, 0) && !math.IsInf(b, 0) {
		e.exc |= fpsrOFC
	}
}

const (
	opMul = iota
	opDiv
	opAdd
	opSub
	opMax
	opMin
	opMaxNM
	opMinNM
)

// arith is the two-source FP data-processing group: FMUL, FDIV, FADD, FSUB,
// FMAX, FMIN, FMAXNM, FMINNM (FNMUL is this with the result negated), and the
// operand-order rules that decide which NaN comes back.
func (e *fpEnv) arith(op int, a, b float64) float64 {
	switch op {
	case opMax, opMin:
		if math.IsNaN(a) || math.IsNaN(b) {
			// The NaN that was an operand is the answer: the first one that
			// is a NaN, quieted by propagateNaN.
			r := a
			if !math.IsNaN(r) {
				r = b
			}
			return e.propagateNaN(r, a, b)
		}
		if a == b { // +0.0 is greater than -0.0
			if math.Signbit(a) {
				return b
			}
			return a
		}
		if op == opMax {
			if a > b {
				return a
			}
			return b
		}
		if a < b {
			return a
		}
		return b

	case opMaxNM, opMinNM:
		// A NaN operand is treated as missing: the other one is the answer.
		// Two NaNs, or a signaling NaN anywhere, go through the usual
		// propagation.
		if math.IsNaN(a) && math.IsNaN(b) {
			return e.propagateNaN(a, a, b)
		}
		if math.IsNaN(a) {
			if isSNaN64(a) {
				e.exc |= fpsrIOC
			}
			return b
		}
		if math.IsNaN(b) {
			if isSNaN64(b) {
				e.exc |= fpsrIOC
			}
			return a
		}
		if a == b {
			if math.Signbit(a) {
				return b
			}
			return a
		}
		if op == opMaxNM {
			if a > b {
				return a
			}
			return b
		}
		if a < b {
			return a
		}
		return b
	}

	var r float64
	switch op {
	case opMul:
		r = a * b
	case opDiv:
		r = a / b
	case opAdd:
		r = a + b
	case opSub:
		r = a - b
	}
	e.raise(op, a, b, r)
	return e.propagateNaN(r, a, b)
}

// mulAdd is the fused three-source group (FMADD/FMSUB/FNMADD/FNMSUB). The
// signs of the addend and of the first multiplicand that each form applies are
// folded into the operands before the NaN processing, because FMSUB of a NaN
// multiplicand returns it negated.
func (e *fpEnv) mulAdd(a, n, m float64) float64 {
	r := math.FMA(n, m, a)
	if math.IsNaN(a) && !isSNaN64(a) &&
		((math.IsInf(n, 0) && m == 0) || (n == 0 && math.IsInf(m, 0))) {
		e.exc |= fpsrIOC
		return math.NaN()
	}
	return e.propagateNaN3(r, a, n, m)
}

// ---- rounding -------------------------------------------------------------

// frint rounds to an integral value. rm: 0 nearest ties-to-even, 1 towards
// +inf, 2 towards -inf, 3 towards zero, 4 nearest ties-away. `exact` is
// FRINTX, which reports Inexact when the value moved; FRINTI does not.
func (e *fpEnv) frint(x float64, rm int, exact bool) float64 {
	var r float64
	switch rm {
	case 0:
		r = math.RoundToEven(x)
	case 1:
		r = math.Ceil(x)
	case 2:
		r = math.Floor(x)
	case 3:
		r = math.Trunc(x)
	case 4:
		r = math.Round(x)
	}
	if exact && r != x && !math.IsNaN(x) && !math.IsInf(x, 0) {
		e.exc |= fpsrIXC
	}
	return r
}

// roundMode maps FPCR.RMode onto the frint modes. RMode 2 and 3 are both
// "towards zero" for the rounding-mode field (3 is reserved but reads as 0).
func (c *CPU) fpRoundMode() int {
	switch (c.FPCR >> 22) & 3 {
	case 1:
		return 1
	case 2:
		return 2
	case 3:
		return 3
	}
	return 0
}

// cvtToInt is FPToInt: round with the instruction's mode, then saturate to the
// destination's range. NaN and anything out of range raise Invalid Operation
// and saturate (NaN to zero).
func (e *fpEnv) cvtToInt(x float64, rm int, signed, x64 bool) uint64 {
	if math.IsNaN(x) {
		e.exc |= fpsrIOC
		return 0
	}
	r := e.frint(x, rm, false)

	lo, hi := -2147483648.0, 2147483647.0
	umax := 4294967295.0
	if x64 {
		lo, hi = -9223372036854775808.0, 9223372036854775807.0
		umax = 18446744073709551616.0
	}
	if signed {
		switch {
		case r < lo:
			e.exc |= fpsrIOC
			return uint64(int64(lo))
		case r > hi:
			e.exc |= fpsrIOC
			return uint64(int64(hi))
		}
		return uint64(int64(r))
	}
	switch {
	case r < -1.0, r > umax:
		e.exc |= fpsrIOC
		if r < 0 {
			return 0
		}
		return uint64(umax)
	}
	return uint64(int64(r))
}

func (e *fpEnv) cvtFromInt(v uint64, unsigned, x64 bool) float64 {
	if unsigned {
		if !x64 {
			v = uint64(uint32(v))
		}
		return float64(v)
	}
	if x64 {
		return float64(int64(v))
	}
	return float64(int32(v))
}

// ---- comparison -----------------------------------------------------------

// compare is FPCompare: -1 less than, 0 equal, 1 greater than, 2 unordered. A
// quiet NaN is only an error for the signaling comparisons (FCMPE/FCCMPE).
func (e *fpEnv) compare(a, b float64, signaling bool) int {
	if math.IsNaN(a) || math.IsNaN(b) {
		if signaling || isSNaN64(a) || isSNaN64(b) {
			e.exc |= fpsrIOC
		}
		return 2
	}
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// setFlags writes the comparison into PSTATE.NZCV the way FPCompare does:
// less than N, equal Z|C, greater than C, unordered C|V.
func (c *CPU) fpSetFlags(cmp int) {
	switch cmp {
	case -1:
		c.NZCV = PSN
	case 0:
		c.NZCV = PSZ | PSC
	case 1:
		c.NZCV = PSC
	default:
		c.NZCV = PSC | PSV
	}
}

// ---- execution ------------------------------------------------------------

// FPSIMDExec, when set, executes the FP/SIMD instruction in `insn` (the
// 0x7/0xf top-level decode groups). It is a variable rather than a direct call
// so the core links and tests without the FP unit, exactly as the C core keeps
// exec_fpsimd weak.
var FPSIMDExec func(c *CPU, insn uint32)

func init() { FPSIMDExec = execFPSIMD }

func execFPSIMD(c *CPU, insn uint32) {
	// FZ and FZ16 are latched for the instruction in flight: they decide a
	// *result*, so they cannot be re-read halfway through one.
	e := fpEnv{c: c, fz: c.FPCR&fpcrFZ != 0, fz16: c.FPCR&fpcrFZ16 != 0}
	defer e.sync()

	switch {
	case insn&0x7f000000 == 0x1e000000:
		execFPScalar(&e, insn)
	case insn&0x7f000000 == 0x1f000000:
		execFPDP3(&e, insn)

	// The Advanced SIMD groups, in the order src/core/exec_fpsimd.c's
	// dispatch tests them. Each predicate is the encoding's own: the ones
	// that are not ported fall through to UNDEFINED rather than being
	// caught by a later, looser test.
	case !bit(insn, 31) && bits(insn, 28, 24) == 0x0e && bit(insn, 21) && bit(insn, 10):
		execSIMDThreeSame(&e, insn) // three-same (integer)
	case !bit(insn, 31) && bits(insn, 28, 19) == 0x1e0 && bit(insn, 10):
		execSIMDModifiedImm(c, insn) // MOVI/MVNI/ORR/BIC/FMOV (immediate)
	case !bit(insn, 31) && bits(insn, 28, 23) == 0x1e && bit(insn, 10) && bits(insn, 22, 19) != 0:
		execSIMDShiftImm(&e, insn) // shift by immediate
	case !bit(insn, 31) && bits(insn, 28, 21) == 0x70 && !bit(insn, 15) && bit(insn, 10):
		execSIMDCopy(c, insn) // DUP/INS/UMOV/SMOV
	case bits(insn, 31, 21) == 0x2f0 && !bit(insn, 15) && bits(insn, 14, 11) == 0 && bit(insn, 10):
		execSIMDScalarCopy(c, insn) // MOV Dd, Vn.<T>[index]
	default:
		undefined(c, insn)
	}
}

func execFPScalar(e *fpEnv, insn uint32) {
	c := e.c
	ftype := bits(insn, 23, 22)
	rn := bits(insn, 9, 5)
	rd := bits(insn, 4, 0)
	dbl := ftype == 1

	// FMOV to and from the high 64 bits of a SIMD register (the 128-bit
	// variant, ptype=10 — not half-precision). It sits in the FP/integer
	// conversion space, so it is caught before the ftype check below.
	if bits(insn, 28, 24) == 0x1e && bit(insn, 21) && bits(insn, 15, 10) == 0 &&
		bit(insn, 31) && ftype == 2 && bits(insn, 20, 19) == 1 {
		switch bits(insn, 18, 16) {
		case 6:
			c.SetX(rd, c.V[rn].U64(1)) // FMOV Xd, Vn.D[1]
		case 7:
			c.V[rd].SetU64(1, c.RegX(rn)) // FMOV Vd.D[1], Xn
		default:
			undefined(c, insn)
		}
		return
	}

	if ftype != 0 && ftype != 1 {
		undefined(c, insn) // half-precision: on demand
		return
	}

	// Floating-point <-> integer conversion (bit21 distinguishes it from the
	// fixed-point forms, which are not ported).
	if bits(insn, 28, 24) == 0x1e && bit(insn, 21) && bits(insn, 15, 10) == 0 {
		sf := bit(insn, 31)
		rmode := bits(insn, 20, 19)
		opcode := bits(insn, 18, 16)
		if dbl {
			switch rmode<<3 | opcode {
			case 0<<3 | 2: // SCVTF
				e.wrD(rd, e.cvtFromInt(c.RegX(rn), false, sf))
			case 0<<3 | 3: // UCVTF
				e.wrD(rd, e.cvtFromInt(c.RegX(rn), true, sf))
			case 0<<3 | 6: // FMOV (fp -> general)
				c.SetX(rd, c.V[rn].U64(0))
			case 0<<3 | 7: // FMOV (general -> fp)
				c.V[rd].SetU64(0, c.RegX(rn))
				c.V[rd].SetU64(1, 0)
			default:
				if rmode > 3 || opcode > 1 {
					undefined(c, insn)
					return
				}
				// FCVT{N,P,M,Z}{S,U}: fp -> int
				c.SetX(rd, e.cvtToInt(e.rdD(rn), int(rmode), opcode == 0, sf))
			}
			return
		}
		switch rmode<<3 | opcode {
		case 0<<3 | 2: // SCVTF
			e.wrS(rd, e.cvtFromInt(c.RegX(rn), false, sf))
		case 0<<3 | 3: // UCVTF
			e.wrS(rd, e.cvtFromInt(c.RegX(rn), true, sf))
		case 0<<3 | 6: // FMOV (fp -> general, 32-bit)
			c.SetX(rd, uint64(c.V[rn].U32(0)))
		case 0<<3 | 7: // FMOV (general -> fp, 32-bit)
			c.V[rd].SetU64(0, uint64(uint32(c.RegX(rn))))
			c.V[rd].SetU64(1, 0)
		default:
			if rmode > 3 || opcode > 1 {
				undefined(c, insn)
				return
			}
			c.SetX(rd, e.cvtToInt(float64(math.Float32frombits(c.V[rn].U32(0))), int(rmode), opcode == 0, sf))
		}
		return
	}

	switch o2 := bits(insn, 11, 10); o2 {
	case 0:
		switch {
		case bit(insn, 12): // FP immediate: FMOV Sd/Dd, #imm
			e.wrD(rd, vfpImm(bits(insn, 20, 13), true))
			if !dbl {
				e.wrS(rd, vfpImm(bits(insn, 20, 13), false))
			}
		case bit(insn, 13): // FP compare
			rm := bits(insn, 20, 16)
			withZero := bit(insn, 3)
			var cmp int
			if dbl {
				b := float64(0)
				if !withZero {
					b = e.rdD(rm)
				}
				cmp = e.compare(e.rdD(rn), b, bit(insn, 4))
			} else {
				b := float64(0)
				if !withZero {
					b = e.rdS(rm)
				}
				cmp = e.compare(e.rdS(rn), b, bit(insn, 4))
			}
			c.fpSetFlags(cmp)
		case bit(insn, 14): // FP data-processing (1 source)
			execFP1Source(e, insn)
		default:
			undefined(c, insn)
		}

	case 2: // FP data-processing (2 source)
		rm := bits(insn, 20, 16)
		op := bits(insn, 15, 12)
		var op2 int
		switch op {
		case 0:
			op2 = opMul
		case 1:
			op2 = opDiv
		case 2:
			op2 = opAdd
		case 3:
			op2 = opSub
		case 4:
			op2 = opMax
		case 5:
			op2 = opMin
		case 6:
			op2 = opMaxNM
		case 7:
			op2 = opMinNM
		case 8: // FNMUL: FPNeg(FPMul(...)), and FPNeg flips a NaN's sign too
			if dbl {
				e.wrD(rd, -e.arith(opMul, e.rdD(rn), e.rdD(rm)))
			} else {
				e.wrS(rd, -e.arith(opMul, e.rdS(rn), e.rdS(rm)))
			}
			return
		default:
			undefined(c, insn)
			return
		}
		if dbl {
			e.wrD(rd, e.arith(op2, e.rdD(rn), e.rdD(rm)))
		} else {
			// Single precision is computed in double, where the exact product
			// or quotient fits, so the only rounding is the one to float32.
			r := e.arith(op2, e.rdS(rn), e.rdS(rm))
			if !math.IsNaN(r) && !math.IsInf(r, 0) && float64(float32(r)) != r {
				e.exc |= fpsrIXC
			}
			e.wrS(rd, r)
		}

	case 3: // FP conditional select
		rm := bits(insn, 20, 16)
		src := rn
		if !c.CondHolds(bits(insn, 15, 12)) {
			src = rm
		}
		if dbl {
			c.V[rd].SetU64(0, c.V[src].U64(0))
			c.V[rd].SetU64(1, 0)
		} else {
			c.V[rd].SetU64(0, uint64(c.V[src].U32(0)))
			c.V[rd].SetU64(1, 0)
		}

	case 1: // FP conditional compare
		rm := bits(insn, 20, 16)
		if c.CondHolds(bits(insn, 15, 12)) {
			var cmp int
			if dbl {
				cmp = e.compare(e.rdD(rn), e.rdD(rm), bit(insn, 4))
			} else {
				cmp = e.compare(e.rdS(rn), e.rdS(rm), bit(insn, 4))
			}
			c.fpSetFlags(cmp)
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

	default:
		undefined(c, insn)
	}
}

func execFP1Source(e *fpEnv, insn uint32) {
	c := e.c
	dbl := bits(insn, 23, 22) == 1
	rn := bits(insn, 9, 5)
	rd := bits(insn, 4, 0)

	switch opc := bits(insn, 20, 15); opc {
	case 0: // FMOV: a register copy, so no rounding, no flush, and a
		// signaling NaN stays signaling.
		if dbl {
			c.V[rd].SetU64(0, c.V[rn].U64(0))
		} else {
			c.V[rd].SetU64(0, uint64(c.V[rn].U32(0)))
		}
		c.V[rd].SetU64(1, 0)
	case 1: // FABS
		if dbl {
			c.V[rd].SetU64(0, c.V[rn].U64(0)&^(1<<63))
		} else {
			c.V[rd].SetU64(0, uint64(c.V[rn].U32(0)&^(1<<31)))
		}
		c.V[rd].SetU64(1, 0)
	case 2: // FNEG
		if dbl {
			c.V[rd].SetU64(0, c.V[rn].U64(0)^(1<<63))
		} else {
			c.V[rd].SetU64(0, uint64(c.V[rn].U32(0)^(1<<31)))
		}
		c.V[rd].SetU64(1, 0)
	case 3: // FSQRT
		if dbl {
			x := e.rdD(rn)
			if x < 0 && x != 0 {
				e.exc |= fpsrIOC
			}
			e.wrD(rd, e.propagateNaN(math.Sqrt(x), x, x))
		} else {
			x := float32(e.rdS(rn))
			if x < 0 {
				e.exc |= fpsrIOC
			}
			e.wrS(rd, e.propagateNaN(float64(math.Sqrt(float64(x))), float64(x), float64(x)))
		}
	case 4: // FCVT to single (from double)
		if !dbl {
			undefined(c, insn)
			return
		}
		e.wrS(rd, e.rdD(rn))
	case 5: // FCVT to double (from single)
		if dbl {
			undefined(c, insn)
			return
		}
		e.wrD(rd, e.rdS(rn))
	case 8, 9, 0xa, 0xb, 0xc, 0xe, 0xf: // FRINT<N,P,M,Z,A,X,I>
		rm := 0
		switch opc {
		case 9:
			rm = 1
		case 0xa:
			rm = 2
		case 0xb:
			rm = 3
		case 0xc:
			rm = 4
		case 0xe, 0xf:
			rm = c.fpRoundMode() // FPCR.RMode
		}
		if dbl {
			e.wrD(rd, e.frint(e.rdD(rn), rm, opc == 0xe))
		} else {
			e.wrS(rd, e.frint(e.rdS(rn), rm, opc == 0xe))
		}
	default:
		undefined(c, insn)
	}
}

// execFPDP3 is the fused multiply-add family: FMADD, FMSUB, FNMADD, FNMSUB.
// The negations each form applies are folded into the operands before the NaN
// processing, because FMSUB of a NaN multiplicand returns it negated.
func execFPDP3(e *fpEnv, insn uint32) {
	c := e.c
	ftype := bits(insn, 23, 22)
	rm := bits(insn, 20, 16)
	ra := bits(insn, 14, 10)
	rn := bits(insn, 9, 5)
	rd := bits(insn, 4, 0)
	o1 := bit(insn, 21)
	o0 := bit(insn, 15)

	if ftype != 0 && ftype != 1 {
		undefined(c, insn)
		return
	}
	if ftype == 1 {
		n, m, a := e.rdD(rn), e.rdD(rm), e.rdD(ra)
		switch {
		case !o1 && !o0: // FMADD:  a + n * m
			e.wrD(rd, e.mulAdd(a, n, m))
		case !o1: // FMSUB:   a - n * m
			e.wrD(rd, e.mulAdd(a, -n, m))
		case o1 && !o0: // FNMADD: -a - n * m
			e.wrD(rd, e.mulAdd(-a, -n, m))
		default: // FNMSUB: -a + n * m
			e.wrD(rd, e.mulAdd(-a, n, m))
		}
		return
	}
	n, m, a := e.rdS(rn), e.rdS(rm), e.rdS(ra)
	switch {
	case !o1 && !o0:
		e.wrS(rd, e.mulAdd(a, n, m))
	case !o1:
		e.wrS(rd, e.mulAdd(a, -n, m))
	case o1 && !o0:
		e.wrS(rd, e.mulAdd(-a, -n, m))
	default:
		e.wrS(rd, e.mulAdd(-a, n, m))
	}
}

// vfpImm expands the 8-bit FP immediate of FMOV Dd/Sd, #imm, whose encoding
// (sign, a replicated exponent, a 4-bit fraction) is spelled out in
// fpsimd_simd.go's vfpImm32/vfpImm64.
func vfpImm(imm8 uint32, is64 bool) float64 {
	if is64 {
		return math.Float64frombits(vfpImm64(imm8))
	}
	return float64(math.Float32frombits(vfpImm32(imm8)))
}

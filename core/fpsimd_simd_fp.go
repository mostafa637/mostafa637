// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Advanced SIMD: the floating-point three-same group — the Go port of
// src/core/exec_fpsimd.c's simd_three_same_fp.
//
// The vector FP instructions on 32- and 64-bit lanes: FADD FSUB FMUL FDIV
// FMLA FMLS FMAX FMIN FMAXNM FMINNM FMAXP FMINP FMAXNMP FMINNMP FADDP FMULX
// FABD FRECPS FRSQRTS FCMEQ FCMGE FCMGT FACGE FACGT. Half-precision and the
// FEAT_FHM FMLAL/FMLSL forms stay UNDEFINED.
//
// The per-lane kernel is fopD/fopS, which is fop_d/fop_s: unpack (flushing a
// denormal operand when FPCR.FZ is set), compute, process NaNs, then round
// (flushing a tiny result). Sharing it with the scalar forms in fpsimd.go is
// what keeps the two agreeing — the architecture defines one operation per
// mnemonic, not one per register width.

package core

import "math"

// Floating-point operations, in the order src/core/exec_fpsimd.c numbers them.
const (
	fopAdd = iota
	fopSub
	fopMul
	fopDiv
	fopMLA
	fopMLS
	fopMax
	fopMin
	fopMaxNM
	fopMinNM
	fopABD
	fopMulX
	fopRECPS
	fopRSQRTS
)

// fopD is one lane of a floating-point three-same operation on doubles. The
// `d` operand is the accumulator: it is read by FMLA/FMLS/FRECPS only, and it
// is the value those forms add to.
func (e *fpEnv) fopD(op int, n, m, d float64) float64 {
	n, m = e.flushD(n), e.flushD(m)
	switch op {
	case fopMLA, fopMLS, fopRECPS:
		d = e.flushD(d)
	}
	var r float64
	switch op {
	case fopAdd:
		r = n + m
	case fopSub:
		r = n - m
	case fopMul:
		r = n * m
	case fopDiv:
		r = n / m
	case fopMLA:
		r = math.FMA(n, m, d)
	case fopMLS:
		r = math.FMA(-n, m, d)
	case fopMax:
		return fopMaxMin(e, 0, n, m)
	case fopMin:
		return fopMaxMin(e, 1, n, m)
	case fopMaxNM:
		return fopMaxMinNM(e, 0, n, m)
	case fopMinNM:
		return fopMaxMinNM(e, 1, n, m)
	case fopABD: // FPAbs(n - m), with the NaN processing inside the abs
		r = n - m
		e.raise(opAdd, n, -m, r)
		return e.outD(math.Abs(e.propagateNaN(r, n, m)))
	case fopMulX: // FMULX: 0 * inf is +/-2, and raises nothing
		if n == 0 && math.IsInf(m, 0) || math.IsInf(n, 0) && m == 0 {
			neg := math.Signbit(n) != math.Signbit(m)
			if neg {
				return e.outD(-2)
			}
			return e.outD(2)
		}
		r = n * m
	case fopRECPS: // FPRecipStepFused: 2 - n*m with a single rounding
		r = math.FMA(-n, m, 2)
	case fopRSQRTS: // FPRSqrtStepFused: (3 - n*m) / 2 with a single rounding.
		// The halving is folded into an operand so it stays one rounding:
		// fma-then-divide rounds twice, which shows at the overflow
		// boundary and in the subnormal range. Halving is exact only above
		// the bottom exponent, so halve whichever operand cannot be
		// clipped.
		switch {
		case math.Abs(n) == 0 || exponentBits(n) >= 2:
			r = math.FMA(-n/2, m, 1.5)
		case exponentBits(m) >= 2:
			r = math.FMA(-n, m/2, 1.5)
		default:
			r = math.FMA(-n, m, 3) / 2
		}
	default:
		return e.outD(math.NaN())
	}
	e.raise(fopRaiseKind(op), n, m, r)

	// NaN processing. The operand order is the one the C documents: a
	// propagated FPNeg applied before it comes back negated.
	switch op {
	case fopMLA:
		r = e.propagateMulAdd(r, d, n, m)
	case fopMLS:
		r = e.propagateMulAdd(r, d, -n, m)
	case fopRECPS, fopRSQRTS:
		r = e.propagateNaN(r, -n, m)
	default:
		r = e.propagateNaN(r, n, m)
	}
	return e.outD(r)
}

// fopS is fopD for single-precision lanes: the value is computed in double,
// where a product, quotient or sum of two floats is exact, so the only
// rounding is the one back to float32.
func (e *fpEnv) fopS(op int, n, m, d float64) float64 {
	v := e.fopD(op, n, m, d)
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return v
	}
	r := float64(float32(v))
	if r != v {
		e.exc |= fpsrIXC
	}
	return r
}

// propagateMulAdd is FPMulAdd's NaN rule: a quiet NaN addend combined with a
// 0 * infinity product is an invalid operation, and the answer is the default
// NaN rather than the addend.
func (e *fpEnv) propagateMulAdd(r, a, n, m float64) float64 {
	if math.IsNaN(a) && !isSNaN64(a) &&
		(math.IsInf(n, 0) && m == 0 || n == 0 && math.IsInf(m, 0)) {
		e.exc |= fpsrIOC
		return math.NaN()
	}
	return e.propagateNaN3(r, a, n, m)
}

// fopRaiseKind maps an operation onto the exception checks in raise(), which
// are written for the four basic ones.
func fopRaiseKind(op int) int {
	switch op {
	case fopSub:
		return opSub
	case fopDiv:
		return opDiv
	case fopMul, fopMulX:
		return opMul
	}
	return opAdd
}

// exponentBits is the unbiased-exponent field of a double, which FRSQRTS reads
// to decide which operand it can halve without clipping it.
func exponentBits(x float64) uint64 {
	return math.Float64bits(x) >> 52 & 0x7ff
}

// fopMaxMin is FPMax/FPMin: a NaN operand decides the answer, and +0.0 is
// greater than -0.0.
func fopMaxMin(e *fpEnv, sel int, n, m float64) float64 {
	if math.IsNaN(n) || math.IsNaN(m) {
		r := n
		if !math.IsNaN(r) {
			r = m
		}
		return e.propagateNaN(r, n, m)
	}
	if n == m {
		if math.Signbit(n) {
			return m
		}
		return n
	}
	if (sel == 0) == (n > m) {
		return n
	}
	return m
}

// fopMaxMinNM is FPMaxNum/FPMinNum: a NaN operand is treated as missing, so
// the other one is the answer.
func fopMaxMinNM(e *fpEnv, sel int, n, m float64) float64 {
	if math.IsNaN(n) && math.IsNaN(m) {
		return e.propagateNaN(n, n, m)
	}
	if math.IsNaN(n) {
		if isSNaN64(n) {
			e.exc |= fpsrIOC
		}
		return m
	}
	if math.IsNaN(m) {
		if isSNaN64(m) {
			e.exc |= fpsrIOC
		}
		return n
	}
	if n == m {
		if math.Signbit(n) {
			return m
		}
		return n
	}
	if (sel == 0) == (n > m) {
		return n
	}
	return m
}

// ---- vector FP compare ----------------------------------------------------

const (
	fcmEQ = iota
	fcmGE
	fcmGT
)

// fcmTest is FPCompare against a lane: unordered (a NaN anywhere) is false
// everywhere but for FCMLE/FCMLT, and a signaling NaN raises Invalid
// Operation — including for FCMEQ, which is the one compare that is quiet in
// the scalar form.
func (e *fpEnv) fcmTest(kind int, x, y float64) bool {
	if math.IsNaN(x) || math.IsNaN(y) {
		if isSNaN64(x) || isSNaN64(y) {
			e.exc |= fpsrIOC
		}
		return false
	}
	switch kind {
	case fcmEQ:
		return x == y
	case fcmGE:
		return x >= y
	}
	return x > y
}

// ---- three-same (FP) ------------------------------------------------------

func execSIMDThreeSameFP(e *fpEnv, insn uint32) {
	c := e.c
	q := bit(insn, 30)
	u := bit(insn, 29)
	a := bit(insn, 23)
	sz := bit(insn, 22)
	opc := bits(insn, 15, 11)
	rm := bits(insn, 20, 16)
	rn := bits(insn, 9, 5)
	rd := bits(insn, 4, 0)

	var key uint32
	if u {
		key |= 1 << 6
	}
	if a {
		key |= 1 << 5
	}
	key |= opc

	// FEAT_FHM FMLAL/FMLSL and their "2" forms: half-precision widening.
	if key == 0x1d || key == 0x3d || key == 0x59 || key == 0x79 {
		undefined(c, insn)
		return
	}

	pair := u && (opc == 0x18 || opc == 0x1e || (opc == 0x1a && !a))
	cmp := opc == 0x1c || opc == 0x1d

	var op int
	switch key {
	case 0x18:
		op = fopMaxNM
	case 0x38:
		op = fopMinNM
	case 0x19:
		op = fopMLA
	case 0x39:
		op = fopMLS
	case 0x1a:
		op = fopAdd
	case 0x3a:
		op = fopSub
	case 0x1b:
		op = fopMulX
	case 0x1e:
		op = fopMax
	case 0x3e:
		op = fopMin
	case 0x1f:
		op = fopRECPS
	case 0x3f:
		op = fopRSQRTS
	case 0x5b:
		op = fopMul
	case 0x5f:
		op = fopDiv
	case 0x7a:
		op = fopABD
	case 0x58:
		op = fopMaxNM // FMAXNMP
	case 0x5a:
		op = fopAdd // FADDP
	case 0x5e:
		op = fopMax // FMAXP
	case 0x78:
		op = fopMinNM // FMINNMP
	case 0x7e:
		op = fopMin // FMINP
	default:
		op = fopAdd // the compares: decided by `cmp` below
	}

	var r V128

	if sz { // .2d (Q=1) or .1d (Q=0)
		n := uint32(1)
		if q {
			n = 2
		}
		if pair { // the lower half folds pairs of Vn, the upper half pairs of Vm
			for i := uint32(0); i < n; i++ {
				src, base := rn, 2*i
				if i >= n/2 {
					src, base = rm, 2*(i-n/2)
				}
				x := c.V[src].U64(int(base))
				y := c.V[src].U64(int(base + 1))
				r.SetU64(int(i), math.Float64bits(e.fopD(op, math.Float64frombits(x), math.Float64frombits(y), 0)))
			}
			c.V[rd] = r
			return
		}
		for i := uint32(0); i < n; i++ {
			x := math.Float64frombits(c.V[rn].U64(int(i)))
			y := math.Float64frombits(c.V[rm].U64(int(i)))
			if cmp {
				kind := fcmGT
				switch key {
				case 0x1c:
					kind = fcmEQ
				case 0x5c:
					kind = fcmGE
				}
				if opc == 0x1d { // FACGE/FACGT compare the magnitudes
					kind = fcmGT
					if key == 0x5d {
						kind = fcmGE
					}
					if e.fcmTest(kind, math.Abs(x), math.Abs(y)) {
						r.SetU64(int(i), ^uint64(0))
					}
					continue
				}
				if e.fcmTest(kind, x, y) {
					r.SetU64(int(i), ^uint64(0))
				}
				continue
			}
			d := math.Float64frombits(c.V[rd].U64(int(i)))
			r.SetU64(int(i), math.Float64bits(e.fopD(op, x, y, d)))
		}
		c.V[rd] = r
		return
	}

	// .4s (Q=1) or .2s (Q=0)
	n := uint32(2)
	if q {
		n = 4
	}
	getS := func(reg, i uint32) float64 {
		return float64(math.Float32frombits(c.V[reg].U32(int(i))))
	}
	if pair {
		for i := uint32(0); i < n; i++ {
			src, base := rn, 2*i
			if i >= n/2 {
				src, base = rm, 2*(i-n/2)
			}
			v := e.fopS(op, getS(src, base), getS(src, base+1), 0)
			r.SetU32(int(i), math.Float32bits(float32(v)))
		}
		c.V[rd] = r
		return
	}
	for i := uint32(0); i < n; i++ {
		x, y := getS(rn, i), getS(rm, i)
		if cmp {
			kind := fcmGT
			switch key {
			case 0x1c:
				kind = fcmEQ
			case 0x5c:
				kind = fcmGE
			}
			if opc == 0x1d {
				kind = fcmGT
				if key == 0x5d {
					kind = fcmGE
				}
				if e.fcmTest(kind, math.Abs(x), math.Abs(y)) {
					r.SetU32(int(i), ^uint32(0))
				}
				continue
			}
			if e.fcmTest(kind, x, y) {
				r.SetU32(int(i), ^uint32(0))
			}
			continue
		}
		v := e.fopS(op, x, y, getS(rd, i))
		r.SetU32(int(i), math.Float32bits(float32(v)))
	}
	c.V[rd] = r
}

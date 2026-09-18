// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Ported to Go from src/core/decode.c — the A64 integer/branch/load-store
// decoder and executor (M1). FP/SIMD lives in fpsimd.go, system registers in
// sysreg.go.

package core

import (
	"fmt"
	mbits "math/bits"
	"sync"
	"sync/atomic"
	"unsafe"
)

// ---- field extraction ----

func bit(insn uint32, i uint) bool { return (insn>>i)&1 != 0 }

func bits(insn uint32, hi, lo uint) uint32 {
	return (insn >> lo) & ((1 << (hi - lo + 1)) - 1)
}

// undefined takes the synchronous UNDEFINED-instruction exception.
func undefined(c *CPU, insn uint32) {
	if c.Trace != nil {
		fmt.Fprintf(c.Trace, "UNDEF insn 0x%08x at pc=0x%x\n", insn, c.CurInsnPC)
	}
	c.RaiseSync(ESRMake(ECUnknown, 0), 0)
}

// ---- arithmetic helpers ----

// addWithCarry is AddWithCarry() from the Arm ARM: x + y + carry-in, with the
// NZCV the addition produces. When wantFlags is false the flag computation is
// skipped, exactly as the C core skips it for the non-S forms.
func addWithCarry(x, y uint64, cin uint64, is64, wantFlags bool) (uint64, uint32) {
	if is64 {
		t := x + y
		res := t + cin
		if !wantFlags {
			return res, 0
		}
		var f uint32
		if t < x || res < t {
			f |= PSC
		}
		if ^(x^y)&(x^res)>>63 != 0 { // operands same sign, result differs
			f |= PSV
		}
		if res>>63 != 0 {
			f |= PSN
		}
		if res == 0 {
			f |= PSZ
		}
		return res, f
	}
	xx, yy := uint32(x), uint32(y)
	u := uint64(xx) + uint64(yy) + cin
	res := uint64(uint32(u))
	if !wantFlags {
		return res, 0
	}
	var f uint32
	if (u>>32)&1 != 0 {
		f |= PSC
	}
	if int64(int32(uint32(res))) != int64(int32(xx))+int64(int32(yy))+int64(cin) {
		f |= PSV
	}
	if uint32(res)>>31 != 0 {
		f |= PSN
	}
	if uint32(res) == 0 {
		f |= PSZ
	}
	return res, f
}

// setLogicalFlags sets N/Z from a logical result and clears C and V.
func setLogicalFlags(c *CPU, res uint64, is64 bool) {
	var f uint32
	if is64 {
		if res>>63 != 0 {
			f |= PSN
		}
		if res == 0 {
			f |= PSZ
		}
	} else {
		r := uint32(res)
		if r>>31 != 0 {
			f |= PSN
		}
		if r == 0 {
			f |= PSZ
		}
	}
	c.NZCV = f
}

// shiftReg applies a shifted-register operand shift (type 0 LSL, 1 LSR,
// 2 ASR, 3 ROR) of `amount`, at the given register width.
func shiftReg(v uint64, typ, amount uint, is64 bool) uint64 {
	w := uint(32)
	if is64 {
		w = 64
	}
	amount &= w - 1
	if !is64 {
		v = uint64(uint32(v))
	}
	switch typ {
	case 0:
		return v << amount
	case 1:
		if is64 {
			return v >> amount
		}
		return uint64(uint32(v) >> amount)
	case 2:
		if is64 {
			return uint64(int64(v) >> amount)
		}
		return uint64(uint32(int32(uint32(v)) >> amount))
	default:
		if is64 {
			return Ror64(v, amount)
		}
		return uint64(Ror32(uint32(v), amount))
	}
}

// ---- CRC32/CRC32C ------------------------------------------------------
//
// Bit-reflected CRC over the low `bytes` bytes of data with accumulator acc;
// poly is the reflected polynomial (0xEDB88320 for CRC32, 0x82F63B78 for
// CRC32C). Slicing-by-8, as in the C core: CRC32X consumes its eight bytes in
// one combined lookup instead of 64 bit-steps. ext4/btrfs metadata checksums
// lean on this.

var crcTables = func() [2][8][256]uint32 {
	var t [2][8][256]uint32
	for p, poly := range [2]uint32{0xEDB88320, 0x82F63B78} {
		for i := 0; i < 256; i++ {
			c := uint32(i)
			for k := 0; k < 8; k++ {
				if c&1 != 0 {
					c = c>>1 ^ poly
				} else {
					c >>= 1
				}
			}
			t[p][0][i] = c
		}
		for i := 0; i < 256; i++ {
			for j := 1; j < 8; j++ {
				t[p][j][i] = t[p][j-1][i]>>8 ^ t[p][0][t[p][j-1][i]&0xff]
			}
		}
	}
	return t
}()

func crc32Step(acc uint32, data uint64, bytes_ uint, poly uint32) uint32 {
	p := 0
	if poly == 0x82F63B78 {
		p = 1
	}
	t := &crcTables[p]
	if bytes_ == 8 {
		acc ^= uint32(data)
		hi := uint32(data >> 32)
		return t[7][acc&0xff] ^ t[6][(acc>>8)&0xff] ^
			t[5][(acc>>16)&0xff] ^ t[4][acc>>24] ^
			t[3][hi&0xff] ^ t[2][(hi>>8)&0xff] ^
			t[1][(hi>>16)&0xff] ^ t[0][hi>>24]
	}
	for i := uint(0); i < bytes_; i++ {
		acc = acc>>8 ^ t[0][(acc^uint32(data>>(8*i)))&0xff]
	}
	return acc
}

// extendReg is the extend/LSL of an extended-register operand:
// option<2:0> selects the source width (byte/half/word/doubleword) and
// option<2> signedness; shift is the LSL amount (0..4).
func extendReg(v uint64, option, shift uint) uint64 {
	var out uint64
	switch option & 3 {
	case 0:
		out = uint64(uint8(v))
	case 1:
		out = uint64(uint16(v))
	case 2:
		out = uint64(uint32(v))
	default:
		out = v
	}
	if option&4 != 0 { // signed
		n := uint(64)
		switch option & 3 {
		case 0:
			n = 8
		case 1:
			n = 16
		case 2:
			n = 32
		}
		out = SignExtend(out, n)
	}
	return out << shift
}

// rorWithin rotates the low `width` bits of v right by r.
func rorWithin(v uint64, r, width uint) uint64 {
	if width == 64 {
		return Ror64(v, r)
	}
	v = uint64(uint32(v))
	r %= width
	if r == 0 {
		return v
	}
	return ((v >> r) | (v << (width - r))) & 0xffffffff
}

// ---- logical immediates and bitfields -----------------------------------
//
// Every logical-immediate and every UBFM/SBFM/BFM (i.e. all LSL/LSR/ASR/
// UBFX/... aliases) funnels through DecodeBitmasks, and the clz+rotate+
// replicate computation depends only on the 13-bit immN:immr:imms key.
//
// The C core memoizes lazily into a static table (a benign race); Go has no
// benign races, and locking the hot path to fill it would cost more than
// filling all 8192 entries at load time. So they are precomputed here.

var bitmaskMemo = func() [1 << 13]struct {
	w, t   uint64
	ok     bool
	filled bool
} {
	var m [1 << 13]struct {
		w, t   uint64
		ok     bool
		filled bool
	}
	for key := 0; key < 1<<13; key++ {
		immN := uint((key >> 12) & 1)
		immr := uint((key >> 6) & 0x3f)
		imms := uint(key & 0x3f)
		m[key].w, m[key].t, m[key].ok = decodeBitmasksSlow(immN, imms, immr)
		m[key].filled = true
	}
	return m
}()

func decodeBitmasksSlow(immN, imms, immr uint) (wmask, tmask uint64, ok bool) {
	nimms := uint32((immN&1)<<6 | (^imms)&0x3f)
	if nimms == 0 {
		return 0, 0, false
	}
	len_ := uint(31 - mbits.LeadingZeros32(nimms))
	if len_ < 1 {
		return 0, 0, false
	}
	levels := uint((1 << len_) - 1)
	S := imms & levels
	R := immr & levels
	diff := (S - R) & levels
	esize := uint(1) << len_
	welem := Ones(S + 1)
	telem := Ones(diff + 1)
	emask := ^uint64(0)
	if esize != 64 {
		emask = (uint64(1) << esize) - 1
	}
	welem &= emask
	telem &= emask
	r := R % esize
	w := welem
	if r != 0 {
		w = ((welem >> r) | (welem << (esize - r))) & emask
	}
	var wm, tm uint64
	for i := uint(0); i < 64; i += esize {
		wm |= w << i
		tm |= telem << i
	}
	return wm, tm, true
}

// DecodeBitmasks returns the wanted-mask and tmask of a bitmask immediate.
// The 32-bit view is the truncated 64-bit result; immN==1 is the one
// 64-bit-only encoding (esize 64 > 32).
func DecodeBitmasks(immN, imms, immr uint32, is64 bool) (wmask, tmask uint64, ok bool) {
	key := (immN&1)<<12 | (immr&0x3f)<<6 | (imms & 0x3f)
	e := bitmaskMemo[key]
	if !e.ok {
		return 0, 0, false
	}
	if !is64 && immN != 0 {
		return 0, 0, false
	}
	wm, tm := e.w, e.t
	if !is64 {
		wm = uint64(uint32(wm))
		tm = uint64(uint32(tm))
	}
	return wm, tm, true
}

// ================= DP-immediate =================

func dpImmediate(c *CPU, insn uint32) {
	t := bits(insn, 28, 23)
	sf := bit(insn, 31)
	rd := bits(insn, 4, 0)
	rn := bits(insn, 9, 5)

	switch {
	case t == 0x20 || t == 0x21: // PC-rel: ADR / ADRP
		immlo := bits(insn, 30, 29)
		immhi := bits(insn, 23, 5)
		imm := SignExtend(uint64(immhi)<<2|uint64(immlo), 21)
		if !bit(insn, 31) {
			c.SetX(rd, c.CurInsnPC+imm)
		} else {
			c.SetX(rd, (c.CurInsnPC&^uint64(0xfff))+(imm<<12))
		}
		return

	case t == 0x22: // add/sub (immediate)
		op := bit(insn, 30)
		S := bit(insn, 29)
		sh := bit(insn, 22)
		imm := uint64(bits(insn, 21, 10))
		if sh {
			imm <<= 12
		}
		n := c.RegXSP(rn)
		var r uint64
		var fl uint32
		if op {
			r, fl = addWithCarry(n, ^imm, 1, sf, S)
		} else {
			r, fl = addWithCarry(n, imm, 0, sf, S)
		}
		if S {
			c.NZCV = fl
			c.SetXSz(rd, sf, r)
		} else if sf {
			c.SetXSP(rd, r)
		} else {
			c.SetXSP(rd, uint64(uint32(r)))
		}
		return

	case t == 0x24: // logical (immediate)
		opc := bits(insn, 30, 29)
		N := bit(insn, 22)
		immr := bits(insn, 21, 16)
		imms := bits(insn, 15, 10)
		wmask, _, ok := DecodeBitmasks(boolToUint(N), imms, immr, sf)
		if !ok {
			undefined(c, insn)
			return
		}
		n := c.RegX(rn)
		var r uint64
		switch opc {
		case 0:
			r = n & wmask
		case 1:
			r = n | wmask
		case 2:
			r = n ^ wmask
		default:
			r = n & wmask
		}
		if !sf {
			r = uint64(uint32(r))
		}
		if opc == 3 {
			setLogicalFlags(c, r, sf)
			c.SetX(rd, r)
		} else {
			c.SetXSP(rd, r)
		}
		return

	case t == 0x25: // move wide (immediate)
		opc := bits(insn, 30, 29)
		hw := bits(insn, 22, 21)
		imm16 := uint64(bits(insn, 20, 5))
		if !sf && hw >= 2 { // 32-bit: hw in {0,1} only
			undefined(c, insn)
			return
		}
		shift := hw * 16
		var r uint64
		switch opc {
		case 0: // MOVN
			r = ^(imm16 << shift)
		case 2: // MOVZ
			r = imm16 << shift
		case 3: // MOVK
			cur := c.RegX(rd)
			r = cur&^(uint64(0xffff)<<shift) | (imm16 << shift)
		default:
			undefined(c, insn)
			return
		}
		c.SetXSz(rd, sf, r)
		return

	case t == 0x26: // bitfield: SBFM / BFM / UBFM
		opc := bits(insn, 30, 29)
		N := bit(insn, 22)
		immr := bits(insn, 21, 16)
		imms := bits(insn, 15, 10)
		// opc==11 has no bitfield instruction, and N must equal sf: the
		// sf==0/N==1 half is rejected inside DecodeBitmasks, this catches
		// sf==1/N==0 (which would run a 32-bit pattern on a 64-bit register).
		if opc == 3 || boolToUint(N) != boolToUint(sf) {
			undefined(c, insn)
			return
		}
		wmask, tmask, ok := DecodeBitmasks(boolToUint(N), imms, immr, sf)
		if !ok {
			undefined(c, insn)
			return
		}
		src := c.RegX(rn)
		ror := rorWithin(src, uint(immr), widthOf(sf))
		bot := ror & wmask
		if opc == 1 {
			bot = c.RegX(rd)&^wmask | (ror & wmask)
		}
		var top uint64
		switch opc {
		case 0: // SBFM
			if (src>>imms)&1 != 0 {
				top = ^uint64(0)
			}
		case 2: // UBFM
			top = 0
		default: // BFM
			top = c.RegX(rd)
		}
		c.SetXSz(rd, sf, top&^tmask|(bot&tmask))
		return

	case t == 0x27: // EXTR
		rm := bits(insn, 20, 16)
		imms := bits(insn, 15, 10)
		// N must equal sf, bit21 must be 0, and the 32-bit form only allows
		// lsb<32 — otherwise unallocated. The guard also avoids shifting a
		// u32 by 32-lsb when lsb>=32.
		if bits(insn, 30, 29) != 0 || boolToUint(bit(insn, 22)) != boolToUint(sf) ||
			bit(insn, 21) || (!sf && imms&0x20 != 0) {
			undefined(c, insn)
			return
		}
		hi := c.RegX(rn)
		lo := c.RegX(rm)
		lsb := uint(imms)
		var r uint64
		if sf {
			if lsb != 0 {
				r = hi<<(64-lsb) | lo>>lsb
			} else {
				r = lo
			}
		} else {
			h, l := uint32(hi), uint32(lo)
			if lsb != 0 {
				r = uint64(h<<(32-lsb) | l>>lsb)
			} else {
				r = uint64(l)
			}
		}
		c.SetXSz(rd, sf, r)
		return
	}
	undefined(c, insn)
}

// ================= DP-register =================

func dpRegister(c *CPU, insn uint32) {
	sf := bit(insn, 31)
	rd := bits(insn, 4, 0)
	rn := bits(insn, 9, 5)
	rm := bits(insn, 20, 16)
	op24 := bits(insn, 28, 24)

	switch op24 {
	case 0x0a: // logical (shifted register)
		opc := bits(insn, 30, 29)
		shift := bits(insn, 23, 22)
		N := bit(insn, 21)
		imm6 := bits(insn, 15, 10)
		if !sf && imm6&0x20 != 0 { // imm6>=32 in 32-bit: unallocated
			undefined(c, insn)
			return
		}
		op2 := shiftReg(c.RegX(rm), uint(shift), uint(imm6), sf)
		if N {
			op2 = ^op2
		}
		n := c.RegX(rn)
		var r uint64
		switch opc {
		case 0:
			r = n & op2 // AND/BIC
		case 1:
			r = n | op2 // ORR/ORN
		case 2:
			r = n ^ op2 // EOR/EON
		default:
			r = n & op2 // ANDS/BICS
		}
		if !sf {
			r = uint64(uint32(r))
		}
		if opc == 3 {
			setLogicalFlags(c, r, sf)
		}
		c.SetX(rd, r)
		return

	case 0x0b: // add/sub (shifted or extended register)
		ext := bit(insn, 21)
		op := bit(insn, 30)
		S := bit(insn, 29)
		var op2, n uint64
		if ext { // add/sub (extended register)
			option := bits(insn, 15, 13)
			imm3 := bits(insn, 12, 10)
			if imm3 > 4 || bits(insn, 23, 22) != 0 {
				undefined(c, insn)
				return
			}
			op2 = extendReg(c.RegX(rm), uint(option), uint(imm3))
			n = c.RegXSP(rn)
		} else { // add/sub (shifted register)
			shift := bits(insn, 23, 22)
			imm6 := bits(insn, 15, 10)
			// ROR is unallocated for add/sub; imm6>=32 in 32-bit too.
			if shift == 3 || (!sf && imm6&0x20 != 0) {
				undefined(c, insn)
				return
			}
			op2 = shiftReg(c.RegX(rm), uint(shift), uint(imm6), sf)
			n = c.RegX(rn)
		}
		var r uint64
		var fl uint32
		if op {
			r, fl = addWithCarry(n, ^op2, 1, sf, S)
		} else {
			r, fl = addWithCarry(n, op2, 0, sf, S)
		}
		switch {
		case S:
			c.NZCV = fl
			c.SetXSz(rd, sf, r)
		case ext:
			if sf {
				c.SetXSP(rd, r)
			} else {
				c.SetXSP(rd, uint64(uint32(r)))
			}
		default:
			c.SetXSz(rd, sf, r)
		}
		return

	case 0x1b: // data processing (3 source)
		op31 := bits(insn, 23, 21)
		o0 := bit(insn, 15)
		ra := bits(insn, 14, 10)
		if bits(insn, 30, 29) != 0 { // op54 is RES0
			undefined(c, insn)
			return
		}
		// The widening/high multiplies require sf=1; sf=0 is unallocated.
		if !sf && op31 != 0 {
			undefined(c, insn)
			return
		}
		n := c.RegX(rn)
		m := c.RegX(rm)
		a := c.RegX(ra)
		var r uint64
		switch op31<<1 | boolToUint(o0) {
		case 0x0:
			r = a + n*m // MADD
		case 0x1:
			r = a - n*m // MSUB
		case 0x2:
			r = a + uint64(int64(int32(uint32(n)))*int64(int32(uint32(m)))) // SMADDL
		case 0x3:
			r = a - uint64(int64(int32(uint32(n)))*int64(int32(uint32(m)))) // SMSUBL
		case 0x4:
			r = SMulH(int64(n), int64(m)) // SMULH
		case 0xa:
			r = a + uint64(uint32(n))*uint64(uint32(m)) // UMADDL
		case 0xb:
			r = a - uint64(uint32(n))*uint64(uint32(m)) // UMSUBL
		case 0xc:
			r = UMulH(n, m) // UMULH
		default:
			undefined(c, insn)
			return
		}
		c.SetXSz(rd, sf, r)
		return

	case 0x1a:
		op21 := bits(insn, 28, 21)
		switch op21 {
		case 0xd0:
			if bits(insn, 15, 10) == 0 { // add/sub (with carry)
				op := bit(insn, 30)
				S := bit(insn, 29)
				cin := uint64(0)
				if c.NZCV&PSC != 0 {
					cin = 1
				}
				m := c.RegX(rm)
				n := c.RegX(rn)
				var r uint64
				var fl uint32
				if op {
					r, fl = addWithCarry(n, ^m, cin, sf, S)
				} else {
					r, fl = addWithCarry(n, m, cin, sf, S)
				}
				if S {
					c.NZCV = fl
				}
				c.SetXSz(rd, sf, r)
				return
			}
			// RMIF (FEAT_FLAGM): rotate Xn right by imm6, move tmp<3:0> into
			// the NZCV bits selected by mask (bit3=N .. bit0=V).
			if bit(insn, 31) && !bit(insn, 30) && bit(insn, 29) &&
				bits(insn, 14, 10) == 1 && !bit(insn, 4) {
				imm6 := bits(insn, 20, 15)
				mask := uint32(bits(insn, 3, 0))
				t := c.RegX(rn)
				if imm6 != 0 {
					t = t>>imm6 | t<<(64-imm6)
				}
				nib := (c.NZCV>>28)&0xf&^mask | uint32(t)&mask
				c.NZCV = nib << 28
				return
			}
			// SETF8/SETF16 (FEAT_FLAGM): narrowing-overflow flags from the
			// low 8/16 bits of Wn: N=sign, Z=zero, V=bit msb+1 EOR msb;
			// C is unchanged.
			if !bit(insn, 31) && !bit(insn, 30) && bit(insn, 29) &&
				bits(insn, 20, 16) == 0 && (bits(insn, 15, 10) == 0x02 || bits(insn, 15, 10) == 0x12) &&
				bits(insn, 4, 0) == 0x0d {
				msb := uint(7)
				if bit(insn, 14) {
					msb = 15
				}
				w := c.RegX(rn)
				f := c.NZCV & PSC
				if (w>>msb)&1 != 0 {
					f |= PSN
				}
				if w&((2<<msb)-1) == 0 {
					f |= PSZ
				}
				if ((w>>(msb+1))^(w>>msb))&1 != 0 {
					f |= PSV
				}
				c.NZCV = f
				return
			}
			undefined(c, insn)
			return

		case 0xd2: // conditional compare (immediate / register)
			if !bit(insn, 29) || bit(insn, 10) || bit(insn, 4) {
				undefined(c, insn)
				return
			}
			op := bit(insn, 30)
			cond := bits(insn, 15, 12)
			nzcv := bits(insn, 3, 0)
			isImm := bit(insn, 11)
			n := c.RegX(rn)
			m := uint64(0)
			if isImm {
				m = uint64(bits(insn, 20, 16))
			} else {
				m = c.RegX(rm)
			}
			if c.CondHolds(cond) {
				var fl uint32
				if op { // CCMP
					_, fl = addWithCarry(n, ^m, 1, sf, true)
				} else { // CCMN
					_, fl = addWithCarry(n, m, 0, sf, true)
				}
				c.NZCV = fl
			} else {
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
			return

		case 0xd4: // conditional select
			if bit(insn, 29) || bit(insn, 11) {
				undefined(c, insn)
				return
			}
			op := bit(insn, 30)
			o2 := bit(insn, 10)
			cond := bits(insn, 15, 12)
			n := c.RegX(rn)
			m := c.RegX(rm)
			r := n
			if !c.CondHolds(cond) {
				switch {
				case !op && !o2:
					r = m // CSEL
				case !op && o2:
					r = m + 1 // CSINC
				case op && !o2:
					r = ^m // CSINV
				default:
					r = 0 - m // CSNEG (unsigned negation: INT64_MIN is
					// arithmetic, not signed overflow)
				}
			}
			c.SetXSz(rd, sf, r)
			return

		case 0xd6:
			if bit(insn, 30) { // data processing (1 source)
				opcode := bits(insn, 15, 10)
				n := c.RegX(rn)
				var r uint64
				switch opcode {
				case 0x00: // RBIT
					w := widthOf(sf)
					v := n
					if !sf {
						v = uint64(uint32(n))
					}
					r = ReverseBits(v, w)
				case 0x01: // REV16
					v := n
					var o uint64
					w := uint(8)
					if !sf {
						w = 4
					}
					for i := uint(0); i < w; i += 2 {
						o |= (v >> (i * 8)) & 0xff << ((i + 1) * 8)
						o |= (v >> ((i + 1) * 8)) & 0xff << (i * 8)
					}
					r = o
				case 0x02: // REV32 (64-bit) or REV (32-bit)
					if sf {
						var o uint64
						for g := uint(0); g < 2; g++ {
							word := uint32(vShift(n, g*32))
							o |= uint64(mbits.ReverseBytes32(word)) << (g * 32)
						}
						r = o
					} else {
						r = uint64(mbits.ReverseBytes32(uint32(n)))
					}
				case 0x03: // REV (64-bit)
					r = mbits.ReverseBytes64(n)
				case 0x04: // CLZ
					r = CountlZero(n, widthOf(sf))
				case 0x05: // CLS
					w := widthOf(sf)
					v := n
					if !sf {
						v = uint64(uint32(n))
					}
					sign := (v >> (w - 1)) & 1
					cnt := uint64(0)
					for i := int(w - 2); i >= 0; i-- {
						if (v>>i)&1 == sign {
							cnt++
						} else {
							break
						}
					}
					r = cnt
				default:
					undefined(c, insn)
					return
				}
				c.SetXSz(rd, sf, r)
				return
			}
			// data processing (2 source)
			opcode := bits(insn, 15, 10)
			n := c.RegX(rn)
			m := c.RegX(rm)
			var r uint64
			switch opcode {
			case 0x02: // UDIV
				if sf {
					if m == 0 {
						r = 0
					} else {
						r = n / m
					}
				} else {
					a, b := uint32(n), uint32(m)
					if b == 0 {
						r = 0
					} else {
						r = uint64(a / b)
					}
				}
			case 0x03: // SDIV
				if sf {
					a, b := int64(n), int64(m)
					switch {
					case b == 0:
						r = 0
					case b == -1 && a == -1<<63:
						r = uint64(a)
					default:
						r = uint64(a / b)
					}
				} else {
					a, b := int32(uint32(n)), int32(uint32(m))
					if b == 0 {
						r = 0
					} else if b == -1 && a == -1<<31 {
						r = uint64(uint32(a))
					} else {
						r = uint64(uint32(a / b))
					}
				}
			case 0x08:
				r = shiftReg(n, 0, uint(m), sf) // LSLV
			case 0x09:
				r = shiftReg(n, 1, uint(m), sf) // LSRV
			case 0x0a:
				r = shiftReg(n, 2, uint(m), sf) // ASRV
			case 0x0b:
				r = shiftReg(n, 3, uint(m), sf) // RORV
			case 0x10:
				r = uint64(crc32Step(uint32(n), m, 1, 0xEDB88320)) // CRC32B
			case 0x11:
				r = uint64(crc32Step(uint32(n), m, 2, 0xEDB88320)) // CRC32H
			case 0x12:
				r = uint64(crc32Step(uint32(n), m, 4, 0xEDB88320)) // CRC32W
			case 0x13:
				r = uint64(crc32Step(uint32(n), m, 8, 0xEDB88320)) // CRC32X
			case 0x14:
				r = uint64(crc32Step(uint32(n), m, 1, 0x82F63B78)) // CRC32CB
			case 0x15:
				r = uint64(crc32Step(uint32(n), m, 2, 0x82F63B78)) // CRC32CH
			case 0x16:
				r = uint64(crc32Step(uint32(n), m, 4, 0x82F63B78)) // CRC32CW
			case 0x17:
				r = uint64(crc32Step(uint32(n), m, 8, 0x82F63B78)) // CRC32CX
			default:
				undefined(c, insn)
				return
			}
			c.SetXSz(rd, sf, r)
			return
		}
	}
	undefined(c, insn)
}

// ================= loads/stores =================

func ldstExtendedOffset(c *CPU, insn uint32, size uint32) uint64 {
	rm := bits(insn, 20, 16)
	option := bits(insn, 15, 13)
	S := bit(insn, 12)
	shift := uint(0)
	if S {
		shift = uint(size)
	}
	// option==011 => LSL/UXTX (64-bit, no real extend)
	return extendReg(c.RegX(rm), uint(option), shift)
}

// doLoad performs an integer load. It returns false if the access faulted; the
// caller MUST then abort the instruction WITHOUT applying base-register
// writeback, because the faulting instruction is re-executed after the handler
// returns and a writeback applied here would be applied twice.
func doLoad(c *CPU, rt uint32, va uint64, size, opc uint32) bool {
	bytes := uint(1) << size
	if opc == 2 && size == 3 {
		return true // PRFM: no register write
	}
	var raw uint64
	if !MemRead(c, va, bytes, &raw) {
		return false
	}
	sign := opc == 2 || opc == 3
	ext64 := false
	switch opc {
	case 2:
		ext64 = true
	case 3:
		ext64 = false
	default:
		ext64 = size == 3
	}
	val := raw
	if sign {
		val = SignExtend(raw, bytes*8)
	}
	if !ext64 {
		val = uint64(uint32(val))
	}
	c.SetX(rt, val)
	return true
}

// vregLoad loads `bytes` (1,2,4,8,16) into a SIMD/FP register, zero-extending.
func vregLoad(c *CPU, vt uint32, va uint64, bytes uint) bool {
	var val V128
	val.Clear()
	if bytes == 16 {
		if !MemRead128(c, va, &val) {
			return false
		}
	} else {
		var t uint64
		if !MemRead(c, va, bytes, &t) {
			return false
		}
		val.SetU64(0, t)
	}
	c.V[vt] = val
	return true
}

func vregStore(c *CPU, vt uint32, va uint64, bytes uint) bool {
	if bytes == 16 {
		return MemWrite128(c, va, &c.V[vt])
	}
	return MemWrite(c, va, bytes, c.V[vt].U64(0))
}

// ---- host atomic primitives --------------------------------------------
//
// The C core does LSE atomics with __atomic builtins on a translated host
// pointer. Go's sync/atomic covers 32- and 64-bit words; for the byte and
// halfword forms (and for the 16-byte CASP) there is no atomic type, so those
// take a sharded mutex — the same fallback the C core uses on hosts without
// lock-free 128-bit CAS. Ordering: Go's atomics are sequentially consistent,
// which satisfies every acquire/release combination the guest can ask for;
// being stronger than asked is always safe.

var atomicShards [64]sync.Mutex

func atomicShard(p []byte) *sync.Mutex {
	addr := uintptr(unsafe.Pointer(&p[0]))
	return &atomicShards[(addr>>3)%64]
}

// hostAtomicLoad atomically loads `size` bytes from p.
func hostAtomicLoad(p []byte, size uint) uint64 {
	switch size {
	case 4:
		return uint64(atomic.LoadUint32((*uint32)(unsafe.Pointer(&p[0]))))
	case 8:
		return atomic.LoadUint64((*uint64)(unsafe.Pointer(&p[0])))
	default:
		m := atomicShard(p)
		m.Lock()
		v := loadLE(p[:size])
		m.Unlock()
		return v
	}
}

// hostAtomicStore atomically stores `size` bytes to p.
func hostAtomicStore(p []byte, size uint, v uint64) {
	switch size {
	case 4:
		atomic.StoreUint32((*uint32)(unsafe.Pointer(&p[0])), uint32(v))
	case 8:
		atomic.StoreUint64((*uint64)(unsafe.Pointer(&p[0])), v)
	default:
		m := atomicShard(p)
		m.Lock()
		storeLE(p[:size], v)
		m.Unlock()
	}
}

// hostAtomicExchange swaps v into p, returning the previous value.
func hostAtomicExchange(p []byte, size uint, v uint64) uint64 {
	switch size {
	case 4:
		return uint64(atomic.SwapUint32((*uint32)(unsafe.Pointer(&p[0])), uint32(v)))
	case 8:
		return atomic.SwapUint64((*uint64)(unsafe.Pointer(&p[0])), v)
	default:
		m := atomicShard(p)
		m.Lock()
		old := loadLE(p[:size])
		storeLE(p[:size], v)
		m.Unlock()
		return old
	}
}

// hostAtomicCAS is a strong compare-and-swap: a spurious failure would leave
// the compare register equal to the expected value, which the guest reads as
// success while memory was not updated — a lost store that breaks every CAS
// based lock.
func hostAtomicCAS(p []byte, size uint, want, newv uint64) (old uint64, ok bool) {
	switch size {
	case 4:
		old = uint64(atomic.LoadUint32((*uint32)(unsafe.Pointer(&p[0]))))
		for {
			if old != want {
				return old, false
			}
			if atomic.CompareAndSwapUint32((*uint32)(unsafe.Pointer(&p[0])), uint32(old), uint32(newv)) {
				return old, true
			}
			old = uint64(atomic.LoadUint32((*uint32)(unsafe.Pointer(&p[0]))))
		}
	case 8:
		old = atomic.LoadUint64((*uint64)(unsafe.Pointer(&p[0])))
		for {
			if old != want {
				return old, false
			}
			if atomic.CompareAndSwapUint64((*uint64)(unsafe.Pointer(&p[0])), old, newv) {
				return old, true
			}
			old = atomic.LoadUint64((*uint64)(unsafe.Pointer(&p[0])))
		}
	default:
		m := atomicShard(p)
		m.Lock()
		old = loadLE(p[:size])
		if old == want {
			storeLE(p[:size], newv)
			ok = true
		}
		m.Unlock()
		return old, ok
	}
}

func loadLE(p []byte) uint64 {
	var v uint64
	for i := len(p) - 1; i >= 0; i-- {
		v = v<<8 | uint64(p[i])
	}
	return v
}

func storeLE(p []byte, v uint64) {
	for i := range p {
		p[i] = byte(v)
		v >>= 8
	}
}

// ---- ARMv8.1-A LSE atomics ---------------------------------------------

func atomicAlignFault(c *CPU, va uint64, write bool) {
	c.RaiseSync(ESRMake(ECDAbortLower, ISSDataAbort(write, FSCAlign)), va)
}

func atomicTransFault(c *CPU, va uint64, write bool) {
	c.RaiseSync(ESRMake(ECDAbortLower, ISSDataAbort(write, FSCTransL3)), va)
}

// ldstAtomic implements the atomic memory operations (LDADD/LDCLR/LDEOR/
// LDSET/LDSMAX/LDSMIN/LDUMAX/LDUMIN and SWP). Modern glibc selects these via
// HWCAP_ATOMICS; -march=armv8.1-a+ code uses them unconditionally.
func ldstAtomic(c *CPU, insn uint32) {
	size := bits(insn, 31, 30)
	A := bit(insn, 23)
	R := bit(insn, 22)
	rs := bits(insn, 20, 16)
	o3 := bit(insn, 15)
	opc := bits(insn, 14, 12)
	rn := bits(insn, 9, 5)
	rt := bits(insn, 4, 0)
	bytes := uint(1) << size

	// o3==1 is SWP (opc==0) or LDAPR (opc==4); other opc are LD<op>.
	if o3 && opc != 0 {
		if opc == 4 { // LDAPR: load-acquire RCpc
			var v uint64
			if !MemRead(c, c.RegXSP(rn), bytes, &v) {
				return
			}
			c.SetX(rt, v)
			return
		}
		undefined(c, insn)
		return
	}

	va := c.RegXSP(rn)
	operand := c.RegX(rs)
	if va&(uint64(bytes)-1) != 0 {
		atomicAlignFault(c, va, true)
		return
	}
	hp := MemHostPtr(c, va, bytes, AccWrite)
	if hp == nil {
		// A stable pointer is missing: distinguish permission from unmapped
		// for the right siginfo.
		if MemHostPtr(c, va, bytes, AccRead) != nil {
			atomicAlignFault(c, va, true) // mapped read-only: treat as abort
		} else {
			atomicTransFault(c, va, true)
		}
		return
	}
	_ = A
	_ = R

	var old uint64
	if o3 && opc == 0 { // SWP
		old = hostAtomicExchange(hp, bytes, operand)
	} else {
		for {
			o := hostAtomicLoad(hp, bytes)
			des := uint64(0)
			switch opc {
			case 0: // LDADD
				des = o + operand
			case 1: // LDCLR
				des = o & ^operand
			case 2: // LDEOR
				des = o ^ operand
			case 3: // LDSET
				des = o | operand
			case 4: // LDSMAX
				if int64Sign(o, bytes) > int64Sign(operand, bytes) {
					des = o
				} else {
					des = operand
				}
			case 5: // LDSMIN
				if int64Sign(o, bytes) < int64Sign(operand, bytes) {
					des = o
				} else {
					des = operand
				}
			case 6: // LDUMAX
				if o > operand {
					des = o
				} else {
					des = operand
				}
			default: // LDUMIN
				if o < operand {
					des = o
				} else {
					des = operand
				}
			}
			if prev, ok := hostAtomicCAS(hp, bytes, o, maskTo(des, bytes)); ok {
				old = prev
				break
			}
		}
	}
	c.SetX(rt, old) // ST<op> forms use Rt==31 and discard
}

// int64Sign sign-extends a `bytes`-wide value for the signed comparisons.
func int64Sign(v uint64, bytes uint) int64 {
	return int64(SignExtend(v, bytes*8))
}

func maskTo(v uint64, bytes uint) uint64 {
	switch bytes {
	case 1:
		return uint64(uint8(v))
	case 2:
		return uint64(uint16(v))
	case 4:
		return uint64(uint32(v))
	}
	return v
}

// ldstCas is CAS/CASA/CASL/CASAL: compare Rs, swap Rt, return old in Rs.
func ldstCas(c *CPU, insn uint32) {
	size := bits(insn, 31, 30)
	rs := bits(insn, 20, 16)
	rn := bits(insn, 9, 5)
	rt := bits(insn, 4, 0)
	bytes := uint(1) << size
	va := c.RegXSP(rn)
	if va&(uint64(bytes)-1) != 0 {
		atomicAlignFault(c, va, true)
		return
	}
	hp := MemHostPtr(c, va, bytes, AccWrite)
	if hp == nil {
		atomicTransFault(c, va, true)
		return
	}
	old, _ := hostAtomicCAS(hp, bytes, maskTo(c.RegX(rs), bytes), maskTo(c.RegX(rt), bytes))
	c.SetX(rs, old)
}

// casp16Lock serializes 16-byte compare-and-swap, which no host provides
// lock-free to Go.
var casp16Lock sync.Mutex

// ldstCasp is CASP/CASPA/CASPL/CASPAL: two-register compare-and-swap.
func ldstCasp(c *CPU, insn uint32) {
	sz := bit(insn, 30)
	rs := bits(insn, 20, 16)
	rn := bits(insn, 9, 5)
	rt := bits(insn, 4, 0)
	bytes := uint(8)
	if sz {
		bytes = 16
	}
	va := c.RegXSP(rn)
	if va&(uint64(bytes)-1) != 0 {
		atomicAlignFault(c, va, true)
		return
	}
	hp := MemHostPtr(c, va, bytes, AccWrite)
	if hp == nil {
		atomicTransFault(c, va, true)
		return
	}
	if !sz { // pair of 32-bit -> one u64
		cmp := c.RegX(rs)&0xffffffff | c.RegX(rs+1)<<32
		newv := c.RegX(rt)&0xffffffff | c.RegX(rt+1)<<32
		old, _ := hostAtomicCAS(hp, 8, cmp, newv)
		c.SetX(rs, uint64(uint32(old)))
		c.SetX(rs+1, uint64(uint32(old>>32)))
		return
	}
	// Pair of 64-bit -> 16 bytes: serialize on a global mutex.
	casp16Lock.Lock()
	defer casp16Lock.Unlock()
	cur0 := loadLE(hp[:8])
	cur1 := loadLE(hp[8:16])
	if cur0 == c.RegX(rs) && cur1 == c.RegX(rs+1) {
		storeLE(hp[:8], c.RegX(rt))
		storeLE(hp[8:16], c.RegX(rt+1))
	}
	c.SetX(rs, cur0)
	c.SetX(rs+1, cur1)
}

// ---- FEAT_LRCPC2: LDAPUR / STLUR ----

func ldstRcpcUnscaled(c *CPU, insn uint32) {
	size := bits(insn, 31, 30)
	opc := bits(insn, 23, 22)
	rn := bits(insn, 9, 5)
	rt := bits(insn, 4, 0)
	// The opc==2/size==3 slot is unallocated here (no PRFM in this space) and
	// must not reach doLoad, which would treat it as a prefetch no-op.
	if (opc == 2 && size == 3) || (opc == 3 && size >= 2) {
		undefined(c, insn)
		return
	}
	va := c.RegXSP(rn) + SignExtend(uint64(bits(insn, 20, 12)), 9)
	if opc == 0 { // STLUR
		MemWrite(c, va, uint(1)<<size, c.RegX(rt))
		return
	}
	doLoad(c, rt, va, size, opc) // LDAPUR*
}

// ---- FEAT_MOPS: memory copy and memory set ------------------------------
//
// CPYFx (memcpy, forced forward), CPYx (memmove) and SETx, in the "Option A"
// register format, matching what the C core (and qemu) implement so the
// guest-visible intermediate state is identical. The prologue performs up to
// the next page boundary and only then rewrites Xd[,Xs] to the final address
// and Xn to -(bytes remaining), writing NZCV=0000 to advertise Option A; M
// does the whole-page middle; E does the sub-page tail and raises the EC 0x27
// mismatch exception if a page or more remains. Restartability across data
// aborts: P keeps the registers in input format until it completes, M/E fold
// progress into Xn after every bounded step, so re-executing the faulting
// instruction always resumes correctly. A step never crosses a page, which
// makes each step fault-atomic (permissions are page-granular).

func mopsPageLimit(addr uint64) uint64    { return (addr+0x1000)&^uint64(0xfff) - addr }
func mopsPageLimitRev(addr uint64) uint64 { return addr&0xfff + 1 }

// mopsISS builds the EC_MOP syndrome:
// isSET[24] | options[22:19] | fromEpilogue[18] | wrongOption[17] |
// OptionA[16] | destreg[14:10] | srcreg[9:5] | sizereg[4:0].
func mopsISS(insn uint32, wrongOption bool) uint32 {
	isSet := bits(insn, 23, 22) == 3
	options := bits(insn, 13, 12)
	epilogue := bits(insn, 15, 14) == 2
	if !isSet {
		options = bits(insn, 15, 12)
		epilogue = bits(insn, 23, 22) == 2
	}
	v := uint32(0)
	if isSet {
		v |= 1 << 24
	}
	if epilogue {
		v |= 1 << 18
	}
	if wrongOption {
		v |= 1 << 17
	}
	v |= options << 19
	v |= 1 << 16
	v |= bits(insn, 4, 0) << 10
	v |= bits(insn, 20, 16) << 5
	v |= bits(insn, 9, 5)
	return v
}

// At EL0 the family is gated by SCTLR_EL1.MSCEn (UNDEF when clear).
func mopsEnabledOK(c *CPU, insn uint32) bool {
	if c.EL == 0 && (c.SCTLR[1]>>33)&1 == 0 {
		undefined(c, insn)
		return false
	}
	return true
}

// mopsSetStep is one bounded memset step: at most to the next page boundary.
// Returns bytes done; 0 means a fault was raised.
func mopsSetStep(c *CPU, toaddr, setsize uint64, data uint8) uint64 {
	n := mopsPageLimit(toaddr)
	if n > setsize {
		n = setsize
	}
	if hp := MemHostPtr(c, toaddr, uint(n), AccWrite); hp != nil {
		for i := range hp[:n] {
			hp[i] = data
		}
		return n
	}
	if MemWrite(c, toaddr, 1, uint64(data)) {
		return 1
	}
	return 0
}

func mopsCopyStep(c *CPU, toaddr, fromaddr, copysize uint64, rev bool) uint64 {
	n := mopsPageLimit(toaddr)
	m := mopsPageLimit(fromaddr)
	if rev {
		n = mopsPageLimitRev(toaddr)
		m = mopsPageLimitRev(fromaddr)
	}
	if m < n {
		n = m
	}
	if n > copysize {
		n = copysize
	}
	w := MemHostPtr(c, toaddr, uint(n), AccWrite)
	r := MemHostPtr(c, fromaddr, uint(n), AccRead)
	if rev {
		w = MemHostPtr(c, toaddr-(n-1), uint(n), AccWrite)
		r = MemHostPtr(c, fromaddr-(n-1), uint(n), AccRead)
	}
	if w != nil && r != nil {
		copy(w[:n], r[:n])
		return n
	}
	var b uint64
	if !MemRead(c, fromaddr, 1, &b) {
		return 0
	}
	if MemWrite(c, toaddr, 1, b) {
		return 1
	}
	return 0
}

func mopsSet(c *CPU, insn uint32, stage uint32) {
	rd := bits(insn, 4, 0)
	rn := bits(insn, 9, 5)
	rs := bits(insn, 20, 16)
	data := uint8(c.RegX(rs)) // Rs may be XZR

	if stage == 0 { // SETP
		toaddr := c.RegX(rd)
		setsize := c.RegX(rn)
		if setsize > 0x7fffffffffffffff {
			setsize = 0x7fffffffffffffff
		}
		stagesetsize := mopsPageLimit(toaddr)
		if stagesetsize > setsize {
			stagesetsize = setsize
		}
		for stagesetsize != 0 {
			c.SetX(rd, toaddr) // input format until completion
			c.SetX(rn, setsize)
			step := mopsSetStep(c, toaddr, stagesetsize, data)
			if step == 0 {
				return
			}
			toaddr += step
			setsize -= step
			stagesetsize -= step
		}
		c.SetX(rd, toaddr+setsize)
		c.SetX(rn, 0-setsize)
		c.NZCV = 0 // NZCV=0000: Option A
		return
	}

	xn := c.RegX(rn) // SETM / SETE
	if xn == 0 {
		return // nothing left: NOP, no checks
	}
	if c.NZCV&PSC != 0 {
		c.RaiseSync(ESRMake(ECMOP, mopsISS(insn, true)), 0)
		return
	}
	toaddr := c.RegX(rd) + xn
	setsize := 0 - xn
	if stage == 2 && setsize >= 0x1000 { // SETE takes only the tail
		c.RaiseSync(ESRMake(ECMOP, mopsISS(insn, false)), 0)
		return
	}
	stagesetsize := setsize &^ uint64(0xfff)
	if stage != 1 {
		stagesetsize = setsize
	}
	for stagesetsize != 0 {
		step := mopsSetStep(c, toaddr, setsize, data)
		if step == 0 {
			return
		}
		toaddr += step
		setsize -= step
		if step >= stagesetsize {
			stagesetsize = 0
		} else {
			stagesetsize -= step
		}
		c.SetX(rn, 0-setsize)
	}
}

func mopsCpy(c *CPU, insn uint32, stage uint32, move bool) {
	rd := bits(insn, 4, 0)
	rn := bits(insn, 9, 5)
	rs := bits(insn, 20, 16)

	if stage == 0 { // CPY[F]P
		fwd := true
		toaddr := c.RegX(rd)
		fromaddr := c.RegX(rs)
		copysize := c.RegX(rn)
		if move {
			// Direction: backward only when the source starts below an
			// overlapping destination; non-overlap is IMPDEF-forward.
			if copysize > 0x007fffffffffffff {
				copysize = 0x007fffffffffffff
			}
			fs := fromaddr & 0xffffffffffffff
			ts := toaddr & 0xffffffffffffff
			fe := (fromaddr + copysize) & 0xffffffffffffff
			if fs < ts && fe > ts {
				fwd = false
			}
		} else if copysize > 0x7fffffffffffffff {
			copysize = 0x7fffffffffffffff
		}
		if fwd {
			stagecopysize := mopsPageLimit(toaddr)
			if m := mopsPageLimit(fromaddr); m < stagecopysize {
				stagecopysize = m
			}
			if stagecopysize > copysize {
				stagecopysize = copysize
			}
			for stagecopysize != 0 {
				c.SetX(rd, toaddr) // input format until completion
				c.SetX(rs, fromaddr)
				c.SetX(rn, copysize)
				step := mopsCopyStep(c, toaddr, fromaddr, stagecopysize, false)
				if step == 0 {
					return
				}
				toaddr += step
				fromaddr += step
				copysize -= step
				stagecopysize -= step
			}
			c.SetX(rd, toaddr+copysize)
			c.SetX(rs, fromaddr+copysize)
			c.SetX(rn, 0-copysize)
		} else {
			// Backward: work from the last byte down. The completed-P
			// register format is the same as the input format (Xn stays
			// positive, which is how M/E recognise the direction).
			t := toaddr + copysize - 1
			f := fromaddr + copysize - 1
			stagecopysize := mopsPageLimitRev(t)
			if m := mopsPageLimitRev(f); m < stagecopysize {
				stagecopysize = m
			}
			if stagecopysize > copysize {
				stagecopysize = copysize
			}
			for stagecopysize != 0 {
				c.SetX(rn, copysize)
				step := mopsCopyStep(c, t, f, stagecopysize, true)
				if step == 0 {
					return
				}
				copysize -= step
				stagecopysize -= step
				t -= step
				f -= step
			}
			c.SetX(rn, copysize)
		}
		c.NZCV = 0 // NZCV=0000: Option A
		return
	}

	xn := c.RegX(rn) // CPY[F]M / CPY[F]E
	if xn == 0 {
		return
	}
	if c.NZCV&PSC != 0 {
		c.RaiseSync(ESRMake(ECMOP, mopsISS(insn, true)), 0)
		return
	}
	fwd := !move || int64(xn) < 0
	var toaddr, fromaddr, copysize uint64
	if fwd {
		toaddr = c.RegX(rd) + xn
		fromaddr = c.RegX(rs) + xn
		copysize = 0 - xn
	} else {
		copysize = xn
		toaddr = c.RegX(rd) + copysize - 1
		fromaddr = c.RegX(rs) + copysize - 1
	}
	if stage == 2 && copysize >= 0x1000 { // CPY[F]E takes only the tail
		c.RaiseSync(ESRMake(ECMOP, mopsISS(insn, false)), 0)
		return
	}
	// M runs while a full page remains; E runs to zero.
	for {
		if stage == 1 {
			if copysize < 0x1000 {
				break
			}
		} else if copysize == 0 {
			break
		}
		step := mopsCopyStep(c, toaddr, fromaddr, copysize, !fwd)
		if step == 0 {
			return
		}
		if fwd {
			toaddr += step
			fromaddr += step
		} else {
			toaddr -= step
			fromaddr -= step
		}
		copysize -= step
		if fwd {
			c.SetX(rn, 0-copysize)
		} else {
			c.SetX(rn, copysize)
		}
	}
}

func mops(c *CPU, insn uint32) {
	op1 := bits(insn, 23, 22) // CPY stage; 11 = SET family
	rd := bits(insn, 4, 0)
	rn := bits(insn, 9, 5)
	rs := bits(insn, 20, 16)
	if bits(insn, 31, 30) != 0 {
		undefined(c, insn)
		return
	}
	if op1 == 3 {
		stage := bits(insn, 15, 14) // 0 P, 1 M, 2 E; 3 unallocated
		// bit26 set = SETG* (MTE tag-setting): not implemented, UNDEF.
		// Rd==Rn, Rd==Rs, Rn==Rs, Rd/Rn==31 are CONSTRAINED UNPREDICTABLE;
		// UNDEF as qemu does (Rs==31 is a valid XZR value operand).
		if bit(insn, 26) || stage == 3 ||
			rs == rn || rs == rd || rn == rd || rd == 31 || rn == 31 {
			undefined(c, insn)
			return
		}
		if !mopsEnabledOK(c, insn) {
			return
		}
		mopsSet(c, insn, stage)
		return
	}
	if rs == rn || rs == rd || rn == rd || rd == 31 || rs == 31 || rn == 31 {
		undefined(c, insn)
		return
	}
	if !mopsEnabledOK(c, insn) {
		return
	}
	mopsCpy(c, insn, op1, bit(insn, 26))
}

// ---- load/store forms ----

func ldstRegister(c *CPU, insn uint32) {
	size := bits(insn, 31, 30)
	opc := bits(insn, 23, 22)
	rn := bits(insn, 9, 5)
	rt := bits(insn, 4, 0)
	V := bit(insn, 26)

	var isLoad, isStore bool
	var bytes, scale uint32
	if V {
		isLoad = opc&1 != 0
		// opc<1> selects the 128-bit Q form, defined only for size==0.
		if opc&2 != 0 && size != 0 {
			undefined(c, insn)
			return
		}
		if opc&2 != 0 {
			bytes, scale = 16, 4
		} else {
			bytes, scale = 1<<size, size
		}
	} else {
		isLoad = opc != 0
		bytes, scale = 1<<size, size
	}
	isStore = !isLoad

	var va, base = uint64(0), c.RegXSP(rn)
	wb := 0 // 0 none, 1 post, 2 pre
	switch {
	case bit(insn, 24): // unsigned immediate offset
		va = base + uint64(bits(insn, 21, 10))<<scale
	case bit(insn, 21): // register offset or atomic
		if bits(insn, 11, 10) == 0 && !V { // LSE atomic memory operation
			ldstAtomic(c, insn)
			return
		}
		if bits(insn, 11, 10) != 2 {
			undefined(c, insn)
			return
		}
		va = base + ldstExtendedOffset(c, insn, scale)
	default:
		mode := bits(insn, 11, 10)
		imm9 := int64(SignExtend(uint64(bits(insn, 20, 12)), 9))
		switch {
		case mode == 0 || mode == 2: // unscaled / unprivileged LDTR/STTR
			// No SIMD&FP unprivileged form; at EL0 the integer LDTR/STTR
			// behave exactly like LDUR/STUR.
			if mode == 2 && V {
				undefined(c, insn)
				return
			}
			va = base + uint64(imm9)
		case mode == 1:
			va, wb = base, 1 // post
		default:
			va, wb = base+uint64(imm9), 2 // pre
		}
		if wb == 1 {
			base += uint64(imm9) // post writeback value
		}
	}

	// opc==3 is the "load signed, 32-bit result" column, which exists only for
	// the byte and halfword sizes: size==2 (LDRSW has opc==2) and size==3 are
	// both unallocated.
	if !V && opc == 3 && size >= 2 {
		undefined(c, insn)
		return
	}

	var ok bool
	if V {
		if isStore {
			ok = vregStore(c, rt, va, uint(bytes))
		} else {
			ok = vregLoad(c, rt, va, uint(bytes))
		}
	} else if isStore {
		ok = MemWrite(c, va, uint(bytes), c.RegX(rt))
	} else {
		ok = doLoad(c, rt, va, size, opc)
	}
	if !ok {
		return // faulted: do NOT write back the base (instruction re-executes)
	}
	switch wb {
	case 1:
		c.SetXSP(rn, base)
	case 2:
		c.SetXSP(rn, va)
	}
}

func ldstPair(c *CPU, insn uint32) {
	opc := bits(insn, 31, 30)
	V := bit(insn, 26)
	mode := bits(insn, 25, 23) // 000 STNP/LDNP, 001 post, 010 offset, 011 pre
	L := bit(insn, 22)
	imm7 := int64(SignExtend(uint64(bits(insn, 21, 15)), 7))
	rt2 := bits(insn, 14, 10)
	rn := bits(insn, 9, 5)
	rt := bits(insn, 4, 0)

	var scale, esz uint32
	var signedWord bool
	if V {
		// opc==3 is unallocated and must be rejected before scale is derived:
		// scale = opc+2 would make esz 32, and a 32-byte "element" would reach
		// vreg_load/vreg_store, whose non-16 path passes the size straight
		// through to mem_read/mem_write.
		if opc == 3 {
			undefined(c, insn)
			return
		}
		scale = opc + 2
		esz = 1 << scale // S/D/Q = 4/8/16 bytes
	} else {
		if opc == 3 { // unallocated
			undefined(c, insn)
			return
		}
		scale = 2
		if opc == 2 {
			scale = 3
		}
		esz = 1 << scale
		signedWord = opc == 1
	}
	offset := uint64(imm7) << scale

	base := c.RegXSP(rn)
	var addr uint64
	var wb bool
	var wbval uint64
	switch mode {
	case 0: // STNP/LDNP (non-temporal)
		addr = base + offset
	case 1: // post
		addr, wb, wbval = base, true, base+offset
	case 2: // offset
		addr = base + offset
	case 3: // pre
		addr, wb, wbval = base+offset, true, base+offset
	default:
		undefined(c, insn)
		return
	}

	switch {
	case V:
		var ok bool
		if L {
			ok = vregLoad(c, rt, addr, uint(esz)) && vregLoad(c, rt2, addr+uint64(esz), uint(esz))
		} else {
			ok = vregStore(c, rt, addr, uint(esz)) && vregStore(c, rt2, addr+uint64(esz), uint(esz))
		}
		if !ok {
			return // faulted: skip writeback (instruction re-executes)
		}
	case L:
		if esz == 8 { // LDP Xt: one 16-byte access
			var v V128
			if !MemRead128(c, addr, &v) {
				return
			}
			c.SetX(rt, v.U64(0))
			c.SetX(rt2, v.U64(1))
		} else {
			var a, b uint64
			if !MemRead(c, addr, uint(esz), &a) {
				return
			}
			if !MemRead(c, addr+uint64(esz), uint(esz), &b) {
				return
			}
			if signedWord {
				c.SetX(rt, SignExtend(a, 32))
				c.SetX(rt2, SignExtend(b, 32))
			} else {
				c.SetX(rt, uint64(uint32(a)))
				c.SetX(rt2, uint64(uint32(b)))
			}
		}
	default:
		if esz == 8 { // STP Xt: one 16-byte access
			var v V128
			v.SetU64(0, c.RegX(rt))
			v.SetU64(1, c.RegX(rt2))
			if !MemWrite128(c, addr, &v) {
				return
			}
		} else {
			if !MemWrite(c, addr, uint(esz), c.RegX(rt)) {
				return
			}
			if !MemWrite(c, addr+uint64(esz), uint(esz), c.RegX(rt2)) {
				return
			}
		}
	}
	if wb {
		c.SetXSP(rn, wbval)
	}
}

func ldstLiteral(c *CPU, insn uint32) {
	opc := bits(insn, 31, 30)
	V := bit(insn, 26)
	rt := bits(insn, 4, 0)
	off := SignExtend(uint64(bits(insn, 23, 5)), 19) << 2
	va := c.CurInsnPC + off
	if V { // SIMD&FP: LDR St/Dt/Qt (literal)
		if opc == 3 {
			undefined(c, insn)
			return
		}
		vregLoad(c, rt, va, uint(4)<<opc) // opc 0/1/2 -> 4/8/16 bytes
		return
	}
	var raw uint64
	switch opc {
	case 0: // LDR Wt
		if MemRead(c, va, 4, &raw) {
			c.SetX(rt, uint64(uint32(raw)))
		}
	case 1: // LDR Xt
		if MemRead(c, va, 8, &raw) {
			c.SetX(rt, raw)
		}
	case 2: // LDRSW
		if MemRead(c, va, 4, &raw) {
			c.SetX(rt, SignExtend(raw, 32))
		}
	default: // PRFM literal: nop
	}
}

func ldstExclusive(c *CPU, insn uint32) {
	size := bits(insn, 31, 30)
	o2 := bit(insn, 23)
	L := bit(insn, 22)
	o1 := bit(insn, 21)
	rs := bits(insn, 20, 16)
	rt2 := bits(insn, 14, 10)
	rn := bits(insn, 9, 5)
	rt := bits(insn, 4, 0)
	bytes := uint(1) << size
	va := c.RegXSP(rn)

	switch {
	case o2 && o1: // CAS/CASA/CASL/CASAL (LSE)
		ldstCas(c, insn)
		return
	case !o2 && o1 && !bit(insn, 31): // CASP/CASPA/CASPL/CASPAL (LSE)
		ldstCasp(c, insn)
		return
	case o2: // LDAR / STLR (ordered, non-exclusive)
		// These are the architecture's acquire/release accesses and commonly
		// pair with CAS/exclusives on the same word (locks), so they must be
		// host atomic accesses, not plain copies, or a release store races
		// non-atomically with another thread's CAS and the lock loses updates.
		acc := AccWrite
		if L {
			acc = AccRead
		}
		hp := MemHostPtr(c, va, bytes, acc)
		if L { // LDAR: load-acquire
			var v uint64
			if hp != nil {
				v = hostAtomicLoad(hp, bytes)
				c.SetX(rt, v)
			} else {
				if MemRead(c, va, bytes, &v) {
					c.SetX(rt, v)
				}
			}
			return
		}
		// STLR: store-release
		v := c.RegX(rt)
		if hp != nil {
			hostAtomicStore(hp, bytes, v)
		} else {
			MemWrite(c, va, bytes, v)
		}
		return
	}

	acq := bit(insn, 15) // o0: LDAXR/STLXR ordering
	if !o1 {             // single-register exclusive
		if L { // LDXR / LDAXR
			var v uint64
			if !MemRead(c, va, bytes, &v) {
				return
			}
			c.SetX(rt, v)
			c.ExclValid = true
			c.ExclAddr = va
			c.ExclSize = uint64(bytes)
			c.ExclVal = v // recorded for the STXR compare-and-swap
			if acq {
				atomicBarrier() // LDAXR: acquire
			}
			return
		}
		// STXR / STLXR
		res := uint64(1)
		if c.ExclValid && c.ExclAddr == va && c.ExclSize == uint64(bytes) {
			// SMP-correct: compare-and-swap against the value LDXR loaded. A
			// concurrent write by another guest thread changed memory -> the
			// CAS fails -> STXR fails, exactly as the architecture requires.
			// Falls back to a plain store only when the address cannot be a
			// stable host pointer.
			if acq {
				atomicBarrier() // STLXR: release
			}
			if hp := MemHostPtr(c, va, bytes, AccWrite); hp != nil {
				want := c.RegX(rt)
				_, ok := hostAtomicCAS(hp, bytes, c.ExclVal, want)
				if ok {
					res = 0
				}
			} else {
				if MemWrite(c, va, bytes, c.RegX(rt)) {
					res = 0
				}
			}
		}
		c.SetX(rs, res)
		c.ExclValid = false
		return
	}

	// pair exclusive LDXP / STXP
	if L {
		var a, b uint64
		if !MemRead(c, va, bytes, &a) {
			return
		}
		if !MemRead(c, va+uint64(bytes), bytes, &b) {
			return
		}
		c.SetX(rt, a)
		c.SetX(rt2, b)
		c.ExclValid = true
		c.ExclAddr = va
		c.ExclSize = uint64(bytes) * 2
		c.ExclVal = a
		c.ExclVal2 = b
		if acq {
			atomicBarrier() // LDAXP: acquire
		}
		return
	}
	res := uint64(1)
	if acq {
		atomicBarrier() // STLXP: release
	}
	if c.ExclValid && c.ExclAddr == va && c.ExclSize == uint64(bytes)*2 {
		if bytes == 8 { // 16-byte pair: one 16-byte CAS
			if hp := MemHostPtr(c, va, 16, AccWrite); hp != nil {
				casp16Lock.Lock()
				cur0 := loadLE(hp[:8])
				cur1 := loadLE(hp[8:16])
				if cur0 == c.ExclVal && cur1 == c.ExclVal2 {
					storeLE(hp[:8], c.RegX(rt))
					storeLE(hp[8:16], c.RegX(rt2))
					res = 0
				}
				casp16Lock.Unlock()
			}
		} else { // 8-byte pair: one u64 CAS
			if hp := MemHostPtr(c, va, 8, AccWrite); hp != nil {
				exp := c.ExclVal&0xffffffff | c.ExclVal2<<32
				want := c.RegX(rt)&0xffffffff | c.RegX(rt2)<<32
				if _, ok := hostAtomicCAS(hp, 8, exp, want); ok {
					res = 0
				}
			}
		}
	}
	c.SetX(rs, res)
	c.ExclValid = false
}

// ldstVectorMulti is AdvSIMD load/store multiple structures: LD1/ST1
// (contiguous, 1-4 registers) and LD2/3/4/ST2/3/4 (de-interleaved), with the
// post-indexed form. Used pervasively for memcpy/memset and NEON block I/O.
func ldstVectorMulti(c *CPU, insn uint32) {
	Q := bit(insn, 30)
	post := bit(insn, 23)
	L := bit(insn, 22)
	rm := bits(insn, 20, 16)
	opcode := bits(insn, 15, 12)
	size := bits(insn, 11, 10)
	rn := bits(insn, 9, 5)
	rt := bits(insn, 4, 0)

	var nregs, sel uint32 // sel = interleave factor (1 = contiguous LD1/ST1)
	switch opcode {
	case 0x0:
		nregs, sel = 4, 4 // LD4/ST4
	case 0x2:
		nregs, sel = 4, 1 // LD1/ST1 x4
	case 0x4:
		nregs, sel = 3, 3 // LD3/ST3
	case 0x6:
		nregs, sel = 3, 1 // LD1/ST1 x3
	case 0x7:
		nregs, sel = 1, 1 // LD1/ST1 x1
	case 0x8:
		nregs, sel = 2, 2 // LD2/ST2
	case 0xa:
		nregs, sel = 2, 1 // LD1/ST1 x2
	case 0xc:
		nregs, sel = 1, 2 // LD1R/ST1R? (LD1R is single-structure; see below)
	default:
		undefined(c, insn)
		return
	}
	ebytes := uint32(1) << size
	if size == 3 && !Q {
		undefined(c, insn)
		return
	}
	if size == 3 {
		ebytes = 8
	}
	elems := uint32(8)
	if Q {
		elems = 16
	}
	elems /= uint32(ebytes)
	total := nregs * elems * ebytes

	base := c.RegXSP(rn)
	addr := base
	for e := uint32(0); e < elems; e++ {
		for r := uint32(0); r < nregs; r++ {
			vt := (rt + r) & 31
			off := e * ebytes
			if L {
				var t uint64
				if !MemRead(c, addr, uint(ebytes), &t) {
					return
				}
				c.V[vt].StoreN(int(off), int(ebytes), t)
			} else {
				t := c.V[vt].LoadN(int(off), int(ebytes))
				if !MemWrite(c, addr, uint(ebytes), t) {
					return
				}
			}
			addr += uint64(ebytes)
		}
	}
	if sel != 1 {
		// De-interleaving the loaded bytes: element r of register r' is at
		// offset (e*sel + r) * ebytes in memory. The loop above stored them
		// contiguously, so redistribute (ST2/3/4 are the exact inverse).
		var tmp [4 * 16]byte
		for r := uint32(0); r < nregs; r++ {
			copy(tmp[r*elems*ebytes:], c.V[(rt+r)&31].Bytes()[:elems*ebytes])
		}
		for r := uint32(0); r < nregs; r++ {
			for e := uint32(0); e < elems; e++ {
				src := (e*sel + r) * ebytes
				dst := e * ebytes
				copy(c.V[(rt+r)&31].Bytes()[dst:dst+ebytes], tmp[src:src+ebytes])
			}
		}
	}
	if post {
		inc := uint64(total)
		if rm != 31 {
			inc = c.RegX(rm)
		}
		c.SetXSP(rn, base+inc)
	}
}

// ldstVectorSingle is AdvSIMD load/store single structure: LD1/2/3/4 and
// ST1/2/3/4 to/from a single lane, plus LD1R/2R/3R/4R (load one element and
// replicate it across all lanes). Single-lane loads leave the other lanes of
// the destination unchanged.
func ldstVectorSingle(c *CPU, insn uint32) {
	Q := bit(insn, 30)
	post := bit(insn, 23)
	L := bit(insn, 22)
	R := bit(insn, 21)
	rm := bits(insn, 20, 16)
	opcode := bits(insn, 15, 13)
	S := bit(insn, 12)
	size := bits(insn, 11, 10)
	rn := bits(insn, 9, 5)
	rt := bits(insn, 4, 0)

	scale := opcode >> 1
	selem := ((opcode&1)<<1 | boolToUint(R)) + 1
	replicate := false
	index := uint32(0)
	ebytes := uint32(0)

	switch scale {
	case 0:
		ebytes = 1
		index = boolToUint(Q)<<3 | boolToUint(S)<<2 | size
	case 1:
		if size&1 != 0 {
			undefined(c, insn)
			return
		}
		ebytes = 2
		index = boolToUint(Q)<<2 | boolToUint(S)<<1 | size>>1
	case 2:
		if size&2 != 0 {
			undefined(c, insn)
			return
		}
		if size&1 == 0 {
			ebytes = 4
			index = boolToUint(Q)<<1 | boolToUint(S)
		} else {
			if S {
				undefined(c, insn)
				return
			}
			ebytes = 8
			index = boolToUint(Q)
		}
	default: // scale == 3: replicate
		if !L || S {
			undefined(c, insn)
			return
		}
		replicate = true
		ebytes = 1 << size
	}

	base := c.RegXSP(rn)
	addr := base
	total := selem * ebytes

	for r := uint32(0); r < selem; r++ {
		vt := (rt + r) & 31
		switch {
		case replicate:
			var elem uint64
			if !MemRead(c, addr, uint(ebytes), &elem) {
				return
			}
			var v V128
			v.Clear()
			lanes := uint32(8)
			if Q {
				lanes = 16
			}
			lanes /= ebytes
			for i := uint32(0); i < lanes; i++ {
				v.StoreN(int(i*ebytes), int(ebytes), elem)
			}
			c.V[vt] = v
		case L: // load one lane, rest unchanged
			var elem uint64
			if !MemRead(c, addr, uint(ebytes), &elem) {
				return
			}
			c.V[vt].StoreN(int(index*ebytes), int(ebytes), elem)
		default: // store one lane
			elem := c.V[vt].LoadN(int(index*ebytes), int(ebytes))
			if !MemWrite(c, addr, uint(ebytes), elem) {
				return
			}
		}
		addr += uint64(ebytes)
	}

	if post {
		inc := uint64(total)
		if rm != 31 {
			inc = c.RegX(rm)
		}
		c.SetXSP(rn, base+inc)
	}
}

func loadsStores(c *CPU, insn uint32) {
	b2927 := bits(insn, 29, 27)
	// Ordered by dynamic frequency: the register forms and pairs are far
	// hotter than exclusives/vector-structure/MOPS. The tests key on disjoint
	// b2927 values (and disjoint subfields within 0x1/0x3), so reordering
	// cannot change which handler wins.
	switch {
	case b2927 == 0x7:
		ldstRegister(c, insn)
	case b2927 == 0x5:
		ldstPair(c, insn)
	case b2927 == 0x3 && bits(insn, 25, 24) == 0:
		ldstLiteral(c, insn)
	case b2927 == 0x1 && bits(insn, 26, 24) == 0:
		ldstExclusive(c, insn)
	case b2927 == 0x1 && bit(insn, 26) && !bit(insn, 25) && !bit(insn, 24):
		ldstVectorMulti(c, insn) // AdvSIMD load/store multiple structures
	case b2927 == 0x1 && bit(insn, 26) && !bit(insn, 25) && bit(insn, 24):
		ldstVectorSingle(c, insn) // AdvSIMD load/store single structure
	case b2927 == 0x3 && !bit(insn, 26) && bits(insn, 25, 24) == 1 && !bit(insn, 21) &&
		bits(insn, 11, 10) == 0:
		ldstRcpcUnscaled(c, insn) // FEAT_LRCPC2 LDAPUR/STLUR
	case b2927 == 0x3 && bits(insn, 25, 24) == 1 && !bit(insn, 21) && bits(insn, 11, 10) == 1:
		mops(c, insn) // FEAT_MOPS CPYx/SETx
	default:
		undefined(c, insn)
	}
}

// ================= branches / exceptions / system =================

func branchSystem(c *CPU, insn uint32) {
	top6 := bits(insn, 31, 26)

	switch {
	case top6 == 0x05: // B
		c.PC = c.CurInsnPC + SignExtend(uint64(bits(insn, 25, 0)), 26)<<2
		return
	case top6 == 0x25: // BL
		c.SetX(30, c.CurInsnPC+4)
		c.PC = c.CurInsnPC + SignExtend(uint64(bits(insn, 25, 0)), 26)<<2
		return
	case bits(insn, 31, 24) == 0x54 && !bit(insn, 4): // B.cond
		if c.CondHolds(bits(insn, 3, 0)) {
			c.PC = c.CurInsnPC + SignExtend(uint64(bits(insn, 23, 5)), 19)<<2
		}
		return
	case bits(insn, 30, 25) == 0x1a: // CBZ / CBNZ
		sf := bit(insn, 31)
		op := bit(insn, 24)
		rt := bits(insn, 4, 0)
		v := c.RegXSz(rt, sf)
		take := false
		if op {
			take = v != 0
		} else {
			take = v == 0
		}
		if take {
			c.PC = c.CurInsnPC + SignExtend(uint64(bits(insn, 23, 5)), 19)<<2
		}
		return
	case bits(insn, 30, 25) == 0x1b: // TBZ / TBNZ
		op := bit(insn, 24)
		bitpos := boolToUint(bit(insn, 31))<<5 | bits(insn, 23, 19)
		rt := bits(insn, 4, 0)
		set := (c.RegX(rt) >> bitpos) & 1
		take := false
		if op {
			take = set != 0
		} else {
			take = set == 0
		}
		if take {
			c.PC = c.CurInsnPC + SignExtend(uint64(bits(insn, 18, 5)), 14)<<2
		}
		return
	case bits(insn, 31, 24) == 0xd4: // exception generation
		opc := bits(insn, 23, 21)
		ll := bits(insn, 1, 0)
		imm16 := bits(insn, 20, 5)
		switch {
		case opc == 0 && ll == 1: // SVC
			c.TakeException(ExcSync, ESRMake(ECSVC64, imm16), 0, c.PC)
		case opc == 0 && ll == 2: // HVC
			undefined(c, insn)
		case opc == 0 && ll == 3: // SMC
			undefined(c, insn)
		case opc == 1 && ll == 0: // BRK
			c.RaiseSync(ESRMake(ECBRK64, imm16), 0)
		case opc == 2 && ll == 0: // HLT: stop the machine (test exit)
			fmt.Fprintf(traceOut(c), "[HLT #%d] x0=0x%x icount=%d\n", imm16, c.X[0], c.ICount)
			c.Stop = true
		default:
			undefined(c, insn)
		}
		return
	case bits(insn, 31, 25) == 0x6b: // unconditional branch (register)
		opc := bits(insn, 24, 21)
		rn := bits(insn, 9, 5)
		tgt := c.RegX(rn)
		switch opc {
		case 0: // BR
			c.PC = tgt
		case 1: // BLR
			c.SetX(30, c.CurInsnPC+4)
			c.PC = tgt
		case 2: // RET
			c.PC = tgt
		case 4: // ERET
			spsr := uint32(c.SPSR[c.EL])
			elr := c.ELR[c.EL]
			c.UnpackSPSR(spsr)
			c.PC = elr
		default:
			undefined(c, insn)
		}
		return
	case bits(insn, 31, 24) == 0xd5: // system instructions
		L := bit(insn, 21)
		op0 := bits(insn, 20, 19)
		op1 := bits(insn, 18, 16)
		CRn := bits(insn, 15, 12)
		CRm := bits(insn, 11, 8)
		op2 := bits(insn, 7, 5)
		rt := bits(insn, 4, 0)
		switch {
		case !L && op0 == 0 && op1 == 3 && CRn == 2 && rt == 31: // hints
			if CRm == 0 && op2 == 2 {
				c.Halted = true // WFE
			} else if CRm == 0 && op2 == 3 {
				c.Halted = true // WFI
			}
			// NOP/YIELD/SEV/SEVL/etc -> no-op
			return
		case !L && op0 == 0 && op1 == 3 && CRn == 3 && rt == 31: // barriers/CLREX
			switch op2 {
			case 2: // CLREX
				c.ExclValid = false
			case 4, 5: // DSB / DMB
				// Map the guest barrier to a host fence: without it the host
				// reorders our plain guest loads/stores across threads,
				// breaking the guest memory model.
				atomicBarrier()
			}
			// ISB (op2==6)/SB: no ordering effect on data in this interpreter
			return
		}
		sysregExec(c, insn)
		return
	}
	undefined(c, insn)
}

// ================= top-level dispatch =================

// ExecA64 executes one already-fetched instruction word.
func ExecA64(c *CPU, insn uint32) {
	switch (insn >> 25) & 0xf {
	case 0x8, 0x9: // 100x
		dpImmediate(c, insn)
	case 0xa, 0xb: // 101x
		branchSystem(c, insn)
	case 0x4, 0x6, 0xc, 0xe: // x1x0
		loadsStores(c, insn)
	case 0x5, 0xd: // x101
		dpRegister(c, insn)
	case 0x7, 0xf: // x111
		execFPSIMD(c, insn)
	default: // 00xx reserved
		undefined(c, insn)
	}
}

// ---- small helpers ----

func boolToUint(b bool) uint32 {
	if b {
		return 1
	}
	return 0
}

func widthOf(sf bool) uint {
	if sf {
		return 64
	}
	return 32
}

func vShift(v uint64, n uint) uint64 { return v >> n }

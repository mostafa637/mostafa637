// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Ported to Go from src/core/types.h.

// Package core is the AArch64 CPU core: the register file, the A64 decoder and
// the integer/branch/load-store executor. It is a translation of the C core
// (src/core/*.c) and, like it, knows nothing about Linux user mode: it reaches
// guest memory through the four-function seam in mem.go and reports every
// synchronous event through the exception seam in cpu.go.
package core

import (
	mbits "math/bits"
	"unsafe"
)

// V128 is a 128-bit SIMD/FP register. Host and guest are both little-endian.
//
// The C original is a union of b/h/s/d arrays; Go has no unions, so the
// register keeps its bytes and the lane views are accessors over them. The
// unsafe casts are the same reinterpretation the union did, and the array
// keeps the value in a single allocation with no indirection.
type V128 struct {
	b [16]byte
}

// U64 returns 64-bit lane i (0 or 1).
func (v *V128) U64(i int) uint64 {
	return *(*uint64)(unsafe.Pointer(&v.b[i*8]))
}

// SetU64 stores 64-bit lane i.
func (v *V128) SetU64(i int, x uint64) {
	*(*uint64)(unsafe.Pointer(&v.b[i*8])) = x
}

// U32 returns 32-bit lane i (0..3).
func (v *V128) U32(i int) uint32 {
	return *(*uint32)(unsafe.Pointer(&v.b[i*4]))
}

// SetU32 stores 32-bit lane i.
func (v *V128) SetU32(i int, x uint32) {
	*(*uint32)(unsafe.Pointer(&v.b[i*4])) = x
}

// U16 returns 16-bit lane i (0..7).
func (v *V128) U16(i int) uint16 {
	return *(*uint16)(unsafe.Pointer(&v.b[i*2]))
}

// SetU16 stores 16-bit lane i.
func (v *V128) SetU16(i int, x uint16) {
	*(*uint16)(unsafe.Pointer(&v.b[i*2])) = x
}

// U8 returns byte lane i (0..15).
func (v *V128) U8(i int) uint8 { return v.b[i] }

// SetU8 stores byte lane i.
func (v *V128) SetU8(i int, x uint8) { v.b[i] = x }

// Bytes returns the 16 raw bytes of the register (little-endian).
func (v *V128) Bytes() []byte { return v.b[:] }

// SetBytes copies 16 raw bytes into the register.
func (v *V128) SetBytes(src []byte) { copy(v.b[:], src) }

// Clear zeroes the register: a SIMD load narrower than 128 bits leaves the
// upper bytes zero, which is what every vreg_load path does.
func (v *V128) Clear() {
	v.b = [16]byte{}
}

// LoadN reads `size` (1,2,4,8) little-endian bytes at byte offset off.
func (v *V128) LoadN(off, size int) uint64 {
	switch size {
	case 1:
		return uint64(v.b[off])
	case 2:
		return uint64(*(*uint16)(unsafe.Pointer(&v.b[off])))
	case 4:
		return uint64(*(*uint32)(unsafe.Pointer(&v.b[off])))
	default:
		return *(*uint64)(unsafe.Pointer(&v.b[off]))
	}
}

// StoreN writes `size` (1,2,4,8) little-endian bytes at byte offset off.
func (v *V128) StoreN(off, size int, val uint64) {
	switch size {
	case 1:
		v.b[off] = byte(val)
	case 2:
		*(*uint16)(unsafe.Pointer(&v.b[off])) = uint16(val)
	case 4:
		*(*uint32)(unsafe.Pointer(&v.b[off])) = uint32(val)
	default:
		*(*uint64)(unsafe.Pointer(&v.b[off])) = val
	}
}

// SignExtend sign-extends the low `bits` of x to 64 bits.
func SignExtend(x uint64, bits_ uint) uint64 {
	if bits_ == 0 || bits_ >= 64 {
		return x
	}
	m := uint64(1) << (bits_ - 1)
	return (x ^ m) - m
}

// Ones returns a mask of n ones (all ones for n >= 64).
func Ones(n uint) uint64 {
	if n >= 64 {
		return ^uint64(0)
	}
	return (uint64(1) << n) - 1
}

// Ror32 rotates a 32-bit value right by n.
func Ror32(x uint32, n uint) uint32 {
	n &= 31
	if n == 0 {
		return x
	}
	return x>>n | x<<(32-n)
}

// Ror64 rotates a 64-bit value right by n.
func Ror64(x uint64, n uint) uint64 {
	n &= 63
	if n == 0 {
		return x
	}
	return x>>n | x<<(64-n)
}

// UMulH returns the high 64 bits of an unsigned 64x64 multiply.
//
// Go has no unsigned __int128; math/mbits.Mul64 gives the same 128-bit product
// as a pair of words, which is precisely what the C used it for.
func UMulH(a, b uint64) uint64 {
	hi, _ := mbits.Mul64(a, b)
	return hi
}

// SMulH returns the high 64 bits of a signed 64x64 multiply.
func SMulH(a, b int64) uint64 {
	hi, _ := mbits.Mul64(uint64(a), uint64(b))
	// The high half of a two's-complement product differs from the unsigned
	// one by one copy of each operand's sign, exactly as the C fallback did.
	if a < 0 {
		hi -= uint64(b)
	}
	if b < 0 {
		hi -= uint64(a)
	}
	return hi
}

// ReverseBits reverses the bit order of the low w bits (w is 32 or 64).
func ReverseBits(v uint64, w uint) uint64 {
	return mbits.Reverse64(v) >> (64 - w)
}

// CountlZero returns the number of leading zero bits (w-bit width for w==32,
// counting within 64 bits for w==64, as CLZ does).
func CountlZero(v uint64, w uint) uint64 {
	if w == 32 {
		if uint32(v) == 0 {
			return 32
		}
		return uint64(mbits.LeadingZeros32(uint32(v)))
	}
	if v == 0 {
		return 64
	}
	return uint64(mbits.LeadingZeros64(v))
}

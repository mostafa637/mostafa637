// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Ported to Go from src/core/esr.h.

package core

// Exception classes (ESR_ELx.EC, bits [31:26]).
const (
	ECUnknown     = 0x00
	ECWFx         = 0x01
	ECFPSIMDTrap  = 0x07 // Access to SVE/SIMD/FP trapped (CPACR)
	ECIllegal     = 0x0e
	ECSVC64       = 0x15
	ECHVC64       = 0x16
	ECSMC64       = 0x17
	ECMSRMRS      = 0x18 // Trapped MSR/MRS/System insn
	ECIAbortLower = 0x20
	ECIAbortSame  = 0x21
	ECPCAlign     = 0x22
	ECDAbortLower = 0x24
	ECDAbortSame  = 0x25
	ECSPAlign     = 0x26
	ECMOP         = 0x27 // FEAT_MOPS memory copy/set state mismatch
	ECBRK64       = 0x3c
)

// Data/Instruction fault status codes (DFSC/IFSC, ISS bits [5:0]).
const (
	FSCTransL0  = 0x04
	FSCTransL1  = 0x05
	FSCTransL2  = 0x06
	FSCTransL3  = 0x07
	FSCAccessL0 = 0x08
	FSCAccessL1 = 0x09
	FSCAccessL2 = 0x0a
	FSCAccessL3 = 0x0b
	FSCPermL0   = 0x0c
	FSCPermL1   = 0x0d
	FSCPermL2   = 0x0e
	FSCPermL3   = 0x0f
	FSCExternal = 0x10 // synchronous external abort, not on table walk
	FSCAlign    = 0x21
)

// ESRMake builds an ESR_ELx word: EC in [31:26], IL (32-bit instruction) set,
// ISS in [24:0].
func ESRMake(ec uint, iss uint32) uint64 {
	return uint64(ec&0x3f)<<26 | 1<<25 | uint64(iss&0x1ffffff)
}

// ISSDataAbort builds a data-abort ISS: WnR (write-not-read) and DFSC.
func ISSDataAbort(write bool, dfsc uint) uint32 {
	iss := uint32(dfsc & 0x3f)
	if write {
		iss |= 1 << 6
	}
	return iss
}

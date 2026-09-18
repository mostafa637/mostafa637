// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// The CPU feature set, and the two places it has to be honest about itself.

package core

// Features is what this emulator implements.
//
// A CPU reports what it can do twice over, in two places that must not
// disagree: the ID registers a guest reads with MRS (ID_AA64PFR0_EL1,
// ID_AA64ISAR*_EL1) and the AT_HWCAP/AT_HWCAP2 words the loader puts on the
// stack. The C core hardcodes both to the same literal, which is fine while
// they are a finished CPU's description and a trap while it is being ported:
// an ID register that promises FP makes the guest issue FP instructions the
// executor has not got, and the failure is a SIGILL three million
// instructions into libc, not at the missing feature.
//
// So the two are derived from one table, and the table starts out naming only
// what decode.go actually executes.
var Features = FeatureSet{
	CRC32:  true, // decode.c: CRC32/CRC32C
	LSE:    true, // decode.c: the ARMv8.1-A atomics
	FlagM:  true, // decode.c: CFINV/RMIF/SETF
	FlagM2: true, // decode.c: AXFLAG/XAFLAG
	MOPS:   true, // decode.c: CPYx/SETx
	LRCPC:  true, // decode.c: LDAPR
	ILRCPC: true, // decode.c: LDAPUR/STLUR
	// FP, ASIMD and the crypto/NEON families are set by fpsimd.go once the
	// families that advertise them are ported.
}

// FeatureSet is the set of architectural features this CPU implements.
type FeatureSet struct {
	FP, ASIMD, FP16 bool
	CRC32           bool
	LSE             bool
	FlagM, FlagM2   bool
	MOPS            bool
	LRCPC, ILRCPC   bool
	RDM, DP, FHM    bool
	JSCVT, FCMA     bool
	AES, PMULL      bool
	SHA1, SHA2      bool
	SHA3, SHA512    bool
}

// HWCap returns AT_HWCAP: the HWCAP_* bits a Linux kernel reports for a CPU
// with this feature set.
func (f FeatureSet) HWCap() uint64 {
	var v uint64
	set := func(bit uint, on bool) {
		if on {
			v |= 1 << bit
		}
	}
	set(0, f.FP)      // HWCAP_FP
	set(1, f.ASIMD)   // HWCAP_ASIMD
	set(3, f.AES)     // HWCAP_AES
	set(4, f.PMULL)   // HWCAP_PMULL
	set(5, f.SHA1)    // HWCAP_SHA1
	set(6, f.SHA2)    // HWCAP_SHA2
	set(7, f.CRC32)   // HWCAP_CRC32
	set(8, f.LSE)     // HWCAP_ATOMICS
	set(9, f.FP16)    // HWCAP_FPHP
	set(10, f.FP16)   // HWCAP_ASIMDHP
	set(12, f.RDM)    // HWCAP_ASIMDRDM
	set(13, f.JSCVT)  // HWCAP_JSCVT
	set(14, f.FCMA)   // HWCAP_FCMA
	set(15, f.LRCPC)  // HWCAP_LRCPC
	set(17, f.SHA3)   // HWCAP_SHA3
	set(20, f.DP)     // HWCAP_ASIMDDP
	set(21, f.SHA512) // HWCAP_SHA512
	set(23, f.FHM)    // HWCAP_ASIMDFHM
	set(26, f.ILRCPC) // HWCAP_ILRCPC
	set(27, f.FlagM)  // HWCAP_FLAGM
	return v
}

// HWCap2 returns AT_HWCAP2.
func (f FeatureSet) HWCap2() uint64 {
	var v uint64
	if f.FlagM2 {
		v |= 1 << 7 // HWCAP2_FLAGM2
	}
	if f.MOPS {
		v |= 1 << 18 // HWCAP2_MOPS
	}
	return v
}

// IDPFR0 is ID_AA64PFR0_EL1: the EL0/EL1 AArch64 fields plus the FP and
// AdvSIMD fields that say whether the FP/SIMD registers exist at all.
func (f FeatureSet) IDPFR0() uint64 {
	v := uint64(0x22) // EL0 = EL1 = AArch64 (+AArch32 at EL0)
	if f.FP {
		v |= 1 << 16
	}
	if f.ASIMD {
		v |= 1 << 20
	}
	return v
}

// IDISAR0 is ID_AA64ISAR0_EL1.
func (f FeatureSet) IDISAR0() uint64 {
	var v uint64
	nib := func(shift uint, val uint64) { v |= val << shift }
	aes := uint64(0)
	if f.AES {
		aes = 2 // AES + PMULL
	} else if f.PMULL {
		aes = 1
	}
	sha1 := uint64(0)
	if f.SHA1 {
		sha1 = 1
	}
	sha2 := uint64(0)
	switch {
	case f.SHA512:
		sha2 = 2 // SHA2-512
	case f.SHA2:
		sha2 = 1
	}
	crc := uint64(0)
	if f.CRC32 {
		crc = 1
	}
	lse := uint64(0)
	if f.LSE {
		lse = 2
	}
	rdm := uint64(0)
	if f.RDM {
		rdm = 1
	}
	sha3 := uint64(0)
	if f.SHA3 {
		sha3 = 1
	}
	dp := uint64(0)
	if f.DP {
		dp = 1
	}
	fhm := uint64(0)
	if f.FHM {
		fhm = 1
	}
	ts := uint64(0)
	if f.FlagM2 {
		ts = 2
	} else if f.FlagM {
		ts = 1
	}
	nib(4, aes)
	nib(8, sha1)
	nib(12, sha2)
	nib(16, crc)
	nib(20, lse)
	nib(28, rdm)
	nib(32, sha3)
	nib(44, dp)
	nib(48, fhm)
	nib(52, ts)
	return v
}

// IDISAR1 is ID_AA64ISAR1_EL1.
func (f FeatureSet) IDISAR1() uint64 {
	var v uint64
	lrcpc := uint64(0)
	switch {
	case f.ILRCPC:
		lrcpc = 2 // LDAPUR/STLUR
	case f.LRCPC:
		lrcpc = 1
	}
	if f.JSCVT {
		v |= 1 << 12
	}
	if f.FCMA {
		v |= 1 << 16
	}
	v |= lrcpc << 20
	return v
}

// IDISAR2 is ID_AA64ISAR2_EL1.
func (f FeatureSet) IDISAR2() uint64 {
	if f.MOPS {
		return 1 << 16
	}
	return 0
}

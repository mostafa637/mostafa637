// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Ported to Go from src/core/sysreg.c — MSR/MRS/SYS/MSR-immediate, the ID
// registers and the generic-timer plumbing.

package core

import "fmt"

// System-register keys, KEY(op0, op1, CRn, CRm, op2) from the C original:
//
//	op0<<16 | op1<<13 | CRn<<9 | CRm<<5 | op2
//
// They are written as the shift expression rather than as precomputed numbers
// so the encoding stays legible next to the Arm ARM's Op0/Op1/CRn/CRm/Op2
// columns.
const (
	keyMIDR      = 3<<16 | 0<<13 | 0<<9 | 0<<5 | 0
	keyMPIDR     = 3<<16 | 0<<13 | 0<<9 | 0<<5 | 5
	keyREVIDR    = 3<<16 | 0<<13 | 0<<9 | 0<<5 | 6
	keyCTR       = 3<<16 | 3<<13 | 0<<9 | 0<<5 | 1
	keyDCZID     = 3<<16 | 3<<13 | 0<<9 | 0<<5 | 7
	keyPFR0      = 3<<16 | 0<<13 | 0<<9 | 4<<5 | 0
	keyPFR1      = 3<<16 | 0<<13 | 0<<9 | 4<<5 | 1
	keyDFR0      = 3<<16 | 0<<13 | 0<<9 | 5<<5 | 0
	keyDFR1      = 3<<16 | 0<<13 | 0<<9 | 5<<5 | 1
	keyISAR0     = 3<<16 | 0<<13 | 0<<9 | 6<<5 | 0
	keyISAR1     = 3<<16 | 0<<13 | 0<<9 | 6<<5 | 1
	keyISAR2     = 3<<16 | 0<<13 | 0<<9 | 6<<5 | 2
	keyMMFR0     = 3<<16 | 0<<13 | 0<<9 | 7<<5 | 0
	keyMMFR1     = 3<<16 | 0<<13 | 0<<9 | 7<<5 | 1
	keyMMFR2     = 3<<16 | 0<<13 | 0<<9 | 7<<5 | 2
	keySCTLR     = 3<<16 | 0<<13 | 1<<9 | 0<<5 | 0
	keyACTLR     = 3<<16 | 0<<13 | 1<<9 | 0<<5 | 1
	keyCPACR     = 3<<16 | 0<<13 | 1<<9 | 0<<5 | 2
	keyTTBR0     = 3<<16 | 0<<13 | 2<<9 | 0<<5 | 0
	keyTTBR1     = 3<<16 | 0<<13 | 2<<9 | 0<<5 | 1
	keyTCR       = 3<<16 | 0<<13 | 2<<9 | 0<<5 | 2
	keyMAIR      = 3<<16 | 0<<13 | 10<<9 | 2<<5 | 0
	keyAMAIR     = 3<<16 | 0<<13 | 10<<9 | 3<<5 | 0
	keyVBAR      = 3<<16 | 0<<13 | 12<<9 | 0<<5 | 0
	keyAFSR0     = 3<<16 | 0<<13 | 5<<9 | 1<<5 | 0
	keyAFSR1     = 3<<16 | 0<<13 | 5<<9 | 1<<5 | 1
	keyESR       = 3<<16 | 0<<13 | 5<<9 | 2<<5 | 0
	keyFAR       = 3<<16 | 0<<13 | 6<<9 | 0<<5 | 0
	keyPAR       = 3<<16 | 0<<13 | 7<<9 | 4<<5 | 0
	keyContextID = 3<<16 | 0<<13 | 13<<9 | 0<<5 | 1
	keyTPIDREL1  = 3<<16 | 0<<13 | 13<<9 | 0<<5 | 4
	keyTPIDREL0  = 3<<16 | 3<<13 | 13<<9 | 0<<5 | 2
	keyTPIDRRO   = 3<<16 | 3<<13 | 13<<9 | 0<<5 | 3
	keyMDSCR     = 2<<16 | 0<<13 | 0<<9 | 2<<5 | 2
	keyCurrentEL = 3<<16 | 0<<13 | 4<<9 | 2<<5 | 2
	keySPSel     = 3<<16 | 0<<13 | 4<<9 | 2<<5 | 0
	keyNZCV      = 3<<16 | 3<<13 | 4<<9 | 2<<5 | 0
	keyDAIF      = 3<<16 | 3<<13 | 4<<9 | 2<<5 | 1
	keyFPCR      = 3<<16 | 3<<13 | 4<<9 | 4<<5 | 0
	keyFPSR      = 3<<16 | 3<<13 | 4<<9 | 4<<5 | 1
	keySPSREL1   = 3<<16 | 0<<13 | 4<<9 | 0<<5 | 0
	keyELREL1    = 3<<16 | 0<<13 | 4<<9 | 0<<5 | 1
	keySPEL0     = 3<<16 | 0<<13 | 4<<9 | 1<<5 | 0
	keyCNTFRQ    = 3<<16 | 3<<13 | 14<<9 | 0<<5 | 0
	keyCNTPCT    = 3<<16 | 3<<13 | 14<<9 | 0<<5 | 1
	keyCNTVCT    = 3<<16 | 3<<13 | 14<<9 | 0<<5 | 2
	keyCNTPTVAL  = 3<<16 | 3<<13 | 14<<9 | 2<<5 | 0
	keyCNTPCTL   = 3<<16 | 3<<13 | 14<<9 | 2<<5 | 1
	keyCNTPCVAL  = 3<<16 | 3<<13 | 14<<9 | 2<<5 | 2
	keyCNTVTVAL  = 3<<16 | 3<<13 | 14<<9 | 3<<5 | 0
	keyCNTVCTL   = 3<<16 | 3<<13 | 14<<9 | 3<<5 | 1
	keyCNTVCVAL  = 3<<16 | 3<<13 | 14<<9 | 3<<5 | 2
	keyCNTKCTL   = 3<<16 | 0<<13 | 14<<9 | 1<<5 | 0
)

func sysregKey(op0, op1, crn, crm, op2 uint32) uint32 {
	return op0<<16 | op1<<13 | crn<<9 | crm<<5 | op2
}

// GTCount, when set, is the generic-timer counter source (loop.c's gt_count in
// the C core). By default the counter is the retired instruction count, which
// is the deterministic mode the C core uses unless AE_RTCLOCK is set.
var GTCount func(c *CPU, virt bool) uint64

// FPSRSync, when set, folds pending FP exception flags into c.FPSR before the
// guest reads it (exec_fpsimd.c).
var FPSRSync func(c *CPU)

func timerCount(c *CPU, virt bool) uint64 {
	if GTCount != nil {
		return GTCount(c, virt)
	}
	return c.ICount + c.TimerSkip
}

func msrImmediate(c *CPU, insn uint32) {
	op1 := bits(insn, 18, 16)
	op2 := bits(insn, 7, 5)
	crm := bits(insn, 11, 8)
	switch {
	case op1 == 0 && op2 == 5: // SPSel
		c.SPSel = uint8(crm & 1)
	case op1 == 0 && crm == 0 && op2 == 0: // CFINV (FEAT_FLAGM)
		c.NZCV ^= PSC
	case op1 == 0 && crm == 0 && op2 == 1: // XAFLAG (FEAT_FLAGM2): undo AXFLAG
		zf := uint32(0)
		if c.NZCV&PSZ != 0 {
			zf = 1
		}
		cf := uint32(0)
		if c.NZCV&PSC != 0 {
			cf = 1
		}
		var f uint32
		if cf == 0 && zf == 0 {
			f |= PSN
		}
		if zf == 1 && cf == 1 {
			f |= PSZ
		}
		if cf == 1 || zf == 1 {
			f |= PSC
		}
		if cf == 0 && zf == 1 {
			f |= PSV
		}
		c.NZCV = f
	case op1 == 0 && crm == 0 && op2 == 2: // AXFLAG (FEAT_FLAGM2): x86-style CF/ZF
		zf := uint32(0)
		if c.NZCV&PSZ != 0 {
			zf = 1
		}
		cf := uint32(0)
		if c.NZCV&PSC != 0 {
			cf = 1
		}
		vf := uint32(0)
		if c.NZCV&PSV != 0 {
			vf = 1
		}
		var f uint32
		if zf == 1 || vf == 1 {
			f |= PSZ
		}
		if cf == 1 && vf == 0 {
			f |= PSC
		}
		c.NZCV = f
	case op1 == 3 && op2 == 6: // DAIFSet
		c.DAIF |= crm & 0xf << 6
	case op1 == 3 && op2 == 7: // DAIFClr
		c.DAIF &= ^(crm & 0xf << 6)
	}
	// PAN/UAO/DIT/etc: ignored
}

// sysOp is SYS: TLBI*, DC/IC maintenance.
func sysOp(c *CPU, insn uint32, op1, crn, crm, op2, rt uint32) {
	if crn == 8 { // TLBI *
		SysregFlush()
		return
	}
	if crn == 7 {
		if crm == 4 && op2 == 1 { // DC ZVA: zero 64 bytes
			base := c.RegX(rt) &^ uint64(63)
			// One translation covers the line (64-aligned: never crosses a
			// page): a writable hit zeroes it directly; a miss falls back to
			// the 8x write loop, which raises the fault identically.
			if hp := MemHostPtr(c, base, 64, AccWrite); hp != nil {
				for i := range hp[:64] {
					hp[i] = 0
				}
				return
			}
			for i := uint64(0); i < 64; i += 8 {
				if !MemWrite(c, base+i, 8, 0) {
					return
				}
			}
		}
		return // IC/DC clean/invalidate: no-op (flat memory)
	}
}

// doMRS is MRS: read a system register into Rt.
func doMRS(c *CPU, key, rt uint32) {
	var v uint64
	switch key {
	// --- identification ---
	case keyMIDR:
		v = 0x411fd070 // cortex-a57-ish
	case keyMPIDR:
		v = c.MPIDR
	case keyREVIDR:
		v = 0
	case keyCTR:
		v = 0x8444c004
	case keyDCZID:
		v = 4 // BS=4 (64B), DZP=0
	case keyPFR0:
		// EL0/EL1 = AArch64 (+AArch32 at EL0), with the FP and AdvSIMD
		// fields derived from what is actually implemented.
		v = Features.IDPFR0()
	case keyPFR1:
		v = 0
	case keyDFR0:
		v = 0x10305106 // QEMU cortex-a57
	case keyDFR1:
		v = 0
	case keyISAR0:
		v = Features.IDISAR0()
	case keyISAR1:
		v = Features.IDISAR1()
	case keyISAR2:
		v = Features.IDISAR2()
	case keyMMFR0:
		v = 0x0F001124 // TGran64=0xF (64K unimplemented), else QEMU cortex-a57
	case keyMMFR1, keyMMFR2:
		v = 0

	// --- control / translation ---
	case keySCTLR:
		v = c.SCTLR[1]
	case keyACTLR:
		v = 0
	case keyCPACR:
		v = c.CPACR
	case keyTTBR0:
		v = c.TTBR0[1]
	case keyTTBR1:
		v = c.TTBR1[1]
	case keyTCR:
		v = c.TCR[1]
	case keyMAIR:
		v = c.MAIR[1]
	case keyAMAIR:
		v = c.AMAIR[1]
	case keyVBAR:
		v = c.VBAR[1]
	case keyAFSR0, keyAFSR1:
		v = 0
	case keyESR:
		v = c.ESR[1]
	case keyFAR:
		v = c.FAR[1]
	case keyPAR:
		v = c.PAR
	case keyContextID:
		v = c.ContextIDR
	case keyTPIDREL1:
		v = c.TPIDR[1]
	case keyTPIDREL0:
		v = c.TPIDR[0]
	case keyTPIDRRO:
		v = c.TPIDRRO
	case keyMDSCR:
		v = c.MDSCR

	// --- PSTATE views ---
	case keyCurrentEL:
		v = uint64(c.EL) << 2
	case keySPSel:
		v = uint64(c.SPSel)
	case keyNZCV:
		v = uint64(c.NZCV & 0xf0000000)
	case keyDAIF:
		v = uint64(c.DAIF & (PSD | PSA | PSI | PSF))
	case keyFPCR:
		v = uint64(c.FPCR)
	case keyFPSR:
		if FPSRSync != nil {
			FPSRSync(c) // fold pending flags
		}
		v = uint64(c.FPSR)
	case keySPSREL1:
		v = c.SPSR[1]
	case keyELREL1:
		v = c.ELR[1]
	case keySPEL0:
		v = c.SPEl[0]

	// --- generic timer ---
	case keyCNTFRQ:
		v = c.CNTFRQ
	case keyCNTPCT:
		v = timerCount(c, false)
	case keyCNTVCT:
		v = timerCount(c, true)
	case keyCNTPTVAL:
		v = uint64(int64(int32(uint32(c.CNTPCVal - timerCount(c, false)))))
	case keyCNTPCTL:
		v = c.CNTPCtl & 3
		if c.CNTPCtl&1 != 0 && timerCount(c, false) >= c.CNTPCVal {
			v |= 4 // ISTATUS
		}
	case keyCNTPCVAL:
		v = c.CNTPCVal
	case keyCNTVTVAL:
		v = uint64(int64(int32(uint32(c.CNTVCVal - timerCount(c, true)))))
	case keyCNTVCTL:
		v = c.CNTVCTL & 3
		if c.CNTVCTL&1 != 0 && timerCount(c, true) >= c.CNTVCVal {
			v |= 4
		}
	case keyCNTVCVAL:
		v = c.CNTVCVal
	case keyCNTKCTL:
		v = c.CNTKCTL

	default: // PMU / misc: read as 0
		if c.Trace != nil {
			fmt.Fprintf(c.Trace, "[sysreg] MRS unimpl key=0x%x pc=0x%x -> 0\n", key, c.CurInsnPC)
		}
		v = 0
	}
	c.SetX(rt, v)
}

// doMSR is MSR: write Rt to a system register.
func doMSR(c *CPU, key, rt uint32) {
	v := c.RegX(rt)
	switch key {
	case keySCTLR:
		c.SCTLR[1] = v
		SysregFlush()
	case keyCPACR:
		c.CPACR = v
	case keyTTBR0:
		c.TTBR0[1] = v
		SysregFlush()
	case keyTTBR1:
		c.TTBR1[1] = v
		SysregFlush()
	case keyTCR:
		c.TCR[1] = v
		SysregFlush()
	case keyMAIR:
		c.MAIR[1] = v
	case keyAMAIR:
		c.AMAIR[1] = v
	case keyVBAR:
		c.VBAR[1] = v
	case keyESR:
		c.ESR[1] = v
	case keyFAR:
		c.FAR[1] = v
	case keyPAR:
		c.PAR = v
	case keyContextID:
		c.ContextIDR = v
	case keyTPIDREL1:
		c.TPIDR[1] = v
	case keyTPIDREL0:
		c.TPIDR[0] = v
	case keyTPIDRRO:
		c.TPIDRRO = v
	case keyMDSCR:
		c.MDSCR = v

	case keySPSel:
		c.SPSel = uint8(v & 1)
	case keyNZCV:
		c.NZCV = uint32(v & 0xf0000000)
	case keyDAIF:
		c.DAIF = uint32(v & (PSD | PSA | PSI | PSF))
	case keyFPCR:
		c.FPCR = uint32(v)
	case keyFPSR:
		if FPSRSync != nil {
			FPSRSync(c) // discard pending host/soft flags
		}
		c.FPSR = uint32(v)
	case keySPSREL1:
		c.SPSR[1] = v
	case keyELREL1:
		c.ELR[1] = v
	case keySPEL0:
		c.SPEl[0] = v

	case keyCNTFRQ:
		c.CNTFRQ = v
	case keyCNTPTVAL:
		c.CNTPCVal = timerCount(c, false) + uint64(int64(int32(uint32(v))))
	case keyCNTPCTL:
		c.CNTPCtl = v
	case keyCNTPCVAL:
		c.CNTPCVal = v
	case keyCNTVTVAL:
		c.CNTVCVal = timerCount(c, true) + uint64(int64(int32(uint32(v))))
	case keyCNTVCTL:
		c.CNTVCTL = v
	case keyCNTVCVAL:
		c.CNTVCVal = v
	case keyCNTKCTL:
		c.CNTKCTL = v

	default:
		if c.Trace != nil {
			fmt.Fprintf(c.Trace, "[sysreg] MSR unimpl key=0x%x val=0x%x pc=0x%x\n",
				key, v, c.CurInsnPC)
		}
	}
}

// sysregExec decodes and executes MSR/MRS/SYS/SYSL/MSR-immediate.
func sysregExec(c *CPU, insn uint32) {
	L := bit(insn, 21)
	op0 := bits(insn, 20, 19)
	op1 := bits(insn, 18, 16)
	crn := bits(insn, 15, 12)
	crm := bits(insn, 11, 8)
	op2 := bits(insn, 7, 5)
	rt := bits(insn, 4, 0)

	switch {
	case op0 == 0: // MSR (immediate)
		msrImmediate(c, insn)
		return
	case op0 == 1: // SYS / SYSL
		if !L {
			sysOp(c, insn, op1, crn, crm, op2, rt)
		} else {
			c.SetX(rt, 0) // SYSL reads -> 0 (AT is not modelled)
		}
		return
	}
	key := sysregKey(op0, op1, crn, crm, op2)
	if L {
		doMRS(c, key, rt)
	} else {
		doMSR(c, key, rt)
	}
}

// sysregInit installs the reset values of the system registers the core owns.
func sysregInit(c *CPU) {
	// RES1 bits set, MMU/caches off; MSCEn (bit 33) set, because the kernel
	// role enables EL0 MOPS when it advertises HWCAP2_MOPS.
	c.SCTLR[1] = 0x00C50838 | 1<<33
	c.CPACR = 0
	c.MDSCR = 0
	if FPSRSync != nil {
		FPSRSync(c)
		c.FPSR = 0 // drop pre-reset FP flags
	}
}

// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Ported to Go from src/core/cpu.h, src/core/cpu.c and src/exception.c.

package core

import (
	"fmt"
	"io"
)

// PSTATE condition flag bit positions (matching SPSR_ELx / NZCV layout).
const (
	PSN = 1 << 31
	PSZ = 1 << 30
	PSC = 1 << 29
	PSV = 1 << 28
)

// DAIF mask bits (as in PSTATE/SPSR bits [9:6]).
const (
	PSD = 1 << 9
	PSA = 1 << 8
	PSI = 1 << 7
	PSF = 1 << 6
)

// ExcKind names the four exception entry kinds.
type ExcKind int

const (
	ExcSync   ExcKind = iota // EXC_SYNC
	ExcIRQ                   // EXC_IRQ
	ExcFIQ                   // EXC_FIQ
	ExcSError                // EXC_SERROR
)

// StepResult is the outcome of one CPU step.
type StepResult int

const (
	StepOK   StepResult = iota
	StepHalt            // fatal: unimplemented / stop the machine
)

// AccType is the access type passed to the memory seam.
type AccType int

const (
	AccRead AccType = iota
	AccWrite
	AccExec
)

// PendExc is a pending synchronous exception, recorded by the exception seam
// and drained by the run loop. In the system emulator exception_take vectors to
// EL1; in user mode there is no guest kernel, so the event is parked here and
// loop.go decides what it means (SVC -> syscall, abort/undef/BRK -> signal).
type PendExc struct {
	Valid bool
	ESR   uint64
	FAR   uint64
}

// ---- the memory seam ------------------------------------------------------
//
// The core performs every guest memory access through these. The C core calls
// them directly (they live in mem.c); Go cannot have mem import core and core
// import mem, so they are function variables that mem/ installs at init and
// the core calls through. Same four-plus-one surface, same contract: false (or
// nil) means a fault was raised and recorded, and the caller must abort the
// instruction without applying writeback.

var (
	// MemRead reads 1,2,4 or 8 bytes (any byte count is allowed for the
	// page-split fragments the callers produce).
	MemRead func(c *CPU, va uint64, size uint, out *uint64) bool
	// MemWrite writes 1,2,4 or 8 bytes.
	MemWrite func(c *CPU, va uint64, size uint, val uint64) bool
	// MemRead128 reads 16 bytes.
	MemRead128 func(c *CPU, va uint64, out *V128) bool
	// MemWrite128 writes 16 bytes.
	MemWrite128 func(c *CPU, va uint64, val *V128) bool
	// MemIFetch is the instruction-fetch fast path.
	MemIFetch func(c *CPU, va uint64, insn *uint32) bool
	// MemHostPtr returns a stable host slice for [va, va+size) when the whole
	// range lies in one guest page and is permitted for acc, nil otherwise.
	MemHostPtr func(c *CPU, va uint64, size uint, acc AccType) []byte
	// TLBFlushAll drops every cached translation. Installed by mem/; called
	// through SysregFlush by the system-register layer (TLBI*, SCTLR, TTBR).
	TLBFlushAll func()
)

// SysregFlush invalidates cached translations. Safe to call before mem/ has
// installed itself.
func SysregFlush() {
	if TLBFlushAll != nil {
		TLBFlushAll()
	}
}

// ---- per-thread state -----------------------------------------------------
//
// Everything the C core keeps in thread-local storage: the fetch cache, the
// data TLB and the pending exception. One of these per guest thread, and the
// CPU points at its own, so there is no TLS and no hidden global.

const DTLBEntries = 1024

// DTlbEntry is one direct-mapped data-TLB slot: the guest page number is the
// tag, Data is that page's host backing, Prot its PTE_R/W/X bits.
//
// The C entry is {u64 page; uintptr_t pte} with the host pointer and the
// protection bits packed into the low bits of one word, because generated JIT
// code probes it. Go holds the backing as a slice instead: storing a bare
// uintptr would not keep the allocation alive, and nothing here is read by
// generated code.
type DTlbEntry struct {
	Page uint64
	Data []byte
	Prot uint32
}

// PerThread is the thread-local state of one guest thread.
type PerThread struct {
	// Fetch cache: the host backing of the page currently being fetched from.
	// FData nil means "no cached page".
	FPage uint64
	FData []byte

	// Data TLB and the address-space generation it reflects.
	DTLBGen uint64
	DTLB    [DTLBEntries]DTlbEntry

	// Pending exception (the exception seam).
	PendExc PendExc

	// Opaque hook for the layer above (linux-user: *Task).
	Owner any
}

// ---- CPU ------------------------------------------------------------------

// CPU is the AArch64 register file and the machine state the core executes
// against.
type CPU struct {
	// General purpose. X[31] is reserved; use the Reg/Set helpers for the
	// XZR/SP semantics.
	X  [31]uint64
	PC uint64

	// PSTATE
	NZCV  uint32 // uses PS_N/Z/C/V bit positions
	DAIF  uint32 // uses PS_D/A/I/F bit positions
	EL    uint8  // current exception level 0..3 (we implement 0 and 1)
	SPSel uint8  // SPSel: 0 => SP_EL0, 1 => SP_ELx

	// Banked stack pointers SP_EL0..SP_EL3
	SPEl [4]uint64

	// SIMD/FP
	V         [32]V128
	FPCR      uint32
	FPSR      uint32
	FPTrapped bool // set if CPACR/CPTR disables FP

	// Banked system registers (index by EL where meaningful)
	SCTLR [4]uint64
	TTBR0 [4]uint64
	TTBR1 [4]uint64
	TCR   [4]uint64
	MAIR  [4]uint64
	AMAIR [4]uint64
	VBAR  [4]uint64
	ESR   [4]uint64
	FAR   [4]uint64
	ELR   [4]uint64
	SPSR  [4]uint64
	TPIDR [4]uint64 // TPIDR_ELx

	TPIDRRO    uint64
	ContextIDR uint64
	CPACR      uint64
	MDSCR      uint64
	PAR        uint64

	// Identification / affinity
	MPIDR uint64

	// Generic timer
	CNTFRQ     uint64
	CNTPCtl    uint64
	CNTPCVal   uint64
	CNTVCTL    uint64
	CNTVCVal   uint64
	CNTKCTL    uint64
	CNTVOFF    int64
	CNTPCTBase uint64
	TimerSkip  uint64

	// Exclusive monitor. The values LDXR/LDXP loaded are kept so STXR/STXP can
	// do a real compare-and-swap (SMP-correct across guest threads on weakly
	// ordered hosts), not just an address match.
	ExclValid bool
	ExclAddr  uint64
	ExclSize  uint64
	ExclVal   uint64
	ExclVal2  uint64

	// Interrupt input lines.
	IRQLine bool
	FIQLine bool

	// Run control
	Halted       bool // WFI/WFE: waiting for an event
	Stop         bool // machine should terminate
	ResetPending bool // PSCI SYSTEM_RESET: warm-reboot the machine, don't exit
	ICount       uint64

	// CurInsnPC is the address of the instruction currently executing. It is
	// recorded before the fetch so that a faulting fetch reports the faulting
	// address, not the previously executed instruction.
	CurInsnPC uint64

	// Thread is this CPU's per-thread state (fetch cache, D-TLB, pending
	// exception). Never nil while the CPU runs.
	Thread *PerThread

	// Trace, when non-nil, receives a line per executed instruction (the C
	// core's -d/g_trace).
	Trace io.Writer
	// StepHook, when non-nil, runs before every instruction.
	StepHook func(c *CPU, insn uint32)
}

// NewCPU returns a CPU bound to a freshly allocated PerThread.
func NewCPU() *CPU {
	return &CPU{Thread: &PerThread{}}
}

// CurSP returns a pointer to the currently selected stack pointer.
func (c *CPU) CurSP() *uint64 {
	if c.SPSel != 0 {
		return &c.SPEl[c.EL]
	}
	return &c.SPEl[0]
}

// RegX reads a GPR with 31 meaning XZR (zero).
func (c *CPU) RegX(n uint32) uint64 {
	if n == 31 {
		return 0
	}
	return c.X[n]
}

// SetX writes a GPR with 31 meaning discard.
func (c *CPU) SetX(n uint32, v uint64) {
	if n != 31 {
		c.X[n] = v
	}
}

// RegXSP reads a GPR with 31 meaning SP.
func (c *CPU) RegXSP(n uint32) uint64 {
	if n == 31 {
		return *c.CurSP()
	}
	return c.X[n]
}

// SetXSP writes a GPR with 31 meaning SP.
func (c *CPU) SetXSP(n uint32, v uint64) {
	if n == 31 {
		*c.CurSP() = v
		return
	}
	c.X[n] = v
}

// RegXSz reads a GPR truncated to 32 bits when is64 is false.
func (c *CPU) RegXSz(n uint32, is64 bool) uint64 {
	v := c.RegX(n)
	if is64 {
		return v
	}
	return uint64(uint32(v))
}

// SetXSz writes a GPR, truncating to 32 bits when is64 is false.
func (c *CPU) SetXSz(n uint32, is64 bool, v uint64) {
	if !is64 {
		v = uint64(uint32(v))
	}
	c.SetX(n, v)
}

// PackSPSR builds a PSTATE/SPSR word from the current state.
func (c *CPU) PackSPSR() uint32 {
	s := c.NZCV & (PSN | PSZ | PSC | PSV)
	s |= c.DAIF & (PSD | PSA | PSI | PSF)
	s |= uint32(c.EL)<<2 | uint32(c.SPSel&1)
	return s
}

// UnpackSPSR applies a PSTATE/SPSR word (as ERET and the signal layer do).
func (c *CPU) UnpackSPSR(spsr uint32) {
	c.NZCV = spsr & (PSN | PSZ | PSC | PSV)
	c.DAIF = spsr & (PSD | PSA | PSI | PSF)
	m := spsr & 0xf
	c.EL = uint8((m >> 2) & 3)
	c.SPSel = uint8(m & 1)
}

// Reset puts the CPU into its reset state at EL `el` with PC = entry.
func (c *CPU) Reset(entry uint64, el uint) {
	th := c.Thread
	*c = CPU{}
	c.Thread = th
	c.PC = entry
	c.EL = uint8(el)
	c.SPSel = 1                    // SPSel=1 at reset (use SP_ELx)
	c.DAIF = PSD | PSA | PSI | PSF // all masked at reset
	c.NZCV = 0
	c.CNTFRQ = 24000000 // 24 MHz, as the system emulator's DTB
	c.MPIDR = 1 << 31   // RES1, single core affinity 0
	sysregInit(c)
}

// CondHolds evaluates condition code cond (0..15) against the current NZCV.
func (c *CPU) CondHolds(cond uint32) bool {
	N := c.NZCV&PSN != 0
	Z := c.NZCV&PSZ != 0
	C := c.NZCV&PSC != 0
	V := c.NZCV&PSV != 0

	var r bool
	switch cond >> 1 {
	case 0:
		r = Z // EQ/NE
	case 1:
		r = C // CS/CC
	case 2:
		r = N // MI/PL
	case 3:
		r = V // VS/VC
	case 4:
		r = C && !Z // HI/LS
	case 5:
		r = N == V // GE/LT
	case 6:
		r = (N == V) && !Z // GT/LE
	default:
		r = true // AL/NV
	}
	if cond&1 != 0 && cond != 0xf {
		r = !r
	}
	return r
}

// TakeException records an exception of `kind` and sets the resume address.
//
// Nothing ever vectors to EL1 in user mode: the event is parked in the
// thread's PendExc and the run loop dispatches it. SVC passes the address of
// the next instruction (execution continues there once the syscall returns);
// faults pass the faulting instruction, because Linux re-executes it when a
// handler repairs the situation.
func (c *CPU) TakeException(kind ExcKind, esr, far, retAddr uint64) {
	c.Thread.PendExc.Valid = true
	c.Thread.PendExc.ESR = esr
	c.Thread.PendExc.FAR = far
	c.PC = retAddr
	c.ExclValid = false
}

// RaiseSync raises a synchronous exception at the current instruction.
func (c *CPU) RaiseSync(esr, far uint64) {
	c.TakeException(ExcSync, esr, far, c.CurInsnPC)
}

// Pending returns the thread's pending exception.
func (c *CPU) Pending() *PendExc { return &c.Thread.PendExc }

// Step fetches, decodes and executes one instruction.
func (c *CPU) Step() StepResult {
	if c.Stop {
		return StepHalt
	}
	// Pending IRQ delivery (single-CPU). FIQ handled the same way.
	if c.FIQLine && c.DAIF&PSF == 0 {
		c.TakeException(ExcFIQ, 0, 0, c.PC)
		return StepOK
	}
	if c.IRQLine && c.DAIF&PSI == 0 {
		c.TakeException(ExcIRQ, 0, 0, c.PC)
		return StepOK
	}
	if c.Halted {
		return StepOK // WFI/WFE: caller waits for an event
	}

	// Record the current instruction address BEFORE the fetch: on an
	// instruction abort the ESR/FAR must name the faulting address.
	c.CurInsnPC = c.PC

	var insn uint32
	if !MemIFetch(c, c.PC, &insn) {
		return StepOK // fetch raised an abort
	}
	if c.StepHook != nil {
		c.StepHook(c, insn)
	}
	if c.Trace != nil {
		fmt.Fprintf(c.Trace, "%016x: %08x  [el%d nzcv=%c%c%c%c]\n", c.PC, insn, c.EL,
			flag(c.NZCV&PSN, 'N'), flag(c.NZCV&PSZ, 'Z'),
			flag(c.NZCV&PSC, 'C'), flag(c.NZCV&PSV, 'V'))
	}
	c.PC += 4
	ExecA64(c, insn)
	c.ICount++
	if c.Stop {
		return StepHalt
	}
	return StepOK
}

func flag(v uint32, ch byte) byte {
	if v != 0 {
		return ch
	}
	return '.'
}

// Dump writes the register file to w.
func (c *CPU) Dump(w io.Writer) {
	for i := 0; i < 31; i++ {
		fmt.Fprintf(w, "x%-2d=%016x%s", i, c.X[i], map[bool]string{true: "\n", false: "  "}[i%4 == 3])
	}
	fmt.Fprintf(w, "\nsp =%016x  pc =%016x  el=%d  spsel=%d\n",
		*c.CurSP(), c.PC, c.EL, c.SPSel)
	fmt.Fprintf(w, "nzcv=%c%c%c%c daif=%c%c%c%c  icount=%d\n",
		flag32(c.NZCV, PSN, 'N'), flag32(c.NZCV, PSZ, 'Z'), flag32(c.NZCV, PSC, 'C'), flag32(c.NZCV, PSV, 'V'),
		flag32(c.DAIF, PSD, 'D'), flag32(c.DAIF, PSA, 'A'), flag32(c.DAIF, PSI, 'I'), flag32(c.DAIF, PSF, 'F'),
		c.ICount)
}

func flag32(v, mask uint32, ch byte) byte {
	if v&mask != 0 {
		return ch
	}
	return '.'
}

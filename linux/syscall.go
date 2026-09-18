// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Ported to Go from src/syscall.c.
//
// Syscall dispatch: the arm64 ABI is x8 = number, x0..x5 = arguments, and the
// result (or -errno) goes back in x0. Handlers are grouped by area in
// sys_*.go, mirroring the C split; anything not implemented returns -ENOSYS.

package linux

import (
	"fmt"
	"os"
	"sort"
	"sync"

	"github.com/mostafa637/mostafa637/core"
	"github.com/mostafa637/mostafa637/jit"
)

// Handler is one syscall implementation. It returns the guest's x0: a value,
// or -errno.
type Handler func(t *Task, a [6]uint64) uint64

var handlers = map[uint64]Handler{}

// Register adds handlers to the table (used by the sys_*.go files).
func Register(m map[uint64]Handler) {
	for k, v := range m {
		handlers[k] = v
	}
}

var warnedOnce sync.Map

// Dispatch runs the syscall the guest has set up.
//
// A pending exception of class EC_SVC64 lands here from the run loop with x8
// holding the number. Unimplemented calls answer -ENOSYS, which is what a
// kernel built without the call does; the one-shot warning names it once per
// process, because a libc that probes and falls back is not a failure and a
// call the guest really needed is worth seeing.
func (t *Task) Dispatch() {
	c := t.CPU
	nr := c.X[8]
	var a [6]uint64
	copy(a[:], c.X[0:6])

	ret := t.call(nr, a)
	c.X[0] = ret

	if t.Strace != nil {
		fmt.Fprintf(t.Strace, "[%d] %s(%s) = ", t.TID, name(nr), argsStr(a))
		if int64(ret) < 0 && int64(ret) > -4096 {
			fmt.Fprintf(t.Strace, "-%d\n", -int64(ret))
		} else {
			fmt.Fprintf(t.Strace, "0x%x\n", ret)
		}
	}
}

func (t *Task) call(nr uint64, a [6]uint64) uint64 {
	h, ok := handlers[nr]
	if !ok {
		if t.Warn != nil {
			if _, dup := warnedOnce.LoadOrStore(nr, true); !dup {
				fmt.Fprintf(t.Warn, "arm64chroot: unimplemented syscall %d (%s)\n", nr, name(nr))
			}
		}
		return negErrno(38) // -ENOSYS
	}
	return h(t, a)
}

func name(nr uint64) string {
	if n, ok := syscallNames[nr]; ok {
		return n
	}
	return fmt.Sprintf("sys_%d", nr)
}

func argsStr(a [6]uint64) string {
	return fmt.Sprintf("0x%x, 0x%x, 0x%x, 0x%x, 0x%x, 0x%x",
		a[0], a[1], a[2], a[3], a[4], a[5])
}

// HandledSyscalls lists the numbers this port implements, for the smoke tests
// and for --help.
func HandledSyscalls() []string {
	var out []string
	for nr := range handlers {
		out = append(out, name(nr))
	}
	sort.Strings(out)
	return out
}

// Run is the run loop: execute guest instructions, and whenever the core
// records a synchronous event, dispatch it.
//
// It is loop.c minus the parts that need a second thread (signals delivered
// from other threads, ptrace stops, the JIT's block loop): the shape —
// step, then drain pend_exc — is the same, and each of those is a later file in
// this port.
func (t *Task) Run() int {
	c := t.CPU
	for {
		if c.Stop {
			break
		}
		if p := c.Pending(); p.Valid {
			p.Valid = false
			t.handleException(p)
			if c.Stop {
				break
			}
			continue
		}
		// Signals are delivered at an instruction boundary, between two
		// steps: a handler is ordinary guest code, and the only state it can
		// observe is a complete instruction's worth.
		t.checkSignals()
		if c.Stop {
			break
		}
		// The JIT, when it is on, runs whole basic blocks and hands control
		// back at the first instruction it has no code for.
		if t.JIT != nil && t.runJIT() {
			continue
		}
		if c.Step() == core.StepHalt {
			break
		}
	}
	if t.Exiting {
		return t.ExitCode
	}
	return 0
}

// handleException is what loop.c does with a pending exception: SVC goes to the
// syscall layer, and every other class is a fault the guest would take as a
// signal. Signal delivery is a later file in this port, so a fault ends the
// process with the diagnostic a kernel's default action would print.
func (t *Task) handleException(p *core.PendExc) {
	c := t.CPU
	ec := uint32(p.ESR >> 26)
	switch ec {
	case core.ECSVC64:
		t.Dispatch()
	case core.ECDAbortLower, core.ECDAbortSame:
		// A write fault past the stack limit is the stack overflow the guest
		// asked for (si_code SEGV_ACCERR would be a permissions problem).
		t.raiseSyncSignal(11, 2 /*SEGV_ACCERR*/, p.FAR)
	case core.ECIAbortLower, core.ECIAbortSame:
		t.raiseSyncSignal(11, 2, p.FAR)
	case core.ECPCAlign, core.ECSPAlign:
		t.raiseSyncSignal(7, 1 /*BUS_ADRALN*/, p.FAR)
	case core.ECBRK64:
		t.raiseSyncSignal(5, 1 /*TRAP_BRKPT*/, c.PC)
	case core.ECMOP:
		t.mopReset(p.ESR)
	default:
		fatalf(t, "SIGILL", c.PC)
	}
}

// mopReset is the kernel's do_el0_mops: it puts the registers back in prologue
// input format and restarts at the prologue, which is architecturally
// consecutive (P/M/E) for exactly this reason.
func (t *Task) mopReset(esr uint64) {
	c := t.CPU
	wrongOption := (esr>>17)&1 != 0
	optionA := (esr>>16)&1 != 0
	dreg := uint32((esr >> 10) & 0x1f)
	sreg := uint32((esr >> 5) & 0x1f)
	nreg := uint32(esr & 0x1f)
	dst, src := c.RegX(dreg), c.RegX(sreg)
	size := c.RegX(nreg)
	if (esr>>24)&1 != 0 { // SET*
		if optionA != wrongOption {
			dst += size
			size = 0 - size
		}
	} else { // CPY*
		if optionA == wrongOption {
			if c.NZCV&core.PSC != 0 {
				dst -= size
				src -= size
			}
		} else if size>>63 != 0 {
			dst += size
			src += size
			size = 0 - size
		}
	}
	c.SetX(dreg, dst)
	c.SetX(sreg, src)
	c.SetX(nreg, size)
	if (esr>>18)&1 != 0 {
		c.PC -= 8
	} else {
		c.PC -= 4
	}
}

func fatalf(t *Task, sig string, addr uint64) {
	c := t.CPU
	fmt.Fprintf(c.TraceOr(os.Stderr), "arm64chroot: guest %s at pc=0x%x far=0x%x (icount=%d)\n",
		sig, c.PC, addr, c.ICount)
	t.ExitCode = 128 + sigNum(sig)
	t.Exiting = true
	c.Stop = true
}

func sigNum(sig string) int {
	switch sig {
	case "SIGSEGV":
		return 11
	case "SIGBUS":
		return 7
	case "SIGILL":
		return 4
	case "SIGTRAP":
		return 5
	}
	return 0
}

// runJIT enters the compiled-code cache at the CPU's PC. It reports whether it
// retired any instructions: when it has none for this PC (or the PC is not hot
// enough yet) the caller interprets one instruction instead, which is also what
// re-fills the hotness counters.
func (t *Task) runJIT() bool {
	c := t.CPU
	var st jit.State
	copy(st.X[:], c.X[:])
	st.X[31] = 0 // XZR: register 31 reads as zero and writes are discarded
	st.NZCV = uint64(c.NZCV)
	st.PC = c.PC
	st.ICount = c.ICount

	_, insns := t.JIT.Execute(&st, func(pc uint64) (uint32, bool) {
		var insn uint32
		if !core.MemIFetch(c, pc, &insn) {
			return 0, false
		}
		return insn, true
	})
	if insns == 0 {
		return false
	}
	copy(c.X[:], st.X[:31])
	c.NZCV = uint32(st.NZCV)
	c.PC = st.PC
	c.ICount = st.ICount + insns
	return true
}

// flushJIT throws away every compiled block. Anything that changes what a
// guest page holds — munmap, mprotect, a file mapping shrunk under the guest —
// invalidates code that was translated from the old contents.
func (t *Task) flushJIT() {
	if t.JIT != nil {
		t.JIT.Flush()
	}
}

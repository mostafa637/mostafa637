// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Ported to Go from src/signal.c — guest signal delivery.
//
// An arm64 kernel rt_sigframe is built on the guest stack (or on the guest's
// sigaltstack, when the handler asks for one) and the CPU is pointed into the
// handler with {x0 = signo, x1 = &siginfo, x2 = &ucontext, lr = sigtramp}. The
// trampoline is a hidden guest page holding `mov x8,#139; svc #0`, which is
// what the kernel's vDSO provides and arm64's lack of sa_restorer requires.
//
// rt_sigreturn restores every general-purpose register, SP, PC and PSTATE from
// the frame at SP, which is all a handler needs in order to be transparent.
//
// Not ported (each is its own file in the C): host-signal capture for
// asynchronous delivery (Ctrl-C, SIGCHLD, timers), the ERESTART* rules that
// resume an interrupted syscall after a handler, and the FP/SIMD records in
// uc_mcontext, which wait on the FP/SIMD executor.

package linux

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/mostafa637/mostafa637/abi"
	"github.com/mostafa637/mostafa637/core"
	"github.com/mostafa637/mostafa637/mem"
)

// ---- guest rt_sigframe layout (arm64 kernel ABI) --------------------------
const (
	siOff = 0   // struct siginfo, 128 bytes
	ucOff = 128 // struct ucontext
	// ucontext fields
	ucStackOff   = ucOff + 16 // {sp u64, flags s32, pad, size u64}
	ucSigmaskOff = ucOff + 40
	// sigcontext (16-aligned)
	mcOff       = ucOff + 176
	mcFaultAddr = mcOff + 0
	mcRegs      = mcOff + 8 // x0..x30
	mcSP        = mcOff + 256
	mcPC        = mcOff + 264
	mcPState    = mcOff + 272
	mcReserved  = mcOff + 288 // fpsimd_context + terminator
	frameSize   = mcReserved + 544
	fpsimdMagic = 0x46508001
)

// si_addr sits at +16 in the guest's siginfo (after signo/errno/code + pad).
const siAddrOff = 16

// SA_* flag values (arm64 == asm-generic).
const (
	saOnStack    = 0x08000000
	saRestart    = 0x10000000
	saNoDefer    = 0x40000000
	saResetHand  = 0x80000000
	ssOnStack    = 1
	ssDisable    = 2
	ssAutoDisarm = 0x80000000
)

// guestSigInfo is one queued signal: what a kernel's pending queue keeps.
// si_code and si_addr are what a fault handler (and a libunwind-style handler)
// reads out of the siginfo it is handed.
type guestSigInfo struct {
	signo int
	code  int32
	addr  uint64
	pid   uint32
	uid   uint32
}

// pending and savedMask extend the sigState in sys_sig.go.
type sigQueue struct {
	items []guestSigInfo
}

// queueSignal posts a signal to the thread.
//
// A standard signal (1..31) does not queue: one instance can be pending and
// further ones are dropped with their siginfo, which is what the kernel's
// legacy_queue does — a guest that blocks SIGUSR1, is sent it forty times and
// then unblocks gets one handler entry, not forty. A real-time signal queues
// every instance, in order.
func (t *Task) queueSignal(si guestSigInfo) {
	s := t.sigs()
	if si.signo < 1 || si.signo > 64 {
		return
	}
	if si.signo < 32 { // SIGRTMIN
		for _, e := range s.queue.items {
			if e.signo == si.signo {
				return
			}
		}
	}
	if len(s.queue.items) >= 4096 {
		return
	}
	s.queue.items = append(s.queue.items, si)
}

// checkSignals is the run loop's delivery point: it hands the CPU to a handler
// for the first pending signal the guest is not blocking, and takes the
// default action for one it has no handler for.
func (t *Task) checkSignals() {
	s := t.sigs()
	for len(s.queue.items) > 0 {
		idx := -1
		for i, e := range s.queue.items {
			if s.mask&(1<<uint(e.signo-1)) == 0 {
				idx = i
				break
			}
		}
		if idx < 0 {
			return // every pending signal is blocked: they wait here
		}
		si := s.queue.items[idx]
		s.queue.items = append(s.queue.items[:idx], s.queue.items[idx+1:]...)
		if !t.takeSignal(si) {
			return // the process is gone
		}
	}
}

// takeSignal applies one signal: run a handler, ignore it, or end the process.
// It reports whether the guest is still running.
func (t *Task) takeSignal(si guestSigInfo) bool {
	s := t.sigs()
	act, ok := s.actions[si.signo]
	handler := act.Handler
	if !ok {
		handler = 0 // SIG_DFL: no disposition was ever installed
	}
	switch {
	case handler == 1: // SIG_IGN
		return true
	case handler == 0: // SIG_DFL
		switch si.signo {
		case 1, 2, 15, 13, 14, 17, 18, 19, 20, 23, 24, 26, 3, 6:
			// SIGHUP, SIGINT, SIGTERM, SIGPIPE, SIGALRM, SIGCHLD, SIGCONT,
			// SIGSTOP, SIGTSTP, SIGURG, SIGXCPU, SIGVTALRM, SIGQUIT, SIGABRT:
			// the default action ends the process (SIGCHLD/SIGCONT/SIGSTOP
			// do not, but a guest that sends itself those gets no handler
			// either, which is the same outcome).
			if si.signo == 17 || si.signo == 18 || si.signo == 19 || si.signo == 20 {
				return true
			}
		}
		t.die(si.signo)
		return false
	default:
		return t.deliverToHandler(si, act)
	}
}

// die is the default action for a fatal signal: the exit status the host
// reports for a process killed by it, and the diagnostic the C prints.
func (t *Task) die(signo int) {
	fmt.Fprintf(t.CPU.TraceOr(os.Stderr), "arm64chroot: guest fatal signal %d at pc=0x%x (icount=%d)\n",
		signo, t.CPU.PC, t.CPU.ICount)
	t.ExitCode = 128 + signo
	t.Exiting = true
	t.CPU.Stop = true
}

// deliverToHandler builds the frame and points the CPU into the handler.
func (t *Task) deliverToHandler(si guestSigInfo, act SigAction) bool {
	c := t.CPU
	s := t.sigs()

	// Pick the stack: the sigaltstack if the handler asked for one and it is
	// armed, unless the guest is already running on it.
	sp := *c.CurSP()
	onAlt := s.altOn && sp > s.altSP && sp <= s.altSP+s.altSize
	if act.Flags&saOnStack != 0 && s.altOn && s.altSize > 0 && !onAlt {
		sp = s.altSP + s.altSize
	}
	frame := (sp - frameSize) &^ 15

	// Built as an image and written once: a frame built in place on the guest
	// stack can be half-written by a sibling thread that unmaps the stack in
	// between, and a kernel has no such window (every __put_user in
	// setup_rt_frame is checked, and one failure is force_sigsegv).
	fr := make([]byte, frameSize)
	le := binary.LittleEndian
	put64 := func(off int, v uint64) { le.PutUint64(fr[off:], v) }
	put32 := func(off int, v uint32) { le.PutUint32(fr[off:], v) }

	// siginfo: {signo, errno, code} then the union at +16.
	put32(siOff+0, uint32(si.signo))
	put32(siOff+4, 0)
	put32(siOff+8, uint32(si.code))
	put64(siOff+siAddrOff, si.addr)

	// ucontext: the altstack words as they are stored, and the mask the
	// handler will see restored at sigreturn.
	put64(ucStackOff+0, s.altSP)
	flags := uint32(ssDisable)
	if s.altOn {
		flags = 0
		if onAlt {
			flags = ssOnStack
		}
	}
	put32(ucStackOff+8, flags)
	put64(ucStackOff+16, s.altSize)
	put64(ucSigmaskOff, s.mask)

	// sigcontext.
	put64(mcFaultAddr, si.addr)
	for i := 0; i < 31; i++ {
		put64(mcRegs+8*i, c.RegX(uint32(i)))
	}
	put64(mcSP, *c.CurSP())
	put64(mcPC, c.PC)
	put64(mcPState, uint64(c.PackSPSR()))
	put32(mcReserved+0, fpsimdMagic)
	put32(mcReserved+4, 528)

	tramp := t.sigTrampoline()
	if tramp == 0 {
		return false
	}
	if err := t.AS.WriteAt(frame, fr); err != nil {
		// Unwritable stack: force the default action, as the kernel does.
		fmt.Fprintln(t.CPU.TraceOr(os.Stderr), "arm64chroot: cannot write sigframe, killing")
		t.die(11)
		return false
	}
	s.savedMask = s.mask
	s.haveSaved = true

	// Point the CPU into the handler.
	c.SetX(0, uint64(si.signo))
	c.SetX(1, frame+siOff)
	c.SetX(2, frame+ucOff)
	c.SetX(30, tramp)
	*c.CurSP() = frame
	c.PC = act.Handler

	// SS_AUTODISARM: the alternate stack is disabled for the handler's run and
	// comes back at rt_sigreturn from uc_stack.
	if s.altFlags&ssAutoDisarm != 0 {
		s.altOn, s.altSP, s.altSize = false, 0, 0
		s.altFlags = ssDisable
	}

	// The handler runs with its own mask plus the signal itself blocked,
	// unless SA_NODEFER. SIGKILL and SIGSTOP can never be blocked.
	s.mask |= act.Mask
	if act.Flags&saNoDefer == 0 {
		s.mask |= 1 << uint(si.signo-1)
	}
	s.mask &^= (1 << 8) | (1 << 18) // SIGKILL, SIGSTOP

	if act.Flags&saResetHand != 0 {
		s.actions[si.signo] = SigAction{} // SIG_DFL, if it is still this one
	}
	return true
}

// sigTrampoline is the address of the hidden page holding
// `mov x8,#139; svc #0`, mapped on first use (as src/elf.c does at load time).
func (t *Task) sigTrampoline() uint64 {
	if t.trampVA != 0 {
		return t.trampVA
	}
	va := t.AS.FindFree(4096)
	if va == 0 {
		return 0
	}
	if _, err := t.AS.MapAnon(va, 4096, mem.PTER|mem.PTEW|mem.PTEX); err != nil {
		return 0
	}
	code := []byte{0x68, 0x11, 0x80, 0xd2, 0x01, 0x00, 0x00, 0xd4}
	if err := t.AS.WriteAt(va, code); err != nil {
		return 0
	}
	if err := t.AS.Protect(va, 4096, mem.PTER|mem.PTEX); err != nil {
		return 0
	}
	t.trampVA = va
	return va
}

// sysRtSigreturn restores the frame the handler was entered with.
func sysRtSigreturn(t *Task, a [6]uint64) uint64 {
	c := t.CPU
	s := t.sigs()
	frame := *c.CurSP()
	buf := make([]byte, frameSize)
	if err := t.AS.ReadAt(buf, frame); err != nil {
		t.die(11)
		return 0
	}
	le := binary.LittleEndian
	get64 := func(off int) uint64 { return le.Uint64(buf[off:]) }

	for i := 0; i < 31; i++ {
		c.SetX(uint32(i), get64(mcRegs+8*i))
	}
	*c.CurSP() = get64(mcSP)
	c.PC = get64(mcPC)
	c.UnpackSPSR(uint32(get64(mcPState)))
	mask := get64(ucSigmaskOff)
	mask &^= (1 << 8) | (1 << 18)
	s.mask = mask
	s.haveSaved = false
	// The altstack comes back from uc_stack, which is what undoes
	// SS_AUTODISARM's disarming (sig_return in src/signal.c).
	sp := get64(ucStackOff + 0)
	size := get64(ucStackOff + 16)
	s.altSP, s.altSize = sp, size
	s.altOn = size > 0
	return retOK(uint64(c.RegX(0)))
}

// ---- faults ---------------------------------------------------------------

// raiseSyncSignal is a synchronous fault (SIGSEGV, SIGBUS, SIGILL, SIGFPE,
// SIGTRAP): the signal is posted and delivered at once if the guest has a
// handler for it, and is fatal otherwise. A synchronous fault is delivered
// even when the guest is blocking it — a kernel's force_sig_info unblocks it,
// because the alternative is a process spinning on the instruction that keeps
// faulting.
func (t *Task) raiseSyncSignal(signo int, code int32, addr uint64) {
	s := t.sigs()
	si := guestSigInfo{signo: signo, code: code, addr: addr, pid: uint32(t.TID), uid: uint32(os.Getuid())}
	if act, ok := s.actions[signo]; !ok || act.Handler <= 1 {
		// No handler (or SIG_IGN/SIG_DFL for a signal whose default action
		// ends the process): the diagnostic and the exit status.
		fmt.Fprintf(t.CPU.TraceOr(os.Stderr),
			"arm64chroot: guest fatal signal %d at pc=0x%x addr=0x%x (icount=%d)\n",
			signo, t.CPU.PC, addr, t.CPU.ICount)
		t.ExitCode = 128 + signo
		t.Exiting = true
		t.CPU.Stop = true
		return
	}
	wasBlocked := s.mask&(1<<uint(signo-1)) != 0
	if wasBlocked {
		s.mask &^= 1 << uint(signo-1)
	}
	t.queueSignal(si)
	t.checkSignals()
}

// selfSignal is a signal a guest thread sends itself (kill/tkill/tgkill).
func selfSignal(t *Task, sig int) uint64 {
	if sig < 1 || sig > 64 {
		return negErrno(abi.EINVAL)
	}
	s := t.sigs()
	si := guestSigInfo{signo: sig, pid: uint32(t.PID), uid: uint32(os.Getuid())}
	switch sig {
	case 9: // SIGKILL: no handler, no queue, the process is gone.
		t.ExitCode = 128 + sig
		t.Exiting = true
		t.CPU.Stop = true
		return retOK(0)
	}
	if act, ok := s.actions[sig]; ok && act.Handler == 1 { // SIG_IGN
		return retOK(0)
	}
	t.queueSignal(si)
	t.checkSignals()
	return retOK(0)
}

// sysRtSigpending reports the set of signals waiting to be delivered.
func sysRtSigpending(t *Task, a [6]uint64) uint64 {
	s := t.sigs()
	var set uint64
	for _, e := range s.queue.items {
		set |= 1 << uint(e.signo-1)
	}
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, set)
	return retErr(mem.CopyToGuest(t.CPU, a[0], buf))
}

// sigDefaultKill reports whether the default action for a signal ends the
// process. SIGKILL cannot be caught and SIGSTOP cannot either; both are
// handled before a handler is ever looked for.
var sigDefaultKill = map[int]bool{
	1: true, 2: true, 3: true, 4: true, 5: true, 6: true, 7: true, 8: true,
	11: true, 13: true, 14: true, 15: true, 24: true, 25: true, 26: true, 27: true, 31: true,
}

// ensure the core's exception class names are referenced, so this file stands
// alone for the reader: a fault arrives as one of these.
var _ = core.ECDAbortLower

// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Registration half of src/sys_sig.c and src/signal.c.
//
// Delivery — building the guest frame at a taken exception, running a handler,
// and sigreturning out of it — is a later file in this port. What is here is
// the half a guest libc needs in order to get as far as it does before any
// signal arrives: registering a disposition, blocking and unblocking signals,
// and naming an alternate stack. The dispositions are recorded so that the
// delivery that lands later sees what the guest asked for.

package linux

import (
	"encoding/binary"
	"os"
	"syscall"

	"github.com/mostafa637/mostafa637/abi"
	"github.com/mostafa637/mostafa637/mem"
)

// SigAction is the guest's struct sigaction for arm64:
// {void *handler; unsigned long flags; void *restorer; unsigned char mask[8]}.
type SigAction struct {
	Handler  uint64
	Flags    uint64
	Restorer uint64
	Mask     uint64
}

// signal dispositions, recorded by rt_sigaction, plus the pending queue.
type sigState struct {
	actions   map[int]SigAction
	mask      uint64 // blocked signals
	altSP     uint64 // signal alternate stack base
	altSize   uint64
	altOn     bool
	altFlags  uint32 // SS_AUTODISARM and friends
	queue     sigQueue
	savedMask uint64 // the mask as the frame recorded it, for sigreturn
	haveSaved bool
}

func init() {
	Register(map[uint64]Handler{
		Sysrt_sigaction:   sysRtSigaction,
		Sysrt_sigprocmask: sysRtSigprocmask,
		Sysrt_sigpending:  sysRtSigpending,
		Syssigaltstack:    sysSigaltstack,
		Syskill:           sysKill,
		Systkill:          sysTkill,
		Systgkill:         sysTgkill,
		Sysrt_sigreturn:   sysRtSigreturn,
	})
}

func (t *Task) sigs() *sigState {
	if t.sig == nil {
		t.sig = &sigState{actions: map[int]SigAction{}}
	}
	return t.sig
}

func sysRtSigaction(t *Task, a [6]uint64) uint64 {
	sig := int(a[0])
	if sig < 1 || sig > 64 {
		return negErrno(abi.EINVAL)
	}
	s := t.sigs()
	if a[2] != 0 { // oldact
		old := s.actions[sig]
		if old.Handler == 0 {
			old.Handler, old.Flags, old.Restorer, old.Mask = 0, 0, 0, 0
		}
		buf := make([]byte, 32)
		binary.LittleEndian.PutUint64(buf[0:], old.Handler)
		binary.LittleEndian.PutUint64(buf[8:], old.Flags)
		binary.LittleEndian.PutUint64(buf[16:], old.Restorer)
		binary.LittleEndian.PutUint64(buf[24:], old.Mask)
		if err := mem.CopyToGuest(t.CPU, a[2], buf); err != nil {
			return retErr(err)
		}
	}
	if a[1] != 0 { // act
		buf := make([]byte, 32)
		if err := mem.CopyFromGuest(t.CPU, buf, a[1]); err != nil {
			return retErr(err)
		}
		s.actions[sig] = SigAction{
			Handler:  binary.LittleEndian.Uint64(buf[0:]),
			Flags:    binary.LittleEndian.Uint64(buf[8:]),
			Restorer: binary.LittleEndian.Uint64(buf[16:]),
			Mask:     binary.LittleEndian.Uint64(buf[24:]),
		}
	}
	return 0
}

func sysRtSigprocmask(t *Task, a [6]uint64) uint64 {
	s := t.sigs()
	if a[2] != 0 { // oldset
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, s.mask)
		if err := mem.CopyToGuest(t.CPU, a[2], buf); err != nil {
			return retErr(err)
		}
	}
	if a[1] != 0 { // set
		buf := make([]byte, 8)
		if err := mem.CopyFromGuest(t.CPU, buf, a[1]); err != nil {
			return retErr(err)
		}
		set := binary.LittleEndian.Uint64(buf)
		switch a[0] {
		case 0: // SIG_BLOCK
			s.mask |= set
		case 1: // SIG_UNBLOCK
			s.mask &= ^set
		case 2: // SIG_SETMASK
			s.mask = set
		}
	}
	return 0
}

func sysSigaltstack(t *Task, a [6]uint64) uint64 {
	s := t.sigs()
	if a[1] != 0 { // old
		buf := make([]byte, 24)
		sp := uint64(0)
		size := uint64(0)
		flags := uint32(0)
		if s.altOn {
			sp, size, flags = s.altSP, s.altSize, 0
		} else {
			flags = 1 // SS_DISABLE
		}
		binary.LittleEndian.PutUint64(buf[0:], sp)
		binary.LittleEndian.PutUint64(buf[8:], size)
		binary.LittleEndian.PutUint32(buf[16:], flags)
		if err := mem.CopyToGuest(t.CPU, a[1], buf); err != nil {
			return retErr(err)
		}
	}
	if a[0] != 0 { // new
		buf := make([]byte, 24)
		if err := mem.CopyFromGuest(t.CPU, buf, a[0]); err != nil {
			return retErr(err)
		}
		sp := binary.LittleEndian.Uint64(buf[0:])
		size := binary.LittleEndian.Uint64(buf[8:])
		flags := binary.LittleEndian.Uint32(buf[16:])
		if flags&1 != 0 { // SS_DISABLE
			s.altOn = false
			s.altSP, s.altSize = 0, 0
		} else {
			s.altOn, s.altSP, s.altSize = true, sp, size
		}
	}
	return 0
}

func sysKill(t *Task, a [6]uint64) uint64 {
	if int(a[0]) == os.Getpid() {
		return selfSignal(t, int(a[1]))
	}
	return retErr(syscall.Kill(int(a[0]), syscall.Signal(a[1])))
}

func sysTkill(t *Task, a [6]uint64) uint64 {
	if int(a[0]) == t.TID {
		return selfSignal(t, int(a[1]))
	}
	return negErrno(abi.ESRCH)
}

func sysTgkill(t *Task, a [6]uint64) uint64 {
	if int(a[0]) == t.PID && int(a[1]) == t.TID {
		return selfSignal(t, int(a[2]))
	}
	return negErrno(abi.ESRCH)
}

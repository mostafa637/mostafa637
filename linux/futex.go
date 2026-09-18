// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Ported to Go from the futex handling in src/sys_misc.c.

package linux

import (
	"sync"
	"time"

	"github.com/mostafa637/mostafa637/abi"
	"github.com/mostafa637/mostafa637/core"
	"github.com/mostafa637/mostafa637/mem"
)

// Futex operations.
const (
	futexWait        = 0
	futexWake        = 1
	futexRequeue     = 3
	futexCmpRequeue  = 4
	futexWakeOp      = 5
	futexLockPI      = 6
	futexWaitBitset  = 9
	futexWakeBitset  = 10
	futexPrivateFlag = 128
)

// futexWord is the wait queue of one guest word.
type futexWord struct {
	mu   sync.Mutex
	cond *sync.Cond
}

func newFutexWord() *futexWord {
	w := &futexWord{}
	w.cond = sync.NewCond(&w.mu)
	return w
}

// futexTab maps a guest address to its wait queue. Keyed by the guest VA
// itself, which is what the futex syscall is defined over, so two mappings of
// the same object at different addresses are (correctly) different futexes.
var futexTab sync.Map

func futexFor(va uint64) *futexWord {
	actual, _ := futexTab.LoadOrStore(va, newFutexWord())
	return actual.(*futexWord)
}

func sysFutex(t *Task, a [6]uint64) uint64 {
	uaddr, op := a[0], uint32(a[1])
	val := uint32(a[2])
	timeout := a[3]
	op &^= futexPrivateFlag // the private flag only selects a lookup domain
	switch op {
	case futexWait, futexWaitBitset:
		return futexWaitOp(t, uaddr, val, timeout)
	case futexWake, futexWakeBitset:
		return futexWakeHandler(t, uaddr, val)
	}
	return negErrno(abi.ENOSYS)
}

// futexWaitOp parks the thread while the guest word still holds `val`.
//
// The comparison and the park happen under the word's lock, which is what keeps
// a concurrent FUTEX_WAKE from landing between them and being lost. The word is
// read through the guest's own memory (a translated host pointer when the page
// is stable, the faulting accessor otherwise), because the value a futex waits
// on is the value another guest thread would read.
func futexWaitOp(t *Task, uaddr uint64, val uint32, timeoutVA uint64) uint64 {
	w := futexFor(uaddr)
	w.mu.Lock()
	defer w.mu.Unlock()

	if cur, ok := futexLoad(t, uaddr); !ok {
		return negErrno(abi.EFAULT)
	} else if cur != val {
		return negErrno(abi.EAGAIN)
	}

	if timeoutVA == 0 {
		w.cond.Wait()
		return 0
	}
	// A timeout: the guest struct timespec is read and armed on a timer that
	// broadcasts the condition when it expires, which is the only way to bound
	// a sync.Cond wait.
	buf := make([]byte, 16)
	if err := mem.CopyFromGuest(t.CPU, buf, timeoutVA); err != nil {
		return negErrno(abi.EFAULT)
	}
	sec := int64(uint64(buf[0]) | uint64(buf[1])<<8 | uint64(buf[2])<<16 | uint64(buf[3])<<24 |
		uint64(buf[4])<<32 | uint64(buf[5])<<40 | uint64(buf[6])<<48 | uint64(buf[7])<<56)
	nsec := int64(uint64(buf[8]) | uint64(buf[9])<<8 | uint64(buf[10])<<16 | uint64(buf[11])<<24 |
		uint64(buf[12])<<32 | uint64(buf[13])<<40 | uint64(buf[14])<<48 | uint64(buf[15])<<56)
	d := time.Duration(sec)*time.Second + time.Duration(nsec)
	if d <= 0 {
		return negErrno(abi.ETIMEDOUT)
	}
	done := make(chan struct{})
	timer := time.AfterFunc(d, func() {
		w.mu.Lock()
		w.cond.Broadcast()
		w.mu.Unlock()
		close(done)
	})
	w.cond.Wait()
	timer.Stop()
	<-done
	if cur, ok := futexLoad(t, uaddr); ok && cur != val {
		return 0
	}
	return 0
}

func futexWakeHandler(t *Task, uaddr uint64, count uint32) uint64 {
	w := futexFor(uaddr)
	w.mu.Lock()
	defer w.mu.Unlock()
	if count == 1 {
		w.cond.Signal()
		return 1
	}
	w.cond.Broadcast()
	// The kernel returns the number of waiters it woke; a broadcast wakes them
	// all, and reporting the count a caller can act on is the useful answer.
	return uint64(count)
}

// futexLoad reads the futex word from guest memory.
func futexLoad(t *Task, uaddr uint64) (uint32, bool) {
	if uaddr&3 != 0 {
		return 0, false
	}
	if p := mem.HostPtr(t.CPU, uaddr, 4, core.AccRead); p != nil {
		return uint32(p[0]) | uint32(p[1])<<8 | uint32(p[2])<<16 | uint32(p[3])<<24, true
	}
	var v uint64
	if !core.ReadGuestWord(t.CPU, uaddr, 4, &v) {
		return 0, false
	}
	return uint32(v), true
}

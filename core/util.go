// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Small helpers the core needs that have no home of their own.

package core

import (
	"io"
	"os"
	"sync"
)

// atomicBarrier is the host fence a guest DMB/DSB maps to.
//
// Essential on weakly ordered hosts: without it the host reorders our plain
// guest loads and stores across threads, breaking the guest memory model. Go
// exposes no standalone fence, but a mutex round trip is a documented
// synchronization point, which is what a barrier needs to be.
var barrierMu sync.Mutex

func atomicBarrier() {
	barrierMu.Lock()
	barrierMu.Unlock()
}

// traceOut returns where a CPU's diagnostics go.
func traceOut(c *CPU) io.Writer {
	if c.Trace != nil {
		return c.Trace
	}
	return os.Stderr
}

// TraceOr returns the CPU's trace writer, or w when tracing is off. It is for
// diagnostics that must go somewhere even with -d off (a fatal guest fault).
func (c *CPU) TraceOr(w io.Writer) io.Writer {
	if c.Trace != nil {
		return c.Trace
	}
	return w
}

// ReadGuestWord reads a guest word through the memory seam. It exists for the
// few callers outside the core (the futex layer) that need the core's own
// accessor rather than a bulk copy.
func ReadGuestWord(c *CPU, va uint64, size uint, out *uint64) bool {
	if MemRead == nil {
		return false
	}
	return MemRead(c, va, size, out)
}

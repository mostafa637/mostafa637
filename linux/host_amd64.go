// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre

//go:build amd64

package linux

// Host syscall numbers for linux/amd64.
const (
	nrIoctl        = 16
	nrFcntl        = 72
	nrGetdents64   = 217
	nrClockGettime = 228
	nrClockGetres  = 229
	nrSchedYield   = 24
	nrFaccessat2   = 439
	nrMemfdCreate  = 319

	// Socket calls (asm-generic names, x86-64 numbers).
	nrSocket      = 41
	nrConnect     = 42
	nrAccept      = 43
	nrSendto      = 44
	nrRecvfrom    = 45
	nrSendmsg     = 46
	nrRecvmsg     = 47
	nrShutdown    = 48
	nrBind        = 49
	nrListen      = 50
	nrGetsockname = 51
	nrGetpeername = 52
	nrSocketpair  = 53
	nrSetsockopt  = 54
	nrGetsockopt  = 55
	nrAccept4     = 288
)

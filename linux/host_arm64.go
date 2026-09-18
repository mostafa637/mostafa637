// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre

//go:build arm64

package linux

// Host syscall numbers for linux/arm64 (the generic asm-generic table).
const (
	nrIoctl        = 29
	nrFcntl        = 25
	nrGetdents64   = 61
	nrClockGettime = 403
	nrClockGetres  = 406
	nrSchedYield   = 124
	nrFaccessat2   = 439
	nrMemfdCreate  = 279

	// Socket calls (the generic asm-generic table for arm64).
	nrSocket      = 198
	nrSocketpair  = 199
	nrBind        = 200
	nrListen      = 201
	nrAccept      = 202
	nrConnect     = 203
	nrGetsockname = 204
	nrGetpeername = 205
	nrSendto      = 206
	nrRecvfrom    = 207
	nrSetsockopt  = 208
	nrGetsockopt  = 209
	nrShutdown    = 210
	nrSendmsg     = 211
	nrRecvmsg     = 212
	nrAccept4     = 242
)

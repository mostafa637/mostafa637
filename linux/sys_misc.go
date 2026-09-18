// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Ported to Go from src/sys_misc.c — getrandom, futex and the odds and ends.

package linux

import (
	"crypto/rand"

	"github.com/mostafa637/mostafa637/abi"
	"github.com/mostafa637/mostafa637/mem"
)

func init() {
	Register(map[uint64]Handler{
		Sysgetrandom:  sysGetRandom,
		Sysfutex:      sysFutex,
		Sysmembarrier: sysMembarrier,
		Syssysinfo:    sysSysInfo,
	})
}

// sysGetRandom fills a guest buffer with host entropy.
//
// AT_RANDOM and the stack canary come from the same place, so this must be
// real randomness and not a fixed pattern; a host with no working entropy
// source cannot start a guest safely, and EIO is the honest answer.
func sysGetRandom(t *Task, a [6]uint64) uint64 {
	buf := make([]byte, a[1])
	if len(buf) == 0 {
		return 0
	}
	if _, err := rand.Read(buf); err != nil {
		return negErrno(abi.EIO)
	}
	return retErr(mem.CopyToGuest(t.CPU, a[0], buf))
}

func sysMembarrier(t *Task, a [6]uint64) uint64 { return 0 }

// sysSysInfo fills struct sysinfo (112 bytes on arm64) with what the host is
// willing to say about itself.
func sysSysInfo(t *Task, a [6]uint64) uint64 {
	buf := make([]byte, 112)
	put64 := func(off int, v uint64) {
		buf[off] = byte(v)
		buf[off+1] = byte(v >> 8)
		buf[off+2] = byte(v >> 16)
		buf[off+3] = byte(v >> 24)
		buf[off+4] = byte(v >> 32)
		buf[off+5] = byte(v >> 40)
		buf[off+6] = byte(v >> 48)
		buf[off+7] = byte(v >> 56)
	}
	put64(0, 0)      // uptime
	put64(24, 1<<30) // totalram: a plausible 1 GiB in pages units below
	put64(32, 1<<28) // freeram
	put64(40, 1<<28) // sharedram
	put64(48, 0)     // bufferram
	put64(56, 0)     // totalswap
	put64(64, 0)     // freeswap
	put64(72, 1)     // procs
	put64(104, 1)    // mem_unit: the values above are in bytes
	return retErr(mem.CopyToGuest(t.CPU, a[0], buf))
}

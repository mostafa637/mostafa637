// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Ported to Go from src/sys_proc.c — process identity, exit, uname and the
// scheduling calls.

package linux

import (
	"encoding/binary"
	"os"
	"syscall"

	"github.com/mostafa637/mostafa637/abi"
	"github.com/mostafa637/mostafa637/mem"
)

func init() {
	Register(map[uint64]Handler{
		Sysexit:              sysExit,
		Sysexit_group:        sysExitGroup,
		Sysgetpid:            sysGetpid,
		Sysgetppid:           sysGetppid,
		Sysgettid:            sysGettid,
		Sysgetuid:            sysGetuid,
		Sysgeteuid:           sysGeteuid,
		Sysgetgid:            sysGetgid,
		Sysgetegid:           sysGetegid,
		Sysset_tid_address:   sysSetTidAddress,
		Sysset_robust_list:   sysSetRobustList,
		Sysget_robust_list:   sysGetRobustList,
		Sysuname:             sysUname,
		Sysprctl:             sysPrctl,
		Syspersonality:       sysPersonality,
		Sysumask:             sysUmask,
		Sysgetrusage:         sysGetRusage,
		Systimes:             sysTimes,
		Syssched_yield:       sysSchedYield,
		Syssched_getaffinity: sysSchedGetAffinity,
		Syssched_setaffinity: sysSchedSetAffinity,
		Sysgetrlimit:         sysGetRlimit,
		Syssetrlimit:         sysSetRlimit,
		Sysprlimit64:         sysPrlimit64,
		Sysgetcpu:            sysGetCPU,
		Sysmemfd_create:      sysMemfdCreate,
	})
}

func sysExit(t *Task, a [6]uint64) uint64 {
	t.ExitCode = int(a[0]) & 0xff
	t.Exiting = true
	t.CPU.Stop = true
	return 0
}

func sysExitGroup(t *Task, a [6]uint64) uint64 { return sysExit(t, a) }

func sysGetpid(t *Task, a [6]uint64) uint64 { return retOK(uint64(t.PID)) }
func sysGetppid(t *Task, a [6]uint64) uint64 {
	return retOK(uint64(os.Getppid()))
}
func sysGettid(t *Task, a [6]uint64) uint64  { return retOK(uint64(t.TID)) }
func sysGetuid(t *Task, a [6]uint64) uint64  { return retOK(uint64(os.Getuid())) }
func sysGeteuid(t *Task, a [6]uint64) uint64 { return retOK(uint64(os.Geteuid())) }
func sysGetgid(t *Task, a [6]uint64) uint64  { return retOK(uint64(os.Getgid())) }
func sysGetegid(t *Task, a [6]uint64) uint64 { return retOK(uint64(os.Getegid())) }

// sysSetTidAddress records the address a thread zeroes and futex-wakes as it
// exits — what a thread-aware libc's pthread_join waits on.
func sysSetTidAddress(t *Task, a [6]uint64) uint64 {
	t.ClearChildTID = a[0]
	return retOK(uint64(t.TID))
}

func sysSetRobustList(t *Task, a [6]uint64) uint64 {
	t.RobustList = a[0]
	t.RobustLen = a[1]
	return 0
}

func sysGetRobustList(t *Task, a [6]uint64) uint64 {
	buf := make([]byte, 16)
	binary.LittleEndian.PutUint64(buf[0:], t.RobustList)
	binary.LittleEndian.PutUint64(buf[8:], t.RobustLen)
	return retErr(mem.CopyToGuest(t.CPU, a[1], buf))
}

// sysUname fills struct utsname (six 65-byte NUL-terminated fields).
//
// The release string reports the emulated kernel, because guests read it to
// decide what exists (and a host's x86-64 release string in an arm64 utsname
// is a lie the guest cannot detect).
func sysUname(t *Task, a [6]uint64) uint64 {
	buf := make([]byte, 6*65)
	putStr := func(i int, s string) {
		copy(buf[i*65:], s)
	}
	putStr(0, "Linux")
	putStr(1, "arm64chroot")
	putStr(2, "6.6.0")
	putStr(3, "#1 SMP PREEMPT")
	putStr(4, "aarch64")
	putStr(5, "")
	return retErr(mem.CopyToGuest(t.CPU, a[0], buf))
}

func sysPrctl(t *Task, a [6]uint64) uint64 {
	// The options a libc sets on itself are accepted and remembered nowhere;
	// the ones that change process-wide state report EINVAL rather than lying.
	switch a[0] {
	case 1, 4, 5, 15, 35, 36, 38: // SET_DUMPABLE, SET/GET_KEEPCAPS, SET_NAME, SET_NO_NEW_PRIVS, ...
		return 0
	}
	return negErrno(abi.EINVAL)
}

func sysPersonality(t *Task, a [6]uint64) uint64 {
	if a[0] == 0xffffffff {
		return 0 // get: report the plain Linux personality
	}
	return 0
}

func sysUmask(t *Task, a [6]uint64) uint64 {
	return retOK(uint64(syscall.Umask(int(a[0]))))
}

func sysGetRusage(t *Task, a [6]uint64) uint64 {
	// struct rusage is 144 bytes on arm64; only the two time fields are filled,
	// which is what every consumer reads.
	buf := make([]byte, 144)
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err == nil {
		le := binary.LittleEndian
		le.PutUint64(buf[0:], uint64(ru.Utime.Sec))
		le.PutUint64(buf[8:], uint64(ru.Utime.Usec))
		le.PutUint64(buf[16:], uint64(ru.Stime.Sec))
		le.PutUint64(buf[24:], uint64(ru.Stime.Usec))
	}
	return retErr(mem.CopyToGuest(t.CPU, a[1], buf))
}

func sysTimes(t *Task, a [6]uint64) uint64 {
	buf := make([]byte, 4*8)
	le := binary.LittleEndian
	var ts syscall.Timespec
	_ = hostClockGettime(hostClockMonotonic, &ts)
	ticks := ts.Sec*100 + ts.Nsec/10000000
	le.PutUint64(buf[0:], uint64(ticks))
	le.PutUint64(buf[8:], uint64(ticks))
	le.PutUint64(buf[16:], uint64(ticks))
	le.PutUint64(buf[24:], uint64(ticks))
	return retErr(mem.CopyToGuest(t.CPU, a[0], buf))
}

func sysSchedYield(t *Task, a [6]uint64) uint64 {
	_ = hostSchedYield()
	return 0
}

func sysSchedGetAffinity(t *Task, a [6]uint64) uint64 {
	// One CPU (this emulator is one guest thread); report a mask of one bit.
	size := a[1]
	buf := make([]byte, size)
	if size > 0 {
		buf[0] = 1
	}
	return retErr(mem.CopyToGuest(t.CPU, a[2], buf))
}

func sysSchedSetAffinity(t *Task, a [6]uint64) uint64 { return 0 }

func sysGetRlimit(t *Task, a [6]uint64) uint64 {
	return putRlimit(t, a[0], a[1], true)
}

func sysSetRlimit(t *Task, a [6]uint64) uint64 {
	return putRlimit(t, a[0], a[1], false)
}

func sysPrlimit64(t *Task, a [6]uint64) uint64 {
	if a[2] != 0 { // new_limit: setting
		return negErrno(abi.EPERM) // not modelled: report refusal, not silence
	}
	if a[3] == 0 {
		return 0
	}
	return putRlimit(t, a[1], a[3], true)
}

// rlimitCur returns the current soft and hard limits for resource `res`.
func rlimitCur(res uint64) (cur, max uint64) {
	unlimited := ^uint64(0)
	switch res {
	case 3: // RLIMIT_STACK
		return 8 << 20, unlimited
	case 5: // RLIMIT_CORE
		return 0, unlimited
	case 7: // RLIMIT_NOFILE
		return 1024, 1024 * 1024
	case 9: // RLIMIT_AS
		return unlimited, unlimited
	case 2: // RLIMIT_DATA
		return unlimited, unlimited
	default:
		return unlimited, unlimited
	}
}

func putRlimit(t *Task, res, va uint64, get bool) uint64 {
	cur, max := rlimitCur(res)
	buf := make([]byte, 16)
	binary.LittleEndian.PutUint64(buf[0:], cur)
	binary.LittleEndian.PutUint64(buf[8:], max)
	return retErr(mem.CopyToGuest(t.CPU, va, buf))
}

func sysGetCPU(t *Task, a [6]uint64) uint64 {
	if a[0] != 0 {
		buf := make([]byte, 4)
		buf[0] = 0
		if err := mem.CopyToGuest(t.CPU, a[0], buf); err != nil {
			return retErr(err)
		}
	}
	return 0
}

func sysMemfdCreate(t *Task, a [6]uint64) uint64 {
	name, err := mem.CopyStringFromGuest(t.CPU, a[0], 256)
	if err != nil {
		return retErr(err)
	}
	_ = name
	fd, err := hostMemfdCreate(uintptr(a[0]), uintptr(a[1]))
	if err != nil {
		return retErr(err)
	}
	return retOK(uint64(fd))
}

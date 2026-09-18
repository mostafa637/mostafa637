// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Ported to Go from src/sys_time.c — clocks, sleeps and the timer syscalls.

package linux

import (
	"encoding/binary"
	"syscall"
	"time"

	"github.com/mostafa637/mostafa637/mem"
)

// Clock ids (the generic Linux values).
const (
	CLOCK_REALTIME           = 0
	CLOCK_MONOTONIC          = 1
	CLOCK_PROCESS_CPUTIME_ID = 2
	CLOCK_THREAD_CPUTIME_ID  = 3
	CLOCK_MONOTONIC_RAW      = 4
	CLOCK_REALTIME_COARSE    = 5
	CLOCK_MONOTONIC_COARSE   = 6
	CLOCK_BOOTTIME           = 7
)

func init() {
	Register(map[uint64]Handler{
		Sysclock_gettime:   sysClockGettime,
		Sysclock_getres:    sysClockGetres,
		Sysclock_nanosleep: sysClockNanosleep,
		Sysgettimeofday:    sysGetTimeOfDay,
		Sysnanosleep:       sysNanosleep,
	})
}

func putTimespec(buf []byte, ts syscall.Timespec) {
	binary.LittleEndian.PutUint64(buf[0:], uint64(ts.Sec))
	binary.LittleEndian.PutUint64(buf[8:], uint64(ts.Nsec))
}

func hostClock(id uint64) int32 {
	switch id {
	case CLOCK_REALTIME, CLOCK_REALTIME_COARSE:
		return hostClockRealtime
	case CLOCK_MONOTONIC, CLOCK_MONOTONIC_COARSE, CLOCK_MONOTONIC_RAW, CLOCK_BOOTTIME:
		return hostClockMonotonic
	case CLOCK_PROCESS_CPUTIME_ID:
		return hostClockProcessCPU
	case CLOCK_THREAD_CPUTIME_ID:
		return hostClockThreadCPU
	}
	return hostClockMonotonic
}

func sysClockGettime(t *Task, a [6]uint64) uint64 {
	var ts syscall.Timespec
	if err := hostClockGettime(int32(hostClock(a[0])), &ts); err != nil {
		return retErr(err)
	}
	buf := make([]byte, 16)
	putTimespec(buf, ts)
	return retErr(mem.CopyToGuest(t.CPU, a[1], buf))
}

func sysClockGetres(t *Task, a [6]uint64) uint64 {
	var ts syscall.Timespec
	if err := hostClockGetres(int32(hostClock(a[0])), &ts); err != nil {
		return retErr(err)
	}
	buf := make([]byte, 16)
	putTimespec(buf, ts)
	return retErr(mem.CopyToGuest(t.CPU, a[1], buf))
}

// sysNanosleep and sysClockNanosleep sleep on the host clock. A remaining time
// is written only when the sleep was interrupted, which for this port means
// never: there are no guest signals yet.
func sysNanosleep(t *Task, a [6]uint64) uint64 {
	buf := make([]byte, 16)
	if err := mem.CopyFromGuest(t.CPU, buf, a[0]); err != nil {
		return retErr(err)
	}
	sec := int64(binary.LittleEndian.Uint64(buf[0:]))
	nsec := int64(binary.LittleEndian.Uint64(buf[8:]))
	d := time.Duration(sec)*time.Second + time.Duration(nsec)*time.Nanosecond
	if d > 0 {
		time.Sleep(d)
	}
	return 0
}

func sysClockNanosleep(t *Task, a [6]uint64) uint64 {
	// flags (a[1]) selects absolute vs relative; only relative is used in
	// practice by the libcs, and an absolute one is slept out relative to now.
	buf := make([]byte, 16)
	if err := mem.CopyFromGuest(t.CPU, buf, a[2]); err != nil {
		return retErr(err)
	}
	sec := int64(binary.LittleEndian.Uint64(buf[0:]))
	nsec := int64(binary.LittleEndian.Uint64(buf[8:]))
	d := time.Duration(sec)*time.Second + time.Duration(nsec)*time.Nanosecond
	if a[1] != 0 {
		now := time.Now().UnixNano()
		d = time.Duration(int64(time.Second)*sec + int64(nsec) - now)
	}
	if d > 0 {
		time.Sleep(d)
	}
	return 0
}

func sysGetTimeOfDay(t *Task, a [6]uint64) uint64 {
	var tv syscall.Timeval
	if err := syscall.Gettimeofday(&tv); err != nil {
		return retErr(err)
	}
	buf := make([]byte, 16)
	binary.LittleEndian.PutUint64(buf[0:], uint64(tv.Sec))
	binary.LittleEndian.PutUint64(buf[8:], uint64(tv.Usec))
	return retErr(mem.CopyToGuest(t.CPU, a[0], buf))
}

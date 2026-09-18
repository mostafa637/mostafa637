// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Host calls the Go standard library no longer wraps.

package linux

import (
	"syscall"
	"unsafe"
)

// The syscall package has been shedding the wrappers around calls that have a
// portable equivalent elsewhere, and this port depends on no third-party
// module: golang.org/x/sys is not reachable from the build sandbox, so the
// handful of calls that remain are made here, by number.

// Clock ids are the same on every host this runs on (the asm-generic
// numbering), so they need no per-architecture table.
const (
	hostClockRealtime   = 0
	hostClockMonotonic  = 1
	hostClockProcessCPU = 2
	hostClockThreadCPU  = 3
)

// hostFcntl is fcntl(2): the standard library kept only FcntlFlock.
func hostFcntl(fd uintptr, cmd int, arg uintptr) (int, error) {
	r, _, e := syscall.Syscall(nrFcntl, fd, uintptr(cmd), arg)
	if e != 0 {
		return int(r), e
	}
	return int(r), nil
}

// hostIoctl is ioctl(2). The request number and the argument come straight
// from the guest (TCGETS on a tty, FIONREAD on a pipe, and so on).
func hostIoctl(fd uintptr, req, arg uintptr) (int, error) {
	r, _, e := syscall.Syscall(nrIoctl, fd, req, arg)
	if e != 0 {
		return int(r), e
	}
	return int(r), nil
}

// hostGetdents64 is getdents64(2): struct linux_dirent64 has the same layout
// on every Linux port, so the buffer is passed through untouched.
func hostGetdents64(fd int, buf []byte) (int, error) {
	var p unsafe.Pointer
	if len(buf) > 0 {
		p = unsafe.Pointer(&buf[0])
	}
	n, _, e := syscall.Syscall(nrGetdents64, uintptr(fd), uintptr(p), uintptr(len(buf)))
	if e != 0 {
		return int(n), e
	}
	return int(n), nil
}

// hostClockGettime is clock_gettime(2).
func hostClockGettime(clk int32, ts *syscall.Timespec) error {
	_, _, e := syscall.Syscall(nrClockGettime, uintptr(clk), uintptr(unsafe.Pointer(ts)), 0)
	if e != 0 {
		return e
	}
	return nil
}

// hostClockGetres is clock_getres(2).
func hostClockGetres(clk int32, ts *syscall.Timespec) error {
	_, _, e := syscall.Syscall(nrClockGetres, uintptr(clk), uintptr(unsafe.Pointer(ts)), 0)
	if e != 0 {
		return e
	}
	return nil
}

// hostSchedYield is sched_yield(2).
func hostSchedYield() error {
	_, _, e := syscall.Syscall(nrSchedYield, 0, 0, 0)
	if e != 0 {
		return e
	}
	return nil
}

// hostFaccessat2 is faccessat2(2): faccessat(2) done with flags, the call the
// guest's faccessat has meant since Linux 5.8.
func hostFaccessat2(dirfd int, path string, mode, flags uint32) error {
	p, err := syscall.BytePtrFromString(path)
	if err != nil {
		return err
	}
	_, _, e := syscall.Syscall6(nrFaccessat2, uintptr(dirfd),
		uintptr(unsafe.Pointer(p)), uintptr(mode), uintptr(flags), 0, 0)
	if e != 0 {
		return e
	}
	return nil
}

// hostMemfdCreate is memfd_create(2).
func hostMemfdCreate(name uintptr, flags uintptr) (int, error) {
	r, _, e := syscall.Syscall(nrMemfdCreate, name, flags, 0)
	if e != 0 {
		return int(r), e
	}
	return int(r), nil
}

// hostOPath is O_PATH: opening a directory to hold a pin on it, without
// reading it. The value is the same on x86-64 and aarch64 (and on every
// Linux but alpha/mips/sparc), and the standard library does not export it.
const hostOPath = 0x200000

// hostSocket is socket(2): the standard library's Socket is domain/type/proto
// only for the families it models, and a guest is entitled to any of them
// (AF_NETLINK included), so the call goes through by number.
func hostSocket(domain, typ, proto uintptr) (int, error) {
	r, _, e := syscall.Syscall(nrSocket, domain, typ, proto)
	if e != 0 {
		return int(r), e
	}
	return int(r), nil
}

// hostSocketpair is socketpair(2). The kernel writes both descriptors into
// the caller's array, so only the return value matters here.
func hostSocketpair(domain, typ, proto uintptr, sv *[2]int32) error {
	_, _, e := syscall.Syscall6(nrSocketpair, domain, typ, proto,
		uintptr(unsafe.Pointer(sv)), 0, 0)
	return errnoOf(e)
}

// hostBind, hostConnect, hostListen, hostShutdown: the address-taking calls
// that need the raw sockaddr the guest handed us, translated or not.
func hostBind(fd uintptr, addr unsafe.Pointer, socklen uintptr) error {
	_, _, e := syscall.Syscall(nrBind, fd, uintptr(addr), socklen)
	return errnoOf(e)
}
func hostConnect(fd uintptr, addr unsafe.Pointer, socklen uintptr) error {
	_, _, e := syscall.Syscall(nrConnect, fd, uintptr(addr), socklen)
	return errnoOf(e)
}
func hostListen(fd uintptr, backlog int) error {
	_, _, e := syscall.Syscall(nrListen, fd, uintptr(backlog), 0)
	return errnoOf(e)
}
func hostShutdown(fd, how uintptr) error {
	_, _, e := syscall.Syscall(nrShutdown, fd, how, 0)
	return errnoOf(e)
}

// hostAccept4 is accept4(2) (and accept(2) with flags 0).
func hostAccept4(fd uintptr, addr unsafe.Pointer, socklen *uint32, flags int) (int, error) {
	r, _, e := syscall.Syscall6(nrAccept4, fd, uintptr(addr), uintptr(unsafe.Pointer(socklen)),
		uintptr(flags), 0, 0)
	if e != 0 {
		return int(r), e
	}
	return int(r), nil
}

// hostSendto / hostRecvfrom move bytes and an address in one go.
func hostSendto(fd uintptr, buf []byte, flags int, addr unsafe.Pointer, socklen uintptr) (int, error) {
	var p uintptr
	if len(buf) > 0 {
		p = uintptr(unsafe.Pointer(&buf[0]))
	}
	r, _, e := syscall.Syscall6(nrSendto, fd, p, uintptr(len(buf)), uintptr(flags),
		uintptr(addr), socklen)
	if e != 0 {
		return int(r), e
	}
	return int(r), nil
}
func hostRecvfrom(fd uintptr, buf []byte, flags int, addr unsafe.Pointer, socklen *uint32) (int, error) {
	var p uintptr
	if len(buf) > 0 {
		p = uintptr(unsafe.Pointer(&buf[0]))
	}
	r, _, e := syscall.Syscall6(nrRecvfrom, fd, p, uintptr(len(buf)), uintptr(flags),
		uintptr(addr), uintptr(unsafe.Pointer(socklen)))
	if e != 0 {
		return int(r), e
	}
	return int(r), nil
}

// hostSendmsg / hostRecvmsg take a 56-byte host struct msghdr. Go 1.27's
// syscall package no longer exports a recvmsg wrapper, and the sendmsg one
// insists on modelling the address itself, so both are made by number.
func hostSendmsg(fd uintptr, msg unsafe.Pointer, flags int) (int, error) {
	r, _, e := syscall.Syscall(nrSendmsg, fd, uintptr(msg), uintptr(flags))
	if e != 0 {
		return int(r), e
	}
	return int(r), nil
}
func hostRecvmsg(fd uintptr, msg unsafe.Pointer, flags int) (int, error) {
	r, _, e := syscall.Syscall(nrRecvmsg, fd, uintptr(msg), uintptr(flags))
	if e != 0 {
		return int(r), e
	}
	return int(r), nil
}

// hostGetsockname / hostGetpeername fill a sockaddr_storage and its length.
func hostGetsockname(fd uintptr, addr unsafe.Pointer, socklen *uint32) error {
	_, _, e := syscall.Syscall(nrGetsockname, fd, uintptr(addr), uintptr(unsafe.Pointer(socklen)))
	return errnoOf(e)
}
func hostGetpeername(fd uintptr, addr unsafe.Pointer, socklen *uint32) error {
	_, _, e := syscall.Syscall(nrGetpeername, fd, uintptr(addr), uintptr(unsafe.Pointer(socklen)))
	return errnoOf(e)
}

// hostSetsockopt / hostGetsockopt pass the option value through as raw bytes:
// which options take which shape is the guest kernel's business, and only the
// ones this emulator has to look inside (SO_ATTACH_*) are examined first.
func hostSetsockopt(fd uintptr, level, opt int, val unsafe.Pointer, vlen uintptr) error {
	_, _, e := syscall.Syscall6(nrSetsockopt, fd, uintptr(level), uintptr(opt),
		uintptr(val), vlen, 0)
	return errnoOf(e)
}
func hostGetsockopt(fd uintptr, level, opt int, val unsafe.Pointer, vlen *uint32) error {
	_, _, e := syscall.Syscall6(nrGetsockopt, fd, uintptr(level), uintptr(opt),
		uintptr(val), uintptr(unsafe.Pointer(vlen)), 0)
	return errnoOf(e)
}

// errnoOf turns a Syscall errno into a plain error (0 -> nil).
func errnoOf(e syscall.Errno) error {
	if e == 0 {
		return nil
	}
	return e
}

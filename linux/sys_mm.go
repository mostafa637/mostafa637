// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Ported to Go from src/sys_mm.c — brk, mmap and friends.

package linux

import (
	"os"

	"github.com/mostafa637/mostafa637/abi"
	"github.com/mostafa637/mostafa637/core"
	"github.com/mostafa637/mostafa637/mem"
)

// Mapping flags (the generic Linux values).
const (
	MapShared         = 0x01
	MapPrivate        = 0x02
	MapFixed          = 0x10
	MapAnonymous      = 0x20
	MapGrowthDown     = 0x0100
	MapDenyWrite      = 0x0800
	MapFixedNoreplace = 0x100000
	MapPopulate       = 0x08000
	MapStack          = 0x20000
	MapHugeMask       = 0xfc0000
)

// madvise advice values.
const (
	MADV_NORMAL   = 0
	MADV_DONTNEED = 4
	MADV_FREE     = 8
)

func init() {
	Register(map[uint64]Handler{
		Sysbrk:      sysBrk,
		Sysmmap:     sysMmap,
		Sysmunmap:   sysMunmap,
		Sysmprotect: sysMprotect,
		Sysmadvise:  sysMadvise,
		Sysmsync:    sysMsync,
		Sysmincore:  sysMincore,
		Sysmremap:   sysMremap,
	})
}

func sysBrk(t *Task, a [6]uint64) uint64 {
	if a[0] == 0 {
		if t.AS.BrkStart == 0 {
			t.AS.InitBrk()
		}
		return retOK(t.AS.BrkEnd)
	}
	if t.AS.BrkStart == 0 {
		t.AS.InitBrk()
	}
	return retOK(t.AS.SetBrk(a[0]))
}

func sysMmap(t *Task, a [6]uint64) uint64 {
	addr, length := a[0], a[1]
	prot := uint32(0)
	if a[2]&1 != 0 {
		prot |= mem.PTER
	}
	if a[2]&2 != 0 {
		prot |= mem.PTEW
	}
	if a[2]&4 != 0 {
		prot |= mem.PTEX
	}
	flags := int(a[3])
	if flags&MapFixed != 0 && addr == 0 {
		// MAP_FIXED at address 0 is a caller bug the kernel refuses.
		return negErrno(abi.EPERM)
	}
	if flags&MapFixed == 0 {
		addr = 0
	}
	fd := int(int32(a[4]))
	off := int64(a[5])

	switch {
	case flags&MapAnonymous != 0:
		got, err := t.AS.MapAnon(addr, length, prot)
		if err != nil {
			return retErr(err)
		}
		return retOK(got)
	case fd >= 0:
		if !t.GuestFD(fd) {
			return negErrno(abi.EBADF)
		}
		shared := flags&MapShared != 0
		path := ""
		if link, err := readLinkFD(fd); err == nil {
			path = link
		}
		got, err := t.AS.MapFile(addr, length, prot, fd, off, shared, path)
		if err != nil {
			return retErr(err)
		}
		return retOK(got)
	}
	return negErrno(abi.EINVAL)
}

func sysMunmap(t *Task, a [6]uint64) uint64 {
	t.flushJIT() // the pages a block was translated from may be gone
	return retErr(t.AS.Unmap(a[0], a[1]))
}

func sysMprotect(t *Task, a [6]uint64) uint64 {
	t.flushJIT() // a page that stops being executable invalidates its blocks
	prot := uint32(0)
	if a[2]&1 != 0 {
		prot |= mem.PTER
	}
	if a[2]&2 != 0 {
		prot |= mem.PTEW
	}
	if a[2]&4 != 0 {
		prot |= mem.PTEX
	}
	return retErr(t.AS.Protect(a[0], a[1], prot))
}

// sysMadvise implements the advices that change memory contents and accepts
// the rest as hints: MADV_DONTNEED and MADV_FREE zero the range, which is what
// a guest allocator relies on when it returns pages.
func sysMadvise(t *Task, a [6]uint64) uint64 {
	switch a[2] {
	case MADV_DONTNEED, MADV_FREE:
		start, end := a[0], a[0]+a[1]
		if start&mem.PageMask != 0 {
			return negErrno(abi.EINVAL)
		}
		for va := start; va < end; va += mem.PageSize {
			if p := mem.HostPtr(t.CPU, va, mem.PageSize, core.AccWrite); p != nil {
				for i := range p {
					p[i] = 0
				}
			}
		}
	}
	return 0
}

func sysMsync(t *Task, a [6]uint64) uint64 {
	if _, ok := t.AS.Backing(a[0], a[1]); !ok {
		return negErrno(abi.ENOMEM)
	}
	return 0
}

func sysMincore(t *Task, a [6]uint64) uint64 {
	// Every mapped page of ours is resident: guest memory is host heap. Say so
	// for the whole range, which is what a host with a lazy mapping would say
	// too once touched.
	pages := (a[1] + mem.PageSize - 1) / mem.PageSize
	for va := a[0]; va < a[0]+a[1]; va += mem.PageSize {
		if t.AS.PageProt(va) == 0 {
			return negErrno(abi.ENOMEM)
		}
	}
	buf := make([]byte, pages)
	for i := range buf {
		buf[i] = 1
	}
	return retErr(mem.CopyToGuest(t.CPU, a[2], buf))
}

// sysMremap supports the two forms a guest allocator actually uses: growing a
// private anonymous mapping in place, and moving one (MREMAP_MAYMOVE).
// Anything else reports EINVAL rather than pretending.
func sysMremap(t *Task, a [6]uint64) uint64 {
	t.flushJIT()
	oldAddr, oldLen, newLen := a[0], a[1], a[2]
	flags := a[3]
	const (
		mremapMayMove   = 1
		mremapFixed     = 2
		mremapDontUnmap = 4
	)
	if flags&mremapDontUnmap != 0 || flags&mremapFixed != 0 {
		return negErrno(abi.EINVAL)
	}
	r := t.AS.FindRegion(oldAddr)
	if r == nil {
		return negErrno(abi.EFAULT)
	}
	if newLen <= oldLen {
		// Shrinking: drop the tail.
		if err := t.AS.Unmap(oldAddr+newLen, oldLen-newLen); err != nil {
			return retErr(err)
		}
		return retOK(oldAddr)
	}
	if flags&mremapMayMove == 0 {
		return negErrno(abi.ENOMEM)
	}
	dst := t.AS.FindFree(newLen)
	if dst == 0 {
		return negErrno(abi.ENOMEM)
	}
	src, ok := t.AS.Backing(oldAddr, oldLen)
	if !ok {
		return negErrno(abi.EFAULT)
	}
	newAddr, err := t.AS.MapAnon(dst, newLen, r.Prot)
	if err != nil {
		return retErr(err)
	}
	dstBuf, ok := t.AS.Backing(newAddr, newLen)
	if !ok {
		return retErr(abi.EFAULT)
	}
	copy(dstBuf, src)
	if err := t.AS.Unmap(oldAddr, oldLen); err != nil {
		return retErr(err)
	}
	return retOK(newAddr)
}

func readLinkFD(fd int) (string, error) {
	return os.Readlink(hostProcFD(fd))
}

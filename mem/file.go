// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Ported to Go from the host-backing parts of src/mem.c.
//
// How a region's host memory is obtained and released, and how a file mapping
// keeps the "pages past end-of-file are not there" property the guest's SIGBUS
// depends on.

package mem

import (
	"os"
	"syscall"

	"github.com/mostafa637/mostafa637/abi"
)

// mapFileShared maps length bytes of hostFD at off with MAP_SHARED, so the
// guest's writes reach the file (and are visible to whoever else has it
// mapped). The host protection follows the guest's: a read-only mapping of a
// file that was not opened for writing must fail with EACCES, as on a host.
func mapFileShared(hostFD int, off int64, length uint64, prot uint32) ([]byte, error) {
	host := syscall.PROT_READ
	if prot&PTEW != 0 {
		host |= syscall.PROT_WRITE
	}
	data, err := syscall.Mmap(hostFD, off, int(length), host, syscall.MAP_SHARED)
	if err == syscall.EACCES {
		return nil, abi.EACCES
	}
	if err != nil {
		return nil, abi.Errno(errnoOf(err))
	}
	return data, nil
}

// readFilePrivate builds the backing of a MAP_PRIVATE file mapping: the file's
// contents copied into anonymous memory, because a private mapping's writes
// must never reach the file.
//
// It returns the backing and how much of it is backed. Pages wholly past
// end-of-file are left out of the count, so touching one faults — the guest's
// SIGBUS/BUS_ADRERR, which the C core produces by keeping those PTEs out of
// the page table. The page that straddles end-of-file is backed (a kernel
// zero-fills its remainder), so it counts as mapped in full.
func readFilePrivate(hostFD int, off int64, length uint64) ([]byte, uint64, error) {
	st, err := fstat(hostFD)
	size := int64(0)
	if err == nil {
		size = st.size
	}
	data := make([]byte, length)
	if off < size {
		want := size - off
		if want > int64(length) {
			want = int64(length)
		}
		n, err := syscall.Pread(hostFD, data[:want], off)
		if err != nil && err != syscall.EINTR {
			return nil, 0, abi.Errno(errnoOf(err))
		}
		_ = n
	}
	mapped := uint64(0)
	if size > off {
		avail := size - off
		if avail > int64(length) {
			avail = int64(length)
		}
		mapped = pgUp(uint64(avail))
	}
	return data, mapped, nil
}

// statResult is the little of struct stat the mapping layer needs.
type statResult struct {
	dev  uint64
	ino  uint64
	size int64
}

func fstat(fd int) (statResult, error) {
	var st syscall.Stat_t
	if err := syscall.Fstat(fd, &st); err != nil {
		return statResult{}, err
	}
	return statResult{dev: uint64(st.Dev), ino: st.Ino, size: st.Size}, nil
}

// unmapBacking releases a region's host memory.
func unmapBacking(r *Region) {
	if r.hostMapped {
		_ = syscall.Munmap(r.Data)
	}
	r.Data = nil
}

// applyHostProt re-applies a region's protection to its host backing.
//
// Only a real host mapping needs it (an anonymous region's protection is
// enforced by translate, and its heap pages are already writable). The one
// case it exists for is a MAP_SHARED file mapping: the host must not be left
// more permissive than the guest asked, or a stray write of ours — as opposed
// to one the guest made through translate, which is checked — would reach the
// file. A sub-page range cannot be protected on a host whose pages are bigger
// than the guest's, so the whole region follows, which is the coarsening
// documented in docs/PORT.md.
func applyHostProt(r *Region) {
	if !r.hostMapped {
		return
	}
	host := syscall.PROT_NONE
	if r.Prot&PTER != 0 {
		host |= syscall.PROT_READ
	}
	if r.Prot&PTEW != 0 && r.WROK {
		host |= syscall.PROT_WRITE
	}
	if r.Prot&PTEX != 0 {
		host |= syscall.PROT_EXEC
	}
	_ = syscall.Mprotect(r.Data, host)
}

// errnoOf maps a Go os/syscall error to a Linux errno number. The emulator
// reports host failures to the guest in the guest's own numbering, and the two
// agree on Linux.
func errnoOf(err error) int {
	if e, ok := err.(syscall.Errno); ok {
		return int(e)
	}
	if e, ok := err.(*os.PathError); ok {
		return errnoOf(e.Err)
	}
	return int(syscall.EIO)
}

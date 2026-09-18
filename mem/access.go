// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Address-space access that does not go through a CPU: the ELF loader writes
// the image and the initial stack before there is a thread to blame for a
// fault, and /proc synthesis reads without one either.

package mem

// WriteAt writes data at guest va, walking the region list directly.
//
// It ignores the region's protection on purpose: the loader fills a region it
// has just created, and writes to an unmapped page are the caller's bug rather
// than a guest-visible event. Returns ErrFault on an unmapped page.
func (as *AddrSpace) WriteAt(va uint64, data []byte) error {
	as.mu.Lock()
	defer as.mu.Unlock()
	for len(data) > 0 {
		r := as.findRegionLocked(va)
		if r == nil {
			return ErrFault
		}
		off := va - r.Start
		n := uint64(len(data))
		if room := r.End - va; n > room {
			n = room
		}
		copy(r.Data[off:], data[:n])
		data = data[n:]
		va += n
	}
	return nil
}

// ReadAt reads len(dst) bytes from guest va, walking the region list directly.
// Short reads (a hole in the middle) report ErrFault.
func (as *AddrSpace) ReadAt(dst []byte, va uint64) error {
	as.mu.Lock()
	defer as.mu.Unlock()
	for len(dst) > 0 {
		r := as.findRegionLocked(va)
		if r == nil {
			return ErrFault
		}
		off := va - r.Start
		n := uint64(len(dst))
		if room := r.End - va; n > room {
			n = room
		}
		copy(dst[:n], r.Data[off:])
		dst = dst[n:]
		va += n
	}
	return nil
}

// Backing returns the host slice covering [va, va+len) if one region does,
// which is what a caller that wants to fill or read a whole range at once
// needs (the loader, /proc/self/maps).
func (as *AddrSpace) Backing(va, length uint64) ([]byte, bool) {
	as.mu.Lock()
	defer as.mu.Unlock()
	r := as.findRegionLocked(va)
	if r == nil || va+length > r.End {
		return nil, false
	}
	off := va - r.Start
	return r.Data[off : off+length], true
}

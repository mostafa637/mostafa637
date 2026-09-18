// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Ported to Go from src/mmu.h and src/mem.c.
//
// Package mem is the guest address space: a software "page table" that maps
// guest 4 KB pages to host memory, plus the region list that remembers what
// each range is (its protection, its backing, and the path /proc/self/maps
// should print for it). Guest VAs are decoupled from host VAs, so a 47-bit
// guest address space works on any host.
//
// It installs itself as the core's memory seam: everything the copied core
// does to guest memory goes through the four accessors below.
//
// # Where this differs from the C original
//
// The C page table is two levels of uintptr PTEs whose low bits are the
// protection, so that JIT-generated code can probe it inline. Go cannot hold a
// host pointer as a bare uintptr (nothing would keep the allocation alive), so
// the translation of a page is a slice: the region's backing, sub-sliced to
// the one page. The D-TLB entry format changes with it — 16 bytes of
// {u64 page; uintptr_t pte} becomes {page, slice, prot} — and nothing else in
// the design moves: same direct-mapped, per-thread, generation-invalidated
// cache, same "never cache a miss" rule, same fault reporting.
package mem

import (
	"encoding/binary"
	"sync"
	"sync/atomic"

	"github.com/mostafa637/mostafa637/abi"
	"github.com/mostafa637/mostafa637/core"
)

// Guest address-space geometry.
const (
	PageSize  = 4096
	PageMask  = PageSize - 1
	VABits    = 47
	TaskSize  = 1 << VABits
	StackTop  = 0x7ffffffff000
	MMAPFloor = 0x6000000000
	StackSize = 8 << 20 // 8 MiB, as the kernel's default

	// AArch64 TBI0: on Linux, EL0 data accesses run with TCR_EL1.TBI0=1, so
	// the top byte (VA bits [63:56]) is ignored during translation. Bionic's
	// scudo allocator stores a tag there and dereferences the tagged pointer;
	// the mask strips it.
	TBIMask = 0x00ffffffffffffff
)

// Guest page protection (software-enforced), as the C PTE_R/W/X.
const (
	PTER     = 1
	PTEW     = 2
	PTEX     = 4
	PTEFlags = 7
)

// ErrFault is what a guest copy reports when it hits an unmapped or
// write-protected page: the EFAULT the C helpers return as -EFAULT, so a
// kernel's copy_to_user would fail the syscall rather than signal. It is the
// target errno itself, so the syscall layer can hand it straight back.
var ErrFault error = abi.EFAULT

// Region is one guest mapping.
//
// The C Region is refcount-shared with a HostMap because several regions can
// reference one host mmap after a split or trim, and because a host page
// larger than the guest's 4 KB cannot be unmapped in slices. Go's backing is a
// slice: a sub-slice of a shared array keeps the whole allocation alive with
// no reference counting at all, which is the same property with none of the
// bookkeeping.
type Region struct {
	Start, End uint64
	Prot       uint32 // PTE_R/W/X
	Shared     bool   // MAP_SHARED file mapping
	FileBacked bool   // backed by a host mapping of a file
	Data       []byte // host backing for [Start, End)
	Path       string // guest path for /proc/self/maps, "" if none
	FileOff    uint64 // file offset at Start (file-backed only)
	Dev, Ino   uint64 // the mapped file's identity
	// WROK says the host backing may be made writable: always so except for a
	// MAP_SHARED mapping of a descriptor that was not opened for writing
	// (mprotect -> EACCES).
	WROK bool
	// HostMapped says Data is a real host mapping that must be munmapped (a
	// MAP_SHARED file mapping), not heap memory.
	hostMapped bool

	// Mapped is how much of Data is actually backed. A file mapping whose file
	// is shorter than the mapping leaves the tail unmapped, so a touch there
	// faults — the guest's SIGBUS past end-of-file, which the C core produces
	// by keeping those PTEs out of the table.
	Mapped uint64
}

// AddrSpace is one guest address space.
type AddrSpace struct {
	mu      sync.Mutex
	Regions []Region // sorted by Start, disjoint

	// Gen is bumped by every PTE mutation; a thread whose D-TLB was filled at
	// an older generation empties it. Same protocol as the C as_gen_bump.
	Gen uint64

	BrkStart, BrkEnd uint64
	MmapNext         uint64
	StackTop         uint64

	// What the ELF loader knows about the image it laid out, named after the
	// mm_struct fields they answer (/proc/self/{status,statm,stat}).
	StartCode, EndCode uint64
	StartData, EndData uint64
	StartStack         uint64
	ArgStart, ArgEnd   uint64
	EnvStart, EnvEnd   uint64
	Peak               uint64 // high-water mapped bytes (VmPeak)
}

// NewAddrSpace returns an empty address space.
func NewAddrSpace() *AddrSpace {
	as := &AddrSpace{MmapNext: MMAPFloor, StackTop: StackTop}
	return as
}

// Install wires this package into the core's memory seam. It is called from
// this package's init, so importing mem is enough.
func Install() {
	core.MemRead = memRead
	core.MemWrite = memWrite
	core.MemRead128 = memRead128
	core.MemWrite128 = memWrite128
	core.MemIFetch = memIFetch
	core.MemHostPtr = memHostPtr
	core.TLBFlushAll = tlbFlushAll
}

func init() { Install() }

// CPUAS returns the address space a CPU is running in. The linux-user layer
// stores it in Thread.Owner when it creates the thread.
func CPUAS(c *core.CPU) *AddrSpace {
	as, _ := c.Thread.Owner.(*AddrSpace)
	return as
}

// Bind attaches a CPU (and its thread) to an address space.
func Bind(c *core.CPU, as *AddrSpace) { c.Thread.Owner = as }

// ---- translation ---------------------------------------------------------

// translate resolves a guest VA to the host page that backs it.
//
// It returns nil when the page is not mapped or the access is not permitted;
// permFault says which. The caller reports the fault (or, for the bulk-copy
// helpers, returns EFAULT).
func translate(c *core.CPU, va uint64, need uint32) (page []byte, permFault bool) {
	// TBI0: data accesses ignore the VA top byte. Instruction fetch is left
	// untouched — the PC is never tagged and the fetch fast path compares the
	// raw page. `va` is by value, so the caller keeps the tagged address for
	// fault reporting (FAR retains the tag, as on hardware).
	if need != PTEX {
		va &= TBIMask
	}
	if va >= TaskSize {
		return nil, false
	}
	as := CPUAS(c)
	if as == nil {
		return nil, false
	}
	th := c.Thread

	// The generation check is what makes a lock-free hit safe: any mapping
	// change bumps Gen, and the next lookup empties this thread's cache.
	gen := atomic.LoadUint64(&as.Gen)
	if gen != th.DTLBGen {
		th.DTLB = [core.DTLBEntries]core.DTlbEntry{}
		th.DTLBGen = gen
	}
	pageNum := va >> 12
	e := &th.DTLB[pageNum&(core.DTLBEntries-1)]
	if e.Prot != 0 && e.Page == pageNum {
		if e.Prot&need != need {
			return nil, true
		}
		return e.Data, false
	}

	// Miss: walk the region list under the lock so a concurrent mapper cannot
	// be mid-rewrite. Misses are never cached.
	as.mu.Lock()
	r := as.findRegionLocked(va)
	if r == nil || va >= r.Start+r.Mapped && r.FileBacked {
		as.mu.Unlock()
		return nil, false
	}
	pageBase := va &^ uint64(PageMask)
	if r.Prot&need != need {
		as.mu.Unlock()
		return nil, true
	}
	data := r.Data[pageBase-r.Start:][:PageSize]
	as.mu.Unlock()

	e.Page = pageNum
	e.Data = data
	e.Prot = r.Prot
	return data, false
}

// findRegionLocked returns the region containing va (caller holds as.mu).
func (as *AddrSpace) findRegionLocked(va uint64) *Region {
	lo, hi := 0, len(as.Regions)
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		r := &as.Regions[mid]
		if va < r.Start {
			hi = mid
		} else if va >= r.End {
			lo = mid + 1
		} else {
			return r
		}
	}
	return nil
}

// writable reports whether the region's host backing may be made writable.
func (r *Region) writable() bool {
	return !r.FileBacked || !r.Shared || r.WROK
}

// FindRegion returns the region containing va, or nil.
func (as *AddrSpace) FindRegion(va uint64) *Region {
	as.mu.Lock()
	defer as.mu.Unlock()
	return as.findRegionLocked(va)
}

// PageProt returns the protection of the page at va as mapped (0 = unmapped).
func (as *AddrSpace) PageProt(va uint64) uint32 {
	as.mu.Lock()
	defer as.mu.Unlock()
	r := as.findRegionLocked(va)
	if r == nil {
		return 0
	}
	if r.FileBacked && va >= r.Start+r.Mapped {
		return 0
	}
	return r.Prot
}

// ---- the access seam -----------------------------------------------------

func loadLE(p []byte) uint64 {
	switch len(p) {
	case 1:
		return uint64(p[0])
	case 2:
		return uint64(binary.LittleEndian.Uint16(p))
	case 4:
		return uint64(binary.LittleEndian.Uint32(p))
	case 8:
		return binary.LittleEndian.Uint64(p)
	default:
		var v uint64
		for i := len(p) - 1; i >= 0; i-- {
			v = v<<8 | uint64(p[i])
		}
		return v
	}
}

func storeLE(p []byte, v uint64) {
	switch len(p) {
	case 1:
		p[0] = byte(v)
	case 2:
		binary.LittleEndian.PutUint16(p, uint16(v))
	case 4:
		binary.LittleEndian.PutUint32(p, uint32(v))
	case 8:
		binary.LittleEndian.PutUint64(p, v)
	default:
		for i := range p {
			p[i] = byte(v)
			v >>= 8
		}
	}
}

// raiseDataAbort reports a failed data access to the guest. The DFSC encodes
// unmapped (translation fault -> SEGV_MAPERR) vs permission (-> SEGV_ACCERR)
// for precise siginfo later, and a hole in a file mapping as an external abort
// (-> SIGBUS/BUS_ADRERR), which is what the kernel reports past end-of-file.
func raiseDataAbort(c *core.CPU, va uint64, write, perm bool) {
	fsc := uint32(core.FSCTransL3)
	switch {
	case perm:
		fsc = core.FSCPermL3
	case isFileHole(c, va):
		fsc = core.FSCExternal
	}
	c.RaiseSync(core.ESRMake(core.ECDAbortLower, core.ISSDataAbort(write, uint(fsc))), va)
}

// isFileHole reports whether va lies in the unmapped tail of a file mapping.
func isFileHole(c *core.CPU, va uint64) bool {
	as := CPUAS(c)
	if as == nil {
		return false
	}
	as.mu.Lock()
	defer as.mu.Unlock()
	r := as.findRegionLocked(va)
	return r != nil && r.FileBacked && va >= r.Start+r.Mapped
}

func memRead(c *core.CPU, va uint64, size uint, out *uint64) bool {
	// Split accesses that cross a page boundary (unaligned in-page accesses
	// are plain copies — EL0 Linux semantics, SCTLR.A clear).
	if (va&PageMask)+uint64(size) > PageSize {
		first := uint(PageSize - (va & PageMask))
		var lo, hi uint64
		if !memRead(c, va, first, &lo) {
			return false
		}
		if !memRead(c, va+uint64(first), size-first, &hi) {
			return false
		}
		*out = lo | hi<<(8*first)
		return true
	}
	p, perm := translate(c, va, PTER)
	if p == nil {
		raiseDataAbort(c, va, false, perm)
		return false
	}
	*out = loadLE(p[va&PageMask:][:size])
	return true
}

func memWrite(c *core.CPU, va uint64, size uint, val uint64) bool {
	if (va&PageMask)+uint64(size) > PageSize {
		first := uint(PageSize - (va & PageMask))
		if !memWrite(c, va, first, val) {
			return false
		}
		return memWrite(c, va+uint64(first), size-first, val>>(8*first))
	}
	p, perm := translate(c, va, PTEW)
	if p == nil {
		raiseDataAbort(c, va, true, perm)
		return false
	}
	storeLE(p[va&PageMask:][:size], val)
	return true
}

func memRead128(c *core.CPU, va uint64, out *core.V128) bool {
	if (va&PageMask)+16 > PageSize {
		var lo, hi uint64
		if !memRead(c, va, 8, &lo) || !memRead(c, va+8, 8, &hi) {
			return false
		}
		out.SetU64(0, lo)
		out.SetU64(1, hi)
		return true
	}
	p, perm := translate(c, va, PTER)
	if p == nil {
		raiseDataAbort(c, va, false, perm)
		return false
	}
	copy(out.Bytes(), p[va&PageMask:][:16])
	return true
}

func memWrite128(c *core.CPU, va uint64, val *core.V128) bool {
	if (va&PageMask)+16 > PageSize {
		return memWrite(c, va, 8, val.U64(0)) && memWrite(c, va+8, 8, val.U64(1))
	}
	p, perm := translate(c, va, PTEW)
	if p == nil {
		raiseDataAbort(c, va, true, perm)
		return false
	}
	copy(p[va&PageMask:][:16], val.Bytes())
	return true
}

// memIFetch is the instruction-fetch fast path: it caches the host backing of
// the current code page so sequential fetches skip the walk. It is invalidated
// by any mapping change (tlbFlushAll).
//
// The 4-byte alignment of the PC has to be tested here and not only in the
// slow path: a branch to a misaligned address inside the page this thread is
// already fetching from would otherwise never reach memIFetchSlow, so the
// PC-alignment exception never happened — the misparsed word was decoded
// instead. Keeping va's low two bits in the compared key (mask ~0xffc rather
// than ~0xfff) makes any misaligned VA miss and take the slow path.
func memIFetch(c *core.CPU, va uint64, insn *uint32) bool {
	th := c.Thread
	if th.FData != nil && va&^uint64(0xffc) == th.FPage {
		*insn = binary.LittleEndian.Uint32(th.FData[va&PageMask:])
		return true
	}
	return memIFetchSlow(c, va, insn)
}

func memIFetchSlow(c *core.CPU, va uint64, insn *uint32) bool {
	if va&3 != 0 {
		c.RaiseSync(core.ESRMake(core.ECPCAlign, 0), va)
		return false
	}
	p, perm := translate(c, va, PTEX)
	if p == nil {
		// As raiseDataAbort: executing from past end-of-file is a bus error.
		fsc := uint32(core.FSCTransL3)
		if perm {
			fsc = core.FSCPermL3
		} else if isFileHole(c, va) {
			fsc = core.FSCExternal
		}
		c.RaiseSync(core.ESRMake(core.ECIAbortLower, fsc), va)
		return false
	}
	th := c.Thread
	th.FPage = va &^ uint64(PageMask)
	th.FData = p
	*insn = binary.LittleEndian.Uint32(p[va&PageMask:])
	return true
}

// memHostPtr returns a stable host slice for [va, va+size) when the whole
// range lies in one guest page and is permitted for acc; nil otherwise. It is
// the substrate for the futex/atomics/DC ZVA fast paths.
func memHostPtr(c *core.CPU, va uint64, size uint, acc core.AccType) []byte {
	if (va&PageMask)+uint64(size) > PageSize {
		return nil
	}
	need := uint32(PTER)
	switch acc {
	case core.AccWrite:
		need = PTEW
	case core.AccExec:
		need = PTEX
	}
	p, _ := translate(c, va, need)
	if p == nil {
		return nil
	}
	return p[va&PageMask:][:size]
}

// HostPtr is the exported form of the seam (used by the syscall layer).
func HostPtr(c *core.CPU, va uint64, size uint, acc core.AccType) []byte {
	return memHostPtr(c, va, size, acc)
}

// tlbFlushAll empties this thread's cached translations. Every thread picks
// the new generation up at its next lookup.
func tlbFlushAll() {
	// The generation lives on the address space; bumping it is what
	// invalidates every thread. Callers reach it through as.bump().
}

// bump publishes a mapping change: it drops every thread's cached translations
// and (for code ranges) any translation the JIT made of them.
func (as *AddrSpace) bump() {
	atomic.AddUint64(&as.Gen, 1)
	if JITInvalidate != nil {
		JITInvalidate(as)
	}
}

// JITInvalidate, when set, drops translations of guest code that a mapping
// change has overwritten. Installed by the jit package.
var JITInvalidate func(as *AddrSpace)

// InvalidateRange drops cached translations of [start, start+len).
func (as *AddrSpace) InvalidateRange(start, len uint64) {
	as.mu.Lock()
	as.bump()
	as.mu.Unlock()
}

// ---- mapping -------------------------------------------------------------

// MapAnon maps anonymous zeroed memory. addr==0 means "anywhere".
func (as *AddrSpace) MapAnon(addr, length uint64, prot uint32) (uint64, error) {
	length = pgUp(length)
	as.mu.Lock()
	defer as.mu.Unlock()
	if addr == 0 {
		addr = as.findFreeLocked(length)
		if addr == 0 {
			return 0, abi.ENOMEM
		}
	}
	if !rangeOK(addr, length) {
		return 0, abi.EINVAL
	}
	as.punchLocked(addr, addr+length)
	r := Region{
		Start: addr, End: addr + length, Prot: prot,
		Data: make([]byte, length), Mapped: length,
	}
	as.insertLocked(r)
	as.bump()
	return addr, nil
}

// MapFile maps host fd at off. shared selects MAP_SHARED semantics: the host
// mapping is shared, so the guest's writes reach the file.
func (as *AddrSpace) MapFile(addr, length uint64, prot uint32, hostFD int,
	off int64, shared bool, path string) (uint64, error) {
	length = pgUp(length)
	as.mu.Lock()
	defer as.mu.Unlock()
	if addr == 0 {
		addr = as.findFreeLocked(length)
		if addr == 0 {
			return 0, abi.ENOMEM
		}
	}
	if !rangeOK(addr, length) {
		return 0, abi.EINVAL
	}
	as.punchLocked(addr, addr+length)

	r := Region{
		Start: addr, End: addr + length, Prot: prot, Shared: shared,
		FileBacked: true, Path: path, FileOff: uint64(off),
		Mapped: length,
	}
	r.WROK = true
	if shared {
		data, err := mapFileShared(hostFD, off, length, prot)
		if err != nil {
			return 0, err
		}
		r.Data = data
		r.hostMapped = true
		if prot&PTEW == 0 {
			r.WROK = false
		}
	} else {
		data, mapped, err := readFilePrivate(hostFD, off, length)
		if err != nil {
			return 0, err
		}
		r.Data = data
		r.Mapped = mapped
	}
	if st, err := fstat(hostFD); err == nil {
		r.Dev, r.Ino = st.dev, st.ino
	}
	as.insertLocked(r)
	as.bump()
	return addr, nil
}

// Unmap removes [addr, addr+len).
func (as *AddrSpace) Unmap(addr, length uint64) error {
	if !rangeOK(addr, length) {
		return abi.EINVAL
	}
	as.mu.Lock()
	defer as.mu.Unlock()
	as.punchLocked(addr, addr+pgUp(length))
	as.bump()
	return nil
}

// Protect changes the protection of [addr, addr+len), splitting regions at the
// edges so nothing outside the range is touched.
func (as *AddrSpace) Protect(addr, length uint64, prot uint32) error {
	if !rangeOK(addr, length) {
		return abi.EINVAL
	}
	start, end := addr, addr+pgUp(length)
	as.mu.Lock()
	defer as.mu.Unlock()
	as.splitAtLocked(start)
	as.splitAtLocked(end)
	for i := range as.Regions {
		r := &as.Regions[i]
		if r.Start >= start && r.End <= end {
			// A read-only shared mapping of a file that was not opened for
			// writing cannot be made writable, as on the host.
			if prot&PTEW != 0 && r.FileBacked && r.Shared && !r.writable() {
				return abi.EACCES
			}
			r.Prot = prot
			applyHostProt(r)
		}
	}
	as.mergeAllLocked()
	as.bump()
	return nil
}

// SetRegionPath names the pathless regions in [start, end) (ELF images, for
// /proc/self/maps).
func (as *AddrSpace) SetRegionPath(start, end uint64, path string) {
	as.mu.Lock()
	defer as.mu.Unlock()
	for i := range as.Regions {
		r := &as.Regions[i]
		if r.Start >= start && r.End <= end && r.Path == "" {
			r.Path = path
		}
	}
}

// FindFree picks an unused guest VA range of `len` bytes for mmap(NULL, ...).
func (as *AddrSpace) FindFree(length uint64) uint64 {
	as.mu.Lock()
	defer as.mu.Unlock()
	return as.findFreeLocked(length)
}

// MappedBytes returns the number of mapped bytes (what RLIMIT_AS bounds).
func (as *AddrSpace) MappedBytes() uint64 {
	as.mu.Lock()
	defer as.mu.Unlock()
	var n uint64
	for _, r := range as.Regions {
		n += r.End - r.Start
	}
	return n
}

// Destroy tears down every mapping and its host backing.
func (as *AddrSpace) Destroy() {
	as.mu.Lock()
	defer as.mu.Unlock()
	for i := range as.Regions {
		unmapBacking(&as.Regions[i])
	}
	as.Regions = nil
	as.bump()
}

// Brk sets the program break and returns it (0 = unchanged, as brk(2) reports
// failure by returning the current break).
func (as *AddrSpace) SetBrk(addr uint64) uint64 {
	as.mu.Lock()
	defer as.mu.Unlock()
	if as.BrkStart == 0 {
		return 0
	}
	if addr < as.BrkStart || addr >= TaskSize {
		return as.BrkEnd
	}
	if addr > as.BrkEnd {
		length := pgUp(addr - as.BrkEnd)
		if as.findRegionLocked(as.BrkEnd) != nil {
			return as.BrkEnd
		}
		as.insertLocked(Region{
			Start: as.BrkEnd, End: as.BrkEnd + length,
			Prot: PTER | PTEW, Data: make([]byte, length), Mapped: length,
		})
		as.bump()
		as.BrkEnd += length
	} else if addr < as.BrkEnd {
		as.punchLocked(pgUp(addr), as.BrkEnd)
		as.bump()
		as.BrkEnd = addr
	}
	return as.BrkEnd
}

// InitBrk places the program break above the highest existing mapping.
func (as *AddrSpace) InitBrk() uint64 {
	as.mu.Lock()
	defer as.mu.Unlock()
	top := uint64(MMAPFloor)
	for _, r := range as.Regions {
		if r.End > top {
			top = r.End
		}
	}
	top = pgUp(top + PageSize)
	as.BrkStart = top
	as.BrkEnd = top
	return top
}

// ---- region list ---------------------------------------------------------

func pgUp(v uint64) uint64   { return (v + PageMask) &^ uint64(PageMask) }
func pgDown(v uint64) uint64 { return v &^ uint64(PageMask) }
func rangeOK(addr, l uint64) bool {
	return addr&(uint64(PageMask)) == 0 && l != 0 && addr < TaskSize && addr+l <= TaskSize
}

func (as *AddrSpace) insertLocked(r Region) {
	i := 0
	for i < len(as.Regions) && as.Regions[i].Start < r.Start {
		i++
	}
	as.Regions = append(as.Regions, Region{})
	copy(as.Regions[i+1:], as.Regions[i:])
	as.Regions[i] = r
	as.mergeAllLocked()
	if total := as.MappedBytesLocked(); total > as.Peak {
		as.Peak = total
	}
}

// punchLocked removes [start, end) from every region it overlaps, splitting
// the regions at the edges.
func (as *AddrSpace) punchLocked(start, end uint64) {
	as.splitAtLocked(start)
	as.splitAtLocked(end)
	kept := as.Regions[:0]
	for _, r := range as.Regions {
		if r.Start >= start && r.End <= end {
			unmapBacking(&r)
			continue
		}
		kept = append(kept, r)
	}
	as.Regions = kept
}

// splitAtLocked splits the region containing va at va.
func (as *AddrSpace) splitAtLocked(va uint64) {
	if va&uint64(PageMask) != 0 {
		return
	}
	for i := range as.Regions {
		r := &as.Regions[i]
		if va > r.Start && va < r.End {
			off := va - r.Start
			right := *r
			right.Start = va
			right.Data = r.Data[off:]
			right.FileOff = r.FileOff + off
			right.Mapped = satSub(r.Mapped, off)
			r.End = va
			r.Data = r.Data[:off]
			if r.Mapped > off {
				r.Mapped = off
			}
			as.Regions = append(as.Regions, Region{})
			copy(as.Regions[i+2:], as.Regions[i+1:])
			as.Regions[i+1] = right
			return
		}
	}
}

// mergeAllLocked coalesces adjacent regions that describe the same thing, so a
// long-lived guest does not accumulate a region per mmap.
func (as *AddrSpace) mergeAllLocked() {
	out := as.Regions[:0]
	for _, r := range as.Regions {
		if len(out) > 0 {
			p := &out[len(out)-1]
			if mergeable(p, &r) {
				p.End = r.End
				p.Data = p.Data[:p.End-p.Start]
				continue
			}
		}
		out = append(out, r)
	}
	as.Regions = out
}

func mergeable(a, b *Region) bool {
	if a.End != b.Start || a.Prot != b.Prot || a.Shared != b.Shared ||
		a.FileBacked != b.FileBacked || a.Path != b.Path {
		return false
	}
	if a.FileBacked && (a.FileOff+a.End-a.Start != b.FileOff || a.Dev != b.Dev || a.Ino != b.Ino) {
		return false
	}
	return true
}

func satSub(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return 0
}

// findFreeLocked picks an unused range of `len` bytes for mmap(NULL, ...).
//
// It bumps rather than recycling a just-freed VA: handing a freed address
// straight back means a thread holding a stale translation for it can read the
// new mapping at the old mapping's contents. Correct guests do not touch a
// range they freed, so either allocator is right, but not reusing the address
// turns that class of guest bug into a fault instead of silent stale data.
func (as *AddrSpace) findFreeLocked(length uint64) uint64 {
	base := as.MmapNext
	for pass := 0; pass < 2; pass++ {
		for base+length <= TaskSize-0x10000000 {
			conflict := uint64(0)
			for i := range as.Regions {
				r := &as.Regions[i]
				if r.Start < base+length && base < r.End {
					conflict = r.End
					break
				}
			}
			if conflict == 0 {
				as.MmapNext = base + length
				return base
			}
			base = conflict
		}
		base = MMAPFloor // wrapped: rescan from the mmap floor
	}
	return 0
}

// MappedBytesLocked is MappedBytes with the lock already held.
func (as *AddrSpace) MappedBytesLocked() uint64 {
	var n uint64
	for _, r := range as.Regions {
		n += r.End - r.Start
	}
	return n
}

// RegionsSnapshot returns a copy of the region list (for /proc/self/maps).
func (as *AddrSpace) RegionsSnapshot() []Region {
	as.mu.Lock()
	defer as.mu.Unlock()
	out := make([]Region, len(as.Regions))
	copy(out, as.Regions)
	return out
}

// ---- bulk copies for the syscall layer ----------------------------------
//
// None of these ever raise a guest exception: a page the walk refuses is
// EFAULT, which is what a kernel's own copy_to_user does.

// CopyFromGuest copies len(dst) bytes from guest va.
func CopyFromGuest(c *core.CPU, dst []byte, va uint64) error {
	for len(dst) > 0 {
		chunk := uint64(PageSize - (va & PageMask))
		if chunk > uint64(len(dst)) {
			chunk = uint64(len(dst))
		}
		p, _ := translate(c, va, PTER)
		if p == nil {
			return ErrFault
		}
		copy(dst[:chunk], p[va&PageMask:])
		dst = dst[chunk:]
		va += chunk
	}
	return nil
}

// CopyToGuest copies src to guest va.
func CopyToGuest(c *core.CPU, va uint64, src []byte) error {
	for len(src) > 0 {
		chunk := uint64(PageSize - (va & PageMask))
		if chunk > uint64(len(src)) {
			chunk = uint64(len(src))
		}
		p, _ := translate(c, va, PTEW)
		if p == nil {
			return ErrFault
		}
		copy(p[va&PageMask:], src[:chunk])
		src = src[chunk:]
		va += chunk
	}
	return nil
}

// CopyFromGuestPartial is CopyFromGuest for process_vm_readv: it returns the
// number of bytes copied before the first bad page.
func CopyFromGuestPartial(c *core.CPU, dst []byte, va uint64) int {
	done := 0
	for len(dst) > 0 {
		chunk := uint64(PageSize - (va & PageMask))
		if chunk > uint64(len(dst)) {
			chunk = uint64(len(dst))
		}
		p, _ := translate(c, va, PTER)
		if p == nil {
			break
		}
		copy(dst[:chunk], p[va&PageMask:])
		dst = dst[chunk:]
		va += chunk
		done += int(chunk)
	}
	return done
}

// CopyToGuestPartial is CopyToGuest for process_vm_writev.
func CopyToGuestPartial(c *core.CPU, va uint64, src []byte) int {
	done := 0
	for len(src) > 0 {
		chunk := uint64(PageSize - (va & PageMask))
		if chunk > uint64(len(src)) {
			chunk = uint64(len(src))
		}
		p, _ := translate(c, va, PTEW)
		if p == nil {
			break
		}
		copy(p[va&PageMask:], src[:chunk])
		src = src[chunk:]
		va += chunk
		done += int(chunk)
	}
	return done
}

// CopyStringFromGuest reads a NUL-terminated guest string.
func CopyStringFromGuest(c *core.CPU, va uint64, max int) (string, error) {
	var b []byte
	for len(b) < max {
		chunk := PageSize - int(va&PageMask)
		if chunk > max-len(b) {
			chunk = max - len(b)
		}
		p, _ := translate(c, va, PTER)
		if p == nil {
			return "", ErrFault
		}
		seg := p[va&PageMask:][:chunk]
		if i := indexByte(seg, 0); i >= 0 {
			b = append(b, seg[:i]...)
			return string(b), nil
		}
		b = append(b, seg...)
		va += uint64(chunk)
	}
	return "", ErrNameTooLong
}

// ErrNameTooLong is ENAMETOOLONG: no NUL within the caller's limit.
var ErrNameTooLong error = abi.ENAMETOOLONG

func indexByte(b []byte, c byte) int {
	for i := range b {
		if b[i] == c {
			return i
		}
	}
	return -1
}

// ReadWord is a non-faulting read for diagnostics (mem_peek in the C core).
func ReadWord(c *core.CPU, va uint64, size uint) (uint64, bool) {
	if (va&PageMask)+uint64(size) > PageSize {
		return 0, false
	}
	p, _ := translate(c, va, PTER)
	if p == nil {
		return 0, false
	}
	return loadLE(p[va&PageMask:][:size]), true
}

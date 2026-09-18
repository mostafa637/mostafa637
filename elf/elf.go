// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Ported to Go from src/elf.c.
//
// Package elf is the AArch64 ELF64 loader: PT_LOAD segments, PT_INTERP (the
// dynamic linker, taken from the rootfs), and the initial stack with
// argv/envp/auxv laid out exactly as a Linux kernel lays it out.
package elf

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"os"

	"github.com/mostafa637/mostafa637/abi"
	"github.com/mostafa637/mostafa637/core"
	"github.com/mostafa637/mostafa637/mem"
)

// Guest stack geometry.
const (
	StackTop = 0x7ffffff000
	// StkLim is the kernel's own _STK_LIM: the 8 MB reference stack its
	// argument budget is measured against.
	StkLim = 8 << 20
	// StackMax is how far the guest's own RLIMIT_STACK is honoured. A kernel
	// grows the stack a page at a time and needs no ceiling; this maps it
	// whole, so a limit past this one is capped. 64 MB is the
	// `ulimit -s 65536` a deeply recursive workload asks for.
	StackMax = 64 << 20
	// StackFixed is what the initial stack holds besides the strings and the
	// pointer vectors: the auxv array, the blocks it points at, and the
	// alignment between them.
	StackFixed = 1024
	// ETDynBase is where a PIE's interpreter-free image would go.
	ETDynBase = 0x5500000000
)

// ELF constants.
const (
	ELFCLASS64  = 2
	ELFDATA2LSB = 1
	EM_AARCH64  = 183
	ET_EXEC     = 2
	ET_DYN      = 3

	PT_LOAD   = 1
	PT_INTERP = 3

	PF_X = 1
	PF_W = 2
	PF_R = 4
)

// Auxiliary-vector tags (the generic Linux numbering).
const (
	AT_NULL        = 0
	AT_PHDR        = 3
	AT_PHENT       = 4
	AT_PHNUM       = 5
	AT_PAGESZ      = 6
	AT_BASE        = 7
	AT_FLAGS       = 8
	AT_ENTRY       = 9
	AT_UID         = 11
	AT_EUID        = 12
	AT_GID         = 13
	AT_EGID        = 14
	AT_PLATFORM    = 15
	AT_HWCAP       = 16
	AT_CLKTCK      = 17
	AT_SECURE      = 23
	AT_RANDOM      = 25
	AT_HWCAP2      = 26
	AT_EXECFN      = 31
	AT_MINSIGSTKSZ = 51
)

// LoadInfo is what the loader learned about an image.
type LoadInfo struct {
	Base      uint64 // load bias (0 for ET_EXEC)
	Entry     uint64 // biased e_entry
	PhdrVA    uint64 // biased VA of the program headers
	PhNum     uint16
	Lo, Hi    uint64 // biased load span
	StartCode uint64
	EndCode   uint64
	StartData uint64
	EndData   uint64
	Interp    string // PT_INTERP contents, "" for a static image
}

// Loader loads AArch64 ELF images into a guest address space.
type Loader struct {
	AS *mem.AddrSpace
	// Resolve turns a guest path into a host path. It is the rootfs
	// containment from src/path.c; the loader needs it for PT_INTERP, whose
	// interpreter lives in the guest's rootfs rather than beside the image.
	Resolve func(guest string) (host string, err error)
}

// elf64Header is the ELF64 file header.
type elf64Header struct {
	Ident     [16]byte
	Type      uint16
	Machine   uint16
	Version   uint32
	Entry     uint64
	PHOff     uint64
	SHOff     uint64
	Flags     uint32
	EHSize    uint16
	PHEntSize uint16
	PHNum     uint16
	SHEntSize uint16
	SHNum     uint16
	SHStrndx  uint16
}

// elf64Prog is one ELF64 program header.
type elf64Prog struct {
	Type   uint32
	Flags  uint32
	Off    uint64
	VAddr  uint64
	PAddr  uint64
	Filesz uint64
	Memsz  uint64
	Align  uint64
}

var elfMagic = []byte{0x7f, 'E', 'L', 'F'}

// headerCheck validates an image and returns its interpreter, if it names one.
func headerCheck(f *os.File) (elf64Header, []elf64Prog, error) {
	var eh elf64Header
	buf := make([]byte, 64)
	if _, err := f.ReadAt(buf, 0); err != nil {
		return eh, nil, abi.ENOEXEC
	}
	if err := binary.Read(bytes.NewReader(buf), binary.LittleEndian, &eh); err != nil {
		return eh, nil, abi.ENOEXEC
	}
	if !bytes.Equal(eh.Ident[:4], elfMagic) || eh.Ident[4] != ELFCLASS64 ||
		eh.Ident[5] != ELFDATA2LSB || eh.Machine != EM_AARCH64 ||
		(eh.Type != ET_EXEC && eh.Type != ET_DYN) {
		return eh, nil, abi.ENOEXEC
	}
	if eh.PHEntSize != 56 || eh.PHNum == 0 || eh.PHNum > 128 {
		return eh, nil, abi.ENOEXEC
	}
	ph := make([]elf64Prog, eh.PHNum)
	pbuf := make([]byte, int(eh.PHNum)*56)
	if _, err := f.ReadAt(pbuf, int64(eh.PHOff)); err != nil {
		return eh, nil, abi.ENOEXEC
	}
	if err := binary.Read(bytes.NewReader(pbuf), binary.LittleEndian, &ph); err != nil {
		return eh, nil, abi.ENOEXEC
	}
	return eh, ph, nil
}

func interpOf(f *os.File, ph []elf64Prog) (string, error) {
	for i := range ph {
		p := &ph[i]
		if p.Type != PT_INTERP {
			continue
		}
		if p.Filesz == 0 || p.Filesz >= 4096 {
			return "", abi.ENOEXEC
		}
		buf := make([]byte, p.Filesz)
		if _, err := f.ReadAt(buf, int64(p.Off)); err != nil {
			return "", abi.ENOEXEC
		}
		return string(bytes.TrimRight(buf, "\x00")), nil
	}
	return "", nil
}

// Load loads one ELF image. fixedBase is the ET_DYN load bias to request, or
// -1 to allocate from the mmap area.
func (l *Loader) Load(f *os.File, fixedBase int64, gpath string) (*LoadInfo, error) {
	eh, ph, err := headerCheck(f)
	if err != nil {
		return nil, err
	}
	interp, err := interpOf(f, ph)
	if err != nil {
		return nil, err
	}

	lo, hi := ^uint64(0), uint64(0)
	sc, ec, sd, ed := ^uint64(0), uint64(0), uint64(0), uint64(0)
	for i := range ph {
		p := &ph[i]
		if p.Type != PT_LOAD {
			continue
		}
		// The kernel's own segment sanity check, made where the kernel makes
		// it: a segment holding more file bytes than it has memory for, or one
		// whose memory extent wraps, cannot be loaded.
		if p.Filesz > p.Memsz {
			return nil, abi.EINVAL
		}
		end := p.VAddr + p.Memsz
		if end < p.VAddr {
			return nil, abi.EINVAL
		}
		if pgDown(p.VAddr) < lo {
			lo = pgDown(p.VAddr)
		}
		if end > hi {
			hi = end
		}
		fend := p.VAddr + p.Filesz
		// start/end_code cover the executable PT_LOADs only, while start_data
		// is the LAST loaded segment's address and end_data the highest file
		// end of any of them — binfmt_elf's own rules, because /proc reports
		// them and status splits VmExe from VmLib with the code span.
		if p.Flags&PF_X != 0 {
			if p.VAddr < sc {
				sc = p.VAddr
			}
			if fend > ec {
				ec = fend
			}
		}
		if p.VAddr > sd {
			sd = p.VAddr
		}
		if fend > ed {
			ed = fend
		}
	}
	if lo == ^uint64(0) {
		return nil, abi.ENOEXEC // nothing to load
	}
	if hi > ^uint64(mem.PageMask) {
		return nil, abi.EINVAL
	}
	hi = pgUp(hi)

	base := uint64(0)
	if eh.Type == ET_DYN {
		if fixedBase >= 0 {
			base = uint64(fixedBase)
		} else {
			free := l.AS.FindFree(hi - lo)
			if free == 0 {
				return nil, abi.ENOMEM
			}
			base = free - lo
		}
	}

	// One anonymous RW region for the whole span, then per-segment content and
	// protection; pages in the span that no segment covers become no-access.
	if _, err := l.AS.MapAnon(base+lo, hi-lo, mem.PTER|mem.PTEW); err != nil {
		return nil, err
	}
	// Segment content. The whole span was mapped writable so the file bytes
	// can be written into it; the segment's real protection goes on after.
	for i := range ph {
		p := &ph[i]
		if p.Type != PT_LOAD {
			continue
		}
		if p.Filesz > 0 {
			buf := make([]byte, p.Filesz)
			if _, err := f.ReadAt(buf, int64(p.Off)); err != nil {
				return nil, abi.EIO
			}
			if err := l.AS.WriteAt(base+p.VAddr, buf); err != nil {
				return nil, err
			}
		}
	}
	// Everything the segments do not cover becomes no-access, exactly as a
	// kernel leaves the holes between them; then each segment's own pages get
	// the protection its flags ask for. Bytes between p_filesz and p_memsz are
	// bss and stay zero — the region was created zeroed.
	if err := l.AS.Protect(base+lo, hi-lo, 0); err != nil {
		return nil, err
	}
	for i := range ph {
		p := &ph[i]
		if p.Type != PT_LOAD {
			continue
		}
		start := pgDown(base + p.VAddr)
		end := pgUp(base + p.VAddr + p.Memsz)
		if err := l.AS.Protect(start, end-start, pfToProt(p.Flags)); err != nil {
			return nil, err
		}
	}
	l.AS.SetRegionPath(base+lo, base+hi, gpath)

	out := &LoadInfo{
		Base: base, Entry: base + eh.Entry, PhNum: eh.PHNum,
		Lo: base + lo, Hi: base + hi, Interp: interp,
	}
	if sc != ^uint64(0) {
		out.StartCode = base + sc
	}
	if ec != 0 {
		out.EndCode = base + ec
	}
	out.StartData = base + sd
	out.EndData = base + ed
	// AT_PHDR: the VA of the phdr table is the PT_LOAD that covers e_phoff.
	for i := range ph {
		p := &ph[i]
		if p.Type != PT_LOAD {
			continue
		}
		if eh.PHOff >= p.Off && eh.PHOff+uint64(eh.PHNum)*56 <= p.Off+p.Filesz {
			out.PhdrVA = base + p.VAddr + (eh.PHOff - p.Off)
			break
		}
	}
	return out, nil
}

// LoadProgram loads the image at guestPath (plus its interpreter, if it names
// one) and returns the entry point and the SP the image starts on.
func (l *Loader) LoadProgram(guestPath string, argv, envp []string, stackLimit uint64) (entry, sp uint64, err error) {
	host, err := l.resolve(guestPath)
	if err != nil {
		return 0, 0, err
	}
	f, err := os.Open(host)
	if err != nil {
		return 0, 0, errnoFrom(err)
	}
	defer f.Close()

	exe, err := l.Load(f, -1, guestPath)
	if err != nil {
		return 0, 0, err
	}

	entry = exe.Entry
	atBase := uint64(0)
	if exe.Interp != "" {
		host, rerr := l.resolve(exe.Interp)
		if rerr != nil {
			return 0, 0, rerr
		}
		interp, ierr := l.LoadFile(host, -1, exe.Interp)
		if ierr != nil {
			return 0, 0, ierr
		}
		entry = interp.Entry
		atBase = interp.Base
	}

	// Program break after the executable image, and the spans /proc reports.
	// The executable's, never the interpreter's: mm_struct records one image,
	// and it is the one that was execve'd.
	l.AS.BrkStart = pgUp(exe.Hi)
	l.AS.BrkEnd = pgUp(exe.Hi)
	l.AS.StartCode = exe.StartCode
	l.AS.EndCode = exe.EndCode
	l.AS.StartData = exe.StartData
	l.AS.EndData = exe.EndData

	sp, err = l.SetupStack(argv, envp, guestPath, exe, atBase, stackLimit)
	if err != nil {
		return 0, 0, err
	}
	return entry, sp, nil
}

// LoadFile loads the image at a host path (used for the interpreter).
func (l *Loader) LoadFile(hostPath string, fixedBase int64, gpath string) (*LoadInfo, error) {
	f, err := os.Open(hostPath)
	if err != nil {
		return nil, errnoFrom(err)
	}
	defer f.Close()
	return l.Load(f, fixedBase, gpath)
}

func (l *Loader) resolve(guest string) (string, error) {
	if l.Resolve == nil {
		return guest, nil
	}
	return l.Resolve(guest)
}

// SetupStack builds the initial stack: strings at the top, then the pointer
// vectors below them, with SP 16-aligned as the AArch64 procedure call
// standard requires.
func (l *Loader) SetupStack(argv, envp []string, canon string, exe *LoadInfo,
	atBase, stackLimit uint64) (uint64, error) {
	size := stackSize(stackLimit)
	if _, err := l.AS.MapAnon(StackTop-size, size, mem.PTER|mem.PTEW); err != nil {
		return 0, err
	}
	l.AS.StackTop = StackTop

	// Strings at the top of the stack, in the kernel's layout: argv strings
	// lowest and ascending, envp strings byte-packed directly above them,
	// execfn topmost. setproctitle-style rewriting derives its writable span
	// from the argv/envp pointers assuming exactly this order; with argv[0]
	// placed above argv[argc-1] the span underflows and the rewrite memsets
	// off the stack top.
	strtab := uint64(len(canon) + 1)
	for _, s := range argv {
		strtab += uint64(len(s) + 1)
	}
	for _, s := range envp {
		strtab += uint64(len(s) + 1)
	}
	sp := StackTop - strtab
	str := sp
	l.AS.ArgStart = sp
	argvp := make([]uint64, len(argv)+1)
	for i, s := range argv {
		if err := l.AS.WriteAt(str, append([]byte(s), 0)); err != nil {
			return 0, err
		}
		argvp[i] = str
		str += uint64(len(s) + 1)
	}
	l.AS.ArgEnd = str
	l.AS.EnvStart = str
	envpp := make([]uint64, len(envp)+1)
	for i, s := range envp {
		if err := l.AS.WriteAt(str, append([]byte(s), 0)); err != nil {
			return 0, err
		}
		envpp[i] = str
		str += uint64(len(s) + 1)
	}
	l.AS.EnvEnd = str
	execfnVA := str
	if err := l.AS.WriteAt(str, append([]byte(canon), 0)); err != nil {
		return 0, err
	}

	// AT_RANDOM is where a guest libc gets its stack canary and pointer guard
	// from, so these sixteen bytes have to be unpredictable: a fixed pattern
	// would hand every guest the same canary and make the guard no guard.
	rnd := make([]byte, 16)
	if _, err := rand.Read(rnd); err != nil {
		return 0, abi.EIO
	}
	sp, rndVA := pushBlock(l.AS, sp, rnd)
	sp, platVA := pushBlock(l.AS, sp, []byte("aarch64\x00"))

	auxv := []uint64{
		AT_PHDR, exe.PhdrVA,
		AT_PHENT, 56,
		AT_PHNUM, uint64(exe.PhNum),
		AT_PAGESZ, mem.PageSize,
		AT_BASE, atBase,
		AT_FLAGS, 0,
		AT_ENTRY, exe.Entry,
		AT_UID, uint64(os.Getuid()),
		AT_EUID, uint64(os.Geteuid()),
		AT_GID, uint64(os.Getgid()),
		AT_EGID, uint64(os.Getegid()),
		AT_SECURE, 0,
		AT_RANDOM, rndVA,
		AT_HWCAP, core.Features.HWCap(),
		AT_HWCAP2, core.Features.HWCap2(),
		AT_CLKTCK, 100,
		AT_PLATFORM, platVA,
		AT_EXECFN, execfnVA,
		AT_MINSIGSTKSZ, 5120,
		AT_NULL, 0,
	}

	// Vector area: argc, argv[], NULL, envp[], NULL, auxv. Keep SP 16-aligned.
	vecBytes := 8*(1+uint64(len(argv))+1+uint64(len(envp))+1) + uint64(len(auxv))*8
	sp &= ^uint64(15)
	if vecBytes&15 != 0 {
		sp -= 16 - (vecBytes & 15)
	}
	sp -= vecBytes
	// start_stack is the address of argc — the SP the image starts on.
	// load_elf_binary assigns it after create_elf_tables has lowered it to the
	// vector area, so it sits below the string block rather than at it.
	l.AS.StartStack = sp

	va := sp
	buf := make([]byte, 8)
	putU64 := func(v uint64) error {
		binary.LittleEndian.PutUint64(buf, v)
		return l.AS.WriteAt(va, buf[:8])
	}
	if err := putU64(uint64(len(argv))); err != nil {
		return 0, err
	}
	va += 8
	for _, v := range argvp {
		if err := putU64(v); err != nil {
			return 0, err
		}
		va += 8
	}
	for _, v := range envpp {
		if err := putU64(v); err != nil {
			return 0, err
		}
		va += 8
	}
	abuf := make([]byte, len(auxv)*8)
	for i, v := range auxv {
		binary.LittleEndian.PutUint64(abuf[i*8:], v)
	}
	if err := l.AS.WriteAt(va, abuf); err != nil {
		return 0, err
	}
	return sp, nil
}

// pushBlock pushes a block onto the guest stack (which grows down), keeping it
// 8-byte aligned.
func pushBlock(as *mem.AddrSpace, sp uint64, data []byte) (uint64, uint64) {
	sp = (sp - uint64(len(data))) &^ 7
	if err := as.WriteAt(sp, data); err != nil {
		return sp, 0
	}
	return sp, sp
}

func stackSize(limit uint64) uint64 {
	// The stack the new image gets is the guest's RLIMIT_STACK: a kernel's
	// stack VMA grows on demand and can never pass that limit, so the limit
	// *is* the size of the stack the program ends up with.
	if limit == 0 || limit == ^uint64(0) {
		return StkLim
	}
	if limit > StackMax {
		return StackMax
	}
	return pgUp(limit)
}

func pfToProt(pf uint32) uint32 {
	var p uint32
	if pf&PF_R != 0 {
		p |= mem.PTER
	}
	if pf&PF_W != 0 {
		p |= mem.PTEW
	}
	if pf&PF_X != 0 {
		p |= mem.PTEX
	}
	return p
}

func pgUp(v uint64) uint64   { return (v + mem.PageMask) &^ uint64(mem.PageMask) }
func pgDown(v uint64) uint64 { return v &^ uint64(mem.PageMask) }

func errnoFrom(err error) error {
	if os.IsNotExist(err) {
		return abi.ENOENT
	}
	if os.IsPermission(err) {
		return abi.EACCES
	}
	return abi.EIO
}

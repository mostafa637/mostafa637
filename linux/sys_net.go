// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Ported to Go from src/sys_net.c — the socket syscalls.
//
// Two things make this more than a pass-through:
//
//  1. AF_UNIX pathname sockets carry a filesystem path, and the guest's view of
//     the filesystem is the rootfs. The path is resolved through the same
//     resolver every other syscall uses, so bind()/connect() reach the guest's
//     socket and not the host's, and what the kernel reports back is stripped
//     of the rootfs prefix again.
//  2. Abstract AF_UNIX sockets (leading NUL) live in one host-wide namespace
//     that an unprivileged emulator cannot partition. Each rootfs gets its own
//     view of it by splicing a tag derived from the rootfs path in after the
//     leading NUL: same rootfs rendezvous, different ones — and the host —
//     do not.
//
// Guest fd == host fd, so SCM_RIGHTS descriptors need no translation of their
// own; only SCM_CREDENTIALS carries an identity that has to be remapped.

package linux

import (
	"crypto/sha256"
	"encoding/binary"
	"path"
	"strings"
	"syscall"
	"unsafe"

	"github.com/mostafa637/mostafa637/abi"
	"github.com/mostafa637/mostafa637/mem"
)

func init() {
	Register(map[uint64]Handler{
		Syssocket:      sysSocket,
		Syssocketpair:  sysSocketPair,
		Sysbind:        sysBind,
		Sysconnect:     sysConnect,
		Syslisten:      sysListen,
		Sysaccept:      sysAccept,
		Sysaccept4:     sysAccept4,
		Sysgetsockname: sysGetSockName,
		Sysgetpeername: sysGetPeerName,
		Syssendto:      sysSendTo,
		Sysrecvfrom:    sysRecvFrom,
		Sysshutdown:    sysShutdown,
		Syssetsockopt:  sysSetSockOpt,
		Sysgetsockopt:  sysGetSockOpt,
		Syssendmsg:     sysSendMsg,
		Sysrecvmsg:     sysRecvMsg,
	})
}

// Layout constants. struct sockaddr_storage is 128 bytes everywhere, and
// struct sockaddr_un puts sun_path two bytes in (sa_family_t is a u16).
const (
	sockStorageSize = 128
	sunPathOff      = 2
	sunPathLen      = 108
	afUNIX          = 1
	// struct msghdr / struct cmsghdr are the same shape on aarch64 and on
	// every 64-bit host, but they are marshalled field by field anyway so a
	// 32-bit host build does not silently misread the guest's.
	msgHdrSize = 56
	cmsgHdrLen = 16
)

// ---- addresses ------------------------------------------------------------

// absTag is this rootfs's tag in the abstract socket namespace: a short,
// printable digest of the rootfs path. Two emulators aimed at the same rootfs
// compute the same tag and can therefore still rendezvous.
func (t *Task) absTag() string {
	sum := sha256.Sum256([]byte(t.Rootfs))
	return ".a64-" + hex8(sum[:])
}

func hex8(b []byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, 8)
	for i := range out {
		out[i] = digits[b[i]&0xf]
	}
	return string(out)
}

// sockaddrIn reads a guest sockaddr and rewrites it for the host.
//
// It returns the bytes to hand to the host call and, when the rewritten host
// path did not fit in sun_path, the descriptor of the socket's parent
// directory, which the caller must close once the call has run (the address
// names the socket through /proc/self/fd/<dirfd>/<base>, which the kernel
// resolves to the directory the walk found — so no rename in between can
// redirect the bind or connect).
func sockaddrIn(t *Task, va uint64, socklen uint32, follow bool) (buf []byte, outLen uint32, dirfd int, err error) {
	dirfd = -1
	// move_addr_to_kernel refuses a length it cannot use rather than trimming
	// it: negative, or past sockaddr_storage, is EINVAL whatever the address
	// itself says.
	if int32(socklen) < 0 || socklen > sockStorageSize {
		return nil, 0, -1, abi.EINVAL
	}
	buf = make([]byte, sockStorageSize)
	if socklen > 0 {
		if err := mem.CopyFromGuest(t.CPU, buf[:socklen], va); err != nil {
			return nil, 0, -1, abi.EFAULT
		}
	}
	outLen = socklen
	if err := unixPathIn(t, buf, &outLen, follow, &dirfd); err != nil {
		return nil, 0, -1, err
	}
	return buf, outLen, dirfd, nil
}

// unixPathIn rewrites an AF_UNIX address for the host: abstract names get the
// rootfs tag spliced in, pathnames are resolved through the rootfs.
func unixPathIn(t *Task, ss []byte, sl *uint32, follow bool, dirfdOut *int) error {
	if int(*sl) < 2 {
		return nil // no family to look at
	}
	if binary.LittleEndian.Uint16(ss) != afUNIX {
		return nil
	}
	if int(*sl) <= sunPathOff {
		return nil // unnamed / autobind
	}
	tag := t.absTag()
	p := ss[sunPathOff:]
	if p[0] == 0 { // abstract namespace
		total := int(*sl) - sunPathOff
		if total > sunPathLen {
			return nil // the kernel's EINVAL for an over-long address
		}
		if total+len(tag) > sunPathLen {
			// Refused rather than passed through untagged: untagged IS the
			// host's own namespace, so letting it through gave a guest a
			// deliberate way out of the isolation.
			return abi.ENAMETOOLONG
		}
		copy(p[1+len(tag):], p[1:total])
		copy(p[1:], tag)
		*sl += uint32(len(tag))
		return nil
	}
	maxp := int(*sl) - sunPathOff
	if maxp > sunPathLen {
		maxp = sunPathLen
	}
	end := 0
	for end < maxp && p[end] != 0 {
		end++
	}
	guest := string(p[:end])
	if guest == "" {
		return nil
	}
	host, err := t.Resolve(guest)
	if err != nil {
		return err
	}
	if len(host)+1 <= sunPathLen {
		copy(ss[sunPathOff:], host)
		ss[sunPathOff+len(host)] = 0
		*sl = uint32(sunPathOff + len(host) + 1)
		return nil
	}
	// Too long for sun_path: name it through a descriptor for its parent
	// directory, so only the basename has to fit.
	d, err := syscall.Open(path.Dir(host), hostOPath|syscall.O_DIRECTORY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return abi.ENAMETOOLONG
	}
	proc := "/proc/self/fd/" + itoa(d) + "/" + path.Base(host)
	if len(proc)+1 > sunPathLen {
		syscall.Close(d)
		return abi.ENAMETOOLONG
	}
	copy(ss[sunPathOff:], proc)
	ss[sunPathOff+len(proc)] = 0
	*sl = uint32(sunPathOff + len(proc) + 1)
	*dirfdOut = d
	return nil
}

// unixPathOut is the reverse: a host address the kernel reported is rewritten
// to the guest's view before the guest sees it.
func unixPathOut(t *Task, ss []byte, sl *uint32) {
	if int(*sl) < 2 || binary.LittleEndian.Uint16(ss) != afUNIX {
		return
	}
	if int(*sl) <= sunPathOff {
		return
	}
	p := ss[sunPathOff:]
	tag := t.absTag()
	if p[0] == 0 { // abstract: strip our tag
		total := int(*sl) - sunPathOff
		if total < 1+len(tag) {
			return
		}
		if string(p[1:1+len(tag)]) != tag {
			return // not ours: a foreign abstract name, left alone
		}
		copy(p[1:], p[1+len(tag):total])
		*sl -= uint32(len(tag))
		return
	}
	maxp := int(*sl) - sunPathOff
	if maxp > sunPathLen {
		maxp = sunPathLen
	}
	end := 0
	for end < maxp && p[end] != 0 {
		end++
	}
	host := string(p[:end])
	if !strings.HasPrefix(host, t.Rootfs) {
		return
	}
	guest := path.Clean("/" + strings.TrimPrefix(host, t.Rootfs))
	if len(guest)+1 > sunPathLen {
		return
	}
	copy(ss[sunPathOff:], guest)
	ss[sunPathOff+len(guest)] = 0
	*sl = uint32(sunPathOff + len(guest) + 1)
}

// sockaddrOut writes a host address back to the guest, clamped to the buffer
// the guest offered (whose length is re-read from the guest: the kernel
// reports the address it has, not the room there was for it).
func sockaddrOut(t *Task, va, lenVA uint64, ss []byte, socklen uint32) uint64 {
	unixPathOut(t, ss, &socklen)
	var want uint32 = sockStorageSize
	if lenVA != 0 {
		b := make([]byte, 4)
		if err := mem.CopyFromGuest(t.CPU, b, lenVA); err != nil {
			return negErrno(abi.EFAULT)
		}
		want = binary.LittleEndian.Uint32(b)
	}
	n := want
	if n > socklen {
		n = socklen
	}
	if va != 0 && n > 0 {
		if err := mem.CopyToGuest(t.CPU, va, ss[:n]); err != nil {
			return negErrno(abi.EFAULT)
		}
	}
	if lenVA != 0 {
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, socklen)
		if err := mem.CopyToGuest(t.CPU, lenVA, b); err != nil {
			return negErrno(abi.EFAULT)
		}
	}
	return retOK(0)
}

// ---- the syscalls ---------------------------------------------------------

func sysSocket(t *Task, a [6]uint64) uint64 {
	fd, err := hostSocket(uintptr(int32(a[0])), uintptr(int32(a[1])), uintptr(int32(a[2])))
	if err != nil {
		return retErr(err)
	}
	return retOK(uint64(fd))
}

func sysSocketPair(t *Task, a [6]uint64) uint64 {
	var sv [2]int32
	if err := hostSocketpair(uintptr(int32(a[0])), uintptr(int32(a[1])), uintptr(int32(a[2])), &sv); err != nil {
		return retErr(err)
	}
	b := make([]byte, 8)
	binary.LittleEndian.PutUint32(b, uint32(sv[0]))
	binary.LittleEndian.PutUint32(b[4:], uint32(sv[1]))
	if err := mem.CopyToGuest(t.CPU, a[3], b); err != nil {
		// The guest never learns the two numbers, so nothing it does can ever
		// close them: close them here, as the kernel does on this path.
		syscall.Close(int(sv[0]))
		syscall.Close(int(sv[1]))
		return retErr(err)
	}
	return retOK(0)
}

func sysBind(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	ss, sl, dirfd, err := sockaddrIn(t, a[1], uint32(a[2]), false)
	if err != nil {
		return retErr(err)
	}
	r := hostBind(uintptr(fd), unsafe.Pointer(&ss[0]), uintptr(sl))
	if dirfd >= 0 {
		syscall.Close(dirfd)
	}
	return retErr(r)
}

func sysConnect(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	ss, sl, dirfd, err := sockaddrIn(t, a[1], uint32(a[2]), true)
	if err != nil {
		return retErr(err)
	}
	r := hostConnect(uintptr(fd), unsafe.Pointer(&ss[0]), uintptr(sl))
	if dirfd >= 0 {
		syscall.Close(dirfd)
	}
	return retErr(r)
}

func sysListen(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	return retErr(hostListen(uintptr(fd), int(int32(a[1]))))
}

func sysShutdown(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	return retErr(hostShutdown(uintptr(fd), uintptr(int32(a[1]))))
}

func sysAccept4(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	ss := make([]byte, sockStorageSize)
	var sl uint32 = sockStorageSize
	nfd, err := hostAccept4(uintptr(fd), unsafe.Pointer(&ss[0]), &sl, int(int32(a[3])))
	if err != nil {
		return retErr(err)
	}
	return sockaddrOut(t, a[1], a[2], ss, sl) | retOK(uint64(nfd))
}

func sysAccept(t *Task, a [6]uint64) uint64 {
	a[3] = 0
	return sysAccept4(t, a)
}

func sysGetSockName(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	ss := make([]byte, sockStorageSize)
	var sl uint32 = sockStorageSize
	if err := hostGetsockname(uintptr(fd), unsafe.Pointer(&ss[0]), &sl); err != nil {
		return retErr(err)
	}
	return sockaddrOut(t, a[1], a[2], ss, sl)
}

func sysGetPeerName(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	ss := make([]byte, sockStorageSize)
	var sl uint32 = sockStorageSize
	if err := hostGetpeername(uintptr(fd), unsafe.Pointer(&ss[0]), &sl); err != nil {
		return retErr(err)
	}
	return sockaddrOut(t, a[1], a[2], ss, sl)
}

func sysSendTo(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	buf := make([]byte, a[2])
	if err := mem.CopyFromGuest(t.CPU, buf, a[1]); err != nil {
		return retErr(err)
	}
	var addr unsafe.Pointer
	var sl uintptr
	var dirfd = -1
	if a[4] != 0 {
		ss, n, d, err := sockaddrIn(t, a[4], uint32(a[5]), true)
		if err != nil {
			return retErr(err)
		}
		addr, sl, dirfd = unsafe.Pointer(&ss[0]), uintptr(n), d
	}
	n, err := hostSendto(uintptr(fd), buf, int(int32(a[3])), addr, sl)
	if dirfd >= 0 {
		syscall.Close(dirfd)
	}
	if n >= 0 && err != nil {
		err = nil // a short send is not an error
	}
	if n < 0 {
		return retErr(err)
	}
	return retOK(uint64(n))
}

func sysRecvFrom(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	buf := make([]byte, a[2])
	ss := make([]byte, sockStorageSize)
	var sl uint32 = sockStorageSize
	var addr unsafe.Pointer
	var slp *uint32
	if a[4] != 0 {
		addr, slp = unsafe.Pointer(&ss[0]), &sl
	}
	n, err := hostRecvfrom(uintptr(fd), buf, int(int32(a[3])), addr, slp)
	if n < 0 {
		return retErr(err)
	}
	if n > 0 {
		if r := mem.CopyToGuest(t.CPU, a[1], buf[:n]); r != nil {
			return retErr(r)
		}
	}
	if a[4] != 0 {
		if r := sockaddrOut(t, a[4], a[5], ss, sl); isErr(r) {
			return r
		}
	}
	return retOK(uint64(n))
}

// ---- options --------------------------------------------------------------

// The SO_ATTACH_* options carry a BPF program: the C emulator translates the
// guest's classic BPF into a host filter (src/sys_net.c sockopt_attach_fprog)
// so that a guest's seccomp-style filters actually run. That translation is
// not ported; the option is refused rather than passed through, because a
// guest filter that silently did nothing would be worse than EINVAL.
const (
	soAttachFilter        = 26
	soAttachBPF           = 50
	soAttachReuseportCBPF = 51
	soAttachReuseportEBPF = 52
	soPeerCred            = 17
	soPassCred            = 16
	solSocket             = 1
	scmCredentials        = 2
	scmRights             = 1
)

func sysSetSockOpt(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	level, opt := int(int32(a[1])), int(int32(a[2]))
	switch {
	case level == solSocket && (opt == soAttachFilter || opt == soAttachBPF ||
		opt == soAttachReuseportCBPF || opt == soAttachReuseportEBPF):
		t.warnOnce("setsockopt(SO_ATTACH_*) is not ported (BPF program translation)")
		return negErrno(abi.ENOSYS)
	}
	val := make([]byte, a[4])
	if a[4] > 0 {
		if err := mem.CopyFromGuest(t.CPU, val, a[3]); err != nil {
			return retErr(err)
		}
	}
	var p unsafe.Pointer
	if len(val) > 0 {
		p = unsafe.Pointer(&val[0])
	}
	return retErr(hostSetsockopt(uintptr(fd), level, opt, p, uintptr(len(val))))
}

func sysGetSockOpt(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	level, opt := int(int32(a[1])), int(int32(a[2]))
	var sl uint32 = 128
	if a[4] != 0 {
		b := make([]byte, 4)
		if err := mem.CopyFromGuest(t.CPU, b, a[4]); err != nil {
			return retErr(err)
		}
		sl = binary.LittleEndian.Uint32(b)
	}
	val := make([]byte, sl)
	var p unsafe.Pointer
	if len(val) > 0 {
		p = unsafe.Pointer(&val[0])
	}
	out := sl
	if err := hostGetsockopt(uintptr(fd), level, opt, p, &out); err != nil {
		return retErr(err)
	}
	// SO_PEERCRED reports the peer's {pid,uid,gid}: the uid and gid the guest
	// sees are the ones it believes it has, not the host's.
	if level == solSocket && opt == soPeerCred && out >= 12 {
		uid := binary.LittleEndian.Uint32(val[4:])
		gid := binary.LittleEndian.Uint32(val[8:])
		uid, gid = t.hostToGuestID(uid, gid)
		binary.LittleEndian.PutUint32(val[4:], uid)
		binary.LittleEndian.PutUint32(val[8:], gid)
	}
	if out > 0 && a[3] != 0 {
		if err := mem.CopyToGuest(t.CPU, a[3], val[:out]); err != nil {
			return retErr(err)
		}
	}
	if a[4] != 0 {
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, out)
		if err := mem.CopyToGuest(t.CPU, a[4], b); err != nil {
			return retErr(err)
		}
	}
	return retOK(0)
}

// ---- sendmsg / recvmsg ----------------------------------------------------

// guestMsgHdr is the guest's struct msghdr (LP64): the same shape as the
// host's on any 64-bit build, marshalled explicitly so it is also right on a
// 32-bit one.
type guestMsgHdr struct {
	name       uint64
	namelen    uint32
	iov        uint64
	iovlen     uint64
	control    uint64
	controllen uint64
	flags      int32
}

const maxIOV = 1024

func (t *Task) readMsgHdr(va uint64) (*guestMsgHdr, error) {
	b := make([]byte, msgHdrSize)
	if err := mem.CopyFromGuest(t.CPU, b, va); err != nil {
		return nil, abi.EFAULT
	}
	le := binary.LittleEndian
	g := &guestMsgHdr{
		name:       le.Uint64(b[0:]),
		namelen:    le.Uint32(b[8:]),
		iov:        le.Uint64(b[16:]),
		iovlen:     le.Uint64(b[24:]),
		control:    le.Uint64(b[32:]),
		controllen: le.Uint64(b[40:]),
		flags:      int32(le.Uint32(b[48:])),
	}
	// The kernel settles the name length before it looks at the vector: NULL
	// msg_name means no address, a negative length is EINVAL, and an over-long
	// one is clamped to sockaddr_storage (not refused — this differs from an
	// addrlen passed as its own argument).
	if g.name == 0 {
		g.namelen = 0
	} else {
		if int32(g.namelen) < 0 {
			return nil, abi.EINVAL
		}
		if g.namelen > sockStorageSize {
			g.namelen = sockStorageSize
		}
	}
	return g, nil
}

// hostMsgHdr builds the host struct msghdr for a guest one, bouncing the
// iovec data through host memory (a guest iovec is a guest address the host
// call cannot follow). For a receive the segments are empty and the guest
// bases come back in bases, so the data can be copied in afterwards.
func (t *Task) hostMsgHdr(g *guestMsgHdr, forSend bool) (msg []byte, bases, lens []uint64, data, ctrl []byte, err error) {
	if g.iovlen > maxIOV {
		return nil, nil, nil, nil, nil, abi.EMSGSIZE
	}
	cnt := int(g.iovlen)
	gi := make([]byte, 16*cnt)
	if cnt > 0 {
		if err := mem.CopyFromGuest(t.CPU, gi, g.iov); err != nil {
			return nil, nil, nil, nil, nil, abi.EFAULT
		}
	}
	le := binary.LittleEndian
	bases = make([]uint64, cnt)
	lens = make([]uint64, cnt)
	iov := make([]byte, 16*cnt)
	total := uint64(0)
	for i := 0; i < cnt; i++ {
		base := le.Uint64(gi[i*16:])
		l := le.Uint64(gi[i*16+8:])
		if int64(l) < 0 {
			return nil, nil, nil, nil, nil, abi.EINVAL
		}
		bases[i] = base
		lens[i] = l
		le.PutUint64(iov[i*16+8:], l)
		total += l
	}
	if total > 1<<62 {
		return nil, nil, nil, nil, nil, abi.EINVAL
	}
	data = make([]byte, total)
	off := uint64(0)
	for i := 0; i < cnt; i++ {
		l := le.Uint64(iov[i*16+8:])
		// The host vector points into one bounce buffer, so its bases have to
		// be written after it is allocated — and after every append that could
		// move it, which is why this loop runs before the syscalls and not
		// inside one.
		le.PutUint64(iov[i*16:], uint64(uintptr(unsafe.Pointer(&data[0])))+off)
		off += l
	}
	if forSend {
		off = 0
		for i := 0; i < cnt; i++ {
			l := le.Uint64(iov[i*16+8:])
			if l > 0 {
				if err := mem.CopyFromGuest(t.CPU, data[off:off+l], bases[i]); err != nil {
					return nil, nil, nil, nil, nil, abi.EFAULT
				}
			}
			off += l
		}
	} else if total == 0 {
		data = []byte{}
	}
	// Ancillary data.
	if g.controllen > 0 {
		ctrl = make([]byte, g.controllen)
		if forSend {
			if err := mem.CopyFromGuest(t.CPU, ctrl, g.control); err != nil {
				return nil, nil, nil, nil, nil, abi.EFAULT
			}
			if err := t.ctrlGuestToHost(ctrl); err != nil {
				return nil, nil, nil, nil, nil, err
			}
		}
	}
	msg = make([]byte, msgHdrSize)
	le.PutUint64(msg[0:], 0) // msg_name: set below
	le.PutUint32(msg[8:], 0)
	le.PutUint64(msg[16:], uint64(uintptr(unsafe.Pointer(&iov[0]))))
	le.PutUint64(msg[24:], uint64(cnt))
	le.PutUint64(msg[32:], 0)
	le.PutUint64(msg[40:], uint64(len(ctrl)))
	le.PutUint32(msg[48:], uint32(g.flags))
	return msg, bases, lens, data, ctrl, nil
}

// ctrlGuestToHost rewrites an SCM_CREDENTIALS element on the way out: the
// kernel checks the uid/gid in it against the sender's real credentials, so a
// guest that believes it is root has to send the host identity it stands for.
func (t *Task) ctrlGuestToHost(buf []byte) error {
	le := binary.LittleEndian
	for off := 0; off+cmsgHdrLen <= len(buf); {
		l := int(le.Uint64(buf[off:]))
		if l < cmsgHdrLen || off+l > len(buf) {
			return abi.EINVAL // the kernel's own CMSG_OK test
		}
		level := int32(le.Uint32(buf[off+8:]))
		typ := int32(le.Uint32(buf[off+12:]))
		if level == solSocket && typ == scmCredentials && l >= cmsgHdrLen+12 {
			uid := le.Uint32(buf[off+cmsgHdrLen+4:])
			gid := le.Uint32(buf[off+cmsgHdrLen+8:])
			uid, gid = t.guestToHostID(uid, gid)
			le.PutUint32(buf[off+cmsgHdrLen+4:], uid)
			le.PutUint32(buf[off+cmsgHdrLen+8:], gid)
		}
		step := (l + 7) &^ 7
		if step == 0 {
			break
		}
		off += step
	}
	return nil
}

// ctrlHostToGuest is the reverse, for the ancillary data a receive brings
// back: SCM_CREDENTIALS arrives with the host peer's identity in it.
func (t *Task) ctrlHostToGuest(buf []byte, used uint64) {
	le := binary.LittleEndian
	n := int(used)
	if n > len(buf) {
		n = len(buf)
	}
	for off := 0; off+cmsgHdrLen <= n; {
		l := int(le.Uint64(buf[off:]))
		if l < cmsgHdrLen || off+l > n {
			return
		}
		level := int32(le.Uint32(buf[off+8:]))
		typ := int32(le.Uint32(buf[off+12:]))
		if level == solSocket && typ == scmCredentials && l >= cmsgHdrLen+12 {
			uid := le.Uint32(buf[off+cmsgHdrLen+4:])
			gid := le.Uint32(buf[off+cmsgHdrLen+8:])
			uid, gid = t.hostToGuestID(uid, gid)
			le.PutUint32(buf[off+cmsgHdrLen+4:], uid)
			le.PutUint32(buf[off+cmsgHdrLen+8:], gid)
		}
		step := (l + 7) &^ 7
		if step == 0 {
			return
		}
		off += step
	}
}

func sysSendMsg(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	g, err := t.readMsgHdr(a[1])
	if err != nil {
		return retErr(err)
	}
	msg, _, _, data, ctrl, err := t.hostMsgHdr(g, true)
	if err != nil {
		return retErr(err)
	}
	le := binary.LittleEndian
	var dirfd = -1
	if g.name != 0 && g.namelen > 0 {
		ss, n, d, rerr := sockaddrIn(t, g.name, g.namelen, true)
		if rerr != nil {
			return retErr(rerr)
		}
		dirfd = d
		le.PutUint64(msg[0:], uint64(uintptr(unsafe.Pointer(&ss[0]))))
		le.PutUint32(msg[8:], n)
	}
	if len(ctrl) > 0 {
		le.PutUint64(msg[32:], uint64(uintptr(unsafe.Pointer(&ctrl[0]))))
	}
	// `data` is the bounce buffer the host vector points into; it has to stay
	// reachable (and un-garbage-collected) until the call has run.
	_ = data
	n, rerr := hostSendmsg(uintptr(fd), unsafe.Pointer(&msg[0]), int(int32(a[2])))
	if dirfd >= 0 {
		syscall.Close(dirfd)
	}
	if n < 0 {
		return retErr(rerr)
	}
	return retOK(uint64(n))
}

func sysRecvMsg(t *Task, a [6]uint64) uint64 {
	fd := int(int32(a[0]))
	if !t.GuestFD(fd) {
		return negErrno(abi.EBADF)
	}
	g, err := t.readMsgHdr(a[1])
	if err != nil {
		return retErr(err)
	}
	msg, bases, lens, data, ctrl, err := t.hostMsgHdr(g, false)
	if err != nil {
		return retErr(err)
	}
	le := binary.LittleEndian
	ss := make([]byte, sockStorageSize)
	if g.name != 0 {
		// Give the kernel the whole staging buffer, so the source address
		// arrives untruncated: unixPathOut needs the complete sun_path to
		// translate it back.
		le.PutUint64(msg[0:], uint64(uintptr(unsafe.Pointer(&ss[0]))))
		le.PutUint32(msg[8:], sockStorageSize)
	}
	if len(ctrl) > 0 {
		le.PutUint64(msg[32:], uint64(uintptr(unsafe.Pointer(&ctrl[0]))))
	}
	n, rerr := hostRecvmsg(uintptr(fd), unsafe.Pointer(&msg[0]), int(int32(a[2])))
	if n < 0 {
		return retErr(rerr)
	}
	// Write the data back into the guest's own segments, in order, up to what
	// the host call actually received (bases and lengths come from the single
	// reading of the guest's iovec array: a sibling thread sharing the address
	// space may have rewritten it while the call was parked).
	off := uint64(0)
	for i := range bases {
		if off >= uint64(n) {
			break
		}
		take := lens[i]
		if off+take > uint64(n) {
			take = uint64(n) - off
		}
		if take > 0 {
			if err := mem.CopyToGuest(t.CPU, bases[i], data[off:off+take]); err != nil {
				return retErr(err)
			}
		}
		off += take
	}
	if g.name != 0 {
		sl := le.Uint32(msg[8:])
		if r := sockaddrOut(t, g.name, a[1]+8, ss, sl); isErr(r) {
			return r
		}
	}
	if len(ctrl) > 0 {
		t.ctrlHostToGuest(ctrl, le.Uint64(msg[40:]))
		if err := mem.CopyToGuest(t.CPU, g.control, ctrl); err != nil {
			return retErr(err)
		}
	}
	// msg_flags and the clamped control length go back into the guest header.
	hdr := make([]byte, msgHdrSize)
	if err := mem.CopyFromGuest(t.CPU, hdr, a[1]); err != nil {
		return retErr(err)
	}
	le.PutUint32(hdr[48:], le.Uint32(msg[48:]))
	le.PutUint64(hdr[40:], le.Uint64(msg[40:]))
	if err := mem.CopyToGuest(t.CPU, a[1], hdr); err != nil {
		return retErr(err)
	}
	return retOK(uint64(n))
}

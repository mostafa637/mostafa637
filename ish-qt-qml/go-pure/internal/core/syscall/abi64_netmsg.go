package syscall

import (
	"encoding/binary"
	"net"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

const (
	socketSOLSocket64  = 1
	socketIPProtoTCP64 = 6

	socketSOError64      = 4
	socketSOType64       = 3
	socketSODomain64     = 39
	socketSOProtocol64   = 38
	socketSOAcceptConn64 = 30
	socketSOReuseAddr64  = 2
	socketSOReusePort64  = 15
	socketSOKeepAlive64  = 9
	socketSOBroadcast64  = 6
	socketSOSndBuf64     = 7
	socketSORcvBuf64     = 8
	socketTCPNoDelay64   = 1

	msgDontWait64    = 0x40
	msgPeek64        = 0x2
	msgNoSignal64    = 0x4000
	msgMore64        = 0x8000
	msgCmsgCloexec64 = 0x40000000

	msghdr64Size          = 56
	msghdr64NameOffset    = 0
	msghdr64NameLenOff    = 8
	msghdr64IOVOffset     = 16
	msghdr64IOVLenOff     = 24
	msghdr64ControlOff    = 32
	msghdr64ControlLenOff = 40
	msghdr64FlagsOffset   = 48
)

type socketOptionKey64 struct {
	level uint64
	name  uint64
}

type guestMsgHdr64 struct {
	name       corecpu.Address64
	nameLength uint32
	iov        corecpu.Address64
	iovLength  uint64
	control    corecpu.Address64
	controlLen uint64
	flags      uint32
}

func readGuestMsgHdr64(ctx *Context64, address corecpu.Address64) (guestMsgHdr64, int64) {
	if ctx == nil || ctx.Memory == nil || address == 0 {
		return guestMsgHdr64{}, int64(EFAULT)
	}
	raw := make([]byte, msghdr64Size)
	if err := ctx.Memory.Read(address, raw); err != nil {
		return guestMsgHdr64{}, int64(EFAULT)
	}
	return guestMsgHdr64{
		name:       corecpu.Address64(binary.LittleEndian.Uint64(raw[msghdr64NameOffset : msghdr64NameOffset+8])),
		nameLength: binary.LittleEndian.Uint32(raw[msghdr64NameLenOff : msghdr64NameLenOff+4]),
		iov:        corecpu.Address64(binary.LittleEndian.Uint64(raw[msghdr64IOVOffset : msghdr64IOVOffset+8])),
		iovLength:  binary.LittleEndian.Uint64(raw[msghdr64IOVLenOff : msghdr64IOVLenOff+8]),
		control:    corecpu.Address64(binary.LittleEndian.Uint64(raw[msghdr64ControlOff : msghdr64ControlOff+8])),
		controlLen: binary.LittleEndian.Uint64(raw[msghdr64ControlLenOff : msghdr64ControlLenOff+8]),
		flags:      binary.LittleEndian.Uint32(raw[msghdr64FlagsOffset : msghdr64FlagsOffset+4]),
	}, 0
}

func validateMsgHdr64(ctx *Context64, header guestMsgHdr64, sending bool) int64 {
	if header.name == 0 && header.nameLength != 0 {
		return int64(EFAULT)
	}
	if header.name != 0 && header.nameLength == 0 && sending {
		return int64(EINVAL)
	}
	if header.control != 0 || header.controlLen != 0 {
		return int64(EOPNOTSUPP)
	}
	if header.iov == 0 && header.iovLength != 0 {
		return int64(EFAULT)
	}
	if header.iovLength > maxIOVecs64 {
		return int64(EINVAL)
	}
	if header.flags != 0 {
		return int64(EINVAL)
	}
	if !sending {
		return 0
	}
	if header.name != 0 {
		if _, result := parseSockaddr64(ctx, header.name, uint64(header.nameLength)); result != 0 {
			return int64(result)
		}
	}
	return 0
}

func sendmsg64(ctx *Context64, args [6]uint64) int64 {
	file, result := context64File(ctx, args[0])
	if result != 0 {
		return result
	}
	handle, ok := socketHandleFromFile64(file)
	if !ok || file.Writer == nil {
		return int64(ENOTSOCK)
	}
	if handle.conn == nil {
		return int64(ENOTCONN)
	}
	header, result := readGuestMsgHdr64(ctx, corecpu.Address64(args[1]))
	if result != 0 {
		return result
	}
	if args[2]&^(msgNoSignal64|msgMore64) != 0 {
		return int64(EOPNOTSUPP)
	}
	if result = validateMsgHdr64(ctx, header, true); result != 0 {
		return result
	}
	return writev64(ctx, [6]uint64{args[0], uint64(header.iov), header.iovLength})
}

func recvmsg64(ctx *Context64, args [6]uint64) int64 {
	file, result := context64File(ctx, args[0])
	if result != 0 {
		return result
	}
	handle, ok := socketHandleFromFile64(file)
	if !ok || file.Reader == nil {
		return int64(ENOTSOCK)
	}
	if handle.conn == nil {
		return int64(ENOTCONN)
	}
	header, result := readGuestMsgHdr64(ctx, corecpu.Address64(args[1]))
	if result != 0 {
		return result
	}
	if args[2]&^uint64(msgCmsgCloexec64) != 0 || args[2]&msgPeek64 != 0 {
		return int64(EOPNOTSUPP)
	}
	if result = validateMsgHdr64(ctx, header, false); result != 0 {
		return result
	}
	result = readv64(ctx, [6]uint64{args[0], uint64(header.iov), header.iovLength})
	if result < 0 {
		return result
	}
	if ctx.Memory == nil {
		return int64(EFAULT)
	}
	var raw [4]byte
	binary.LittleEndian.PutUint32(raw[:], 0)
	if err := ctx.Memory.Write(corecpu.Address64(args[1]+msghdr64NameLenOff), raw[:]); err != nil {
		return int64(EFAULT)
	}
	if err := ctx.Memory.Write(corecpu.Address64(args[1]+msghdr64FlagsOffset), raw[:]); err != nil {
		return int64(EFAULT)
	}
	return result
}

func getsockopt64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || args[1] == 0 || args[2] == 0 || args[3] == 0 || args[4] == 0 {
		return int64(EFAULT)
	}
	file, err := ctx.GetFile(args[0])
	if err != nil {
		return int64(EBADF)
	}
	handle, ok := socketHandleFromFile64(file)
	if !ok {
		return int64(ENOTSOCK)
	}
	length, result := readSocketOptLength64(ctx, args[4])
	if result != 0 {
		return result
	}
	if length < 4 {
		return int64(EINVAL)
	}
	value, supported := socketOptionValue64(handle, args[1], args[2])
	if !supported {
		return int64(EOPNOTSUPP)
	}
	var raw [4]byte
	binary.LittleEndian.PutUint32(raw[:], uint32(value))
	if err := ctx.Memory.Write(corecpu.Address64(args[3]), raw[:]); err != nil {
		return int64(EFAULT)
	}
	binary.LittleEndian.PutUint32(raw[:], 4)
	if err := ctx.Memory.Write(corecpu.Address64(args[4]), raw[:]); err != nil {
		return int64(EFAULT)
	}
	return 0
}

func setsockopt64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil {
		return int64(EFAULT)
	}
	if args[3] == 0 {
		return int64(EFAULT)
	}
	if args[4] < 4 {
		return int64(EINVAL)
	}
	file, err := ctx.GetFile(args[0])
	if err != nil {
		return int64(EBADF)
	}
	handle, ok := socketHandleFromFile64(file)
	if !ok {
		return int64(ENOTSOCK)
	}
	if args[4] != 4 {
		return int64(EINVAL)
	}
	var raw [4]byte
	if err := ctx.Memory.Read(corecpu.Address64(args[3]), raw[:]); err != nil {
		return int64(EFAULT)
	}
	value := int32(binary.LittleEndian.Uint32(raw[:]))
	if !socketOptionSupported64(args[1], args[2]) {
		return int64(EOPNOTSUPP)
	}
	if args[1] == socketIPProtoTCP64 && args[2] == socketTCPNoDelay64 {
		if conn, ok := handle.conn.(*net.TCPConn); ok {
			if err := conn.SetNoDelay(value != 0); err != nil {
				return int64(EOPNOTSUPP)
			}
		}
	}
	if handle.options == nil {
		handle.options = make(map[socketOptionKey64]int32)
	}
	handle.options[socketOptionKey64{level: args[1], name: args[2]}] = value
	return 0
}

func readSocketOptLength64(ctx *Context64, address uint64) (uint32, int64) {
	var raw [4]byte
	if err := ctx.Memory.Read(corecpu.Address64(address), raw[:]); err != nil {
		return 0, int64(EFAULT)
	}
	return binary.LittleEndian.Uint32(raw[:]), 0
}

func socketOptionSupported64(level, name uint64) bool {
	if level == socketSOLSocket64 {
		switch name {
		case socketSOError64, socketSOType64, socketSODomain64, socketSOProtocol64, socketSOAcceptConn64, socketSOReuseAddr64, socketSOReusePort64, socketSOKeepAlive64, socketSOBroadcast64, socketSOSndBuf64, socketSORcvBuf64:
			return true
		}
	}
	return level == socketIPProtoTCP64 && name == socketTCPNoDelay64
}

func socketOptionValue64(handle *socketHandle64, level, name uint64) (int32, bool) {
	if !socketOptionSupported64(level, name) {
		return 0, false
	}
	switch {
	case level == socketSOLSocket64 && name == socketSOError64:
		return 0, true
	case level == socketSOLSocket64 && name == socketSOType64:
		return int32(handle.kind), true
	case level == socketSOLSocket64 && name == socketSODomain64:
		return int32(handle.family), true
	case level == socketSOLSocket64 && name == socketSOProtocol64:
		if handle.family == socketAFInet || handle.family == socketAFInet6 {
			return 6, true
		}
		return 0, true
	case level == socketSOLSocket64 && name == socketSOAcceptConn64:
		if handle.listener != nil {
			return 1, true
		}
		return 0, true
	}
	if handle.options != nil {
		if value, exists := handle.options[socketOptionKey64{level: level, name: name}]; exists {
			return value, true
		}
	}
	return 0, true
}

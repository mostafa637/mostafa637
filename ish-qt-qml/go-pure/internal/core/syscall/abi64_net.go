package syscall

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
)

const (
	socketAFUnix  = 1
	socketAFInet  = 2
	socketAFInet6 = 10

	socketStream = 1
	socketDgram  = 2

	socketCloexec  = 0x80000
	socketNonblock = 0x800
)

type socketAddr64 struct {
	family uint16
	port   uint16
	host   string
	path   string
}

type socketHandle64 struct {
	family   uint16
	kind     int
	network  string
	listener net.Listener
	conn     net.Conn
}

type bufferedPipe64 struct {
	mu      sync.Mutex
	cond    *sync.Cond
	buffer  bytes.Buffer
	closed  bool
	peerEOF bool
	peer    *bufferedPipe64
}

type bufferedPipeAddr64 string

func (a bufferedPipeAddr64) Network() string { return "socketpair" }
func (a bufferedPipeAddr64) String() string  { return string(a) }

func newBufferedPipe64() (*bufferedPipe64, *bufferedPipe64) {
	left := &bufferedPipe64{}
	right := &bufferedPipe64{}
	left.cond = sync.NewCond(&left.mu)
	right.cond = sync.NewCond(&right.mu)
	left.peer = right
	right.peer = left
	return left, right
}

func (p *bufferedPipe64) Read(dst []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for p.buffer.Len() == 0 && !p.peerEOF && !p.closed {
		p.cond.Wait()
	}
	if p.buffer.Len() == 0 {
		if p.closed || p.peerEOF {
			return 0, io.EOF
		}
		return 0, nil
	}
	return p.buffer.Read(dst)
}

func (p *bufferedPipe64) Write(src []byte) (int, error) {
	if p == nil || p.peer == nil {
		return 0, net.ErrClosed
	}
	p.mu.Lock()
	closed := p.closed
	p.mu.Unlock()
	if closed {
		return 0, net.ErrClosed
	}
	p.peer.mu.Lock()
	defer p.peer.mu.Unlock()
	if p.peer.closed || p.peer.peerEOF {
		return 0, net.ErrClosed
	}
	_, _ = p.peer.buffer.Write(src)
	p.peer.cond.Broadcast()
	return len(src), nil
}

func (p *bufferedPipe64) Close() error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.cond.Broadcast()
	p.mu.Unlock()
	if p.peer != nil {
		p.peer.mu.Lock()
		p.peer.peerEOF = true
		p.peer.cond.Broadcast()
		p.peer.mu.Unlock()
	}
	return nil
}

func (p *bufferedPipe64) LocalAddr() net.Addr              { return bufferedPipeAddr64("local") }
func (p *bufferedPipe64) RemoteAddr() net.Addr             { return bufferedPipeAddr64("remote") }
func (p *bufferedPipe64) SetDeadline(time.Time) error      { return nil }
func (p *bufferedPipe64) SetReadDeadline(time.Time) error  { return nil }
func (p *bufferedPipe64) SetWriteDeadline(time.Time) error { return nil }

var (
	errSocketNotConnected = errors.New("socket: not connected")
	errSocketNotListener  = errors.New("socket: not a listener")
)

func (s *socketHandle64) Read(p []byte) (int, error) {
	if s == nil || s.conn == nil {
		return 0, errSocketNotConnected
	}
	return s.conn.Read(p)
}

func (s *socketHandle64) Write(p []byte) (int, error) {
	if s == nil || s.conn == nil {
		return 0, errSocketNotConnected
	}
	return s.conn.Write(p)
}

func (s *socketHandle64) Close() error {
	if s == nil {
		return nil
	}
	if s.conn != nil {
		err := s.conn.Close()
		s.conn = nil
		return err
	}
	if s.listener != nil {
		err := s.listener.Close()
		s.listener = nil
		return err
	}
	return nil
}

func socketFile64(handle *socketHandle64, flags uint64) *corefd.File {
	return &corefd.File{
		Reader:  handle,
		Writer:  handle,
		Closer:  handle,
		Opaque:  handle,
		Cloexec: flags&socketCloexec != 0,
	}
}

func socketHandleFromFile64(file *corefd.File) (*socketHandle64, bool) {
	if file == nil {
		return nil, false
	}
	handle, ok := file.Opaque.(*socketHandle64)
	return handle, ok && handle != nil
}

func parseSockaddr64(ctx *Context64, address corecpu.Address64, length uint64) (socketAddr64, int32) {
	if ctx == nil || ctx.Memory == nil || address == 0 || length < 2 || length > 128 {
		return socketAddr64{}, EFAULT
	}
	data := make([]byte, int(length))
	if err := ctx.Memory.Read(address, data); err != nil {
		return socketAddr64{}, EFAULT
	}
	family := binary.LittleEndian.Uint16(data[:2])
	switch family {
	case socketAFInet:
		if len(data) < 8 {
			return socketAddr64{}, EINVAL
		}
		return socketAddr64{
			family: family,
			port:   binary.BigEndian.Uint16(data[2:4]),
			host:   net.IPv4(data[4], data[5], data[6], data[7]).String(),
		}, 0
	case socketAFInet6:
		if len(data) < 24 {
			return socketAddr64{}, EINVAL
		}
		return socketAddr64{
			family: family,
			port:   binary.BigEndian.Uint16(data[2:4]),
			host:   net.IP(data[8:24]).String(),
		}, 0
	case socketAFUnix:
		if len(data) < 3 {
			return socketAddr64{}, EINVAL
		}
		path := string(data[2:])
		if index := strings.IndexByte(path, 0); index >= 0 {
			path = path[:index]
		}
		if path == "" {
			return socketAddr64{}, EINVAL
		}
		return socketAddr64{family: family, path: path}, 0
	default:
		return socketAddr64{}, EAFNOSUPPORT
	}
}

func socketNetwork64(family uint16, kind int) (string, int32) {
	if kind != socketStream {
		return "", EOPNOTSUPP
	}
	switch family {
	case socketAFInet:
		return "tcp4", 0
	case socketAFInet6:
		return "tcp6", 0
	default:
		return "", EAFNOSUPPORT
	}
}

func socket64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FDs == nil {
		return int64(ENOMEM)
	}
	family := uint16(args[0])
	kind := int(args[1] & 0xf)
	network, result := socketNetwork64(family, kind)
	if result != 0 {
		return int64(result)
	}
	if args[2] != 0 {
		return int64(EOPNOTSUPP)
	}
	handle := &socketHandle64{family: family, kind: kind, network: network}
	fd, err := ctx.FDs.Open(socketFile64(handle, args[1]))
	if err != nil {
		return int64(EMFILE)
	}
	return int64(fd)
}

func socketpair64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FDs == nil || args[0] != socketAFUnix || int(args[1]&0xf) != socketStream || args[2] != 0 {
		return int64(EOPNOTSUPP)
	}
	left, right := newBufferedPipe64()
	leftHandle := &socketHandle64{family: socketAFUnix, kind: socketStream, conn: left}
	rightHandle := &socketHandle64{family: socketAFUnix, kind: socketStream, conn: right}
	leftFD, err := ctx.FDs.Open(socketFile64(leftHandle, args[1]))
	if err != nil {
		_ = left.Close()
		_ = right.Close()
		return int64(EMFILE)
	}
	rightFD, err := ctx.FDs.Open(socketFile64(rightHandle, args[1]))
	if err != nil {
		_ = ctx.FDs.Close(leftFD)
		_ = right.Close()
		return int64(EMFILE)
	}
	if ctx.Memory == nil || args[3] == 0 {
		_ = ctx.FDs.Close(leftFD)
		_ = ctx.FDs.Close(rightFD)
		return int64(EFAULT)
	}
	var pair [8]byte
	binary.LittleEndian.PutUint32(pair[0:4], uint32(leftFD))
	binary.LittleEndian.PutUint32(pair[4:8], uint32(rightFD))
	if err := ctx.Memory.Write(corecpu.Address64(args[3]), pair[:]); err != nil {
		_ = ctx.FDs.Close(leftFD)
		_ = ctx.FDs.Close(rightFD)
		return int64(EFAULT)
	}
	return 0
}

func bind64(ctx *Context64, args [6]uint64) int64 {
	file, err := ctx.GetFile(args[0])
	if err != nil {
		return int64(EBADF)
	}
	handle, ok := socketHandleFromFile64(file)
	if !ok {
		return int64(ENOTSOCK)
	}
	if handle.listener != nil || handle.conn != nil {
		return int64(EINVAL)
	}
	address, result := parseSockaddr64(ctx, corecpu.Address64(args[1]), args[2])
	if result != 0 {
		return int64(result)
	}
	if address.family != handle.family {
		return int64(EINVAL)
	}
	if address.family == socketAFUnix {
		return int64(EOPNOTSUPP)
	}
	listener, listenErr := net.Listen(handle.network, net.JoinHostPort(address.host, itoaPort64(address.port)))
	if listenErr != nil {
		return int64(errnoNetwork64(listenErr))
	}
	handle.listener = listener
	return 0
}

func listen64(ctx *Context64, args [6]uint64) int64 {
	file, err := ctx.GetFile(args[0])
	if err != nil {
		return int64(EBADF)
	}
	handle, ok := socketHandleFromFile64(file)
	if !ok {
		return int64(ENOTSOCK)
	}
	if handle.listener == nil {
		return int64(EINVAL)
	}
	return 0
}

func connect64(ctx *Context64, args [6]uint64) int64 {
	file, err := ctx.GetFile(args[0])
	if err != nil {
		return int64(EBADF)
	}
	handle, ok := socketHandleFromFile64(file)
	if !ok {
		return int64(ENOTSOCK)
	}
	if handle.conn != nil || handle.listener != nil {
		return int64(EISCONN)
	}
	address, result := parseSockaddr64(ctx, corecpu.Address64(args[1]), args[2])
	if result != 0 {
		return int64(result)
	}
	if address.family != handle.family || address.family == socketAFUnix {
		return int64(EOPNOTSUPP)
	}
	conn, dialErr := net.DialTimeout(handle.network, net.JoinHostPort(address.host, itoaPort64(address.port)), 5*time.Second)
	if dialErr != nil {
		return int64(errnoNetwork64(dialErr))
	}
	handle.conn = conn
	return 0
}

func accept64(ctx *Context64, args [6]uint64) int64 {
	return acceptCommon64(ctx, args[0], args[1], args[2], 0)
}

func accept464(ctx *Context64, args [6]uint64) int64 {
	return acceptCommon64(ctx, args[0], args[1], args[2], args[3])
}

func acceptCommon64(ctx *Context64, fd, address, addressLength, flags uint64) int64 {
	file, err := ctx.GetFile(fd)
	if err != nil {
		return int64(EBADF)
	}
	handle, ok := socketHandleFromFile64(file)
	if !ok {
		return int64(ENOTSOCK)
	}
	if handle.listener == nil {
		return int64(EINVAL)
	}
	listener, ok := handle.listener.(interface{ Accept() (net.Conn, error) })
	if !ok {
		return int64(EINVAL)
	}
	conn, acceptErr := listener.Accept()
	if acceptErr != nil {
		return int64(errnoNetwork64(acceptErr))
	}
	accepted := &socketHandle64{family: handle.family, kind: handle.kind, network: handle.network, conn: conn}
	newFD, openErr := ctx.FDs.Open(socketFile64(accepted, flags))
	if openErr != nil {
		_ = conn.Close()
		return int64(EMFILE)
	}
	if address != 0 {
		if result := writeSocketPeer64(ctx, corecpu.Address64(address), corecpu.Address64(addressLength), conn.RemoteAddr()); result != 0 {
			_ = ctx.FDs.Close(newFD)
			return int64(result)
		}
	}
	return int64(newFD)
}

func getsockname64(ctx *Context64, args [6]uint64) int64 {
	file, err := ctx.GetFile(args[0])
	if err != nil {
		return int64(EBADF)
	}
	handle, ok := socketHandleFromFile64(file)
	if !ok {
		return int64(ENOTSOCK)
	}
	var address net.Addr
	if handle.conn != nil {
		address = handle.conn.LocalAddr()
	} else if handle.listener != nil {
		address = handle.listener.Addr()
	} else {
		return int64(ENOTCONN)
	}
	return int64(writeSocketPeer64(ctx, corecpu.Address64(args[1]), corecpu.Address64(args[2]), address))
}

func sendto64(ctx *Context64, args [6]uint64) int64 {
	if args[4] != 0 {
		return int64(EOPNOTSUPP)
	}
	return write64(ctx, [6]uint64{args[0], args[1], args[2]})
}

func recvfrom64(ctx *Context64, args [6]uint64) int64 {
	if args[4] != 0 && args[5] != 0 {
		return int64(EOPNOTSUPP)
	}
	return read64(ctx, [6]uint64{args[0], args[1], args[2]})
}

func shutdown64(ctx *Context64, args [6]uint64) int64 {
	file, err := ctx.GetFile(args[0])
	if err != nil {
		return int64(EBADF)
	}
	handle, ok := socketHandleFromFile64(file)
	if !ok || handle.conn == nil {
		return int64(ENOTSOCK)
	}
	if args[1] > 2 {
		return int64(EINVAL)
	}
	if tcp, ok := handle.conn.(*net.TCPConn); ok {
		var shutdownErr error
		switch args[1] {
		case 0:
			shutdownErr = tcp.CloseRead()
		case 1:
			shutdownErr = tcp.CloseWrite()
		default:
			shutdownErr = tcp.Close()
		}
		if shutdownErr != nil {
			return int64(errnoNetwork64(shutdownErr))
		}
		return 0
	}
	return int64(EOPNOTSUPP)
}

func writeSocketPeer64(ctx *Context64, address, addressLength corecpu.Address64, peer net.Addr) int32 {
	if ctx == nil || ctx.Memory == nil || address == 0 || addressLength == 0 || peer == nil {
		return EFAULT
	}
	lengthBytes := make([]byte, 4)
	if err := ctx.Memory.Read(addressLength, lengthBytes); err != nil {
		return EFAULT
	}
	capacity := binary.LittleEndian.Uint32(lengthBytes)
	encoded, result := encodeSocketAddr64(peer)
	if result != 0 {
		return result
	}
	if capacity < uint32(len(encoded)) {
		encoded = encoded[:capacity]
	}
	if err := ctx.Memory.Write(address, encoded); err != nil {
		return EFAULT
	}
	binary.LittleEndian.PutUint32(lengthBytes, uint32(len(encoded)))
	if err := ctx.Memory.Write(addressLength, lengthBytes); err != nil {
		return EFAULT
	}
	return 0
}

func encodeSocketAddr64(peer net.Addr) ([]byte, int32) {
	switch address := peer.(type) {
	case *net.TCPAddr:
		if address.IP.To4() != nil {
			encoded := make([]byte, 16)
			binary.LittleEndian.PutUint16(encoded[0:2], socketAFInet)
			binary.BigEndian.PutUint16(encoded[2:4], uint16(address.Port))
			copy(encoded[4:8], address.IP.To4())
			return encoded, 0
		}
		encoded := make([]byte, 28)
		binary.LittleEndian.PutUint16(encoded[0:2], socketAFInet6)
		binary.BigEndian.PutUint16(encoded[2:4], uint16(address.Port))
		copy(encoded[8:24], address.IP.To16())
		return encoded, 0
	case *net.UnixAddr:
		encoded := make([]byte, 2+len(address.Name)+1)
		binary.LittleEndian.PutUint16(encoded[:2], socketAFUnix)
		copy(encoded[2:], address.Name)
		return encoded, 0
	default:
		return nil, EOPNOTSUPP
	}
}

func itoaPort64(port uint16) string {
	if port == 0 {
		return "0"
	}
	var buffer [5]byte
	index := len(buffer)
	value := uint32(port)
	for value > 0 {
		index--
		buffer[index] = byte('0' + value%10)
		value /= 10
	}
	return string(buffer[index:])
}

func errnoNetwork64(err error) int32 {
	if err == nil {
		return 0
	}
	if errors.Is(err, os.ErrNotExist) {
		return ENOENT
	}
	if errors.Is(err, os.ErrPermission) {
		return EACCES
	}
	if errors.Is(err, net.ErrClosed) {
		return EBADF
	}
	if errors.Is(err, io.EOF) {
		return EPIPE
	}
	return EIO
}

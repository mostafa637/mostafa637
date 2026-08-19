package syscall

import (
	"bytes"
	"encoding/binary"
	"errors"
	"runtime"
	"sync"
	"time"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
)

const (
	pollIn        uint16 = 0x0001
	pollOut       uint16 = 0x0004
	pollErr       uint16 = 0x0008
	pollHup       uint16 = 0x0010
	pollNVal      uint16 = 0x0020
	pollRDNORM    uint16 = 0x0040
	pollWRNORM    uint16 = 0x0100
	pipe2Cloexec         = 0x00080000
	pipe2Nonblock        = 0x00000800
)

type guestPipe struct {
	mu          sync.Mutex
	cond        *sync.Cond
	buffer      bytes.Buffer
	readClosed  bool
	writeClosed bool
}

func newGuestPipe() *guestPipe {
	pipe := &guestPipe{}
	pipe.cond = sync.NewCond(&pipe.mu)
	return pipe
}

func (p *guestPipe) read(dst []byte) (int, error) {
	if len(dst) == 0 {
		return 0, nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for p.buffer.Len() == 0 && !p.writeClosed && !p.readClosed {
		p.cond.Wait()
	}
	if p.buffer.Len() != 0 {
		return p.buffer.Read(dst)
	}
	if p.readClosed {
		return 0, errors.New("pipe: read end closed")
	}
	return 0, nil
}

func (p *guestPipe) write(src []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.readClosed || p.writeClosed {
		return 0, errors.New("pipe: closed")
	}
	n, err := p.buffer.Write(src)
	p.cond.Broadcast()
	return n, err
}

func (p *guestPipe) closeRead() error {
	p.mu.Lock()
	p.readClosed = true
	p.cond.Broadcast()
	p.mu.Unlock()
	return nil
}

func (p *guestPipe) closeWrite() error {
	p.mu.Lock()
	p.writeClosed = true
	p.cond.Broadcast()
	p.mu.Unlock()
	return nil
}

func (p *guestPipe) ready(events uint16) uint16 {
	p.mu.Lock()
	defer p.mu.Unlock()
	var result uint16
	if events&(pollIn|pollRDNORM) != 0 && (p.buffer.Len() > 0 || p.writeClosed) {
		result |= events & (pollIn | pollRDNORM)
	}
	if events&(pollOut|pollWRNORM) != 0 && !p.readClosed && !p.writeClosed {
		result |= events & (pollOut | pollWRNORM)
	}
	if p.readClosed || p.writeClosed {
		result |= pollHup
	}
	return result
}

type pipeReader struct{ pipe *guestPipe }

func (r *pipeReader) Read(p []byte) (int, error) { return r.pipe.read(p) }
func (r *pipeReader) Close() error               { return r.pipe.closeRead() }

func (r *pipeReader) poll(events uint16) uint16 { return r.pipe.ready(events) }

type pipeWriter struct{ pipe *guestPipe }

func (w *pipeWriter) Write(p []byte) (int, error) { return w.pipe.write(p) }
func (w *pipeWriter) Close() error                { return w.pipe.closeWrite() }

func (w *pipeWriter) poll(events uint16) uint16 { return w.pipe.ready(events) }

func pipe(context *Context, state *corecpu.MachineState, args [6]uint32) int32 {
	return makePipe(context, state, corecpu.Address(args[0]), 0)
}

func pipe2(context *Context, state *corecpu.MachineState, args [6]uint32) int32 {
	flags := args[1]
	if flags&^(uint32(pipe2Cloexec)|uint32(pipe2Nonblock)) != 0 {
		return EINVAL
	}
	return makePipe(context, state, corecpu.Address(args[0]), flags)
}

func makePipe(context *Context, state *corecpu.MachineState, address corecpu.Address, _ uint32) int32 {
	if context == nil || context.Memory == nil || context.FDs == nil || address == 0 {
		return EFAULT
	}
	pipe := newGuestPipe()
	reader := &corefd.File{Reader: &pipeReader{pipe: pipe}, Closer: &pipeReader{pipe: pipe}}
	writer := &corefd.File{Writer: &pipeWriter{pipe: pipe}, Closer: &pipeWriter{pipe: pipe}}
	reader.Poll = func(events uint16) uint16 { return pipe.ready(events) }
	writer.Poll = func(events uint16) uint16 { return pipe.ready(events) }
	readFD, err := context.FDs.Open(reader)
	if err != nil {
		return EMFILE
	}
	writeFD, err := context.FDs.Open(writer)
	if err != nil {
		_ = context.FDs.Close(readFD)
		return EMFILE
	}
	if context.Files == nil {
		context.Files = make(map[uint32]*File)
	}
	context.Files[uint32(readFD)] = reader
	context.Files[uint32(writeFD)] = writer
	var result [8]byte
	binary.LittleEndian.PutUint32(result[0:4], uint32(readFD))
	binary.LittleEndian.PutUint32(result[4:8], uint32(writeFD))
	if err := context.Memory.Write(address, result[:]); err != nil {
		_ = context.FDs.Close(readFD)
		_ = context.FDs.Close(writeFD)
		delete(context.Files, uint32(readFD))
		delete(context.Files, uint32(writeFD))
		return EFAULT
	}
	return 0
}

func poll(context *Context, state *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || context.Memory == nil {
		return EFAULT
	}
	nfds := args[1]
	if nfds > 4096 {
		return EINVAL
	}
	timeout := int32(args[2])
	deadline := time.Time{}
	if timeout >= 0 {
		deadline = time.Now().Add(time.Duration(timeout) * time.Millisecond)
	}
	for {
		ready := int32(0)
		for index := uint32(0); index < nfds; index++ {
			address := corecpu.Address(args[0] + index*8)
			var raw [8]byte
			if err := context.Memory.Read(address, raw[:]); err != nil {
				return EFAULT
			}
			fd := int32(binary.LittleEndian.Uint32(raw[0:4]))
			events := binary.LittleEndian.Uint16(raw[4:6])
			var revents uint16
			if fd < 0 {
				revents = 0
				revents |= pollNVal
			} else {
				file := context.file(uint32(fd))
				if file == nil {
					revents = pollNVal
				} else if file.Poll != nil {
					revents = file.Poll(events)
				} else if events&(pollOut|pollWRNORM) != 0 && file.Writer != nil {
					revents = events & (pollOut | pollWRNORM)
				}
			}
			binary.LittleEndian.PutUint16(raw[6:8], revents)
			if err := context.Memory.Write(address+6, raw[6:8]); err != nil {
				return EFAULT
			}
			if revents != 0 {
				ready++
			}
		}
		if ready != 0 || timeout == 0 {
			return ready
		}
		if !deadline.IsZero() && !time.Now().Before(deadline) {
			return 0
		}
		runtime.Gosched()
		time.Sleep(time.Millisecond)
	}
}

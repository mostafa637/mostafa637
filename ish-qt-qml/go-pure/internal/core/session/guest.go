package session

import (
	"context"
	"io"
	"sync"

	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"
	corekernel "github.com/mostafa637/mostafa637/go-pure/internal/core/kernel"
	coreloader "github.com/mostafa637/mostafa637/go-pure/internal/core/loader"
)

type guestTransport struct {
	elfPath string
	input   chan []byte
	output  chan []byte
	done    chan struct{}
	process *corekernel.Process

	stopOnce   sync.Once
	outputOnce sync.Once
}

func newGuestTransport(elfPath string) *guestTransport {
	return &guestTransport{
		elfPath: elfPath,
		input:   make(chan []byte, 32),
		output:  make(chan []byte, 32),
		done:    make(chan struct{}),
	}
}

func (g *guestTransport) start(ctx context.Context, process *corekernel.Process, fake *corefs.FS, cols, rows int) error {
	if g == nil || process == nil || fake == nil {
		return transportError("guest session: nil process or fakefs")
	}
	if g.elfPath == "" {
		return transportError("guest session: ELF path is empty")
	}
	data, err := fake.ReadFile(g.elfPath)
	if err != nil {
		return err
	}
	if cols < 1 {
		cols = 80
	}
	if rows < 1 {
		rows = 24
	}
	if isELF64Image(data) {
		return g.start64(ctx, process, fake, data, cols, rows)
	}
	if err := process.SetWindowSize(uint16(cols), uint16(rows)); err != nil {
		return err
	}
	g.process = process
	reader := &guestInput{chunks: g.input, done: g.done}
	writer := &guestOutput{chunks: g.output, done: g.done}
	if err := process.AttachFile(0, reader, nil); err != nil {
		return err
	}
	if err := process.AttachFile(1, nil, writer); err != nil {
		return err
	}
	if err := process.AttachFile(2, nil, writer); err != nil {
		return err
	}
	stack := coreloader.DefaultStackConfig()
	stack.Argv = []string{g.elfPath}
	stack.Env = []string{"PATH=/bin:/usr/bin"}
	if _, err := process.LoadELF(bytesReader(data), int64(len(data)), g.elfPath, 0, stack); err != nil {
		return err
	}
	runDone := make(chan struct{})
	go func() {
		defer close(runDone)
		_ = process.Run(10_000_000)
		g.closeOutput()
	}()
	if ctx != nil {
		go func() {
			select {
			case <-ctx.Done():
				g.stop()
			case <-runDone:
			}
		}()
	}
	return nil
}

func (g *guestTransport) resize(cols, rows int) error {
	if g == nil || g.process == nil {
		return transportError("guest session: not started")
	}
	if cols < 1 || rows < 1 {
		return nil
	}
	return g.process.SetWindowSize(uint16(cols), uint16(rows))
}

func (g *guestTransport) write(data []byte) error {
	if g == nil {
		return transportError("guest session: nil transport")
	}
	copyData := append([]byte(nil), data...)
	select {
	case <-g.done:
		return io.ErrClosedPipe
	case g.input <- copyData:
		return nil
	}
}

func (g *guestTransport) stop() {
	if g == nil {
		return
	}
	g.stopOnce.Do(func() { close(g.done) })
}

func (g *guestTransport) closeOutput() {
	if g == nil {
		return
	}
	g.outputOnce.Do(func() { close(g.output) })
}

func (g *guestTransport) outputChannel() <-chan []byte {
	if g == nil {
		return nil
	}
	return g.output
}

type guestInput struct {
	chunks  <-chan []byte
	done    <-chan struct{}
	pending []byte
}

func (r *guestInput) Read(p []byte) (int, error) {
	for len(r.pending) == 0 {
		select {
		case <-r.done:
			return 0, io.EOF
		case chunk, ok := <-r.chunks:
			if !ok {
				return 0, io.EOF
			}
			r.pending = chunk
		}
	}
	n := copy(p, r.pending)
	r.pending = r.pending[n:]
	return n, nil
}

type guestOutput struct {
	chunks chan<- []byte
	done   <-chan struct{}
}

func (w *guestOutput) Write(p []byte) (int, error) {
	copyData := append([]byte(nil), p...)
	select {
	case <-w.done:
		return 0, io.ErrClosedPipe
	case w.chunks <- copyData:
		return len(p), nil
	}
}

type transportError string

func (e transportError) Error() string { return string(e) }

// bytesReader is kept local so the guest transport has no dependency on
// os.File or a host-specific descriptor implementation.
type byteReader struct{ data []byte }

func bytesReader(data []byte) io.ReaderAt { return &byteReader{data: data} }

func (r *byteReader) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 || off >= int64(len(r.data)) {
		return 0, io.EOF
	}
	n := copy(p, r.data[off:])
	if n != len(p) {
		return n, io.EOF
	}
	return n, nil
}

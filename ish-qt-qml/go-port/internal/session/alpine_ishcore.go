//go:build ishcore && cgo

package session

/*
#cgo CFLAGS: -I${SRCDIR}/../../../upstream/ish-ios/app/core -I${SRCDIR}/../../../upstream/ish-ios -I${SRCDIR}/../../native
#cgo LDFLAGS: -L${SRCDIR}/../../native/lib -Wl,--start-group -lish_core_session -lish_kernel -lish_emu -lish_fakefs -lish_sqlite -Wl,--end-group -lz -lm -pthread
#include <stdint.h>
#include <stdlib.h>
#include "CoreSession.h"

extern void goIshOutput(void *cookie, char *bytes, size_t length);
extern void goIshState(void *cookie, int exit_code);

static void ishGoOutput(void *cookie, const char *bytes, size_t length) {
    goIshOutput(cookie, (char *)bytes, length);
}

static void ishGoState(void *cookie, int exit_code) {
    goIshState(cookie, exit_code);
}

static IshCoreSession *ishGoCreate(const char *root, uintptr_t cookie) {
    const char *launch[] = {"/bin/ash", "-l"};
    return ish_core_session_create(root, NULL, 0, launch, 2,
                                   ishGoOutput, ishGoState, (void *)cookie);
}
*/
import "C"

import (
	bytespkg "bytes"
	"context"
	"errors"
	"io"
	"log"
	"runtime/cgo"
	"sync"
	"unsafe"
)

func nativeCoreAvailable() bool { return true }

type nativeSession struct {
	core   *C.IshCoreSession
	handle cgo.Handle

	mu     sync.Mutex
	closed bool
	out    chan []byte
	done   chan error
}

//export goIshOutput
func goIshOutput(cookie unsafe.Pointer, bytes *C.char, length C.size_t) {
	handle := cgo.Handle(uintptr(cookie))
	s, ok := handle.Value().(*nativeSession)
	if !ok || s == nil || bytes == nil || length == 0 {
		return
	}
	chunk := C.GoBytes(unsafe.Pointer(bytes), C.int(length))
	if bytespkg.Contains(chunk, []byte("GO_ALPINE_AVD_OK")) ||
		bytespkg.Contains(chunk, []byte("GO-ALPINE-AVD-OK")) {
		log.Print("iSH Alpine smoke marker received")
	}

	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return
	}
	select {
	case s.out <- chunk:
	default:
		// Never block the native iSH worker if Gio is between frames.
	}
}

//export goIshState
func goIshState(cookie unsafe.Pointer, exitCode C.int) {
	handle := cgo.Handle(uintptr(cookie))
	s, ok := handle.Value().(*nativeSession)
	if !ok || s == nil {
		return
	}
	select {
	case s.done <- nil:
	default:
	}
}

func startAlpine(ctx context.Context, rootPath string) (*Session, error) {
	if rootPath == "" {
		return nil, errors.New("session: empty Alpine rootfs path")
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s := &nativeSession{
		out:  make(chan []byte, 64),
		done: make(chan error, 1),
	}
	s.handle = cgo.NewHandle(s)
	root := C.CString(rootPath)
	defer C.free(unsafe.Pointer(root))
	s.core = C.ishGoCreate(root, C.uintptr_t(s.handle))
	if s.core == nil {
		s.handle.Delete()
		return nil, errors.New("session: iSH core allocation failed")
	}
	if !bool(C.ish_core_session_start(s.core)) {
		C.ish_core_session_destroy(s.core)
		s.handle.Delete()
		return nil, errors.New("session: iSH core failed to start")
	}
	return newSession(s), nil
}

func (s *nativeSession) Output() <-chan []byte { return s.out }
func (s *nativeSession) Done() <-chan error    { return s.done }

func (s *nativeSession) Write(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return io.ErrClosedPipe
	}
	if len(data) == 0 {
		return nil
	}
	ptr := C.CBytes(data)
	defer C.free(ptr)
	written := C.ish_core_session_write(s.core, (*C.char)(ptr), C.size_t(len(data)))
	if int(written) != len(data) {
		return io.ErrShortWrite
	}
	return nil
}

func (s *nativeSession) Resize(cols, rows uint16) error {
	if cols == 0 || rows == 0 {
		return errors.New("session: terminal dimensions must be non-zero")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return io.ErrClosedPipe
	}
	C.ish_core_session_resize(s.core, C.int(cols), C.int(rows))
	return nil
}

func (s *nativeSession) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.mu.Unlock()
	C.ish_core_session_destroy(s.core)
	s.handle.Delete()
	close(s.out)
	return nil
}

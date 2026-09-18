// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre

package jit

import (
	"syscall"
	"testing"
	"unsafe"
)

// TestCallBlockShim checks the one piece of assembly in the package: that a
// block is entered with the state pointer where the backend expects it and
// that its result comes back. Everything else in the JIT depends on it, and a
// host ABI change would break it silently (a wrong argument register reads
// plausible garbage rather than faulting).
func TestCallBlockShim(t *testing.T) {
	code, err := syscall.Mmap(-1, 0, 4096,
		syscall.PROT_READ|syscall.PROT_WRITE|syscall.PROT_EXEC,
		syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		t.Skip("no executable memory: ", err)
	}
	defer syscall.Munmap(code)

	// mov eax, 42 ; ret
	copy(code, []byte{0xb8, 42, 0, 0, 0, 0xc3})
	st := &State{}
	if got := callBlock(uintptr(unsafe.Pointer(&code[0])), st); got != 42 {
		t.Fatalf("callBlock: got %d, want 42", got)
	}

	// mov rax, [rdi+8] ; ret  -- the state pointer has to arrive in rdi.
	copy(code, []byte{0x48, 0x8b, 0x47, 0x08, 0xc3})
	st.X[1] = 0x1234
	if got := callBlock(uintptr(unsafe.Pointer(&code[0])), st); got != 0x1234 {
		t.Fatalf("callBlock argument: got 0x%x, want 0x1234 (X[1])", got)
	}
}

// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre

//go:build amd64

#include "textflag.h"

// callBlock enters compiled code with the C-like convention the backend
// emits: the state pointer in RDI, the result in RAX.
//
// A Go func value cannot be pointed at machine code any more (the runtime
// validates what it calls), so the jump goes through this shim instead — which
// is also the one place the host ABI is named, and the reason this port needs
// no cgo and no C compiler to build.
TEXT ·callBlock(SB), NOSPLIT, $0-20
	MOVQ fn+0(FP), AX
	MOVQ st+8(FP), DI
	CALL AX
	MOVL AX, ret+16(FP)
	RET

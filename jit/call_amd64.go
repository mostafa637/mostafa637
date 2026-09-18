// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre

//go:build amd64

package jit

// callBlock calls the machine code at fn with st in the register the backend
// expects. Implemented in call_amd64.s.
//
//go:noescape
func callBlock(fn uintptr, st *State) uint32

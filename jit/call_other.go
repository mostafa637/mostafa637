// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre

//go:build !amd64

package jit

// callBlock does not exist off amd64: there is no backend for those hosts yet,
// so no block is ever compiled and the interpreter runs everything.
func callBlock(fn uintptr, st *State) uint32 { return ExitUnsup }

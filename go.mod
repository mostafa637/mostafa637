// arm64chroot-go — a Go port of https://github.com/sylirre/arm64emu-user
//
// The layout mirrors the C sources it was translated from:
//
//	src/core/{cpu,decode,sysreg,exec_fpsimd}.c     -> core/      (the CPU core)
//	src/{mmu.h,mem.c}                              -> mem/       (guest address space)
//	src/elf.c                                      -> elf/       (loader, stack, auxv)
//	src/{syscall.c,sys_*.c}                        -> linux/     (the ~190 Linux syscalls)
//	src/{loop.c,exception.c,main.c}                -> ./main.go  (run loop + CLI)
//	src/jit/*                                      -> jit/       (IR + code generators)
//	src/predecode.c                                -> core/predecode.go
//
// Where Go forces a different shape than C (GC-safe host pointers instead of
// raw uintptr page-table entries, atomics through sync/atomic instead of
// __atomic builtins on a void*, ...), the difference is recorded in
// docs/PORT.md rather than papered over.
module github.com/mostafa637/mostafa637

go 1.24

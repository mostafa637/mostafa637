# arm64chroot, in Go

An AArch64 Linux user-mode emulator with a chroot around it: it runs an
`aarch64` static binary against a Linux syscall layer that keeps every path
inside a directory tree you hand it. This tree is a port of
[sylirre/arm64emu-user](https://github.com/sylirre/arm64emu-user) (C, ~57k
lines) to Go, at the root of this repository.

    go build -o arm64chroot-go .
    ./arm64chroot-go /path/to/rootfs /bin/busybox sh

The options are the original's:

| option | meaning |
|---|---|
| `-0`, `-argv0` | override `argv[0]` for the guest |
| `-w`, `-work-dir` | guest working directory (default `/`) |
| `-E`, `-env` | set a guest environment variable (repeatable) |
| `-b`, `-bind` | bind a host directory into the guest as `SRC:DST` (repeatable) |
| `-u`, `-fake-id` | present a fake identity `uid[:gid]` |
| `-j`, `-jit` | translate hot guest code instead of interpreting it |
| `--strace` | log guest syscalls to stderr |
| `-d`, `-debug` | per-instruction trace |
| `--max-insns` | stop after N guest instructions (diagnostic) |
| `-v` | version |

Standard library only, no cgo, no C compiler needed to build.

## Layout

```
abi/     errno values the guest sees
core/    CPU: registers, instruction decode, system registers, exceptions
mem/     the guest address space
elf/     AArch64 ELF loader
linux/   Linux-user layer: 107 syscalls, process state, guest signals
jit/     IR + A64 lifter + x86-64 backend
core/fpsimd.go       scalar floating point (single and double)
core/fpsimd_simd.go  the Advanced SIMD integer groups
main.go  command line and run loop
tests/   differential test corpus and runner
```

## Status

The interpreter is complete except for half-precision, the FP/vector groups
and the crypto extensions: a 224-result AArch64 corpus, run under both the C
emulator and this one, matches on all but five words, and those five are the
feature fields (AdvSIMD, AES, SHA, RDM, DotProd, FHM, JSCVT, FCMA) that are
deliberately advertised as absent until the executor behind them exists. The
JIT covers the integer subset that hot loops are made of and falls back to the
interpreter for everything else.

    go build ./... && go vet ./... && go test ./...
    python3 tests/asm/diffrun.py                    # vs the C emulator
    ARM64EMU_FLAGS=-j python3 tests/asm/diffrun.py  # same, through the JIT

`docs/PORT.md` has the whole account: how each C subsystem maps to Go, the
design decisions that had to change (memory representation, bitmask memoization,
the assembly shim the JIT is entered through, feature advertisement), and a
table of what is not ported yet with the reason for each.

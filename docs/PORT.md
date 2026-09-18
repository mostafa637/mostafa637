# The Go port

`https://github.com/sylirre/arm64emu-user` is an ARM64 Linux user-mode
emulator with a chroot around it, written in C (~57k lines across `src/`).
This directory tree is that emulator translated to Go, at the repository
root:

```
abi/     errno values the guest sees
core/    the CPU: registers, decode, system registers, exceptions   (src/core/)
mem/     the guest address space                                    (src/mem.c, mmu.h)
elf/     the AArch64 ELF loader                                     (src/elf.c)
linux/   the Linux-user layer: syscalls, process state, signals     (src/sys_*.c, signal.c, path.c)
jit/     the IR, the A64->IR lifter, the x86-64 backend             (src/jit/)
main.go  the command line and the run loop                          (src/main.c, loop.c)
tests/   a differential test corpus and runner
```

    cd mostafa637 && go build -o arm64chroot-go .
    ./arm64chroot-go [options] <rootfs> <program> [args...]

The options are the C emulator's: `--strace`, `-d/--debug`, `-0/--argv0`,
`-w/--work-dir`, `-E/--env`, `-b/--bind`, `-u/--fake-id`, `-j/--jit`,
`-v/--version`, plus `--max-insns` as a diagnostic.

Only the standard library is used — there is no module dependency, and the
build needs no cgo and no C compiler.

## How it is kept honest

`tests/asm/mkelf.py` builds an AArch64 test corpus with
[keystone](https://www.keystone-engine.org/) (there is no AArch64 toolchain in
the build sandbox) and `tests/asm/diffrun.py` runs it under the C emulator and
under this one, comparing the results word by word:

    pip install --break-system-packages keystone-engine
    go build -o arm64chroot-go .
    python3 tests/asm/diffrun.py             # interpreter
    ARM64EMU_FLAGS=-j python3 tests/asm/diffrun.py   # JIT

Every snippet leaves one 64-bit result in `x0`; the harness stores all of them
into a data page and writes the block to fd 1, so a run is a byte-for-byte
comparison. The corpus covers the move-wide, add/subtract, logical, bitfield,
PC-relative, shift, multiply/divide, conditional-select, data-processing-1/2-
source, CRC32 and FLAGM groups, loads and stores, the LSE atomics, branches,
every system register the emulator exposes, the stack, a loop, a hot loop, and
signal delivery (handler entry through a sigreturn trampoline, a blocked
signal that waits and then runs, and `SIG_IGN`).

Each group is built and run as its own guest program, so a snippet that traps
costs one group's results rather than the whole corpus, and every reported word
carries the name of the group that produced it.

Current status: **219 of 224 results match**, with five documented differences
(see *Feature advertisements* and *Generic timer* below). Passing `-j` in
`ARM64EMU_FLAGS` runs the same corpus through the JIT, with the same totals.

## Where the design follows the C, and where it cannot

**Memory.** The C core reaches guest memory through four function pointers
(`mem_read`/`mem_write`/`mem_read128`/`mem_host_ptr`) that `mem.c` fills in at
load time; the port keeps the same seam with package-level function variables
on `core` that `mem` installs in its `init()`. The CPU therefore has no import
of the address space, exactly as `src/core/` has no link dependency on
`mem.c`.

The C stores page-table entries as `uintptr`s with permission bits in the low
ones. Go has no place for that: a page is a `[]byte` sub-slice of a region's
backing store, and the D-TLB caches `{page, []byte, prot}` so the hot path is a
map lookup plus a slice index. Permission checks moved from bit tests on the
PTE to a comparison against the access being made.

**Exceptions.** The C's `exception_take` records into thread-local state and
lets `loop.c` dispatch; the port records into `CPU.Pending()` and lets
`linux.Task.Run` dispatch, keeping the CPU at EL0 for its whole life, as the
original does.

**Bitmask immediates.** `DecodeBitmasks` is memoized in C with a benign race.
Go has no benign races, and locking the hot path to fill a table costs more
than filling it, so all 8192 entries are computed at package init.

**Negative errno.** `-uint64(abi.EBADF)` does not compile in Go: negating an
untyped unsigned constant overflows. Every error return in `linux/` goes
through `negErrno`, which routes the value through a variable.

**Host calls.** Go 1.27's `syscall` package has shed the wrappers this port
needs (`Fcntl`, `ioctl`, `getdents64`, `clock_gettime`, `clock_getres`,
`sched_yield`, `faccessat2`, `memfd_create`, the socket calls, `recvmsg`).
`golang.org/x/sys` is not reachable from the build sandbox, so `linux/host.go`
makes them by number, with the numbers in `host_amd64.go` / `host_arm64.go`.

## Feature advertisements

The C core hardcodes the ID registers to one literal and the `AT_HWCAP` words
to another. That is a trap while a port is in progress: an ID register that
promises FP makes the guest issue FP instructions the executor does not have,
and the failure is a SIGILL three million instructions into libc rather than at
the missing feature. So `core/features.go` is the single source: the `MRS`-able
ID registers and `AT_HWCAP`/`AT_HWCAP2` are both derived from it, and it names
only what the decoder actually executes.

The consequence is the five differences the differential runner reports: the
AdvSIMD, AES, SHA, RDM, DotProd, FHM, JSCVT and FCMA fields read as zero. They
will switch on with the executor they describe, not before.

## The JIT

`src/jit/` is the largest single piece of the original: an IR, a lifter, and
four host backends (~11.8k lines). This port carries the IR, the lifter, and
one backend (x86-64), and lifts the integer subset a hot loop is made of:
move-wide, add/subtract (immediate and register, with and without flags),
logical operations, variable shifts, multiply, `CSEL`, and branches. Anything
else — memory accesses above all — ends the block with `OpUnsup`, and the
interpreter runs that one instruction before the JIT is entered again.

The backend is deliberately simpler than `backend_x86_64.c`, which is a
register allocator with a peephole pass. Here every IR operation is a
load/compute/store against the guest register file in memory, with two scratch
registers and nothing kept across IR instructions. That buys a short,
obviously-correct encoder over the maybe 30% a real allocator would add.

Two things about it are worth knowing before changing anything:

* **Blocks are entered through assembly, not a Go func value.** `jit/call_amd64.s`
  is a six-line shim (`MOVQ fn+0(FP), AX; MOVQ st+8(FP), DI; CALL AX`), which
  is also why the port needs no cgo: the backend emits the C-like convention
  (state pointer in `rdi`), and the shim is the only place a host ABI is named.
* **NZCV is not left in the host flags.** Each flag-setting operation captures
  it with `LAHF` + `SETO` into the stored word, and a conditional branch
  rebuilds the condition from that word. The host flags do not survive the
  intervening loads and stores, so expecting them to is the one bug worth
  looking for first if a loop misbehaves.

`-j` on a one-million-iteration integer loop: 0.15 s interpreted, 0.05 s
compiled (the C emulator takes 0.04 s). The same loop under `-j` produces the
same answer as both interpreters.

Known limitation: the cache is flushed on `munmap`, `mprotect` and `mremap`,
but a store into a page that holds compiled code is not detected, so
self-modifying code has to be excluded until that is wired up.

## Generic timer

`CNTVCT_EL0`/`CNTPCT_EL0` come from the host's `CLOCK_MONOTONIC` scaled to the
24 MHz the guest sees in `CNTFRQ_EL0`. The C reads the clock directly, so its
counter is time since boot; `main.go` reads `/proc/uptime` once and adds the
elapsed monotonic time, which is the same origin. The absolute value therefore
still differs between two processes started milliseconds apart — the
differential runner marks those results volatile.

## What is not ported

Ordered roughly by size, with the reason each is a separate piece of work
rather than an oversight:

| C source | lines | state here |
|---|---|---|
| `core/exec_fpsimd.c` | 4501 | **ported in three slices.** `core/fpsimd.go` has scalar FP (single and double); `core/fpsimd_simd.go` has the Advanced SIMD integer groups (three-same, modified immediate, copy, shift by immediate). Still missing: half-precision, the FP three-same and FP16 vector groups, the by-element and table/permute groups, the narrowing shifts and the crypto extensions. |
| `jit/backend_a64.c`, `backend_x86_32.c`, `backend_arm32.c` | 8354 | not ported; see *The JIT* |
| `sys_netlink.c` | 2048 | not ported (NETLINK_ROUTE dumps for `getaddrinfo`) |
| `sys_procfs.c` | 1991 | not ported (the synthetic `/proc` and `/sys`) |
| `proctab.c` | 3336 | not ported (the process table behind `/proc/<pid>`) |
| `signal.c` (host half) | ~1200 | host-signal capture is not ported: no `Ctrl-C` forwarding, no `SIGCHLD`, no POSIX timers. Guest-to-guest signals work. |
| `predecode.c` | 1521 | not ported |
| `strace.c` | 991 | `--strace` prints names and arguments, not the decoded structures |
| `sys_seccomp.c`, `sys_ptrace.c`, `ptracetab.c` | 2156 | not ported |
| `path.c` | 1897 | containment is lexical (`path.Clean` under the rootfs, then `EvalSymlinks`, then a prefix check). The C pins every component against a re-opened `O_PATH` descriptor, which defeats a rename or a symlink swapped mid-walk; this does not, and is the one place the security model is weaker than the original. |
| `sys_ipc.c` | 493 | not ported |
| the `ERESTART*` rules in `signal.c` | ~200 | a syscall interrupted by a handler reports `EINTR` where a kernel would sometimes resume it |

## Floating point

Scalar FP (single and double) is complete, including the fused multiply-add
family and the FP<->integer conversions with the architecture's saturation
rules. `core/fpsimd_simd.go` adds the integer half of Advanced SIMD: the
three-same group (arithmetic, compares, min/max, the halving and saturating
variants, the register shifts, PMUL, the pairwise forms and the whole-register
logicals), the modified immediate (MOVI/MVNI/ORR/BIC/FMOV), the copy group
(DUP/INS/UMOV/SMOV) and the shifts by immediate (SHL/SSHR/USHR/SLI/SRI/SSRA/
USRA/SRSHR/URSHR/SRSRA/URSRA/SQSHL/UQSHL/SQSHLU/SSHLL/USHLL).

Both files compute element-wise, taking the element size and lane count from
the encoding exactly as the C does, so the port stays diffable against it.

## Testing

    go build ./... && go vet ./... && go test ./...

`jit/shim_test.go` and `jit/block_test.go` are the JIT's own unit tests: the
assembly shim (the one piece that a host ABI change would break silently) and
one compiled block each for arithmetic, flag capture and conditional branches.

// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Ported to Go from src/main.c and src/loop.c.
//
// arm64chroot — a Linux user-space emulator for running unprivileged, isolated
// AArch64 chroot environments. This is the Go port; the C original is
// https://github.com/sylirre/arm64emu-user.
//
// Usage:
//
//	arm64chroot [options] <rootfs> <program> [args...]
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mostafa637/mostafa637/core"
	"github.com/mostafa637/mostafa637/elf"
	"github.com/mostafa637/mostafa637/jit"
	"github.com/mostafa637/mostafa637/linux"
)

// Version is the emulator's own version, reported by --version.
const Version = "arm64chroot-go 0.1.0 (ported from arm64emu-user)"

type options struct {
	strace   bool
	debug    bool
	version  bool
	argv0    string
	workDir  string
	env      arrayFlags
	bind     arrayFlags
	maxInsns uint64
	fakeID   string
	jit      bool
}

type arrayFlags []string

func (a *arrayFlags) String() string { return strings.Join(*a, ",") }
func (a *arrayFlags) Set(v string) error {
	*a = append(*a, v)
	return nil
}

func main() {
	opts := &options{}
	fs := flag.NewFlagSet("arm64chroot", flag.ContinueOnError)
	fs.BoolVar(&opts.strace, "strace", false, "log guest syscalls to stderr")
	fs.BoolVar(&opts.debug, "d", false, "per-instruction trace (very verbose)")
	fs.BoolVar(&opts.debug, "debug", false, "per-instruction trace (very verbose)")
	fs.BoolVar(&opts.jit, "j", false, "translate hot guest code (JIT) instead of interpreting it")
	fs.BoolVar(&opts.jit, "jit", false, "translate hot guest code (JIT) instead of interpreting it")
	fs.BoolVar(&opts.version, "v", false, "show version and exit")
	fs.BoolVar(&opts.version, "version", false, "show version and exit")
	fs.StringVar(&opts.argv0, "0", "", "override argv[0] for the guest program")
	fs.StringVar(&opts.argv0, "argv0", "", "override argv[0] for the guest program")
	fs.StringVar(&opts.workDir, "w", "/", "guest working directory")
	fs.StringVar(&opts.workDir, "work-dir", "/", "guest working directory")
	fs.StringVar(&opts.fakeID, "u", "", "present a fake identity (uid[:gid])")
	fs.StringVar(&opts.fakeID, "fake-id", "", "present a fake identity (uid[:gid])")
	fs.Uint64Var(&opts.maxInsns, "max-insns", 0, "stop after this many guest instructions (diagnostic)")
	fs.Var(&opts.env, "E", "set a guest environment variable VAR=VAL (repeatable)")
	fs.Var(&opts.bind, "b", "bind a host directory into the guest (SRC:DST) (repeatable)")
	fs.Var(&opts.bind, "bind", "bind a host directory into the guest (SRC:DST) (repeatable)")

	fs.Usage = func() { usage(fs) }
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if opts.version {
		fmt.Println(Version)
		return
	}
	if fs.NArg() < 2 {
		usage(fs)
		os.Exit(2)
	}

	rootfs := fs.Arg(0)
	program := fs.Arg(1)
	args := fs.Args()[2:]

	if abs, err := filepath.Abs(rootfs); err == nil {
		rootfs = abs
	}
	if st, err := os.Stat(rootfs); err != nil || !st.IsDir() {
		fmt.Fprintf(os.Stderr, "arm64chroot: %s: not a directory\n", rootfs)
		os.Exit(1)
	}

	// The guest's generic timer runs off the host's monotonic clock, at the
	// frequency the guest sees in CNTFRQ_EL0 -- exactly what src/loop.c's
	// gt_count does. Its origin is the host's boot time (read once from
	// /proc/uptime) rather than the process start, so a guest that compares
	// CNTVCT against CLOCK_MONOTONIC sees the same numbers it would natively.
	bootNS := hostBootNanos()
	start := time.Now()
	core.GTCount = func(c *core.CPU, virt bool) uint64 {
		ns := bootNS + time.Since(start).Nanoseconds()
		return uint64(ns) * 3 / 125 // ns -> 24 MHz ticks
	}

	t := linux.NewTask(rootfs)
	t.WorkDir = opts.workDir
	if opts.strace {
		t.Strace = os.Stderr
	}
	t.Warn = os.Stderr
	if opts.debug {
		t.CPU.Trace = os.Stderr
	}
	if opts.jit {
		t.JIT = jit.New()
	}
	// stdin/stdout/stderr are the emulator's own, and the guest shares them.
	t.Own(-1)

	argv := make([]string, 0, len(args)+1)
	argv0 := program
	if opts.argv0 != "" {
		argv0 = opts.argv0
	}
	argv = append(argv, argv0)
	argv = append(argv, args...)

	envp := defaultEnv()
	for _, e := range opts.env {
		envp = setEnv(envp, e)
	}

	l := &elf.Loader{AS: t.AS, Resolve: t.Resolve}
	entry, sp, err := l.LoadProgram(program, argv, envp, 8<<20)
	if err != nil {
		fmt.Fprintf(os.Stderr, "arm64chroot: %s: %v\n", program, err)
		os.Exit(1)
	}

	c := t.CPU
	c.Reset(entry, 0)
	// The reset state the guest starts in (src/core/cpu.c cpu_reset): SPSel=1
	// and every exception masked. SPSel is read back as 1 by MRS and changes
	// nothing at EL0, where SP is always SP_EL0 -- the stack the loader laid
	// out below.
	c.SPSel = 1
	c.SPEl[0] = sp
	c.DAIF = core.PSD | core.PSA | core.PSI | core.PSF // all masked at reset
	if opts.maxInsns != 0 {
		c.StepHook = func(c *core.CPU, insn uint32) {
			if c.ICount >= opts.maxInsns {
				c.Stop = true
			}
		}
	}

	code := t.Run()
	if opts.debug {
		c.Dump(os.Stderr)
	}
	if t.JIT != nil {
		fmt.Fprintf(os.Stderr, "arm64chroot: jit %d blocks compiled, %d block exits\n",
			t.JIT.Blocks, t.JIT.Exits)
	}
	os.Exit(code)
}

// hostBootNanos is the host's CLOCK_MONOTONIC reading (nanoseconds since
// boot), which is what the guest's generic timer is measured from. /proc/uptime
// is the only portable way to read it; if it is unavailable the timer simply
// starts at zero, which is still monotonic from the guest's point of view.
func hostBootNanos() int64 {
	b, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(b))
	if len(fields) == 0 {
		return 0
	}
	up, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}
	return int64(up * 1e9)
}

// defaultEnv is the environment the guest starts with.
//
// The guest does not inherit the host environment: TERM and COLORTERM pass
// through (a program that cannot tell what the terminal is draws garbage),
// PATH and HOME get the values a root shell in a chroot expects, and -E
// overrides any of them.
func defaultEnv() []string {
	env := []string{
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"HOME=/root",
	}
	for _, k := range []string{"TERM", "COLORTERM"} {
		if v, ok := os.LookupEnv(k); ok {
			env = append(env, k+"="+v)
		}
	}
	return env
}

func setEnv(env []string, kv string) []string {
	name := kv
	if i := strings.IndexByte(kv, '='); i >= 0 {
		name = kv[:i]
	}
	for i, e := range env {
		if strings.HasPrefix(e, name+"=") {
			env[i] = kv
			return env
		}
	}
	return append(env, kv)
}

func usage(fs *flag.FlagSet) {
	fmt.Fprintf(os.Stderr, "%s\n\n", Version)
	fmt.Fprintf(os.Stderr, "usage: arm64chroot [options] <rootfs> <program> [args...]\n\n")
	fs.PrintDefaults()
}

#!/usr/bin/env python3
"""Build a static AArch64 ELF from assembly text, so the Go port can be run
against the same binaries as the C emulator it was translated from.

There is no AArch64 toolchain in the build sandbox, so this assembles with
keystone (pip install keystone-engine) and writes the ELF by hand: one RWX
PT_LOAD, code at 0x401000, data at 0x402000.

A test program is a list of snippets. Each snippet is a list of items; an item
is either assembly text or a raw 32-bit instruction word (an int) for the
handful of encodings keystone does not know (FEAT_FLAGM, FEAT_LSE). Every
snippet leaves one 64-bit result in x0; the harness stores each result into
the data page and writes the whole block to fd 1, so a differential run is a
byte-for-byte comparison of the two emulators' stdout.

Usage: mkelf.py <out.elf> [<group> ...]
"""

import re
import struct
import sys

try:
    from keystone import Ks, KS_ARCH_ARM64, KS_MODE_LITTLE_ENDIAN
except ImportError:
    sys.exit("mkelf: needs keystone-engine (pip install --break-system-packages keystone-engine)")

BASE = 0x400000
CODE_VA = 0x401000
DATA_VA = 0x402000
CODE_OFF = 0x1000
DATA_OFF = 0x4000   # the corpus grew past a single page
IMAGE = 0x3000

SCRATCH = DATA_VA + 0x100   # scratch doubleword for the load/store corpus
BUF = DATA_VA + 0x300       # results buffer

_KS = None


_BIG_IMM = re.compile(r"\bmov\s+(x\d+),\s*#(0x[0-9a-fA-F]+|\d+)")


def expand_imms(text):
    """Keystone only accepts a `mov xN, #imm` that fits MOVZ/MOVN or the ORR
    bitmask form; anything wider has to be spelled out with MOVZ/MOVK."""
    def repl(m):
        val = int(m.group(2), 0) & 0xFFFFFFFFFFFFFFFF
        if val <= 0xFFFF:
            return m.group(0)
        return movabs(int(m.group(1)[1:]), val)
    return _BIG_IMM.sub(repl, text)


def asm(text):
    global _KS
    if _KS is None:
        _KS = Ks(KS_ARCH_ARM64, KS_MODE_LITTLE_ENDIAN)
    text = expand_imms(text)
    try:
        enc, _n = _KS.asm(text)
    except Exception as e:
        raise SystemExit("mkelf: assembly failed (%s): %s" % (e, text))
    if enc is None:
        raise SystemExit("mkelf: assembly failed: " + text)
    return bytes(enc)


def movabs(reg, value):
    """Load a 64-bit constant into x<reg> with MOVZ/MOVK; returns asm text."""
    value &= 0xFFFFFFFFFFFFFFFF
    out = ["movz x%d, #0x%x" % (reg, value & 0xFFFF)]
    for shift in (16, 32, 48):
        part = (value >> shift) & 0xFFFF
        if part:
            out.append("movk x%d, #0x%x, lsl #%d" % (reg, part, shift))
    return "; ".join(out)


# --- raw instruction words -------------------------------------------------
# Keystone has no FEAT_FLAGM or FEAT_LSE support; these are the encodings the
# C emulator's decode.c recognises (the same fields the Go port decodes).
def rmif(rn, imm6, mask):
    return 0xBA000000 | ((imm6 >> 1) << 16) | ((imm6 & 1) << 15) | (rn << 5) | mask


def setf8(rn):
    return 0x3A000000 | 0x800 | (rn << 5) | 0x0D


def mrs(rt, op0, op1, crn, crm, op2):
    """MRS <Xt>, <system register> by (op0,op1,CRn,CRm,op2) fields."""
    return 0xD5000000 | 0x00200000 | (op0 << 19) | (op1 << 16) | (crn << 12) | (crm << 8) | (op2 << 5) | rt


CFINV = 0xD500401F
XAFLAG = 0xD500403F
AXFLAG = 0xD500405F


def ldop(opc, rs, rn, rt):
    """LDADD/LDCLR/LDEOR/LDSET family, 64-bit, no acquire/release."""
    return 0xF8200000 | (opc << 12) | (rs << 16) | (rn << 5) | rt


SWP = lambda rs, rn, rt: 0xF8208000 | (rs << 16) | (rn << 5) | rt
CAS = lambda rs, rn, rt: 0xC8A00000 | (rs << 16) | (rn << 5) | rt


def S(*items):
    """Flatten a snippet: strings are assembled, ints are raw words."""
    return list(items)


def LBL(name):
    """A label: marks the address B/ADR below can jump to."""
    return ("label", name)


def ADR(reg, name):
    """ADR x<reg>, <label>: the label's page-aligned-ish address -> the reg."""
    return ("adr", reg, name)


def B(name, cond=None):
    """B/B.cond to a label (cond None = unconditional)."""
    return ("b", cond, name)


def encode(items):
    """Assemble a snippet, resolving labels in a second pass.

    An item is not always four bytes (a `movz; movk` pair is eight), so both
    passes measure the assembled size rather than assuming it.
    """
    def size(it):
        if isinstance(it, int):
            return 4
        if isinstance(it, str):
            return len(asm(it))
        return 0 if it[0] == "label" else 4

    offs, off = [0] * len(items), 0
    for i, it in enumerate(items):
        offs[i] = off
        off += size(it)

    labels = {}
    for i, it in enumerate(items):
        if isinstance(it, tuple) and it[0] == "label":
            labels[it[1]] = offs[i]

    out = bytearray()
    for i, it in enumerate(items):
        if isinstance(it, int):
            out += struct.pack("<I", it)
        elif isinstance(it, str):
            out += asm(it)
        elif it[0] == "label":
            continue
        elif it[0] == "adr":
            _, reg, name = it
            imm = labels[name] - offs[i]
            word = 0x10000000 | ((imm & 3) << 29) | (((imm >> 2) & 0x7FFFF) << 5) | reg
            out += struct.pack("<I", word & 0xFFFFFFFF)
        else:  # it[0] == "b"
            _, cond, name = it
            imm = labels[name] - offs[i]
            if cond is None:
                word = 0x14000000 | ((imm // 4) & 0x3FFFFFF)
            else:
                word = 0x54000000 | (((imm // 4) & 0x7FFFF) << 5) | cond
            out += struct.pack("<I", word & 0xFFFFFFFF)
    return bytes(out)


def build(snippets, data=b""):
    """Assemble a program that runs every snippet and dumps its results."""
    code = b""
    code += asm(movabs(9, BUF))
    for snip in snippets:
        code += encode(snip)
        code += asm("str x0, [x9], #8")
    nres = len(snippets)
    code += asm("; ".join([
        "mov x0, #1",
        movabs(1, BUF),
        "mov x2, #%d" % (nres * 8),
        "mov x8, #64",   # write
        "svc #0",
        "mov x0, #0",
        "mov x8, #93",   # exit
        "svc #0",
    ]))
    if len(code) > DATA_OFF - CODE_OFF:
        raise SystemExit("mkelf: program too large (%d bytes)" % len(code))

    img = bytearray(IMAGE)
    img[CODE_OFF:CODE_OFF + len(code)] = code
    img[DATA_OFF:DATA_OFF + len(data)] = data

    eh = struct.pack(
        "<16sHHIQQQIHHHHHH",
        b"\x7fELF" + bytes([2, 1, 1, 0]) + b"\0" * 8,
        2,        # e_type = ET_EXEC
        183,      # e_machine = EM_AARCH64
        1,        # e_version
        CODE_VA,  # e_entry
        64,       # e_phoff
        0,        # e_shoff
        0,        # e_flags
        64, 56, 1, 64, 0, 0,
    )
    ph = struct.pack("<IIQQQQQQ",
                     1,          # p_type = PT_LOAD
                     7,          # p_flags = R|W|X
                     0,          # p_offset
                     BASE,       # p_vaddr
                     BASE,       # p_paddr
                     IMAGE,      # p_filesz
                     IMAGE,      # p_memsz
                     0x1000)     # p_align
    img[0:64] = eh
    img[64:64 + 56] = ph
    return bytes(img)


# ---------------------------------------------------------------------------
# The corpus: one snippet per group of instructions the decoder owns.
# ---------------------------------------------------------------------------

SNIPPETS = {
    # data processing - immediate
    "movwide": [S("movz x0, #0x1234; movk x0, #0xabcd, lsl #16; movk x0, #0xef01, lsl #32"),
                S("movn x0, #0"),
                S("mov x0, #0xffffffffffffffff")],
    "addsub_imm": [S("mov x0, #10; add x0, x0, #5; sub x0, x0, #3"),
                   S("mov x0, #1; add x0, x0, #0x123, lsl #12"),
                   S("mov x0, #5; adds x0, x0, #0; mrs x0, nzcv"),
                   S("mov x0, #0; subs x0, x0, #1; mrs x0, nzcv"),
                   S("mov x0, #0x7fffffff; adds x0, x0, #1; mrs x0, nzcv")],
    "logical_imm": [S(movabs(0, 0xF0F0F0F0F0F0F0F0), "and x0, x0, #0x0f0f0f0f0f0f0f0f"),
                    S("mov x0, #1; orr x0, x0, #0xaaaaaaaaaaaaaaaa"),
                    S("mov x0, #0xffffffffffffffff; eor x0, x0, #0xff"),
                    S("mov x0, #0xff; ands x0, x0, #0xf0; mrs x0, nzcv")],
    "bitfield": [S(movabs(0, 0xDEADBEEF), "ubfx x0, x0, #4, #8"),
                 S("mov x0, #0x80; sbfx x0, x0, #7, #1"),
                 S("mov x0, #0; mov x1, #0xffff; bfi x0, x1, #8, #16"),
                 S(movabs(0, 0x1122334455667788), movabs(1, 0x99AABBCCDDEEFF00),
                   "extr x0, x0, x1, #32"),
                 S("mov x0, #0xff; lsl x0, x0, #8; lsr x0, x0, #4; asr x0, x0, #2")],
    "adr": [S("adr x0, .+8"), S("adrp x0, .")],

    # data processing - register
    "shift_reg": [S("mov x0, #1; mov x1, #63; lsl x0, x0, x1"),
                  S("mov x0, #0x8000000000000000; mov x1, #63; lsr x0, x0, x1"),
                  S("mov x0, #0x8000000000000000; mov x1, #63; asr x0, x0, x1"),
                  S("mov x0, #1; mov x1, #4; ror x0, x0, x1"),
                  S("mov x0, #0xff; mov x1, #0x0f; bic x0, x0, x1"),
                  S("mov x0, #0xf0; mov x1, #0x0f; orn x0, x0, x1"),
                  S("mov x0, #0; mov x1, #0; ands x0, x0, x1; mrs x0, nzcv")],
    "addsub_reg": [S("mov x0, #20; mov x1, #22; add x0, x0, x1"),
                   S("mov x0, #20; mov x1, #22; sub x0, x0, x1"),
                   S("mov x0, #20; mov x1, #1; add x0, x0, x1, lsl #3"),
                   S("mov x0, #0xffffffff; mov x1, #1; add x0, x0, w1, uxtb #0"),
                   S("mov x0, #5; mov x1, #3; adds x0, x0, x1; mrs x0, nzcv")],
    "muldiv": [S("mov x0, #7; mov x1, #9; mul x0, x0, x1"),
               S("mov x0, #7; mov x1, #9; mov x2, #100; madd x0, x0, x1, x2"),
               S("mov x0, #7; mov x1, #9; mov x2, #100; msub x0, x0, x1, x2"),
               S(movabs(0, 123456789), movabs(1, 987654321), "smulh x0, x0, x1"),
               S(movabs(0, 123456789), movabs(1, 987654321), "umulh x0, x0, x1"),
               S("mov x0, #100; mov x1, #7; udiv x0, x0, x1"),
               S("mov x0, #100; mov x1, #7; sdiv x0, x0, x1"),
               S("mov x0, #100; mov x1, #0; udiv x0, x0, x1"),
               S(movabs(0, 0x40000000), movabs(1, 0x40000000), "mov x2, #0",
                 "smaddl x0, w0, w1, x2"),
               S(movabs(0, 0xFFFFFFFF), movabs(1, 0xFFFFFFFF), "mov x2, #0",
                 "umaddl x0, w0, w1, x2")],
    "cond": [S("mov x0, #1; mov x1, #2; cmp x0, x1; csel x0, x0, x1, lo"),
             S("mov x0, #1; mov x1, #2; cmp x0, x1; csinc x0, x0, x1, eq"),
             S("mov x0, #1; mov x1, #2; cmp x0, x1; csinv x0, x0, x1, hi"),
             S("mov x0, #1; mov x1, #2; cmp x0, x1; csneg x0, x0, x1, lt"),
             S("mov x0, #5; mov x1, #5; cmp x0, x1; ccmp x0, x1, #0, eq; mrs x0, nzcv"),
             S("mov x0, #5; mov x1, #5; cmp x0, x1; ccmn x0, x1, #0, ne; mrs x0, nzcv")],
    "dp1src": [S(movabs(0, 0x0123456789ABCDEF), "rbit x0, x0"),
               S(movabs(0, 0x0123456789ABCDEF), "rev x0, x0"),
               S(movabs(0, 0x0123456789ABCDEF), "rev16 x0, x0"),
               S(movabs(0, 0x0123456789ABCDEF), "rev32 x0, x0"),
               S(movabs(0, 0x0000FFFF00000000), "clz x0, x0"),
               S(movabs(0, 0xFFFFFFFFFFFFFFF0), "cls x0, x0")],
    "dp2src": [S("mov x0, #8; mov x1, #3; lslv x0, x0, x1"),
               S("mov x0, #0xff; mov x1, #4; lsrv x0, x0, x1"),
               S("mov x0, #0xff; mov x1, #4; asrv x0, x0, x1"),
               S("mov x0, #0xff; mov x1, #4; rorv x0, x0, x1")],
    "crc32": [S("mov x0, #0; mov x1, #0x41; crc32b w0, w0, w1"),
              S("mov x0, #0; mov x1, #0x4142; crc32h w0, w0, w1"),
              S("mov x0, #0", movabs(1, 0x41424344), "crc32w w0, w0, w1"),
              S("mov x0, #0", movabs(1, 0x4142434445464748), "crc32x w0, w0, x1"),
              S("mov x0, #0; mov x1, #0x41; crc32cb w0, w0, w1")],
    "flagm": [S("mov x0, #0", CFINV, "mrs x0, nzcv"),
              S("mov x0, #0; msr nzcv, x0; mrs x0, nzcv"),
              S(movabs(0, 0xF0000000), rmif(0, 4, 0xF), "mrs x0, nzcv"),
              S("mov x0, #0x80", setf8(0), "mrs x0, nzcv"),
              S("mov x0, #0x2; msr nzcv, x0", XAFLAG, "mrs x0, nzcv"),
              S("mov x0, #0x2; msr nzcv, x0", AXFLAG, "mrs x0, nzcv")],

    # loads and stores
    "ldst": [S(movabs(1, SCRATCH), movabs(2, 0x1122334455667788),
               "str x2, [x1]; ldr x0, [x1]"),
             S(movabs(1, SCRATCH), "mov x2, #0x11223344; str w2, [x1]; ldrsw x0, [x1]"),
             S(movabs(1, SCRATCH), "mov x2, #0x1ff; strh w2, [x1]; ldrh w0, [x1]"),
             S(movabs(1, SCRATCH), "mov x2, #0xff; strb w2, [x1]; ldrb w0, [x1]"),
             S(movabs(1, SCRATCH), "mov x2, #0xffffffff; str w2, [x1]; ldr x0, [x1]"),
             S(movabs(1, SCRATCH), "mov x2, #0x1111; mov x3, #0x2222; "
               "stp x2, x3, [x1]; ldp x0, x4, [x1]; mov x0, x4"),
             S(movabs(1, SCRATCH + 8), "mov x2, #0x1111; stur x2, [x1, #-8]; ldur x0, [x1, #-8]"),
             S(movabs(1, SCRATCH), "mov x2, #0x1111; str x2, [x1, #8]!; ldr x0, [x1], #8"),
             S(movabs(1, SCRATCH), "mov x2, #0x1111; mov x3, #8; "
               "str x2, [x1, x3, lsl #3]; ldr x0, [x1, x3, lsl #3]")],
    "atomic": [S(movabs(1, SCRATCH), "mov x2, #0; str x2, [x1]; mov x2, #5",
                 ldop(0, 2, 1, 2), "mov x0, x2"),
               S(movabs(1, SCRATCH), "mov x2, #0x0f; str x2, [x1]; mov x2, #0xf0",
                 ldop(3, 2, 1, 2), "ldr x0, [x1]"),
               S(movabs(1, SCRATCH), "mov x2, #0xff; str x2, [x1]; mov x2, #0x0f",
                 ldop(1, 2, 1, 2), "ldr x0, [x1]"),
               S(movabs(1, SCRATCH), "mov x2, #7; str x2, [x1]; mov x2, #9",
                 SWP(2, 1, 2)),
               S(movabs(1, SCRATCH), "mov x2, #7; str x2, [x1]; mov x2, #7; mov x3, #9",
                 CAS(2, 1, 3), "ldr x0, [x1]"),
               S(movabs(1, SCRATCH), "mov x2, #7; str x2, [x1]; mov x2, #3; "
                 "stlr x2, [x1]; ldar x0, [x1]"),
               S(movabs(1, SCRATCH), "mov x2, #7; str x2, [x1]; ldxr x0, [x1]; "
                 "mov x3, #9; stxr w4, x3, [x1]; add x0, x0, x4")],

    # branches
    "branch": [S("mov x0, #0; mov x1, #3; cmp x1, #3; b.eq .+8; mov x0, #1; mov x0, #42"),
               S("mov x0, #0; mov x1, #0; cbz x1, .+8; mov x0, #1; mov x0, #43"),
               S("mov x0, #0; mov x1, #8; tbz x1, #3, .+8; mov x0, #1; mov x0, #44"),
               S("bl .+8; b .+12; mov x0, #45; ret")],

    # system
    "sysreg": [S("mrs x0, cntfrq_el0"),
               S("mrs x0, midr_el1"),
               S("mrs x0, currentel"),
               S("mrs x0, dczid_el0"),
               S("mrs x0, id_aa64isar0_el1"),
               S("mov x0, #0; msr tpidr_el0, x0; mrs x0, tpidr_el0"),
               S("mrs x0, cntvct_el0")],

    # stack
    "stack": [S("mov x0, #1; str x0, [sp, #-16]!; ldr x0, [sp], #16"),
              S("sub sp, sp, #32; mov x0, #7; str x0, [sp, #8]; ldr x0, [sp, #8]; add sp, sp, #32")],

    # a loop, to exercise the fetch cache and the branch paths together
    "loop": [S("mov x0, #0; mov x1, #10; add x0, x0, x1; subs x1, x1, #1; b.ne .-8")],

    # Every system register the emulator exposes, read back through MRS. The
    # two names keystone does not know are spelled out as raw words.
    "ids": [S("mrs x0, " + n) for n in [
        "midr_el1", "mpidr_el1", "revidr_el1", "ctr_el0", "dczid_el0",
        "id_aa64pfr0_el1", "id_aa64pfr1_el1", "id_aa64dfr0_el1", "id_aa64dfr1_el1",
        "id_aa64isar0_el1", "id_aa64isar1_el1", "id_aa64mmfr0_el1", "id_aa64mmfr1_el1",
        "sctlr_el1", "actlr_el1", "cpacr_el1", "ttbr0_el1", "ttbr1_el1", "tcr_el1",
        "mair_el1", "amair_el1", "vbar_el1", "afsr0_el1", "afsr1_el1", "esr_el1",
        "far_el1", "par_el1", "contextidr_el1", "tpidr_el1", "tpidr_el0", "tpidrro_el0",
        "mdscr_el1", "currentel", "spsel", "daif", "fpcr", "fpsr", "spsr_el1",
        "elr_el1", "sp_el0", "cntfrq_el0", "cntp_ctl_el0", "cntp_cval_el0",
        "cntv_ctl_el0", "cntv_cval_el0",
    ]] + [S(mrs(0, 3, 0, 0, 6, 2)),    # ID_AA64ISAR2_EL1
          S(mrs(0, 3, 0, 0, 7, 2))],   # ID_AA64MMFR2_EL1

    # Signal delivery: install a handler with rt_sigaction, raise the signal
    # with kill(getpid(), sig), and count the handler's entries in memory. The
    # handler returns through lr, which the port points at the rt_sigreturn
    # trampoline, so a correct run also exercises frame build and sigreturn.
    "signal": [
        S(movabs(11, DATA_VA + 0xE00),           # struct sigaction
          movabs(12, DATA_VA + 0xF00),           # handler entry counter
          "str xzr, [x12]",
          ADR(13, "h1"),
          "str x13, [x11]",                      # sa_handler
          "str xzr, [x11, #8]",                  # sa_flags
          "str xzr, [x11, #16]",                 # sa_restorer
          "str xzr, [x11, #24]",                 # sa_mask
          "mov x0, #10",                         # SIGUSR1
          "mov x1, x11",
          "mov x2, #0",
          "mov x3, #8",
          "mov x8, #134",                        # rt_sigaction
          "svc #0",
          "mov x8, #172",                        # getpid
          "svc #0",
          "mov x1, #10",
          "mov x8, #129",                        # kill
          "svc #0",
          "ldr x0, [x12]",                       # result: handler entries
          B("h1end"),
          LBL("h1"),
          "ldr x10, [x12]",
          "add x10, x10, #1",
          "str x10, [x12]",
          "ret",
          LBL("h1end")),
        # A blocked signal waits: the handler must not have run yet.
        S(movabs(11, DATA_VA + 0xE40),
          movabs(12, DATA_VA + 0xF08),
          "str xzr, [x12]",
          ADR(13, "h2"),
          "str x13, [x11]",
          "str xzr, [x11, #8]",
          "str xzr, [x11, #16]",
          "str xzr, [x11, #24]",
          "mov x0, #12",                         # SIGUSR2
          "mov x1, x11",
          "mov x2, #0",
          "mov x3, #8",
          "mov x8, #134",
          "svc #0",
          "mov x0, #2048",                       # 1 << (SIGUSR2 - 1)
          "str x0, [sp, #-16]!",
          "mov x0, #0",                          # SIG_BLOCK
          "mov x1, sp",
          "mov x2, #0",
          "mov x3, #8",
          "mov x8, #135",                        # rt_sigprocmask
          "svc #0",
          "add sp, sp, #16",
          "mov x8, #172",
          "svc #0",
          "mov x1, #12",
          "mov x8, #129",
          "svc #0",
          "ldr x0, [x12]",                       # result: 0, it is blocked
          B("h2end"),
          LBL("h2"),
          "ldr x10, [x12]",
          "add x10, x10, #1",
          "str x10, [x12]",
          "ret",
          LBL("h2end")),
        # ... and runs the moment the guest unblocks it.
        S(movabs(12, DATA_VA + 0xF08),
          "mov x0, #2048",                       # 1 << (SIGUSR2 - 1)
          "str x0, [sp, #-16]!",
          "mov x0, #1",                          # SIG_UNBLOCK
          "mov x1, sp",
          "mov x2, #0",
          "mov x3, #8",
          "mov x8, #135",
          "svc #0",
          "add sp, sp, #16",
          "ldr x0, [x12]"),                      # result: 1
        # SIG_IGN: no handler, nothing happens.
        S(movabs(11, DATA_VA + 0xE80),
          movabs(12, DATA_VA + 0xF10),
          "str xzr, [x12]",
          ADR(13, "h4"),
          "str x13, [x11]",
          "str xzr, [x11, #8]",
          "str xzr, [x11, #16]",
          "str xzr, [x11, #24]",
          "mov x0, #10",
          "mov x1, x11",
          "mov x2, #0",
          "mov x3, #8",
          "mov x8, #134",
          "svc #0",
          "mov x13, #1",                         # SIG_IGN
          "str x13, [x11]",
          "mov x0, #10",
          "mov x1, x11",
          "mov x2, #0",
          "mov x3, #8",
          "mov x8, #134",
          "svc #0",
          "mov x8, #172",
          "svc #0",
          "mov x1, #10",
          "mov x8, #129",
          "svc #0",
          "ldr x0, [x12]",                       # result: 0
          B("h4end"),
          LBL("h4"),
          "ldr x10, [x12]",
          "add x10, x10, #1",
          "str x10, [x12]",
          "ret",
          LBL("h4end")),
    ],

    # Scalar floating point. The operands are materialized in a GPR and moved
    # with FMOV, so the bit patterns are exact and the comparison is about the
    # arithmetic, not about how each emulator spells a float constant.
    "fp": [
        # single precision: 1.0 + 2.0
        S("mov x2, #0x3f800000; fmov s0, w2; mov x3, #0x40000000; fmov s1, w3; "
          "fadd s0, s0, s1; fmov w0, s0"),
        S("mov x2, #0x40490fdb; fmov s0, w2; mov x3, #0x3fc90fdb; fmov s1, w3; "
          "fsub s0, s0, s1; fmov w0, s0"),
        S("mov x2, #0x40490fdb; fmov s0, w2; mov x3, #0x40000000; fmov s1, w3; "
          "fmul s0, s0, s1; fmov w0, s0"),
        S("mov x2, #0x40490fdb; fmov s0, w2; mov x3, #0x40800000; fmov s1, w3; "
          "fdiv s0, s0, s1; fmov w0, s0"),
        S("mov x2, #0x40490fdb; fmov s0, w2; fsqrt s0, s0; fmov w0, s0"),
        S("mov x2, #0xc0490fdb; fmov s0, w2; fabs s0, s0; fmov w0, s0"),
        S("mov x2, #0xc0490fdb; fmov s0, w2; fneg s0, s0; fmov w0, s0"),
        S("mov x2, #0x40490fdb; fmov s0, w2; mov x3, #0xc0000000; fmov s1, w3; "
          "fmax s0, s0, s1; fmov w0, s0"),
        S("mov x2, #0x40490fdb; fmov s0, w2; mov x3, #0xc0000000; fmov s1, w3; "
          "fmin s0, s0, s1; fmov w0, s0"),
        S("mov x2, #0x7fc00000; fmov s0, w2; mov x3, #0x40490fdb; fmov s1, w3; "
          "fmaxnm s0, s0, s1; fmov w0, s0"),
        S("mov x2, #0x7fc00000; fmov s0, w2; mov x3, #0x40490fdb; fmov s1, w3; "
          "fmax s0, s0, s1; fmov w0, s0"),
        S("mov x2, #0x40490fdb; fmov s0, w2; mov x3, #0x40000000; fmov s1, w3; "
          "fnmul s0, s0, s1; fmov w0, s0"),
        # double precision
        S(movabs(2, 0x3ff0000000000000), "fmov d0, x2; " + movabs(3, 0x4000000000000000),
          "fmov d1, x3; fadd d0, d0, d1; fmov x0, d0"),
        S(movabs(2, 0x400921fb54442d18), "fmov d0, x2; " + movabs(3, 0x4008000000000000),
          "fmov d1, x3; fmul d0, d0, d1; fmov x0, d0"),
        S(movabs(2, 0x400921fb54442d18), "fmov d0, x2; " + movabs(3, 0x4008000000000000),
          "fmov d1, x3; fdiv d0, d0, d1; fmov x0, d0"),
        S(movabs(2, 0x400921fb54442d18), "fmov d0, x2; fsqrt d0, d0; fmov x0, d0"),
        S(movabs(2, 0x400921fb54442d18), "fmov d0, x2; fabs d0, d0; fmov x0, d0"),
        S(movabs(2, 0xc00921fb54442d18), "fmov d0, x2; fneg d0, d0; fmov x0, d0"),
        # conversions
        S(movabs(2, 0x400921fb54442d18), "fmov d0, x2; fcvt s0, d0; fmov w0, s0"),
        S("mov x2, #0x40490fdb; fmov s0, w2; fcvt d0, s0; fmov x0, d0"),
        S("mov x1, #-7; scvtf s0, w1; fmov w0, s0"),
        S("mov x1, #-7; scvtf d0, x1; fmov x0, d0"),
        S("mov x1, #0xffffffffffffffff; ucvtf d0, x1; fmov x0, d0"),
        S(movabs(2, 0x400921fb54442d18), "fmov d0, x2; fcvtzs w0, d0"),
        S(movabs(2, 0x400921fb54442d18), "fmov d0, x2; fcvtzu x0, d0"),
        S(movabs(2, 0xc00921fb54442d18), "fmov d0, x2; fcvtns x0, d0"),
        S(movabs(2, 0xc00921fb54442d18), "fmov d0, x2; fcvtps x0, d0"),
        S(movabs(2, 0xc00921fb54442d18), "fmov d0, x2; fcvtms x0, d0"),
        S(movabs(2, 0x400921fb54442d18), "fmov d0, x2; fcvtzu w0, d0"),
        # rounding to integral
        S(movabs(2, 0x400921fb54442d18), "fmov d0, x2; frintn d0, d0; fmov x0, d0"),
        S(movabs(2, 0x400921fb54442d18), "fmov d0, x2; frintp d0, d0; fmov x0, d0"),
        S(movabs(2, 0x400921fb54442d18), "fmov d0, x2; frintz d0, d0; fmov x0, d0"),
        # compare and conditional select: results are the NZCV the compare set
        S(movabs(2, 0x400921fb54442d18), "fmov d0, x2; " + movabs(3, 0x4008000000000000),
          "fmov d1, x3; fcmp d0, d1; cset x0, gt"),
        S(movabs(2, 0x4008000000000000), "fmov d0, x2; " + movabs(3, 0x400921fb54442d18),
          "fmov d1, x3; fcmp d0, d1; cset x0, mi"),
        S(movabs(2, 0x400921fb54442d18), "fmov d0, x2; " + movabs(3, 0x4008000000000000),
          "fmov d1, x3; fcmp d0, d1; fcsel d0, d0, d1, gt; fmov x0, d0"),
        S(movabs(2, 0x400921fb54442d18), "fmov d0, x2; " + movabs(3, 0x4008000000000000),
          "fmov d1, x3; fcmp d0, d1; fcsel d0, d0, d1, le; fmov x0, d0"),
        # fused multiply-add
        S(movabs(2, 0x4008000000000000), "fmov d0, x2; " + movabs(3, 0x4000000000000000),
          "fmov d1, x3; " + movabs(4, 0x3ff0000000000000), "fmov d2, x4; "
          "fmadd d0, d1, d2, d0; fmov x0, d0"),
        S(movabs(2, 0x4008000000000000), "fmov d0, x2; " + movabs(3, 0x4000000000000000),
          "fmov d1, x3; " + movabs(4, 0x3ff0000000000000), "fmov d2, x4; "
          "fmsub d0, d1, d2, d0; fmov x0, d0"),
        S(movabs(2, 0x4008000000000000), "fmov d0, x2; " + movabs(3, 0x4000000000000000),
          "fmov d1, x3; " + movabs(4, 0x3ff0000000000000), "fmov d2, x4; "
          "fnmadd d0, d1, d2, d0; fmov x0, d0"),
        S(movabs(2, 0x4008000000000000), "fmov d0, x2; " + movabs(3, 0x4000000000000000),
          "fmov d1, x3; " + movabs(4, 0x3ff0000000000000), "fmov d2, x4; "
          "fnmsub d0, d1, d2, d0; fmov x0, d0"),
        # FPCR.FZ: a denormal operand flushes to zero. Without it the sum is
        # the denormal itself; with it, it is the other operand unchanged.
        S("msr fpcr, xzr; " + movabs(2, 0x0000000000000001), "fmov d0, x2; "
          "mov x3, #0x3ff0000000000000; fmov d1, x3; fadd d0, d0, d1; fmov x0, d0"),
        # NaN propagation: a quiet NaN operand wins over the other operand.
        S(movabs(2, 0x7ff8000000000000), "fmov d0, x2; " + movabs(3, 0x4008000000000000),
          "fmov d1, x3; fadd d0, d0, d1; fmov x0, d0"),
    ],

    # Advanced SIMD integer. Vectors are built with LD1/ST1 (single-structure
    # forms the loader and store path already have) and read back through
    # UMOV, so the comparison is the lane arithmetic.
    #
    # v0 = 01 02 03 04 05 06 07 08 | 09 0a 0b 0c 0d 0e 0f 10
    # v1 = 10 20 30 40 50 60 70 80 | 90 a0 b0 c0 d0 e0 f0 08
    "simd": [
        S(movabs(1, SCRATCH), "mov x2, #0x0807060504030201; " + movabs(3, 0x100f0e0d0c0b0a09),
          "mov v0.d[0], x2; mov v0.d[1], x3; "
          + movabs(2, 0x8070605040302010), "mov v1.d[0], x2; "
          + movabs(3, 0x08f0e0d0c0b0a090), "mov v1.d[1], x3; "
          "add v0.8b, v0.8b, v1.8b; umov x0, v0.d[0]"),
        S(movabs(1, SCRATCH), "mov x2, #0x0807060504030201; " + movabs(3, 0x100f0e0d0c0b0a09),
          "mov v0.d[0], x2; mov v0.d[1], x3; "
          + movabs(2, 0x8070605040302010), "mov v1.d[0], x2; "
          + movabs(3, 0x08f0e0d0c0b0a090), "mov v1.d[1], x3; "
          "add v0.16b, v0.16b, v1.16b; umov x0, v0.d[1]"),
        S(movabs(1, SCRATCH), "mov x2, #0x0807060504030201; " + movabs(3, 0x100f0e0d0c0b0a09),
          "mov v0.d[0], x2; mov v0.d[1], x3; "
          + movabs(2, 0x8070605040302010), "mov v1.d[0], x2; "
          + movabs(3, 0x08f0e0d0c0b0a090), "mov v1.d[1], x3; "
          "sub v0.8h, v0.8h, v1.8h; umov x0, v0.d[0]"),
        S(movabs(1, SCRATCH), "mov x2, #0x0807060504030201; " + movabs(3, 0x100f0e0d0c0b0a09),
          "mov v0.d[0], x2; mov v0.d[1], x3; "
          + movabs(2, 0x8070605040302010), "mov v1.d[0], x2; "
          + movabs(3, 0x08f0e0d0c0b0a090), "mov v1.d[1], x3; "
          "mul v0.8b, v0.8b, v1.8b; umov x0, v0.d[0]"),
        S(movabs(1, SCRATCH), "mov x2, #0x0807060504030201; " + movabs(3, 0x100f0e0d0c0b0a09),
          "mov v0.d[0], x2; mov v0.d[1], x3; "
          + movabs(2, 0x8070605040302010), "mov v1.d[0], x2; "
          + movabs(3, 0x08f0e0d0c0b0a090), "mov v1.d[1], x3; "
          "cmeq v0.8b, v0.8b, v1.8b; umov x0, v0.d[0]"),
        S(movabs(1, SCRATCH), "mov x2, #0x0807060504030201; " + movabs(3, 0x100f0e0d0c0b0a09),
          "mov v0.d[0], x2; mov v0.d[1], x3; "
          + movabs(2, 0x8070605040302010), "mov v1.d[0], x2; "
          + movabs(3, 0x08f0e0d0c0b0a090), "mov v1.d[1], x3; "
          "cmgt v0.8b, v0.8b, v1.8b; umov x0, v0.d[0]"),
        S(movabs(1, SCRATCH), "mov x2, #0x0807060504030201; " + movabs(3, 0x100f0e0d0c0b0a09),
          "mov v0.d[0], x2; mov v0.d[1], x3; "
          + movabs(2, 0x8070605040302010), "mov v1.d[0], x2; "
          + movabs(3, 0x08f0e0d0c0b0a090), "mov v1.d[1], x3; "
          "smax v0.8b, v0.8b, v1.8b; umov x0, v0.d[0]"),
        S(movabs(1, SCRATCH), "mov x2, #0x0807060504030201; " + movabs(3, 0x100f0e0d0c0b0a09),
          "mov v0.d[0], x2; mov v0.d[1], x3; "
          + movabs(2, 0x8070605040302010), "mov v1.d[0], x2; "
          + movabs(3, 0x08f0e0d0c0b0a090), "mov v1.d[1], x3; "
          "umin v0.4s, v0.4s, v1.4s; umov w0, v0.s[1]"),
        S(movabs(1, SCRATCH), "mov x2, #0x0807060504030201; " + movabs(3, 0x100f0e0d0c0b0a09),
          "mov v0.d[0], x2; mov v0.d[1], x3; "
          + movabs(2, 0x8070605040302010), "mov v1.d[0], x2; "
          + movabs(3, 0x08f0e0d0c0b0a090), "mov v1.d[1], x3; "
          "uabd v0.8b, v0.8b, v1.8b; umov x0, v0.d[0]"),
        S(movabs(1, SCRATCH), "mov x2, #0x0807060504030201; " + movabs(3, 0x100f0e0d0c0b0a09),
          "mov v0.d[0], x2; mov v0.d[1], x3; "
          + movabs(2, 0x8070605040302010), "mov v1.d[0], x2; "
          + movabs(3, 0x08f0e0d0c0b0a090), "mov v1.d[1], x3; "
          "sqadd v0.8b, v0.8b, v1.8b; umov x0, v0.d[0]"),
        S(movabs(1, SCRATCH), "mov x2, #0x0807060504030201; " + movabs(3, 0x100f0e0d0c0b0a09),
          "mov v0.d[0], x2; mov v0.d[1], x3; "
          + movabs(2, 0x8070605040302010), "mov v1.d[0], x2; "
          + movabs(3, 0x08f0e0d0c0b0a090), "mov v1.d[1], x3; "
          "uhadd v0.8b, v0.8b, v1.8b; umov x0, v0.d[0]"),
        # logical (whole-register)
        S(movabs(1, SCRATCH), "mov x2, #0x0f0f0f0f0f0f0f0f; " + movabs(3, 0x3333333333333333),
          "mov v0.d[0], x2; mov v0.d[1], x3; "
          "mov x2, #0x5555555555555555; mov v1.d[0], x2; mov v1.d[1], x2; "
          "and v0.16b, v0.16b, v1.16b; umov x0, v0.d[0]"),
        S(movabs(1, SCRATCH), "mov x2, #0x0f0f0f0f0f0f0f0f; " + movabs(3, 0x3333333333333333),
          "mov v0.d[0], x2; mov v0.d[1], x3; "
          "mov x2, #0x5555555555555555; mov v1.d[0], x2; mov v1.d[1], x2; "
          "eor v0.16b, v0.16b, v1.16b; umov x0, v0.d[0]"),
        S(movabs(1, SCRATCH), "mov x2, #0x0f0f0f0f0f0f0f0f; " + movabs(3, 0x3333333333333333),
          "mov v0.d[0], x2; mov v0.d[1], x3; "
          "mov x2, #0x5555555555555555; mov v1.d[0], x2; mov v1.d[1], x2; "
          "orr v0.8b, v0.8b, v1.8b; umov x0, v0.d[0]"),
        # shifts by immediate
        S(movabs(1, SCRATCH), "mov x2, #0x0807060504030201; " + movabs(3, 0x100f0e0d0c0b0a09),
          "mov v0.d[0], x2; mov v0.d[1], x3; shl v0.8h, v0.8h, #3; umov x0, v0.d[0]"),
        S(movabs(1, SCRATCH), "mov x2, #0x0807060504030201; " + movabs(3, 0x100f0e0d0c0b0a09),
          "mov v0.d[0], x2; mov v0.d[1], x3; ushr v0.4s, v0.4s, #4; umov x0, v0.d[0]"),
        S(movabs(1, SCRATCH), "mov x2, #0x0807060504030201; " + movabs(3, 0x100f0e0d0c0b0a09),
          "mov v0.d[0], x2; mov v0.d[1], x3; sshr v0.8h, v0.8h, #2; umov x0, v0.d[0]"),
        S(movabs(1, SCRATCH), "mov x2, #0x0807060504030201; " + movabs(3, 0x100f0e0d0c0b0a09),
          "mov v0.d[0], x2; mov v0.d[1], x3; sqshl v0.8b, v0.8b, #4; umov x0, v0.d[0]"),
        # modified immediate
        S("movi v0.16b, #0x5a; umov x0, v0.d[0]"),
        S("movi v0.4s, #0x1a; umov x0, v0.d[0]"),
        S("movi d0, #0x00ff00ff00ff00ff; fmov x0, d0"),
        S("mvni v0.8h, #0x33; umov x0, v0.d[0]"),
        S(movabs(2, 0x3f8000003f800000), "mov v0.d[0], x2; mov v0.d[1], x2; "
          "fmov v0.4s, #1.0; umov x0, v0.d[0]"),
        # copy: DUP, INS, SMOV, UMOV
        S(movabs(2, 0x0807060504030201), "mov v0.d[0], x2; dup v1.8b, v0.b[3]; umov x0, v1.d[0]"),
        S(movabs(2, 0x0807060504030201), "mov v0.d[0], x2; mov x3, #0x5a; dup v1.16b, w3; "
          "umov x0, v1.d[1]"),
        S(movabs(2, 0x0807060504030201), "mov v0.d[0], x2; " + movabs(3, 0x1122334455667788),
          "mov v1.d[0], x3; ins v0.b[2], v1.b[5]; umov x0, v0.d[0]"),
        S(movabs(2, 0x08070605040302ff), "mov v0.d[0], x2; smov x0, v0.b[0]"),
        S(movabs(2, 0x08070605040302ff), "mov v0.d[0], x2; smov w0, v0.h[1]"),
        S(movabs(1, SCRATCH), "mov x2, #0x0807060504030201; " + movabs(3, 0x100f0e0d0c0b0a09),
          "mov v0.d[0], x2; mov v0.d[1], x3; "
          + movabs(2, 0x8070605040302010), "mov v1.d[0], x2; "
          + movabs(3, 0x08f0e0d0c0b0a090), "mov v1.d[1], x3; "
          "addp v0.16b, v0.16b, v1.16b; umov x0, v0.d[1]"),
        # scalar DUP: MOV Dd, Vn.<T>[index]
        S(movabs(2, 0x0807060504030201), "mov v0.d[0], x2; dup s1, v0.s[1]; fmov w0, s1"),
    ],

    # A loop long enough to go hot: the JIT's reason to exist. One million
    # iterations of an add/compare/branch, which is the shape of every hot
    # loop in real code.
    "hotloop": [S("mov x0, #0; mov x1, #1000000; add x0, x0, x1; subs x1, x1, #1; b.ne .-8")],

    # The generic timer: read twice in a row, so the comparison is about the
    # counter advancing and not about the host's uptime. Flagged volatile.
    "counter": [S("mrs x0, cntvct_el0; mov x1, x0; mrs x0, cntvct_el0; sub x0, x0, x1"),
                S("mrs x0, cntpct_el0")],
}

# Groups whose results depend on the host clock (or otherwise differ run to
# run); the differential runner compares them loosely instead of exactly.
VOLATILE = {"counter"}

# Results the Go port is known to differ on, and why. The port reports only the
# features its decoder actually executes: advertising the FP and crypto
# families before the SIMD/FP executor is ported would promise a guest
# instructions that end in SIGILL a million cycles into libc. See
# docs/PORT.md, "Feature advertisements".
KNOWN = {
    "sysreg[4]": "ID_AA64ISAR0_EL1: the FP/crypto features are not ported yet",
    "ids[5]":    "ID_AA64PFR0_EL1: FP and AdvSIMD fields not ported yet",
    "ids[9]":    "ID_AA64ISAR0_EL1: the FP/crypto features are not ported yet",
    "ids[10]":   "ID_AA64ISAR1_EL1: JSCVT and FCMA are not ported yet",
    "sysreg[6]": "CNTVCT_EL0: the generic timer's origin is the host's uptime",
}


def main():
    out = sys.argv[1] if len(sys.argv) > 1 else "test.elf"
    only = set(sys.argv[2:])
    snippets = []
    for name, snips in SNIPPETS.items():
        if only and name not in only:
            continue
        snippets.extend(snips)
    blob = build(snippets)
    with open(out, "wb") as f:
        f.write(blob)
    print("wrote %s (%d snippets, %d result bytes)" % (out, len(snippets), len(snippets) * 8))


if __name__ == "__main__":
    main()

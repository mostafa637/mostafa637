// Reference-output generator for the decoder differential test.
//
// emu/decode.h is not a library: it is a template that asbestos/gen.c includes
// twice (OP_SIZE 32 and OP_SIZE 16), supplying ~150 macros that turn each
// decoded instruction into gadget emissions. This file supplies the same
// macros as *recorders* - every one stringifies its arguments instead of
// generating anything - and then drives the unmodified header over a corpus of
// byte strings, so the Rust decoder has something to be compared against.
//
//   cc -O2 -I<ish-src> -o decode-dump decode-dump.c <ish-src>/emu/tlb.c
//
// What is recorded per case:
//   E READMODRM                a ModRM (and maybe SIB) byte was consumed
//   E READIMM <bits>           an immediate of that width was consumed
//   E <MACRO> <arg>...         a semantic macro, with its arguments verbatim
//   E SEG_GS / UNDEFINED / SEGFAULT
//   R <ret> <end_ip>           ret: 0 finished, 1 finished and ended the block,
//                              2 undefined instruction, 3 faulted
//
// The fixture header describes the memory image (the fill rule, the unmapped
// page, the base ip) and each case line carries its own bytes, so
// tests/decode_differential.rs rebuilds the same image and duplicates nothing.

#include <stdio.h>
#include <stdarg.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

#include "misc.h"
#include "emu/cpu.h"
#include "emu/tlb.h"
#include "emu/modrm.h"
#include "emu/interrupt.h"

#define FAKE_PAGES 16
#define FAKE_BYTES (FAKE_PAGES << PAGE_BITS)
#define UNMAPPED_PAGE 8
#define BASE_IP 0x400
// 8 memory classes + 64 register classes; see class_of below
#define NCLASS (8 + 64)
#define FILL_MUL 7
#define FILL_ADD 3

static unsigned char backing[FAKE_BYTES];

static void *fake_translate(struct mmu *mmu, addr_t addr, int type) {
    (void) mmu;
    (void) type;
    if ((addr >> PAGE_BITS) == UNMAPPED_PAGE)
        return NULL;
    return backing + (size_t) (addr & (FAKE_BYTES - 1));
}

static struct mmu_ops fake_ops = { .translate = fake_translate };
static struct mmu mmu;
static struct tlb tlb;

int current_pid(void) { return 0; }

// ---- recording ----
static int live; // only record while a case is being decoded

// When capture is non-NULL the record goes into that buffer instead of stdout.
// The class invariant check below uses it to compare traces without printing
// half a million of them.
static char *capture;
static size_t capture_len, capture_cap;

static void out(const char *fmt, ...) {
    char buf[512];
    va_list ap;
    va_start(ap, fmt);
    int n = vsnprintf(buf, sizeof(buf), fmt, ap);
    va_end(ap);
    if (n < 0) return;
    if (!capture) {
        fputs(buf, stdout);
        return;
    }
    if (capture_len + (size_t) n + 1 > capture_cap) {
        fprintf(stderr, "capture buffer overflow\n");
        exit(1);
    }
    memcpy(capture + capture_len, buf, (size_t) n + 1);
    capture_len += (size_t) n;
}

static void rec0(const char *op) {
    if (!live) return;
    out("E %s\n", op);
}
static void rec1(const char *op, const char *a) {
    if (!live) return;
    out("E %s %s\n", op, a);
}
static void rec2(const char *op, const char *a, const char *b) {
    if (!live) return;
    out("E %s %s %s\n", op, a, b);
}
static void rec3(const char *op, const char *a, const char *b, const char *c) {
    if (!live) return;
    out("E %s %s %s %s\n", op, a, b, c);
}
static void rec4(const char *op, const char *a, const char *b, const char *c, const char *d) {
    if (!live) return;
    out("E %s %s %s %s %s\n", op, a, b, c, d);
}

#define R0(M) rec0(#M)
#define R1(M,a) rec1(#M, a)
#define R2(M,a,b) rec2(#M, a, b)
#define R3(M,a,b,c) rec3(#M, a, b, c)
#define R4(M,a,b,c,d) rec4(#M, a, b, c, d)

// ---- the ~150 semantic macros, as recorders ----
// Generated from emu/decode.h by counting each macro's arguments; a macro used
#define ADC(a,b,c) R3(ADC,#a,#b,#c)
#define ADD(a,b,c) R3(ADD,#a,#b,#c)
#define AND(a,b,c) R3(AND,#a,#b,#c)
#define ATOMIC_ADC(a,b,c) R3(ATOMIC_ADC,#a,#b,#c)
#define ATOMIC_ADD(a,b,c) R3(ATOMIC_ADD,#a,#b,#c)
#define ATOMIC_AND(a,b,c) R3(ATOMIC_AND,#a,#b,#c)
#define ATOMIC_BTC(a,b,c) R3(ATOMIC_BTC,#a,#b,#c)
#define ATOMIC_BTR(a,b,c) R3(ATOMIC_BTR,#a,#b,#c)
#define ATOMIC_BTS(a,b,c) R3(ATOMIC_BTS,#a,#b,#c)
#define ATOMIC_CMPXCHG(a,b,c) R3(ATOMIC_CMPXCHG,#a,#b,#c)
#define ATOMIC_CMPXCHG8B(a,b) R2(ATOMIC_CMPXCHG8B,#a,#b)
#define ATOMIC_DEC(a,b) R2(ATOMIC_DEC,#a,#b)
#define ATOMIC_INC(a,b) R2(ATOMIC_INC,#a,#b)
#define ATOMIC_OR(a,b,c) R3(ATOMIC_OR,#a,#b,#c)
#define ATOMIC_SBB(a,b,c) R3(ATOMIC_SBB,#a,#b,#c)
#define ATOMIC_SUB(a,b,c) R3(ATOMIC_SUB,#a,#b,#c)
#define ATOMIC_XADD(a,b,c) R3(ATOMIC_XADD,#a,#b,#c)
#define ATOMIC_XOR(a,b,c) R3(ATOMIC_XOR,#a,#b,#c)
#define BSF(a,b,c) R3(BSF,#a,#b,#c)
#define BSR(a,b,c) R3(BSR,#a,#b,#c)
#define BSWAP(a) R1(BSWAP,#a)
#define BT(a,b,c) R3(BT,#a,#b,#c)
#define BTC(a,b,c) R3(BTC,#a,#b,#c)
#define BTR(a,b,c) R3(BTR,#a,#b,#c)
#define BTS(a,b,c) R3(BTS,#a,#b,#c)
#define CALL(a) do { R1(CALL,#a); end_block = true; } while (0)
#define CALL_REL(a) do { R1(CALL_REL,#a); end_block = true; } while (0)
#define CMOV(a,b,c,d) R4(CMOV,#a,#b,#c,#d)
#define CMOVN(a,b,c,d) R4(CMOVN,#a,#b,#c,#d)
#define CMP(a,b,c) R3(CMP,#a,#b,#c)
#define CMPXCHG(a,b,c) R3(CMPXCHG,#a,#b,#c)
#define CMPXCHG8B(a,b) R2(CMPXCHG8B,#a,#b)
#define CPUID() R0(CPUID)
#define DEC(a,b) R2(DEC,#a,#b)
#define DIV(a,b) R2(DIV,#a,#b)
#define F2XM1() R0(F2XM1)
#define FABS() R0(FABS)
#define FADD(a,b) R2(FADD,#a,#b)
#define FADDM(a,b) R2(FADDM,#a,#b)
#define FCHS() R0(FCHS)
#define FCLEX() R0(FCLEX)
#define FCMOVB(a) R1(FCMOVB,#a)
#define FCMOVBE(a) R1(FCMOVBE,#a)
#define FCMOVE(a) R1(FCMOVE,#a)
#define FCMOVNB(a) R1(FCMOVNB,#a)
#define FCMOVNBE(a) R1(FCMOVNBE,#a)
#define FCMOVNE(a) R1(FCMOVNE,#a)
#define FCMOVNU(a) R1(FCMOVNU,#a)
#define FCMOVU(a) R1(FCMOVU,#a)
#define FCOM() R0(FCOM)
#define FCOMI() R0(FCOMI)
#define FCOMM(a,b) R2(FCOMM,#a,#b)
#define FCOS() R0(FCOS)
#define FDIV(a,b) R2(FDIV,#a,#b)
#define FDIVM(a,b) R2(FDIVM,#a,#b)
#define FDIVR(a,b) R2(FDIVR,#a,#b)
#define FDIVRM(a,b) R2(FDIVRM,#a,#b)
#define FIADD(a,b) R2(FIADD,#a,#b)
#define FICOM(a,b) R2(FICOM,#a,#b)
#define FIDIV(a,b) R2(FIDIV,#a,#b)
#define FIDIVR(a,b) R2(FIDIVR,#a,#b)
#define FILD(a,b) R2(FILD,#a,#b)
#define FIMUL(a,b) R2(FIMUL,#a,#b)
#define FINCSTP() R0(FINCSTP)
#define FIST(a,b) R2(FIST,#a,#b)
#define FISUB(a,b) R2(FISUB,#a,#b)
#define FISUBR(a,b) R2(FISUBR,#a,#b)
#define FLD() R0(FLD)
#define FLDC(a) R1(FLDC,#a)
#define FLDCW(a) R1(FLDCW,#a)
#define FLDENV(a,b) R2(FLDENV,#a,#b)
#define FLDM(a,b) R2(FLDM,#a,#b)
#define FMUL(a,b) R2(FMUL,#a,#b)
#define FMULM(a,b) R2(FMULM,#a,#b)
#define FPATAN() R0(FPATAN)
#define FPREM() R0(FPREM)
#define FRESTORE(a,b) R2(FRESTORE,#a,#b)
#define FRNDINT() R0(FRNDINT)
#define FSAVE(a,b) R2(FSAVE,#a,#b)
#define FSCALE() R0(FSCALE)
#define FSIN() R0(FSIN)
#define FSQRT() R0(FSQRT)
#define FST() R0(FST)
#define FSTCW(a) R1(FSTCW,#a)
#define FSTENV(a,b) R2(FSTENV,#a,#b)
#define FSTM(a,b) R2(FSTM,#a,#b)
#define FSTSW(a) R1(FSTSW,#a)
#define FSUB(a,b) R2(FSUB,#a,#b)
#define FSUBM(a,b) R2(FSUBM,#a,#b)
#define FSUBR(a,b) R2(FSUBR,#a,#b)
#define FSUBRM(a,b) R2(FSUBRM,#a,#b)
#define FTST() R0(FTST)
#define FUCOM() R0(FUCOM)
#define FUCOMI() R0(FUCOMI)
#define FXAM() R0(FXAM)
#define FXCH() R0(FXCH)
#define FXTRACT() R0(FXTRACT)
#define FYL2X() R0(FYL2X)
#define IDIV(a,b) R2(IDIV,#a,#b)
#define IMUL1(a,b) R2(IMUL1,#a,#b)
#define IMUL2(a,b,c) R3(IMUL2,#a,#b,#c)
#define IMUL3(a,b,c,d) R4(IMUL3,#a,#b,#c,#d)
#define INC(a,b) R2(INC,#a,#b)
#define INT(a) do { R1(INT,#a); end_block = true; } while (0)
#define JCXZ_REL(a) do { R1(JCXZ_REL,#a); end_block = true; } while (0)
#define JMP(a) do { R1(JMP,#a); end_block = true; } while (0)
#define JMP_REL(a) do { R1(JMP_REL,#a); end_block = true; } while (0)
#define JN_REL(a,b) do { R2(JN_REL,#a,#b); end_block = true; } while (0)
#define J_REL(a,b) do { R2(J_REL,#a,#b); end_block = true; } while (0)
#define MOV(a,b,c) R3(MOV,#a,#b,#c)
#define MOVSX(a,b,c,d) R4(MOVSX,#a,#b,#c,#d)
#define MOVZX(a,b,c,d) R4(MOVZX,#a,#b,#c,#d)
#define MUL1(a,b) R2(MUL1,#a,#b)
#define NEG(a,b) R2(NEG,#a,#b)
#define NOT(a,b) R2(NOT,#a,#b)
#define OR(a,b,c) R3(OR,#a,#b,#c)
#define POP(a,b) R2(POP,#a,#b)
#define POPF() R0(POPF)
#define PUSH(a,b) R2(PUSH,#a,#b)
#define PUSHF() R0(PUSHF)
#define RCL(a,b,c) R3(RCL,#a,#b,#c)
#define RCR(a,b,c) R3(RCR,#a,#b,#c)
#define REP(a,b) R2(REP,#a,#b)
#define REPNZ(a,b) R2(REPNZ,#a,#b)
#define REPZ(a,b) R2(REPZ,#a,#b)
#define RET_NEAR(a) do { R1(RET_NEAR,#a); end_block = true; } while (0)
#define ROL(a,b,c) R3(ROL,#a,#b,#c)
#define ROR(a,b,c) R3(ROR,#a,#b,#c)
#define SAR(a,b,c) R3(SAR,#a,#b,#c)
#define SBB(a,b,c) R3(SBB,#a,#b,#c)
#define SET(a,b) R2(SET,#a,#b)
#define SETN(a,b) R2(SETN,#a,#b)
#define SHL(a,b,c) R3(SHL,#a,#b,#c)
#define SHLD(a,b,c,d) R4(SHLD,#a,#b,#c,#d)
#define SHR(a,b,c) R3(SHR,#a,#b,#c)
#define SHRD(a,b,c,d) R4(SHRD,#a,#b,#c,#d)
#define STR(a,b) R2(STR,#a,#b)
#define SUB(a,b,c) R3(SUB,#a,#b,#c)
#define TEST(a,b,c) R3(TEST,#a,#b,#c)
#define VMOV(a,b,c) R3(VMOV,#a,#b,#c)
#define VMOV_MERGE_REG(a,b,c) R3(VMOV_MERGE_REG,#a,#b,#c)
#define V_OP(a,b,c,d) R4(V_OP,#a,#b,#c,#d)
#define V_OP_IMM(a,b,c,d) R4(V_OP_IMM,#a,#b,#c,#d)
#define XADD(a,b,c) R3(XADD,#a,#b,#c)
#define XCHG(a,b,c) R3(XCHG,#a,#b,#c)
#define XOR(a,b,c) R3(XOR,#a,#b,#c)
#define CLD R0(CLD)
#define STD R0(STD)
#define CVT R0(CVT)
#define CVTE R0(CVTE)
#define SAHF R0(SAHF)
#define RDTSC R0(RDTSC)
#define FPOP R0(FPOP)

// ---- the backend hooks emu/decode.h expects ----
struct decode_state {
    addr_t ip;
    addr_t orig_ip;
};

#define RET_OK 0
#define RET_END_BLOCK 1
#define RET_UNDEFINED 2
#define RET_SEGFAULT 3

#define DECLARE_LOCALS \
    dword_t addr_offset = 0; \
    bool end_block = false; \
    bool seg_gs = false; \
    (void) addr_offset; (void) seg_gs
#define FINISH do { rec0(end_block ? "END_BLOCK" : "DONE"); return end_block ? RET_END_BLOCK : RET_OK; } while (0)
#define RESTORE_IP state->ip = state->orig_ip
#define _READIMM(name, size) do { \
    rec1("READIMM", #size); \
    state->ip += size/8; \
    if (!tlb_read(tlb, state->ip - size/8, &name, size/8)) SEGFAULT; \
} while (0)
#define READMODRM do { rec0("READMODRM"); if (!modrm_decode32(&state->ip, tlb, &modrm)) SEGFAULT; } while (0)
#define READADDR _READIMM(addr_offset, 32)
#define SEG_GS() do { seg_gs = true; rec0("SEG_GS"); } while (0)
#define UNDEFINED do { rec0("UNDEFINED"); return RET_UNDEFINED; } while (0)
#define SEGFAULT do { rec0("SEGFAULT"); return RET_SEGFAULT; } while (0)

#define DECODER_RET static int
#define DECODER_NAME decode_step
#define DECODER_ARGS struct decode_state *state, struct tlb *tlb
#define DECODER_PASS_ARGS state, tlb

static int decode_step16(struct decode_state *state, struct tlb *tlb);
static int decode_step32(struct decode_state *state, struct tlb *tlb);

#define OP_SIZE 32
#include "emu/decode.h"
#undef OP_SIZE
#define OP_SIZE 16
#include "emu/decode.h"
#undef OP_SIZE


static const unsigned char filler[8] = { 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88 };
static long classes_checked;
static long classes_excepted;
// The eight opcode maps emu/decode.h reaches: the base map, the two-byte map
// behind 0x0f, the lock map behind 0xf0 and its own two-byte map, and the two
// scalar maps behind 0xf2/0xf3 with theirs.
#define NMAP 8
static const unsigned char map_prefix[NMAP][2] = {
    { 0, 0 }, { 0x0f, 0 }, { 0xf0, 0 }, { 0xf0, 0x0f },
    { 0xf2, 0 }, { 0xf2, 0x0f }, { 0xf3, 0 }, { 0xf3, 0x0f },
};
static const int map_len[NMAP] = { 0, 1, 1, 2, 1, 2, 1, 2 };
static const char *const map_name[NMAP] = {
    "base", "0f", "f0", "f0/0f", "f2", "f2/0f", "f3", "f3/0f",
};
// maps swept with two classes only; the generator fails if any opcode in them
// turns out to depend on the class
static const int map_sse[NMAP] = { 0, 0, 0, 0, 1, 1, 1, 1 };

// The opcodes in each map whose next byte is another opcode (or another
// instance) rather than a ModRM byte - exactly the sites emu/decode.h reaches
// with `goto restart` / `goto lockrestart` / `return glue(DECODER_NAME, ...)`,
// plus the map switches (header lines 643, 648, 685, 704/707, 710, 1110,
// 1117/1119 and the four nested `case 0x0f:` blocks). The class invariant skips
// these, and the decoder drives them itself instead of through the table.
#define MAXPFX 9
static const unsigned char map_pfx_ops[NMAP][MAXPFX] = {
    { 0x0f, 0x2e, 0x3e, 0x65, 0x66, 0x67, 0xf0, 0xf2, 0xf3 }, // base
    { 0 },                                                    // 0f
    { 0x0f, 0x65, 0x66 },                                     // f0
    { 0 },                                                    // f0/0f
    { 0x0f },                                                 // f2
    { 0 },                                                    // f2/0f
    { 0x0f },                                                 // f3
    { 0 },                                                    // f3/0f
};
static const int map_npfx[NMAP] = { 9, 0, 3, 0, 1, 0, 1, 0 };

static unsigned char except_map[NMAP][256];
static struct { int map, op; } exceptions[64];
static int nexceptions;
static unsigned long class_traces[NCLASS];

static unsigned long class_hash(const char *s) {
    unsigned long h = 1469598103934665603UL;
    for (; *s; s++) {
        h ^= (unsigned char) *s;
        h *= 1099511628211UL;
    }
    return h;
}

// ---- the class invariant ----
// 16 classes: the eight reg fields with a memory operand, and the eight with a
// register operand. decode.h only ever inspects modrm.opcode (the reg field, in
// the GRP groups) and modrm.type (in READMODRM_MEM/NOMEM), so nothing else about
// the ModRM byte may reach the dispatch - and this checks that nothing does.
static char trace_a[1 << 16];
static char trace_b[1 << 16];

// The ModRM class: what part of the ModRM byte may influence the dispatch.
// Almost every opcode looks only at modrm.opcode (the reg field, in the GRP
// groups) and modrm.type (in READMODRM_MEM/NOMEM and the x87 split) - 8 memory
// classes and 8 register classes. The x87 register form is the exception: its
// fallback switch is `insn << 8 | modrm.opcode << 4 | modrm.rm_opcode`, which
// also reads the rm field, so the register form gets 8 * 8 classes.
//   class <  8 : memory operand, class is the reg field
//   class >= 8 : register operand, class is 8 + reg * 8 + rm

static int class_of(int modrm_byte) {
    if ((modrm_byte >> 6) == 3)
        return 8 + (((modrm_byte >> 3) & 7) << 3) + (modrm_byte & 7);
    return (modrm_byte >> 3) & 7;
}

// the canonical ModRM byte for a class - what the corpus actually feeds
static int class_modrm(int cls) {
    if (cls < 8)
        return cls << 3;                       // mod=00, rm=eax
    int r = (cls - 8) >> 3, m = (cls - 8) & 7;
    return (3 << 6) | (r << 3) | m;            // mod=11
}

static void decode_capture(int size, const unsigned char *b, int n, char *buf, size_t cap) {
    memcpy(backing + BASE_IP, b, n);
    struct decode_state st = { .ip = BASE_IP, .orig_ip = BASE_IP };
    capture = buf;
    capture_len = 0;
    capture_cap = cap;
    live = 1;
    int ret = size == 32 ? decode_step32(&st, &tlb) : decode_step16(&st, &tlb);
    live = 0;
    capture = NULL;
    char tail[32];
    int m = snprintf(tail, sizeof(tail), "R %d\n", ret);
    memcpy(buf + capture_len, tail, (size_t) m + 1);
}

// Opcodes whose next byte is another opcode rather than a ModRM byte: the four
// prefixes of the base map, plus the two the lock map accepts before it
// restarts. For these the ModRM class dimension is meaningless, so they are
// excluded from the invariant and handled by the decoder's own prefix logic
// rather than by the generated table.
// The opcodes whose next byte is another opcode (or another instance) rather
// than a ModRM byte. These are exactly the sites emu/decode.h reaches with
// `goto restart` / `goto lockrestart` / `return glue(DECODER_NAME, ...)`, plus
// the four map switches - lines 643, 648, 685, 704/707, 710 and 1110,
// 1117/1119 of the unmodified header:
//   base map  0x0f (two-byte map), 0x2e 0x3e 0x65 0x67 (restart),
//             0x66 (the other operand size), 0xf0 0xf2 0xf3 (their own maps)
//   lock map  0x65 (lockrestart), 0x66 (lockrestart, or the other size in 32),
//             0x0f (its own two-byte map: lock bts/btr/btc)
// For these the ModRM class dimension is meaningless, so they are excluded from
// the invariant and driven by the decoder's own prefix logic instead of the
// generated table. Anything else that violates the invariant is a bug in this
// reasoning, and the check below fails rather than quietly widening the list.
static int is_prefix_op(int map, int op) {
    for (int i = 0; i < map_npfx[map]; i++)
        if (map_pfx_ops[map][i] == op)
            return 1;
    return 0;
}

static void check_classes(void) {
    unsigned char b[16];
    long checked = 0;
    long excepted = 0;
    for (int pass = 0; pass < 2; pass++) {
        int size = pass ? 16 : 32;
        for (int mi = 0; mi < NMAP; mi++) {
            int np = map_len[mi];
            memcpy(b, map_prefix[mi], np);
            for (int op = 0; op < 256; op++) {
                b[np] = (unsigned char) op;
                int bad = 0;
                memset(class_traces, 0, sizeof(class_traces));
                int distinct = 0;
                for (int cls = 0; cls < NCLASS && !bad; cls++) {
                    b[np + 1] = (unsigned char) class_modrm(cls);
                    memcpy(b + np + 2, filler, 8);
                    decode_capture(size, b, np + 10, trace_a, sizeof(trace_a));
                    size_t ref_len = strlen(trace_a);
                    int seen = 0;
                    for (int prev = 0; prev < cls; prev++)
                        if (class_traces[prev] && class_traces[prev] == class_hash(trace_a))
                            seen = 1;
                    if (!seen) {
                        distinct++;
                        class_traces[cls] = class_hash(trace_a);
                    }
                    for (int m2 = 0; m2 < 256; m2++) {
                        if (class_of(m2) != cls)
                            continue;
                        b[np + 1] = (unsigned char) m2;
                        decode_capture(size, b, np + 10, trace_b, sizeof(trace_b));
                        checked++;
                        if (strlen(trace_b) != ref_len || memcmp(trace_a, trace_b, ref_len) != 0) {
                            bad = 1;
                            break;
                        }
                    }
                }
                if (map_sse[mi] && distinct > 1 && !is_prefix_op(mi, op)) {
                    fprintf(stderr,
                            "opcode %s/%02x depends on the ModRM class (%d variants) but its map "
                            "is only swept with two classes - widen the corpus\n",
                            map_name[mi], op, distinct);
                    exit(1);
                }
                if (bad) {
                    except_map[mi][op] = 1;
                    excepted++;
                    if (!is_prefix_op(mi, op)) {
                        fprintf(stderr,
                                "class invariant violated by a non-prefix opcode: "
                                "instance %d map %s opcode %02x\n", size, map_name[mi], op);
                        exit(1);
                    }
                }
            }
        }
    }
    classes_checked = checked;
    classes_excepted = excepted;
    for (int mi = 0; mi < NMAP; mi++)
        for (int op = 0; op < 256; op++)
            if (except_map[mi][op])
                exceptions[nexceptions].map = mi, exceptions[nexceptions].op = op, nexceptions++;
}

// ---- the corpus ----
// Every opcode of every map the header has (plain, 0x0f, the lock map behind
// 0xf0, and the 0xf2/0xf3 scalar maps), under a ModRM byte for each of the
// eight reg fields with mod=00 and again with mod=11 - the reg field selects
// the operation inside the GRP groups, and mod=11 is what turns
// READMODRM_MEM/NOMEM into UNDEFINED.
static int ncases;

static void emit_case(int size, addr_t ip, const unsigned char *b, int n) {
    printf("C %d %x", size, ip);
    for (int i = 0; i < n; i++)
        printf(" %02x", b[i]);
    printf("\n");
    memcpy(backing + ip, b, n);
    struct decode_state st = { .ip = ip, .orig_ip = ip };
    live = 1;
    int ret = size == 32 ? decode_step32(&st, &tlb) : decode_step16(&st, &tlb);
    live = 0;
    printf("R %d %x\n", ret, st.ip);
    ncases++;
}

static void sweep(int size, const unsigned char *prefix, int nprefix, int nclass) {
    unsigned char b[16];
    memcpy(b, prefix, nprefix);
    for (int op = 0; op < 256; op++) {
        b[nprefix] = (unsigned char) op;
        for (int cls = 0; cls < nclass; cls++) {
            b[nprefix + 1] = (unsigned char) class_modrm(cls);
            memcpy(b + nprefix + 2, filler, 8);
            emit_case(size, BASE_IP, b, nprefix + 10);
        }
    }
}

// The x87 opcodes need every register-form class in the corpus, not just the
// eight canonical ones, because their dispatch reads the rm field too.
static void sweep_x87(int size) {
    unsigned char b[16];
    for (int op = 0xd8; op <= 0xdf; op++) {
        b[0] = (unsigned char) op;
        for (int cls = 8; cls < NCLASS; cls++) {
            b[1] = (unsigned char) class_modrm(cls);
            memcpy(b + 2, filler, 8);
            emit_case(size, BASE_IP, b, 10);
        }
    }
}

#define PER_INSTANCE (4 * 256 * 16 + 4 * 256 * 2 + 8 * (NCLASS - 8))
#define NFAULT 3

int main(void) {
    mmu.ops = &fake_ops;
    mmu.changes = 0;
    memset(&tlb, 0, sizeof(tlb));
    tlb_refresh(&tlb, &mmu);
    for (int i = 0; i < FAKE_BYTES; i++)
        backing[i] = (unsigned char) ((i * FILL_MUL + FILL_ADD) & 0xff);

    // Before emitting anything, prove the assumption the generated Rust table
    // rests on: that after the ModRM byte is consumed, what the decoder does
    // next depends on the ModRM byte only through (reg field, register-form).
    // Every one of the 256 ModRM bytes is decoded for every opcode of every map
    // in both instances and grouped into the 16 classes; any class whose members
    // disagree is a hard failure.
    check_classes();

    printf("# decode reference output, generated from unmodified iSH emu/decode.h\n");
    printf("# pages %d fill %d %d unmap %d base_ip %x\n",
           FAKE_PAGES, FILL_MUL, FILL_ADD, UNMAPPED_PAGE, BASE_IP);
    printf("# classes %d (8 memory + 64 register)\n", NCLASS);
    printf("# class_invariant %ld decodes, %d opcodes excepted:", classes_checked, nexceptions);
    for (int i = 0; i < nexceptions; i++)
        printf(" %s:%02x", map_name[exceptions[i].map], exceptions[i].op);
    printf("\n");
    printf("# cases %d\n", 2 * PER_INSTANCE + NFAULT);

    for (int pass = 0; pass < 2; pass++) {
        int size = pass ? 16 : 32;
        for (int mi = 0; mi < NMAP; mi++)
            sweep(size, map_prefix[mi], map_len[mi], map_sse[mi] ? 2 : 16);
        sweep_x87(size);
    }

    // faults: the modrm byte itself unmapped, a disp32 straddling into the
    // unmapped page, and an immediate doing the same
    {
        // mod=00 rm=101 is the disp32-with-no-base form; rm=110 would be a plain
        // [esi] with no displacement and would not fault at all
        unsigned char b[2] = { 0x8b, 0b00000101 }; // mov eax, [disp32]
        // two bytes, so the ModRM byte really is the disp32 form and the
        // displacement is what runs into the unmapped page
        emit_case(32, UNMAPPED_PAGE * PAGE_SIZE - 3, b, 2);
        emit_case(32, UNMAPPED_PAGE * PAGE_SIZE - 1, b, 1);
        emit_case(32, UNMAPPED_PAGE * PAGE_SIZE, b, 1);
    }
    if (ncases != 2 * PER_INSTANCE + NFAULT) {
        fprintf(stderr, "emitted %d cases, declared %d\n", ncases, 2 * PER_INSTANCE + NFAULT);
        return 1;
    }
    return 0;
}

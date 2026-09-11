// Reference-output generator for the vec/mmx differential test.
//
// Compiles against the *unmodified* iSH emu/vec.c + emu/mmx.c and calls every
// one of the 166 functions emu/vec.h declares. The op table is generated from
// that header by tools/gen_vec_ops.py, so an operation cannot be silently
// skipped: if vec.h grows a function, the table grows with it.
//
//   cc -O2 -fno-strict-aliasing -I<ish-src> -o vec-dump vec-dump.c \
//        <ish-src>/emu/vec.c <ish-src>/emu/mmx.c -lm
//
// Build with -fno-strict-aliasing because the operands are handed to the
// functions as double*/float*/int32_t* pointing into an xmm register, which is
// exactly how the emulator itself calls them.
//
// Every record dumps the destination register, the source register (some ops
// modify it), a side buffer for the ops whose destination is a bare scalar,
// and the nine CPU flag fields - the last proving that only the two ucomi ops
// touch the CPU.

#include <stdio.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

#include "emu/cpu.h"
#include "emu/vec.h"
#include "vec-ops.inc"

struct vcase {
    union xmm_reg src;
    union xmm_reg dst;
    uint8_t encoding;
    uint8_t index;
    uint8_t amount;
    uint8_t type;
};

// 16 byte patterns chosen to hit the corners: saturating arithmetic, signed
// compares, movmask, the float compares, and the conversion overflow cases.
static const unsigned char patterns[][16] = {
    { 0x00 },                                                          // zeros
    { [0 ... 15] = 0xff },                                             // all ones
    { [0 ... 15] = 0x80 },                                             // sign bits
    { 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
      0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f },
    { [0 ... 15] = 0x55 },
    { 0xf0, 0x0f, 0xf0, 0x0f, 0xf0, 0x0f, 0xf0, 0x0f,
      0xf0, 0x0f, 0xf0, 0x0f, 0xf0, 0x0f, 0xf0, 0x0f },
    // words: 0x7fff 0x8000 0xffff 0x0001 (little endian)
    { 0xff, 0x7f, 0x00, 0x80, 0xff, 0xff, 0x01, 0x00,
      0xff, 0x7f, 0x00, 0x80, 0xff, 0xff, 0x01, 0x00 },
    // dwords: 0x7fffffff 0x80000000 0xffffffff 0x00000001
    { 0xff, 0xff, 0xff, 0x7f, 0x00, 0x00, 0x00, 0x80,
      0xff, 0xff, 0xff, 0xff, 0x01, 0x00, 0x00, 0x00 },
    // f32: 0.0, -0.0, 1.0, -1.0
    { 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x80,
      0x00, 0x00, 0x80, 0x3f, 0x00, 0x00, 0x80, 0xbf },
    // f32: NaN, +inf, -inf, smallest denormal
    { 0x00, 0x00, 0xc0, 0x7f, 0x00, 0x00, 0x80, 0x7f,
      0x00, 0x00, 0x80, 0xff, 0x01, 0x00, 0x00, 0x00 },
    // f64: 0.0, -0.0, NaN, +inf
    { 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
      0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x80 },
    // f64: 1.5, -2.25, 1e300, -1e-300
    { 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf8, 0x3f,
      0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xc0 },
    // high-bit pattern for movmask
    { 0x80, 0x00, 0x80, 0x00, 0xff, 0x7f, 0x00, 0x80,
      0x80, 0x80, 0x01, 0x00, 0xfe, 0xff, 0x80, 0x80 },
    // i * 17
    { 0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77,
      0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff },
    { 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0x00, 0x00,
      0x00, 0x00, 0xff, 0xff, 0x00, 0x80, 0x00, 0x80 },
    // f64: 2147483647.0, -2147483648.0 (the cvtt boundaries)
    { 0x00, 0xc0, 0xff, 0xff, 0xff, 0xff, 0xdf, 0x41,
      0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xe0, 0xc1 },
    // words: 0xff80 0xff81 0x0080 0x007f ... - the exact points where satsw
    // changes branch
    { 0x80, 0xff, 0x81, 0xff, 0x80, 0x00, 0x7f, 0x00,
      0x81, 0xff, 0x80, 0xff, 0x7f, 0x00, 0x80, 0x00 },
    // the byte-wise complement of the pattern above, so every byte pair sums
    // to exactly 255 - the point where the unsigned saturating adds clamp
    { 0x7f, 0x00, 0x7e, 0x00, 0x7f, 0xff, 0x80, 0xff,
      0x7e, 0x00, 0x7f, 0x00, 0x80, 0xff, 0x7f, 0xff },
    // f64: 0x7ff8000000000000 (quiet NaN) and 0xfff1ff8000000000 (a NaN with
    // the sign bit and a payload)
    { 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf8, 0x7f,
      0x00, 0x00, 0x00, 0x00, 0x80, 0xff, 0xf1, 0xff },
    // f64: 0x7ffc000000000000 (a different payload) and 0xfff0000000000001
    // (a signalling NaN)
    { 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xfc, 0x7f,
      0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf0, 0xff },
};

// NPATTERNS value cases + 12 shift amounts + 6 encodings + 8 indices + 8
// compare types + 4 saturation-boundary pairs
#define NPATTERNS ((int) (sizeof(patterns) / sizeof(patterns[0])))
#define NCASES (NPATTERNS + 12 + 6 + 8 + 8 + 5)
static struct vcase cases[NCASES];


static void init_cases(void) {
    int n = 0;
    // one value pair per pattern, with generic parameters
    for (int i = 0; i < NPATTERNS; i++) {
        memcpy(&cases[n].src, patterns[i], 16);
        memcpy(&cases[n].dst, patterns[(i + 7) % NPATTERNS], 16);
        cases[n].encoding = 0x1b;
        cases[n].index = 2;
        cases[n].amount = 3;
        cases[n].type = 1;
        n++;
    }
    // shift amounts, including every "just past the element width" boundary
    static const uint8_t amounts[12] = { 0, 1, 7, 8, 15, 16, 31, 32, 63, 64, 127, 255 };
    for (int i = 0; i < 12; i++) {
        memcpy(&cases[n].src, patterns[3], 16);
        memcpy(&cases[n].dst, patterns[6], 16);
        cases[n].amount = amounts[i];
        cases[n].encoding = 0x1b;
        cases[n].index = 2;
        cases[n].type = 1;
        n++;
    }
    // shuffle encodings
    static const uint8_t encodings[6] = { 0x00, 0x1b, 0xff, 0xa4, 0xe4, 0x55 };
    for (int i = 0; i < 6; i++) {
        memcpy(&cases[n].src, patterns[13], 16);
        memcpy(&cases[n].dst, patterns[4], 16);
        cases[n].encoding = encodings[i];
        cases[n].amount = 3;
        cases[n].index = 2;
        cases[n].type = 1;
        n++;
    }
    // insert/extract indices
    for (int i = 0; i < 8; i++) {
        memcpy(&cases[n].src, patterns[13], 16);
        memcpy(&cases[n].dst, patterns[14], 16);
        cases[n].index = (uint8_t) i;
        cases[n].encoding = 0x1b;
        cases[n].amount = 3;
        cases[n].type = 1;
        n++;
    }
    // compare types, including the >= 4 "invert" half
    for (int i = 0; i < 8; i++) {
        memcpy(&cases[n].src, patterns[(i % 2) ? 9 : 11], 16);
        memcpy(&cases[n].dst, patterns[(i % 2) ? 8 : 10], 16);
        cases[n].type = (uint8_t) i;
        cases[n].encoding = 0x1b;
        cases[n].amount = 3;
        cases[n].index = 2;
        n++;
    }
    // pairs chosen so byte sums land exactly on 255 and byte differences on
    // 0 and -1, which is where the saturating adds and subs change branch
    // {18, 19} pairs NaN with NaN in the *f64* view as well, which is what
    // pins the NaN payload of the commutative arithmetic
    static const int bpair[5][2] = { { 16, 17 }, { 17, 16 }, { 16, 16 }, { 17, 17 }, { 18, 19 } };
    for (int i = 0; i < 5; i++) {
        memcpy(&cases[n].src, patterns[bpair[i][0]], 16);
        memcpy(&cases[n].dst, patterns[bpair[i][1]], 16);
        cases[n].encoding = 0x1b;
        cases[n].index = 2;
        cases[n].amount = 3;
        cases[n].type = 1;
        n++;
    }
    if (n != NCASES) {
        fprintf(stderr, "case count mismatch: built %d, declared %d\n", n, NCASES);
        exit(1);
    }
}

static void run(const struct vec_op *op, union xmm_reg *src, union xmm_reg *dst,
                unsigned char *out, const struct vcase *c, struct cpu_state *cpu) {
    union mm_reg *srcm = (union mm_reg *) src;
    union mm_reg *dstm = (union mm_reg *) dst;
    switch (op->kind) {
    case K_VOID_VOID:   op->fn.void_void(cpu, src, dst); break;
    case K_XMM_XMM_C:   op->fn.xmm_xmm_c(cpu, src, dst); break;
    case K_XMM_XMM:     op->fn.xmm_xmm(cpu, src, dst); break;
    case K_MM_MM_C:     op->fn.mm_mm_c(cpu, srcm, dstm); break;
    case K_MM_MM:       op->fn.mm_mm(cpu, srcm, dstm); break;
    case K_IMM_XMM:     op->fn.imm_xmm(cpu, c->amount, dst); break;
    case K_IMM_XMM_NC:  op->fn.imm_xmm_nc(cpu, c->amount, dst); break;
    case K_IMM_MM:      op->fn.imm_mm(cpu, c->amount, dstm); break;
    case K_F64_F64:     op->fn.f64_f64(cpu, (double *) src, (double *) dst); break;
    case K_F32_F32:     op->fn.f32_f32(cpu, (float *) src, (float *) dst); break;
    case K_XMM_XMM_ENC: op->fn.xmm_xmm_enc(cpu, src, dst, c->encoding); break;
    case K_MM_MM_ENC:   op->fn.mm_mm_enc(cpu, srcm, dstm, c->encoding); break;
    case K_U64_XMM:     op->fn.u64_xmm(cpu, (uint64_t *) src, dst); break;
    case K_XMM_U64:     op->fn.xmm_u64(cpu, src, (uint64_t *) out); break;
    case K_XMM_U32:     op->fn.xmm_u32(cpu, src, (uint32_t *) out); break;
    case K_MM_U32:      op->fn.mm_u32(cpu, srcm, (uint32_t *) out); break;
    case K_F32_F32_CC:  op->fn.f32_f32_cc(cpu, (float *) src, (float *) dst); break;
    case K_F64_F64_CC:  op->fn.f64_f64_cc(cpu, (double *) src, (double *) dst); break;
    case K_F64_XMM_TYPE: op->fn.f64_xmm_type(cpu, (double *) src, dst, c->type); break;
    case K_F32_XMM_TYPE: op->fn.f32_xmm_type(cpu, (float *) src, dst, c->type); break;
    case K_XMM_XMM_TYPE: op->fn.xmm_xmm_type(cpu, src, dst, c->type); break;
    case K_I32_F64:     op->fn.i32_f64(cpu, (int32_t *) src, (double *) dst); break;
    case K_F64_I32:     op->fn.f64_i32(cpu, (double *) src, (int32_t *) dst); break;
    case K_F64_F32:     op->fn.f64_f32(cpu, (double *) src, (float *) dst); break;
    case K_I32_F32:     op->fn.i32_f32(cpu, (int32_t *) src, (float *) dst); break;
    case K_F32_I32:     op->fn.f32_i32(cpu, (float *) src, (int32_t *) dst); break;
    case K_F32_F64:     op->fn.f32_f64(cpu, (float *) src, (double *) dst); break;
    case K_U32_MM_IDX:  op->fn.u32_mm_idx(cpu, (uint32_t *) src, dstm, c->index); break;
    case K_U32_XMM_IDX: op->fn.u32_xmm_idx(cpu, (uint32_t *) src, dst, c->index); break;
    case K_XMM_U32_IDX: op->fn.xmm_u32_idx(cpu, src, (uint32_t *) out, c->index); break;
    default:
        fprintf(stderr, "unknown kind %d for %s\n", op->kind, op->name);
        exit(1);
    }
}

static void emit_bytes(const unsigned char *b, int n) {
    for (int i = 0; i < n; i++)
        printf(" %02x", b[i]);
}

int main(void) {
    init_cases();

    printf("# vec/mmx reference output, generated from unmodified iSH "
           "emu/vec.c + emu/mmx.c\n");
    printf("# ops %d, cases %d\n", N_VEC_OPS, NCASES);
    for (int i = 0; i < NCASES; i++) {
        printf("# C %d", i);
        emit_bytes((unsigned char *) &cases[i].src, 16);
        emit_bytes((unsigned char *) &cases[i].dst, 16);
        printf(" %02x %02x %02x %02x\n", cases[i].encoding, cases[i].index,
               cases[i].amount, cases[i].type);
    }

    for (int o = 0; o < N_VEC_OPS; o++) {
        for (int c = 0; c < NCASES; c++) {
            union xmm_reg src = cases[c].src;
            union xmm_reg dst = cases[c].dst;
            unsigned char out[16];
            struct cpu_state cpu;
            memset(out, 0, sizeof(out));
            memset(&cpu, 0, sizeof(cpu));

            run(&vec_ops[o], &src, &dst, out, &cases[c], &cpu);

            printf("V %s %d", vec_ops[o].name, c);
            emit_bytes((unsigned char *) &dst, 16);
            emit_bytes((unsigned char *) &src, 16);
            emit_bytes(out, 16);
            printf(" %d %d %d %d %d %d %d %d %d\n",
                   cpu.zf, cpu.cf, cpu.pf, cpu.of, cpu.sf, cpu.af,
                   cpu.zf_res, cpu.pf_res, cpu.sf_res);
        }
    }
    return 0;
}

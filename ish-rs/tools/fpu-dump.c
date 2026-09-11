// Reference-output generator for the cpu/fpu differential test.
//
// Compiles against the *unmodified* iSH sources (emu/cpu.h, emu/fpu.c,
// emu/float80.c) and dumps the complete cpu_state after every single FPU
// operation, plus the flag-macro evaluations from emu/cpu.h. The Rust port is
// then required to reproduce every one of those words exactly.
//
//   cc -O2 -DNDEBUG -I<ish-src> -o fpu-dump fpu-dump.c <ish-src>/emu/fpu.c \
//      <ish-src>/emu/float80.c -lm
//
// See tools/gen_fpu_reference.sh for the scripted version.

#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>
#include <float.h>
#include "emu/cpu.h"
#include "emu/fpu.h"
#include "emu/float80.h"

// ---- memory operand pool ------------------------------------------------

struct memval {
    uint64_t bits64;   // read as int64/int32/int16/double
    float80 f80;       // used by the 80-bit ops
};

static struct memval memvals[] = {
    { 0x0000000000000000ull, { 0x0000000000000000ull, 0x0000 } },
    { 0x0000000000000001ull, { 0x8000000000000000ull, 0x3fff } },
    { 0xffffffffffffffffull, { 0x8000000000000000ull, 0xbfff } },
    { 0x000000007fffffffll, { 0xc000000000000000ull, 0x4000 } },
    { 0xffffffff80000000ull, { 0xa000000000000000ull, 0x4000 } },
    { 0x0000000000007fffll, { 0x9a209a84fbcff799ull, 0x3ffd } },
    { 0x00000000ffff8000ull, { 0x8000000000000000ull, 0x0000 } },
    { 0x0000000000009c40ll, { 0x0000000000000001ull, 0x0000 } },
    { 0x400921fb54442d18ull, { 0xc90fdaa22168c235ull, 0x4000 } }, // pi
    { 0xc00921fb54442d18ull, { 0xffffffffffffffffull, 0x7ffe } },
    { 0x7fefffffffffffffull, { 0x7fffffffffffffffull, 0x0000 } },
    { 0x3ff0000000000000ull, { 0xc000000000000000ull, 0xffff } }, // 1.0
};
#define N_MEM ((int) (sizeof(memvals) / sizeof(memvals[0])))

// ---- state pool ---------------------------------------------------------

static uint64_t lcg_state = 0x9e3779b97f4a7c15ull;
static uint64_t lcg(void) {
    lcg_state = lcg_state * 6364136223846793005ull + 1442695040888963407ull;
    return lcg_state >> 11;
}

// The values the eight stack registers get filled with.
static float80 pool[] = {
    { 0x8000000000000000ull, 0x3fff },  //  1.0
    { 0x8000000000000000ull, 0xbfff },  // -1.0
    { 0xc000000000000000ull, 0x4000 },  //  3.0
    { 0x9a209a84fbcff799ull, 0x3ffd },  //  log10(2)
    { 0x0000000000000000ull, 0x0000 },  // +0
    { 0x0000000000000000ull, 0x8000 },  // -0
    { 0x8000000000000000ull, 0x7fff },  // +inf
    { 0xc000000000000000ull, 0x7fff },  //  nan
    { 0x0000000000000001ull, 0x0000 },  //  smallest denormal
    { 0x8000000000000000ull, 0x0001 },  //  smallest normal
    { 0xffffffffffffffffull, 0x7ffe },  //  largest finite
    { 0xd49a784bcd1b8afeull, 0x4000 },  //  log2(10)
    { 0xb17217f7d1cf79acull, 0x3ffe },  //  ln2
    { 0xa000000000000000ull, 0x4002 },  //  10.0
    { 0x4000000000000000ull, 0x3fff },  //  unormal (unsupported)
};
#define N_POOL ((int) (sizeof(pool) / sizeof(pool[0])))

#define N_STATES 24

static struct cpu_state states[N_STATES];
static int n_states;

static void build_states(void) {
    for (int s = 0; s < N_STATES; s++) {
        struct cpu_state *cpu = &states[s];
        memset(cpu, 0, sizeof(*cpu));
        for (int i = 0; i < 8; i++) {
            if (s < 4) {
                // a few hand-picked, easy-to-read configurations
                cpu->fp[i] = pool[(s * 8 + i) % N_POOL];
            } else {
                cpu->fp[i] = pool[lcg() % N_POOL];
            }
        }
        cpu->top = s % 8;
        cpu->fcw = (uint16_t) (lcg() & 0x0fff);
        cpu->fsw = (uint16_t) ((cpu->fsw & ~(0x47ff)) | (lcg() & 0x4700));
        // keep top where we just put it (the fsw write above skipped its bits)
        cpu->cf = lcg() & 1;
        cpu->of = lcg() & 1;
        cpu->zf = lcg() & 1;
        cpu->sf = lcg() & 1;
        cpu->pf = lcg() & 1;
        cpu->af = lcg() & 1;
        cpu->cf_bit = lcg() & 1;
        cpu->of_bit = lcg() & 1;
        cpu->res = (dword_t) lcg();
        cpu->op1 = (dword_t) lcg();
        cpu->op2 = (dword_t) lcg();
        cpu->flags_res = (byte_t) (lcg() & 0xf);
        cpu->eflags = (cpu->eflags & 0x3fff) | ((dword_t) (lcg() & 0x3ffff) << 14);
        n_states++;
    }
}

static void emit_state(struct cpu_state *cpu) {
    printf(" %x %x %x %x %x %x %x %x %x %x %x %x %x %x %x %x",
           (unsigned) cpu->top, (unsigned) cpu->fsw, (unsigned) cpu->fcw,
           (unsigned) cpu->cf, (unsigned) cpu->of,
           (unsigned) cpu->zf, (unsigned) cpu->sf,
           (unsigned) cpu->pf, (unsigned) cpu->af,
           (unsigned) cpu->cf_bit, (unsigned) cpu->of_bit,
           cpu->res, cpu->op1, cpu->op2,
           (unsigned) cpu->flags_res, cpu->eflags);
    for (int i = 0; i < 8; i++)
        printf(" %016llx %04x",
               (unsigned long long) cpu->fp[i].signif, (unsigned) cpu->fp[i].signExp);
    printf("\n");
}

// struct fpu_env32 and struct fpu_state32 are declared inside emu/fpu.c
// itself, not in the header, so they are invisible here. These views have the
// same layout (7 x uint32, then 8 x 10 bytes, no padding) and we cast.
struct env32_view {
    uint32_t control, status, tag, ip, ip_selector, operand, operand_selector;
};
struct state32_view {
    struct env32_view env;
    uint8_t regs[8][10];
};

// f80_log2 (used by fyl2x) loops forever on +inf and on unsupported encodings,
// which is upstream behaviour; skip those operands rather than hang.
static int log2_safe(float80 f) {
    return f80_is_supported(f) && !f80_isnan(f) && !f80_isinf(f);
}

#define ST(i) cpu->fp[(cpu->top + i) % 8]

// Run `body` against a fresh copy of every state and dump the result.
#define FOR_EACH_STATE(name, args, body) \
    do { \
        for (int s = 0; s < n_states; s++) { \
            struct cpu_state *cpu = &run; \
            memcpy(cpu, &states[s], sizeof(*cpu)); \
            body; \
            printf("P %s %d%s", name, s, args); \
            emit_state(cpu); \
        } \
    } while (0)

static struct cpu_state run;

int main(void) {
    build_states();
    printf("# fpu reference output, generated from unmodified iSH emu/fpu.c + emu/cpu.h\n");
    printf("# states %d, memory operands %d\n", n_states, N_MEM);

    // ---- stack manipulation ----
    FOR_EACH_STATE("pop", "", fpu_pop(cpu));
    FOR_EACH_STATE("incstp", "", fpu_incstp(cpu));
    for (int i = 0; i < 8; i++) {
        char args[32];
        snprintf(args, sizeof(args), " %d", i);
        FOR_EACH_STATE("xch", args, fpu_xch(cpu, i));
        FOR_EACH_STATE("ld", args, fpu_ld(cpu, i));
        FOR_EACH_STATE("st", args, fpu_st(cpu, i));
        FOR_EACH_STATE("com", args, fpu_com(cpu, i));
        FOR_EACH_STATE("comi", args, fpu_comi(cpu, i));
        FOR_EACH_STATE("cmovb", args, fpu_cmovb(cpu, i));
        FOR_EACH_STATE("cmove", args, fpu_cmove(cpu, i));
        FOR_EACH_STATE("cmovbe", args, fpu_cmovbe(cpu, i));
        FOR_EACH_STATE("cmovu", args, fpu_cmovu(cpu, i));
        FOR_EACH_STATE("cmovnb", args, fpu_cmovnb(cpu, i));
        FOR_EACH_STATE("cmovne", args, fpu_cmovne(cpu, i));
        FOR_EACH_STATE("cmovnbe", args, fpu_cmovnbe(cpu, i));
        FOR_EACH_STATE("cmovnu", args, fpu_cmovnu(cpu, i));
    }
    for (int c = 0; c <= fconst_zero; c++) {
        char args[32];
        snprintf(args, sizeof(args), " %d", c);
        FOR_EACH_STATE("ldc", args, fpu_ldc(cpu, c));
    }

    // ---- single-operand math ----
    FOR_EACH_STATE("prem", "", fpu_prem(cpu));
    FOR_EACH_STATE("rndint", "", fpu_rndint(cpu));
    FOR_EACH_STATE("scale", "", fpu_scale(cpu));
    FOR_EACH_STATE("abs", "", fpu_abs(cpu));
    FOR_EACH_STATE("chs", "", fpu_chs(cpu));
    FOR_EACH_STATE("sqrt", "", fpu_sqrt(cpu));
    FOR_EACH_STATE("2xm1", "", fpu_2xm1(cpu));
    FOR_EACH_STATE("tst", "", fpu_tst(cpu));
    FOR_EACH_STATE("xam", "", fpu_xam(cpu));
    FOR_EACH_STATE("xtract", "", fpu_xtract(cpu));
    FOR_EACH_STATE("sin", "", fpu_sin(cpu));
    FOR_EACH_STATE("cos", "", fpu_cos(cpu));
    FOR_EACH_STATE("patan", "", fpu_patan(cpu));
    FOR_EACH_STATE("clex", "", fpu_clex(cpu));
    FOR_EACH_STATE("stcw16", "", { uint16_t w; fpu_stcw16(cpu, &w); cpu->res = w; });
    FOR_EACH_STATE("stenv32", "", {
        struct env32_view env;
        fpu_stenv32(cpu, (struct fpu_env32 *) &env);
        cpu->res = env.control;
        cpu->op1 = env.status;
        cpu->op2 = env.tag;
    });
    FOR_EACH_STATE("save32", "", {
        struct state32_view st;
        fpu_save32(cpu, (struct fpu_state32 *) &st);
        cpu->res = st.env.control;
        cpu->op1 = st.env.status;
        memcpy(&cpu->fp[0], st.regs[0], 10);
        memcpy(&cpu->fp[1], st.regs[7], 10);
    });
    FOR_EACH_STATE("restore32", "", {
        struct state32_view st;
        fpu_save32(cpu, (struct fpu_state32 *) &st);
        st.env.status = 0x1234;
        st.env.control = 0x0fff;
        st.regs[3][0] ^= 0xff;
        fpu_restore32(cpu, (struct fpu_state32 *) &st);
    });

    // fyl2x needs the log2-safe guard
    for (int s = 0; s < n_states; s++) {
        struct cpu_state *cpu = &run;
        memcpy(cpu, &states[s], sizeof(*cpu));
        if (!log2_safe(ST(0)))
            continue;
        fpu_yl2x(cpu);
        printf("P yl2x %d", s);
        emit_state(cpu);
    }

    // ---- register-to-register arithmetic ----
    {
        static const char *names[] = { "add", "sub", "subr", "mul", "div", "divr" };
        int pairs[][2] = { { 0, 0 }, { 1, 0 }, { 0, 1 }, { 2, 1 }, { 3, 0 } };
        for (unsigned op = 0; op < 6; op++) {
            for (unsigned p = 0; p < sizeof(pairs) / sizeof(pairs[0]); p++) {
                int srci = pairs[p][0], dsti = pairs[p][1];
                char args[32];
                snprintf(args, sizeof(args), " %d %d", srci, dsti);
                for (int s = 0; s < n_states; s++) {
                    struct cpu_state *cpu = &run;
                    memcpy(cpu, &states[s], sizeof(*cpu));
                    switch (op) {
                        case 0: fpu_add(cpu, srci, dsti); break;
                        case 1: fpu_sub(cpu, srci, dsti); break;
                        case 2: fpu_subr(cpu, srci, dsti); break;
                        case 3: fpu_mul(cpu, srci, dsti); break;
                        case 4: fpu_div(cpu, srci, dsti); break;
                        case 5: fpu_divr(cpu, srci, dsti); break;
                    }
                    printf("P %s %d%s", names[op], s, args);
                    emit_state(cpu);
                }
            }
        }
    }

    // ---- memory-operand arithmetic and conversions ----
    for (int m = 0; m < N_MEM; m++) {
        char args[32];
        snprintf(args, sizeof(args), " %d", m);
        int16_t i16 = (int16_t) memvals[m].bits64;
        int32_t i32 = (int32_t) memvals[m].bits64;
        int64_t i64 = (int64_t) memvals[m].bits64;
        float f32;
        memcpy(&f32, &memvals[m].bits64, 4);
        double f64;
        memcpy(&f64, &memvals[m].bits64, 8);
        float80 f80v = memvals[m].f80;

        FOR_EACH_STATE("ild16", args, fpu_ild16(cpu, &i16));
        FOR_EACH_STATE("ild32", args, fpu_ild32(cpu, &i32));
        FOR_EACH_STATE("ild64", args, fpu_ild64(cpu, &i64));
        FOR_EACH_STATE("ldm32", args, fpu_ldm32(cpu, &f32));
        FOR_EACH_STATE("ldm64", args, fpu_ldm64(cpu, &f64));
        FOR_EACH_STATE("ldm80", args, fpu_ldm80(cpu, &f80v));

        FOR_EACH_STATE("ist16", args, { int16_t o; fpu_ist16(cpu, &o); cpu->res = (uint16_t) o; });
        FOR_EACH_STATE("ist32", args, { int32_t o; fpu_ist32(cpu, &o); cpu->res = (dword_t) o; });
        FOR_EACH_STATE("ist64", args, { int64_t o; fpu_ist64(cpu, &o); cpu->res = (dword_t) o; cpu->op1 = (dword_t) (o >> 32); });
        FOR_EACH_STATE("stm32", args, { float o; fpu_stm32(cpu, &o); uint32_t b; memcpy(&b, &o, 4); cpu->res = b; });
        FOR_EACH_STATE("stm64", args, { double o; fpu_stm64(cpu, &o); uint64_t b; memcpy(&b, &o, 8); cpu->res = (dword_t) b; cpu->op1 = (dword_t) (b >> 32); });
        FOR_EACH_STATE("stm80", args, { float80 o; fpu_stm80(cpu, &o); cpu->fp[1] = o; });

        FOR_EACH_STATE("icom16", args, fpu_icom16(cpu, &i16));
        FOR_EACH_STATE("icom32", args, fpu_icom32(cpu, &i32));
        FOR_EACH_STATE("comm32", args, fpu_comm32(cpu, &f32));
        FOR_EACH_STATE("comm64", args, fpu_comm64(cpu, &f64));

        FOR_EACH_STATE("iadd16", args, fpu_iadd16(cpu, &i16));
        FOR_EACH_STATE("isub16", args, fpu_isub16(cpu, &i16));
        FOR_EACH_STATE("isubr16", args, fpu_isubr16(cpu, &i16));
        FOR_EACH_STATE("imul16", args, fpu_imul16(cpu, &i16));
        FOR_EACH_STATE("idiv16", args, fpu_idiv16(cpu, &i16));
        FOR_EACH_STATE("idivr16", args, fpu_idivr16(cpu, &i16));
        FOR_EACH_STATE("iadd32", args, fpu_iadd32(cpu, &i32));
        FOR_EACH_STATE("isub32", args, fpu_isub32(cpu, &i32));
        FOR_EACH_STATE("isubr32", args, fpu_isubr32(cpu, &i32));
        FOR_EACH_STATE("imul32", args, fpu_imul32(cpu, &i32));
        FOR_EACH_STATE("idiv32", args, fpu_idiv32(cpu, &i32));
        FOR_EACH_STATE("idivr32", args, fpu_idivr32(cpu, &i32));
        FOR_EACH_STATE("addm32", args, fpu_addm32(cpu, &f32));
        FOR_EACH_STATE("subm32", args, fpu_subm32(cpu, &f32));
        FOR_EACH_STATE("subrm32", args, fpu_subrm32(cpu, &f32));
        FOR_EACH_STATE("mulm32", args, fpu_mulm32(cpu, &f32));
        FOR_EACH_STATE("divm32", args, fpu_divm32(cpu, &f32));
        FOR_EACH_STATE("divrm32", args, fpu_divrm32(cpu, &f32));
        FOR_EACH_STATE("addm64", args, fpu_addm64(cpu, &f64));
        FOR_EACH_STATE("subm64", args, fpu_subm64(cpu, &f64));
        FOR_EACH_STATE("subrm64", args, fpu_subrm64(cpu, &f64));
        FOR_EACH_STATE("mulm64", args, fpu_mulm64(cpu, &f64));
        FOR_EACH_STATE("divm64", args, fpu_divm64(cpu, &f64));
        FOR_EACH_STATE("divrm64", args, fpu_divrm64(cpu, &f64));
        FOR_EACH_STATE("ldcw16", args, { uint16_t w = (uint16_t) memvals[m].bits64; fpu_ldcw16(cpu, &w); });
    }

    // ---- flag macros from emu/cpu.h ----
    for (int s = 0; s < n_states; s++) {
        struct cpu_state *cpu = &run;
        memcpy(cpu, &states[s], sizeof(*cpu));
        printf("F %d %x %x %x %x %x %x %x %x %x %x %x %x", s,
               cpu->res, cpu->op1, cpu->op2, (unsigned) cpu->flags_res,
               (unsigned) cpu->cf, (unsigned) cpu->of,
               (unsigned) cpu->zf, (unsigned) cpu->sf,
               (unsigned) cpu->pf, (unsigned) cpu->af,
               (unsigned) cpu->cf_bit, (unsigned) cpu->of_bit);
        printf(" %d %d %d %d %d %d", ZF, SF, CF, OF, PF, AF);
        collapse_flags(cpu);
        printf(" %x %x %x %x %x %x %x %x",
               (unsigned) cpu->zf, (unsigned) cpu->sf, (unsigned) cpu->pf,
               (unsigned) cpu->af, (unsigned) cpu->cf_bit, (unsigned) cpu->of_bit,
               (unsigned) cpu->flags_res, cpu->eflags);
        expand_flags(cpu);
        printf(" %x %x %x\n",
               (unsigned) cpu->cf, (unsigned) cpu->of, (unsigned) cpu->flags_res);
    }

    return 0;
}

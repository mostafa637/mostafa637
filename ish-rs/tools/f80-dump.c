// Reference-output generator for the float80 differential test.
//
// This is NOT part of iSH. It compiles against the *unmodified* iSH sources
// (emu/float80.c + emu/float80.h) and dumps a deterministic table of results
// that the Rust port is then compared against bit-for-bit.
//
//   cc -O2 -I<ish-src> -o f80-dump f80-dump.c <ish-src>/emu/float80.c -lm
//   ./f80-dump > ../tests/fixtures/f80_reference.txt
//
// See tools/gen_f80_reference.sh for the scripted version.

#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>
#include <math.h>
#include <float.h>
#include "float80.h"

#define CURSED_BIT (1ul << 63)

// Emit one float80 as "signif signExp" (hex).
static void emit_f80(float80 f) {
    printf("%016llx %04x", (unsigned long long) f.signif, (unsigned) f.signExp);
}

// Deterministic pseudo-random significands, so the corpus is reproducible.
static uint64_t lcg_state = 0x12345678abcdef01ull;
static uint64_t lcg(void) {
    lcg_state = lcg_state * 6364136223846793005ull + 1442695040888963407ull;
    return lcg_state >> 11;
}

static float80 make_raw(uint64_t signif, uint16_t sign_exp) {
    float80 f;
    f.signif = signif;
    f.signExp = sign_exp;
    return f;
}

// ---- corpus -------------------------------------------------------------

static float80 ops[64];
static int n_ops;

static void add_f80(float80 f) {
    if (n_ops < (int) (sizeof(ops) / sizeof(ops[0])))
        ops[n_ops++] = f;
}

static void build_corpus(void) {
    int64_t ints[] = {
        0, 1, -1, 2, -2, 3, 7, -7, 10, -10, 100, 1000,
        123456789, -987654321,
        INT64_MAX, INT64_MIN,
        (int64_t) 1 << 53, ((int64_t) 1 << 53) + 1,
        (int64_t) 1 << 62, -((int64_t) 1 << 62),
    };
    for (unsigned i = 0; i < sizeof(ints) / sizeof(ints[0]); i++)
        add_f80(f80_from_int(ints[i]));

    double dbls[] = {
        0.0, -0.0, 1.0, -1.0, 0.5, -0.5, 2.5, 3.14, -2.718281828,
        0.1, -0.1, 1e10, -1e10, 1e-10, 1e300, -1e300,
        1.11253692925360069155e-308,   // the DENORMAL from emu/float80-test.c
        5e-324,                        // smallest denormal double
        DBL_MAX, DBL_MIN,
        INFINITY, -INFINITY, NAN,
    };
    for (unsigned i = 0; i < sizeof(dbls) / sizeof(dbls[0]); i++)
        add_f80(f80_from_double(dbls[i]));

    // raw encodings, including the awkward ones
    add_f80(F80_NAN);
    add_f80(F80_INF);
    add_f80(make_raw(0x8000000000000000ull, 0xffff));        // -inf
    add_f80(make_raw(0xffffffffffffffffull, 0x7ffe));        // largest finite
    add_f80(make_raw(CURSED_BIT, 0x0001));                   // smallest normal
    add_f80(make_raw(1, 0x0000));                            // smallest denormal
    add_f80(make_raw(0x7fffffffffffffffull, 0x0000));        // largest denormal
    add_f80(make_raw(0xc000000000000000ull, 0xffff));        // -nan
    add_f80(make_raw(CURSED_BIT - 1, 0x3fff));               // unormal (unsupported)
    add_f80(make_raw(0, 0x3fff));                            // unsupported "zero"

    // pseudo-random normals
    for (int i = 0; i < 8; i++) {
        uint64_t signif = lcg() | CURSED_BIT;
        uint16_t exp = (uint16_t) (lcg() % 0x7ffe) + 1;
        add_f80(make_raw(signif, (uint16_t) (exp | ((lcg() & 1) << 15))));
    }
}

// f80_log2 loops forever on operands it cannot reduce: it rejects NaN but not
// unsupported encodings, and +inf satisfies f80_gt(x, two) forever because
// f80_div(inf, 2) == inf. It also never terminates at all under round_up: the
// `bit` sequence 1, 1/2, 1/4, ... rounds the smallest denormal back up to
// itself, so `f80_gt(bit, zero)` stays true. All of that is upstream
// behaviour, so the corpus only feeds log2 the inputs that terminate.
static int log2_safe(float80 f, int mode) {
    return mode == round_to_nearest &&
        f80_is_supported(f) && !f80_isnan(f) && !f80_isinf(f);
}

int main(void) {
    build_corpus();
    printf("# f80 reference output, generated from unmodified iSH emu/float80.c\n");
    printf("# corpus size %d\n", n_ops);

    int modes[] = { round_to_nearest, round_down, round_up, round_chop };

    for (unsigned m = 0; m < 4; m++) {
        f80_rounding_mode = modes[m];

        // ---- binary ops over every ordered pair ----
        for (int i = 0; i < n_ops; i++) {
            for (int j = 0; j < n_ops; j++) {
                float80 a = ops[i], b = ops[j], r;
                printf("B %u add ", m); emit_f80(a); printf(" "); emit_f80(b);
                printf(" "); r = f80_add(a, b); emit_f80(r); printf("\n");

                printf("B %u sub ", m); emit_f80(a); printf(" "); emit_f80(b);
                printf(" "); r = f80_sub(a, b); emit_f80(r); printf("\n");

                printf("B %u mul ", m); emit_f80(a); printf(" "); emit_f80(b);
                printf(" "); r = f80_mul(a, b); emit_f80(r); printf("\n");

                printf("B %u div ", m); emit_f80(a); printf(" "); emit_f80(b);
                printf(" "); r = f80_div(a, b); emit_f80(r); printf("\n");

                printf("B %u mod ", m); emit_f80(a); printf(" "); emit_f80(b);
                printf(" "); r = f80_mod(a, b); emit_f80(r); printf("\n");

                printf("C %u lt ", m); emit_f80(a); printf(" "); emit_f80(b);
                printf(" %u\n", f80_lt(a, b));
                printf("C %u eq ", m); emit_f80(a); printf(" "); emit_f80(b);
                printf(" %u\n", f80_eq(a, b));
                printf("C %u uncomparable ", m); emit_f80(a); printf(" "); emit_f80(b);
                printf(" %u\n", f80_uncomparable(a, b));
            }
        }

        // ---- unary ops ----
        for (int i = 0; i < n_ops; i++) {
            float80 a = ops[i], r;

            printf("N %u neg ", m); emit_f80(a); printf(" ");
            r = f80_neg(a); emit_f80(r); printf("\n");
            printf("N %u abs ", m); emit_f80(a); printf(" ");
            r = f80_abs(a); emit_f80(r); printf("\n");
            printf("N %u round ", m); emit_f80(a); printf(" ");
            r = f80_round(a); emit_f80(r); printf("\n");
            printf("N %u sqrt ", m); emit_f80(a); printf(" ");
            r = f80_sqrt(a); emit_f80(r); printf("\n");
            if (log2_safe(a, modes[m])) {
                printf("N %u log2 ", m); emit_f80(a); printf(" ");
                r = f80_log2(a); emit_f80(r); printf("\n");
            }

            printf("U %u isnan ", m); emit_f80(a); printf(" %u\n", f80_isnan(a));
            printf("U %u isinf ", m); emit_f80(a); printf(" %u\n", f80_isinf(a));
            printf("U %u iszero ", m); emit_f80(a); printf(" %u\n", f80_iszero(a));
            printf("U %u isdenormal ", m); emit_f80(a); printf(" %u\n", f80_isdenormal(a));
            printf("U %u is_supported ", m); emit_f80(a); printf(" %u\n", f80_is_supported(a));

            {
                int exp;
                float80 signif;
                printf("X %u xtract ", m); emit_f80(a); printf(" ");
                f80_xtract(a, &exp, &signif);
                printf("%08x ", (unsigned) exp); emit_f80(signif); printf("\n");
            }

            printf("E %u to_double ", m); emit_f80(a); printf(" ");
            {
                double d = f80_to_double(a);
                uint64_t bits;
                memcpy(&bits, &d, sizeof(bits));
                printf("%016llx\n", (unsigned long long) bits);
            }

            int scales[] = { -1000, -65, -64, -63, -1, 0, 1, 63, 64, 127, 128, 129, 1000 };
            for (unsigned s = 0; s < sizeof(scales) / sizeof(scales[0]); s++) {
                printf("S %u scale ", m); emit_f80(a); printf(" %d ", scales[s]);
                r = f80_scale(a, scales[s]); emit_f80(r); printf("\n");
            }
        }

        // ---- integer conversions ----
        int64_t conv[] = {
            0, 1, -1, 2, -2, 7, 1000, -1000, 123456789, -987654321,
            INT64_MAX, INT64_MIN, INT64_MAX - 1,
            (int64_t) 1 << 53, ((int64_t) 1 << 53) + 1, (int64_t) 1 << 62,
        };
        for (unsigned i = 0; i < sizeof(conv) / sizeof(conv[0]); i++) {
            float80 f = f80_from_int(conv[i]);
            printf("I %u from_int %016llx ", m, (unsigned long long) (uint64_t) conv[i]);
            emit_f80(f); printf("\n");
        }
        for (int i = 0; i < n_ops; i++) {
            printf("T %u to_int ", m); emit_f80(ops[i]);
            printf(" %016llx\n", (unsigned long long) (uint64_t) f80_to_int(ops[i]));
        }

        // ---- double conversions ----
        double dconv[] = {
            0.0, -0.0, 1.0, -1.0, 0.5, -0.5, 3.14, -3.14, 0.1,
            1e300, -1e300, 1e-300, DBL_MAX, DBL_MIN, 5e-324,
            1.11253692925360069155e-308,
            INFINITY, -INFINITY, NAN, -NAN,
        };
        for (unsigned i = 0; i < sizeof(dconv) / sizeof(dconv[0]); i++) {
            uint64_t bits;
            memcpy(&bits, &dconv[i], sizeof(bits));
            float80 f = f80_from_double(dconv[i]);
            printf("D %u from_double %016llx ", m, (unsigned long long) bits);
            emit_f80(f); printf("\n");
        }
    }

    return 0;
}

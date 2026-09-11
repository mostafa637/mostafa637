// Reference-output generator for the modrm differential test.
//
// Compiles against the *unmodified* iSH emu/modrm.h (modrm_decode32 is
// `static inline`, so including the header is using the real thing) and decodes
// a corpus of instruction prefixes: every one of the 256 ModRM bytes, every one
// of the 256 SIB bytes under each of the four mod values, and three accesses
// placed so they fault against an unmapped page.
//
//   cc -O2 -I<ish-src> -o modrm-dump modrm-dump.c <ish-src>/emu/tlb.c
//
// The corpus is described in the fixture header - the backing-store fill, the
// unmapped page mask, every byte the driver wrote, and each starting ip - so
// tests/modrm_differential.rs rebuilds the identical memory image instead of
// duplicating this file's tables.

#include <stdio.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

#include "emu/cpu.h"
#include "emu/tlb.h"
#include "emu/modrm.h"

#define FAKE_PAGES 16
#define FAKE_BYTES (FAKE_PAGES << PAGE_BITS)
#define UNMAPPED_PAGE 8 // the page the fault cases run into

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

#define MAX_CASES 4096
static addr_t case_ip[MAX_CASES];
static int ncases;

static void put(addr_t at, const unsigned char *b, int n) {
    memcpy(backing + at, b, n);
    printf("# W %x", at);
    for (int i = 0; i < n; i++)
        printf(" %02x", b[i]);
    printf("\n");
}

static void add_case(addr_t ip) {
    if (ncases >= MAX_CASES) {
        fprintf(stderr, "too many cases\n");
        exit(1);
    }
    case_ip[ncases++] = ip;
}

static void build(void) {
    for (int i = 0; i < FAKE_BYTES; i++)
        backing[i] = (unsigned char) ((i * 7u + 3u) & 0xff);

    // every ModRM byte, in its own 8-byte slot so the displacement bytes the
    // decoder reads are fixed and known
    for (int m = 0; m < 256; m++) {
        unsigned char s[8] = { (unsigned char) m, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77 };
        addr_t at = 1024 + m * 8;
        put(at, s, 8);
        add_case(at);
    }
    // every SIB byte under each mod. mod=11 selects the register form, where
    // rm=100 is *not* a SIB escape - that case has to be covered too.
    for (int mod = 0; mod < 4; mod++) {
        for (int sib = 0; sib < 256; sib++) {
            unsigned char s[8] = { (unsigned char) ((mod << 6) | 4), (unsigned char) sib,
                                   0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34 };
            addr_t at = 4096 + (mod * 256 + sib) * 8;
            put(at, s, 8);
            add_case(at);
        }
    }
    // a disp32 straddling into the unmapped page
    {
        unsigned char s[1] = { 0b10000110 }; // mod=10 rm=esi -> disp32
        put(UNMAPPED_PAGE * PAGE_SIZE - 3, s, 1);
        add_case(UNMAPPED_PAGE * PAGE_SIZE - 3);
    }
    // a SIB byte that lands in the unmapped page
    {
        unsigned char s[1] = { 0b00000100 }; // mod=00 rm=100 -> SIB follows
        put(UNMAPPED_PAGE * PAGE_SIZE - 1, s, 1);
        add_case(UNMAPPED_PAGE * PAGE_SIZE - 1);
    }
    // the ModRM byte itself in the unmapped page
    add_case(UNMAPPED_PAGE * PAGE_SIZE);
}

int main(void) {
    mmu.ops = &fake_ops;
    mmu.changes = 0;
    memset(&tlb, 0, sizeof(tlb));
    tlb_refresh(&tlb, &mmu);

    printf("# modrm reference output, generated from unmodified iSH emu/modrm.h\n");
    printf("# pages %d fill 7 3 unmap %d\n", FAKE_PAGES, UNMAPPED_PAGE);
    build();
    printf("# cases %d\n", ncases);

    for (int i = 0; i < ncases; i++) {
        struct modrm m;
        addr_t ip = case_ip[i];
        // the C leaves index/shift untouched when there is no SIB, so start
        // from a known state; the port's Default matches it
        memset(&m, 0, sizeof(m));
        bool ok = modrm_decode32(&ip, &tlb, &m);
        printf("M %d %x %d %x %u %u %u %x %u %u\n", i, case_ip[i], ok, ip, m.type,
               m.reg, m.base, (unsigned) m.offset, m.index, m.shift);
    }
    return 0;
}

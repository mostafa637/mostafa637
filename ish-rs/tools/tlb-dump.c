// Reference-output generator for the mmu/tlb differential test.
//
// Compiles against the *unmodified* iSH emu/tlb.c + emu/tlb.h + emu/mmu.h, with
// a fake `mmu_ops` backend standing in for the real page table (which lives in
// kernel/memory.c and is not ported yet). Every step dumps the observable
// result *and* the whole TLB, so the Rust port has to reproduce not just the
// bytes but the caching behaviour: which entries are filled, whether an access
// hit or missed, what got marked dirty, and where a fault was recorded.
//
//   cc -O2 -I<ish-src> -o tlb-dump tlb-dump.c <ish-src>/emu/tlb.c
//
// The Rust side (tests/tlb_differential.rs) implements the same fake backend
// byte for byte, including the deterministic fill of the backing store.

#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>
#include "emu/tlb.h"
#include "emu/mmu.h"

// ---- fake backend -------------------------------------------------------

#define FAKE_PAGES 16
#define FAKE_BYTES (FAKE_PAGES << PAGE_BITS)

static unsigned char backing[FAKE_BYTES];
static unsigned ro_mask;      // bit set -> page is read-only
static unsigned unmap_mask;   // bit set -> page is not mapped at all
static unsigned translate_calls;

// The address space aliases every FAKE_PAGES pages onto the same backing
// store, so guest addresses far apart can still collide in the TLB.
static void *fake_translate(struct mmu *mmu, addr_t addr, int type) {
    (void) mmu;
    translate_calls++;
    unsigned idx = (addr >> PAGE_BITS) & (FAKE_PAGES - 1);
    if (unmap_mask & (1u << idx))
        return NULL;
    // note: MEM_WRITE_PTRACE is deliberately *not* blocked
    if (type == MEM_WRITE && (ro_mask & (1u << idx)))
        return NULL;
    return backing + ((size_t) idx << PAGE_BITS);
}

static struct mmu_ops fake_ops = { .translate = fake_translate };
static struct mmu mmu;
static struct tlb tlb;

// FNV-1a over the backing store, so the two implementations can be compared
// without shipping 64 KiB per step.
static unsigned long long hash_backing(void) {
    unsigned long long h = 14695981039346656037ull;
    for (size_t i = 0; i < FAKE_BYTES; i++) {
        h ^= backing[i];
        h *= 1099511628211ull;
    }
    return h;
}

// ---- steps --------------------------------------------------------------

enum {
    OP_READ, OP_WRITE, OP_READ_PTR, OP_WRITE_PTR,
    OP_REFRESH, OP_FLUSH, OP_BUMP, OP_RO, OP_UNMAP,
};

struct step {
    int op;
    unsigned addr;
    unsigned size;
};

// Hand-written so every interesting path is covered: hits, misses, cross-page
// accesses in both directions, read-only faults, unmapped faults, a fault on
// the *second* page of a cross-page access, TLB collisions between addresses
// 4 MiB apart, refresh/flush, and the unsigned wrap for size > PAGE_SIZE.
static struct step steps[] = {
    { OP_REFRESH, 0, 0 },
    { OP_READ, 0x0000, 4 },
    { OP_READ, 0x0004, 4 },          // hit
    { OP_WRITE, 0x0010, 4 },
    { OP_READ, 0x0010, 4 },          // read back what was written
    { OP_READ, 0x0ffc, 8 },          // cross-page 0 -> 1
    { OP_WRITE, 0x0ffe, 8 },         // cross-page write
    { OP_READ, 0x0ffe, 8 },
    { OP_READ_PTR, 0x2000, 0 },
    { OP_WRITE_PTR, 0x2004, 0 },
    { OP_RO, 1u << 3, 0 },           // page 3 becomes read-only
    { OP_READ, 0x3000, 4 },          // read of a read-only page works
    { OP_WRITE, 0x3000, 4 },         // write must fault
    { OP_WRITE_PTR, 0x3004, 0 },     // so must write_ptr
    { OP_READ, 0x3008, 4 },          // reads keep working
    { OP_RO, 0, 0 },
    { OP_UNMAP, 1u << 5, 0 },        // page 5 disappears
    { OP_READ, 0x5020, 4 },          // fault, records segfault_addr
    { OP_WRITE, 0x5020, 4 },
    { OP_READ, 0x4ffe, 8 },          // cross-page into the missing page
    { OP_WRITE, 0x4ffe, 8 },         // must not write the first half
    { OP_UNMAP, 0, 0 },
    { OP_READ, 0x1000, 4 },          // TLB index 1
    { OP_READ, 0x400000, 4 },        // same TLB index, evicts the entry above
    { OP_READ, 0x1000, 4 },          // missed again
    { OP_READ, 0x401000, 4 },        // index 0, no collision with 0x1000
    { OP_BUMP, 0, 0 },               // mmu->changes++
    { OP_REFRESH, 0, 0 },            // must flush
    { OP_REFRESH, 0, 0 },            // must be a no-op
    { OP_BUMP, 0, 0 },               // flush must resync mem_changes, not just
                                     // empty the entries
    { OP_FLUSH, 0, 0 },
    { OP_READ, 0x7000, 1 },
    { OP_WRITE, 0x7fff, 2 },         // cross-page 7 -> 8
    { OP_READ, 0x7fff, 2 },
    { OP_READ, 0xf000, 16 },
    { OP_WRITE, 0xfffc, 8 },         // last page, next page wraps to index 0
    { OP_READ, 0xfffc, 8 },
    { OP_READ, 0x0010, 5000 },       // size > PAGE_SIZE: the unsigned wrap takes
                                     // the fast path and copies straight from
                                     // the page base; a saturating subtract
                                     // would cross pages instead

    { OP_WRITE, 0x0010, 5000 },      // same wrap, on the write path: the
                                     // cross-page route would retranslate the
                                     // next page, which the call count catches
    { OP_WRITE, 0x0000, 4096 },      // exactly one page
    { OP_READ, 0x0000, 4096 },
    { OP_RO, 0xffffffff, 0 },        // everything read-only
    { OP_WRITE, 0x9000, 4 },
    { OP_RO, 0, 0 },
    { OP_UNMAP, 0xffffffff, 0 },     // nothing mapped
    { OP_READ, 0x0000, 4 },
    { OP_REFRESH, 0, 0 },
    { OP_UNMAP, 0, 0 },
    { OP_BUMP, 0, 0 },
    { OP_REFRESH, 0, 0 },
    { OP_READ, 0xb000, 4 },
    { OP_WRITE, 0xb004, 4 },
    { OP_READ, 0xb004, 4 },
    // write *hits*: the entry is already writable, so __tlb_write_ptr has to
    // set dirty_page itself - handle_miss is not involved
    { OP_WRITE, 0xb008, 4 },         // hit on page 11
    { OP_WRITE, 0x0020, 4 },         // hit on page 0 -> dirty_page 0
    { OP_WRITE, 0xb00c, 4 },         // hit on page 11 again -> dirty_page 0xb000
};
#define N_STEPS ((int) (sizeof(steps) / sizeof(steps[0])))

static void emit_tlb(void) {
    printf(" %x %x %x %x", tlb.dirty_page, tlb.segfault_addr,
           tlb.mem_changes, translate_calls);
    printf(" %016llx", hash_backing());
    for (unsigned i = 0; i < TLB_SIZE; i++) {
        struct tlb_entry *e = &tlb.entries[i];
        printf(" %x:%x:%d", e->page, e->page_if_writable, e->data_minus_addr != 0);
    }
}

int main(void) {
    for (size_t i = 0; i < FAKE_BYTES; i++)
        backing[i] = (unsigned char) ((i * 31u + 7u) & 0xff);

    mmu.ops = &fake_ops;
    mmu.changes = 0;
    memset(&tlb, 0, sizeof(tlb));

    printf("# tlb reference output, generated from unmodified iSH emu/tlb.c\n");
    printf("# steps %d, pages %d\n", N_STEPS, FAKE_PAGES);

    for (int s = 0; s < N_STEPS; s++) {
        struct step *st = &steps[s];
        int ok = 0;
        char buf[8192];
        unsigned out_size = 0;

        memset(buf, 0, sizeof(buf));
        switch (st->op) {
        case OP_READ:
            ok = tlb_read(&tlb, st->addr, buf, st->size);
            out_size = st->size;
            break;
        case OP_WRITE:
            for (unsigned i = 0; i < st->size; i++)
                buf[i] = (char) ((st->addr + i) & 0xff);
            ok = tlb_write(&tlb, st->addr, buf, st->size);
            break;
        case OP_READ_PTR:
            ok = __tlb_read_ptr(&tlb, st->addr) != NULL;
            break;
        case OP_WRITE_PTR:
            ok = __tlb_write_ptr(&tlb, st->addr) != NULL;
            break;
        case OP_REFRESH:
            tlb_refresh(&tlb, &mmu);
            ok = 1;
            break;
        case OP_FLUSH:
            tlb_flush(&tlb);
            ok = 1;
            break;
        case OP_BUMP:
            mmu.changes++;
            ok = 1;
            break;
        case OP_RO:
            ro_mask = st->addr;
            ok = 1;
            break;
        case OP_UNMAP:
            unmap_mask = st->addr;
            ok = 1;
            break;
        }

        static const char *names[] = {
            "read", "write", "read_ptr", "write_ptr",
            "refresh", "flush", "bump", "ro", "unmap",
        };
        printf("T %d %s %x %x %d", s, names[st->op], st->addr, st->size, ok);
        for (unsigned i = 0; i < out_size; i++)
            printf(" %02x", (unsigned char) buf[i]);
        // pad so the state always starts at a fixed token offset
        for (unsigned i = out_size; i < 16; i++)
            printf(" --");
        emit_tlb();
        printf("\n");
    }

    return 0;
}

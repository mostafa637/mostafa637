// Reference-output generator for the mmap differential test.
//
// Links the *unmodified* kernel/mmap.c on top of kernel/memory.c and
// kernel/user.c, and drives the address-space syscalls over a space built to
// reach their corners: hints that are and are not free, sub-page lengths,
// growing into an occupied hole, brk against an existing mapping, and the old
// six-argument mmap whose arguments live in guest memory.
//
//   cc -O2 -I<ish-src> -I<ish-rs>/tools/stub-include -o mmap-dump
//      mmap-dump.c <ish-src>/kernel/mmap.c <ish-src>/kernel/memory.c
//      <ish-src>/kernel/user.c <ish-src>/kernel/errno.c
//
// The return value is printed as a raw 32-bit hex word, because that is what
// these syscalls return: an address on success and a negated errno reinterpreted
// as addr_t on failure. `-EINVAL` comes back as ffffffea, and printing it in
// decimal would hide which half of the union a caller is looking at.

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/mman.h>

#include "kernel/calls.h"
#include "kernel/memory.h"
#include "kernel/mm.h"
#include "kernel/errno.h"
#include "kernel/signal.h"
#include "util/sync.h"

// ---- stubs ----

static long asbestos_invalidations;
void *asbestos_new(struct mmu *mmu) {
    (void) mmu;
    return (void *) 1;
}
void asbestos_free(void *asbestos) { (void) asbestos; }
void asbestos_invalidate_page(void *asbestos, page_t page) {
    (void) asbestos;
    (void) page;
    asbestos_invalidations++;
}

const char *vdso_data;

_Noreturn void die(const char *msg, ...) {
    fprintf(stderr, "die: %s\n", msg);
    exit(1);
}
void ish_printk(const char *msg, ...) { (void) msg; }
int current_pid(void) { return 0; }

__thread struct task *current;

static long signals_sent;
void send_signal(struct task *task, int sig, struct siginfo_ info) {
    (void) task;
    (void) sig;
    (void) info;
    signals_sent++;
}

static long fd_closes;
int fd_close(struct fd *fd) {
    (void) fd;
    fd_closes++;
    return 0;
}

// fs/fd.c. No mapping in this corpus is file-backed - the fd table is not ported
// - so f_get never finds anything, which is exactly what makes the C take its
// EBADF branch. The counter proves that is the only reason.
static long fd_lookups;
struct fd *f_get(fd_t f_flags) {
    (void) f_flags;
    fd_lookups++;
    return NULL;
}
struct fd *fd_retain(struct fd *fd) {
    (void) fd;
    return NULL;
}

// ---- the address space ----

static struct mm *mm;
static struct task task;

// these two live inside mmap.c, not in a header
#ifndef MREMAP_MAYMOVE_
#define MREMAP_MAYMOVE_ 1
#endif
#ifndef MREMAP_FIXED_
#define MREMAP_FIXED_ 2
#endif

static void dump_state(void) {
    struct mem *m = &mm->mem;
    printf("S main pgdir_used %d changes %llu brk %x start_brk %x refcount %u\n",
           m->pgdir_used, (unsigned long long) m->mmu.changes, mm->brk,
           mm->start_brk, mm->refcount);
    struct data *seen[64];
    int nseen = 0;
    for (int top = 0; top < MEM_PGDIR_SIZE; top++) {
        if (m->pgdir[top] == NULL)
            continue;
        for (int bottom = 0; bottom < MEM_PGDIR_SIZE; bottom++) {
            struct pt_entry *pt = &m->pgdir[top][bottom];
            if (pt->data == NULL)
                continue;
            int id = -1;
            for (int i = 0; i < nseen; i++)
                if (seen[i] == pt->data)
                    id = i;
            if (id < 0) {
                id = nseen;
                if (nseen >= 64)
                    abort();
                seen[nseen++] = pt->data;
            }
            printf("P main %x %x %zx %d %u\n", (top << 10) | bottom, pt->flags,
                   pt->offset, id, pt->data->refcount);
        }
    }
}

int main(void) {
    printf("# mmap reference output, generated from unmodified iSH kernel/mmap.c\n");
    printf("# returns are raw addr_t in hex: an address, or a negated errno\n");
    printf("# radix: addresses, lengths and offsets are hex; flags are decimal\n");

    mm = mm_new();
    // a heap that starts one page in, the way exec would leave it
    mm->start_brk = 0x1000 << PAGE_BITS;
    mm->brk = mm->start_brk;
    task.mm = mm;
    task.mem = &mm->mem;
    current = &task;

#define RET() printf("R %x\n", res)
    addr_t res;

    // ---- mmap2 ----
    // no hint at all: pt_find_hole picks somewhere near the top of the space
    printf("O M2 0 1000 7 %u 0\n", MMAP_PRIVATE | MMAP_ANONYMOUS);
    res = sys_mmap2(0, 0x1000, P_RWX, MMAP_PRIVATE | MMAP_ANONYMOUS, -1, 0);
    RET();
    dump_state();

    // an explicit hint that is free
    printf("O M2 400000 2000 7 %u 0\n", MMAP_ANONYMOUS);
    res = sys_mmap2(0x400 << PAGE_BITS, 0x2000, P_RWX, MMAP_ANONYMOUS, -1, 0);
    RET();
    dump_state();

    // the same hint again, not FIXED: the C's "find a hole instead" branch is
    // dead, so this replaces the mapping it just made
    printf("O M2 400000 1000 1 %u 0\n", MMAP_ANONYMOUS);
    res = sys_mmap2(0x400 << PAGE_BITS, 0x1000, P_READ, MMAP_ANONYMOUS, -1, 0);
    RET();
    dump_state();

    // FIXED over something that is there
    printf("O M2 400000 1000 7 %u 0\n", MMAP_FIXED | MMAP_ANONYMOUS);
    res = sys_mmap2(0x400 << PAGE_BITS, 0x1000, P_RWX, MMAP_FIXED | MMAP_ANONYMOUS, -1, 0);
    RET();
    dump_state();

    // SHARED carries P_SHARED into the page flags
    printf("O M2 0 1000 7 %u 0\n", MMAP_SHARED | MMAP_ANONYMOUS);
    res = sys_mmap2(0, 0x1000, P_RWX, MMAP_SHARED | MMAP_ANONYMOUS, -1, 0);
    RET();
    dump_state();

    // a length under one page rounds up here, unlike in mremap
    printf("O M2 0 1 7 %u 0\n", MMAP_ANONYMOUS);
    res = sys_mmap2(0, 1, P_RWX, MMAP_ANONYMOUS, -1, 0);
    RET();
    dump_state();

    // the refusals
    printf("O M2 0 0 7 %u 0\n", MMAP_ANONYMOUS);
    res = sys_mmap2(0, 0, P_RWX, MMAP_ANONYMOUS, -1, 0);
    RET();
    printf("O M2 400001 1000 7 %u 0\n", MMAP_ANONYMOUS);
    res = sys_mmap2((0x400 << PAGE_BITS) | 1, 0x1000, P_RWX, MMAP_ANONYMOUS, -1, 0);
    RET();
    printf("O M2 0 1000 107 %u 0\n", MMAP_ANONYMOUS);
    res = sys_mmap2(0, 0x1000, P_RWX | 0x100, MMAP_ANONYMOUS, -1, 0);
    RET();
    printf("O M2 0 1000 7 %u 0\n", MMAP_PRIVATE | MMAP_SHARED | MMAP_ANONYMOUS);
    res = sys_mmap2(0, 0x1000, P_RWX, MMAP_PRIVATE | MMAP_SHARED | MMAP_ANONYMOUS, -1, 0);
    RET();
    // not anonymous, and there is no fd table
    printf("O M2 0 1000 1 %u 0\n", MMAP_PRIVATE);
    res = sys_mmap2(0, 0x1000, P_READ, MMAP_PRIVATE, 3, 0);
    RET();
    dump_state();

    // ---- the old six-argument mmap, arguments in guest memory ----
    printf("O MN 100 1 7\n");
    res = pt_map_nothing(&mm->mem, 0x100, 1, P_RWX);
    RET();
    {
        dword_t args[6] = { 0, 0x1000, P_RWX, MMAP_PRIVATE | MMAP_ANONYMOUS, 0, 0 };
        if (user_write(0x100 << PAGE_BITS, args, sizeof args) != 0)
            abort();
        printf("O MM 100000 0 1000 7 %u 0 0\n", MMAP_PRIVATE | MMAP_ANONYMOUS);
        res = sys_mmap(0x100 << PAGE_BITS);
        RET();
        dump_state();
        // and a fault reading the arguments
        printf("O MM 900000 0 1000 7 %u 0 0\n", MMAP_PRIVATE | MMAP_ANONYMOUS);
        res = sys_mmap(0x900 << PAGE_BITS);
        RET();
    }

    // ---- mprotect ----
    // mprotect replaces the flags rather than or-ing them, so this clears
    // P_ANONYMOUS - which is what makes the mremap grow below fail
    printf("O MP 400000 1000 1\n");
    res = sys_mprotect(0x400 << PAGE_BITS, 0x1000, P_READ);
    RET();
    dump_state();
    printf("O MP 400000 1000 107\n");
    res = sys_mprotect(0x400 << PAGE_BITS, 0x1000, P_RWX | 0x100);
    RET();
    printf("O MP 400001 1000 7\n");
    res = sys_mprotect((0x400 << PAGE_BITS) | 1, 0x1000, P_RWX);
    RET();
    printf("O MP 900000 1000 7\n");
    res = sys_mprotect(0x900 << PAGE_BITS, 0x1000, P_RWX);
    RET();
    dump_state();

    // ---- mremap ----
    // both lengths under a page: PAGE, not PAGE_ROUND_UP, so nothing moves
    printf("O MR 400000 100 200 0\n");
    res = sys_mremap(0x400 << PAGE_BITS, 0x100, 0x200, 0);
    RET();
    dump_state();
    // grow in place - refused, because the mprotect above cleared P_ANONYMOUS
    // and the C only grows anonymous mappings
    printf("O MR 400000 1000 3000 0\n");
    res = sys_mremap(0x400 << PAGE_BITS, 0x1000, 0x3000, 0);
    RET();
    dump_state();
    // shrink back
    printf("O MR 400000 3000 1000 0\n");
    res = sys_mremap(0x400 << PAGE_BITS, 0x3000, 0x1000, 0);
    RET();
    dump_state();
    // grow where the next page is taken
    printf("O M2 401000 1000 7 %u 0\n", MMAP_FIXED | MMAP_ANONYMOUS);
    res = sys_mmap2(0x401 << PAGE_BITS, 0x1000, P_RWX, MMAP_FIXED | MMAP_ANONYMOUS, -1, 0);
    RET();
    printf("O MR 400000 1000 2000 0\n");
    res = sys_mremap(0x400 << PAGE_BITS, 0x1000, 0x2000, 0);
    RET();
    dump_state();
    // the refusals
    printf("O MR 400000 1000 2000 2\n");
    res = sys_mremap(0x400 << PAGE_BITS, 0x1000, 0x2000, MREMAP_FIXED_);
    RET();
    printf("O MR 400000 1000 2000 4\n");
    res = sys_mremap(0x400 << PAGE_BITS, 0x1000, 0x2000, 4);
    RET();
    printf("O MR 400001 1000 2000 0\n");
    res = sys_mremap((0x400 << PAGE_BITS) | 1, 0x1000, 0x2000, 0);
    RET();
    printf("O MR 900000 1000 2000 0\n");
    res = sys_mremap(0x900 << PAGE_BITS, 0x1000, 0x2000, 0);
    RET();
    // MAYMOVE is accepted but does nothing different without FIXED
    printf("O MR 400000 1000 1000 1\n");
    res = sys_mremap(0x400 << PAGE_BITS, 0x1000, 0x1000, MREMAP_MAYMOVE_);
    RET();

    // ---- munmap ----
    printf("O MU 401000 1000\n");
    res = sys_munmap(0x401 << PAGE_BITS, 0x1000);
    RET();
    dump_state();
    printf("O MU 401000 1000\n");
    res = sys_munmap(0x401 << PAGE_BITS, 0x1000);
    RET();
    printf("O MU 400001 1000\n");
    res = sys_munmap((0x400 << PAGE_BITS) | 1, 0x1000);
    RET();
    printf("O MU 400000 0\n");
    res = sys_munmap(0x400 << PAGE_BITS, 0);
    RET();
    // a length that is not a whole number of pages rounds up
    printf("O MU 400000 1001\n");
    res = sys_munmap(0x400 << PAGE_BITS, 0x1001);
    RET();
    dump_state();

    // ---- the cases the first draft of this corpus did not reach ----
    // mprotect with a length that is not a whole number of pages rounds up, so
    // both pages change; note this clears P_ANONYMOUS as a side effect
    printf("O M2 700000 2000 7 %u 0\n", MMAP_FIXED | MMAP_ANONYMOUS);
    res = sys_mmap2(0x700 << PAGE_BITS, 0x2000, P_RWX, MMAP_FIXED | MMAP_ANONYMOUS, -1, 0);
    RET();
    printf("O MP 700000 1001 1\n");
    res = sys_mprotect(0x700 << PAGE_BITS, 0x1001, P_READ);
    RET();
    dump_state();

    // a shrink that actually succeeds, on a mapping that is still anonymous
    printf("O M2 750000 3000 7 %u 0\n", MMAP_FIXED | MMAP_ANONYMOUS);
    res = sys_mmap2(0x750 << PAGE_BITS, 0x3000, P_RWX, MMAP_FIXED | MMAP_ANONYMOUS, -1, 0);
    RET();
    printf("O MR 750000 3000 1000 0\n");
    res = sys_mremap(0x750 << PAGE_BITS, 0x3000, 0x1000, 0);
    RET();
    dump_state();
    // sub-page old length and page-sized new length: PAGE(old_len) is 0 and
    // PAGE(new_len) is 1, so this is the *grow* branch, and the page it would
    // grow into is the one it already owns
    printf("O MR 750000 100 1000 0\n");
    res = sys_mremap(0x750 << PAGE_BITS, 0x100, 0x1000, 0);
    RET();
    dump_state();

    // munmap with a length that is not a whole number of pages rounds up, so
    // both pages go rather than just the first
    printf("O M2 800000 2000 7 %u 0\n", MMAP_FIXED | MMAP_ANONYMOUS);
    res = sys_mmap2(0x800 << PAGE_BITS, 0x2000, P_RWX, MMAP_FIXED | MMAP_ANONYMOUS, -1, 0);
    RET();
    printf("O MU 800000 1001\n");
    res = sys_munmap(0x800 << PAGE_BITS, 0x1001);
    RET();
    dump_state();

    // ---- brk ----
    printf("O BR %x\n", mm->start_brk + 0x2000);
    res = sys_brk(mm->start_brk + 0x2000);
    RET();
    dump_state();
    // "if the brk is 0x2000, page 0x2000 shouldn't be mapped, but it should be
    // if the brk is 0x2001"
    printf("O BR %x\n", mm->start_brk + 0x2001);
    res = sys_brk(mm->start_brk + 0x2001);
    RET();
    dump_state();
    // shrink back
    printf("O BR %x\n", mm->start_brk);
    res = sys_brk(mm->start_brk);
    RET();
    dump_state();
    // below start_brk: refused, and the old value comes back
    printf("O BR %x\n", mm->start_brk - 1);
    res = sys_brk(mm->start_brk - 1);
    RET();
    dump_state();
    // growing by one page - nothing is in the way, so this succeeds
    printf("O BR %x\n", mm->start_brk + 0x1000);
    res = sys_brk(mm->start_brk + 0x1000);
    RET();
    // Refused, and the reason is the shrink above: brk's shrink branch unmaps
    // PAGE(old_brk) - PAGE(new_brk) pages, so shrinking from 0x1002001 to
    // 0x1000000 left page 0x1002 mapped. That stray page is inside the range
    // this request needs, so pt_is_hole says no.
    printf("O BR %x\n", mm->start_brk + 0x200000);
    res = sys_brk(mm->start_brk + 0x200000);
    RET();
    dump_state();

    // ---- the syscalls Linux lets do nothing ----
    printf("O MA 400000 1000 4\n");
    res = sys_madvise(0x400 << PAGE_BITS, 0x1000, 4);
    RET();
    printf("O ML 0 0\n");
    res = sys_mlock(0, 0);
    RET();
    printf("O MS 0 0 0\n");
    res = sys_msync(0, 0, 0);
    RET();

    // ---- mm refcounting ----
    printf("O RT\n");
    mm_retain(mm);
    res = mm->refcount;
    RET();
    printf("O RL\n");
    mm_release(mm);
    res = mm->refcount;
    RET();

    printf("# asbestos_invalidations %ld fd_closes %ld fd_lookups %ld signals_sent %ld\n",
           asbestos_invalidations, fd_closes, fd_lookups, signals_sent);
    return 0;
}

// Reference-output generator for the memory differential test.
//
// Links the *unmodified* kernel/memory.c and drives its page table through a
// scripted sequence of operations, dumping the resulting state after each one.
// The parts of iSH that memory.c calls into but that are not being ported here
// (the asbestos block cache, the vdso, the fd table) are stubbed - see below for
// exactly what each stub gives up.
//
//   cc -O2 -I<ish-src> -o memory-dump memory-dump.c <ish-src>/kernel/memory.c
//
// The script is printed into the fixture as it runs, so
// tests/memory_differential.rs replays the same operations instead of
// duplicating this file's tables.
//
// Nothing process-local is printed. Host pointers are turned into small integer
// identities assigned in first-appearance order, so "these two pages share one
// backing object" survives the trip; the addresses themselves do not.

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/mman.h>

#include "kernel/memory.h"
#include "kernel/errno.h"
#include "kernel/signal.h"
#include "util/sync.h"

// ---- stubs for the parts of iSH this port does not include ----

// asbestos is the JIT. memory.c only tells it "throw away any compiled code for
// this page", so a counter is a faithful stand-in: it records that the
// invalidation happened without needing a code generator.
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

// kernel/vdso.c provides this as a real ELF image in the data segment. memory.c
// only uses it to decide whether a mapping came from mmap (and so may be
// munmapped) - a NULL stand-in makes every mapping look mmap'd, which is what
// the corpus uses.
const char *vdso_data;

// kernel/log.c. The generator has no log to write to; die() is only reached if
// a munmap fails, which would mean the host refused to unmap memory this
// process owns.
_Noreturn void die(const char *msg, ...) {
    fprintf(stderr, "die: %s\n", msg);
    exit(1);
}
void ish_printk(const char *msg, ...) { (void) msg; }
int current_pid(void) { return 0; }

// kernel/errno.c's errno_map() signals SIGPIPE when the host errno is EPIPE.
// Nothing in this corpus writes to a pipe, so these are stubs that count; if
// the count at the end is not zero the errno path was reached after all.
__thread struct task *current;
static long signals_sent;
void send_signal(struct task *task, int sig, struct siginfo_ info) {
    (void) task;
    (void) sig;
    (void) info;
    signals_sent++;
}

// fs/fd.c. No mapping in the corpus carries an fd, so this is never reached;
// the counter proves that if it ever is.
static long fd_closes;
void fd_close(struct fd *fd) {
    (void) fd;
    fd_closes++;
}

// ---- the state dump ----

static struct mem mem;
static struct mem child;

static void dump_mem(const char *tag, struct mem *m) {
    printf("S %s pgdir_used %d changes %llu\n", tag, m->pgdir_used,
           (unsigned long long) m->mmu.changes);
    for (page_t page = 0; page < MEM_PAGES; page++) {
        struct pt_entry *pt = mem_pt(m, page);
        if (pt == NULL)
            continue;
        // identity is "which mapping does this page share its backing with",
        // computed inside this address space only
        int id = 0, found = -1;
        for (page_t q = 0; q < MEM_PAGES && found < 0; q++) {
            struct pt_entry *qt = mem_pt(m, q);
            if (qt == NULL)
                continue;
            if (qt->data == pt->data)
                found = id;
            else {
                int earlier = 0;
                for (page_t r = 0; r < q; r++) {
                    struct pt_entry *rt = mem_pt(m, r);
                    if (rt != NULL && rt->data == qt->data)
                        earlier = 1;
                }
                if (!earlier)
                    id++;
            }
        }
        printf("P %s %x %x %zx %d %u\n", tag, page, pt->flags, pt->offset, found,
               pt->data->refcount);
    }
}

// ---- the operations ----

static unsigned long rng_state = 0x2545f4914f6cdd1dUL;
static unsigned long rnd(void) {
    rng_state ^= rng_state << 13;
    rng_state ^= rng_state >> 7;
    rng_state ^= rng_state << 17;
    return rng_state;
}

static void *fresh(size_t bytes) {
    void *p = mmap(NULL, bytes, PROT_READ | PROT_WRITE, MAP_PRIVATE | MAP_ANONYMOUS, 0, 0);
    if (p == MAP_FAILED) {
        perror("mmap");
        exit(1);
    }
    // deterministic contents so a CoW copy is checkable
    memset(p, (int) ((uintptr_t) p >> 12) & 0xff, bytes);
    return p;
}

int main(void) {
    real_page_size = (size_t) sysconf(_SC_PAGESIZE);
    mem_init(&mem);
    mem_init(&child);

    printf("# memory reference output, generated from unmodified iSH kernel/memory.c\n");
    printf("# real_page_size %zu\n", real_page_size);

#define RET(fmt, ...) printf("R " fmt "\n", ##__VA_ARGS__)

    // ---- fixed corner cases first ----
    printf("O MN 100 4 %u\n", P_RWX);
    RET("%d", pt_map_nothing(&mem, 0x100, 4, P_RWX));
    dump_mem("main", &mem);

    printf("O IH 100 4\n");
    RET("%d", pt_is_hole(&mem, 0x100, 4));
    printf("O IH 104 4\n");
    RET("%d", pt_is_hole(&mem, 0x104, 4));
    printf("O IH 102 2\n");
    RET("%d", pt_is_hole(&mem, 0x102, 2));

    // an unmap that is only partly mapped must fail and change nothing
    printf("O UN 102 4\n");
    RET("%d", pt_unmap(&mem, 0x102, 4));
    dump_mem("main", &mem);
    // ...and unmap_always does it anyway
    printf("O UA 102 4\n");
    RET("%d", pt_unmap_always(&mem, 0x102, 4));
    dump_mem("main", &mem);

    // set_flags on an unmapped page
    printf("O SF 200 1 %u\n", P_READ);
    RET("%d", pt_set_flags(&mem, 0x200, 1, P_READ));

    // write-protect, then a write through mem_ptr, then a read
    printf("O MN 300 2 %u\n", P_READ);
    RET("%d", pt_map_nothing(&mem, 0x300, 2, P_READ));
    printf("O PT %x %d %s\n", 0x300 << PAGE_BITS, MEM_WRITE, "main");
    read_wrlock(&mem.lock);
    RET("%d", mem_ptr(&mem, 0x300 << PAGE_BITS, MEM_WRITE) != NULL);
    read_wrunlock(&mem.lock);
    printf("O PT %x %d %s\n", 0x300 << PAGE_BITS, MEM_READ, "main");
    read_wrlock(&mem.lock);
    RET("%d", mem_ptr(&mem, 0x300 << PAGE_BITS, MEM_READ) != NULL);
    read_wrunlock(&mem.lock);
    printf("O SG %x\n", 0x300 << PAGE_BITS);
    RET("%d", mem_segv_reason(&mem, 0x300 << PAGE_BITS));
    printf("O SG %x\n", 0x999 << PAGE_BITS);
    RET("%d", mem_segv_reason(&mem, 0x999 << PAGE_BITS));

    // make it writable and write again
    printf("O SF 300 2 %u\n", P_RWX);
    RET("%d", pt_set_flags(&mem, 0x300, 2, P_RWX));
    dump_mem("main", &mem);
    printf("O PT %x %d %s\n", 0x300 << PAGE_BITS, MEM_WRITE, "main");
    read_wrlock(&mem.lock);
    RET("%d", mem_ptr(&mem, 0x300 << PAGE_BITS, MEM_WRITE) != NULL);
    read_wrunlock(&mem.lock);
    dump_mem("main", &mem);

    // copy-on-write into the child, then write in both
    printf("O CW 300 2\n");
    RET("%d", pt_copy_on_write(&mem, &child, 0x300, 2));
    dump_mem("main", &mem);
    dump_mem("child", &child);
    printf("O PT %x %d %s\n", 0x300 << PAGE_BITS, MEM_WRITE, "main");
    read_wrlock(&mem.lock);
    RET("%d", mem_ptr(&mem, 0x300 << PAGE_BITS, MEM_WRITE) != NULL);
    read_wrunlock(&mem.lock);
    printf("O PT %x %d %s\n", 0x300 << PAGE_BITS, MEM_WRITE, "child");
    read_wrlock(&child.lock);
    RET("%d", mem_ptr(&child, 0x300 << PAGE_BITS, MEM_WRITE) != NULL);
    read_wrunlock(&child.lock);
    dump_mem("main", &mem);
    dump_mem("child", &child);

    // a ptrace write bypasses the write bit and leaves the page CoW
    printf("O MN 400 1 %u\n", P_READ);
    RET("%d", pt_map_nothing(&mem, 0x400, 1, P_READ));
    printf("O PT %x %d %s\n", 0x400 << PAGE_BITS, MEM_WRITE_PTRACE, "main");
    read_wrlock(&mem.lock);
    RET("%d", mem_ptr(&mem, 0x400 << PAGE_BITS, MEM_WRITE_PTRACE) != NULL);
    read_wrunlock(&mem.lock);
    dump_mem("main", &mem);

    // grows-down: the region above a hole decides whether the hole gets mapped
    printf("O MN 500 1 %u\n", P_RWX | P_GROWSDOWN);
    RET("%d", pt_map_nothing(&mem, 0x500, 1, P_RWX | P_GROWSDOWN));
    printf("O PT %x %d %s\n", 0x4ff << PAGE_BITS, MEM_READ, "main");
    read_wrlock(&mem.lock);
    RET("%d", mem_ptr(&mem, 0x4ff << PAGE_BITS, MEM_READ) != NULL);
    read_wrunlock(&mem.lock);
    dump_mem("main", &mem);
    // and one with nothing above it stays unmapped
    printf("O PT %x %d %s\n", 0xf0000 << PAGE_BITS, MEM_READ, "main");
    read_wrlock(&mem.lock);
    RET("%d", mem_ptr(&mem, 0xf0000 << PAGE_BITS, MEM_READ) != NULL);
    read_wrunlock(&mem.lock);

    // hole finding, on an address space with known holes
    printf("O FH 4\n");
    RET("%x", pt_find_hole(&mem, 4));
    printf("O FH 1\n");
    RET("%x", pt_find_hole(&mem, 1));

    // mem_next_page skips unallocated page directories
    printf("O NP 100\n");
    {
        page_t p = 0x100;
        mem_next_page(&mem, &p);
        RET("%x", p);
    }
    printf("O NP 103\n");
    {
        page_t p = 0x103;
        mem_next_page(&mem, &p);
        RET("%x", p);
    }

    // mapping over an existing mapping replaces it and drops the old refcount
    {
        void *m2 = fresh(2 << PAGE_BITS);
        printf("O MP %x %u %zx %u\n", 0x100, 2, (size_t) 0, P_RWX);
        RET("%d", pt_map(&mem, 0x100, 2, m2, 0, P_RWX));
    }
    dump_mem("main", &mem);

    // an offset into the backing object, which is what mem_ptr adds
    {
        // every page-number field in the script is hex, so print the real
        // offset rather than spelling it out in the format string
        const size_t off = 0x1000;
        void *m3 = fresh((2 << PAGE_BITS) + off);
        printf("O MP %x %u %zx %u\n", 0x600, 2, off, P_RWX);
        RET("%d", pt_map(&mem, 0x600, 2, m3, off, P_RWX));
    }
    dump_mem("main", &mem);

    // ---- then a deterministic pseudo-random walk over the same API ----
    for (int i = 0; i < 400; i++) {
        unsigned long r = rnd();
        page_t start = 0x1000 + (page_t) (r % 0x2000);
        pages_t pages = 1 + (pages_t) ((r >> 20) % 8);
        int which = (int) ((r >> 40) % 6);
        unsigned flags = (unsigned) ((r >> 30) % 4) == 0 ? P_READ : P_RWX;
        switch (which) {
        case 0:
            printf("O MN %x %u %u\n", start, pages, flags);
            RET("%d", pt_map_nothing(&mem, start, pages, flags));
            break;
        case 1:
            printf("O UN %x %u\n", start, pages);
            RET("%d", pt_unmap(&mem, start, pages));
            break;
        case 2:
            printf("O UA %x %u\n", start, pages);
            RET("%d", pt_unmap_always(&mem, start, pages));
            break;
        case 3:
            printf("O SF %x %u %u\n", start, pages, flags);
            RET("%d", pt_set_flags(&mem, start, pages, (int) flags));
            break;
        case 4:
            printf("O IH %x %u\n", start, pages);
            RET("%d", pt_is_hole(&mem, start, pages));
            break;
        default:
            printf("O PT %x %d %s\n", start << PAGE_BITS, MEM_WRITE, "main");
            read_wrlock(&mem.lock);
            RET("%d", mem_ptr(&mem, start << PAGE_BITS, MEM_WRITE) != NULL);
            read_wrunlock(&mem.lock);
            break;
        }
    }
    dump_mem("main", &mem);
    printf("# asbestos_invalidations %ld fd_closes %ld signals_sent %ld\n",
           asbestos_invalidations, fd_closes, signals_sent);

    mem_destroy(&child);
    mem_destroy(&mem);
    return 0;
}

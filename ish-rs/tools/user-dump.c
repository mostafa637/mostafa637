// Reference-output generator for the user-memory differential test.
//
// Links the *unmodified* kernel/user.c (and the kernel/memory.c it sits on) and
// drives user_read/user_write/user_write_task_ptrace/user_read_string/
// user_write_string over an address space built to make their corner cases
// reachable: ranges that span pages, ranges that straddle a hole, ranges that
// straddle a read-only page, copy-on-write pages, and a grow-down region.
//
//   cc -O2 -I<ish-src> -I<ish-rs>/tools/stub-include -o user-dump
//      user-dump.c <ish-src>/kernel/user.c <ish-src>/kernel/memory.c
//      <ish-src>/kernel/errno.c
//
// The script is printed into the fixture as it runs, so
// tests/user_differential.rs replays the same operations instead of
// duplicating this file's tables.
//
// What makes this generator different from memory-dump.c is that it dumps
// *bytes*, on both sides of every operation:
//
//   B <hex>                     the host buffer after the call
//   G <tag> <page:x> <f:x> <hex> the guest pages the call could have touched
//
// Both matter because these functions do not roll back: __user_write_task
// memcpy's each page as it goes, so a fault on page N leaves pages 0..N-1
// written. A return-code-only comparison would not see that.

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/mman.h>

#include "kernel/calls.h"
#include "kernel/memory.h"
#include "kernel/errno.h"
#include "kernel/signal.h"
#include "util/sync.h"

// ---- stubs for the parts of iSH this port does not include ----
// Same set as memory-dump.c; see there for what each one gives up.

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

// kernel/task.c owns the per-thread current task; user.c reads it in every
// function without a _task suffix
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

// ---- the address spaces ----

static struct mem main_mem;
static struct mem child_mem;
static struct task main_task;
static struct task child_task;

static const char *tag_of(struct mem *m) { return m == &child_mem ? "child" : "main"; }

static void *fresh(size_t size) {
    void *p = mmap(NULL, size, PROT_READ | PROT_WRITE,
                   MAP_PRIVATE | MAP_ANONYMOUS, -1, 0);
    if (p == MAP_FAILED) {
        perror("mmap");
        exit(1);
    }
    return p;
}

// the pattern every filled mapping and every write buffer uses, so the Rust side
// can regenerate it from the two numbers on the op line
static unsigned char pat(size_t j, unsigned seed) {
    return (unsigned char) ((j * 31 + seed) & 0xff);
}
static unsigned char wpat(size_t j, unsigned seed) {
    return (unsigned char) ((j * 7 + seed) & 0xff);
}

// ---- the dumps ----

static void dump_state(struct mem *m) {
    const char *tag = tag_of(m);
    printf("S %s pgdir_used %d changes %llu\n", tag, m->pgdir_used,
           (unsigned long long) m->mmu.changes);
    // walk the directory rather than all 1M pages: the corpus maps a handful
    struct data *seen[64];
    int nseen = 0;
    for (int top = 0; top < MEM_PGDIR_SIZE; top++) {
        if (m->pgdir[top] == NULL)
            continue;
        for (int bottom = 0; bottom < MEM_PGDIR_SIZE; bottom++) {
            struct pt_entry *pt = &m->pgdir[top][bottom];
            if (pt->data == NULL)
                continue;
            page_t page = (top << 10) | bottom;
            int id = -1;
            for (int i = 0; i < nseen; i++)
                if (seen[i] == pt->data)
                    id = i;
            if (id < 0) {
                id = nseen;
                seen[nseen++] = pt->data;
                if (nseen > 64)
                    abort();
            }
            printf("P %s %x %x %zx %d %u\n", tag, page, pt->flags, pt->offset, id,
                   pt->data->refcount);
        }
    }
}

// dump the bytes of [start, start+pages), reading the backing object directly
// rather than through mem_ptr so the dump itself cannot fault or grow anything
static void dump_bytes(struct mem *m, page_t start, pages_t pages) {
    const char *tag = tag_of(m);
    for (page_t page = start; page < start + pages; page++) {
        struct pt_entry *pt = mem_pt(m, page);
        if (pt == NULL) {
            printf("G %s %x hole\n", tag, page);
            continue;
        }
        const unsigned char *p = (const unsigned char *) pt->data->data + pt->offset;
        printf("G %s %x %x", tag, page, pt->flags);
        for (size_t i = 0; i < (size_t) 1 << PAGE_BITS; i++)
            printf(" %02x", p[i]);
        printf("\n");
    }
}

static void dump_buf(const unsigned char *buf, size_t n) {
    printf("B");
    for (size_t i = 0; i < n; i++)
        printf(" %02x", buf[i]);
    printf("\n");
}

// ---- the operations ----

#define RET(fmt, ...) printf("R " fmt "\n", __VA_ARGS__)

// how many pages an operation starting at addr and running count bytes spans
static page_t first_page(addr_t addr) { return PAGE(addr); }
static pages_t span(addr_t addr, size_t count) {
    if (count == 0)
        return 1; // still worth dumping: the op touched nothing but must not fault
    // PAGE(addr + count - 1) is computed in size_t, so a range running off the
    // top of the address space yields a page number that does not exist. The C's
    // mem_pt would index pgdir[1024], one past the array; clamp instead.
    page_t last = PAGE(addr + count - 1);
    if (last >= MEM_PAGES)
        last = MEM_PAGES - 1;
    return last - PAGE(addr) + 1;
}

static void op_read(const char *which, struct task *task, addr_t addr, size_t count) {
    unsigned char buf[16384];
    memset(buf, 0xaa, sizeof buf);
    printf("O %s %s %x %zu\n", which, tag_of(task->mem), addr, count);
    int res;
    if (strcmp(which, "RD") == 0) {
        // the no-suffix form follows `current`, so the fixture must not claim
        // an address space it is not writing to
        if (task->mem != current->mem)
            abort();
        res = user_read(addr, buf, count);
    }
    else
        res = user_read_task(task, addr, buf, count);
    RET("%d", res);
    dump_buf(buf, count);
    dump_bytes(task->mem, first_page(addr), span(addr, count));
    dump_state(task->mem);
}

static void op_write(const char *which, struct task *task, addr_t addr, size_t count,
                     unsigned seed) {
    unsigned char buf[16384];
    for (size_t i = 0; i < count; i++)
        buf[i] = wpat(i, seed);
    printf("O %s %s %x %zu %u\n", which, tag_of(task->mem), addr, count, seed);
    int res;
    if (strcmp(which, "WR") == 0) {
        if (task->mem != current->mem)
            abort();
        res = user_write(addr, buf, count);
    } else if (strcmp(which, "WP") == 0)
        res = user_write_task_ptrace(task, addr, buf, count);
    else
        res = user_write_task(task, addr, buf, count);
    RET("%d", res);
    dump_bytes(task->mem, first_page(addr), span(addr, count));
    dump_state(task->mem);
}

static void op_read_string(addr_t addr, size_t max) {
    char buf[64];
    memset(buf, 0xaa, sizeof buf);
    printf("O RS %x %zu\n", addr, max);
    int res = user_read_string(addr, buf, max);
    RET("%d", res);
    dump_buf((const unsigned char *) buf, max);
    dump_bytes(current->mem, first_page(addr), span(addr, max));
    dump_state(current->mem);
}

static void op_write_string(addr_t addr, const char *s) {
    printf("O WS %x", addr);
    for (const char *p = s; ; p++) {
        printf(" %02x", (unsigned char) *p);
        if (*p == '\0')
            break;
    }
    printf("\n");
    int res = user_write_string(addr, s);
    RET("%d", res);
    dump_bytes(current->mem, first_page(addr), strlen(s) + 1);
    dump_state(current->mem);
}

int main(void) {
    printf("# user reference output, generated from unmodified iSH kernel/user.c\n");
    printf("# real_page_size %zu\n", real_page_size);
    printf("# pattern mapping: byte j = (j*31+seed)&0xff; write: (j*7+seed)&0xff\n");
    // state the radix once, because a field that silently changes base is how
    // this fixture's first draft lied to its own test
    printf("# radix: addresses, page numbers, offsets and bytes are hex;\n");
    printf("#        counts, page counts, flags and seeds are decimal\n");

    mem_init(&main_mem);
    mem_init(&child_mem);
    main_task.mem = &main_mem;
    child_task.mem = &child_mem;
    current = &main_task;

    // ---- the address space, described in the fixture so the port can build it
    // 0x100: one writable page, filled from seed 1
    {
        unsigned char *m = fresh(1 << PAGE_BITS);
        for (size_t i = 0; i < (1u << PAGE_BITS); i++)
            m[i] = pat(i, 1);
        printf("O MF 100 1 %u %u\n", P_RWX, 1);
        RET("%d", pt_map(&main_mem, 0x100, 1, m, 0, P_RWX));
    }
    // 0x200: two read-only pages, filled from seed 2
    {
        unsigned char *m = fresh(2 << PAGE_BITS);
        for (size_t i = 0; i < (2u << PAGE_BITS); i++)
            m[i] = pat(i, 2);
        printf("O MF 200 2 %u %u\n", P_READ, 2);
        RET("%d", pt_map(&main_mem, 0x200, 2, m, 0, P_READ));
    }
    // 0x300: two writable pages, filled from seed 3, then shared with the child
    {
        unsigned char *m = fresh(2 << PAGE_BITS);
        for (size_t i = 0; i < (2u << PAGE_BITS); i++)
            m[i] = pat(i, 3);
        printf("O MF 300 2 %u %u\n", P_RWX, 3);
        RET("%d", pt_map(&main_mem, 0x300, 2, m, 0, P_RWX));
    }
    // 0x400 and 0x401 stay holes, on purpose
    // 0x500: a grow-down region
    printf("O MN 500 1 %u\n", P_RWX | P_GROWSDOWN);
    RET("%d", pt_map_nothing(&main_mem, 0x500, 1, P_RWX | P_GROWSDOWN));
    // 0x600: a zeroed writable page, the target for the string operations
    printf("O MN 600 1 %u\n", P_RWX);
    RET("%d", pt_map_nothing(&main_mem, 0x600, 1, P_RWX));
    // the fork
    printf("O CW 300 2\n");
    RET("%d", pt_copy_on_write(&main_mem, &child_mem, 0x300, 2));
    dump_state(&main_mem);
    dump_state(&child_mem);

    // ---- reads ----
    // inside one page
    op_read("RD", &main_task, 0x100 << PAGE_BITS, 16);
    // spanning a page boundary
    op_read("RD", &main_task, (0x100 << PAGE_BITS) + 4090, 16);
    // running off the end of the mapping into the hole at 0x101
    op_read("RD", &main_task, (0x100 << PAGE_BITS) + 4090, 32);
    // running off the end into the hole at 0x400
    op_read("RD", &main_task, (0x201 << PAGE_BITS) + 4090, 32);
    // starting in a hole
    op_read("RD", &main_task, 0x400 << PAGE_BITS, 8);
    // zero bytes
    op_read("RD", &main_task, 0x100 << PAGE_BITS, 0);
    // off the top of the address space, where addr+count is 64-bit and
    // chunk_end is not
    op_read("RD", &main_task, 0xfffff000, 8192);
    // the read-only pages
    op_read("RD", &main_task, (0x200 << PAGE_BITS) + 4080, 32);
    // the explicit-task form, against the child's copy-on-write pages
    op_read("RT", &child_task, 0x300 << PAGE_BITS, 16);
    // reading a grow-down region does not grow it in from below... or does it?
    // mem_ptr grows down for reads too, which is what this pins down
    op_read("RD", &main_task, (0x4ff << PAGE_BITS) + 4080, 32);

    // ---- writes ----
    // inside one page
    op_write("WR", &main_task, (0x100 << PAGE_BITS) + 16, 16, 10);
    // spanning a page boundary, both pages writable
    op_write("WR", &main_task, (0x300 << PAGE_BITS) + 4090, 16, 11);
    // into the read-only mapping: faults, and nothing is written
    op_write("WR", &main_task, (0x200 << PAGE_BITS) + 4080, 32, 12);
    // straddling writable-then-readonly: the first page IS written before the
    // fault, because __user_write_task does not roll back
    op_write("WR", &main_task, (0x100 << PAGE_BITS) + 4090, 16, 13);
    // off the end of the address space
    op_write("WR", &main_task, 0xfffff000, 8192, 14);
    // zero bytes
    op_write("WR", &main_task, 0x100 << PAGE_BITS, 0, 15);
    // the explicit-task form against the child, breaking its copy-on-write
    op_write("WT", &child_task, 0x300 << PAGE_BITS, 16, 16);
    // the same write again: the child's page is private now, so it just lands
    op_write("WT", &child_task, 0x300 << PAGE_BITS, 16, 17);
    // and the parent's page must still hold what it held before the fork
    op_read("RT", &main_task, 0x300 << PAGE_BITS, 16);
    // the parent writing to its own side of the fork
    op_write("WR", &main_task, (0x301 << PAGE_BITS) + 4080, 16, 18);
    op_read("RT", &child_task, (0x301 << PAGE_BITS) + 4080, 16);

    // ---- ptrace writes, which bypass write protection ----
    // a plain write to the read-only page first, to show the contrast
    op_write("WR", &main_task, 0x200 << PAGE_BITS, 8, 20);
    op_write("WP", &main_task, 0x200 << PAGE_BITS, 8, 21);
    op_read("RD", &main_task, 0x200 << PAGE_BITS, 8);
    // a ptrace write spanning out of the read-only mapping into the hole
    op_write("WP", &main_task, (0x201 << PAGE_BITS) + 4090, 16, 22);
    // ptrace against the child
    op_write("WP", &child_task, 0x301 << PAGE_BITS, 8, 23);

    // ---- strings ----
    // a NUL-terminated string sitting in the middle of a page
    op_write_string(0x600 << PAGE_BITS, "hello");
    op_read_string(0x600 << PAGE_BITS, 32);
    // a string that runs off the end of its page into a hole
    op_write_string((0x600 << PAGE_BITS) + 4090, "overflow");
    // reading it back faults partway through
    op_read_string((0x600 << PAGE_BITS) + 4090, 32);
    // max shorter than the string: the buffer comes back unterminated
    op_read_string(0x600 << PAGE_BITS, 3);
    // max exactly the string length: the NUL just fits
    op_read_string(0x600 << PAGE_BITS, 6);
    // a NULL address
    op_read_string(0, 32);
    op_write_string(0, "nope");
    // an empty string is still one byte
    op_write_string((0x600 << PAGE_BITS) + 200, "");
    op_read_string((0x600 << PAGE_BITS) + 200, 32);
    // reading from a read-only page
    op_read_string(0x200 << PAGE_BITS, 8);
    // writing to a read-only page
    op_write_string(0x200 << PAGE_BITS, "no");
    // reading a string out of the grow-down region, which grows it in
    op_read_string((0x4ff << PAGE_BITS) + 4090, 8);
    // page 0 mapped, so the NULL address checks are the only thing left to
    // refuse these two - without a mapping there they would fault anyway and the
    // checks would look redundant
    printf("O MN 0 1 %u\n", P_RWX);
    RET("%d", pt_map_nothing(&main_mem, 0, 1, P_RWX));
    op_read_string(0, 32);
    op_write_string(0, "nope");

    // ---- the `current`-following forms against the child ----
    printf("O CUR child\n");
    current = &child_task;
    op_read("RD", &child_task, 0x300 << PAGE_BITS, 16);
    op_write("WR", &child_task, (0x300 << PAGE_BITS) + 32, 16, 30);
    printf("O CUR main\n");
    current = &main_task;

    printf("# asbestos_invalidations %ld fd_closes %ld signals_sent %ld\n",
           asbestos_invalidations, fd_closes, signals_sent);
    return 0;
}

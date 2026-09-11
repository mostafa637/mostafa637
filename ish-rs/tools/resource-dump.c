// Reference-output generator for the resource differential test.
//
// This links the *unmodified* kernel/resource.c and drives every guest-visible
// resource-limit, rusage, affinity, and scheduler path through a deterministic
// task/group and guest-memory harness. The two host calls in resource.c are
// linker-wrapped: getrusage(RUSAGE_THREAD) and sysconf(_SC_NPROCESSORS_ONLN)
// receive values selected by the corpus rather than values from the machine
// running the generator.
//
// The guest-memory stub has the same success/fault contract needed by these
// calls: its two mapped pages are writable and all other addresses fault. The
// real kernel/user.c and kernel/memory.c have their own C-derived differential
// suites; keeping this harness small makes resource.c's branch and ordering
// rules visible without folding another subsystem into this oracle.
//
// Build (normally via tools/gen_resource_reference.sh):
//   cc -O2 -Wall -Wextra -I<ish-src> -o resource-dump resource-dump.c
//      <ish-src>/kernel/resource.c -pthread
//      -Wl,--wrap=getrusage -Wl,--wrap=sysconf

#define _GNU_SOURCE

#include <assert.h>
#include <stdarg.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/resource.h>
#include <unistd.h>

#include "kernel/calls.h"

#define GUEST_BASE ((addr_t) 0x001000)
#define GUEST_SIZE ((size_t) 0x002000)

#define A0 ((addr_t) 0x001000)
#define A1 ((addr_t) 0x001020)
#define A2 ((addr_t) 0x001040)
#define A3 ((addr_t) 0x001060)
#define A4 ((addr_t) 0x001080)
#define A5 ((addr_t) 0x0010c0)
#define A6 ((addr_t) 0x001140)
#define A7 ((addr_t) 0x0011c0)
#define BAD ((addr_t) 0x009000)

static byte_t guest[GUEST_SIZE];
static struct task task;
static struct tgroup group;
static struct task *known_task;

__thread struct task *current;
lock_t pids_lock = LOCK_INITIALIZER;

static struct rusage host_usage;
static unsigned host_cpus;

// ---- controlled replacements for the two host-dependent resource.c calls --

int __wrap_getrusage(int who, struct rusage *usage) {
    if (who != RUSAGE_THREAD)
        abort();
    *usage = host_usage;
    return 0;
}

long __wrap_sysconf(int name) {
    if (name != _SC_NPROCESSORS_ONLN)
        abort();
    return host_cpus;
}

// ---- the tiny guest-memory/task boundary required by kernel/resource.c -----

static bool guest_range(addr_t addr, size_t count) {
    if (count == 0)
        return true;
    if (addr < GUEST_BASE)
        return false;
    size_t offset = (size_t) (addr - GUEST_BASE);
    return offset <= GUEST_SIZE && count <= GUEST_SIZE - offset;
}

int user_read(addr_t addr, void *buf, size_t count) {
    if (!guest_range(addr, count))
        return 1;
    if (count != 0)
        memcpy(buf, &guest[addr - GUEST_BASE], count);
    return 0;
}

int user_write(addr_t addr, const void *buf, size_t count) {
    if (!guest_range(addr, count))
        return 1;
    if (count != 0)
        memcpy(&guest[addr - GUEST_BASE], buf, count);
    return 0;
}

struct task *pid_get_task(dword_t pid) {
    return pid == 1 ? known_task : NULL;
}

int current_pid(void) {
    return current == NULL ? 0 : current->pid;
}

_Noreturn void die(const char *msg, ...) {
    va_list ap;
    va_start(ap, msg);
    fputs("die: ", stderr);
    vfprintf(stderr, msg, ap);
    fputc('\n', stderr);
    va_end(ap);
    exit(99);
}

void ish_printk(const char *msg, ...) {
    (void) msg;
}

// ---- fixture encoding -----------------------------------------------------

static void emit_bytes(const void *data, size_t count) {
    const byte_t *bytes = data;
    for (size_t i = 0; i < count; i++)
        printf("%02x", bytes[i]);
}

static void emit_rusage_words(const struct rusage_ *usage) {
    dword_t words[sizeof *usage / sizeof(dword_t)];
    _Static_assert(sizeof *usage == 18 * sizeof(dword_t), "rusage_ ABI");
    memcpy(words, usage, sizeof words);
    for (size_t i = 0; i < sizeof words / sizeof words[0]; i++)
        printf(" %08x", words[i]);
}

static struct rusage_ rusage_from_words(const dword_t words[18]) {
    struct rusage_ usage;
    memcpy(&usage, words, sizeof usage);
    return usage;
}

static uint64_t guest_hash(void) {
    uint64_t hash = UINT64_C(0xcbf29ce484222325);
    for (size_t i = 0; i < GUEST_SIZE; i++) {
        hash ^= guest[i];
        hash *= UINT64_C(0x100000001b3);
    }
    return hash;
}

// A snapshot follows every C operation. It covers every limit, every child
// rusage word, the effective UID used by superuser(), and all two guest pages.
static void snapshot(void) {
    printf("S %08x", task.euid);
    for (int i = 0; i < RLIMIT_NLIMITS_; i++)
        printf(" %016llx %016llx", (unsigned long long) group.limits[i].cur,
               (unsigned long long) group.limits[i].max);
    emit_rusage_words(&group.children_rusage);
    printf(" %016llx\n", (unsigned long long) guest_hash());
}

static void prep_uid(uid_t_ euid) {
    task.euid = euid;
    printf("P UID %08x\n", euid);
}

static void prep_limit(dword_t resource, rlim_t_ cur, rlim_t_ max) {
    assert(resource < RLIMIT_NLIMITS_);
    group.limits[resource] = (struct rlimit_) { .cur = cur, .max = max };
    printf("P LIM %08x %016llx %016llx\n", resource,
           (unsigned long long) cur, (unsigned long long) max);
}

static void prep_children(const dword_t words[18]) {
    group.children_rusage = rusage_from_words(words);
    printf("P CHILD");
    emit_rusage_words(&group.children_rusage);
    putchar('\n');
}

static void prep_host(dword_t utime_sec, dword_t utime_usec, dword_t stime_sec,
                      dword_t stime_usec) {
    memset(&host_usage, 0, sizeof host_usage);
    host_usage.ru_utime.tv_sec = utime_sec;
    host_usage.ru_utime.tv_usec = utime_usec;
    host_usage.ru_stime.tv_sec = stime_sec;
    host_usage.ru_stime.tv_usec = stime_usec;
    printf("P HOST %08x %08x %08x %08x\n", utime_sec, utime_usec, stime_sec,
           stime_usec);
}

static void prep_cpus(unsigned cpus) {
    host_cpus = cpus;
    printf("P CPUS %08x\n", cpus);
}

static void guest_write_record(addr_t addr, const void *bytes, size_t count) {
    assert(user_write(addr, bytes, count) == 0);
    printf("W %08x ", addr);
    emit_bytes(bytes, count);
    putchar('\n');
}

static void guest_write_limit(addr_t addr, rlim_t_ cur, rlim_t_ max) {
    byte_t bytes[sizeof(struct rlimit_)];
    for (unsigned i = 0; i < 8; i++) {
        bytes[i] = (byte_t) (cur >> (8 * i));
        bytes[i + 8] = (byte_t) (max >> (8 * i));
    }
    guest_write_record(addr, bytes, sizeof bytes);
}

static void guest_write_i32(addr_t addr, int_t value) {
    dword_t word = (dword_t) value;
    byte_t bytes[sizeof word];
    for (unsigned i = 0; i < sizeof bytes; i++)
        bytes[i] = (byte_t) (word >> (8 * i));
    guest_write_record(addr, bytes, sizeof bytes);
}

#define OP(...) \
    do { \
        printf("O "); \
        printf(__VA_ARGS__); \
        putchar('\n'); \
    } while (0)

#define RET(value) printf("R %08x\n", (dword_t) (value))

static void finish(dword_t value) {
    RET(value);
    snapshot();
}

static void clear_unspecified_rusage_tail(addr_t addr) {
    // rusage_get_current initializes only utime/stime in the C source. The
    // trailing 56 bytes are indeterminate, so the SELF syscall's defined
    // sixteen-byte prefix is retained and the unspecified tail is cleared
    // before hashing fixture state. CHILDREN has no such exception.
    assert(guest_range(addr, sizeof(struct rusage_)));
    memset(&guest[addr - GUEST_BASE + 16], 0, sizeof(struct rusage_) - 16);
}

static void add_children(const dword_t words[18]) {
    struct rusage_ src = rusage_from_words(words);
    printf("O ADD");
    emit_rusage_words(&src);
    putchar('\n');
    rusage_add(&group.children_rusage, &src);
    finish(0);
}

static void current_rusage(void) {
    OP("RC");
    struct rusage_ usage = rusage_get_current();
    RET(0);
    // The defined part of rusage_get_current is exactly its two timeval_s.
    printf("V CURRENT %08x %08x %08x %08x\n", usage.utime.sec,
           usage.utime.usec, usage.stime.sec, usage.stime.usec);
    snapshot();
}

static const struct rlimit_ initial_limits[RLIMIT_NLIMITS_] = {
    [RLIMIT_CPU_] = { .cur = RLIM_INFINITY_, .max = RLIM_INFINITY_ },
    [RLIMIT_FSIZE_] = { .cur = RLIM_INFINITY_, .max = RLIM_INFINITY_ },
    [RLIMIT_DATA_] = { .cur = RLIM_INFINITY_, .max = RLIM_INFINITY_ },
    [RLIMIT_STACK_] = { .cur = 8 * 1024 * 1024, .max = RLIM_INFINITY_ },
    [RLIMIT_CORE_] = { .cur = 0, .max = RLIM_INFINITY_ },
    [RLIMIT_RSS_] = { .cur = RLIM_INFINITY_, .max = RLIM_INFINITY_ },
    [RLIMIT_NPROC_] = { .cur = 1024, .max = 1024 },
    [RLIMIT_NOFILE_] = { .cur = 1024, .max = 4096 },
    [RLIMIT_MEMLOCK_] = { .cur = 64 * 1024, .max = 64 * 1024 },
    [RLIMIT_AS_] = { .cur = RLIM_INFINITY_, .max = RLIM_INFINITY_ },
    [RLIMIT_LOCKS_] = { .cur = RLIM_INFINITY_, .max = RLIM_INFINITY_ },
    [RLIMIT_SIGPENDING_] = { .cur = 1024, .max = 1024 },
    [RLIMIT_MSGQUEUE_] = { .cur = 819200, .max = 819200 },
    [RLIMIT_NICE_] = { .cur = 0, .max = 0 },
    [RLIMIT_RTPRIO_] = { .cur = 0, .max = 0 },
    [RLIMIT_RTTIME_] = { .cur = RLIM_INFINITY_, .max = RLIM_INFINITY_ },
};

int main(void) {
    assert(sizeof(struct rlimit_) == 16);
    assert(sizeof(struct rlimit32_) == 8);
    assert(sizeof(struct rusage_) == 72);

    for (size_t i = 0; i < GUEST_SIZE; i++)
        guest[i] = (byte_t) ((i * 37 + 23) & 0xff);

    lock_init(&group.lock);
    task.group = &group;
    task.pid = 1;
    group.leader = &task;
    known_task = &task;
    current = &task;
    memcpy(group.limits, initial_limits, sizeof group.limits);

    printf("# resource reference output, generated from unmodified iSH kernel/resource.c\n");
    printf("# all numeric operation arguments and returns are raw hexadecimal words\n");
    printf("# P and W records configure the deterministic harness; O records call C\n");
    printf("# S: euid, 16 cur/max pairs, children rusage's 18 words, guest FNV-1a\n");
    printf("# SELF rusage state hashes only defined timeval bytes; its C tail is cleared\n");
    prep_uid(0);
    prep_host(0x11, 0x222, 0x33, 0x444);
    prep_cpus(1);
    snapshot();

    // Valid direct helper and both 32-bit getter ABIs, including truncation and
    // the historical INT_MAX compatibility clamp.
    OP("RL %08x", RLIMIT_STACK_);
    printf("R %016llx\n", (unsigned long long) rlimit(RLIMIT_STACK_));
    snapshot();

    OP("G32 %08x %08x", RLIMIT_STACK_, A0);
    finish(sys_getrlimit32(RLIMIT_STACK_, A0));
    OP("OG32 %08x %08x", RLIMIT_STACK_, A1);
    finish(sys_old_getrlimit32(RLIMIT_STACK_, A1));
    // do_getrlimit32 narrows before sys_old_getrlimit32 applies its signed
    // compatibility clamp: exact 4 GiB therefore becomes zero, not INT_MAX.
    prep_limit(RLIMIT_CORE_, UINT64_C(0x100000000), UINT64_C(0x100000000));
    OP("OG32 %08x %08x", RLIMIT_CORE_, A1);
    finish(sys_old_getrlimit32(RLIMIT_CORE_, A1));
    OP("G32 %08x %08x", RLIMIT_NLIMITS_, A0);
    finish(sys_getrlimit32(RLIMIT_NLIMITS_, A0));
    OP("G32 %08x %08x", RLIMIT_NOFILE_, BAD);
    finish(sys_getrlimit32(RLIMIT_NOFILE_, BAD));

    // sys_setrlimit32 reads the full 64-bit rlimit_ despite its historical
    // name. Root may raise a limit; a non-root may lower max while raising cur.
    guest_write_limit(A1, UINT64_C(0x100000001), UINT64_C(0x100000002));
    OP("S32 %08x %08x", RLIMIT_NOFILE_, A1);
    finish(sys_setrlimit32(RLIMIT_NOFILE_, A1));
    OP("G32 %08x %08x", RLIMIT_NOFILE_, A0);
    finish(sys_getrlimit32(RLIMIT_NOFILE_, A0));

    prep_uid(1000);
    guest_write_limit(A1, UINT64_C(0x100000003), UINT64_C(0x100000003));
    OP("S32 %08x %08x", RLIMIT_NOFILE_, A1);
    finish(sys_setrlimit32(RLIMIT_NOFILE_, A1));
    guest_write_limit(A1, UINT64_C(0xfffffffffffffff0), UINT64_C(0x100000001));
    OP("S32 %08x %08x", RLIMIT_NOFILE_, A1);
    finish(sys_setrlimit32(RLIMIT_NOFILE_, A1));
    OP("S32 %08x %08x", RLIMIT_NLIMITS_, BAD);
    finish(sys_setrlimit32(RLIMIT_NLIMITS_, BAD));

    // Root's check_setrlimit short-circuit still leaves rlimit_set to reject an
    // invalid resource after a successful user read.
    prep_uid(0);
    guest_write_limit(A1, 7, 8);
    OP("S32 %08x %08x", RLIMIT_NLIMITS_, A1);
    finish(sys_setrlimit32(RLIMIT_NLIMITS_, A1));

    // prlimit64's PID gate, no-pointer special case, old-before-new ordering,
    // write/read faults, and non-root permission branch.
    OP("PR %08x %08x %08x %08x", 1u, RLIMIT_STACK_, A1, A2);
    finish(sys_prlimit64(1, RLIMIT_STACK_, A1, A2));
    OP("PR %08x %08x %08x %08x", 0u, RLIMIT_NLIMITS_, 0u, 0u);
    finish(sys_prlimit64(0, RLIMIT_NLIMITS_, 0, 0));
    OP("PR %08x %08x %08x %08x", 0u, RLIMIT_STACK_, 0u, A2);
    finish(sys_prlimit64(0, RLIMIT_STACK_, 0, A2));
    OP("PR %08x %08x %08x %08x", 0u, RLIMIT_STACK_, BAD, BAD);
    finish(sys_prlimit64(0, RLIMIT_STACK_, BAD, BAD));
    guest_write_limit(A3, 7, 8);
    OP("PR %08x %08x %08x %08x", 0u, RLIMIT_STACK_, A3, A3);
    finish(sys_prlimit64(0, RLIMIT_STACK_, A3, A3));
    OP("PR %08x %08x %08x %08x", 0u, RLIMIT_NLIMITS_, 0u, A4);
    finish(sys_prlimit64(0, RLIMIT_NLIMITS_, 0, A4));
    OP("PR %08x %08x %08x %08x", 0u, RLIMIT_STACK_, BAD, 0u);
    finish(sys_prlimit64(0, RLIMIT_STACK_, BAD, 0));
    prep_uid(1000);
    guest_write_limit(A4, UINT64_C(0xfffffffffffffff1), UINT64_C(0x100000002));
    OP("PR %08x %08x %08x %08x", 0u, RLIMIT_NOFILE_, A4, 0u);
    finish(sys_prlimit64(0, RLIMIT_NOFILE_, A4, 0));
    prep_uid(0);

    // rusage_get_current is host-controlled and its defined timeval prefix is
    // checked directly. CHILDREN is fully initialized and checked byte-for-byte.
    static const dword_t child_initial[18] = {
        1, 999999, 0xfffffffe, 999999,
        0x11111111, 0x22222222, 0x33333333, 0x44444444,
        0x55555555, 0x66666666, 0x77777777, 0x88888888,
        0x99999999, 0xaaaaaaaa, 0xbbbbbbbb, 0xcccccccc,
        0xdddddddd, 0xeeeeeeee,
    };
    static const dword_t child_add[18] = {
        2, 1, 2, 2,
        0x01234567, 0x89abcdef, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
    };
    prep_children(child_initial);
    current_rusage();
    OP("RU %08x %08x", RUSAGE_SELF_, A5);
    dword_t usage_result = sys_getrusage(RUSAGE_SELF_, A5);
    RET(usage_result);
    clear_unspecified_rusage_tail(A5);
    snapshot();
    OP("RU %08x %08x", (dword_t) RUSAGE_CHILDREN_, A6);
    finish(sys_getrusage(RUSAGE_CHILDREN_, A6));
    OP("RU %08x %08x", 1u, A6);
    finish(sys_getrusage(1, A6));
    OP("RU %08x %08x", (dword_t) RUSAGE_CHILDREN_, BAD);
    finish(sys_getrusage(RUSAGE_CHILDREN_, BAD));
    add_children(child_add);
    OP("RU %08x %08x", (dword_t) RUSAGE_CHILDREN_, A6);
    finish(sys_getrusage(RUSAGE_CHILDREN_, A6));

    // Affinity gets a deterministic online-CPU count through __wrap_sysconf.
    prep_cpus(0);
    OP("AFF %08x %08x %08x", 0u, 0u, A7);
    finish(sys_sched_getaffinity(0, 0, A7));
    OP("AFF %08x %08x %08x", 0u, 1u, A7);
    finish(sys_sched_getaffinity(0, 1, A7));
    prep_cpus(1);
    OP("AFF %08x %08x %08x", 0u, 1u, A7);
    finish(sys_sched_getaffinity(0, 1, A7));
    prep_cpus(8);
    OP("AFF %08x %08x %08x", 0u, 1u, A7);
    finish(sys_sched_getaffinity(0, 1, A7));
    OP("AFF %08x %08x %08x", 0u, 2u, A7);
    finish(sys_sched_getaffinity(0, 2, A7));
    prep_cpus(9);
    OP("AFF %08x %08x %08x", 0u, 2u, A7);
    finish(sys_sched_getaffinity(0, 2, A7));
    OP("AFF %08x %08x %08x", 1u, 2u, A7);
    finish(sys_sched_getaffinity(1, 2, A7));
    OP("AFF %08x %08x %08x", 2u, 0u, A7);
    finish(sys_sched_getaffinity(2, 0, A7));
    OP("AFF %08x %08x %08x", 0u, 2u, BAD);
    finish(sys_sched_getaffinity(0, 2, BAD));
    prep_cpus(17);
    OP("AFF %08x %08x %08x", 0u, 3u, A7);
    finish(sys_sched_getaffinity(0, 3, A7));

    // The intentionally narrow scheduling/ioprio compatibility stubs.
    OP("SAFF %08x %08x %08x", 2u, 0u, BAD);
    finish(sys_sched_setaffinity(2, 0, BAD));
    OP("GPRI %08x %08x", 0xffffffffu, 0x80000000u);
    finish(sys_getpriority(-1, INT32_MIN));
    OP("SPRI %08x %08x %08x", 1u, 2u, 0xffffff80u);
    finish(sys_setpriority(1, 2, -128));
    OP("GPAR %08x %08x", 1u, A7);
    finish(sys_sched_getparam(1, A7));
    OP("GPAR %08x %08x", 1u, BAD);
    finish(sys_sched_getparam(1, BAD));
    OP("GSCH %08x", 0xffffffffu);
    finish(sys_sched_getscheduler(-1));
    guest_write_i32(A7, -1);
    OP("SSCH %08x %08x %08x", 1u, 0u, A7);
    finish(sys_sched_setscheduler(1, 0, A7));
    guest_write_i32(A7, 0);
    OP("SSCH %08x %08x %08x", 1u, 0u, A7);
    finish(sys_sched_setscheduler(1, 0, A7));
    OP("SSCH %08x %08x %08x", 1u, 1u, BAD);
    finish(sys_sched_setscheduler(1, 1, BAD));
    OP("SSCH %08x %08x %08x", 1u, 0u, BAD);
    finish(sys_sched_setscheduler(1, 0, BAD));
    OP("PMAX %08x", 0u);
    finish(sys_sched_get_priority_max(0));
    OP("PMAX %08x", 0xffffffffu);
    finish(sys_sched_get_priority_max(-1));
    OP("IGET %08x %08x %08x", 1u, 2u, 3u);
    finish(sys_ioprio_get(1, 2, 3));
    OP("ISET %08x %08x %08x", 4u, 5u, 6u);
    finish(sys_ioprio_set(4, 5, 6));

    return 0;
}

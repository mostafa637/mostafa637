// Reference-output generator for the uname/sysinfo differential test.
//
// It links the unmodified kernel/uname.c. uname(2) and Linux sysinfo(2) are
// linker-wrapped, while platform get_uptime() is supplied by this harness. The
// controlled values make the fixture independent of the host name, boot time,
// load, and RAM. gen_uname_reference.sh also fixes SOURCE_DATE_EPOCH so C's
// __DATE__/__TIME__ version suffix is reproducible.

#define _GNU_SOURCE

#include <assert.h>
#include <stdarg.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/sysinfo.h>
#include <sys/utsname.h>

#include "kernel/calls.h"
#include "platform/platform.h"

#define GUEST_BASE ((addr_t) 0x001000)
#define GUEST_SIZE ((size_t) 0x002000)
#define A0 ((addr_t) 0x001000)
#define A1 ((addr_t) 0x001200)
#define BAD ((addr_t) 0x009000)

extern const char *uname_version;
extern const char *uname_hostname_override;

static byte_t guest[GUEST_SIZE];
static char host_hostname[sizeof(((struct utsname *) 0)->nodename)];
static char override_hostname[128];
static char version[128];
static struct uptime_info host_uptime;
static struct sysinfo host_sysinfo;
static unsigned uname_calls;
static unsigned uptime_calls;
static unsigned sysinfo_calls;

int __wrap_uname(struct utsname *uts) {
    memset(uts, 0, sizeof *uts);
    strcpy(uts->nodename, host_hostname);
    uname_calls++;
    return 0;
}

int __wrap_sysinfo(struct sysinfo *info) {
    *info = host_sysinfo;
    sysinfo_calls++;
    return 0;
}

struct uptime_info get_uptime(void) {
    uptime_calls++;
    return host_uptime;
}

static bool guest_range(addr_t addr, size_t count) {
    if (count == 0)
        return true;
    if (addr < GUEST_BASE)
        return false;
    size_t offset = (size_t) (addr - GUEST_BASE);
    return offset <= GUEST_SIZE && count <= GUEST_SIZE - offset;
}

int user_write(addr_t addr, const void *buf, size_t count) {
    if (!guest_range(addr, count))
        return 1;
    if (count != 0)
        memcpy(&guest[addr - GUEST_BASE], buf, count);
    return 0;
}

int current_pid(void) { return 1; }
_Noreturn void die(const char *msg, ...) {
    va_list args;
    va_start(args, msg);
    vfprintf(stderr, msg, args);
    va_end(args);
    abort();
}
void ish_printk(const char *msg, ...) { (void) msg; }

static void emit_bytes(const void *data, size_t count) {
    const byte_t *bytes = data;
    for (size_t i = 0; i < count; i++)
        printf("%02x", bytes[i]);
}

static uint64_t guest_hash(void) {
    uint64_t hash = UINT64_C(0xcbf29ce484222325);
    for (size_t i = 0; i < GUEST_SIZE; i++) {
        hash ^= guest[i];
        hash *= UINT64_C(0x100000001b3);
    }
    return hash;
}

static void snapshot(void) {
    printf("S %08x %08x %08x %016llx\n", uname_calls, uptime_calls,
           sysinfo_calls, (unsigned long long) guest_hash());
}

static void copy_string(char *dst, size_t cap, const char *src) {
    assert(strlen(src) + 1 <= cap);
    strcpy(dst, src);
}

static void prep_hostname(const char *name) {
    copy_string(host_hostname, sizeof host_hostname, name);
    printf("P HOSTNAME ");
    emit_bytes(name, strlen(name));
    putchar('\n');
}

static void prep_override(const char *name) {
    if (name == NULL) {
        uname_hostname_override = NULL;
        puts("P OVERRIDE -");
        return;
    }
    copy_string(override_hostname, sizeof override_hostname, name);
    uname_hostname_override = override_hostname;
    printf("P OVERRIDE ");
    emit_bytes(name, strlen(name));
    putchar('\n');
}

static void prep_version(const char *new_version) {
    copy_string(version, sizeof version, new_version);
    uname_version = version;
    printf("P VERSION ");
    emit_bytes(new_version, strlen(new_version));
    putchar('\n');
}

static void prep_uptime(uint64_t uptime, uint64_t load_1m, uint64_t load_5m,
                        uint64_t load_15m) {
    host_uptime = (struct uptime_info) {
        .uptime_ticks = uptime,
        .load_1m = load_1m,
        .load_5m = load_5m,
        .load_15m = load_15m,
    };
    printf("P UPTIME %016llx %016llx %016llx %016llx\n",
           (unsigned long long) uptime, (unsigned long long) load_1m,
           (unsigned long long) load_5m, (unsigned long long) load_15m);
}

static void prep_sysinfo(uint64_t totalram, uint64_t freeram, uint64_t sharedram,
                         uint64_t totalswap, uint64_t freeswap, unsigned procs,
                         uint64_t totalhigh, uint64_t freehigh, uint32_t mem_unit) {
    memset(&host_sysinfo, 0, sizeof host_sysinfo);
    host_sysinfo.totalram = (unsigned long) totalram;
    host_sysinfo.freeram = (unsigned long) freeram;
    host_sysinfo.sharedram = (unsigned long) sharedram;
    // uname.c intentionally does not copy this native field to guest bufferram.
    host_sysinfo.bufferram = 0xdecafbad;
    host_sysinfo.totalswap = (unsigned long) totalswap;
    host_sysinfo.freeswap = (unsigned long) freeswap;
    host_sysinfo.procs = (unsigned short) procs;
    host_sysinfo.totalhigh = (unsigned long) totalhigh;
    host_sysinfo.freehigh = (unsigned long) freehigh;
    host_sysinfo.mem_unit = mem_unit;
    printf("P SYS %016llx %016llx %016llx %016llx %016llx %08x %016llx %016llx %08x\n",
           (unsigned long long) totalram, (unsigned long long) freeram,
           (unsigned long long) sharedram, (unsigned long long) totalswap,
           (unsigned long long) freeswap, procs, (unsigned long long) totalhigh,
           (unsigned long long) freehigh, mem_unit);
}

#define OP(...) \
    do { \
        printf("O "); \
        printf(__VA_ARGS__); \
        putchar('\n'); \
    } while (0)

static void do_uname_record(void) {
    struct uname uts;
    OP("DU");
    do_uname(&uts);
    puts("R 00000000");
    printf("V ");
    emit_bytes(&uts, sizeof uts);
    putchar('\n');
    snapshot();
}

static void sys_uname_record(addr_t addr) {
    OP("UN %08x", addr);
    printf("R %08x\n", sys_uname(addr));
    snapshot();
}

static void sysinfo_record(addr_t addr) {
    OP("SI %08x", addr);
    printf("R %08x\n", sys_sysinfo(addr));
    snapshot();
}

int main(void) {
    assert(sizeof(struct uname) == 390);
    assert(sizeof(struct sys_info) == 60);
    for (size_t i = 0; i < GUEST_SIZE; i++)
        guest[i] = (byte_t) ((i * 13 + 3) & 0xff);

    puts("# uname reference output, generated from unmodified iSH kernel/uname.c");
    puts("# uname/sysinfo host calls are deterministic linker/harness boundaries");
    puts("# C __DATE__ and __TIME__ are fixed by SOURCE_DATE_EPOCH=0");
    puts("# S: uname, uptime, sysinfo host-call counts and guest FNV-1a");
    prep_hostname("native-host.example");
    prep_override(NULL);
    prep_version("SUPER AWESOME");
    prep_uptime(123, 0x10000, 0x20000, 0x30000);
    prep_sysinfo(0x100000, 0x200000, 0x300000, 0x400000, 0x500000, 17,
                 0x600000, 0x700000, 4096);
    snapshot();

    do_uname_record();
    sys_uname_record(A0);
    // do_uname still happens before a guest-output fault.
    sys_uname_record(BAD);
    OP("SH %08x %08x", A0, 0xffffffffu);
    printf("R %08x\n", sys_sethostname(A0, 0xffffffffu));
    snapshot();
    sysinfo_record(A1);
    // get_uptime/sysinfo are both reached before the output fault.
    sysinfo_record(BAD);

    // The override wins over host uname's nodename, but host uname is still
    // called. A long version exercises C snprintf's 64-byte payload limit.
    prep_hostname("changed-native.example");
    prep_override("override-host.example");
    prep_version("THIS-IS-A-LONG-CUSTOM-VERSION-STRING-THAT-EXCEEDS-SIXTY-FOUR-BYTES-ON-PURPOSE");
    do_uname_record();
    sys_uname_record(A0);

    // Drop the override and use high bits to pin all guest dword truncations.
    prep_override(NULL);
    prep_version("CUSTOM BUILD");
    prep_uptime(UINT64_C(0x100000001), UINT64_C(0xffffffff00000002),
                UINT64_C(0x200000003), UINT64_C(0x300000004));
    prep_sysinfo(UINT64_C(0x100000005), UINT64_C(0x200000006),
                 UINT64_C(0x300000007), UINT64_C(0x400000008),
                 UINT64_C(0x500000009), 0x2345, UINT64_C(0x60000000a),
                 UINT64_C(0x70000000b), 0x11223344);
    do_uname_record();
    sysinfo_record(A1);

    return 0;
}

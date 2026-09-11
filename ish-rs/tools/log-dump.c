// Reference-output generator for the log differential test.
//
// This links the unmodified kernel/log.c and util/fifo.c. log.c's external
// output handler is selected as LOG_HANDLER_DPRINTF and writev is linker-wrapped
// below, so emitted log lines become a deterministic observable stream instead
// of host output. user_write is the only other boundary: a small guest window
// records exactly what sys_syslog writes. The G operation produces many valid
// (<16 KiB) printk lines without storing megabytes in the fixture; it exercises
// log.c's one-MiB circular buffer and its FIFO_LAST behavior.

#define _GNU_SOURCE

#include <assert.h>
#include <stdarg.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/uio.h>

#include "kernel/calls.h"

#define GUEST_BASE ((addr_t) 0x001000)
#define GUEST_SIZE ((size_t) 0x008000)
#define A0 ((addr_t) 0x001000)
#define A1 ((addr_t) 0x005000)
#define BAD ((addr_t) 0x090000)

#define SYSLOG_ACTION_CLOSE_ 0
#define SYSLOG_ACTION_OPEN_ 1
#define SYSLOG_ACTION_READ_ 2
#define SYSLOG_ACTION_READ_ALL_ 3
#define SYSLOG_ACTION_READ_CLEAR_ 4
#define SYSLOG_ACTION_CLEAR_ 5
#define SYSLOG_ACTION_CONSOLE_OFF_ 6
#define SYSLOG_ACTION_CONSOLE_ON_ 7
#define SYSLOG_ACTION_CONSOLE_LEVEL_ 8
#define SYSLOG_ACTION_SIZE_UNREAD_ 9
#define SYSLOG_ACTION_SIZE_BUFFER_ 10

static byte_t guest[GUEST_SIZE];
static uint32_t sink_calls;
static uint64_t sink_hash = UINT64_C(0xcbf29ce484222325);

static void hash_bytes(const void *data, size_t count) {
    const byte_t *bytes = data;
    for (size_t i = 0; i < count; i++) {
        sink_hash ^= bytes[i];
        sink_hash *= UINT64_C(0x100000001b3);
    }
}

// log.c's LOG_HANDLER_DPRINTF path always calls writev(666, {line, "\n"}).
ssize_t __wrap_writev(int fd, const struct iovec *iov, int iov_count) {
    assert(fd == 666);
    assert(iov_count == 2);
    sink_calls++;
    size_t total = 0;
    for (int i = 0; i < iov_count; i++) {
        hash_bytes(iov[i].iov_base, iov[i].iov_len);
        total += iov[i].iov_len;
    }
    return (ssize_t) total;
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

// log.c defines current_pid(), but that definition references this TLS symbol.
__thread struct task *current;

static uint64_t guest_hash(void) {
    uint64_t hash = UINT64_C(0xcbf29ce484222325);
    for (size_t i = 0; i < GUEST_SIZE; i++) {
        hash ^= guest[i];
        hash *= UINT64_C(0x100000001b3);
    }
    return hash;
}

static void emit_bytes(const void *data, size_t count) {
    const byte_t *bytes = data;
    for (size_t i = 0; i < count; i++)
        printf("%02x", bytes[i]);
}

static void snapshot(void) {
    printf("S %08x %016llx %08x %016llx\n", sink_calls,
           (unsigned long long) sink_hash,
           (dword_t) sys_syslog(SYSLOG_ACTION_SIZE_UNREAD_, 0, 0),
           (unsigned long long) guest_hash());
}

#define OP(...) \
    do { \
        printf("O "); \
        printf(__VA_ARGS__); \
        putchar('\n'); \
    } while (0)

static void print_text(const char *text) {
    printf("O T ");
    emit_bytes(text, strlen(text));
    putchar('\n');
    ish_printk("%s", text);
    puts("R 00000000");
    snapshot();
}

static byte_t generated_byte(unsigned line, unsigned index, unsigned seed) {
    return (byte_t) ('A' + ((line * 7 + index * 11 + seed) % 26));
}

static void print_generated_lines(unsigned line_len, unsigned count, unsigned seed) {
    assert(line_len >= 2 && line_len < 16384);
    OP("G %08x %08x %08x", line_len, count, seed);
    char *line = malloc((size_t) line_len + 1);
    assert(line != NULL);
    for (unsigned row = 0; row < count; row++) {
        for (unsigned index = 0; index + 1 < line_len; index++)
            line[index] = (char) generated_byte(row, index, seed);
        line[line_len - 1] = '\n';
        line[line_len] = '\0';
        ish_printk("%s", line);
    }
    free(line);
    puts("R 00000000");
    snapshot();
}

static bool is_read_action(int_t type) {
    return type == SYSLOG_ACTION_READ_ || type == SYSLOG_ACTION_READ_ALL_
        || type == SYSLOG_ACTION_READ_CLEAR_;
}

static void syslog_record(int_t type, addr_t addr, int_t len) {
    OP("SL %08x %08x %08x", (dword_t) type, addr, (dword_t) len);
    int_t result = sys_syslog(type, addr, len);
    printf("R %08x\n", (dword_t) result);
    if (is_read_action(type) && result >= 0) {
        // do_syslog returns 0 after a successful READ_CLEAR even though its
        // preceding syslog_read copied `len` bytes. Every READ_CLEAR here has
        // a nonnegative request known not to exceed its max-since-clear cap.
        size_t output_count = type == SYSLOG_ACTION_READ_CLEAR_ ? (size_t) len
                                                                : (size_t) result;
        assert(guest_range(addr, output_count));
        printf("V ");
        emit_bytes(&guest[addr - GUEST_BASE], output_count);
        putchar('\n');
    }
    snapshot();
}

int main(void) {
    for (size_t i = 0; i < GUEST_SIZE; i++)
        guest[i] = (byte_t) ((i * 17 + 9) & 0xff);

    puts("# log reference output, generated from unmodified iSH kernel/log.c");
    puts("# util/fifo.c is linked unchanged; writev(666) is the only wrapped output boundary");
    puts("# S: emitted-line count/FNV, unread FIFO size, and full guest-window FNV");
    snapshot();

    // A partial printk does not reach either handler until a newline arrives.
    print_text("alpha");
    syslog_record(SYSLOG_ACTION_SIZE_UNREAD_, 0, 0);
    print_text(" beta\nnext\ntrail");
    syslog_record(SYSLOG_ACTION_SIZE_UNREAD_, 0, 0);
    syslog_record(SYSLOG_ACTION_READ_ALL_, A0, 12);
    syslog_record(SYSLOG_ACTION_READ_, A1, 6);
    // A failed normal read has already consumed its FIFO prefix.
    syslog_record(SYSLOG_ACTION_READ_, BAD, 4);
    syslog_record(SYSLOG_ACTION_SIZE_UNREAD_, 0, 0);
    // PEEK actions do not consume; READ_CLEAR clears only after a successful
    // guest write.
    syslog_record(SYSLOG_ACTION_READ_ALL_, BAD, 6);
    syslog_record(SYSLOG_ACTION_READ_CLEAR_, BAD, 6);
    syslog_record(SYSLOG_ACTION_READ_ALL_, A0, 6);
    syslog_record(SYSLOG_ACTION_READ_CLEAR_, A1, 6);
    syslog_record(SYSLOG_ACTION_CLEAR_, 0, 0);
    print_text("tail-end\n");
    syslog_record(SYSLOG_ACTION_READ_, A0, 6);
    syslog_record(SYSLOG_ACTION_CLEAR_, 0, 0);
    // CLEAR resets the "since clear" count but retains the FIFO bytes.
    print_text("z\n");
    syslog_record(SYSLOG_ACTION_READ_ALL_, A1, 9);
    syslog_record(SYSLOG_ACTION_READ_, A0, 11);
    syslog_record(SYSLOG_ACTION_CLEAR_, 0, 0);

    // Cover non-reading control and error branches without touching guest RAM.
    syslog_record(SYSLOG_ACTION_CLOSE_, BAD, -1);
    syslog_record(SYSLOG_ACTION_OPEN_, BAD, -1);
    syslog_record(SYSLOG_ACTION_CONSOLE_OFF_, BAD, -1);
    syslog_record(SYSLOG_ACTION_CONSOLE_ON_, BAD, -1);
    syslog_record(SYSLOG_ACTION_CONSOLE_LEVEL_, BAD, -1);
    syslog_record(SYSLOG_ACTION_SIZE_BUFFER_, 0, 0);
    syslog_record(SYSLOG_ACTION_READ_, A0, -1);
    syslog_record(0x7f, 0, 0);

    // 255 complete 8 KiB lines leave the 1 MiB FIFO full, with its start 8 KiB
    // before the physical end (plus the prior deliberate 27-byte offset). The
    // final tail reads exercise the original fifo_read FIFO_LAST split exactly.
    print_generated_lines(8192, 255, 17);
    syslog_record(SYSLOG_ACTION_READ_ALL_, A0, 0x3000);
    syslog_record(SYSLOG_ACTION_READ_CLEAR_, A1, 0x3000);

    return 0;
}

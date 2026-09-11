// Reference-output generator for the fs/mount differential test.
//
// This *includes* fs/mount.c, so every function exercised below — mount_find,
// mount_retain, mount_release, do_mount, mount_remove, do_umount, fs_register
// and mount_param_flag — is the real one, along with its constants and the
// struct mount it fills in. fs/path.c is linked as well, because mount_find
// asserts path_is_normalized.
//
// The driver is a script: it builds a table, mounts a fake filesystem at a
// series of points, looks paths up in it, takes and gives back references, and
// removes mounts — one record per observation.
// tests/mount_differential.rs reads these records and performs the same
// operations through the port, comparing each answer, so this file is both the
// corpus and the script.
//
// Records, all of which tests/mount_differential.rs replays:
//   K <name>=<value>                  the constants of fs/mount.c and kernel/calls.h
//   S <slot> name=<name>              the four filesystems fs/mount.c seeds
//   G name=<name> slot=<slot>         fs_register filled that slot
//   M fs=<name> source=<hex> point=<hex> info=<hex> flags=<n> ret=<n>
//                                     do_mount, called with mounts_lock held
//   Q path=<hex> point=<hex> source=<hex> refcount=<n>
//                                     mount_find found that mount and retained it
//   T point=<hex> refcount=<n>        mount_retain: the count after
//   R point=<hex> refcount=<n>        mount_release: the count after
//   X point=<hex> refcount=<n> ret=<n>  mount_remove: the count before, the result
//   U point=<hex> ret=<n>             do_umount
//   P info=<hex> flag=<hex> ret=<0|1> mount_param_flag
//   L <index> point=<hex> source=<hex> info=<hex> flags=<n> refcount=<n> root_fd=<n>
//                                     one entry of the mount list, in order
//   E count=<n>                       the list has that many entries
//   F count=<n>                       the fake filesystem's callback log follows
//   F <index> mount|umount ...        one callback, in the order it was called
//
// Bytes are hex; the empty string is `-`, since a hex byte string can be empty
// but a field cannot. The root filesystem is mounted at the *empty* point
// exactly as kernel/init.c mounts it — `strncmp(path, "", 0)` matches every
// path, so that one mount is what makes every lookup find something.
//
// What is deliberately not recorded, because the C has no value to record: a
// registration past MAX_FILESYSTEMS, mount_find with a non-normalized path,
// mount_find on an empty table (all three are assert failures), and a
// mount_param_flag input whose first comma is reached without a match (the loop
// never skips the comma and spins forever). The first and the last are checked
// by unit tests in src/mount.rs instead.
//
// Build (normally via tools/gen_mount_reference.sh):
//   cc -O2 -Wall -Wextra -Werror -I<ish-src> -I<ish-rs>/tools/stub-include
//      -o mount-dump tools/mount-dump.c

#define _GNU_SOURCE

#include <stdarg.h>
#include <stdbool.h>
#include <stddef.h>
#include <stdio.h>
#include <string.h>

#include "kernel/calls.h"
#include "kernel/fs.h"
#include "fs/path.h"
#include "fs/real.h"

// ---- stubs for the parts of iSH this oracle does not link ------------------

// `current` is a thread-local in kernel/task.c. fs/path.c's path_normalize reads
// `current->fs`; only path_is_normalized is called here, so this is only there
// to satisfy the linker.
__thread struct task *current;

int user_read_string(addr_t addr, char *buf, size_t max) {
    (void) addr;
    (void) buf;
    (void) max;
    return 1;
}

int generic_statat(struct fd *at, const char *path, struct statbuf *stat, bool follow_links) {
    (void) at;
    (void) path;
    (void) stat;
    (void) follow_links;
    return _EINVAL;
}

int generic_getpath(struct fd *fd, char *buf) {
    (void) fd;
    (void) buf;
    return _EINVAL;
}

int access_check(struct statbuf *stat, int check) {
    (void) stat;
    (void) check;
    return _EINVAL;
}

struct mount *find_mount_and_trim_path(char *path) {
    (void) path;
    return NULL;
}

// The real one formats into the kernel log; here it goes to stderr, so that
// anything unexpected is visible without corrupting the records on stdout.
void ish_printk(const char *msg, ...) {
    va_list args;
    va_start(args, msg);
    vfprintf(stderr, msg, args);
    va_end(args);
}

#include "fs/path.c"
#include "fs/mount.c"

// The four filesystems kernel/fs.h declares and fs/mount.c seeds its table with.
// The real definitions live in fs/real.c, fs/proc.c, fs/pty.c and fs/tmp.c,
// none of which is ported (or linkable) yet; the driver records their names and
// tools/gen_mount_reference.sh checks those against the real sources.
const struct fs_ops realfs = {.name = "real", .magic = 0x7265616c};
const struct fs_ops procfs = {.name = "proc", .magic = 0x9fa0};
const struct fs_ops devptsfs = {.name = "devpts", .magic = 0x1cd1};
const struct fs_ops tmpfs = {.name = "tmpfs", .magic = 0x01021994};

// ---- hex -------------------------------------------------------------------

// Hex, or `-` for the empty string.
static void hex_into(char *dst, size_t size, const char *data) {
    if (*data == '\0') {
        snprintf(dst, size, "-");
        return;
    }
    size_t at = 0;
    for (const unsigned char *p = (const unsigned char *) data; *p != '\0' && at + 3 <= size; p++) {
        snprintf(dst + at, size - at, "%02x", *p);
        at += 2;
    }
    dst[at] = '\0';
}

static void print_hex(const char *data) {
    char hex[MAX_PATH * 2];
    hex_into(hex, sizeof(hex), data);
    printf("%s", hex);
}

// ---- the fake filesystem ---------------------------------------------------

// The descriptor number the fake filesystem's `mount` op opens its root on, so
// the list records can show that the callback's write reached the mount.
#define FAKE_ROOT_FD 7

static char callbacks[64][512];
static size_t callback_count;

static void callback(const char *format, ...) {
    va_list args;
    va_start(args, format);
    vsnprintf(callbacks[callback_count], sizeof(callbacks[callback_count]), format, args);
    va_end(args);
    callback_count++;
}

// `mount`: everything succeeds except the point "/fail", which is the one the
// corpus uses to show that a failed mount never reaches the list.
static int fake_mount(struct mount *mount) {
    int ret = strcmp(mount->point, "/fail") == 0 ? _EINVAL : 0;
    char point[MAX_PATH], source[MAX_PATH], info[MAX_PATH];
    hex_into(point, sizeof(point), mount->point);
    hex_into(source, sizeof(source), mount->source);
    hex_into(info, sizeof(info), mount->info);
    callback("mount point=%s source=%s info=%s ret=%d", point, source, info, ret);
    if (ret == 0)
        mount->root_fd = FAKE_ROOT_FD;
    return ret;
}

static int fake_umount(struct mount *mount) {
    char point[MAX_PATH];
    hex_into(point, sizeof(point), mount->point);
    callback("umount point=%s", point);
    return 0;
}

static const struct fs_ops fake = {
    .name = "fake",
    .magic = 0x66616b65,
    .mount = fake_mount,
    .umount = fake_umount,
};

// ---- records ---------------------------------------------------------------

// The mount list, longest point first: one record per entry, then the count.
static void print_mounts(void) {
    struct mount *mount;
    unsigned index = 0;
    list_for_each_entry(&mounts, mount, mounts) {
        printf("L %u point=", index);
        print_hex(mount->point);
        printf(" source=");
        print_hex(mount->source);
        printf(" info=");
        print_hex(mount->info);
        printf(" flags=%d refcount=%u root_fd=%d\n", mount->flags, mount->refcount, mount->root_fd);
        index++;
    }
    printf("E count=%u\n", index);
}

static void do_mount_record(const char *fs, const char *source, const char *point, const char *info,
                            unsigned flags) {
    lock(&mounts_lock);
    int err = do_mount(&fake, source, point, info, flags);
    unlock(&mounts_lock);
    printf("M fs=%s source=", fs);
    print_hex(source);
    printf(" point=");
    print_hex(point);
    printf(" info=");
    print_hex(info);
    printf(" flags=%u ret=%d\n", flags, err);
}

// mount_find, retaining: the caller has to give the reference back with
// mount_release, which is what the `R` records below are.
static struct mount *find_record(const char *path) {
    struct mount *mount = mount_find((char *) path);
    printf("Q path=");
    print_hex(path);
    printf(" point=");
    print_hex(mount->point);
    printf(" source=");
    print_hex(mount->source);
    printf(" refcount=%u\n", mount->refcount);
    return mount;
}

static void release_record(struct mount *mount) {
    mount_release(mount);
    printf("R point=");
    print_hex(mount->point);
    printf(" refcount=%u\n", mount->refcount);
}

static void retain_record(struct mount *mount) {
    mount_retain(mount);
    printf("T point=");
    print_hex(mount->point);
    printf(" refcount=%u\n", mount->refcount);
}

static void remove_record(struct mount *mount) {
    unsigned refcount = mount->refcount;
    char point[MAX_PATH];
    hex_into(point, sizeof(point), mount->point);
    lock(&mounts_lock);
    int err = mount_remove(mount);
    unlock(&mounts_lock);
    printf("X point=%s refcount=%u ret=%d\n", point, refcount, err);
}

static void umount_record(const char *point) {
    lock(&mounts_lock);
    int err = do_umount(point);
    unlock(&mounts_lock);
    printf("U point=");
    print_hex(point);
    printf(" ret=%d\n", err);
}

static void param_record(const char *info, const char *flag) {
    printf("P info=");
    print_hex(info);
    printf(" flag=");
    print_hex(flag);
    printf(" ret=%d\n", mount_param_flag(info, flag) ? 1 : 0);
}

// ---- the script ------------------------------------------------------------

int main(void) {
    setvbuf(stdout, NULL, _IONBF, 0);
    printf("# mount reference, generated from unmodified fs/mount.c\n");

    for (unsigned i = 0; i < sizeof(filesystems) / sizeof(filesystems[0]); i++) {
        if (filesystems[i] != NULL)
            printf("S %u name=%s\n", i, filesystems[i]->name);
    }

    printf("K MAX_FILESYSTEMS=%d\n", MAX_FILESYSTEMS);
    printf("K MS_READONLY=%d\n", MS_READONLY_);
    printf("K MS_NOSUID=%d\n", MS_NOSUID_);
    printf("K MS_NODEV=%d\n", MS_NODEV_);
    printf("K MS_NOEXEC=%d\n", MS_NOEXEC_);
    printf("K MS_SILENT=%d\n", MS_SILENT_);
    printf("K MS_SUPPORTED=%d\n", MS_SUPPORTED);
    printf("K MS_FLAGS=%d\n", MS_FLAGS);

    // fs_register fills the first empty slot, so the six registrations below
    // take the table from its four seeded entries to MAX_FILESYSTEMS.
    static struct fs_ops extra[MAX_FILESYSTEMS - 4];
    static const char *const extra_names[] = {"fake0", "fake1", "fake2",
                                              "fake3", "fake4", "fake5"};
    for (unsigned i = 0; i < MAX_FILESYSTEMS - 4; i++) {
        extra[i] = fake;
        extra[i].name = extra_names[i];
        unsigned slot = 0;
        while (filesystems[slot] != NULL)
            slot++;
        fs_register(&extra[i]);
        printf("G name=%s slot=%u\n", extra_names[i], slot);
    }

    // The root filesystem, mounted the way kernel/init.c mounts it: at "".
    do_mount_record("fake", "/dev/disk", "", "", 0);
    print_mounts();

    // The order the list has to keep: a longer point goes in front of every
    // shorter one, and an equal-length point goes in front of the entries that
    // are already there — which is why a lookup of "/dup/x" finds the *second*
    // /dup, and why the list is not the order they were mounted in.
    do_mount_record("fake", "/dev/mnt", "/mnt", "rw,nosuid", 0);
    do_mount_record("fake", "db", "/var/db/fakefs", "ro", MS_READONLY_);
    do_mount_record("fake", "db2", "/mnt/db", "", 0);
    do_mount_record("fake", "var", "/var", "", 0);
    do_mount_record("fake", "dup-old", "/dup", "size=64k", 0);
    do_mount_record("fake", "dup-new", "/dup", "size=128k", 0);
    print_mounts();

    // A filesystem whose mount op fails is not in the list.
    do_mount_record("fake", "broken", "/fail", "", 0);
    print_mounts();

    // mount_find: the longest point that prefixes the path and ends at a
    // component boundary, then the root at the empty point for everything else.
    static const char *const lookups[] = {
        "/",
        "/mnt",
        "/mnt/x",
        "/mnt2",
        "/mnt/x/y",
        "/mnt-db/x",
        "/var",
        "/vary",
        "/var/db",
        "/var/db/fakefs",
        "/var/db/fakefs/x",
        "/dup/y",
        "",
        "/elsewhere",
        "/proc/self",
    };
    for (unsigned i = 0; i < sizeof(lookups) / sizeof(lookups[0]); i++)
        release_record(find_record(lookups[i]));
    print_mounts();

    // A reference keeps its mount alive: while one is outstanding, neither
    // mount_remove nor do_umount will touch it.
    struct mount *referenced = find_record("/var/db/fakefs/x");
    remove_record(referenced);
    umount_record("/var/db/fakefs");
    retain_record(referenced);
    release_record(referenced);
    release_record(referenced);
    remove_record(referenced);
    print_mounts();
    release_record(find_record("/var/db/fakefs/x"));

    // do_umount finds its mount by point, and removes the first of two mounts
    // that share one.
    umount_record("/dup");
    print_mounts();
    umount_record("/nope");
    umount_record("/mnt");
    print_mounts();

    // mount_param_flag: prefix matches, fields that are not flags, and every
    // shape C terminates on — which means the flag has to match inside the
    // first field, or there has to be no comma at all. A parameter list whose
    // *second* field is asked about ("ro,nosuid" and "nosuid") is the loop that
    // never skips the comma, i.e. a hang, and lives in src/mount.rs's unit
    // tests rather than here.
    param_record("ro,nosuid", "ro");
    param_record("ro,nosuid", "ro,nosuid");
    param_record("readonlyx=1", "readonly");
    param_record("nosuid=a", "nosuid");
    param_record("size=64k", "size");
    param_record("size=64k", "size=65k");
    param_record("", "ro");
    param_record("ro", "");
    param_record("", "");
    param_record("user=1000,group=1000", "user=1000");
    param_record("a,b", ",b");
    param_record("ro,", "ro");

    // The fake filesystem's own log, in the order it was called: one mount per
    // successful do_mount, and one umount per mount_remove that got that far.
    printf("F count=%zu\n", callback_count);
    for (size_t i = 0; i < callback_count; i++)
        printf("F %zu %s\n", i, callbacks[i]);
    return 0;
}

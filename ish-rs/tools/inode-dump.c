// Reference-output generator for the fs/inode differential test.
//
// This *includes* fs/inode.c, so inode_get, inode_get_unlocked, inode_retain,
// inode_release and inode_check_orphaned are the real functions, with their
// constants and their static table; fs/mount.c is included as well, because an
// inode is keyed by the mount it belongs to and retains it. Including inode.c
// is also what lets the driver walk the table: its static inodes_hash[] is
// visible here, so the walk records can print every live inode and the bucket
// it is in.
//
// The driver is a script: it mounts three filesystems, gets inodes on them,
// takes and gives back references, writes socket ids, and asks about orphaned
// inodes — one record per observation. tests/inode_differential.rs performs the
// same calls through the port and compares every record, so this file is both
// the corpus and the script.
//
// Records, all of which tests/inode_differential.rs replays:
//   K <name>=<value>                  the constants of fs/inode.c and fs/inode.h
//   M fs=<name> source=<hex> point=<hex> info=<hex> flags=<n> ret=<n>
//                                     do_mount, called with mounts_lock held
//   S point=<hex> refcount=<n>        one mount in the list, in list order
//   Z count=<n>                       the mount list has that many entries
//   G point=<hex> number=<n> new=<0|1> refcount=<n> socket_id=<n> mount_refcount=<n>
//                                     inode_get
//   U point=<hex> number=<n> new=<0|1> refcount=<n> socket_id=<n> mount_refcount=<n>
//                                     inode_get_unlocked, called with inodes_lock held
//   T point=<hex> number=<n> refcount=<n>  inode_retain
//   R point=<hex> number=<n> refcount=<n> gone=<0|1> mount_refcount=<n>
//                                     inode_release: the count before the call
//                                     and whether that was the last reference
//   C point=<hex> number=<n> called=<0|1>  inode_check_orphaned
//   W point=<hex> number=<n> socket_id=<n>  socket_id after the script wrote it
//   L <index> bucket=<n> number=<n> refcount=<n> socket_id=<n> point=<hex>
//                                     one inode in the table, buckets in order
//   E count=<n>                       the table has that many inodes
//   F count=<n>                       the orphan hook's log follows
//   F <index> orphaned point=<hex> number=<n> in_table=<0|1>
//                                     one call, in the order it was made, with
//                                     whether the inode was still in the table
//
// Bytes are hex; the empty string is `-`, since a hex byte string can be empty
// but a field cannot.
//
// The walk prints the inodes of one bucket sorted by (number, mount point)
// rather than in list order: inode_get_unlocked adds each new inode at the head
// of its bucket and nothing in the C can observe that order (the key is unique,
// so a lookup stops at the first match either way), so sorting is a
// canonicalization both sides can produce. A mount is named by its point here,
// so two mounts of the *same* point would be indistinguishable in a walk; the
// script does not mount two filesystems at one point (src/inode.rs's unit tests
// cover what that does to the key).
//
// What is deliberately not recorded, because the C has no value to record: a
// release of an inode with no references left (the count wraps), and
// inode_check_orphaned on a filesystem with no inode_orphaned hook (the call
// goes through NULL). Both are checked by unit tests in src/inode.rs.
//
// Build (normally via tools/gen_inode_reference.sh):
//   cc -O2 -Wall -Wextra -Werror -I<ish-src> -I<ish-rs>/tools/stub-include
//      -o inode-dump tools/inode-dump.c

#define _GNU_SOURCE

#include <assert.h>
#include <pthread.h>
#include <stdarg.h>
#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "kernel/calls.h"
#include "kernel/fs.h"
#include "fs/path.h"
#include "fs/real.h"
#include "fs/inode.h"

// ---- stubs for the parts of iSH this oracle does not link ------------------

// `current` is a thread-local in kernel/task.c. fs/path.c's path_normalize
// reads `current->fs`; only path_is_normalized is called here, so this is only
// there to satisfy the linker.
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

// util/sync.c's, which fs/inode.c initializes each inode's condition with. This
// oracle never waits on one.
void cond_init(cond_t *cond) {
    pthread_cond_init(&cond->cond, NULL);
}

#include "fs/path.c"
#include "fs/mount.c"
#include "fs/inode.c"

// fs/inode.h's private names for the lock types a guest's `struct flock_` asks
// for (asm/fcntl.h), which is why fs/lock.c can compare them to `flock->type`
// directly. Checked here, where both spellings are in scope.
#include <fcntl.h>
_Static_assert(F_RDLCK_ == F_RDLCK && F_WRLCK_ == F_WRLCK && F_UNLCK_ == F_UNLCK,
               "the internal lock types are the guest's");

// The four filesystems kernel/fs.h declares and fs/mount.c seeds its table with.
// The real definitions live in fs/real.c, fs/proc.c, fs/pty.c and fs/tmp.c,
// none of which is ported (or linkable) yet.
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

#define MAX_LOG 64

static char log_entries[MAX_LOG][256];
static size_t log_count;

static void logged(const char *format, ...) {
    assert(log_count < MAX_LOG);
    va_list args;
    va_start(args, format);
    vsnprintf(log_entries[log_count], sizeof(log_entries[log_count]), format, args);
    va_end(args);
    log_count++;
}

// `umount`, kept so that a mount of this filesystem is a whole filesystem; the
// inode table never calls it.
static int fake_umount(struct mount *mount) {
    char point[MAX_PATH];
    hex_into(point, sizeof(point), mount->point);
    logged("umount point=%s", point);
    return 0;
}

// `inode_orphaned`: the callback inode_release and inode_check_orphaned make
// when no inode holds the file any more. fs/fake.c uses it to drop metadata.
//
// It asks the table whether the inode is still there, which is what makes the
// order inside inode_release observable: the inode is unlinked *before* the
// callback runs, so a hook that asked always sees it gone.
static void fake_orphaned(struct mount *mount, ino_t ino) {
    char point[MAX_PATH];
    hex_into(point, sizeof(point), mount->point);
    logged("orphaned point=%s number=%llu in_table=%d", point, (unsigned long long) ino,
           inode_get_data(mount, ino) != NULL ? 1 : 0);
}

static const struct fs_ops fake = {
    .name = "fake",
    .magic = 0x66616b65,
    .umount = fake_umount,
    .inode_orphaned = fake_orphaned,
};

// ---- records ---------------------------------------------------------------

// do_mount, with mounts_lock held the way mount.c's callers hold it.
static void mount_record(const char *source, const char *point, const char *info, unsigned flags) {
    lock(&mounts_lock);
    int err = do_mount(&fake, source, point, info, flags);
    unlock(&mounts_lock);
    printf("M fs=%s source=", fake.name);
    print_hex(source);
    printf(" point=");
    print_hex(point);
    printf(" info=");
    print_hex(info);
    printf(" flags=%u ret=%d\n", flags, err);
}

// The mount list, so that the mount refcounts the inode table moves are
// visible: C's do_mount leaves a new mount at 0 and only references raise it.
static void print_mounts(void) {
    struct mount *mount;
    unsigned index = 0;
    list_for_each_entry(&mounts, mount, mounts) {
        printf("S point=");
        print_hex(mount->point);
        printf(" refcount=%u\n", mount->refcount);
        index++;
    }
    printf("Z count=%u\n", index);
}

// The whole table: every bucket in order, the inodes of a bucket sorted by
// (number, mount point). C's list_add puts each new inode at the head of its
// bucket, which no lookup can observe, so the walk does not depend on it.
#define MAX_WALK 32

static int compare_inodes(const void *left, const void *right) {
    const struct inode_data *a = *(struct inode_data *const *) left;
    const struct inode_data *b = *(struct inode_data *const *) right;
    if (a->number != b->number)
        return a->number < b->number ? -1 : 1;
    return strcmp(a->mount->point, b->mount->point);
}

static void print_inodes(void) {
    struct inode_data *found[MAX_WALK];
    unsigned index = 0;
    for (unsigned bucket = 0; bucket < INODES_HASH_SIZE; bucket++) {
        if (list_null(&inodes_hash[bucket]))
            continue;
        size_t count = 0;
        struct inode_data *inode;
        list_for_each_entry(&inodes_hash[bucket], inode, chain) {
            assert(count < MAX_WALK);
            found[count++] = inode;
        }
        qsort(found, count, sizeof(found[0]), compare_inodes);
        for (size_t i = 0; i < count; i++) {
            inode = found[i];
            printf("L %u bucket=%u number=%llu refcount=%u socket_id=%u point=", index, bucket,
                   (unsigned long long) inode->number, inode->refcount, inode->socket_id);
            print_hex(inode->mount->point);
            printf("\n");
            index++;
        }
    }
    printf("E count=%u\n", index);
}

// inode_get, or inode_get_unlocked with inodes_lock held the way generic_open
// holds it. `new` says whether the call created the inode, which the driver can
// see because it is inside inode.c.
static struct inode_data *get_record(struct mount *mount, ino_t ino, bool unlocked) {
    bool is_new = inode_get_data(mount, ino) == NULL;
    struct inode_data *inode;
    if (unlocked) {
        lock(&inodes_lock);
        inode = inode_get_unlocked(mount, ino);
        unlock(&inodes_lock);
    } else {
        inode = inode_get(mount, ino);
    }
    printf("%s point=", unlocked ? "U" : "G");
    print_hex(mount->point);
    printf(" number=%llu new=%d refcount=%u socket_id=%u mount_refcount=%u\n",
           (unsigned long long) ino, is_new ? 1 : 0, inode->refcount, inode->socket_id,
           mount->refcount);
    return inode;
}

static void retain_record(struct inode_data *inode) {
    inode_retain(inode);
    printf("T point=");
    print_hex(inode->mount->point);
    printf(" number=%llu refcount=%u\n", (unsigned long long) inode->number, inode->refcount);
}

static void release_record(struct inode_data *inode) {
    char point[MAX_PATH];
    hex_into(point, sizeof(point), inode->mount->point);
    unsigned refcount = inode->refcount;
    unsigned long long number = inode->number;
    struct mount *mount = inode->mount;
    inode_release(inode);
    // The inode is gone when this was the last reference, so the count and the
    // number are the ones read before the call; the mount outlives every inode.
    printf("R point=%s number=%llu refcount=%u gone=%d mount_refcount=%u\n", point, number,
           refcount, refcount == 1 ? 1 : 0, mount->refcount);
}

static void check_record(struct mount *mount, ino_t ino) {
    size_t before = log_count;
    inode_check_orphaned(mount, ino);
    printf("C point=");
    print_hex(mount->point);
    printf(" number=%llu called=%d\n", (unsigned long long) ino, log_count != before ? 1 : 0);
}

static void socket_record(struct inode_data *inode, uint32_t id) {
    inode->socket_id = id;
    printf("W point=");
    print_hex(inode->mount->point);
    printf(" number=%llu socket_id=%u\n", (unsigned long long) inode->number, inode->socket_id);
}

// The mount the script wants to hold: C's callers have a `struct mount *`, and
// the driver takes one out of the list without retaining it, so that every
// change to the count is one the inode table made.
static struct mount *bare_mount(const char *point) {
    struct mount *mount;
    list_for_each_entry(&mounts, mount, mounts) {
        if (strcmp(mount->point, point) == 0)
            return mount;
    }
    fprintf(stderr, "no mount at %s\n", point);
    abort();
}

// ---- the script ------------------------------------------------------------

int main(void) {
    setvbuf(stdout, NULL, _IONBF, 0);
    printf("# inode reference, generated from unmodified fs/inode.c\n");

    printf("K INODES_HASH_SIZE=%d\n", INODES_HASH_SIZE);
    printf("K F_RDLCK_=%d\n", F_RDLCK_);
    printf("K F_WRLCK_=%d\n", F_WRLCK_);
    printf("K F_UNLCK_=%d\n", F_UNLCK_);

    // Three filesystems, so that the same inode number on two of them is two
    // different inodes. The root is mounted at the empty point, the way
    // kernel/init.c mounts it.
    mount_record("/dev/disk", "", "", 0);
    mount_record("/dev/mnt", "/mnt", "", 0);
    mount_record("/dev/other", "/other", "", 0);
    print_mounts();

    // Nothing is open yet.
    print_inodes();

    struct mount *root = bare_mount("");
    struct mount *mnt = bare_mount("/mnt");
    struct mount *other = bare_mount("/other");

    // inode_get: the first call creates the inode and retains the mount, the
    // second finds the same inode and only takes another reference.
    struct inode_data *first = get_record(root, 1, false);
    struct inode_data *again = get_record(root, 1, false);
    print_inodes();

    // inode_get_unlocked, called the way generic_open holds inodes_lock across
    // the open and the reference that follows.
    struct inode_data *unlocked = get_record(root, 1, true);
    retain_record(again);
    // The file is open, so inode_check_orphaned has nothing to report.
    check_record(root, 1);
    print_inodes();

    // Giving references back, one at a time. The hook fires only when the last
    // one goes, and that is also when the mount loses the inode's reference.
    release_record(first);
    release_record(again);
    release_record(unlocked);
    print_inodes();
    check_record(root, 1);
    release_record(first);
    print_inodes();
    check_record(root, 1);

    // The key is the (mount, number) pair: the same number on another mount is
    // another inode, and the hash puts it in the same bucket, as it does a
    // number one hash size away. The remaining gets are the shapes the bucket
    // index has to get right: 0, the largest numbers, and a number with bits
    // above the bucket count.
    struct inode_data *root_one = get_record(root, 1, false);
    struct inode_data *other_one = get_record(other, 1, false);
    struct inode_data *collision = get_record(root, 1 + INODES_HASH_SIZE, false);
    struct inode_data *zero = get_record(root, 0, false);
    struct inode_data *huge = get_record(root, ~(ino_t) 0, false);
    struct inode_data *huge_next = get_record(root, ~(ino_t) 0 - 1, false);
    struct inode_data *big = get_record(mnt, 0x100000005ULL, false);
    print_inodes();

    // socket_id is the field fs/sock.c writes when a Unix socket binds an
    // inode; a fresh inode starts at 0.
    socket_record(root_one, 0x1234);
    socket_record(other_one, 0xffffffff);
    print_inodes();

    // Releasing the last inode of a mount releases the mount, and inodes on
    // other mounts are untouched.
    release_record(zero);
    release_record(collision);
    release_record(huge);
    release_record(huge_next);
    release_record(big);
    print_inodes();
    release_record(other_one);
    print_inodes();
    release_record(root_one);
    print_inodes();
    print_mounts();

    printf("F count=%zu\n", log_count);
    for (size_t i = 0; i < log_count; i++)
        printf("F %zu %s\n", i, log_entries[i]);
    return 0;
}

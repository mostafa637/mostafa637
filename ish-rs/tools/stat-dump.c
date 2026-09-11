// Reference-output generator for the fs/stat differential test.
//
// This *includes* fs/stat.c, so `stat_convert_newstat64` is the real thing, and
// it takes every size and offset below from `sizeof`/`offsetof` on the real
// fs/stat.h rather than from a transcription. fs/stat.h needs nothing but
// misc.h, and none of fs/stat.c's syscalls is called, so the stubs after the
// includes are all that has to stand in for the rest of iSH.
//
// Records, all of which tests/stat_differential.rs replays:
//   S <type> size=N                    the struct's size
//   F <type> <field> off=N size=N      one field's offset and width
//   V <type> <field>=<hex>             the distinct value a field was given
//   E <type> bytes=<hex>               the struct's bytes with every field set
//                                      and its padding left at 0xAA, so the
//                                      image shows both where the fields are
//                                      and which bytes are not fields
//   C <case> bytes=<hex>               stat_convert_newstat64's output image
//   C <case> fields …                  the same output, field by field
//   W …                                a note about something indeterminate
//   K statx_basic_stats=<hex>          STATX_BASIC_STATS_
//
// Build (normally via tools/gen_stat_reference.sh):
//   cc -O2 -Wall -Wextra -Werror -I<ish-src> -I<ish-rs>/tools/stub-include
//      -o stat-dump tools/stat-dump.c

#define _GNU_SOURCE

#include <stddef.h>
#include <stdio.h>
#include <string.h>

#include "kernel/fs.h"
#include "fs/path.h"
#include "fs/fd.h"

// ---- stubs for the parts of iSH this oracle does not link ------------------
//
// fs/stat.c defines six syscalls besides the conversion; they have to compile,
// and none of them is called from here.

int path_normalize(struct fd *at, const char *path, char *out, int flags) {
    (void) at;
    (void) path;
    (void) flags;
    out[0] = '\0';
    return -1;
}

struct mount *find_mount_and_trim_path(char *path) {
    (void) path;
    return NULL;
}

void mount_release(struct mount *mount) {
    (void) mount;
}

struct fd *f_get(fd_t f) {
    (void) f;
    return NULL;
}

int user_read_string(addr_t addr, char *buf, size_t max) {
    (void) addr;
    if (max > 0)
        buf[0] = '\0';
    return 1;
}

int user_write(addr_t addr, const void *buf, size_t count) {
    (void) addr;
    (void) buf;
    (void) count;
    return 1;
}

#include "fs/stat.c"

// ---- layout records --------------------------------------------------------

#define SIZE(type) printf("S %s size=%zu\n", #type, sizeof(struct type))
#define FIELD(type, field) \
    printf("F %s %s off=%zu size=%zu\n", #type, #field, offsetof(struct type, field), \
           sizeof(((struct type *) 0)->field))

// ---- value records ---------------------------------------------------------
//
// `SET` assigns a field and records the value, so the corpus's inputs and the
// byte image below can never drift apart.

#define SET(type, field, v) \
    do { \
        value.field = (v); \
        printf("V %s %s=%#llx\n", #type, #field, (unsigned long long) value.field); \
    } while (0)

#define SET_SUB(type, field, sub, v) \
    do { \
        value.field.sub = (v); \
        printf("V %s %s_" #sub "=%#llx\n", #type, #field, \
               (unsigned long long) value.field.sub); \
    } while (0)

#define SET_ELEM(type, field, index, v) \
    do { \
        value.field[index] = (v); \
        printf("V %s %s%zu=%#llx\n", #type, #field, (size_t) (index), \
               (unsigned long long) value.field[index]); \
    } while (0)

static void print_bytes(const char *type, const void *object, size_t size) {
    printf("E %s bytes=", type);
    const unsigned char *bytes = object;
    for (size_t i = 0; i < size; i++)
        printf("%02x", bytes[i]);
    printf("\n");
}

// The value of every field is distinct, so a field that the port writes to the
// wrong offset, truncates, or swaps with another one changes the image. The
// bytes no field covers keep the 0xAA the struct was memset to, which is how
// the fixture says where the padding is.
static void dump_statbuf(void) {
    SIZE(statbuf);
    FIELD(statbuf, dev);
    FIELD(statbuf, inode);
    FIELD(statbuf, mode);
    FIELD(statbuf, nlink);
    FIELD(statbuf, uid);
    FIELD(statbuf, gid);
    FIELD(statbuf, rdev);
    FIELD(statbuf, size);
    FIELD(statbuf, blksize);
    FIELD(statbuf, blocks);
    FIELD(statbuf, atime);
    FIELD(statbuf, atime_nsec);
    FIELD(statbuf, mtime);
    FIELD(statbuf, mtime_nsec);
    FIELD(statbuf, ctime);
    FIELD(statbuf, ctime_nsec);

    struct statbuf value;
    memset(&value, 0xAA, sizeof(value));
    SET(statbuf, dev, 0x1122334455667788);
    SET(statbuf, inode, 0x2233445566778899);
    SET(statbuf, mode, 0x11223344);
    SET(statbuf, nlink, 0x22334455);
    SET(statbuf, uid, 0x33445566);
    SET(statbuf, gid, 0x44556677);
    SET(statbuf, rdev, 0x5566778899aabbcc);
    SET(statbuf, size, 0x66778899aabbccdd);
    SET(statbuf, blksize, 0x778899aa);
    SET(statbuf, blocks, 0x8899aabbccddeeff);
    SET(statbuf, atime, 0x99001122);
    SET(statbuf, atime_nsec, 0xaa112233);
    SET(statbuf, mtime, 0xbb223344);
    SET(statbuf, mtime_nsec, 0xcc334455);
    SET(statbuf, ctime, 0xdd445566);
    SET(statbuf, ctime_nsec, 0xee556677);
    print_bytes("statbuf", &value, sizeof(value));
}

static void dump_oldstat(void) {
    SIZE(oldstat);
    FIELD(oldstat, dev);
    FIELD(oldstat, ino);
    FIELD(oldstat, mode);
    FIELD(oldstat, nlink);
    FIELD(oldstat, uid);
    FIELD(oldstat, gid);
    FIELD(oldstat, rdev);
    FIELD(oldstat, size);
    FIELD(oldstat, atime);
    FIELD(oldstat, mtime);
    FIELD(oldstat, ctime);

    struct oldstat value;
    memset(&value, 0xAA, sizeof(value));
    SET(oldstat, dev, 0x1122);
    SET(oldstat, ino, 0x2233);
    SET(oldstat, mode, 0x3344);
    SET(oldstat, nlink, 0x4455);
    SET(oldstat, uid, 0x5566);
    SET(oldstat, gid, 0x6677);
    SET(oldstat, rdev, 0x7788);
    SET(oldstat, size, 0x8899aabb);
    SET(oldstat, atime, 0x99001122);
    SET(oldstat, mtime, 0xaa112233);
    SET(oldstat, ctime, 0xbb223344);
    print_bytes("oldstat", &value, sizeof(value));
}

static void dump_newstat(void) {
    SIZE(newstat);
    FIELD(newstat, dev);
    FIELD(newstat, ino);
    FIELD(newstat, mode);
    FIELD(newstat, nlink);
    FIELD(newstat, uid);
    FIELD(newstat, gid);
    FIELD(newstat, rdev);
    FIELD(newstat, size);
    FIELD(newstat, blksize);
    FIELD(newstat, blocks);
    FIELD(newstat, atime);
    FIELD(newstat, atime_nsec);
    FIELD(newstat, mtime);
    FIELD(newstat, mtime_nsec);
    FIELD(newstat, ctime);
    FIELD(newstat, ctime_nsec);
    FIELD(newstat, pad);

    struct newstat value;
    memset(&value, 0xAA, sizeof(value));
    SET(newstat, dev, 0x11223344);
    SET(newstat, ino, 0x22334455);
    SET(newstat, mode, 0x3344);
    SET(newstat, nlink, 0x4455);
    SET(newstat, uid, 0x5566);
    SET(newstat, gid, 0x6677);
    SET(newstat, rdev, 0x778899aa);
    SET(newstat, size, 0x8899aabb);
    SET(newstat, blksize, 0x99001122);
    SET(newstat, blocks, 0xaa112233);
    SET(newstat, atime, 0xbb223344);
    SET(newstat, atime_nsec, 0xcc334455);
    SET(newstat, mtime, 0xdd445566);
    SET(newstat, mtime_nsec, 0xee556677);
    SET(newstat, ctime, 0xff667788);
    SET(newstat, ctime_nsec, 0x00112233);
    for (size_t i = 0; i < sizeof(value.pad); i++)
        SET_ELEM(newstat, pad, i, 0xa0 + i);
    print_bytes("newstat", &value, sizeof(value));
}

static void dump_newstat64(void) {
    SIZE(newstat64);
    FIELD(newstat64, dev);
    FIELD(newstat64, _pad1);
    FIELD(newstat64, fucked_ino);
    FIELD(newstat64, mode);
    FIELD(newstat64, nlink);
    FIELD(newstat64, uid);
    FIELD(newstat64, gid);
    FIELD(newstat64, rdev);
    FIELD(newstat64, _pad2);
    FIELD(newstat64, size);
    FIELD(newstat64, blksize);
    FIELD(newstat64, blocks);
    FIELD(newstat64, atime);
    FIELD(newstat64, atime_nsec);
    FIELD(newstat64, mtime);
    FIELD(newstat64, mtime_nsec);
    FIELD(newstat64, ctime);
    FIELD(newstat64, ctime_nsec);
    FIELD(newstat64, ino);

    struct newstat64 value;
    memset(&value, 0xAA, sizeof(value));
    SET(newstat64, dev, 0x1122334455667788);
    SET(newstat64, _pad1, 0x12345678);
    SET(newstat64, fucked_ino, 0x23456789);
    SET(newstat64, mode, 0x11223344);
    SET(newstat64, nlink, 0x33445566);
    SET(newstat64, uid, 0x44556677);
    SET(newstat64, gid, 0x55667788);
    SET(newstat64, rdev, 0x2233445566778899);
    SET(newstat64, _pad2, 0x66778899);
    SET(newstat64, size, 0x33445566778899aa);
    SET(newstat64, blksize, 0x778899aa);
    SET(newstat64, blocks, 0x445566778899aabb);
    SET(newstat64, atime, 0x99001122);
    SET(newstat64, atime_nsec, 0xaa112233);
    SET(newstat64, mtime, 0xbb223344);
    SET(newstat64, mtime_nsec, 0xcc334455);
    SET(newstat64, ctime, 0xdd445566);
    SET(newstat64, ctime_nsec, 0xee556677);
    SET(newstat64, ino, 0x5566778899aabbcc);
    print_bytes("newstat64", &value, sizeof(value));
}

static void dump_statfsbuf(void) {
    SIZE(statfsbuf);
    FIELD(statfsbuf, type);
    FIELD(statfsbuf, bsize);
    FIELD(statfsbuf, blocks);
    FIELD(statfsbuf, bfree);
    FIELD(statfsbuf, bavail);
    FIELD(statfsbuf, files);
    FIELD(statfsbuf, ffree);
    FIELD(statfsbuf, fsid);
    FIELD(statfsbuf, namelen);
    FIELD(statfsbuf, frsize);
    FIELD(statfsbuf, flags);
    FIELD(statfsbuf, spare);

    struct statfsbuf value;
    memset(&value, 0xAA, sizeof(value));
    SET(statfsbuf, type, 0x1122334455667788);
    SET(statfsbuf, bsize, 0x2233445566778899);
    SET(statfsbuf, blocks, 0x33445566778899aa);
    SET(statfsbuf, bfree, 0x445566778899aabb);
    SET(statfsbuf, bavail, 0x5566778899aabbcc);
    SET(statfsbuf, files, 0x66778899aabbccdd);
    SET(statfsbuf, ffree, 0x778899aabbccddee);
    SET(statfsbuf, fsid, 0x8899aabbccddeeff);
    SET(statfsbuf, namelen, 0x9900112233445566);
    SET(statfsbuf, frsize, 0xaa11223344556677);
    SET(statfsbuf, flags, 0xbb22334455667788);
    for (size_t i = 0; i < 4; i++)
        SET_ELEM(statfsbuf, spare, i, 0xcc33445566778899 + i);
    print_bytes("statfsbuf", &value, sizeof(value));
}

static void dump_statfs_(void) {
    SIZE(statfs_);
    FIELD(statfs_, type);
    FIELD(statfs_, bsize);
    FIELD(statfs_, blocks);
    FIELD(statfs_, bfree);
    FIELD(statfs_, bavail);
    FIELD(statfs_, files);
    FIELD(statfs_, ffree);
    FIELD(statfs_, fsid);
    FIELD(statfs_, namelen);
    FIELD(statfs_, frsize);
    FIELD(statfs_, flags);
    FIELD(statfs_, spare);

    struct statfs_ value;
    memset(&value, 0xAA, sizeof(value));
    SET(statfs_, type, 0x11223344);
    SET(statfs_, bsize, 0x22334455);
    SET(statfs_, blocks, 0x33445566);
    SET(statfs_, bfree, 0x44556677);
    SET(statfs_, bavail, 0x55667788);
    SET(statfs_, files, 0x66778899);
    SET(statfs_, ffree, 0x778899aa);
    SET(statfs_, fsid, 0x8899aabbccddeeff);
    SET(statfs_, namelen, 0x99001122);
    SET(statfs_, frsize, 0xaa112233);
    SET(statfs_, flags, 0xbb223344);
    for (size_t i = 0; i < 4; i++)
        SET_ELEM(statfs_, spare, i, 0xcc334455 + i);
    print_bytes("statfs_", &value, sizeof(value));
}

static void dump_statfs64_(void) {
    SIZE(statfs64_);
    FIELD(statfs64_, type);
    FIELD(statfs64_, bsize);
    FIELD(statfs64_, blocks);
    FIELD(statfs64_, bfree);
    FIELD(statfs64_, bavail);
    FIELD(statfs64_, files);
    FIELD(statfs64_, ffree);
    FIELD(statfs64_, fsid);
    FIELD(statfs64_, namelen);
    FIELD(statfs64_, frsize);
    FIELD(statfs64_, flags);
    FIELD(statfs64_, pad);

    struct statfs64_ value;
    memset(&value, 0xAA, sizeof(value));
    SET(statfs64_, type, 0x11223344);
    SET(statfs64_, bsize, 0x22334455);
    SET(statfs64_, blocks, 0x33445566778899aa);
    SET(statfs64_, bfree, 0x445566778899aabb);
    SET(statfs64_, bavail, 0x5566778899aabbcc);
    SET(statfs64_, files, 0x66778899aabbccdd);
    SET(statfs64_, ffree, 0x778899aabbccddee);
    SET(statfs64_, fsid, 0x8899aabbccddeeff);
    SET(statfs64_, namelen, 0x99001122);
    SET(statfs64_, frsize, 0xaa112233);
    SET(statfs64_, flags, 0xbb223344);
    for (size_t i = 0; i < 4; i++)
        SET_ELEM(statfs64_, pad, i, 0xcc334455 + i);
    print_bytes("statfs64_", &value, sizeof(value));
}

static void dump_statx(void) {
    SIZE(statx_timestamp_);
    FIELD(statx_timestamp_, sec);
    FIELD(statx_timestamp_, nsec);
    FIELD(statx_timestamp_, _pad);

    struct statx_timestamp_ stamp;
    memset(&stamp, 0xAA, sizeof(stamp));
    stamp.sec = 0x1122334455667788;
    stamp.nsec = 0x22334455;
    stamp._pad = 0x33445566;
    printf("V statx_timestamp_ sec=%#llx\n", (unsigned long long) stamp.sec);
    printf("V statx_timestamp_ nsec=%#llx\n", (unsigned long long) stamp.nsec);
    printf("V statx_timestamp_ _pad=%#llx\n", (unsigned long long) stamp._pad);
    print_bytes("statx_timestamp_", &stamp, sizeof(stamp));

    SIZE(statx_);
    FIELD(statx_, mask);
    FIELD(statx_, blksize);
    FIELD(statx_, attributes);
    FIELD(statx_, nlink);
    FIELD(statx_, uid);
    FIELD(statx_, gid);
    FIELD(statx_, mode);
    FIELD(statx_, _pad1);
    FIELD(statx_, ino);
    FIELD(statx_, size);
    FIELD(statx_, blocks);
    FIELD(statx_, attributes_mask);
    FIELD(statx_, atime);
    FIELD(statx_, btime);
    FIELD(statx_, ctime);
    FIELD(statx_, mtime);
    FIELD(statx_, rdev_major);
    FIELD(statx_, rdev_minor);
    FIELD(statx_, dev_major);
    FIELD(statx_, dev_minor);
    FIELD(statx_, mnt_id);
    FIELD(statx_, dio_mem_align);
    FIELD(statx_, dio_offset_align);
    FIELD(statx_, _pad2);

    struct statx_ value;
    memset(&value, 0xAA, sizeof(value));
    SET(statx_, mask, 0x11223344);
    SET(statx_, blksize, 0x22334455);
    SET(statx_, attributes, 0x33445566778899aa);
    SET(statx_, nlink, 0x44556677);
    SET(statx_, uid, 0x55667788);
    SET(statx_, gid, 0x66778899);
    SET(statx_, mode, 0x7788);
    SET(statx_, _pad1, 0x8899);
    SET(statx_, ino, 0x9900112233445566);
    SET(statx_, size, 0xaa11223344556677);
    SET(statx_, blocks, 0xbb22334455667788);
    SET(statx_, attributes_mask, 0xcc33445566778899);
    SET_SUB(statx_, atime, sec, 0x1111111111111111);
    SET_SUB(statx_, atime, nsec, 0x22222222);
    SET_SUB(statx_, atime, _pad, 0x23232323);
    SET_SUB(statx_, btime, sec, 0x3333333333333333);
    SET_SUB(statx_, btime, nsec, 0x44444444);
    SET_SUB(statx_, btime, _pad, 0x45454545);
    SET_SUB(statx_, ctime, sec, 0x5555555555555555);
    SET_SUB(statx_, ctime, nsec, 0x66666666);
    SET_SUB(statx_, ctime, _pad, 0x67676767);
    SET_SUB(statx_, mtime, sec, 0x7777777777777777);
    SET_SUB(statx_, mtime, nsec, 0x88888888);
    SET_SUB(statx_, mtime, _pad, 0x89898989);
    SET(statx_, rdev_major, 0xdd445566);
    SET(statx_, rdev_minor, 0xee556677);
    SET(statx_, dev_major, 0xff667788);
    SET(statx_, dev_minor, 0x00112233);
    SET(statx_, mnt_id, 0x1122334455667788);
    SET(statx_, dio_mem_align, 0x22446688);
    SET(statx_, dio_offset_align, 0x33557799);
    for (size_t i = 0; i < 24; i++)
        SET_ELEM(statx_, _pad2, i, 0xc0 + i);
    print_bytes("statx_", &value, sizeof(value));
}

// ---- the conversion corpus -------------------------------------------------

static struct statbuf case_zero(void) {
    return (struct statbuf) {};
}

static struct statbuf case_all_ones(void) {
    struct statbuf stat;
    memset(&stat, 0xFF, sizeof(stat));
    // Every field at its full width, so a narrow copy shows up as a truncation.
    return stat;
}

static struct statbuf case_typical(void) {
    struct statbuf stat = {};
    stat.dev = 0x1234;
    stat.inode = 0x5678;
    stat.mode = 0100644;
    stat.nlink = 3;
    stat.uid = 1000;
    stat.gid = 1000;
    stat.rdev = 0;
    stat.size = 4096;
    stat.blksize = 4096;
    stat.blocks = 8;
    stat.atime = 1;
    stat.atime_nsec = 2;
    stat.mtime = 3;
    stat.mtime_nsec = 4;
    stat.ctime = 5;
    stat.ctime_nsec = 6;
    return stat;
}

static struct statbuf case_wide(void) {
    struct statbuf stat = {};
    stat.dev = 0xffffffffffffffff;
    stat.inode = 0xaaaaaaaa55555555;
    stat.mode = 0xffffffff;
    stat.nlink = 0xffffffff;
    stat.uid = 0xffffffff;
    stat.gid = 0xffffffff;
    stat.rdev = 0xffffffffffffffff;
    stat.size = 0x7fffffffffffffff;
    stat.blksize = 0x00010000;
    stat.blocks = 0xffffffffffffffff;
    stat.atime = 0x7fffffff;
    stat.atime_nsec = 0x3b9aca00;
    stat.mtime = 0xffffffff;
    stat.mtime_nsec = 0xffffffff;
    stat.ctime = 0x00000000;
    stat.ctime_nsec = 0x00000001;
    return stat;
}

// An inode whose low 32 bits and high 32 bits differ, so `fucked_ino` cannot be
// mistaken for the whole value.
static struct statbuf case_inode_split(void) {
    struct statbuf stat = {};
    stat.inode = 0xdeadbeef00000001;
    stat.dev = 0x0000000200000001;
    return stat;
}

struct conversion_case {
    const char *label;
    struct statbuf (*build)(void);
};

static const struct conversion_case cases[] = {
    {"zero", case_zero},
    {"all-ones", case_all_ones},
    {"typical", case_typical},
    {"wide", case_wide},
    {"inode-split", case_inode_split},
};

int main(void) {
    setvbuf(stdout, NULL, _IONBF, 0);
    printf("# stat reference, generated from unmodified fs/stat.h and fs/stat.c\n");

    dump_statbuf();
    dump_oldstat();
    dump_newstat();
    dump_newstat64();
    dump_statfsbuf();
    dump_statfs_();
    dump_statfs64_();
    dump_statx();

    // `stat_convert_newstat64` assigns every field of its result except `_pad1`
    // and `_pad2`, which stay indeterminate and then travel to the guest whole.
    // What the compiler leaves there is not an ABI -O0 leaks the previous stack
    // frame, -O2 happens to leave zeros - so the oracle defines them as zero
    // before the image is printed, which is what the Rust port does too.
    printf("W indeterminate _pad1 _pad2 (never assigned by stat_convert_newstat64, "
           "defined as zero by this oracle)\n");

    for (size_t i = 0; i < sizeof(cases) / sizeof(cases[0]); i++) {
        struct statbuf input = cases[i].build();
        printf("I %s dev=%#llx inode=%#llx mode=%#x nlink=%#x uid=%#x gid=%#x rdev=%#llx "
               "size=%#llx blksize=%#x blocks=%#llx atime=%#x atime_nsec=%#x mtime=%#x "
               "mtime_nsec=%#x ctime=%#x ctime_nsec=%#x\n",
               cases[i].label, (unsigned long long) input.dev, (unsigned long long) input.inode,
               input.mode, input.nlink, input.uid, input.gid, (unsigned long long) input.rdev,
               (unsigned long long) input.size, input.blksize,
               (unsigned long long) input.blocks, input.atime, input.atime_nsec, input.mtime,
               input.mtime_nsec, input.ctime, input.ctime_nsec);
        struct newstat64 out = stat_convert_newstat64(input);
        out._pad1 = 0;
        out._pad2 = 0;
        printf("C %s bytes=", cases[i].label);
        const unsigned char *bytes = (const unsigned char *) &out;
        for (size_t b = 0; b < sizeof(out); b++)
            printf("%02x", bytes[b]);
        printf("\n");
        printf("C %s fields dev=%#llx fucked_ino=%#x ino=%#llx mode=%#x nlink=%#x uid=%#x "
               "gid=%#x rdev=%#llx size=%#llx blksize=%#x blocks=%#llx atime=%#x "
               "atime_nsec=%#x mtime=%#x mtime_nsec=%#x ctime=%#x ctime_nsec=%#x\n",
               cases[i].label, (unsigned long long) out.dev, out.fucked_ino,
               (unsigned long long) out.ino, out.mode, out.nlink, out.uid, out.gid,
               (unsigned long long) out.rdev, (unsigned long long) out.size, out.blksize,
               (unsigned long long) out.blocks, out.atime, out.atime_nsec, out.mtime,
               out.mtime_nsec, out.ctime, out.ctime_nsec);
    }

    printf("K statx_basic_stats=%#x\n", STATX_BASIC_STATS_);
    return 0;
}

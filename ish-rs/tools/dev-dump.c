// Reference-output generator for the fs/dev differential test.
//
// This *includes* the unmodified fs/dev.h, so every constant and every one of
// the four conversions below is the header's own arithmetic (or, for the two
// host conversions, the host libc's `makedev`/`major`/`minor` that the header
// calls). fs/dev.h includes fs/fd.h, which is only needed for the `struct
// dev_ops` declarations at the bottom; nothing from it is used here.
//
// Records:
//   S dev_t_ size=N                     the type's size
//   M <major> <minor> dev=<hex>         dev_make
//   D <dev> major=<n> minor=<n>         dev_major / dev_minor
//   T <dev> back=<hex>                  dev_make(dev_major(dev), dev_minor(dev))
//   H <major> <minor> fake=<hex> real=<hex>
//                                       dev_make, then dev_real_from_fake
//   R <major> <minor> real=<hex> fake=<hex>
//                                       makedev, then dev_fake_from_real
//   X <major> <minor> makedev=<hex>     the host libc's makedev with fields of
//                                       its own, not only ones dev_make can make
//   Y <dev> major=<n> minor=<n>         the host libc's major/minor on a raw
//                                       64-bit device number
//   K <name>=<value>                    the constants of dev.h/devices.h
//
// Build (normally via tools/gen_dev_reference.sh):
//   cc -O2 -Wall -Wextra -Werror -I<ish-src> -I<ish-rs>/tools/stub-include
//      -o dev-dump tools/dev-dump.c

#define _GNU_SOURCE

#include <stddef.h>
#include <stdio.h>

#include "kernel/calls.h"
#include "fs/dev.h"
#include "fs/devices.h"

// `dev_make` computes in `int`, so a large minor overflows it; -O2 and the
// shifts involved happen to wrap, which is what the records below pin. Reading
// through a `volatile` keeps the compiler from folding a signed overflow it is
// allowed to assume away.
static dev_t_ dev_make_reference(int major, int minor) {
    volatile int m = minor;
    volatile int M = major;
    return dev_make(M, m);
}

static void print_dev(int major, int minor) {
    dev_t_ dev = dev_make_reference(major, minor);
    printf("M %d %d dev=%#x\n", major, minor, dev);

    printf("D %#x major=%d minor=%d\n", dev, dev_major(dev), dev_minor(dev));
    printf("T %#x back=%#x\n", dev, dev_make(dev_major(dev), dev_minor(dev)));
}

struct major_minor {
    int major;
    int minor;
};

// The devices fs/devices.h names, plus the edges of each field: the minor is
// split across two fields and the major has twelve bits, so a value that
// crosses a boundary is the interesting case.
static const struct major_minor corpus[] = {
    {0, 0},
    {MEM_MAJOR, DEV_NULL_MINOR},
    {MEM_MAJOR, DEV_ZERO_MINOR},
    {MEM_MAJOR, DEV_FULL_MINOR},
    {MEM_MAJOR, DEV_RANDOM_MINOR},
    {MEM_MAJOR, DEV_URANDOM_MINOR},
    {TTY_CONSOLE_MAJOR, 0},
    {TTY_CONSOLE_MAJOR, 15},
    {TTY_ALTERNATE_MAJOR, DEV_TTY_MINOR},
    {TTY_ALTERNATE_MAJOR, DEV_CONSOLE_MINOR},
    {TTY_ALTERNATE_MAJOR, DEV_PTMX_MINOR},
    {TTY_PSEUDO_MASTER_MAJOR, 0},
    {TTY_PSEUDO_MASTER_MAJOR, 255},
    {TTY_PSEUDO_SLAVE_MAJOR, 1},
    {DYN_DEV_MAJOR, DEV_CLIPBOARD_MINOR},
    {DYN_DEV_MAJOR, DEV_LOCATION_MINOR},
    {1, 0xff},
    {1, 0x100},
    {1, 0xfff},
    {1, 0x1000},
    {1, 0xfff00},
    {1, 0xfffff},
    {1, 0x100000},
    {0xfff, 0},
    {0x1000, 0},
    {-1, -1},
    {-1, 0},
    {0, -1},
};

// The host's `makedev`, `major` and `minor` see more of their arguments than
// dev_make's output ever does: a major can be wider than twelve bits, and a
// device number can come from a host filesystem with bits the fake encoding has
// nowhere to put. Those limbs are invisible through `dev_real_from_fake` and
// `dev_fake_from_real` - whatever the major contributes above bit 11 lands in
// bits the minor's high half already covers - so they are recorded here, against
// the host, and checked directly by the unit test that can call the private
// transcription.
struct host_case {
    int major;
    int minor;
};

static const struct host_case host_cases[] = {
    {0, 0},
    {1, DEV_NULL_MINOR},
    {MEM_MAJOR, 0xff},
    {MEM_MAJOR, 0x100},
    {1, 0x100000},
    {1, 0xfffff},
    {0xfff, 0xfffff},
    {0x1000, 0},
    {0xffff, 0xffff},
    {0x7fffffff, 0x7fffffff},
    {-1, -1},
    {-1, 0},
    {0, -1},
    {0x10000000, 0x10000000},
};

static const unsigned long long host_devices[] = {
    0,
    0x103,
    0xfff001ff,
    0x100000000000ULL,
    0xfffff000000fff00ULL,
    0xffffff000ffULL,
    0xffffffffffffffffULL,
    0x123456789abcdef0ULL,
    0x8000000000000000ULL,
    0x0000000100000000ULL,
    0x0fff000000000000ULL,
    0x00f00000ULL,
    0x0000f00000000000ULL,
    0x00000000000f0000ULL,
};

int main(void) {
    setvbuf(stdout, NULL, _IONBF, 0);
    printf("# dev reference, generated from unmodified fs/dev.h and fs/devices.h\n");
    printf("S dev_t_ size=%zu\n", sizeof(dev_t_));

    for (size_t i = 0; i < sizeof(corpus) / sizeof(corpus[0]); i++)
        print_dev(corpus[i].major, corpus[i].minor);

    // The two host conversions, in both directions. `dev_real_from_fake` takes
    // a fake number and asks the host libc what device it names;
    // `dev_fake_from_real` goes the other way and can lose bits the host
    // encoding has room for and the fake one does not.
    for (size_t i = 0; i < sizeof(corpus) / sizeof(corpus[0]); i++) {
        dev_t_ fake = dev_make(corpus[i].major, corpus[i].minor);
        printf("H %d %d fake=%#x real=%#llx\n", corpus[i].major, corpus[i].minor, fake,
               (unsigned long long) dev_real_from_fake(fake));

        dev_t real = makedev(corpus[i].major, corpus[i].minor);
        printf("R %d %d real=%#llx fake=%#x\n", corpus[i].major, corpus[i].minor,
               (unsigned long long) real, dev_fake_from_real(real));
    }

    // The host macros with arguments of their own, including majors too wide to
    // be fake device numbers and device numbers the fake encoding cannot hold.
    for (size_t i = 0; i < sizeof(host_cases) / sizeof(host_cases[0]); i++) {
        int major = host_cases[i].major;
        int minor = host_cases[i].minor;
        printf("X %d %d makedev=%#llx\n", major, minor,
               (unsigned long long) makedev(major, minor));
        printf("Y %#llx major=%d minor=%d\n", (unsigned long long) makedev(major, minor),
               major(makedev(major, minor)), minor(makedev(major, minor)));
    }

    for (size_t i = 0; i < sizeof(host_devices) / sizeof(host_devices[0]); i++) {
        unsigned long long dev = host_devices[i];
        printf("Y %#llx major=%d minor=%d\n", dev, major(dev), minor(dev));
    }

    printf("K DEV_BLOCK=%d\n", DEV_BLOCK);
    printf("K DEV_CHAR=%d\n", DEV_CHAR);
    printf("K MEM_MAJOR=%d\n", MEM_MAJOR);
    printf("K DEV_NULL_MINOR=%d\n", DEV_NULL_MINOR);
    printf("K DEV_ZERO_MINOR=%d\n", DEV_ZERO_MINOR);
    printf("K DEV_FULL_MINOR=%d\n", DEV_FULL_MINOR);
    printf("K DEV_RANDOM_MINOR=%d\n", DEV_RANDOM_MINOR);
    printf("K DEV_URANDOM_MINOR=%d\n", DEV_URANDOM_MINOR);
    printf("K TTY_CONSOLE_MAJOR=%d\n", TTY_CONSOLE_MAJOR);
    printf("K TTY_ALTERNATE_MAJOR=%d\n", TTY_ALTERNATE_MAJOR);
    printf("K DEV_TTY_MINOR=%d\n", DEV_TTY_MINOR);
    printf("K DEV_CONSOLE_MINOR=%d\n", DEV_CONSOLE_MINOR);
    printf("K DEV_PTMX_MINOR=%d\n", DEV_PTMX_MINOR);
    printf("K TTY_PSEUDO_MASTER_MAJOR=%d\n", TTY_PSEUDO_MASTER_MAJOR);
    printf("K TTY_PSEUDO_SLAVE_MAJOR=%d\n", TTY_PSEUDO_SLAVE_MAJOR);
    printf("K DYN_DEV_MAJOR=%d\n", DYN_DEV_MAJOR);
    printf("K DEV_CLIPBOARD_MINOR=%d\n", DEV_CLIPBOARD_MINOR);
    printf("K DEV_LOCATION_MINOR=%d\n", DEV_LOCATION_MINOR);
    return 0;
}

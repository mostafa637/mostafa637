// Reference-output generator for the fs/path differential test.
//
// This *includes* fs/path.c, so `path_is_normalized` and `path_next_component`
// below are the real functions, and the constants are the real macros. Only
// those two are called: `__path_normalize` and `path_normalize` need mounts,
// the current task and the fd layer, none of which is ported yet, so the stubs
// after the includes are all that has to stand in for the rest of iSH to get
// this file to link.
//
// Records, all of which tests/path_differential.rs replays:
//   K <name>=<value>                  the constants of fs/path.h and kernel/fs.h
//   P <path> normalized=<0|1>         path_is_normalized
//   C <input> ret=1 component=<rest>  path_next_component returned true
//   C <input> ret=0 err=<n>           it returned false: either the end of the
//                                     path (err=0) or an error
//
// Bytes are hex; the empty string is `-`, since a hex byte string can be empty
// but a field cannot. `C` records always name the input they were given, so
// replaying them is a matter of threading `rest` of one record into the `input`
// of the next — and `ret=0` ends a path, which is what separates one walk from
// the next.
//
// Two C behaviours are deliberately not recorded, because the port cannot have
// them: on `_ENAMETOOLONG` the C has already copied MAX_NAME bytes into the
// caller's buffer and writes no terminator (the buffer is dead on return, since
// every caller aborts the walk when the function returns false), and a walk
// over a path that does not begin with `/` is an assertion failure rather than
// a value.
//
// Build (normally via tools/gen_path_reference.sh):
//   cc -O2 -Wall -Wextra -Werror -I<ish-src> -I<ish-rs>/tools/stub-include
//      -o path-dump tools/path-dump.c

#define _GNU_SOURCE

#include <stdbool.h>
#include <stddef.h>
#include <stdio.h>
#include <string.h>

#include "kernel/calls.h"
#include "fs/path.h"

// ---- stubs for the parts of iSH this oracle does not link ------------------

// `current` is a thread-local in kernel/task.c; path_normalize reads
// `current->fs` and is never called here.
__thread struct task *current;

struct mount *find_mount_and_trim_path(char *path) {
    (void) path;
    return NULL;
}

void mount_release(struct mount *mount) {
    (void) mount;
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

#include "fs/path.c"

// ---- record helpers --------------------------------------------------------

// Hex, or `-` for the empty string.
static void print_hex(const char *data) {
    if (*data == '\0') {
        printf("-");
        return;
    }
    for (const unsigned char *p = (const unsigned char *) data; *p != '\0'; p++)
        printf("%02x", *p);
}

static void print_normalized(const char *path) {
    printf("P ");
    print_hex(path);
    printf(" normalized=%d\n", path_is_normalized(path) ? 1 : 0);
}

// One walk: the C's own calling convention, `while (path_next_component(...))`,
// so the final record is the call that stopped it. The component buffer is
// MAX_NAME + 1 as every caller declares it.
static void walk(const char *path) {
    const char *p = path;
    for (;;) {
        char component[MAX_NAME + 1];
        memset(component, 0xAA, sizeof(component));
        int err = 0;
        const char *input = p;

        if (!path_next_component(&p, component, &err)) {
            printf("C ");
            print_hex(input);
            printf(" ret=0 err=%d\n", err);
            return;
        }

        printf("C ");
        print_hex(input);
        printf(" ret=1 component=");
        print_hex(component);
        printf(" rest=");
        print_hex(p);
        printf("\n");
    }
}

// A path of one component of exactly `name_len` characters.
static void make_long_path(char *dst, size_t name_len) {
    dst[0] = '/';
    memset(dst + 1, 'a', name_len);
    dst[name_len + 1] = '\0';
    // keep the first character distinct so a hex dump is readable
    dst[1] = 'l';
}

// The corpus. Every path that is walked begins with a single slash; the ones
// that do not are only fed to path_is_normalized.
static const char *const paths[] = {
    // the shapes normalization can produce
    "",
    "/",
    "/a",
    "/a/b",
    "/a/b/c",
    "/foo.txt",
    "/dir/file",
    "/a/bb/ccc/dddd/eeeee",
    "/1",
    "/1/2/3/4/5/6/7/8/9",

    // what it must reject
    "a",
    "a/b",
    "a/",
    "//",
    "//a",
    "/a//b",
    "/a/",
    "/a//",
    "///",
    "/a///b",
    " /a",
    "/a b",
    "/a\tb",
    "/a\r\nb",

    // components that are not names, and names that are not components
    "/.",
    "/..",
    "/./a",
    "/../a",
    "/a/.",
    "/a/..",
    "/a/./b",
    "/a/../b",
    "/...",
    "/....a",
    "/.a",
    "/a.",
    "/ /",

    // dots that only look like path components
    "/..a",
    "/..a/b",
    "/a..b",
    "/a.../b",

    // bytes that are not text
    "/\377\376",
    "/\200",
    "/caf\303\251",
    "/\001\002",
    "/\177",
    "/\"",
    "/'",
    "/;",
    "/%s",
    "/%n%n%n",
    "/\\",
    "/$HOME",
    "/~",
    "/a=b",
    "/-",
    "/*",

    // trailing slashes and lone slashes in the middle
    "/a/b/",
    "/a/b//",
    "/a/ /",
};

int main(void) {
    setvbuf(stdout, NULL, _IONBF, 0);
    printf("# path reference, generated from unmodified fs/path.c\n");
    printf("K MAX_PATH=%d\n", MAX_PATH);
    printf("K MAX_NAME=%d\n", MAX_NAME);
    printf("K N_SYMLINK_FOLLOW=%d\n", N_SYMLINK_FOLLOW);
    printf("K N_SYMLINK_NOFOLLOW=%d\n", N_SYMLINK_NOFOLLOW);
    printf("K N_PARENT_DIR_WRITE=%d\n", N_PARENT_DIR_WRITE);

    for (size_t i = 0; i < sizeof(paths) / sizeof(paths[0]); i++)
        print_normalized(paths[i]);

    for (size_t i = 0; i < sizeof(paths) / sizeof(paths[0]); i++) {
        if (paths[i][0] == '/')
            walk(paths[i]);
    }

    // Components of every length around MAX_NAME: the last one that fits is
    // MAX_NAME - 1 characters, because the buffer holds a terminator too.
    static char long_path[MAX_NAME + 2];
    const size_t lengths[] = {1, 2, 3, MAX_NAME - 4, MAX_NAME - 2, MAX_NAME - 1, MAX_NAME,
                              MAX_NAME + 1, 2 * MAX_NAME};
    for (size_t i = 0; i < sizeof(lengths) / sizeof(lengths[0]); i++) {
        make_long_path(long_path, lengths[i]);
        printf("L %zu bytes=%zu\n", lengths[i], strlen(long_path));
        print_normalized(long_path);
        walk(long_path);
    }
    return 0;
}

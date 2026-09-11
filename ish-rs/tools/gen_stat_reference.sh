#!/bin/sh
# Regenerate tests/fixtures/stat_reference.txt from unmodified iSH C.
#
#   ISH_SRC=/path/to/ish tools/gen_stat_reference.sh
#
# The oracle *includes* fs/stat.c, so `stat_convert_newstat64` and the struct
# definitions it reads are the unmodified C, and it takes its sizes and offsets
# from `sizeof`/`offsetof` on those definitions rather than from a transcription.
# fs/stat.h needs nothing but misc.h and fs/stat.c's other functions are never
# called, so the stubs at the top of tools/stat-dump.c are all that has to stand
# in for the rest of iSH.
#
# Two kinds of evidence are recorded:
#
#   S/F  the size of each struct and the offset/width of each field, so a Rust
#        layout that disagrees with the C compiler fails loudly;
#   E    a byte image of each struct with every field holding a distinct value,
#        which pins the *padding* too (statbuf has four bytes of it between
#        `blksize` and `blocks`);
#   C    `stat_convert_newstat64` applied to four statbufs, as bytes and as
#        fields, so the copies and the one duplication (`fucked_ino` and `ino`
#        both come from `stat.inode`) are both visible.
#
# The struct's two unwritten padding fields are zeroed by the oracle (see the
# `W` record) because the C leaves them indeterminate: at -O0 they leak the
# previous stack frame, at -O2 they happen to be zero. That is not an ABI, so it
# is not compared.
set -eu

cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish-src}"

for source in "$ISH_SRC/fs/stat.h" "$ISH_SRC/fs/stat.c" "$ISH_SRC/misc.h"; do
    if [ ! -f "$source" ]; then
        echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout" >&2
        exit 1
    fi
done

cc -O2 -Wall -Wextra -Werror -I"$ISH_SRC" -Itools/stub-include \
    -o /tmp/stat-dump tools/stat-dump.c

tmp=/tmp/stat_reference.txt.new
/tmp/stat-dump >"$tmp"

count() {
    grep -c "$1" "$tmp"
}

sizes=$(count '^S ')
fields=$(count '^F ')
values=$(count '^V ')
images=$(count '^E ')
cases=$(count '^C .* bytes=')
records=$(count '^C .* fields ')
inputs=$(count '^I ')

if [ "$sizes" -ne 9 ] || [ "$fields" -ne 126 ] || [ "$values" -ne 173 ] || \
   [ "$images" -ne 9 ] || [ "$cases" -ne 5 ] || [ "$records" -ne "$cases" ] || \
   [ "$inputs" -ne "$cases" ]; then
    echo "unexpected corpus shape: $sizes sizes, $fields fields, $values values, $images images, $cases conversions" >&2
    exit 1
fi

# The three layouts the guest ABI depends on. `newstat64` is packed: if the
# `__attribute__((packed))` ever stopped applying, its size would jump and the
# syscall would start writing the wrong bytes into the guest.
grep -q '^S statbuf size=88$' "$tmp" || { echo "statbuf is no longer 88 bytes" >&2; exit 1; }
grep -q '^S newstat size=64$' "$tmp" || { echo "newstat is no longer 64 bytes" >&2; exit 1; }
grep -q '^S newstat64 size=96$' "$tmp" || { echo "newstat64 is no longer 96 bytes" >&2; exit 1; }
grep -q '^S statfs_ size=64$' "$tmp" || { echo "statfs_ is no longer 64 bytes" >&2; exit 1; }
grep -q '^S statfs64_ size=84$' "$tmp" || { echo "statfs64_ is no longer 84 bytes" >&2; exit 1; }
grep -q '^S statx_ size=256$' "$tmp" || { echo "statx_ is no longer 256 bytes" >&2; exit 1; }

# statbuf's four bytes of padding after `blksize`, and newstat64's two fields
# that sit at offsets a naturally-aligned struct could not use.
grep -q '^F statbuf blksize off=48 size=4$' "$tmp" || exit 1
grep -q '^F statbuf blocks off=56 size=8$' "$tmp" || exit 1
grep -q '^F newstat64 _pad1 off=8 size=4$' "$tmp" || exit 1
grep -q '^F newstat64 fucked_ino off=12 size=4$' "$tmp" || exit 1
grep -q '^F newstat64 _pad2 off=40 size=4$' "$tmp" || exit 1
grep -q '^F newstat64 size off=44 size=8$' "$tmp" || exit 1
grep -q '^F statfs_ fsid off=28 size=8$' "$tmp" || exit 1

# The conversion: `dev`/`ino` are the full 64-bit fields and `fucked_ino` is the
# 32-bit view of the same inode, so a case whose inode halves differ is the one
# that catches a port that keeps only one of them.
grep -q '^C typical fields dev=0x1234 fucked_ino=0x5678 ino=0x5678 ' "$tmp" || {
    echo "the inode is no longer used for both fucked_ino and ino" >&2
    exit 1
}
# An inode whose halves differ, and a device whose high half is zero: a port
# that narrows either to 32 bits cannot reproduce both of these.
grep -q '^C inode-split fields dev=0x200000001 fucked_ino=0x1 ino=0xdeadbeef00000001 ' "$tmp" || {
    echo "the 64-bit dev/ino copy changed shape" >&2
    exit 1
}
grep -q '^C zero bytes=0000' "$tmp" || { echo "the zero case is not zero" >&2; exit 1; }
grep -q '^W indeterminate _pad1 _pad2 ' "$tmp" || {
    echo "the record of the unwritten padding is missing" >&2
    exit 1
}
grep -q '^K statx_basic_stats=0x7ff$' "$tmp" || { echo "STATX_BASIC_STATS_ changed" >&2; exit 1; }

out=tests/fixtures/stat_reference.txt
mv "$tmp" "$out"
echo "wrote $out: $(wc -l <"$out") lines, $sizes struct layouts, $cases conversions"

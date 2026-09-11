#!/bin/sh
# Regenerate tests/fixtures/mount_reference.txt from unmodified fs/mount.c.
#
#   ISH_SRC=/path/to/ish tools/gen_mount_reference.sh
#
# The oracle *includes* fs/mount.c (and links fs/path.c, whose predicate
# mount_find asserts), so every function in the fixture is the real one. It
# mounts a fake filesystem at a fixed series of points, looks paths up, takes
# and gives back references, and removes mounts, recording one line per
# observation; tests/mount_differential.rs replays those records through the
# port. The records are therefore both the corpus and the script, and the checks
# below are the ones that make sense without a filesystem tree: the list stays
# in descending order of mount point length, every lookup found a mount point
# the list actually had, every block of list records agrees with its own count,
# and the fake filesystem's log is complete.
set -eu

cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish-src}"

for source in \
    "$ISH_SRC/fs/mount.c" \
    "$ISH_SRC/fs/path.c" \
    "$ISH_SRC/kernel/fs.h" \
    "$ISH_SRC/kernel/calls.h" \
    "$ISH_SRC/fs/real.c" \
    "$ISH_SRC/fs/proc.c" \
    "$ISH_SRC/fs/pty.c" \
    "$ISH_SRC/fs/tmp.c"
do
    if [ ! -f "$source" ]; then
        echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout: no $source" >&2
        exit 1
    fi
done

cc -O2 -Wall -Wextra -Werror -I"$ISH_SRC" -Itools/stub-include \
    -o /tmp/mount-dump tools/mount-dump.c

tmp=/tmp/mount_reference.txt.new
/tmp/mount-dump >"$tmp"

count() {
    grep -c "$1" "$tmp"
}

# The shape of the script, so that a change to the driver is a failure here
# rather than a weaker differential test.
if [ "$(count '^S ')" -ne 4 ] || [ "$(count '^K ')" -ne 8 ] || [ "$(count '^G ')" -ne 6 ]; then
    echo "unexpected corpus shape: $(count '^S ') seeded, $(count '^K ') constants, $(count '^G ') registrations" >&2
    exit 1
fi
if [ "$(count '^M ')" -ne 8 ] || [ "$(count '^Q ')" -ne 17 ] || [ "$(count '^T ')" -ne 1 ]; then
    echo "unexpected corpus shape: $(count '^M ') mounts, $(count '^Q ') lookups, $(count '^T ') retains" >&2
    exit 1
fi
if [ "$(count '^R ')" -ne 18 ] || [ "$(count '^X ')" -ne 2 ] || [ "$(count '^U ')" -ne 4 ]; then
    echo "unexpected corpus shape: $(count '^R ') releases, $(count '^X ') removals, $(count '^U ') umounts" >&2
    exit 1
fi
if [ "$(count '^P ')" -ne 12 ] || [ "$(count '^L ')" -ne 37 ] || [ "$(count '^E ')" -ne 7 ]; then
    echo "unexpected corpus shape: $(count '^P ') parameters, $(count '^L ') list entries, $(count '^E ') blocks" >&2
    exit 1
fi
if [ "$(count '^F ')" -ne 12 ]; then
    echo "unexpected corpus shape: $(count '^F ') filesystem callback records" >&2
    exit 1
fi

# The four filesystems fs/mount.c seeds its table with are values defined in
# other files, so the fixture's names are checked against those files' own
# initializers rather than against the oracle's own stubs.
seeded_name() {
    awk -v var="$2" '
        $0 ~ ("const struct fs_ops " var " = ") { found = 1 }
        found && match($0, /\.name = "[^"]*"/) {
            print substr($0, RSTART + 9, RLENGTH - 10)
            exit
        }' "$1"
}

expected="$(seeded_name "$ISH_SRC/fs/real.c" realfs) $(seeded_name "$ISH_SRC/fs/proc.c" procfs) \
$(seeded_name "$ISH_SRC/fs/pty.c" devptsfs) $(seeded_name "$ISH_SRC/fs/tmp.c" tmpfs)"
actual="$(awk '/^S / { sub(/name=/, "", $3); printf "%s ", $3 }' "$tmp")"
if [ "$actual" != "$expected " ]; then
    echo "the seeded filesystems moved: fixture says '$actual', sources say '$expected '" >&2
    exit 1
fi

# The constants the port defines, as fs/mount.c and kernel/calls.h define them.
for constant in \
    "K MAX_FILESYSTEMS=10" \
    "K MS_READONLY=1" \
    "K MS_NOSUID=2" \
    "K MS_NODEV=4" \
    "K MS_NOEXEC=8" \
    "K MS_SILENT=32768" \
    "K MS_SUPPORTED=32783" \
    "K MS_FLAGS=15"
do
    grep -qx "$constant" "$tmp" || { echo "$constant moved" >&2; exit 1; }
done

# The shapes that are easy to get wrong: the root is mounted at the empty
# point, a lookup falls through to it, a failed mount is not in the list, and a
# second mount on the same point is the one a lookup finds.
grep -q '^M fs=fake source=2f6465762f6469736b point=- info=- flags=0 ret=0$' "$tmp" || {
    echo "the root is no longer mounted at the empty point" >&2
    exit 1
}
grep -q '^Q path=2f6d6e7432 point=- source=2f6465762f6469736b refcount=1$' "$tmp" || {
    echo "/mnt2 no longer falls through to the root" >&2
    exit 1
}
grep -q '^Q path=2f6475702f79 point=2f647570 source=6475702d6e6577 refcount=1$' "$tmp" || {
    echo "the newer /dup is no longer the one a lookup finds" >&2
    exit 1
}
grep -q '^M fs=fake source=62726f6b656e point=2f6661696c info=- flags=0 ret=-22$' "$tmp" || {
    echo "the failing filesystem's mount no longer fails" >&2
    exit 1
}
grep -q '^X point=2f7661722f64622f66616b656673 refcount=1 ret=-16$' "$tmp" || {
    echo "a referenced mount is no longer busy" >&2
    exit 1
}
grep -q '^F 8 umount point=2f7661722f64622f66616b656673$' "$tmp" || {
    echo "mount_remove no longer calls the filesystem's umount" >&2
    exit 1
}

# Replay the records the way tests/mount_differential.rs does, and check the
# invariants of the script itself.
python3 - "$tmp" <<'CHECK'
import re, sys

lines = open(sys.argv[1]).read().splitlines()

def unhex(field):
    return b"" if field == "-" else bytes.fromhex(field)

def field(line, key):
    for part in line.split(" "):
        if part.startswith(key + "="):
            return part[len(key) + 1:]
    raise AssertionError(f"no {key}= in {line!r}")

# The list records, in blocks ended by a count.
blocks = 0
index = 0
lengths = []
for line in lines:
    if line.startswith("L "):
        assert line.split(" ")[1] == str(index), f"out of order: {line!r}"
        lengths.append(len(unhex(field(line, "point"))))
        index += 1
    elif line.startswith("E count="):
        assert int(field(line, "count")) == index, f"count disagrees with the block: {line!r}"
        # the invariant the whole lookup rule rests on
        assert lengths == sorted(lengths, reverse=True), f"block {blocks} is out of order: {lengths}"
        lengths = []
        index = 0
        blocks += 1
assert index == 0, "a block of list records was left unterminated"
assert blocks == 7, f"{blocks} blocks"

# Every lookup, retain, release and removal names a mount point the list has
# held by then.
known = set()
for line in lines:
    if line.startswith("L "):
        known.add(unhex(field(line, "point")))
    elif line.startswith(("Q ", "T ", "R ", "X ")):
        point = unhex(field(line, "point"))
        assert point in known, f"{line!r} names a mount point the list never had"

# Every do_umount that succeeded named one, too.
for line in lines:
    if line.startswith("U ") and line.endswith("ret=0"):
        assert unhex(field(line, "point")) in known, f"{line!r} removed nothing"

# The callback log is complete, indexed from 0, and last.
count_line = [i for i, line in enumerate(lines) if line.startswith("F count=")]
assert len(count_line) == 1, "expected exactly one F count record"
at = count_line[0]
total = int(field(lines[at], "count"))
tail = lines[at + 1:]
assert len(tail) == total, f"{len(tail)} callback records for a count of {total}"
for i, line in enumerate(tail):
    assert line.startswith(f"F {i} "), f"callback {i} is out of order: {line!r}"
    assert re.match(r"^F \d+ (mount point=(-|[0-9a-f]+) source=(-|[0-9a-f]+) info=(-|[0-9a-f]+) ret=-?\d+|umount point=(-|[0-9a-f]+))$", line), \
        f"malformed callback record: {line!r}"
print(f"replayed {blocks} list blocks and {total} callbacks")
CHECK

out=tests/fixtures/mount_reference.txt
mv "$tmp" "$out"
echo "wrote $out: $(wc -l <"$out") lines, md5 $(md5sum <"$out" | cut -d' ' -f1)"

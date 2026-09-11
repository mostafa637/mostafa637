#!/bin/sh
# Regenerate tests/fixtures/inode_reference.txt from unmodified fs/inode.c.
#
#   ISH_SRC=/path/to/ish tools/gen_inode_reference.sh
#
# The oracle *includes* fs/inode.c and fs/mount.c, so every function in the
# fixture is the real one — including the static inodes_hash[] the walk records
# are read out of. It mounts a fake filesystem with an orphan hook at three
# points, gets inodes on them, takes and gives back references, writes socket
# ids, and asks about orphaned inodes; tests/inode_differential.rs replays the
# records through the port. The records are therefore both the corpus and the
# script, and the checks below are the ones that define what the script means:
# every mount refcount in the file is exactly the number of inodes alive on that
# mount, every hook call is accounted for by a release or an orphan check, the
# hash keeps its bucket order, and the table is empty when nothing holds an
# inode.
set -eu

cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish-src}"

for source in \
    "$ISH_SRC/fs/inode.c" \
    "$ISH_SRC/fs/inode.h" \
    "$ISH_SRC/fs/mount.c" \
    "$ISH_SRC/fs/path.c" \
    "$ISH_SRC/fs/lock.c" \
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
    -o /tmp/inode-dump tools/inode-dump.c

tmp=/tmp/inode_reference.txt.new
/tmp/inode-dump >"$tmp"

count() {
    grep -c "$1" "$tmp"
}

# The shape of the script, so that a change to the driver is a failure here
# rather than a weaker differential test.
if [ "$(count '^K ')" -ne 4 ] || [ "$(count '^M ')" -ne 3 ] || [ "$(count '^S ')" -ne 6 ] || [ "$(count '^Z ')" -ne 2 ]; then
    echo "unexpected corpus shape: $(count '^K ') constants, $(count '^M ') mounts, $(count '^S ') mount records, $(count '^Z ') mount blocks" >&2
    exit 1
fi
if [ "$(count '^G ')" -ne 9 ] || [ "$(count '^U ')" -ne 1 ] || [ "$(count '^T ')" -ne 1 ]; then
    echo "unexpected corpus shape: $(count '^G ') gets, $(count '^U ') unlocked gets, $(count '^T ') retains" >&2
    exit 1
fi
if [ "$(count '^R ')" -ne 11 ] || [ "$(count '^C ')" -ne 3 ] || [ "$(count '^W ')" -ne 2 ]; then
    echo "unexpected corpus shape: $(count '^R ') releases, $(count '^C ') orphan checks, $(count '^W ') socket writes" >&2
    exit 1
fi
if [ "$(count '^L ')" -ne 20 ] || [ "$(count '^E ')" -ne 10 ] || [ "$(count '^F ')" -ne 10 ]; then
    echo "unexpected corpus shape: $(count '^L ') table entries, $(count '^E ') walks, $(count '^F ') log records" >&2
    exit 1
fi

# The constants the port defines, as fs/inode.c and fs/inode.h define them. The
# oracle static-asserts that the three lock types are the guest's F_RDLCK,
# F_WRLCK and F_UNLCK, which is what makes fs/lock.c's comparisons meaningful.
for constant in \
    "K INODES_HASH_SIZE=1024" \
    "K F_RDLCK_=0" \
    "K F_WRLCK_=1" \
    "K F_UNLCK_=2"
do
    grep -qx "$constant" "$tmp" || { echo "$constant moved" >&2; exit 1; }
done

# The root filesystem is mounted at the empty point, the way kernel/init.c
# mounts it, and only an inode raises a mount's count from 0.
grep -q '^M fs=fake source=2f6465762f6469736b point=- info=- flags=0 ret=0$' "$tmp" || {
    echo "the root is no longer mounted at the empty point" >&2
    exit 1
}
grep -q '^S point=- refcount=0$' "$tmp" || {
    echo "a fresh mount is no longer at no references" >&2
    exit 1
}

# The hash function and the key: inode 1 and inode 1025 share bucket 1, the same
# number on two mounts is two inodes in that bucket, and the index is `ino %
# 1024` for 0 and for the largest numbers there are.
grep -q '^L 0 bucket=0 number=0 refcount=1 socket_id=0 point=-$' "$tmp" || {
    echo "inode 0 no longer lands in bucket 0" >&2
    exit 1
}
grep -q '^L 1 bucket=1 number=1 refcount=1 socket_id=0 point=-$' "$tmp" || {
    echo "inode 1 no longer lands in bucket 1" >&2
    exit 1
}
grep -q '^L 2 bucket=1 number=1 refcount=1 socket_id=0 point=2f6f74686572$' "$tmp" || {
    echo "the same number on another mount is no longer another inode in the same bucket" >&2
    exit 1
}
grep -q '^L 3 bucket=1 number=1025 refcount=1 socket_id=0 point=-$' "$tmp" || {
    echo "inode 1025 no longer shares bucket 1 with inode 1" >&2
    exit 1
}
grep -q '^L 4 bucket=5 number=4294967301 refcount=1 socket_id=0 point=2f6d6e74$' "$tmp" || {
    echo "a number above the bucket count no longer lands in its own bucket" >&2
    exit 1
}
grep -q '^L 5 bucket=1022 number=18446744073709551614 refcount=1 socket_id=0 point=-$' "$tmp" || {
    echo "the second largest inode number no longer lands in bucket 1022" >&2
    exit 1
}
grep -q '^L 6 bucket=1023 number=18446744073709551615 refcount=1 socket_id=0 point=-$' "$tmp" || {
    echo "the largest inode number no longer lands in bucket 1023" >&2
    exit 1
}

# A reference is per inode, not per get: the second get finds the same inode and
# does not retain the mount a second time, and inode_get_unlocked finds it too.
grep -q '^G point=- number=1 new=1 refcount=1 socket_id=0 mount_refcount=1$' "$tmp" || {
    echo "the first inode_get no longer creates the inode and retains the mount" >&2
    exit 1
}
grep -q '^G point=- number=1 new=0 refcount=2 socket_id=0 mount_refcount=1$' "$tmp" || {
    echo "the second inode_get no longer finds the same inode" >&2
    exit 1
}
grep -q '^U point=- number=1 new=0 refcount=3 socket_id=0 mount_refcount=1$' "$tmp" || {
    echo "inode_get_unlocked no longer finds the inode inode_get made" >&2
    exit 1
}
grep -q '^T point=- number=1 refcount=4$' "$tmp" || {
    echo "inode_retain no longer takes a reference" >&2
    exit 1
}
grep -q '^C point=- number=1 called=0$' "$tmp" || {
    echo "inode_check_orphaned no longer stays quiet about a live inode" >&2
    exit 1
}
grep -q '^R point=- number=1 refcount=4 gone=0 mount_refcount=1$' "$tmp" || {
    echo "the first inode_release no longer gives one reference back" >&2
    exit 1
}
grep -q '^R point=- number=1 refcount=1 gone=1 mount_refcount=0$' "$tmp" || {
    echo "the last inode_release no longer destroys the inode and releases the mount" >&2
    exit 1
}
grep -q '^C point=- number=1 called=1$' "$tmp" || {
    echo "inode_check_orphaned no longer reports a destroyed inode" >&2
    exit 1
}
grep -q '^W point=- number=1 socket_id=4660$' "$tmp" || {
    echo "socket_id no longer round-trips" >&2
    exit 1
}
grep -q '^W point=2f6f74686572 number=1 socket_id=4294967295$' "$tmp" || {
    echo "socket_id is no longer 32 bits wide" >&2
    exit 1
}
tail -1 "$tmp" | grep -q '^F [0-9]* orphaned point=- number=1 in_table=0$' || {
    echo "the last orphan hook call is no longer the last release" >&2
    exit 1
}

# Replay the records the way tests/inode_differential.rs does, and check the
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

# The mounts the script has, in list order, and their refcounts: C's do_mount
# leaves a mount at 0 references, so every reference in this file is one an
# inode took.
mount_points = [unhex(field(line, "point")) for line in lines if line.startswith("S ")]
mount_blocks = [int(field(line, "count")) for line in lines if line.startswith("Z ")]
assert mount_blocks == [3, 3], mount_blocks
assert mount_points[:3] == [b"/other", b"/mnt", b""], mount_points

# Live inodes per mount, from the gets that created one and the releases that
# destroyed one; every mount_refcount in the file has to agree, and that is what
# says the inode table owns the references the mount's count moves.
live = {point: 0 for point in mount_points}
keys = set()
walk_keys = []
expected_hooks = []
for line in lines:
    kind = line.split(" ")[0]
    if kind in ("G", "U"):
        point = unhex(field(line, "point"))
        number = int(field(line, "number"))
        if field(line, "new") == "1":
            live[point] += 1
        assert int(field(line, "mount_refcount")) == live[point], f"mount count disagrees: {line!r}"
        assert int(field(line, "refcount")) >= 1, line
        assert point in live, f"{line!r} names a mount the list never had"
        keys.add((point, number))
    elif kind == "T":
        assert int(field(line, "refcount")) >= 2, line
    elif kind == "R":
        point = unhex(field(line, "point"))
        number = int(field(line, "number"))
        assert (point, number) in keys, f"{line!r} releases an inode that was never got"
        before = int(field(line, "refcount"))
        gone = field(line, "gone") == "1"
        assert gone == (before == 1), f"gone disagrees with the count: {line!r}"
        if gone:
            live[point] -= 1
            expected_hooks.append((point, number))
        assert int(field(line, "mount_refcount")) == live[point], f"mount count disagrees: {line!r}"
        if gone:
            keys.discard((point, number))
    elif kind == "C":
        point = unhex(field(line, "point"))
        number = int(field(line, "number"))
        called = field(line, "called") == "1"
        assert called == ((point, number) not in keys), f"the orphan check disagrees: {line!r}"
        if called:
            expected_hooks.append((point, number))
    elif kind == "E":
        # What the walk below has to see: the references the records above left
        # behind, on the mounts those references are for.
        walk_keys.append(len(keys))
        per_mount = {}
        for key_point, _ in keys:
            per_mount[key_point] = per_mount.get(key_point, 0) + 1
        assert per_mount == {point: n for point, n in live.items() if n}, per_mount

# The walk: block indices in order, counts agreeing, buckets ascending, and the
# inodes of a bucket sorted by (number, point) — the order both sides can make.
blocks = 0
index = 0
last_bucket = -1
last_key = (0, b"")
max_bucket = 0
counts = []
for line in lines:
    if line.startswith("L "):
        assert line.split(" ")[1] == str(index), f"out of order: {line!r}"
        bucket = int(field(line, "bucket"))
        number = int(field(line, "number"))
        point = unhex(field(line, "point"))
        assert 0 <= bucket < 1024, line
        assert number % 1024 == bucket, f"{line!r} is in the wrong bucket"
        if bucket != last_bucket:
            assert bucket > last_bucket, f"the buckets are out of order: {line!r}"
        else:
            assert (number, point) >= last_key, f"a bucket is out of order: {line!r}"
        last_bucket, last_key = bucket, (number, point)
        max_bucket = max(max_bucket, bucket)
        index += 1
    elif line.startswith("E count="):
        count = int(field(line, "count"))
        assert count == index, f"count disagrees with the block: {line!r}"
        # ... and with the references the records above left behind, which is
        # what makes the walk a statement about the table and not just itself.
        assert count == walk_keys[blocks], \
            f"{line!r}: the walk sees {count} inodes, the records left {walk_keys[blocks]}"
        counts.append(count)
        index = 0
        last_bucket = -1
        last_key = (0, b"")
        blocks += 1
assert index == 0, "a walk was left unterminated"
assert blocks == 10, f"{blocks} walks"
assert blocks == len(walk_keys), f"{blocks} walks for {len(walk_keys)} E records"
assert max_bucket == 1023, f"the walk never reached the last bucket: {max_bucket}"

# The table is empty exactly when no inode has a reference: nothing is open at
# the top of the script, nothing is open at the bottom, and in between the walks
# agree with the number of keys the records left behind. The mount list is back
# to 0 references under each of its mounts at the end.
assert counts[0] == 0, "the first walk is not an empty table"
assert counts[-1] == 0, "the last walk is not an empty table"
assert lines[-1].startswith("F ")
last_z = max(i for i, line in enumerate(lines) if line.startswith("Z "))
assert lines[last_z] == "Z count=3", lines[last_z]
tail_s = lines[last_z - 3:last_z]
assert all(line.startswith("S ") for line in tail_s), tail_s
assert all(line.endswith("refcount=0") for line in tail_s), \
    f"the mounts are not back to no references at the end: {tail_s}"

# The callback log is complete, indexed from 0, and last, and it is exactly the
# hook calls the releases and the orphan checks above account for.
at = [i for i, line in enumerate(lines) if line.startswith("F count=")]
assert len(at) == 1, "expected exactly one F count record"
at = at[0]
total = int(field(lines[at], "count"))
tail = lines[at + 1:]
assert len(tail) == total, f"{len(tail)} log records for a count of {total}"
assert total == len(expected_hooks), f"{total} hook calls for {len(expected_hooks)} accounted for"
for i, line in enumerate(tail):
    assert line.startswith(f"F {i} "), f"log record {i} is out of order: {line!r}"
    match = re.fullmatch(r"F \d+ orphaned point=(-|[0-9a-f]+) number=(\d+) in_table=([01])", line)
    assert match, f"malformed log record: {line!r}"
    assert (unhex(match.group(1)), int(match.group(2))) == expected_hooks[i], \
        f"{line!r} is not the call the script made"
    # The inode is out of the table before the hook runs, always: both callers
    # are reporting an inode that nothing holds any more.
    assert match.group(3) == "0", f"{line!r} saw the inode still in the table"
print(f"replayed {blocks} table walks and {total} hook calls")
CHECK

out=tests/fixtures/inode_reference.txt
mv "$tmp" "$out"
echo "wrote $out: $(wc -l <"$out") lines, md5 $(md5sum <"$out" | cut -d' ' -f1)"

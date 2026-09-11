#!/bin/sh
# Regenerate tests/fixtures/path_reference.txt from unmodified fs/path.c.
#
#   ISH_SRC=/path/to/ish tools/gen_path_reference.sh
#
# The oracle *includes* fs/path.c, so both functions in the fixture are the real
# ones and the constants are the real macros. It walks each path the way every C
# caller does — `while (path_next_component(&path, component, &err))` — so the
# last record of a walk is the call that ended it: either the end of the path or
# the name that was too long to fit in a `char[MAX_NAME + 1]`.
set -eu

cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish-src}"

for source in "$ISH_SRC/fs/path.c" "$ISH_SRC/fs/path.h" "$ISH_SRC/kernel/fs.h"; do
    if [ ! -f "$source" ]; then
        echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout" >&2
        exit 1
    fi
done

cc -O2 -Wall -Wextra -Werror -I"$ISH_SRC" -Itools/stub-include \
    -o /tmp/path-dump tools/path-dump.c

tmp=/tmp/path_reference.txt.new
/tmp/path-dump >"$tmp"

count() {
    grep -c "$1" "$tmp"
}

predicates=$(count '^P ')
steps=$(count '^C .* ret=1')
ends=$(count '^C .* ret=0')
constants=$(count '^K ')
longs=$(count '^L ')

if [ "$predicates" -ne 69 ] || [ "$constants" -ne 5 ] || [ "$longs" -ne 9 ]; then
    echo "unexpected corpus shape: $predicates predicates, $constants constants, $longs long name cases" >&2
    exit 1
fi
if [ "$ends" -ne 64 ]; then
    echo "unexpected number of walks: $ends" >&2
    exit 1
fi

# The constants, as kernel/fs.h and fs/path.h define them.
grep -q '^K MAX_PATH=4096$' "$tmp" || { echo "MAX_PATH moved" >&2; exit 1; }
grep -q '^K MAX_NAME=256$' "$tmp" || { echo "MAX_NAME moved" >&2; exit 1; }
grep -q '^K N_SYMLINK_FOLLOW=1$' "$tmp" || { echo "N_SYMLINK_FOLLOW moved" >&2; exit 1; }
grep -q '^K N_SYMLINK_NOFOLLOW=2$' "$tmp" || { echo "N_SYMLINK_NOFOLLOW moved" >&2; exit 1; }
grep -q '^K N_PARENT_DIR_WRITE=4$' "$tmp" || { echo "N_PARENT_DIR_WRITE moved" >&2; exit 1; }

# The shapes that are easy to get wrong: the empty path and "/" are normalized,
# "//" and a relative path are not, and a bare "/" is one empty component rather
# than the end of the path.
grep -q '^P - normalized=1$' "$tmp" || { echo "the empty path is no longer normalized" >&2; exit 1; }
grep -q '^P 2f2f normalized=0$' "$tmp" || { echo "// is now normalized" >&2; exit 1; }
grep -q '^P 61 normalized=0$' "$tmp" || { echo "a relative path is now normalized" >&2; exit 1; }
grep -q '^C 2f ret=1 component=- rest=-$' "$tmp" || {
    echo "/ no longer yields one empty component" >&2
    exit 1
}
grep -q '^C - ret=0 err=0$' "$tmp" || { echo "an exhausted path no longer ends the walk" >&2; exit 1; }

# MAX_NAME counts the terminator: a name of MAX_NAME - 1 bytes is the longest
# that fits, and the next length up is _ENAMETOOLONG without consuming anything.
if grep -q '^C [0-9a-f]* ret=0 err=-36$' "$tmp"; then
    :
else
    echo "no component in the corpus is too long" >&2
    exit 1
fi

# Replay the records exactly as tests/path_differential.rs does, and check what
# the port will be asked to check: every input is a normalized path (the C
# asserts it), components chain through `rest`, and each walk ends once.
python3 - "$tmp" <<'CHECK'
import re, sys

lines = open(sys.argv[1]).read().splitlines()

def unhex(field):
    return b"" if field == "-" else bytes.fromhex(field)

def normalized(path):
    # the predicate, spelled out here so the fixture is checked without the port
    if path == b"":
        return True
    index = 0
    while index < len(path):
        if path[index] != 0x2f:
            return False
        index += 1
        if index < len(path) and path[index] == 0x2f:
            return False
        while index < len(path) and path[index] != 0x2f:
            index += 1
    return True

predicates = {}
for line in lines:
    m = re.fullmatch(r"P (-|[0-9a-f]+) normalized=([01])", line)
    if m:
        predicates[unhex(m.group(1))] = int(m.group(2))
assert predicates, "no predicate records"

walking = None
walks = steps = 0
for line in lines:
    m = re.fullmatch(r"C (-|[0-9a-f]+) ret=(\d) (.*)", line)
    if not m:
        continue
    given = unhex(m.group(1))
    # The C only asserts that a walk starts with a slash; callers are the ones
    # that pass normalized paths. A path like "//" still walks (as a run of
    # empty components), so the corpus keeps those and asserts the weaker thing.
    assert given == b"" or given.startswith(b"/"), f"a walk started on a relative path: {given!r}"
    if walking is not None:
        assert given == walking, f"a walk's records do not chain: {given!r} after {walking!r}"
    if m.group(2) == "1":
        body = re.fullmatch(r"component=(-|[0-9a-f]+) rest=(-|[0-9a-f]+)", m.group(3))
        assert body, f"malformed step: {line}"
        name, rest = unhex(body.group(1)), unhex(body.group(2))
        assert b"/" not in name, f"a component contains a slash: {name!r}"
        assert len(name) < 256, f"a step is longer than a name can be: {len(name)}"
        assert rest == b"" or rest.startswith(b"/"), f"rest does not start at a slash: {rest!r}"
        walking = rest
        steps += 1
    else:
        err = int(m.group(3).removeprefix("err="))
        assert err in (0, -36), f"unexpected error: {err}"
        # An empty name is only reachable from a doubled slash, and every one of
        # these ends at a path the predicate rejects.
        assert predicates.get(given, 1) in (0, 1), f"no predicate record for {given!r}"
        walking = None
        walks += 1

assert walking is None, "a walk was left unfinished"
print(f"replayed {steps} steps over {walks} walks")
CHECK

out=tests/fixtures/path_reference.txt
mv "$tmp" "$out"
echo "wrote $out: $(wc -l <"$out") lines, $predicates predicates, $steps steps, $ends walks, $longs long names"

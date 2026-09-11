#!/bin/sh
# Regenerate tests/fixtures/dev_reference.txt from unmodified iSH C.
#
#   ISH_SRC=/path/to/ish tools/gen_dev_reference.sh
#
# The oracle *includes* fs/dev.h, so `dev_make`, `dev_major` and `dev_minor`
# are the header's own arithmetic, and it prints `dev_real_from_fake` /
# `dev_fake_from_real` next to the host libc's `makedev`/`major`/`minor` that
# those two call, so the port's transcription of the host encoding is checked
# against the host itself rather than against a reading of it.
#
# The corpus is every device number fs/devices.h names, plus the edges of each
# field: the fake encoding splits the minor across bits 0..7 and 20..31 and
# gives the major bits 8..19, so values that cross those boundaries (a minor of
# 0x100000, a major of 0x1000, -1) are where a transcription goes wrong. The
# `X`/`Y` records then give the host's own macros arguments the fake encoding
# cannot produce, so the limbs the two conversions cannot see are still checked
# - by the unit test in src/dev.rs, which can call the private transcription.
set -eu

cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish-src}"

for source in "$ISH_SRC/fs/dev.h" "$ISH_SRC/fs/devices.h" "$ISH_SRC/fs/fd.h"; do
    if [ ! -f "$source" ]; then
        echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout" >&2
        exit 1
    fi
done

cc -O2 -Wall -Wextra -Werror -I"$ISH_SRC" -Itools/stub-include \
    -o /tmp/dev-dump tools/dev-dump.c

tmp=/tmp/dev_reference.txt.new
/tmp/dev-dump >"$tmp"

count() {
    grep -c "$1" "$tmp"
}

sizes=$(count '^S ')
makes=$(count '^M ')
decodes=$(count '^D ')
backs=$(count '^T ')
fakes=$(count '^H ')
reals=$(count '^R ')
constants=$(count '^K ')
makedevs=$(count '^X ')
host_decodes=$(count '^Y ')
entries=$makes

# One M/D/T triple per corpus entry (plus the two H/R records each), and the
# eighteen constants.
if [ "$sizes" -ne 1 ] || [ "$entries" -ne 28 ] || [ "$decodes" -ne "$entries" ] || \
   [ "$backs" -ne "$entries" ] || [ "$fakes" -ne "$entries" ] || [ "$reals" -ne "$entries" ] || \
   [ "$constants" -ne 18 ] || [ "$makedevs" -ne 14 ] || [ "$host_decodes" -ne 28 ]; then
    echo "unexpected corpus shape: $sizes sizes, $makes makes, $decodes decodes, $backs backs, $fakes fakes, $reals reals, $constants constants, $makedevs host makedevs, $host_decodes host decodes" >&2
    exit 1
fi

# The type is the guest's and the guest's is 32 bits.
grep -q '^S dev_t_ size=4$' "$tmp" || { echo "dev_t_ is no longer 32 bits" >&2; exit 1; }

# The encoding of the devices iSH names, spelled out: /dev/null, /dev/zero and
# the tty majors are what every open() in the guest compares against.
grep -q '^M 1 3 dev=0x103$' "$tmp" || { echo "mem major/minor encoding changed" >&2; exit 1; }
grep -q '^M 128 0 dev=0x8000$' "$tmp" || { echo "pseudo-tty master major moved" >&2; exit 1; }
grep -q '^M 240 1 dev=0xf001$' "$tmp" || { echo "dynamic device major moved" >&2; exit 1; }

# The two boundaries the split minor creates: a minor past bit 19 is dropped by
# the mask (0x100000 keeps only the major), and a major past bit 11 leaks into
# the minor's high half. Both are the C's answers, not an interpretation.
grep -q '^M 1 1048576 dev=0x100$' "$tmp" || {
    echo "the minor's high half is no longer dropped at bit 20" >&2
    exit 1
}
grep -q '^M 4096 0 dev=0x100000$' "$tmp" || {
    echo "the major no longer leaks into the minor's high half" >&2
    exit 1
}

# Every fake number the corpus builds decodes back to itself: the encoding is a
# bijection on 32 bits, which is why a `dev_t_` can be passed around as an
# opaque value.
python3 - "$tmp" <<'CHECK'
import re, sys
lines = open(sys.argv[1]).read().splitlines()
makes = {}
for line in lines:
    m = re.match(r'^M (-?\d+) (-?\d+) dev=(0x[0-9a-f]+|\d+)$', line)
    if m:
        makes[(int(m.group(1)), int(m.group(2)))] = int(m.group(3), 0)
backs = {int(m.group(1), 0): int(m.group(2), 0)
         for m in (re.match(r'^T (0x[0-9a-f]+|\d+) back=(0x[0-9a-f]+|\d+)$', line) for line in lines)
         if m}
assert backs, "no round-trip records"
for (major, minor), dev in makes.items():
    assert backs[dev] == dev, f"dev_make({major}, {minor}) does not decode back to itself"
print(f"checked {len(backs)} round-trips")
CHECK

# The host macros with arguments of their own: a major wider than twelve bits,
# which the conversion into the fake encoding would otherwise hide, and the
# device number whose major's bits 12..15 would leak through a wider mask.
grep -q '^X 4096 0 makedev=0x100000000000$' "$tmp" || {
    echo "the host's major no longer reaches bit 32" >&2
    exit 1
}
grep -q '^Y 0xf00000 major=0 minor=3840$' "$tmp" || {
    echo "the host major mask changed width" >&2
    exit 1
}

# The host encoding: a fake number the host cannot represent comes back
# truncated, and that is the C's behaviour too.
grep -q '^R 1 1048576 real=0x100000100 fake=0x100$' "$tmp" || {
    echo "the host's minor is no longer wider than the fake one" >&2
    exit 1
}
grep -q '^H 1 3 fake=0x103 real=0x103$' "$tmp" || {
    echo "/dev/null no longer survives the trip through the host encoding" >&2
    exit 1
}

out=tests/fixtures/dev_reference.txt
mv "$tmp" "$out"
echo "wrote $out: $(wc -l <"$out") lines, $entries encodings, $constants constants"

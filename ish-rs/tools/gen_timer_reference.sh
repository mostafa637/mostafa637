#!/bin/sh
# Regenerate tests/fixtures/timer_reference.txt from unmodified iSH C.
#
#   ISH_SRC=/path/to/ish tools/gen_timer_reference.sh
#
# The harness links the untouched util/timer.c and runs its real worker thread
# on a scripted platform: the clock only moves when a scripted sleep completes,
# sleeps are released or poked by the driver, and `free` is wrapped so the
# transcript says whether the caller or the worker freed the timer. The corpus
# is driven through a handshake, so the transcript is identical on every run.
set -eu

cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish-src}"

for source in "$ISH_SRC/util/timer.c" "$ISH_SRC/util/timer.h"; do
    if [ ! -f "$source" ]; then
        echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout" >&2
        exit 1
    fi
done

cc -O2 -Wall -Wextra -I"$ISH_SRC" -Itools/stub-include -o /tmp/timer-dump \
    tools/timer-dump.c "$ISH_SRC/util/timer.c" -pthread \
    -Wl,--wrap=clock_gettime -Wl,--wrap=nanosleep -Wl,--wrap=free

tmp=/tmp/timer_reference.txt.new
/tmp/timer-dump >"$tmp"

count() {
    grep -c "^$1" "$tmp"
}

news=$(count 'N ')
sets=$(count 'T ')
states=$(count 'X ')
sleeps=$(count 'S#')
fires=$(count 'K ')
frees=$(count 'F ')
poke_free=$(grep -c '^F thread' "$tmp" || true)
caller_free=$(grep -c '^F caller' "$tmp" || true)
probes=$(count 'D ')
summaries=$(count 'Z ')

if [ "$news" -ne 4 ] || [ "$sets" -ne 7 ] || [ "$states" -ne 10 ]; then
    echo "unexpected corpus shape: $news timers, $sets sets, $states states" >&2
    exit 1
fi
if [ "$sleeps" -lt 6 ] || [ "$fires" -ne 3 ] || [ "$frees" -ne 4 ]; then
    echo "unexpected activity: $sleeps sleeps, $fires callbacks, $frees frees" >&2
    exit 1
fi
if [ "$poke_free" -ne 1 ] || [ "$caller_free" -ne 3 ]; then
    echo "the corpus must free three times inline and once from the worker" >&2
    exit 1
fi
if [ "$probes" -ne 1 ] || [ "$summaries" -ne 1 ]; then
    echo "missing probe or summary record" >&2
    exit 1
fi

# The corpus must keep covering: a set that starts no thread, the interval
# re-arm, a poke that reschedules a sleeping worker, deactivation without a
# free, a free from inside the sleep, and the single-subtract carry.
grep -q '^T zero rc=0 old=NULL' "$tmp" || { echo "missing the fresh-timer set" >&2; exit 1; }
grep -q '^X zero .* active=0 running=0 dead=0' "$tmp" || {
    echo "a zero-value set must not start a thread" >&2
    exit 1
}
grep -q '^T arm rc=0 old_value=0.000000000 old_interval=0.000000000' "$tmp" || {
    echo "missing the arm set" >&2
    exit 1
}
grep -q '^S#2 0.250000000' "$tmp" || { echo "missing the interval re-arm" >&2; exit 1; }
grep -q '^T requeue rc=0 old_value=0.250000000 old_interval=0.250000000' "$tmp" || {
    echo "missing the reschedule's old spec" >&2
    exit 1
}
grep -q '^S#3 2.000000000' "$tmp" || {
    echo "a poke must make the worker sleep for the new value" >&2
    exit 1
}
grep -q '^X stop .* active=0 running=0 dead=0' "$tmp" || {
    echo "missing the deactivate-without-free path" >&2
    exit 1
}
grep -q '^D free_while_sleeping in_sleep=1' "$tmp" || { echo "missing the mid-sleep free" >&2; exit 1; }
grep -q '^F thread' "$tmp" || { echo "the worker must free the timer itself" >&2; exit 1; }
grep -q '^S#6 1.500000000' "$tmp" || { echo "missing the unnormalized remaining" >&2; exit 1; }
grep -q '^X carry .* end=103.1000000000' "$tmp" || {
    echo "timespec_add must carry once and leave the value unnormalized" >&2
    exit 1
}
# The clock only moves when a sleep completes: every read before the first
# release must still be the epoch the harness started from.
grep -q '^C 100.000000000' "$tmp" || { echo "the clock moved without a sleep" >&2; exit 1; }

out=tests/fixtures/timer_reference.txt
mv "$tmp" "$out"
echo "wrote $out: $(wc -l <"$out") lines, $sets sets, $sleeps sleeps, $fires callbacks, $frees frees"

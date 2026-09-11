"""Execute the book's interpreter transcripts without the Calepin CLI.

The book sources carry every interpreter exchange as ordinary Typst blocks:

    #snippet(```python ... ```)   session code, rendered but silent
    #output(```python ... ```)    session code whose *print output* is shown

This script walks the book's `#include` graph in document order, replays the
blocks against one persistent Python session — the exact semantics of the
Calepin runtime this project originally targeted (`lib/code.typ`'s
`transcript-source`: exec under a shared namespace, capture stdout for
`#output` blocks, swallow and continue on errors) — and writes the captured
outputs into `.calepin/calepin.typ` as a drop-in runtime module:

    #let store  = (get: (key, default: "") => ...)
    #let chunk(.._) = none
    #let setup(.._) = none

`lib/code.typ` imports that module and renders each `#output` through
listings.typ, so plain `typst compile book-ar.typ` shows real interpreter
responses. The book's own environment primitives (pair/head/tail/llist/…)
are preloaded exactly as the text defines them.

Usage:
    python3 tools/execute_transcripts.py [book.typ] [--out PATH]

Defaults: both books, writing .calepin/calepin.typ (one book at a time;
run again before compiling the other book).
"""
import io
import json
import math
import re
import sys
from contextlib import redirect_stdout
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent  # sicp-typst/
RUNTIME = ROOT / ".calepin" / "calepin.typ"

BLOCK = re.compile(
    r"#(snippet|output)\(```python\n(.*?)``` \)|#(snippet|output)\(```python\n(.*?)```\)",
    re.S,
)
INCLUDE = re.compile(r'#include "([^"]+)"')


# ── The book's Python environment (primitives the text itself declares) ─────
class Pair:
    """Mutable cons cell: the text's set_head/set_tail mutate in place
    (3.3.1 aliasing, 3.3.2 queues, 3.3.4 agenda all depend on this)."""

    __slots__ = ("head", "tail")

    def __init__(self, head, tail):
        self.head = head
        self.tail = tail

    def __repr__(self):
        return _llist_repr(self)


def _pair(a, b):
    return Pair(a, b)


def _head(p):
    return p.head


def _tail(p):
    return p.tail


def _set_head(p, v):
    p.head = v


def _set_tail(p, v):
    p.tail = v


def _is_pair(x):
    return isinstance(x, Pair)


def _llist(*items):
    result = None
    for item in reversed(items):
        result = Pair(item, result)
    return result


def _is_null(x):
    return x is None


def _is_llist(x):
    return x is None or _is_pair(x)


def _llist_ref(items, n):
    while n > 0:
        items, n = _tail(items), n - 1
    return _head(items)


def _llist_map(f, items):
    return _llist(*(f(x) for x in _iter_llist(items)))


def _iter_llist(items):
    while items is not None:
        yield _head(items)
        items = _tail(items)


def _llist_repr(x):
    if _is_pair(x):
        return "[" + _llist_repr(_head(x)) + ", " + _llist_repr(_tail(x)) + "]"
    if x is None:
        return "null"
    return str(x)


def _print_llist(items):
    print(_llist_repr(items))


def _for_each(f, items):
    for x in _iter_llist(items):
        f(x)


def _error(*args):
    raise RuntimeError(" ".join(str(a) for a in args))


def _display(x):
    # str(x) would fall back to object repr for Pair — render book-style.
    # Newline-terminated: every printed edition stacks stream displays one
    # term per line (JS edition p.329), and the only transcripts whose
    # captured output flows through display are the stream displays.
    print(str(_show(x)))
    return x


def _newline():
    print()


def _llist_length(items):
    n = 0
    for _ in _iter_llist(items):
        n += 1
    return n


def _llist_append(xs, ys):
    return _llist(*[*_iter_llist(xs), *_iter_llist(ys)])


def _llist_reverse(items):
    return _llist(*reversed(list(_iter_llist(items))))


def _last_pair(items):
    while _is_pair(_tail(items)):
        items = _tail(items)
    return items


def _llist_filter(pred, items):
    return _llist(*(x for x in _iter_llist(items) if pred(x)))


def _llist_reduce(f, init, items):
    acc = init
    for x in _iter_llist(items):
        acc = f(acc, x)
    return acc


def _is_number(x):
    return isinstance(x, (int, float)) and not isinstance(x, bool)


def _is_string(x):
    return isinstance(x, str)


def _is_boolean(x):
    return isinstance(x, bool)


def _is_function(x):
    return callable(x)


def _is_undefined(x):
    return x is None


def _show(x):
    # The book language has no bare tuples: every tuple is a pair chain, and
    # null is the empty list terminator. Sessions print in the same style the
    # text's own print_llist uses.
    if _is_pair(x):
        return _llist_repr(x)
    if x is None:
        return "null"
    return x  # print() applies str() itself


def _session_print(*args, **kwargs):
    import builtins
    builtins.print(*[_show(a) for a in args], **kwargs)


def book_prelude():
    ns = {
        "pair": _pair, "head": _head, "tail": _tail, "is_pair": _is_pair,
        "llist": _llist, "is_null": _is_null, "is_llist": _is_llist,
        "llist_ref": _llist_ref, "llist_map": _llist_map,
        "print_llist": _print_llist, "for_each": _for_each,
        "error": _error,
        "is_none": _is_null, "is_undefined": _is_undefined,
        "is_number": _is_number, "is_string": _is_string,
        "is_boolean": _is_boolean, "is_function": _is_function,
        "length": _llist_length, "append": _llist_append,
        "reverse": _llist_reverse, "last_pair": _last_pair,
        "filter": _llist_filter, "reduce": _llist_reduce,
        "display": _display, "newline": _newline,
        "print": _session_print,
        "set_head": _set_head, "set_tail": _set_tail,
        "random_init": 17,
    }
    for name in ("sin", "cos", "tan", "atan", "atan2", "log", "pow",
                 "sqrt", "floor", "ceil", "exp"):
        ns["math_" + name] = getattr(math, name)
    return ns


# ── Document-order walk of the include graph ────────────────────────────────
def included_files(book: Path):
    seen = [book.resolve()]
    def walk(f: Path):
        for m in INCLUDE.finditer(f.read_text()):
            t = (f.parent / m.group(1)).resolve()
            if t not in seen:
                seen.append(t)
                walk(t)
    walk(book)
    return seen


def blocks_in_order(book: Path):
    for f in included_files(book):
        text = f.read_text()
        for m in BLOCK.finditer(text):
            kind = m.group(1) or m.group(3)
            code = m.group(2) if m.group(1) else m.group(4)
            yield kind, code


def _exec_block(code: str, ns: dict, seconds: float = 3.0):
    """Exec one transcript block under a wall-clock watchdog.

    A runaway cell (e.g. an accidental cycle reachable through the mutable
    Pair structure) must fail the way a timed-out calepin cell would, not
    hang the whole build.
    """
    import signal

    def _on_alarm(signum, frame):
        raise TimeoutError(f"transcript block exceeded {seconds}s")

    old = signal.signal(signal.SIGALRM, _on_alarm)
    signal.setitimer(signal.ITIMER_REAL, seconds)
    try:
        exec(compile(code, "<transcript>", "exec"), ns)
    finally:
        signal.setitimer(signal.ITIMER_REAL, 0.0)
        signal.signal(signal.SIGALRM, old)


# Every edition renders an infinite stream's display as 5 terms then an
# ellipsis line (JS edition p.329); honest finite cells never exceed 4 lines.
# Anything beyond this is an unbounded stream display, so cut the same way.
MAX_LINES = 5


def run_book(book: Path):
    # Deep-but-bounded book recursion (solve forces ~1000 stream cells)
    # needs headroom over the default 1000; the per-block watchdog above
    # bounds anything unbounded.
    import sys as _sys
    _sys.setrecursionlimit(3000)
    ns = book_prelude()
    prelude_names = dict(ns)  # restored after every file (see note)
    outputs = {}
    occurrences = {}  # output code -> how many identical cells came before
    pending = None  # stdout of the most recent snippet, claimed by its output cell
    last_value = ""  # consecutive output cells (2.2.1) replay the same value
    for f in included_files(book):
        text = f.read_text()
        for m in BLOCK.finditer(text):
            kind = m.group(1) or m.group(3)
            code = m.group(2) if m.group(1) else m.group(4)
            if kind == "output":
                # An output cell is only a marker: 226/227 of them verbatim-echo
                # the preceding snippet. It must NOT be re-executed — mutating
                # calls (insert_queue, set_tail, …) would run twice and corrupt
                # the session. The value is the snippet's captured stdout.
                out = last_value if pending is None else pending
                lines = out.split("\n")
                if len(lines) > MAX_LINES + 1:  # trailing "" from the last \n
                    out = "\n".join(lines[:MAX_LINES]) + "\n..."
                # Key = the cell's TRIMMED code + occurrence index among
                # identical codes — mirrors lib/code.typ's _cell-key exactly
                # (the ```-block form and this raw source disagree about
                # trailing newlines), so layout position never enters it.
                key_code = code.strip()
                k = occurrences.get(key_code, 0)
                occurrences[key_code] = k + 1
                key = json.dumps(key_code, ensure_ascii=False) + "#" + str(k)
                outputs[key] = out
                last_value = out
                pending = None
            else:
                buf = io.StringIO()
                try:
                    with redirect_stdout(buf):
                        _exec_block(code, ns)
                except Exception:  # noqa: BLE001 — matches transcript-source: pass
                    pass
                pending = buf.getvalue()
        # The book's environment primitives always mean the book's
        # primitives: exercise-local redefinitions (e.g. the functional
        # pair/head/tail of 2.1.3) must not leak into later files, while
        # ordinary session state (integers, accounts, …) carries on.
        for k, v in prelude_names.items():
            ns[k] = v
    return outputs


def write_runtime(outputs: dict, path: Path = RUNTIME):
    entries = "\n".join(
        f"  {json.dumps(k, ensure_ascii=False)}: {json.dumps(v, ensure_ascii=False)}," for k, v in outputs.items()
    )
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(
        "// Generated by tools/execute_transcripts.py — do not edit.\n"
        "// Drop-in Calepin runtime: captured transcript outputs for one book.\n"
        "#let _outputs = (:" + ("\n" + entries if entries else "") + ")\n"
        '#let _get(key, default: "") = _outputs.at(key, default: default)\n'
        "#let store = (get: _get)\n"
        "#let chunk(.._args) = none\n"
        "#let setup(.._args) = none\n"
        "#let document(.._args) = none\n"
        "#let results(.._args) = none\n"
    )


def main() -> int:
    argv = sys.argv[1:]
    out = RUNTIME
    if "--out" in argv:
        i = argv.index("--out")
        out = Path(argv[i + 1])
        argv = argv[:i] + argv[i + 2:]
    books = [Path(a) for a in argv] or [ROOT / "book.typ", ROOT / "book-ar.typ"]
    for book in books:
        outputs = run_book(book)
        filled = sum(1 for v in outputs.values() if v.strip())
        write_runtime(outputs, out)
        print(f"{book.name}: {len(outputs)} transcripts, {filled} with output -> {out}")
    return 0


if __name__ == "__main__":
    sys.exit(main())

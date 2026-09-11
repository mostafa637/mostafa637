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
def _pair(a, b):
    return (a, b)


def _head(p):
    return p[0]


def _tail(p):
    return p[1]


def _is_pair(x):
    return isinstance(x, tuple) and len(x) == 2


def _llist(*items):
    result = None
    for item in reversed(items):
        result = (item, result)
    return result


def _is_null(x):
    return x is None


def _is_llist(x):
    return x is None or _is_pair(x)


def _llist_ref(items, n):
    while n > 0:
        items, n = items[1], n - 1
    return items[0]


def _llist_map(f, items):
    return _llist(*(f(x) for x in _iter_llist(items)))


def _iter_llist(items):
    while items is not None:
        yield items[0]
        items = items[1]


def _llist_repr(x):
    if _is_pair(x):
        return "[" + _llist_repr(x[0]) + ", " + _llist_repr(x[1]) + "]"
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
    print(str(x), end="")
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
    while _is_pair(items[1]):
        items = items[1]
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


def run_book(book: Path):
    ns = book_prelude()
    prelude_names = dict(ns)  # restored after every file (see note)
    outputs = {}
    n_out = 0
    for f in included_files(book):
        text = f.read_text()
        for m in BLOCK.finditer(text):
            kind = m.group(1) or m.group(3)
            code = m.group(2) if m.group(1) else m.group(4)
            buf = io.StringIO()
            try:
                with redirect_stdout(buf):
                    exec(compile(code, "<transcript>", "exec"), ns)
            except Exception:  # noqa: BLE001 — matches transcript-source: pass
                pass
            if kind == "output":
                outputs[f"_sicp_t{n_out}"] = buf.getvalue()
                n_out += 1
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

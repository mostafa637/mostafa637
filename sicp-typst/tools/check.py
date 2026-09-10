#!/usr/bin/env python3
"""Compile every generated file on its own and report the failures.

Compiling the whole book stops at the first error, which makes fixing a batch
of problems slow. This drives each content file through Typst independently,
so one run lists every file that still needs attention.

Usage:
    python3 tools/check.py            # check every file under content/
    python3 tools/check.py chapter1   # check a subtree
"""

from __future__ import annotations

import re
import sys
import tempfile
from pathlib import Path

try:
    import typst
except ImportError:  # pragma: no cover
    print("This script needs the `typst` package: pip install typst",
          file=sys.stderr)
    raise SystemExit(2)

ROOT = Path(__file__).resolve().parent.parent


def labels_in(path: Path) -> set[str]:
    """The labels one file defines."""
    text = path.read_text(encoding="utf-8")
    return (
        set(re.findall(r"label-name: <([^>]+)>", text))
        | set(re.findall(r"#anchor\(<([^>]+)>\)", text))
        | set(re.findall(r"\]<([^>]+)>", text))
    )


def includes_of(path: Path) -> list[Path]:
    """The files a file pulls in with #include, resolved to real paths."""
    text = path.read_text(encoding="utf-8")
    out = []
    for rel in re.findall(r'#include\s+"([^"]+)"', text):
        target = (ROOT / rel.lstrip("/")) if rel.startswith("/") else (
            path.parent / rel)
        target = target.resolve()
        if target.exists():
            out.append(target)
    return out


def reachable_labels(path: Path) -> set[str]:
    """Labels defined by a file *and everything it includes*.

    A chapter includes its sections, so those labels are already in the
    document; declaring stubs for them would collide.
    """
    seen: set[Path] = set()
    found: set[str] = set()
    stack = [path.resolve()]
    while stack:
        cur = stack.pop()
        if cur in seen:
            continue
        seen.add(cur)
        found |= labels_in(cur)
        stack.extend(includes_of(cur))
    return found


def all_labels() -> set[str]:
    """Every label defined anywhere in the generated book."""
    found: set[str] = set()
    for path in (ROOT / "content").rglob("*.typ"):
        found |= labels_in(path)
    return found


def check(path: Path, labels: set[str], tmp: Path) -> str | None:
    """Compile one file on its own; return the error message, or None.

    A chapter file references labels defined in other chapters, which would
    fail when it is compiled alone. The file is therefore wrapped in a stub
    that declares every label the book defines, so only genuine problems in
    this file are reported.
    """
    own = reachable_labels(path)

    rel = path.relative_to(ROOT)
    stub = tmp / "stub.typ"
    # Stub targets are headings: `@ref` needs a referenceable element, and a
    # box is not one.
    decls = "\n".join(
        f"#heading(level: 6, outlined: false)[stub]<{name}>"
        for name in sorted(labels - own)
    )
    stub.write_text(
        f'#import "/lib/sicp.typ": *\n'
        f"#show: sicp-book\n"
        f'#include "/{rel.as_posix()}"\n'
        f"#set heading(numbering: \"1.1.1\")\n"
        f"= Label stubs\n{decls}\n",
        encoding="utf-8",
    )
    try:
        typst.compile(str(stub), output="/dev/null", root=str(ROOT))
        return None
    except Exception as exc:  # typst.TypstError
        return str(exc).strip()


def main() -> int:
    pattern = sys.argv[1] if len(sys.argv) > 1 else ""
    files = sorted(
        p for p in (ROOT / "content").rglob("*.typ") if pattern in str(p)
    )
    if not files:
        print(f"no files matching {pattern!r}", file=sys.stderr)
        return 1

    labels = all_labels()
    failures = 0
    tmpdir = Path(tempfile.mkdtemp(prefix="sicp-check-", dir=ROOT))
    for path in files:
        err = check(path, labels, tmpdir)
        if err:
            failures += 1
            rel = path.relative_to(ROOT)
            print(f"\n=== {rel}")
            print("   ", err.replace("\n", "\n    "))

    import shutil
    shutil.rmtree(tmpdir, ignore_errors=True)

    ok = len(files) - failures
    print(f"\n{ok}/{len(files)} files compile")
    return 1 if failures else 0


if __name__ == "__main__":
    raise SystemExit(main())

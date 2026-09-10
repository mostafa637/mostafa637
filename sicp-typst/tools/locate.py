#!/usr/bin/env python3
"""Find the line in a generated file that Typst rejects.

The Typst Python binding reports the message but not the position, so this
bisects: it compiles a growing prefix of the file until the error appears.

Usage:
    python3 tools/locate.py content/chapter5/section5/subsection1.typ
"""

from __future__ import annotations

import re
import sys
import tempfile
from pathlib import Path

import typst

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

from tools.check import all_labels  # noqa: E402


def compiles(text: str, tmp: Path, labels: str) -> str | None:
    stub = tmp / "frag.typ"
    stub.write_text(
        '#import "/lib/sicp.typ": *\n#show: sicp-book\n'
        + text
        + "\n"
        + labels,
        encoding="utf-8",
    )
    try:
        typst.compile(str(stub), output="/dev/null", root=str(ROOT))
        return None
    except Exception as exc:
        return str(exc).strip()


def main() -> int:
    path = ROOT / sys.argv[1]
    lines = path.read_text(encoding="utf-8").split("\n")
    # Drop the generated import header; the stub provides its own.
    body = [l for l in lines if not l.startswith("#import ")]

    known = all_labels()
    own = set(re.findall(r"label-name: <([^>]+)>|#anchor\(<([^>]+)>\)|\]<([^>]+)>",
                         "\n".join(body)))
    own = {x for tup in own for x in tup if x}
    labels = "\n".join(
        f"#heading(level: 6, outlined: false)[s]<{n}>"
        for n in sorted(known - own)
    )

    with tempfile.TemporaryDirectory(dir=ROOT) as td:
        tmp = Path(td)
        full = compiles("\n".join(body), tmp, labels)
        if full is None:
            print("file compiles cleanly")
            return 0
        print(f"error: {full}\n")

        # Grow the prefix until the error first appears.
        lo, hi = 0, len(body)
        while lo < hi:
            mid = (lo + hi) // 2
            if compiles("\n".join(body[: mid + 1]), tmp, labels) == full:
                hi = mid
            else:
                lo = mid + 1
        # `lo` indexes the offending line within `body`; map back to the file.
        target = body[lo] if lo < len(body) else ""
        for n, line in enumerate(lines, 1):
            if line == target:
                print(f"line {n}: {line}")
                break
        else:
            print(f"near: {target}")
    return 1


if __name__ == "__main__":
    raise SystemExit(main())

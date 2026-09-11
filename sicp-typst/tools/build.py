"""Compile the EN and AR books to PDF.

Usage (from anywhere):
    python3 tools/build.py [output-dir]

Defaults to sicp-typst/dist/. Exits non-zero if any book fails.
Requires: pip install typst==0.15.0  and the calepin 0.1.0 package under
~/.local/share/typst/packages/preview/ (the committed .calepin bootstrap
facade re-exports it, so plain `typst compile` works on a fresh checkout).
"""
import sys
import time
from pathlib import Path

import typst

ROOT = Path(__file__).resolve().parent.parent  # sicp-typst/
BOOKS = [
    ("book.typ", "book-en.pdf"),
    ("book-ar.typ", "book-ar.pdf"),
]


def main() -> int:
    out_dir = Path(sys.argv[1]) if len(sys.argv) > 1 else ROOT / "dist"
    out_dir.mkdir(parents=True, exist_ok=True)
    failed = False
    for src, name in BOOKS:
        out = out_dir / name
        started = time.time()
        try:
            typst.compile(str(ROOT / src), output=str(out))
            size_mb = out.stat().st_size / 1e6
            print(f"OK   {src} -> {out} ({size_mb:.1f} MB, {time.time() - started:.0f}s)")
        except Exception as exc:  # noqa: BLE001 - report and continue
            failed = True
            print(f"FAIL {src}: {str(exc)[:500]}", file=sys.stderr)
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())

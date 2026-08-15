#!/usr/bin/env python3
"""Generate Asbestos offsets from the compiler's preprocessed assembly output."""

from __future__ import annotations

import argparse
import re
import subprocess
from pathlib import Path


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--compiler", required=True)
    parser.add_argument("--source", required=True)
    parser.add_argument("--include-root", required=True)
    parser.add_argument("--staticdefine", required=True)
    parser.add_argument("--output", required=True)
    args = parser.parse_args()

    command = [
        args.compiler,
        "-std=gnu11",
        "-I",
        args.include_root,
        "-include",
        args.staticdefine,
        "-S",
        args.source,
        "-o",
        "-",
    ]
    assembly = subprocess.run(command, check=True, capture_output=True, text=True).stdout
    result: list[str] = []
    for line in assembly.splitlines():
        match = re.search(r'^\s*\.ascii\s+"(.*)"', line)
        if not match:
            continue
        value = match.group(1)
        if not value.startswith("->"):
            continue
        value = value[2:]
        parts = value.split(" ", 2)
        if len(parts) < 2:
            continue
        name, number = parts[0], parts[1].lstrip("$#")
        comment = parts[2] if len(parts) == 3 else ""
        result.append(f"#define {name} {number} /* {comment} */")

    output = Path(args.output)
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text("/* Generated; do not edit. */\n" + "\n".join(result) + "\n", encoding="utf-8")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

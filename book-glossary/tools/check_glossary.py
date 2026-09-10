#!/usr/bin/env python3
"""Check GLOSSARY.md against its own strictness rule.

The rule the glossary states about itself:

  1. every English term maps to one fixed Arabic term;
  2. different English terms do not share an Arabic term;
  3. Job = مهمة, Task = مهمة فرعية;
  4. Errant = شاردة, Runaway = جامحة.

Rules 1 and 2 are mechanical, so they are checked here. Rule 2 has real
exceptions -- a plural and its singular, or a term deliberately repeated in
two chapters -- so a violation is reported, not assumed to be a mistake.

Usage:  python3 tools/check_glossary.py [path/to/GLOSSARY.md]
Exit code is 1 if a genuine conflict is found, 0 otherwise.
"""
from __future__ import annotations

import re
import sys
import unicodedata
from collections import defaultdict
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
DEFAULT = ROOT / "GLOSSARY.md"

# A term line lives inside a fenced block: English, 2+ spaces, Arabic.
ENTRY_RE = re.compile(r"^(\S.*?)\s{2,}(\S.*)$")
ARABIC_RE = re.compile(r"[\u0600-\u06FF]")


def strip_marks(s: str) -> str:
    """Drop Arabic diacritics and tatweel, so ألفة == ألفة regardless of marks."""
    out = []
    for ch in unicodedata.normalize("NFC", s):
        if unicodedata.category(ch) == "Mn" or ch == "\u0640":
            continue
        out.append(ch)
    return "".join(out)


def normalise(s: str) -> str:
    s = strip_marks(s)
    s = s.replace("\u200f", "").replace("\u200e", "")
    s = re.sub(r"[\u0622\u0623\u0625]", "\u0627", s)  # آ أ إ -> ا
    s = s.replace("\u0649", "\u064a")                  # ى -> ي
    s = s.replace("\u0629", "\u0647")                  # ة -> ه
    return re.sub(r"\s+", " ", s).strip()


def parse(path: Path) -> list[tuple[int, str, str]]:
    """Return (line number, English, Arabic) for every term line."""
    entries: list[tuple[int, str, str]] = []
    in_block = False
    for n, raw in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
        if raw.startswith("```"):
            in_block = not in_block
            continue
        if not in_block:
            continue
        m = ENTRY_RE.match(raw.rstrip())
        if m:
            entries.append((n, m.group(1).strip(), m.group(2).strip()))
    return entries


def main() -> int:
    path = Path(sys.argv[1]) if len(sys.argv) > 1 else DEFAULT
    if not path.exists():
        print(f"not found: {path}")
        return 1

    entries = parse(path)
    print(f"{path.name}: {len(entries)} terms")

    problems = 0

    # Rule 1: one English term, one Arabic term.
    by_en: dict[str, list[tuple[int, str]]] = defaultdict(list)
    for n, en, ar in entries:
        by_en[en.lower()].append((n, ar))
    clashes = {
        en: v for en, v in by_en.items()
        if len({normalise(a) for _, a in v}) > 1
    }
    if clashes:
        problems += len(clashes)
        print(f"\nrule 1 -- one English term with several Arabic terms ({len(clashes)}):")
        for en, v in sorted(clashes.items()):
            print(f"  {en}")
            for n, ar in v:
                print(f"      line {n}: {ar}")

    # Rule 2: one Arabic term, one English term.
    by_ar: dict[str, list[tuple[int, str]]] = defaultdict(list)
    for n, en, ar in entries:
        # An entry that is only Latin (fork(), SIGINT) is a term kept as-is,
        # not a translation, so it cannot collide in the Arabic direction.
        if ARABIC_RE.search(ar):
            by_ar[normalise(ar)].append((n, en))
    shared = {
        ar: v for ar, v in by_ar.items()
        if len({e.lower() for _, e in v}) > 1
    }
    # Two terms that differ *only* by diacritics (معامِل vs مُعامَل) are not the
    # same word, so this is not a rule-2 violation -- but the distinction rests
    # entirely on marks a reader may not see and a search will not match, so it
    # is reported separately as fragile rather than passed in silence.
    hard, fragile = {}, {}
    raw = {(n, en): ar for n, en, ar in entries}
    for ar, v in shared.items():
        forms = {raw[(n, en)] for n, en in v}
        (fragile if len(forms) > 1 else hard)[ar] = v

    if hard:
        problems += len(hard)
        print(f"\nrule 2 -- one Arabic term shared by several English terms ({len(hard)}):")
        for ar, v in sorted(hard.items()):
            print(f"  {ar}")
            for n, en in v:
                print(f"      line {n}: {en}")

    if fragile:
        print(f"\nfragile -- distinguished only by diacritics ({len(fragile)}):")
        for ar, v in sorted(fragile.items()):
            for n, en in v:
                print(f"      line {n}: {en} -> {raw[(n, en)]}")
            print("      (same letters; keep the marks in the text, "
                  "a search for one will find the other)")

    # Rules 3 and 4 are named pairs, so they are checked by name.
    # Rules 3 and 4 are named pairs. A term that is simply absent is not a
    # violation -- TERMS.md is an extension of the main table and does not
    # repeat it -- so only terms that are present are judged.
    pairs = [("Job", "مهمة"), ("Task", "مهمة فرعية"),
             ("Errant Process", "عملية شاردة"), ("Runaway Process", "عملية جامحة")]
    lookup = {en.lower(): ar for _, en, ar in entries}
    present = [(en, want) for en, want in pairs if en.lower() in lookup]
    if present:
        print("\nrules 3 and 4 -- the pairs called out in the glossary:")
        for en, want in present:
            got = lookup[en.lower()]
            ok = normalise(got) == normalise(want)
            if not ok:
                problems += 1
            print(f"  {'ok  ' if ok else 'FAIL'} {en} -> {got!r} (expected {want!r})")
    else:
        print("\nrules 3 and 4 -- none of those terms are in this file, skipped")

    print(f"\n{problems} item(s) to review" if problems else "\nno conflicts")
    return 1 if problems else 0


if __name__ == "__main__":
    raise SystemExit(main())

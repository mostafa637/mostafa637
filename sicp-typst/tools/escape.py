"""Escaping helpers for emitting Typst markup."""

from __future__ import annotations

import re

# Characters that start Typst markup and therefore need a backslash when they
# appear in ordinary prose.
_SPECIAL = set("\\#$*_`<>@[]")


def escape_text(s: str) -> str:
    """Escape a run of plain prose for Typst markup mode."""
    out = []
    for ch in s:
        if ch in _SPECIAL:
            out.append("\\" + ch)
        else:
            out.append(ch)
    text = "".join(out)
    # A line starting with "- ", "+ " or "1. " would become a list item.
    text = re.sub(r"(?m)^(\s*)([-+])(\s)", r"\1\\\2\3", text)
    text = re.sub(r"(?m)^(\s*)(\d+)\.(\s)", r"\1\2\\.\3", text)
    text = re.sub(r"(?m)^(\s*)(=+)(\s)", r"\1\\\2\3", text)
    return text


def escape_string(s: str) -> str:
    """Escape a value for a double-quoted Typst string literal."""
    return s.replace("\\", "\\\\").replace('"', '\\"')


def typst_label(name: str) -> str:
    """Render an XML LABEL/REF name as a Typst label token.

    Typst labels accept letters, digits, `-`, `_` and `:`; the SICP sources
    only ever use those plus a stray `.`, which is mapped to `-`.
    """
    clean = re.sub(r"[^0-9A-Za-z:_-]", "-", name.strip())
    return f"<{clean}>"


def label_target(name: str) -> str:
    """The bare (unbracketed) form of a label, for `@ref` syntax."""
    return re.sub(r"[^0-9A-Za-z:_-]", "-", name.strip())

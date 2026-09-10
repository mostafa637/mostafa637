#!/usr/bin/env python3
"""Convert the SICP Python-edition XML sources into Typst.

Usage:
    python3 tools/convert.py --source /path/to/sicp [--out .]

The upstream repository (https://github.com/source-academy/sicp) keeps the
book as XML in ``xml_py/``, with the prose shared between the Scheme and
Python editions and language-specific parts wrapped in ``<SPLIT>`` /
``<SPLITINLINE>``. This script walks that tree, keeps the Python side, and
writes one ``.typ`` file per source file into ``content/``, mirroring the
upstream directory layout.

The generated files call into the small Typst library in ``lib/``; nothing in
``content/`` needs hand-editing after a run.
"""

from __future__ import annotations

import argparse
import html
import re
import shutil
import sys
import xml.etree.ElementTree as ET
from collections import Counter
from dataclasses import dataclass, field
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

from tools.escape import escape_string, escape_text, label_target, typst_label
from tools.latex_math import (convert_latex_fragment, latex_to_text,
                              latex_to_typst)

# ── the edition being generated ────────────────────────────────────────────

LANG_BLOCK = "PYTHON"
LANG_INLINE = "PYTHONINLINE"
LANG_OUTPUT = "PYTHON_OUTPUT"
LANG_PROMPT = "PYTHON_PROMPT"
LANG_RUN = "PYTHON_RUN"
LANG_TEST = "PYTHON_TEST"
LANG_NAME = "Python"

COMPANION_BLOCK = "SCHEME"
COMPANION_INLINE = "SCHEMEINLINE"
COMPANION_OUTPUT = "SCHEMEOUTPUT"
COMPANION_PROMPT = "SCHEMEPROMPT"

# Tags dropped wholesale: Scheme-only content, build metadata, and the
# web-only variants of passages that the print edition words differently.
DROP_TAGS = {
    COMPANION_BLOCK, COMPANION_OUTPUT, COMPANION_PROMPT,
    "HISTORY", "COMMENT", "EDIT", "EXCLUDE", "FRAGILE", "SOLUTION",
    "WEB_ONLY", "ORDER", "PRIMITIVE", "OPERATOR", "FUNCTION", "PARSING",
    "SEE", "SEEALSO", "OPEN", "CLOSE", "SUBINDEX", "ECMA", "PLR",
    "CHAPTERCONTENT", "SECTIONCONTENT", "NAME",
    LANG_RUN, LANG_TEST, "REQUIRES", "EXAMPLE", "EXPECTED",
}

# Tags that contribute nothing of their own but whose children are kept.
TRANSPARENT = {"SPLIT", "span", LANG_BLOCK}

# Purely typographic LaTeX hints in the sources with no Typst equivalent.
IGNORE_TAGS = {
    "SHRINK_PARAGRAPH", "STRETCH_PARAGRAPH", "DONT_BREAK_PAGE",
    "LONG_PAGE", "SHORT_PAGE", "FILBREAK", "KEEP_TOGETHER",
    "START_KEEP_TOGETHER", "STOP_KEEP_TOGETHER", "WATCH",
    "EXERCISE_FOLLOWED_BY_TEXT", "NOINDENT", "SOFT_HYP",
    "ALLOW_BREAK", "SHORT_SPACE_AND_ALLOW_BREAK", "SHORT_SPACE",
    "FORCE_PAGE_BREAK_AND_FILL",
}

# Character-entity tags. These hold the *literal* character; `raw_code` uses
# them verbatim inside code, while `Converter.convert` escapes them for prose.
SYMBOL_TAGS = {
    "APOS": "'", "AMP": "&", "DOLLAR": "$", "SHARP": "#", "SECT": "§",
    "ELLIPSIS": "…", "EMDASH": "—", "ENDASH": "–",
    "SPACE": "\u00a0", "FIXED_SPACE": "\u00a0", "BREAKINGNONSPACE": "",
    "AACUTE_LOWER": "á", "AACUTE_UPPER": "Á",
    "AGRAVE_LOWER": "à", "AGRAVE_UPPER": "À", "ACIRC_LOWER": "â",
    "EACUTE_LOWER": "é", "EACUTE_UPPER": "É",
    "EGRAVE_LOWER": "è", "EGRAVE_UPPER": "È", "ECIRC_LOWER": "ê",
    "OUML_LOWER": "ö", "OUML_UPPER": "Ö",
    "UUML_LOWER": "ü", "UUML_UPPER": "Ü", "CCEDIL_LOWER": "ç",
    "WJ": "\u2060",
}


@dataclass
class Stats:
    files: int = 0
    unknown_tags: Counter = field(default_factory=Counter)
    unknown_latex: set = field(default_factory=set)
    labels: set = field(default_factory=set)
    refs: set = field(default_factory=set)
    duplicate_labels: Counter = field(default_factory=Counter)


# ── XML loading ────────────────────────────────────────────────────────────

# Entity references like &subsection1.1.1; are include directives; the
# converter resolves them from the file layout instead of expanding them.
# The sources write their includes as `&amp;section4.1;`, which the XML
# parser hands back already decoded to `&section4.1;`.
ENTITY_RE = re.compile(r"&([a-zA-Z][\w.]*);")


def load_xml(path: Path) -> ET.Element:
    """Parse one source file into an element tree.

    The sources are XML fragments rather than documents: several have more
    than one root element, contain SGML-style comments with stray markup, and
    use bare ``&`` in code. They are normalised here and wrapped in a synthetic
    root so a standard parser accepts them.
    """
    text = path.read_text(encoding="utf-8")
    text = re.sub(r"<!--.*?-->", "", text, flags=re.S)
    # Bare ampersands that are not part of a real entity reference.
    text = re.sub(r"&(?!(?:[a-zA-Z][\w.]*|#\d+);)", "&amp;", text)
    return ET.fromstring("<ROOT>" + text + "</ROOT>")


def child(node: ET.Element, tag: str) -> ET.Element | None:
    for c in node:
        if c.tag == tag:
            return c
    return None


def node_text(node: ET.Element | None) -> str:
    """All descendant text of a node, whitespace-collapsed."""
    if node is None:
        return ""
    return re.sub(r"\s+", " ", "".join(node.itertext())).strip()


def raw_code(node: ET.Element) -> str:
    """Verbatim text of a code element, with entity tags substituted."""
    parts: list[str] = []

    def walk(n: ET.Element) -> None:
        if n.text:
            parts.append(n.text)
        for c in n:
            if c.tag in SYMBOL_TAGS:
                parts.append(SYMBOL_TAGS[c.tag])
            elif c.tag in ("META", "METAPHRASE"):
                parts.append(node_text(c))
            elif c.tag in DROP_TAGS or c.tag in IGNORE_TAGS:
                pass
            else:
                walk(c)
            if c.tail:
                parts.append(c.tail)

    walk(node)
    return "".join(parts)


def dedent_code(code: str) -> str:
    """Strip the common indentation the XML nesting adds to code blocks."""
    lines = code.replace("\t", "    ").split("\n")
    while lines and not lines[0].strip():
        lines.pop(0)
    while lines and not lines[-1].strip():
        lines.pop()
    if not lines:
        return ""
    indents = [len(l) - len(l.lstrip()) for l in lines if l.strip()]
    cut = min(indents) if indents else 0
    return "\n".join(l[cut:] if len(l) >= cut else l.lstrip() for l in lines)


# ── converter ──────────────────────────────────────────────────────────────

# Inside a LATEX="yes" snippet, `$...$` is real maths when it contains a
# command, a subscript or a superscript. A bare `$` is a pattern variable in
# the logic-programming chapters and must stay literal.
_SNIPPET_MATH_RE = re.compile(
    r"\$((?:\\.|[^$\n\\])*(?:\\[A-Za-z]+|[_^])(?:\\.|[^$\n\\])*)\$"
)


# `$word$` in one of these snippets is LaTeX for an italic placeholder
# (`rule(meeting_time($person, $day), $rule$-$body$)`). A single `$` glued to
# a name is a pattern variable of the query language and stays literal.
_META_VAR_RE = re.compile(r"\$([A-Za-z][A-Za-z_]*)\$")


def _split_meta_vars(text: str) -> list[str]:
    """Split a snippet literal into strings and `meta(..)` placeholders."""
    out: list[str] = []
    pos = 0
    for m in _META_VAR_RE.finditer(text):
        if m.start() > pos:
            out.append(f'"{escape_string(text[pos:m.start()])}"')
        out.append(f'meta("{escape_string(m.group(1))}")')
        pos = m.end()
    if pos < len(text):
        out.append(f'"{escape_string(text[pos:])}"')
    return out


class Converter:
    def __init__(self, stats: Stats):
        self.stats = stats
        # Set while emitting code, where prose escaping must not apply.
        self.in_code = False

    # -- inline text ------------------------------------------------------

    # Display maths is normally wrapped in <LATEX>, but a couple of places
    # leave a bare `\[ ... \]` sitting in the prose.
    _BARE_DISPLAY_RE = re.compile(r"\\\[(.*?)\\\]", re.S)

    def text(self, s: str | None, *, collapse: bool = True) -> str:
        if not s:
            return ""
        if collapse:
            s = re.sub(r"[ \t]*\n[ \t]*", "\n", s)
            s = re.sub(r"[ \t]{2,}", " ", s)
        # A stray inline styling command in otherwise plain prose. The markers
        # use private-use characters so that escape_text() leaves them alone.
        if re.search(r"\\(?:emph|textit|textbf|texttt|textrm)\{", s):
            s = re.sub(r"\\(?:emph|textit)\{([^{}]*)\}", "\uE000\\1\uE002", s)
            s = re.sub(r"\\textbf\{([^{}]*)\}", "\uE001\\1\uE002", s)
            s = re.sub(r"\\(?:texttt|textrm)\{([^{}]*)\}", r"\1", s)
            return (escape_text(s)
                    .replace("\uE000", "#emph[")
                    .replace("\uE001", "#strong[")
                    .replace("\uE002", "]"))
        if "\\[" in s:
            out, pos = [], 0
            for m in self._BARE_DISPLAY_RE.finditer(s):
                out.append(escape_text(s[pos:m.start()]))
                body = latex_to_typst(m.group(1), self.stats.unknown_latex)
                out.append(f"\n$ {body} $\n")
                pos = m.end()
            out.append(escape_text(s[pos:]))
            return "".join(out)
        return escape_text(s)

    def children(self, node: ET.Element, sep: str = "") -> str:
        """Convert a node's children (and their tails) to Typst markup."""
        parts: list[str] = [self.text(node.text)]
        for c in node:
            parts.append(self.convert(c))
            parts.append(self.text(c.tail))
        return sep.join(p for p in parts if p)

    def convert(self, node: ET.Element) -> str:
        tag = node.tag
        handler = getattr(self, "do_" + tag.lower().replace("-", "_"), None)
        if handler is not None:
            return handler(node)
        if tag in SYMBOL_TAGS:
            # `$`, `#` and friends start markup in Typst and must be escaped
            # when they reach the prose (in code they are handled by raw_code).
            return escape_text(SYMBOL_TAGS[tag])
        if tag in DROP_TAGS:
            return ""
        if tag in IGNORE_TAGS:
            return ""
        if tag in TRANSPARENT:
            return self.children(node)
        self.stats.unknown_tags[tag] += 1
        return self.children(node)

    # -- structure --------------------------------------------------------

    def _claim(self, lab: ET.Element | None) -> str | None:
        """Take ownership of a LABEL, returning its unique Typst name.

        A label may only appear once in a Typst document, but the sources
        repeat a few (the Scheme and Python sides of a split carry the same
        name). The first occurrence wins; later ones are dropped, exactly as
        the LaTeX build's last-one-wins behaviour would resolve them.
        """
        if lab is None or not lab.get("NAME"):
            return None
        # Mark the element itself, so that the generic LABEL handler below
        # knows not to emit it a second time as a stray anchor.
        lab.set("_claimed", "yes")
        name = label_target(lab.get("NAME"))
        if name in self.stats.labels:
            self.stats.duplicate_labels[name] += 1
            return None
        self.stats.labels.add(name)
        return name

    def _label_arg(self, node: ET.Element) -> str:
        name = self._claim(child(node, "LABEL"))
        return f", label-name: <{name}>" if name else ""

    def do_label(self, node):
        """A LABEL not attached to a numbered element becomes an anchor."""
        if node.get("_claimed"):
            return ""
        name = self._claim(node)
        return f"#anchor(<{name}>)" if name else ""

    def _heading(self, node: ET.Element, fn: str) -> str:
        title = self.inline_of(child(node, "NAME"))
        label = self._label_arg(node)  # claim before the body is walked
        body = self.children(node)
        return f'\n#{fn}([{title}]{label})\n\n{body}\n'

    def do_chapter(self, node):
        return self._heading(node, "chapter")

    def do_section(self, node):
        return self._heading(node, "section")

    def do_subsection(self, node):
        return self._heading(node, "subsection")

    def do_subsubsection(self, node):
        return self._heading(node, "subsubsection")

    def do_matter(self, node):
        return self._heading(node, "matter")

    def do_references(self, node):
        return self._heading(node, "matter")

    def do_mattersection(self, node):
        title = self.inline_of(child(node, "NAME"))
        return f"\n#matter-section([{title}])\n\n{self.children(node)}\n"

    def do_subheading(self, node):
        title = self.inline_of(child(node, "NAME"))
        return f"\n#subheading([{title}])\n\n{self.children(node)}\n"

    def do_subsubheading(self, node):
        """<SUBSUBHEADING> — nested inside a SUBHEADING, so one level down."""
        title = self.inline_of(child(node, "NAME"))
        return f"\n#subsubheading([{title}])\n\n{self.children(node)}\n"

    def do_text(self, node):
        body = self.children(node).strip()
        return f"\n\n{body}\n\n" if body else ""

    do_p = do_text

    def inline_of(self, node: ET.Element | None) -> str:
        """Convert a node's children as a single inline run."""
        if node is None:
            return ""
        out = self.children(node)
        return re.sub(r"\s+", " ", out).strip()

    # -- inline markup ----------------------------------------------------

    def do_em(self, node):
        return f"#emph[{self.inline_of(node)}]"

    do_em_no_index = do_em
    do_e = do_em

    def do_b(self, node):
        return f"#strong[{self.inline_of(node)}]"

    def do_tt(self, node):
        return f'#py-plain("{escape_string(node_text(node))}")'

    def do_quote(self, node):
        return f'"{self.inline_of(node)}"'

    def do_br(self, node):
        return "\\\n"

    def do_break(self, node):
        return "\n\n"

    def do_blockquote(self, node):
        return f"\n#blockquote[{self.children(node).strip()}]\n"

    def do_link(self, node):
        addr = node.get("address", "")
        body = self.inline_of(node)
        return f'#link("{escape_string(addr)}")[{body}]'

    def do_splitinline(self, node):
        """Keep the Python side of an inline Scheme/Python split.

        The chosen branch sits on its own indented lines in the sources, and
        that vertical whitespace would read as a paragraph break in Typst —
        mid-sentence. Being inline by definition, the run is collapsed.
        """
        py = child(node, LANG_BLOCK)
        if py is not None:
            return self.inline_of(py)
        return re.sub(r"\s+", " ", "".join(
            self.convert(c) + self.text(c.tail)
            for c in node
            if c.tag not in (COMPANION_BLOCK, COMPANION_INLINE)
        )).strip()

    # -- code -------------------------------------------------------------

    def do_pythoninline(self, node):
        # A container that mixes text with nested inline code (index
        # declarations do this) has to be recursed into instead of quoted.
        if list(node) and not all(
            c.tag in SYMBOL_TAGS or c.tag in DROP_TAGS for c in node
        ):
            inner = "".join(
                self.convert(c) + self.text(c.tail) for c in node
            )
            return (self.text(node.text) + inner).strip()
        code = raw_code(node).strip()
        if not code:
            return ""
        # '@' marks an allowed line-break point in the LaTeX build.
        code = code.replace("_@", "_").replace("@", "")
        return f'#py("{escape_string(code)}")'

    do_schemeinline = do_pythoninline
    do_declaration = do_pythoninline
    do_use = do_pythoninline

    def do_meta(self, node):
        return f'#meta("{escape_string(node_text(node))}")'

    def do_metaphrase(self, node):
        return f"#metaphrase[{self.inline_of(node)}]"

    def do_latexinline(self, node):
        src = raw_code(node)
        return convert_latex_fragment(src, self.stats.unknown_latex).strip()

    def do_latex(self, node):
        src = raw_code(node)
        body = convert_latex_fragment(src, self.stats.unknown_latex).strip()
        return f"\n{body}\n"

    def do_snippet(self, node):
        if node.get("HIDE") == "yes":
            return ""
        out: list[str] = []
        for idx_node in node.findall("INDEX"):
            out.append(self.convert(idx_node))

        prompt = node.find(LANG_PROMPT)
        if prompt is not None:
            code = dedent_code(raw_code(prompt))
            if code:
                out.append(f"\n#prompt({self._raw_literal(code)})\n")

        code_node = node.find(LANG_BLOCK)
        if code_node is not None:
            if node.get("LATEX") == "yes" or code_node.findall(".//META"):
                out.append(self._syntax_snippet(code_node))
            else:
                code = dedent_code(raw_code(code_node))
                if code:
                    out.append(f"\n#snippet({self._raw_literal(code)})\n")

        outp = node.find(LANG_OUTPUT)
        if outp is not None:
            code = dedent_code(raw_code(outp))
            if code:
                out.append(f"\n#output({self._raw_literal(code)})\n")
        return "".join(out)

    def _raw_literal(self, code: str) -> str:
        """Emit code as a Typst raw block, choosing a safe fence."""
        fence = "```"
        while fence in code:
            fence += "`"
        # A raw block's content must not start on the fence line for the
        # language tag to work, and must not end with a backtick.
        if code.endswith("`"):
            code += "\n"
        return f"{fence}python\n{code}\n{fence}"

    def _syntax_snippet(self, code_node: ET.Element) -> str:
        """A LATEX="yes" snippet: literal code plus italic meta-variables."""
        parts: list[str] = []
        buf: list[str] = []

        def flush() -> None:
            if not buf:
                return
            joined = "".join(buf)
            buf.clear()
            if not joined:
                return
            # These snippets are literal text, but a few embed real maths in
            # a comment (`# $\\texttt{n}$`). Lift those out: the literal
            # becomes a Typst string, where maths would not be rendered.
            # Only `$...$` containing a command is maths — a bare `$` is a
            # pattern variable in the logic-programming chapters.
            for i, part in enumerate(re.split(_SNIPPET_MATH_RE, joined)):
                if not part:
                    continue
                if i % 2:
                    parts.append("$" + latex_to_typst(part) + "$")
                else:
                    parts.extend(_split_meta_vars(part))

        def walk(n: ET.Element) -> None:
            if n.text:
                buf.append(n.text)
            for c in n:
                if c.tag == "META":
                    flush()
                    parts.append(f'meta("{escape_string(node_text(c))}")')
                elif c.tag == "METAPHRASE":
                    flush()
                    parts.append(f"metaphrase[{self.inline_of(c)}]")
                elif c.tag in SYMBOL_TAGS:
                    buf.append(SYMBOL_TAGS[c.tag])
                elif c.tag in DROP_TAGS or c.tag in IGNORE_TAGS:
                    pass
                else:
                    walk(c)
                if c.tail:
                    buf.append(c.tail)

        walk(code_node)
        flush()

        # Normalise the indentation that XML nesting introduced.
        text_parts = []
        for p in parts:
            text_parts.append(p)
        joined = ", ".join(text_parts)
        # Trim leading/trailing whitespace-only literals.
        joined = re.sub(r'^"[\s\\n]*",\s*', "", joined)
        joined = re.sub(r',\s*"[\s\\n]*"$', "", joined)
        if not joined:
            return ""
        return f"\n#syntax({self._clean_syntax_args(joined)})\n"

    @staticmethod
    def _clean_syntax_args(args: str) -> str:
        # Collapse the newline+indent runs the XML layout inserts.
        return re.sub(r"\\n\s+", r"\\n", args)

    # -- lists, tables, figures -------------------------------------------

    def _list(self, node: ET.Element, marker: str) -> str:
        """Convert UL/OL, indenting any list nested inside an item.

        An item is *not* collapsed to a single line: the sources nest a list
        inside an <LI> five times, and squeezing the whitespace out would put
        the inner markers mid-line, where Typst reads them as literal `+` or
        `-` text instead of a sub-list. Instead each item keeps its own lines,
        and continuation lines are indented under the marker, which is how
        Typst marks a nested list as belonging to the item above it.
        """
        pad = " " * (len(marker) + 1)
        items: list[str] = []
        for li in node.findall("LI"):
            # The item's own prose is inline: the sources wrap it over several
            # lines, and that vertical space would read as a paragraph break.
            # A nested list is the exception -- it must keep its own lines.
            blocks: list[str] = []
            run: list[str] = [self.text(li.text)]

            def flush() -> None:
                prose = re.sub(r"\s+", " ", "".join(run)).strip()
                if prose:
                    blocks.append(prose)
                run.clear()

            for c in li:
                if c.tag in ("UL", "OL"):
                    flush()
                    blocks.append(self.convert(c).strip("\n"))
                else:
                    run.append(self.convert(c))
                run.append(self.text(c.tail))
            flush()

            if not blocks:
                continue
            body = "\n\n".join(blocks)
            lines = [ln.rstrip() for ln in body.split("\n")]
            out = [f"{marker} {lines[0]}"]
            # Continuation lines are indented under the marker, which is what
            # attaches them (and a nested list) to the item above.
            out += [pad + ln if ln else "" for ln in lines[1:]]
            items.append("\n".join(out))
        return "\n" + "\n".join(items) + "\n\n"

    def do_ul(self, node):
        return self._list(node, "-")

    def do_ol(self, node):
        return self._list(node, "+")

    def do_li(self, node):
        return self.children(node)

    def do_table(self, node):
        rows = node.findall("TR")
        if not rows:
            return self.children(node)
        width = max(len(r.findall("TD")) for r in rows)
        cells: list[str] = []
        for r in rows:
            tds = r.findall("TD")
            for td in tds:
                cells.append(f"[{self.inline_of(td)}]")
            cells.extend(["[]"] * (width - len(tds)))
        joined = ", ".join(cells)
        return f"\n#sicp-table(columns: {width}, {joined})\n"

    def do_figure(self, node):
        # Claimed up front so the label is not also emitted as a stray anchor
        # while the figure's contents are converted.
        fig_label = self._claim(node.find("LABEL"))
        src = node.get("src")
        if src is None:
            inner = node.find("FIGURE")
            if inner is not None:
                src = inner.get("src")

        body_parts: list[str] = []
        if src:
            body_parts.append(self._image(src, node))
        else:
            images = node.findall(".//IMAGE")
            if images:
                stack = ", ".join(
                    self._image(im.get("src", ""), node) for im in images
                )
                body_parts.append(f"stack(dir: ttb, spacing: 1em, {stack})")

        for snip in node.findall(".//SNIPPET"):
            s = self.do_snippet(snip).strip()
            if s:
                body_parts.append(f"[{s}]")
        table = node.find(".//TABLE")
        if table is not None:
            body_parts.append(f"[{self.do_table(table).strip()}]")
        pdf_only = node.find("PDF_ONLY")
        if pdf_only is not None and src is None:
            extra = self.children(pdf_only).strip()
            if extra:
                body_parts.append(f"[{extra}]")

        if not body_parts:
            body_parts.append("[]")
        body = (
            body_parts[0]
            if len(body_parts) == 1
            else "stack(dir: ttb, spacing: 1em, " + ", ".join(body_parts) + ")"
        )

        caption_node = node.find("CAPTION")
        caption = ""
        if caption_node is not None:
            caption = f", caption: [{self.inline_of(caption_node)}]"

        label = f", label-name: <{fig_label}>" if fig_label else ""

        return f"\n#sicp-figure({body}{caption}{label})\n"

    def _image(self, src: str, node: ET.Element) -> str:
        """An <IMAGE>/figure source as a Typst image call."""
        scale = node.get("scale")
        try:
            width = f"{min(float(scale) * 100, 100):.0f}%" if scale else "70%"
        except ValueError:
            width = "70%"
        # Keep the source's subdirectory: img_javascript/ and img_original/
        # hold different variants under the same file name.
        rel = src.lstrip("/")
        return f'image("/images/{escape_string(rel)}", width: {width})'

    def do_image(self, node):
        return f"\n#figure({self._image(node.get('src', ''), node)})\n"

    def do_exercise(self, node):
        name = self._claim(child(node, "LABEL"))
        label = f"label-name: <{name}>, " if name else ""
        body = self.children(node).strip()
        return f"\n#exercise({label}[\n{body}\n])\n"

    def do_footnote(self, node):
        # Claim the label first: it must not also be emitted as an anchor
        # while the body is converted.
        name = self._claim(child(node, "LABEL"))
        body = self.children(node).strip()
        label = f"<{name}>" if name else ""
        return f"#footnote[{body}]{label}"

    def do_epigraph(self, node):
        attribution = child(node, "ATTRIBUTION")
        body_parts = [
            self.convert(c) + self.text(c.tail)
            for c in node
            if c.tag != "ATTRIBUTION"
        ]
        body = (self.text(node.text) + "".join(body_parts)).strip()
        args = [f"[{body}]"]
        if attribution is not None:
            args.append(self._attribution_args(attribution))
        return "\n#epigraph(" + ", ".join(a for a in args if a) + ")\n"

    def _attribution_args(self, node: ET.Element) -> str:
        parts = []
        for tag, key in (("AUTHOR", "author"), ("TITLE", "title"),
                         ("DATE", "date")):
            el = node.find(tag)
            if el is not None:
                parts.append(f"{key}: [{self.inline_of(el)}]")
        return ", ".join(parts)

    def do_signature(self, node):
        attribution = child(node, "ATTRIBUTION")
        if attribution is None:
            return self.children(node)
        args = self._attribution_args(attribution)
        return f"\n#align(left)[#emph[{args}]]\n" if args else ""

    def do_attribution(self, node):
        return self.children(node)

    def do_citation(self, node):
        text_el = node.find("TEXT")
        return self.inline_of(text_el if text_el is not None else node)

    def do_reference(self, node):
        body = self.children(node).strip()
        return f"\n#block(inset: (left: 1.2em), spacing: 0.8em)[{body}]\n"

    def do_ref(self, node):
        name = label_target(node.get("NAME", ""))
        if not name:
            return ""
        self.stats.refs.add(name)
        return f"@{name}"

    do_pageref = do_ref

    def do_index(self, node):
        """An index entry: term, optional sub-entry, and cross references."""
        decl = node.find(".//DECLARATION") is not None
        sub_el = child(node, "SUBINDEX")
        see_el = child(node, "SEE")
        seealso_el = child(node, "SEEALSO")
        order_el = child(node, "ORDER")

        term = self._index_term(node, skip={"SUBINDEX", "SEE", "SEEALSO",
                                            "ORDER", "OPEN", "CLOSE"})
        if not term:
            return ""

        args = [f'"{escape_string(term)}"']
        if sub_el is not None:
            sub = self._index_term(sub_el, skip={"ORDER", "OPEN", "CLOSE",
                                                 "ECMA", "PLR"})
            if sub:
                args.append(f'sub: "{escape_string(sub)}"')
        if order_el is not None:
            key = node_text(order_el)
            if key:
                args.append(f'sort: "{escape_string(key)}"')
        if decl:
            args.append("decl: true")
        # Cross-references are printed as plain strings too, so any maths in
        # them gets the same treatment as the term itself.
        if see_el is not None:
            see = latex_to_text(node_text(see_el))
            args.append(f'see: "{escape_string(see)}"')
        if seealso_el is not None:
            also = latex_to_text(node_text(seealso_el))
            args.append(f'see-also: "{escape_string(also)}"')
        return "#idx(" + ", ".join(args) + ")"

    @staticmethod
    def _index_term(node: ET.Element, skip: set[str]) -> str:
        """Plain-text form of an index term, ignoring structural children."""
        parts: list[str] = [node.text or ""]
        for c in node:
            if c.tag in skip:
                pass
            elif c.tag in SYMBOL_TAGS:
                parts.append(SYMBOL_TAGS[c.tag])
            elif c.tag in ("DECLARATION", "USE", LANG_INLINE,
                           COMPANION_INLINE, "EM", "B", "TT"):
                parts.append("".join(c.itertext()))
            elif c.tag in ("SPLIT", "SPLITINLINE"):
                # Only the Python branch: itertext() would run the Scheme and
                # Python wordings together ("procedurefunction").
                # An Element with no children is falsy, so `a or b` would
                # discard a branch that was actually found.
                branch = child(c, LANG_BLOCK)
                if branch is None:
                    branch = child(c, LANG_INLINE)
                parts.append("".join(branch.itertext()) if branch is not None
                             else "")
            elif c.tag in DROP_TAGS:
                pass
            else:
                parts.append("".join(c.itertext()))
            parts.append(c.tail or "")
        term = re.sub(r"\s+", " ", "".join(parts)).strip()
        # Index entries are printed and sorted as plain strings, so any maths
        # in them has to become literal characters, not Typst markup.
        term = latex_to_text(term)
        # makeindex uses `"` to escape its own control characters, so
        # `TK"!Solver` is the product name TK!Solver.
        term = re.sub(r'"([!@|"])', r"\1", term)
        return term.strip(",; ")

    def do_pythonoutput(self, node):
        code = dedent_code(raw_code(node))
        return f"\n#output({self._raw_literal(code)})\n" if code else ""

    do_python_output = do_pythonoutput

    def do_pythonprompt(self, node):
        code = dedent_code(raw_code(node))
        return f"\n#prompt({self._raw_literal(code)})\n" if code else ""

    do_python_prompt = do_pythonprompt

    def do_do_break_page(self, node):
        return "\n#pagebreak(weak: true)\n"

    def do_l(self, node):
        return "L"

    def do_t(self, node):
        return "T"

    def do_pdf_only(self, node):
        """<PDF_ONLY> — content the LaTeX build keeps and the web build drops.

        Usually real content, but the sources also use it for bare page-layout
        directives (`\\newpage`, `\\suppressfloats`, `\\addtocontents{...}`),
        which have no Typst equivalent and must not reach the page as text.
        """
        if len(node) == 0:
            text = (node.text or "").strip()
            if not text or text.startswith("\\") or text in ("$\\!$",):
                return ""
        return self.children(node)

    def do_latex_(self, node):
        return "LaTeX"

    def do_tex(self, node):
        """`<TeX/>` — the logo. Distinct from <LATEX>, which is math."""
        return "TeX"


# ── file-level driver ──────────────────────────────────────────────────────

@dataclass
class Unit:
    """One source file and the generated file it becomes."""
    src: Path
    dest: Path          # relative to content/
    entity: str         # entity name used to include it, e.g. "subsection1.1.1"


def plan_units(source: Path) -> list[Unit]:
    """Mirror the upstream layout: chapters, their sections and subsections."""
    units: list[Unit] = []
    xml_root = source / "xml_py"

    for chap in sorted(xml_root.glob("chapter*/chapter*.xml")):
        n = re.search(r"chapter(\d+)", chap.stem).group(1)
        units.append(Unit(chap, Path(f"chapter{n}/chapter{n}.typ"), f"chapter{n}"))
        for sec in sorted(chap.parent.glob("section*/section*.xml")):
            s = re.search(r"section(\d+)", sec.stem).group(1)
            units.append(Unit(
                sec,
                Path(f"chapter{n}/section{s}/section{s}.typ"),
                f"section{n}.{s}",
            ))
            for sub in sorted(sec.parent.glob("subsection*.xml")):
                b = re.search(r"subsection(\d+)", sub.stem).group(1)
                units.append(Unit(
                    sub,
                    Path(f"chapter{n}/section{s}/subsection{b}.typ"),
                    f"subsection{n}.{s}.{b}",
                ))

    for other in sorted((xml_root / "others").glob("*.xml")):
        units.append(Unit(other, Path("others") / (other.stem + ".typ"),
                          other.stem))
    return units


def include_path(from_dest: Path, to_dest: Path) -> str:
    """Path of `to_dest` as written inside the file at `from_dest`."""
    up = [".."] * len(from_dest.parent.parts)
    return "/".join([*up, *to_dest.parts])


def convert_unit(unit: Unit, units: list[Unit], conv: Converter) -> str:
    """Convert one source file to Typst markup (without the file header)."""
    body = conv.children(load_xml(unit.src))

    by_entity = {u.entity: u for u in units}

    def include(m: re.Match) -> str:
        target = by_entity.get(m.group(1))
        if target is None:
            return ""
        return f'\n#include "{include_path(unit.dest, target.dest)}"\n'

    body = ENTITY_RE.sub(include, body)
    return body


# `@ref` to a label that no target defines is a hard error in Typst, and the
# sources contain a handful (they are dangling in the LaTeX build too, where
# they silently render as "??"). They are rewritten to plain text instead.
#
# Typst ends a reference at trailing punctuation, so `@sec:foo.` refers to
# `sec:foo` — the pattern has to match that rule exactly, or valid references
# get rewritten by mistake.
REF_RE = re.compile(r"@([A-Za-z0-9_-]+(?::[A-Za-z0-9_-]+)*)")


# One reference is broken by a rename rather than by missing content: the
# footnote it points at is right there in the sources, but its LABEL was
# renamed with a `_2` suffix and the REF was never updated. Redirecting it
# restores a real cross-reference. (The other dangling name,
# `foot:function-decl-vs-lambda`, has no target in this edition at all -- the
# footnote it refers to is about JavaScript hoisting and was dropped when the
# book was ported to Python -- so it stays a "?" as in the LaTeX build.)
REF_ALIASES = {"foot:value_producing": "foot:value_producing_2"}


def fix_dangling_refs(text: str, defined: set[str], seen: Counter) -> str:
    def sub(m: re.Match) -> str:
        name = m.group(1)
        if name in defined:
            return m.group(0)
        target = REF_ALIASES.get(name)
        if target is not None and target in defined:
            return "@" + target
        seen[name] += 1
        return "#text(fill: red)[?]"

    return REF_RE.sub(sub, text)


# Each generated file is compilable on its own (`typst compile` on a single
# chapter works), so every one imports the library for itself.
FILE_HEADER = (
    "// Generated from the SICP XML sources by tools/convert.py — do not edit.\n"
    '#import "{lib}": *\n\n'
)


def tidy(dest: Path, body: str) -> str:
    """Add the file header and collapse the blank-line noise from nesting."""
    body = re.sub(r"[ \t]+\n", "\n", body)
    body = re.sub(r"\n{3,}", "\n\n", body)
    up = [".."] * (len(dest.parent.parts) + 1)  # +1 to escape content/
    lib = "/".join([*up, "lib", "sicp.typ"])
    return FILE_HEADER.format(lib=lib) + body.strip() + "\n"


def copy_images(source: Path, out: Path) -> tuple[int, list[str]]:
    """Copy every image the sources reference into images/."""
    used: set[str] = set()
    for xml in (source / "xml_py").rglob("*.xml"):
        for m in re.finditer(r'src="([^"]+)"', xml.read_text(encoding="utf-8")):
            used.add(m.group(1))

    dest = out / "images"
    dest.mkdir(parents=True, exist_ok=True)
    copied, missing = 0, []
    for rel in sorted(used):
        srcfile = source / "static" / rel
        if not srcfile.exists():
            missing.append(rel)
            continue
        target = dest / rel
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(srcfile, target)
        copied += 1
    return copied, missing


def main() -> int:
    ap = argparse.ArgumentParser(
        description="Convert the SICP Python-edition XML sources to Typst.")
    ap.add_argument("--source", required=True, type=Path,
                    help="checkout of https://github.com/source-academy/sicp")
    ap.add_argument("--out", type=Path,
                    default=Path(__file__).resolve().parent.parent,
                    help="project root to write into (default: this project)")
    ap.add_argument("--skip-images", action="store_true",
                    help="do not copy figure images")
    args = ap.parse_args()

    source: Path = args.source.expanduser().resolve()
    out: Path = args.out.expanduser().resolve()
    if not (source / "xml_py").is_dir():
        print(f"error: {source} has no xml_py/ directory", file=sys.stderr)
        return 1

    units = plan_units(source)
    content = out / "content"
    if content.exists():
        shutil.rmtree(content)

    stats = Stats()
    conv = Converter(stats)

    # Pass 1: convert everything, collecting the set of labels that exist.
    bodies: dict[Path, str] = {}
    for unit in units:
        bodies[unit.dest] = convert_unit(unit, units, conv)
        stats.files += 1

    # Pass 2: neutralise references with no target, then write the files.
    dangling: Counter = Counter()
    for dest, body in bodies.items():
        body = fix_dangling_refs(body, stats.labels, dangling)
        path = content / dest
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(tidy(dest, body), encoding="utf-8")

    if not args.skip_images:
        copied, missing = copy_images(source, out)
        print(f"images: copied {copied}")
        for m in missing:
            print(f"  warning: missing image {m}", file=sys.stderr)

    print(f"converted {stats.files} files, {len(stats.labels)} labels")
    if stats.unknown_tags:
        print("unhandled tags:", dict(stats.unknown_tags))
    if stats.unknown_latex:
        print("unhandled LaTeX:", sorted(stats.unknown_latex))
    if stats.duplicate_labels:
        print(f"duplicate labels dropped: {dict(stats.duplicate_labels)}")
    if dangling:
        print(f"references with no target: {dict(dangling)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

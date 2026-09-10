"""Translate the LaTeX math used in the SICP sources into Typst math.

The XML sources carry mathematics as LaTeX inside <LATEX> / <LATEXINLINE>
elements, plus a few LaTeX-isms that leak into snippets marked LATEX="yes".
The subset actually used in the book is small and regular, so a focused
recursive translator handles it exactly rather than approximately.

Everything this module cannot translate is reported through `unknown`, so the
build can surface it instead of silently emitting broken math.
"""

from __future__ import annotations

import re

# ── symbols ────────────────────────────────────────────────────────────────

GREEK = {
    "alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta", "theta",
    "iota", "kappa", "lambda", "mu", "nu", "xi", "pi", "rho", "sigma", "tau",
    "upsilon", "phi", "chi", "psi", "omega",
    "Gamma", "Delta", "Theta", "Lambda", "Xi", "Pi", "Sigma", "Upsilon",
    "Phi", "Psi", "Omega",
}

SYMBOLS = {
    "cdot": "dot.op", "cdots": "dots.c", "ldots": "dots.h", "vdots": "dots.v",
    "ddots": "dots.down", "dots": "dots.h",
    "times": "times", "div": "div", "pm": "plus.minus", "mp": "minus.plus",
    "leq": "lt.eq", "le": "lt.eq", "geq": "gt.eq", "ge": "gt.eq",
    "neq": "eq.not", "ne": "eq.not", "equiv": "equiv", "approx": "approx",
    "sim": "tilde.op", "propto": "prop", "ll": "lt.double", "gg": "gt.double",
    "in": "in", "notin": "in.not", "subset": "subset", "subseteq": "subset.eq",
    "cup": "union", "cap": "inter", "emptyset": "nothing",
    "infty": "infinity", "partial": "diff", "nabla": "nabla",
    "forall": "forall", "exists": "exists", "neg": "not",
    "land": "and", "lor": "or", "wedge": "and", "vee": "or",
    "rightarrow": "arrow.r", "to": "arrow.r", "leftarrow": "arrow.l",
    "gets": "arrow.l", "leftrightarrow": "arrow.l.r",
    "Rightarrow": "arrow.r.double", "Leftarrow": "arrow.l.double",
    "Leftrightarrow": "arrow.l.r.double",
    "mapsto": "arrow.r.bar", "longmapsto": "arrow.r.long.bar",
    "uparrow": "arrow.t", "downarrow": "arrow.b",
    "longrightarrow": "arrow.r.long", "longleftarrow": "arrow.l.long",
    # typst 0.15 has no angle.l/angle.r modifiers: use the characters.
    "langle": "⟨", "rangle": "⟩",
    "lfloor": "floor.l", "rfloor": "floor.r",
    "lceil": "ceil.l", "rceil": "ceil.r",
    "prime": "prime", "circ": "circle.small", "bullet": "bullet",
    "star": "star", "ast": "ast.op",
    # Written as literal characters: the `plus.circle` / `times.circle`
    # modifier spellings are not available in every Typst release.
    "oplus": "\u2295", "otimes": "\u2297",
    "perp": "perp", "angle": "angle",
    "ell": "ell", "Re": "Re", "Im": "Im", "aleph": "aleph",
    "hbar": "\u210f", "degree": "degree",
    "quad": "quad", "qquad": "wide", ",": "thin", ";": "med", "!": "",
    " ": "space", ":": "med",
    # Escaped punctuation. Braces stay escaped in Typst too, otherwise they
    # open a code block; the rest are literal.
    "%": "%", "&": "&", "#": "#", "_": "_", "$": "$",
    "{": "\\{", "}": "\\}",
    "mhyphen": "-", "-": "",
}

# \log x, \sin x, ... — Typst spells these the same way.
FUNCTIONS = {
    "log", "ln", "lg", "exp", "sin", "cos", "tan", "cot", "sec", "csc",
    "arcsin", "arccos", "arctan", "sinh", "cosh", "tanh", "min", "max",
    "inf", "sup", "lim", "det", "dim", "gcd", "deg", "arg", "mod",
}

# \big( ... \right] — size hints Typst applies automatically.
# Primitives that take an argument which should not be rendered: rigid
# boxes, rules and manual kerning.
DISCARD_ARG = {"hbox", "raise", "hskip", "vskip", "rule", "kern", "vspace",
               "hspace", "makebox", "phantom"}

SIZE_HINTS = {
    "big", "Big", "bigg", "Bigg", "bigl", "Bigl", "biggl", "Biggl",
    "bigr", "Bigr", "biggr", "Biggr", "left", "right", "displaystyle",
    "textstyle", "scriptstyle", "limits", "nolimits", "normalsize",
    "small", "footnotesize", "normalcodesize", "protect", "noindent",
    "smallskip", "medskip", "bigskip", "hfill", "hfil", "quad", "qquad",
    "relax", "leavevmode", "strut",
}

# One mandatory argument, rendered as a styled run of text.
TEXT_STYLES = {
    "textrm": "upright", "textup": "upright", "mbox": "upright",
    "text": "upright", "textnormal": "upright",
    "mathrm": "upright", "operatorname": "upright",
    "textit": "italic", "mathit": "italic", "emph": "italic",
    "textsl": "italic",
    "textbf": "bold", "mathbf": "bold",
    "texttt": "mono", "mathtt": "mono",
    "mathsf": "sans", "textsf": "sans",
    "mathcal": "cal", "mathbb": "bb", "mathfrak": "frak",
}

# Font *declarations*: unlike the commands above these take no argument and
# apply to the rest of the enclosing group (`{\tt foo bar}`, or a whole
# tabular cell).
FONT_DECLARATIONS = {
    "tt": "mono", "bf": "bold", "it": "italic", "sl": "italic",
    "rm": "upright", "sf": "sans", "sc": "upright",
}

TWO_ARG = {"frac": "frac", "dfrac": "frac", "tfrac": "frac", "binom": "binom"}

ACCENTS = {
    "hat": "hat", "bar": "macron", "vec": "arrow", "tilde": "tilde",
    "dot": "dot", "ddot": "dot.double", "overline": "overline",
    "underline": "underline", "widehat": "hat", "widetilde": "tilde",
    "acute": "acute", "grave": "grave", "breve": "breve", "check": "caron",
    "mathring": "circle",
}


class MathError(Exception):
    pass


# Typst reads a run of letters as one identifier, while LaTeX reads it as a
# product of single-letter variables: `$ax$` is a·x, not a variable "ax".
# Runs of plain letters are therefore spaced out — except for the names Typst
# itself defines (`sum`, `frac`, `pi`, ...), which must stay intact.
_KNOWN_NAMES = (
    GREEK
    | set(FUNCTIONS)
    | {v for v in SYMBOLS.values() if v and v[0].isalpha()}
    | {"frac", "binom", "sqrt", "mat", "sum", "product", "integral", "lim",
       "upright", "italic", "bold", "mono", "sans", "cal", "bb", "frak",
       "hat", "macron", "arrow", "tilde", "dot", "overline", "underline",
       "delim", "none", "thin", "med", "quad", "wide", "space", "abs",
       "floor", "ceil", "norm", "text", "op", "attach", "t", "b", "tr", "bl",
       "cases", "lr", "vec", "display", "inline", "scripts", "limits",
       "acute", "grave", "breve", "caron", "circle", "double"}
)

_WORD_RE = re.compile(r"[A-Za-z]{2,}")


# A dotted symbol name (`arrow.r.bar`) is a single token: its parts must not
# be split, even though `bar` on its own is not a known name.
_DOTTED_RE = re.compile(r"[A-Za-z]+(?:\.[A-Za-z]+)+")


def separate_variables(expr: str) -> str:
    """Space out multi-letter variable runs so Typst reads them as products."""
    out: list[str] = []
    pos = 0
    # Quoted text and dotted symbol names are literal and must not be touched.
    protected = re.compile(r'"(?:[^"\\]|\\.)*"|' + _DOTTED_RE.pattern)
    for lit in protected.finditer(expr):
        out.append(_split_words(expr[pos:lit.start()]))
        out.append(lit.group(0))
        pos = lit.end()
    out.append(_split_words(expr[pos:]))
    return "".join(out)


def _split_words(chunk: str) -> str:
    def sub(m: re.Match) -> str:
        word = m.group(0)
        if word in _KNOWN_NAMES:
            return word
        return " ".join(word)

    return _WORD_RE.sub(sub, chunk)


class Tokenizer:
    """Split LaTeX math source into command / brace / character tokens."""

    def __init__(self, src: str):
        self.src = src
        self.i = 0

    def peek(self) -> str | None:
        return self.src[self.i] if self.i < len(self.src) else None

    def next_token(self) -> tuple[str, str] | None:
        if self.i >= len(self.src):
            return None
        c = self.src[self.i]
        if c == "\\":
            m = re.match(r"\\([A-Za-z]+|.)", self.src[self.i:], re.S)
            if not m:
                self.i += 1
                return ("char", "\\")
            self.i += m.end()
            return ("cmd", m.group(1))
        self.i += 1
        return ("char", c)


def _esc_text(s: str) -> str:
    """Escape a literal string for use inside a Typst string."""
    return s.replace("\\", "\\\\").replace('"', '\\"')


def _unquote(body: str) -> str | None:
    """The plain text of `body`, if it is entirely literal; else None.

    Styling functions like `mono(..)` read best with a string argument, but
    only when the content really is plain text — `mono("bold(x)")` would
    print the markup instead of applying it.
    """
    body = body.strip()
    # Already a bare string literal.
    m = re.fullmatch(r'"((?:[^"\\]|\\.)*)"', body)
    if m:
        return m.group(1)
    # Plain letters, digits and punctuation, with no Typst markup in sight.
    # `separate_variables` may already have spaced the letters out, so those
    # single-character gaps are closed again here.
    if re.fullmatch(r"[A-Za-z0-9 ,.:;!?()\[\]+*/=<>@_-]*", body):
        return re.sub(r"(?<=\b\w) (?=\w\b)", "", body)
    return None


class MathTranslator:
    def __init__(self, src: str, unknown: set[str] | None = None):
        self.tk = Tokenizer(src)
        self.unknown = unknown if unknown is not None else set()

    # -- helpers ------------------------------------------------------------

    def _group(self) -> str:
        """Read one argument: a braced group, a command, or a single char."""
        self._skip_space()
        c = self.tk.peek()
        if c is None:
            return ""
        if c == "{":
            self.tk.i += 1
            return self._until_close()
        tok = self.tk.next_token()
        return self._token_to_typst(tok)

    def _raw_group(self) -> str | None:
        """Consume one braced group and return its source, untranslated.

        Used by the text-styling commands: `\\texttt{list("x", )}` is a run of
        literal characters, and reproducing it verbatim inside a Typst string
        is both simpler and safer than translating it as math.
        Returns None (consuming nothing) when the group contains commands,
        which have to go through the translator instead.
        """
        self._skip_space()
        if self.tk.peek() != "{":
            return None
        start = self.tk.i
        self.tk.i += 1
        depth = 0
        out: list[str] = []
        while self.tk.i < len(self.tk.src):
            c = self.tk.src[self.tk.i]
            if c == "{":
                depth += 1
            elif c == "}":
                if depth == 0:
                    self.tk.i += 1
                    text = "".join(out)
                    # A command (other than escaped punctuation) or an
                    # unescaped `$` means this is real math, not literal text.
                    if re.search(r"\\(?![$#%_&{}])", text) or re.search(
                        r"(?<!\\)\$", text
                    ):
                        self.tk.i = start  # let the translator handle it
                        return None
                    # `\$`, `\#`, `\%`, `\_`, `\&` are literal characters.
                    return re.sub(r"\\([$#%_&{}])", r"\1", text)
                depth -= 1
            out.append(c)
            self.tk.i += 1
        self.tk.i = start
        return None

    def _skip_space(self) -> None:
        while self.tk.peek() is not None and self.tk.peek() in " \t\n":
            self.tk.i += 1

    def _until_close(self) -> str:
        """Translate up to the matching closing brace (already consumed '{')."""
        out: list[str] = []
        depth = 0
        while True:
            c = self.tk.peek()
            if c is None:
                break
            if c == "}" and depth == 0:
                self.tk.i += 1
                break
            tok = self.tk.next_token()
            if tok[0] == "char" and tok[1] == "{":
                depth += 1
            elif tok[0] == "char" and tok[1] == "}":
                depth -= 1
            out.append(self._token_to_typst(tok))
        return "".join(out)

    # -- main loop ----------------------------------------------------------

    def translate(self) -> str:
        out: list[str] = []
        while True:
            tok = self.tk.next_token()
            if tok is None:
                break
            out.append(self._token_to_typst(tok))
        result = re.sub(r"[ ]{2,}", " ", "".join(out)).strip()
        # The sources attach a subscript to the *preceding* prose word, as in
        # `<META>seq</META>$_1$`. Typst needs something to attach it to, so an
        # empty base is supplied.
        if result.startswith(("_", "^")):
            result = '""' + result
        return separate_variables(result)

    def _token_to_typst(self, tok: tuple[str, str]) -> str:
        kind, val = tok
        if kind == "char":
            return self._char(val)
        return self._command(val)

    def _char(self, c: str) -> str:
        if c == "{":
            return "("
        if c == "}":
            return ")"
        if c == "$":
            return ""
        if c == "~":
            return " "
        if c == "'":
            return "'"
        if c in "^_":
            # Superscript/subscript. The argument is always parenthesised:
            # `sum_i v_i` would otherwise run together into `sum_iv_i`, whose
            # subscript Typst reads as the single identifier `iv`.
            arg = self._group().strip()
            if not arg:
                return ""
            if not (arg.startswith("(") and arg.endswith(")")):
                arg = "(" + arg + ")"
            return c + arg
        if c == "&":
            return "&"
        if c == "%":
            return ""
        if c == "\n":
            return " "
        return c

    def _command(self, name: str) -> str:
        if name == "left":
            return self._left_right()
        if name == "char":
            return self._char_primitive()
        if name in ("def", "newcommand", "renewcommand"):
            self._skip_macro_definition()
            return ""
        if name in DISCARD_ARG:
            # Rigid-box and rule primitives: their measurements mean nothing
            # here, and their argument is not content to show.
            self._group()
            if name == "rule":
                self._group()  # \rule takes width *and* height
            return ""
        if name in SIZE_HINTS:
            return ""
        if name == "\\":  # row break inside an array
            return "\\ "
        # Named tokens are padded with spaces: in Typst math two adjacent
        # identifiers written without one read as a single longer identifier
        # (`ll` + `mathit` would become the unknown variable `llmathit`).
        if name in GREEK:
            return f" {name} "
        if name in SYMBOLS:
            sym = SYMBOLS[name]
            needs_space = bool(re.match(r"^[A-Za-z]", sym))
            return f" {sym} " if needs_space else sym
        if name in FUNCTIONS:
            return f" {name} "
        if name in TWO_ARG:
            a, b = self._group(), self._group()
            return f"{TWO_ARG[name]}({a}, {b})"
        if name in ACCENTS:
            return f"{ACCENTS[name]}({self._group()})"
        if name in TEXT_STYLES:
            style = TEXT_STYLES[name]
            # Literal runs are reproduced verbatim; anything containing
            # further commands goes through the translator.
            literal = self._raw_group()
            if literal is not None:
                return f'{style}("{_esc_text(literal)}")' if literal else ""
            return self._styled(style, self._group())
        if name in FONT_DECLARATIONS:
            # A declaration styles everything up to the end of its group, so
            # the rest of this group is consumed and wrapped.
            rest = self._rest_of_group()
            return self._styled(FONT_DECLARATIONS[name], rest)
        if name == "sqrt":
            return f"sqrt({self._group()})"
        if name in ("sum", "prod", "int", "lim"):
            return {"sum": "sum", "prod": "product", "int": "integral",
                    "lim": "lim"}[name]
        if name == "begin":
            return self._environment()
        if name == "end":
            self._group()
            return ""
        if name == "verb":
            return self._verb()
        if name == "hspace" or name == "vspace":
            self._group()
            return " "
        if name == "hline":
            return ""
        if name == "ensuremath":
            return self._group()
        self.unknown.add("\\" + name)
        return ""

    @staticmethod
    def _styled(style: str, body: str) -> str:
        """Wrap already-translated `body` in a Typst styling function."""
        if not body.strip():
            return ""
        # These three take a string; the rest take math content.
        if style in ("upright", "italic", "mono"):
            plain = _unquote(body)
            if plain is not None:
                return f'{style}("{_esc_text(plain)}")'
        return f"{style}({body})"

    def _rest_of_group(self) -> str:
        """Translate the remainder of the current group (for declarations)."""
        out: list[str] = []
        depth = 0
        while True:
            c = self.tk.peek()
            if c is None:
                break
            if c == "}" and depth == 0:
                break  # leave the closing brace to the caller
            tok = self.tk.next_token()
            if tok[0] == "char" and tok[1] == "{":
                depth += 1
            elif tok[0] == "char" and tok[1] == "}":
                depth -= 1
            out.append(self._token_to_typst(tok))
        return "".join(out).strip()

    def _verb(self) -> str:
        """\\verb+text+ — delimiter is the next character."""
        delim = self.tk.peek()
        if delim is None:
            return ""
        self.tk.i += 1
        end = self.tk.src.find(delim, self.tk.i)
        if end < 0:
            end = len(self.tk.src)
        body = self.tk.src[self.tk.i:end]
        self.tk.i = end + 1
        return f'mono("{_esc_text(body)}")'

    def _left_right(self) -> str:
        """`\\left<d> ... \\right<d>` — a delimited group.

        `\\left\\{ \\begin{array}{ll} ... \\right.` is how the sources write a
        piecewise definition, which is Typst's `cases`; anything else keeps
        its delimiters and is wrapped in `lr(..)` so they scale.
        """
        self._skip_space()
        opener = self._delimiter()
        start = self.tk.i
        depth = 0
        # Find the matching \right, skipping any nested \left..\right pairs.
        for m in re.finditer(r"\\(left|right)", self.tk.src[start:]):
            if m.group(1) == "left":
                depth += 1
                continue
            if depth:
                depth -= 1
                continue
            inner_src = self.tk.src[start:start + m.start()]
            self.tk.i = start + m.end()
            closer = self._delimiter()
            break
        else:  # unmatched \left: treat the rest as the body
            inner_src = self.tk.src[start:]
            self.tk.i = len(self.tk.src)
            closer = ""

        stripped = inner_src.strip()
        if opener in ("{", "\\{") and re.match(r"\\begin\s*\{array\}", stripped):
            rows = MathTranslator(stripped, self.unknown)._cases_rows()
            if rows is not None:
                return "cases(" + ", ".join(rows) + ")"

        inner = latex_to_typst(inner_src, self.unknown)
        if opener in ("", ".") and closer in ("", "."):
            return inner
        return f"lr({opener} {inner} {closer})".replace("  ", " ")

    def _skip_macro_definition(self) -> None:
        """Consume a `\\def\\name{body}` (the sources define one, `\\aal`).

        The macro is never actually used in the text that follows, so the
        definition is simply dropped rather than expanded.
        """
        self._skip_space()
        if self.tk.peek() == "\\":  # the macro's name
            self.tk.i += 1
            m = re.match(r"[A-Za-z]+", self.tk.src[self.tk.i:])
            if m:
                self.tk.i += m.end()
        self._skip_space()
        if self.tk.peek() == "{":  # its body
            self.tk.i += 1
            depth = 0
            while self.tk.i < len(self.tk.src):
                c = self.tk.src[self.tk.i]
                self.tk.i += 1
                if c == "\\":
                    self.tk.i += 1  # an escaped brace is not a delimiter
                elif c == "{":
                    depth += 1
                elif c == "}":
                    if depth == 0:
                        return
                    depth -= 1

    def _char_primitive(self) -> str:
        """TeX's `\\char` — a character by code.

        The sources use it only as `\\char`_` (backtick-quoted literal) to get
        an underscore into a `\\texttt` name, e.g. `math\\char`_pow`.
        """
        self._skip_space()
        c = self.tk.peek()
        if c == "`":  # `\char`X` — the character X itself
            self.tk.i += 1
            if self.tk.peek() == "\\":
                self.tk.i += 1
            ch = self.tk.peek() or ""
            self.tk.i += 1
            return ch
        m = re.match(r"'?\d+", self.tk.src[self.tk.i:])
        if m:  # octal or decimal code point
            self.tk.i += m.end()
            digits = m.group(0)
            code = int(digits[1:], 8) if digits[0] == "'" else int(digits)
            return chr(code)
        return ""

    def _delimiter(self) -> str:
        """Read the delimiter after \\left / \\right."""
        self._skip_space()
        c = self.tk.peek()
        if c is None:
            return ""
        if c == ".":  # `\right.` — no delimiter at all
            self.tk.i += 1
            return ""
        tok = self.tk.next_token()
        if tok[0] == "char":
            return {"{": "\\{", "}": "\\}"}.get(tok[1], tok[1])
        return SYMBOLS.get(tok[1], "")

    def _cases_rows(self) -> list[str] | None:
        """The rows of a `\\begin{array}` as Typst `cases` arguments."""
        self.tk.i = 0
        m = re.match(r"\\begin\s*\{array\}\s*(\{[^}]*\})?", self.tk.src)
        if not m:
            return None
        self.tk.i = m.end()
        body = self._until_end("array")
        rows = []
        for row in _split_top_level(body, "row"):
            if not row.strip():
                continue
            cells = [latex_to_typst(c, self.unknown).strip()
                     for c in _split_top_level(row, "&")]
            # `value & condition` reads better with a gap between the two.
            rows.append(" quad ".join(c for c in cells if c))
        return rows or None

    def _environment(self) -> str:
        env = self._group().strip()
        # Layout-only wrappers carry no structure of their own: keep
        # translating their contents in place. (`\begin{flushleft}` around a
        # `tabular` must not swallow the table.)
        if env in ("flushleft", "flushright", "center", "quote", "quotation"):
            return latex_to_typst(self._until_end(env), self.unknown)
        if env in ("array", "tabular", "matrix", "aligned", "align",
                   "align*", "eqnarray", "eqnarray*", "smallequation",
                   "Parsing", "ParsingNoPostPadding"):
            if env in ("array", "tabular"):
                self._skip_space()
                if self.tk.peek() == "{":  # column spec
                    self.tk.i += 1
                    depth = 0
                    while self.tk.peek() is not None:
                        c = self.tk.src[self.tk.i]
                        self.tk.i += 1
                        if c == "{":
                            depth += 1
                        elif c == "}":
                            if depth == 0:
                                break
                            depth -= 1
            body = self._until_end(env)
            return self._as_matrix(body, env)
        self.unknown.add(r"\begin{%s}" % env)
        self._until_end(env)
        return ""

    def _until_end(self, env: str) -> str:
        """Collect raw source up to \\end{env}."""
        pat = re.compile(r"\\end\s*\{" + re.escape(env) + r"\}")
        m = pat.search(self.tk.src, self.tk.i)
        end = m.start() if m else len(self.tk.src)
        body = self.tk.src[self.tk.i:end]
        self.tk.i = m.end() if m else len(self.tk.src)
        return body

    def _as_matrix(self, body: str, env: str) -> str:
        # `tabular` cells are LaTeX *text* mode; `array` cells are math mode.
        cell = text_mode_to_typst if env == "tabular" else latex_to_typst
        rows = _split_top_level(body, "row")
        cells: list[list[str]] = []
        for row in rows:
            row = row.strip()
            # A rule is not a row of data. It may be written `\hline`, or
            # `\hline{}` with an empty group -- which must not survive as a
            # stray `()` cell -- and it may lead a row that does carry cells.
            row = re.sub(r"^\\hline\s*(?:\{\s*\})?\s*", "", row)
            if not row or set(row) <= {"&", " "}:
                continue
            cols = [cell(c, self.unknown) for c in _split_top_level(row, "&")]
            cells.append(cols)
        if not cells:
            return ""
        if len(cells) == 1 and len(cells[0]) == 1:
            return cells[0][0]
        width = max(len(r) for r in cells)
        for r in cells:
            r.extend([""] * (width - len(r)))
        inner = "; ".join(", ".join(r) for r in cells)
        return f"mat(delim: #none, {inner})"


def _split_top_level(body: str, sep: str) -> list[str]:
    """Split on `\\` (rows) or `&` (cells) at the outermost level only.

    A nested `\left\{ \begin{array} ... \right.` carries its own separators,
    and those belong to the inner table, not to this one.
    """
    parts: list[str] = []
    depth = i = start = 0
    row_re = re.compile(r"\\\\(?:\s*\[[^\]]*\])?")
    while i < len(body):
        if body.startswith(r"\begin", i) or body.startswith(r"\left", i):
            depth += 1
            i += 5
            continue
        if body.startswith(r"\end", i) or body.startswith(r"\right", i):
            depth = max(0, depth - 1)
            i += 4
            continue
        if depth == 0:
            if sep == "&" and body[i] == "&" and (i == 0 or body[i - 1] != "\\"):
                parts.append(body[start:i])
                start = i = i + 1
                continue
            if sep == "row":
                m = row_re.match(body, i)
                if m:
                    parts.append(body[start:i])
                    start = i = m.end()
                    continue
        i += 1
    parts.append(body[start:])
    return parts


def latex_to_typst(src: str, unknown: set[str] | None = None) -> str:
    """Translate a LaTeX math fragment to Typst math markup (no $ delimiters)."""
    return MathTranslator(src, unknown).translate()


_TEXT_DECL_RE = re.compile(r"\\([A-Za-z]+)\s*")


def text_mode_to_typst(src: str, unknown: set[str] | None = None) -> str:
    """Translate a LaTeX *text-mode* fragment to Typst math content.

    Used for `tabular` cells, which — unlike `array` cells — hold prose:
    ``\\tt\\textbf{def}`` is the word "def" in bold monospace, not a product
    of the variables d, e and f. The result is math content because it is
    spliced into `mat(..)`, so literal text comes back quoted.
    """
    src = src.strip()
    if not src:
        return ""

    style: str | None = None
    out: list[str] = []
    pos = 0
    while pos < len(src):
        c = src[pos]
        if c == "$":  # embedded math, translated as such
            end = src.find("$", pos + 1)
            if end < 0:
                end = len(src)
            out.append(latex_to_typst(src[pos + 1:end], unknown))
            pos = end + 1
            continue
        if c == "\\":
            m = _TEXT_DECL_RE.match(src, pos)
            if m:
                name = m.group(1)
                pos = m.end()
                if name == "verb":
                    sub = MathTranslator(src[m.end(1):], unknown)
                    out.append(sub._verb())
                    pos = m.end(1) + sub.tk.i
                elif name in FONT_DECLARATIONS:
                    style = FONT_DECLARATIONS[name]
                elif name in TEXT_STYLES:
                    sub = MathTranslator(src[pos:], unknown)
                    arg = sub._raw_group()
                    if arg is None:
                        arg = sub._group()
                    pos += sub.tk.i
                    out.append(f'{TEXT_STYLES[name]}("{_esc_text(arg)}")')
                elif name in DISCARD_ARG:
                    sub = MathTranslator(src[pos:], unknown)
                    sub._group()
                    if name == "rule":
                        sub._group()  # \rule takes width *and* height
                    pos += sub.tk.i
                elif name in SIZE_HINTS or name in ("hline", "noalign"):
                    pass  # \normalsize, \hline: no Typst equivalent here
                elif name in SYMBOLS:
                    out.append(SYMBOLS[name])
                else:
                    unknown is not None and unknown.add("\\" + name)
                continue
            out.append(src[pos + 1])  # escaped punctuation
            pos += 2
            continue
        # A run of literal text.
        end = pos
        while end < len(src) and src[end] not in "$\\":
            end += 1
        chunk = src[pos:end]
        if chunk.strip():
            out.append(f'"{_esc_text(chunk.strip())}"')
        pos = end

    body = " ".join(p for p in out if p)
    if style and body:
        # A declaration applies to the whole remaining cell.
        return f"{style}({body})"
    return body


# ── whole <LATEXINLINE> / <LATEX> bodies ───────────────────────────────────

_DISPLAY_RE = re.compile(r"\\\[(.*?)\\\]", re.S)
_INLINE_RE = re.compile(r"(?<!\\)\$(.+?)(?<!\\)\$", re.S)
_BARE_ENV_RE = re.compile(r"^\\begin\{")


_MARKUP_STYLES = {
    "textbf": "strong", "bf": "strong",
    "textit": "emph", "textsl": "emph", "emph": "emph", "it": "emph",
    "texttt": "raw", "tt": "raw",
    "textrm": None, "textnormal": None, "mbox": None, "text": None,
}


def _text_command_to_markup(src: str, unknown: set[str] | None = None) -> str:
    """Render an undelimited `\\textbf{...}` chain as Typst *markup*."""
    m = re.fullmatch(r"\\([A-Za-z]+)\{(.*)\}", src.strip(), re.S)
    if not m:
        from .escape import escape_text
        return escape_text(src)
    name, body = m.group(1), m.group(2)
    if name not in _MARKUP_STYLES:
        if unknown is not None:
            unknown.add("\\" + name)
        from .escape import escape_text
        return escape_text(body)
    inner = _text_command_to_markup(body, unknown) if body.lstrip().startswith(
        "\\") else None
    style = _MARKUP_STYLES[name]
    if inner is None:
        from .escape import escape_text
        inner = f'#raw("{_esc_text(body)}")' if style == "raw" else escape_text(
            body)
        return inner if style in (None, "raw") else f"#{style}[{inner}]"
    return inner if style is None else f"#{style}[{inner}]"


def strip_latex_comments(src: str) -> str:
    """Drop LaTeX comments: an unescaped ``%`` to the end of its line.

    The sources use them both to annotate a formula (``\\hline{} %-----``,
    where the dashes merely draw the rule in the source) and to swallow the
    newline after ``\\end{Parsing}``. Either way the text is invisible in the
    LaTeX build, so it must not reach the page. ``\\%`` is a literal percent
    sign and is left alone.
    """
    return re.sub(r"(?<!\\)%[^\n]*", "", src)


def convert_latex_fragment(src: str, unknown: set[str] | None = None) -> str:
    """Convert a mixed text/math LaTeX fragment into Typst markup.

    Math delimited by ``$...$`` or ``\\[...\\]`` becomes Typst math; anything
    outside stays literal text (escaped by the caller).
    """
    from .escape import escape_text  # local import: avoids a cycle

    src = strip_latex_comments(src)

    # A fragment that is nothing but a LaTeX environment (the \\begin{Parsing}
    # grammar rules, alignment arrays, ...) is display math in its entirety;
    # it carries no $...$ delimiters of its own.
    if _BARE_ENV_RE.match(src.strip()):
        return "\n$ " + latex_to_typst(src.strip(), unknown) + " $\n"

    # A styling command with no `$` around it (`\textbf{\texttt{continue}}`)
    # is text, not maths, so it is rendered as markup rather than a formula.
    stripped = src.strip()
    if "$" not in stripped and re.fullmatch(r"\\[A-Za-z]+\{.*\}", stripped, re.S):
        return _text_command_to_markup(stripped, unknown)

    out: list[str] = []
    pos = 0
    for m in _DISPLAY_RE.finditer(src):
        out.append(escape_text(src[pos:m.start()]))
        body = latex_to_typst(m.group(1), unknown)
        out.append(f"\n$ {body} $\n")
        pos = m.end()
    rest = src[pos:]

    pos = 0
    for m in _INLINE_RE.finditer(rest):
        out.append(escape_text(rest[pos:m.start()]))
        body = latex_to_typst(m.group(1), unknown)
        out.append(f"${body}$")
        pos = m.end()
    out.append(escape_text(rest[pos:]))
    return "".join(out)


# ── plain-text rendering, for index terms ──────────────────────────────────

_GREEK_CHARS = {
    "alpha": "α", "beta": "β", "gamma": "γ", "delta": "δ", "epsilon": "ε",
    "zeta": "ζ", "eta": "η", "theta": "θ", "iota": "ι", "kappa": "κ",
    "lambda": "λ", "mu": "μ", "nu": "ν", "xi": "ξ", "pi": "π", "rho": "ρ",
    "sigma": "σ", "tau": "τ", "upsilon": "υ", "phi": "φ", "chi": "χ",
    "psi": "ψ", "omega": "ω",
    "Gamma": "Γ", "Delta": "Δ", "Theta": "Θ", "Lambda": "Λ", "Xi": "Ξ",
    "Pi": "Π", "Sigma": "Σ", "Upsilon": "Υ", "Phi": "Φ", "Psi": "Ψ",
    "Omega": "Ω",
}

_TEXT_SYMBOLS = {
    "mapsto": "↦", "rightarrow": "→", "to": "→", "leftarrow": "←",
    "Rightarrow": "⇒", "Leftarrow": "⇐", "leftrightarrow": "↔",
    "times": "×", "cdot": "·", "div": "÷", "pm": "±", "mp": "∓",
    "leq": "≤", "le": "≤", "geq": "≥", "ge": "≥", "neq": "≠", "ne": "≠",
    "approx": "≈", "equiv": "≡", "sim": "∼", "propto": "∝",
    "infty": "∞", "partial": "∂", "nabla": "∇", "sum": "∑", "prod": "∏",
    "int": "∫", "sqrt": "√", "in": "∈", "notin": "∉", "subset": "⊂",
    "cup": "∪", "cap": "∩", "emptyset": "∅", "forall": "∀", "exists": "∃",
    "ldots": "…", "cdots": "…", "dots": "…", "langle": "⟨", "rangle": "⟩",
    "ll": "≪", "gg": "≫", "circ": "∘", "bullet": "•", "star": "⋆",
    "mhyphen": "-", "lfloor": "⌊", "rfloor": "⌋", "lceil": "⌈",
    "rceil": "⌉", "ell": "ℓ", "hbar": "ℏ", "oplus": "⊕", "otimes": "⊗",
}

_SUPERSCRIPTS = str.maketrans("0123456789+-()n x", "⁰¹²³⁴⁵⁶⁷⁸⁹⁺⁻⁽⁾ⁿ ˣ")

_PLAIN_CMD_RE = re.compile(r"\\([A-Za-z]+)\s*|\\(.)")


def latex_to_text(src: str) -> str:
    """Render a LaTeX fragment as plain text.

    Index entries are sorted and printed as strings, so the maths in them
    (`$\\Sigma$ (sigma) notation`, `$e^x$, power series for`) has to become
    literal characters rather than Typst markup.
    """
    if not src:
        return ""
    out: list[str] = []
    i = 0
    while i < len(src):
        c = src[i]
        if c in "${}":  # mode switches and grouping carry no text
            i += 1
            continue
        if c == "^" or c == "_":
            i += 1
            if i < len(src) and src[i] == "{":
                end = src.find("}", i)
                arg = src[i + 1:end if end > 0 else len(src)]
                i = end + 1 if end > 0 else len(src)
            elif i < len(src):
                arg = src[i]
                i += 1
            else:
                arg = ""
            arg = latex_to_text(arg)
            out.append(arg.translate(_SUPERSCRIPTS) if c == "^" else arg)
            continue
        if c == "\\":
            m = _PLAIN_CMD_RE.match(src, i)
            if m:
                name, punct = m.group(1), m.group(2)
                i = m.end()
                if punct is not None:
                    out.append(punct)  # \$, \#, \& …
                elif name in _GREEK_CHARS:
                    out.append(_GREEK_CHARS[name])
                elif name in _TEXT_SYMBOLS:
                    out.append(_TEXT_SYMBOLS[name])
                # Style commands (\texttt, \textbf, …) keep only their text.
                continue
            i += 1
            continue
        out.append(c)
        i += 1
    return re.sub(r"\s+", " ", "".join(out)).strip()

// Code snippets, inline code, and interpreter transcripts.
//
// The XML sources distinguish four kinds of code material, all of which land
// here:
//
//   <PYTHON>        -> `snippet(..)`      a runnable program fragment
//   <PYTHONINLINE>  -> `py(..)`           inline code inside prose
//   <PYTHON_PROMPT> -> `prompt(..)`       what the reader types at a prompt
//   <PYTHON_OUTPUT> -> `output(..)`       what the interpreter prints back
//
// Snippets marked LATEX="yes" in the XML contain typeset meta-variables
// rather than real code; they are emitted as `syntax(..)` and rendered
// without syntax highlighting.

#let code-font = ("DejaVu Sans Mono", "Liberation Mono", "Menlo", "Consolas")
#let code-size = 8.8pt

#let code-block(body, fill: luma(248), stroke: luma(225)) = block(
  width: 100%,
  fill: fill,
  stroke: (paint: stroke, thickness: 0.5pt),
  radius: 2pt,
  inset: (x: 7pt, y: 6pt),
  breakable: true,
  spacing: 1em,
  body,
)

// Code arrives either as a plain string or as a ```-delimited raw block;
// accept both so generated files can use whichever is cleaner.
#let as-raw(code, lang: none) = if type(code) == str {
  raw(code, lang: lang, block: true)
} else {
  code
}

// Interpreter transcripts are executed through Calepin
// (https://typst.app/universe/package/calepin). `calepin compile` runs every
// transcript against a real, persistent python3 session and stores the
// results in `.calepin/`; plain `typst compile` cannot execute code, so it
// renders whatever the last Calepin run stored (nothing on a fresh clone).
// The facade in /.calepin/ is that runtime once generated, and a thin shim
// re-exporting the published compatibility package until then.
#import "/.calepin/calepin.typ" as calepin
#import "listings.typ": listings

/// The code Calepin executes for one interpreter transcript: the snippet runs
/// under `exec` (script semantics, exactly as the book narrates), so only
/// `print` output appears — a bare expression such as `486` prints nothing —
/// and a snippet that raises shows nothing either. The code is embedded as a
/// JSON literal, which is also a valid Python string literal.
#let transcript-source(code) = (
  "import sys, io",
  "_u = " + json.encode(code),
  "_b = io.StringIO()",
  "_o = sys.stdout",
  "sys.stdout = _b",
  "try:",
  "    exec(_u)",
  "except Exception:",
  "    pass",
  "finally:",
  "    sys.stdout = _o",
  "_out = _b.getvalue()",
  "if _out:",
  "    print(_out, end=\"\")",
).join("\n")

/// Interpreter response, shown slanted as in the printed book. The transcript
/// is emitted as a Calepin chunk; `calepin compile` fills in the output, and
/// silence (no box at all) when the snippet prints nothing or fails.
#let output(code) = {
  let text-code = if type(code) == str { code } else { code.text }
  calepin.chunk(
    "python",
    raw(transcript-source(text-code), lang: "python", block: true),
    echo: false,
    warning: false,
    message: false,
    error: false,
  )
}

/// The book's styling for one interpreter transcript, applied document-wide
/// by the templates via a show rule on Calepin's `<calepin-output>` carrier.
#let interpreter-output(body) = code-block(fill: white, stroke: luma(235))[
  #set text(font: code-font, size: code-size, style: "oblique")
  #set par(justify: false, leading: 0.55em)
  #body
]

/// Activate the transcript styling. Call with `#show:` in each book template.
#let show-interpreter-outputs(body) = {
  show <calepin-output>: it => interpreter-output(it.body)
  body
}

/// A Python program fragment that displays code using listings.
#let snippet(code) = {
  let text-code = if type(code) == str { code } else { code.text }
  listings(text-code, language: "python")
}

/// A syntax template: code mixed with italic meta-variables. `parts` is a
/// sequence of strings (literal code) and content (already-typeset
/// meta-variables), so it cannot go through the syntax highlighter.
#let syntax(..parts) = code-block(fill: luma(252))[
  #set text(font: code-font, size: code-size)
  #set par(justify: false, leading: 0.55em)
  #parts.pos().join()
]

/// Inline code inside running prose.
#let py(code) = box(
  raw(code, lang: "python"),
)

/// Inline code that must not be highlighted (meta-variables, fragments).
#let py-plain(code) = box(text(font: code-font, size: code-size, code))

/// A meta-variable: <META> in the XML, italic in print.
#let meta(name) = box(text(style: "italic", font: "Libertinus Serif", name))

/// Angle-bracketed meta-phrase: <METAPHRASE> in the XML.
#let metaphrase(body) = box[⟨#text(style: "italic", body)⟩]

/// The prompt line preceding user input in an interpreter transcript.
#let prompt(code) = code-block(fill: luma(242))[
  #set text(font: code-font, size: code-size, fill: luma(70))
  #set par(justify: false, leading: 0.55em)
  #as-raw(code)
]



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

#import "@preview/pyrunner:0.3.0" as py-runner
#import "listings.typ": listings

/// Run Python code dynamically using pyrunner and return printed output as string.
#let py-exec(code) = {
  let text-code = if type(code) == str { code } else { code.text }
  let runner-code = ```python
import sys, io
_buf = io.StringIO()
_old_out = sys.stdout
sys.stdout = _buf
try:
    exec(code_str)
except Exception:
    pass
finally:
    sys.stdout = _old_out
_buf.getvalue()
```
  py-runner.block(runner-code, globals: (code_str: text-code))
}

/// Interpreter response, shown slanted as in the printed book.
/// Executes the code via pyrunner and displays the output.
#let output(code) = {
  let text-code = if type(code) == str { code } else { code.text }
  let out = py-exec(text-code)
  if out != "" and out != none [
    code-block(fill: white, stroke: luma(235))[
      #set text(font: code-font, size: code-size, style: "oblique")
      #set par(justify: false, leading: 0.55em)
      #raw(out)
    ]
  ]
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



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
// transcript against a real, persistent python3 session and publishes the
// captured output through the Calepin store; the store then hands the output
// string to the book's own listings engine, which typesets it with the same
// machinery as every other code block. Plain `typst compile` cannot execute
// code, so transcripts render empty on a fresh clone and after any prose
// edit, until Calepin runs again. The facade in /.calepin/ is the generated
// runtime once Calepin has run, and a thin shim re-exporting the published
// compatibility package until then.
#import "/.calepin/calepin.typ" as calepin
#import "listings.typ": listings

/// `calepin.store.get`, spelled so it works with both exports: the generated
/// runtime exports `store` as a module (dot call), the compatibility shim as
/// a dictionary (dot calls on dictionary values are an error).
#let _store-get(key, default: "") = {
  let get = if type(calepin.store) == module {
    calepin.store.get
  } else {
    calepin.store.at("get")
  }
  get(key, default: default)
}

// One store key (and engine variable) per transcript, numbered in document
// order. The state advances identically in Calepin's query pass — which turns
// each chunk into an execution spec — and the render pass, so key N names the
// same transcript in both.
#let _transcript-index = state("sicp-transcript", 0)

/// The code Calepin executes for one interpreter transcript: the snippet runs
/// under `exec` (script semantics, exactly as the book narrates), so only
/// `print` output appears — a bare expression such as `486` prints nothing —
/// and a snippet that raises shows nothing either. The captured output is
/// published under `var` for `store-set`; the code is embedded as a JSON
/// literal, which is also a valid Python string literal.
#let transcript-source(code, var) = (
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
  var + " = _b.getvalue()",
).join("\n")

/// The listings configuration of an interpreter response: slanted mono on
/// white inside the book's hairline border, as in the printed book.
#let transcript-options = (
  basicstyle: (font: code-font, size: code-size, style: "oblique"),
  backgroundcolor: white,
  frame: "single",
  framerule: 0.5pt,
  rulecolor: luma(235),
  frameround: "t",
  framesep: 6pt,
  aboveskip: 1em,
  belowskip: 1em,
)

/// Interpreter response, shown slanted as in the printed book. The transcript
/// is emitted as a hidden Calepin chunk that publishes the interpreter's
/// output through the store; `calepin compile` fills it in, and the output is
/// typeset here by listings. Silence — no box at all — when the snippet
/// prints nothing (or nothing is stored yet).
#let output(code) = context {
  let n = _transcript-index.get()
  let step = _transcript-index.update(n + 1)
  let var = "_sicp_t" + str(n)
  let text-code = if type(code) == str { code } else { code.text }
  [
    #step
    #calepin.chunk(
      "python",
      raw(transcript-source(text-code, var), lang: "python", block: true),
      echo: false,
      results: "hide",
      warning: false,
      message: false,
      error: false,
      ..(("store-set": var,)),
    )
    #context {
      let stored = _store-get(var)
      let out = if type(stored) == str { stored } else { "" }
      if out != "" {
        listings(out, options: transcript-options)
      }
    }
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

/// The listings configuration of a prompt line: dim code on a gray plate.
#let prompt-options = (
  basicstyle: (font: code-font, size: code-size, fill: luma(70)),
  backgroundcolor: luma(242),
  frame: "none",
  aboveskip: 1em,
  belowskip: 1em,
)

/// The prompt line preceding user input in an interpreter transcript.
#let prompt(code) = listings(
  if type(code) == str { code } else { code.text },
  options: prompt-options,
)



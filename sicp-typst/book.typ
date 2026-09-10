// Structure and Interpretation of Computer Programs — Python edition
//
// Build with:   typst compile book.typ
//
// The chapters under content/ are generated from the XML sources of
// https://github.com/source-academy/sicp by tools/convert.py; edit the
// converter (or lib/), not the generated files.

#import "lib/sicp.typ": *

#show: sicp-book

// ── front matter ───────────────────────────────────────────────────────────

#page(numbering: none, align(center + horizon)[
  #text(size: 22pt)[Structure and Interpretation \ of Computer Programs]
  #v(1em)
  #text(size: 14pt)[Python Edition]
  #v(3em)
  #text(size: 13pt)[Harold Abelson and Gerald Jay Sussman]
  #v(0.6em)
  #text(size: 11pt)[adapted to Python by Martin Henz]
  #v(4em)
  #text(size: 9pt)[
    Typeset with Typst from the XML sources of the Source Academy
    #link("https://github.com/source-academy/sicp")[SICP project].
  ]
])

#counter(page).update(1)

#outline(depth: 3, indent: auto)

#include "content/others/02foreword84.typ"
#include "content/others/03prefaces96.typ"

// ── chapters ───────────────────────────────────────────────────────────────

#include "content/chapter1/chapter1.typ"
#include "content/chapter2/chapter2.typ"
#include "content/chapter3/chapter3.typ"
#include "content/chapter4/chapter4.typ"
#include "content/chapter5/chapter5.typ"

// ── back matter ────────────────────────────────────────────────────────────

#include "content/others/97references97.typ"

#index-page()

#exercises-page()

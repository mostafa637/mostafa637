// Page setup, headings and the structural elements of the book.

#import "code.typ": show-interpreter-outputs
#import "index.typ": make-index
#import "exercise.typ": exercise-counter, exercise-style, list-of-exercises

#let serif = ("Libertinus Serif", "TeX Gyre Termes", "Times New Roman")
#let sans = ("Libertinus Sans", "TeX Gyre Heros", "Helvetica")

/// Document-wide styling. Wrap the whole book in this.
// ── Reference rules, one per kind of target ─────────────────────────────────
//
// `show ref.where(element: ...)` cannot be used here. `element` is a *derived*
// field: it is resolved after the reference is matched, so `.where()` accepts
// it syntactically and then never matches -- silently, with no error and no
// warning. (An invented field name is rejected outright, which is how one can
// tell the two cases apart.) The only constructor field `ref` can be matched
// on is `target`, i.e. one specific label.
//
// So the dispatch is done by hand instead, with one small named rule per kind
// of target and a table mapping to it. Adding a kind means writing a rule and
// adding one row -- the per-type separation, without the silent failure.

/// Reference to an anchor: it carries no number, so link to its page.
#let ref-anchor(it, e) = link(
  e.location(),
  context [page #counter(page).at(e.location()).first()],
)

/// Reference to a figure: `<chapter>.<n within the chapter>`.
///
/// Both counters are read `.at()` the figure's own location, never through a
/// bare `context` at the reference site -- otherwise a pointer from a later
/// chapter would print the chapter it sits in, not the one the figure is in.
#let ref-figure(it, e) = {
  let loc = e.location()
  let chap = counter(heading).at(loc).at(0, default: 0)
  let n = counter(figure.where(kind: image)).at(loc).first()
  link(loc, numbering("1.1", chap, n))
}

/// A `figure` covers several unrelated things, told apart by `kind`.
///
/// A plain array of pairs, not a dictionary: a `kind` may be a string
/// ("anchor") or an element function (`image`), and only `==` compares those
/// two reliably. Dictionary keys would force everything through `repr`, which
/// renders a string *with* its quotes and an element without -- an easy way to
/// build a table that never matches.
#let ref-figure-rules = (
  ("anchor", ref-anchor),
  (image, ref-figure),
)

#let ref-dispatch(it) = {
  let e = it.element
  if e == none { return it }
  if e.func() == figure {
    for (kind, rule) in ref-figure-rules {
      if e.kind == kind { return rule(it, e) }
    }
  }
  it  // headings and exercises already number themselves correctly
}

#let sicp-book(
  title: "Structure and Interpretation of Computer Programs",
  edition: "Python Edition",
  authors: "Harold Abelson and Gerald Jay Sussman",
  adapted: "adapted to Python by Martin Henz",
  body,
) = {
  set document(title: title, author: authors)
  set page(
    paper: "us-letter",
    margin: (inside: 2.6cm, outside: 2.2cm, top: 2.4cm, bottom: 2.4cm),
    numbering: "1",
    number-align: center,
  )
  set text(font: serif, size: 10.5pt, lang: "en")
  set par(justify: true, leading: 0.62em, first-line-indent: 1.2em)
  set heading(numbering: none)
  show heading: set text(font: sans)
  show heading.where(level: 1): set text(size: 20pt)
  show heading.where(level: 2): set text(size: 14pt)
  show heading.where(level: 3): set text(size: 11.5pt)
  show heading.where(level: 4): set text(size: 10.5pt)
  set footnote.entry(separator: line(length: 30%, stroke: 0.5pt))
  show footnote.entry: set text(size: 8.5pt)
  set ref(supplement: none)
  show link: set text(fill: rgb("#123a7a"))
  show ref: set text(fill: rgb("#123a7a"))
  show ref: it => ref-dispatch(it)

  show: exercise-style
  show: show-interpreter-outputs

  body
}

// ── Structure ───────────────────────────────────────────────────────────────

/// A numbered chapter. Resets the exercise and footnote counters, as the
/// printed book does.
#let chapter(title, label-name: none) = {
  pagebreak(weak: true, to: "odd")
  counter(footnote).update(0)
  exercise-counter.update(0)
  counter(figure.where(kind: image)).update(0)
  set heading(numbering: "1.1.1")
  let h = heading(level: 1, title)
  if label-name == none { h } else { [#h#label-name] }
  v(1em)
}

#let section(title, label-name: none) = {
  set heading(numbering: "1.1.1")
  let h = heading(level: 2, title)
  if label-name == none { h } else { [#h#label-name] }
  v(1em)
}

#let subsection(title, label-name: none) = {
  set heading(numbering: "1.1.1")
  let h = heading(level: 3, title)
  if label-name == none { h } else { [#h#label-name] }
  v(1em)
}

#let subsubsection(title, label-name: none) = {
  set heading(numbering: "1.1.1")
  let h = heading(level: 4, title)
  if label-name == none { h } else { [#h#label-name] }
  v(1em)
}

/// An unnumbered heading inside a subsection (<SUBHEADING> in the XML).
#let subheading(title) = heading(level: 4, numbering: none, outlined: false, title)

/// A run-in heading one level below `subheading` (<SUBSUBHEADING>).
#let subsubheading(title) = heading(
  level: 5, numbering: none, outlined: false, title,
)

/// A zero-width anchor, for the handful of labels the sources attach to a
/// spot in the prose rather than to a numbered element. Referencing one of
/// these yields a plain page link.
#let anchor(name) = box(width: 0pt, [
  #figure(kind: "anchor", supplement: none, outlined: false, none)#name
])

/// Unnumbered front/back matter chapter (<MATTER>, <REFERENCES>).
#let matter(title, label-name: none) = {
  pagebreak(weak: true, to: "odd")
  counter(footnote).update(0)
  let h = heading(level: 1, numbering: none, title)
  if label-name == none { h } else { [#h#label-name] }
  v(1em)
}

#let matter-section(title) = heading(level: 2, numbering: none, title)

// ── Text elements ───────────────────────────────────────────────────────────

/// Chapter-opening epigraph with its attribution.
#let epigraph(body, author: none, title: none, date: none) = block(
  width: 100%,
  inset: (left: 10%, top: 0.6em, bottom: 1.6em),
  {
    set par(justify: true, first-line-indent: 0pt)
    set text(size: 9.5pt, style: "italic")
    body
    if author != none or title != none {
      linebreak()
      set text(style: "normal")
      align(right)[
        —#{ if author != none { author } }#{
          if title != none [, #emph(title)]
        }#{ if date != none [ (#date)] }
      ]
    }
  },
)

#let blockquote(body) = block(
  width: 100%,
  inset: (left: 1.6em, right: 1.6em),
  spacing: 1em,
  {
    set par(justify: true, first-line-indent: 0pt)
    body
  },
)

// "1.5" — chapter number from the heading counter, then the figure number,
// matching the printed book (and the exercise numbering above).
#let figure-numbering(..n) = context {
  let chap = counter(heading).get().at(0, default: 0)
  numbering("1.1", chap, ..n.pos())
}

/// A figure with a caption and, usually, a label.
#let sicp-figure(body, caption: none, label-name: none) = {
  let f = figure(
    body,
    caption: caption,
    kind: image,
    supplement: [Figure],
    numbering: figure-numbering,
  )
  if label-name == none { f } else { [#f#label-name] }
}

/// Table rendered as a plain grid, as the sources use them for layout.
#let sicp-table(columns: 2, ..cells) = block(
  width: 100%,
  inset: (left: 1.2em),
  spacing: 1em,
  {
    set par(justify: false, first-line-indent: 0pt)
    grid(
      columns: columns,
      column-gutter: 1.4em,
      row-gutter: 0.6em,
      ..cells,
    )
  },
)

/// Back matter: the index.
#let index-page(title: "Index") = {
  pagebreak(weak: true, to: "odd")
  heading(level: 1, numbering: none, title)
  block(inset: (bottom: 0.8em))[
    #set text(size: 9pt)
    Page numbers for Python declarations are in #emph[italics].
  ]
  make-index()
}

/// Back matter: the list of exercises.
#let exercises-page(title: "List of Exercises") = {
  pagebreak(weak: true, to: "odd")
  heading(level: 1, numbering: none, title)
  columns(2, gutter: 1.4em, list-of-exercises())
}

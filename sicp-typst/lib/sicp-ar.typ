// Arabic RTL book template for SICP.
// Re-exports everything from the English lib, plus Arabic-specific helpers.
//
//   #import "../lib/sicp-ar.typ": *
//

#import "book.typ": *
#import "code.typ": *
#import "exercise.typ": *
#import "index.typ": *

// ── Arabic fonts ────────────────────────────────────────────────────────────
// Primary Arabic serif + fallback to English serif for Latin/math glyphs.
#let ar-serif = ("Amiri", "Noto Naskh Arabic", "Scheherazade New",
                 "Libertinus Serif", "TeX Gyre Termes")
#let ar-sans  = ("Amiri", "Noto Naskh Arabic",
                 "Libertinus Sans", "TeX Gyre Heros")

// ── Helper: wrap English (LTR) text inside Arabic (RTL) context ─────────────
/// Use `#en[English text here]` or `#en("inline")` to embed LTR English
/// inside the Arabic RTL flow. Keeps the same font size and color.
#let en(body) = text(dir: ltr, lang: "en", body)

// ── Arabic exercise styling ─────────────────────────────────────────────────
// Same as English but with Arabic label "تمرين" instead of "Exercise".
#let exercise-style-ar(body) = {
  show figure.where(kind: "exercise"): it => block(
    width: 100%,
    breakable: true,
    spacing: 1.2em,
    {
      set align(right)
      set par(justify: true, first-line-indent: 0pt)
      box[*تمرين #context it.counter.display(it.numbering)*]
      h(0.5em)
      it.body
    },
  )
  body
}

// ── Arabic-aware sicp-book ──────────────────────────────────────────────────
#let sicp-book-ar(
  title: "بنية وتفسير برامج الحاسوب",
  edition: "نسخة بايثون",
  authors: "هارولد أبلسون وجيرالد جاي سسمان",
  adapted: "مُعَدَّل إلى بايثون بواسطة مارتن هنز",
  body,
) = {
  set document(title: title, author: authors)
  set page(
    paper: "us-letter",
    margin: (inside: 2.6cm, outside: 2.2cm, top: 2.4cm, bottom: 2.4cm),
    numbering: "1",
    number-align: center,
  )

  // ── Global RTL + Arabic text ──────────────────────────────────────────
  set text(font: ar-serif, size: 10.5pt, lang: "ar", dir: rtl)
  set par(justify: true, leading: 0.62em, first-line-indent: 1.2em)

  // ── Headings ──────────────────────────────────────────────────────────
  set heading(numbering: none)
  show heading: set text(font: ar-sans)
  show heading.where(level: 1): set text(size: 20pt)
  show heading.where(level: 2): set text(size: 14pt)
  show heading.where(level: 3): set text(size: 11.5pt)
  show heading.where(level: 4): set text(size: 10.5pt)

  // ── Footnotes ─────────────────────────────────────────────────────────
  set footnote.entry(separator: line(length: 30%, stroke: 0.5pt))
  show footnote.entry: set text(size: 8.5pt)

  // ── References & links ────────────────────────────────────────────────
  set ref(supplement: none)
  show link: set text(fill: rgb("#123a7a"))
  show ref: set text(fill: rgb("#123a7a"))
  show ref: it => ref-dispatch(it)

  // ── Code blocks: force LTR inside RTL context ─────────────────────────
  // All raw/code blocks are LTR (Python code is always left-to-right).
  show raw: set text(dir: ltr)

  // ── Exercises ─────────────────────────────────────────────────────────
  show: exercise-style-ar

  body
}

// ── Arabic figure supplement ────────────────────────────────────────────────
// Override the English "Figure" with Arabic "شكل".
#let sicp-figure-ar(body, caption: none, label-name: none) = {
  let f = figure(
    body,
    caption: caption,
    kind: image,
    supplement: [شكل],
    numbering: figure-numbering,
  )
  if label-name == none { f } else { [#f#label-name] }
}

// ── Arabic index page ───────────────────────────────────────────────────────
#let index-page-ar(title: "الفهرس") = {
  pagebreak(weak: true, to: "odd")
  heading(level: 1, numbering: none, title)
  block(inset: (bottom: 0.8em))[
    #set text(size: 9pt)
    أرقام الصفحات لتصريحات بايثون بخط #emph[مائل].
  ]
  make-index()
}

// ── Arabic exercises page ───────────────────────────────────────────────────
#let list-of-exercises-ar() = context {
  let items = query(figure.where(kind: "exercise"))
  set par(justify: false, first-line-indent: 0pt)
  set text(size: 9pt)
  for it in items {
    let loc = it.location()
    let page = counter(page).at(loc).first()
    let chap = counter(heading).at(loc).at(0, default: 0)
    let num = numbering("1.1", chap, ..it.counter.at(loc))
    block(spacing: 0.5em)[
      #link(loc)[تمرين #num] #box(width: 1fr, repeat(gap: 3pt)[.]) #page
    ]
  }
}

#let exercises-page-ar(title: "قائمة التمارين") = {
  pagebreak(weak: true, to: "odd")
  heading(level: 1, numbering: none, title)
  columns(2, gutter: 1.4em, list-of-exercises-ar())
}

// Exercises, and the "List of Exercises" back-matter page.
//
// An exercise is a `figure` of kind "exercise": that makes it a real,
// referenceable element, so `@ex:foo` links to it. Exercises are numbered
// <chapter>.<n> (1.1, 1.2, ...) like the printed edition — the chapter part
// is read from the heading counter, and `chapter()` in book.typ resets the
// per-chapter part.

#let exercise-kind = "exercise"
#let exercise-counter = counter(figure.where(kind: exercise-kind))

// "1.5" — chapter number from the heading counter, then the exercise number.
#let exercise-numbering(..n) = context {
  let chap = counter(heading).get().at(0, default: 0)
  numbering("1.1", chap, ..n.pos())
}

/// A numbered exercise block.
///
/// - label-name (label, none): attach a label so `@ex:foo` resolves to it
#let exercise(body, label-name: none) = {
  let item = figure(
    kind: exercise-kind,
    supplement: [Exercise],
    numbering: exercise-numbering,
    caption: none,
    placement: none,
    body,
  )
  if label-name == none { item } else { [#item#label-name] }
}

/// Styling for exercise figures: a bold run-in "Exercise n.m" heading.
#let exercise-style(body) = {
  // `figure` centers its content by default, so the alignment has to be
  // reset here for the run-in heading to sit at the left margin.
  show figure.where(kind: exercise-kind): it => block(
    width: 100%,
    breakable: true,
    spacing: 1.2em,
    {
      set align(left)
      set par(justify: true, first-line-indent: 0pt)
      box[*Exercise #context it.counter.display(it.numbering)*]
      h(0.5em)
      it.body
    },
  )
  body
}

/// Back-matter list of every exercise, with its page number.
#let list-of-exercises() = context {
  let items = query(figure.where(kind: exercise-kind))
  set par(justify: false, first-line-indent: 0pt)
  set text(size: 9pt)
  for it in items {
    let loc = it.location()
    let page = counter(page).at(loc).first()
    let chap = counter(heading).at(loc).at(0, default: 0)
    let num = numbering("1.1", chap, ..it.counter.at(loc))
    block(spacing: 0.5em)[
      #link(loc)[Exercise #num] #box(width: 1fr, repeat(gap: 3pt)[.]) #page
    ]
  }
}

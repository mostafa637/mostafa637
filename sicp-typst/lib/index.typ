// Index machinery.
//
// Every <INDEX> entry of the XML sources becomes an `idx(..)` call, which
// drops an invisible metadata marker into the document flow. `make-index()`
// queries all of those markers at the end of the book, groups them by term
// and sub-term, and prints the page numbers it collected.
//
// Page numbers of *declarations* (where a function or name is defined) are
// printed in italics, as in the printed editions of SICP.

#let index-marker = <sicp-index-entry>

/// Record one index entry at the current position in the document.
///
/// - term (str): main index term, e.g. "abstraction"
/// - sub (str, none): sub-entry, e.g. "means of"
/// - sort (str, none): sort key overriding `term`
/// - decl (bool): true when this position *declares* the indexed name
/// - see (str, none): "see ..." cross reference instead of a page number
/// - see-also (str, none): "see also ..." cross reference
#let idx(
  term,
  sub: none,
  sort: none,
  decl: false,
  see: none,
  see-also: none,
) = [#metadata((
    term: term,
    sub: sub,
    sort: if sort == none { term } else { sort },
    decl: decl,
    see: see,
    see-also: see-also,
  ))#index-marker]

// Sort keys are compared case-insensitively and ignoring leading punctuation,
// so that `_pi_`, "Pi" and "pi" end up next to each other.
#let sort-key(s) = lower(s).replace(regex("[^0-9a-z ]"), "")

#let fmt-pages(pages) = {
  let seen = ()
  let parts = ()
  for p in pages {
    let key = (p.page, p.decl)
    if key in seen { continue }
    seen.push(key)
    parts.push(if p.decl { emph[#p.page] } else [#p.page])
  }
  parts.join(", ")
}

/// Collect every recorded entry and typeset the book index.
#let make-index(cols: 2) = context {
  let entries = query(index-marker)

  // term -> (pages: (..), subs: (sub -> (..)), see: .., see-also: ..)
  let terms = (:)
  for e in entries {
    let v = e.value
    let page = counter(page).at(e.location()).first()
    let key = sort-key(v.sort)
    let rec = terms.at(
      key,
      default: (term: v.term, pages: (), subs: (:), see: (), see-also: ()),
    )

    if v.see != none and v.see not in rec.see { rec.see.push(v.see) }
    if v.see-also != none and v.see-also not in rec.see-also {
      rec.see-also.push(v.see-also)
    }

    if v.see == none and v.see-also == none {
      if v.sub == none {
        rec.pages.push((page: page, decl: v.decl))
      } else {
        let sk = sort-key(v.sub)
        let sub = rec.subs.at(sk, default: (term: v.sub, pages: ()))
        sub.pages.push((page: page, decl: v.decl))
        rec.subs.insert(sk, sub)
      }
    }
    terms.insert(key, rec)
  }

  set text(size: 8.2pt)
  set par(justify: false, first-line-indent: 0pt, leading: 0.45em)

  columns(cols, gutter: 1.2em)[
    #for key in terms.keys().sorted() {
      let rec = terms.at(key)
      block(inset: (left: 1em), outset: 0pt, spacing: 0.35em)[
        #set par(hanging-indent: 1em)
        #rec.term#{
          if rec.pages.len() > 0 [, #fmt-pages(rec.pages)]
        }#{
          for s in rec.see [, #emph[see] #s]
        }#{
          for s in rec.see-also [, #emph[see also] #s]
        }
        #for sk in rec.subs.keys().sorted() {
          let sub = rec.subs.at(sk)
          linebreak()
          h(1em)
          sub.term
          if sub.pages.len() > 0 [, #fmt-pages(sub.pages)]
        }
      ]
    }
  ]
}

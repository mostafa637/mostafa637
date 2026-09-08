// fancyhdr.typ — a Typst-native page-style toolkit inspired by fancyhdr 5.x.
//
// The public API is deliberately pure: build a style value with `fancy-style`,
// transform it with `fancy-head`/`fancy-foot` and related functions, then apply
// it with `fancy-apply`. This avoids hidden global mutation and makes style
// snapshots, nesting, and per-section layouts predictable in Typst.
//
// Public API quick reference:
//   fancy-style(...)                 create a pure page-style value.
//   fancy-head/foot/hf(style, places, body)
//                                   set fields using E/O and L/C/R/H/F selectors.
//   fancy-head-width/...             set field widths; `fixed: true` is the
//                                   starred, definition-time counterpart.
//   fancy-head-offset/...            extend the computed headwidth at a side.
//   fancy-apply(style, body, ...)    install page header/footer and conditions.
//   fancy-page-style-scope(name, body)
//                                   select one named style for the page containing
//                                   the marker, the Typst-native thispagestyle form.
//   fancy-heading-marks(body, ...)   emit chapter/section marks automatically in
//                                   a scoped show-rule environment.
//   direction: "rtl"                 interpret L/R as logical start/end (L is
//                                   physical right, R physical left).
//   overflow: "error"|"ignore"      reject or deliberately suppress height checks.
//                                   `compat-v3: true` selects adaptive page space;
//                                   `nocheck: true` selects ignore.
//
#let _parities = ("odd", "even")
#let _sides = ("left", "center", "right")
#let _kinds = ("header", "footer")
#let _directions = ("ltr", "rtl")
#let _direction-valid(value) = value == "ltr" or value == "rtl"
#let _logical-to-physical(side, direction) = {
  if direction == "rtl" and side == "left" { "right" }
  else if direction == "rtl" and side == "right" { "left" }
  else { side }
}

#let _field-key(parity, kind, side) = parity + "_" + kind + "_" + side
#let _width-key(parity, kind, side) = _field-key(parity, kind, side) + "_width"
#let _align-key(parity, kind, side) = _field-key(parity, kind, side) + "_align"
#let _offset-key(parity, kind, side) = parity + "_" + kind + "_" + side + "_offset"
#let _empty-field = (fancyhdr_empty_field: true)
#let _is-empty-field(value) = value == none or (type(value) == dictionary and value.at("fancyhdr_empty_field", default: false))

#let _tokens(places) = {
  if type(places) == str {
    let cleaned = places.trim()
if cleaned == "" { ("",) } else { cleaned.split(",").map(token => token.trim()) }
  } else if type(places) == array {
    places.map(token => token)
  } else {
panic("fancyhdr:
places must be a selector string or an array of selector strings")
  }
}

#let _has(token, upper, lower) = token.contains(upper) or token.contains(lower)

#let _parity-list(token) = {
  if _has(token, "E", "e") and not _has(token, "O", "o") {
    ("even",)
  } else if _has(token, "O", "o") and not _has(token, "E", "e") {
    ("odd",)
  } else {
    _parities
  }
}

#let _side-list(token) = {
  let result = ()
if not _has(token, "L", "l") and not _has(token, "C", "c") and not _has(token, "R", "r") {
    _sides
  } else {
    if _has(token, "L", "l") { result.push("left") }
    if _has(token, "C", "c") { result.push("center") }
    if _has(token, "R", "r") { result.push("right") }
    result
  }
}

#let _kind-list(token, default-kind) = {
  if _has(token, "H", "h") and _has(token, "F", "f") {
    _kinds
  } else if _has(token, "H", "h") {
    ("header",)
  } else if _has(token, "F", "f") {
    ("footer",)
  } else if default-kind == none {
    _kinds
  } else {
    (default-kind,)
  }
}

#let _default-mark-query(side, which) = context {
  let current-page = counter(page).get().first()
  let items = query(metadata).filter(item => {
    let value = item.value
type(value) == dictionary and value.at("fancyhdr_kind", default:
none) == "standard" and value.at(side, default: none) != none
  })
let current = items.filter(item => counter(page).at(item.location()).first() == current-page)
let prior = items.filter(item => counter(page).at(item.location()).first() < current-page)
  let chosen = if current.len() > 0 {
    if which == "first" { current.first() } else { current.last() }
  } else if prior.len() > 0 {
    prior.last()
  } else {
    none
  }
  if chosen == none { none } else { chosen.value.at(side, default: none) }
}

#let _clear-all-fields(style) = {
  let out = style
  for parity in _parities {
    for kind in _kinds {
      for side in _sides {
        out.insert(_field-key(parity, kind, side), none)
      }
    }
  }
  out
}

#let _default-style = (
  // fancyhdr's default `fancy` style: the standard marks in the sides and
  // the page number in the footer center. The content functions are resolved
  // in page context when the style is rendered.
odd_header_left: conditions => if conditions.at("two-sided", default:
false) { _default-mark-query("left", "last") } else { _default-mark-query("right", "first") },
  odd_header_center: none,
odd_header_right: conditions => if conditions.at("two-sided", default:
false) { _default-mark-query("right", "first") } else { _default-mark-query("left", "last") },
  even_header_left: conditions => _default-mark-query("right", "first"),
  even_header_center: none,
  even_header_right: conditions => _default-mark-query("left", "last"),
  odd_footer_left: none,
  odd_footer_center: conditions => context counter(page).display("1"),
  odd_footer_right: none,
  even_footer_left: none,
  even_footer_center: conditions => context counter(page).display("1"),
  even_footer_right: none,

  odd_header_left_width: auto,
  odd_header_center_width: auto,
  odd_header_right_width: auto,
  even_header_left_width: auto,
  even_header_center_width: auto,
  even_header_right_width: auto,
  odd_footer_left_width: auto,
  odd_footer_center_width: auto,
  odd_footer_right_width: auto,
  even_footer_left_width: auto,
  even_footer_center_width: auto,
  even_footer_right_width: auto,

  odd_header_left_align: auto,
  odd_header_center_align: auto,
  odd_header_right_align: auto,
  even_header_left_align: auto,
  even_header_center_align: auto,
  even_header_right_align: auto,
  odd_footer_left_align: auto,
  odd_footer_center_align: auto,
  odd_footer_right_align: auto,
  even_footer_left_align: auto,
  even_footer_center_align: auto,
  even_footer_right_align: auto,

  odd_header_left_offset: 0pt,
  odd_header_right_offset: 0pt,
  even_header_left_offset: 0pt,
  even_header_right_offset: 0pt,
  odd_footer_left_offset: 0pt,
  odd_footer_right_offset: 0pt,
  even_footer_left_offset: 0pt,
  even_footer_right_offset: 0pt,

  two_sided: false,
  headwidth: auto,
  headwidth_delta: 0pt,
  footer_align: auto,
  nocheck: false,
  compat_v3: false,
  option_headings: false,
  option_myheadings: false,
  head_rule: (width: 0.4pt, stroke: auto, body: none),
  foot_rule: (width: 0pt, stroke: auto, body: none),
  plain_head_rule: (width: 0pt, stroke: auto, body: none),
  plain_foot_rule: (width: 0pt, stroke: auto, body: none),
  // fancyhdr defaults: headruleskip = 0pt and footruleskip = .3
  // normalbaselineskip. `em` is the closest context-relative Typst length.
  head_skip: 0pt,
  foot_skip: 0.3em,
  head_init: none,
  foot_init: none,
  header_ascent: 30%,
  footer_descent: 30%,
  direction: "ltr",
  overflow: "error",
  hooks: (
    before: (),
    after: (),
    head_begin: (),
    head_end: (),
    foot_begin: (),
    foot_end: (),
  ),
)

// --------------------------------------------------------------------------
// Internal rendering helpers. They are defined before fancy-apply because
// Typst resolves references when the function body is created.
// --------------------------------------------------------------------------

#let _is-justified(requested) = requested != auto and (requested.contains("j") or requested.contains("J"))
#let _width-alignment-valid(requested) = {
  if requested == auto or requested == "" { true }
  else if type(requested) != str { false }
  else if requested.len() == 1 {
    let letter = requested.first(default: "")
letter == "T" or letter == "t" or letter == "c" or letter == "C" or letter == "b" or letter == "B" or letter == "-" or letter == "l" or letter == "L" or letter == "r" or letter == "R" or letter == "j" or letter == "J"
  } else if requested.len() == 2 {
    let vertical = requested.first(default: "")
    let horizontal = requested.at(1, default: "")
(vertical == "T" or vertical == "t" or vertical == "c" or vertical == "b" or vertical == "B" or vertical == "-") and (horizontal == "l" or horizontal == "L" or horizontal == "c" or horizontal == "C" or horizontal == "r" or horizontal == "R" or horizontal == "j" or horizontal == "J")
  } else { false }
}

#let _horizontal-align(requested, default) = {
  if requested == auto or requested == "" {
    default
  } else if _is-justified(requested) {
    left
  } else if requested.len() == 1 {
    let letter = requested.first(default: "")
    if letter == "r" or letter == "R" { right }
    else if letter == "l" or letter == "L" { left }
    else { default }
  } else {
    let letter = requested.at(1, default: "")
    if letter == "r" or letter == "R" { right }
    else if letter == "c" or letter == "C" { center }
    else { left }
  }
}

#let _vertical-align(requested, default) = {
  if requested == auto {
    default
  } else {
    let letter = requested.first(default: "")
    if letter == "T" or letter == "t" { top }
    else if letter == "B" or letter == "b" { bottom }
    else if letter == "c" or letter == "C" { horizon }
    else { default }
  }
}

#let _render-field(style, parity, kind, side, conditions: (), row-width: auto, page-width: auto, physical-side: auto) = {
  let content = style.at(_field-key(parity, kind, side))
let content = if type(content) == function { content(conditions) } else { content }
  if _is-empty-field(content) {
    box(width: 0pt)[ ]
  } else {
    let requested = style.at(_align-key(parity, kind, side))
    let display-side = if physical-side == auto { side } else { physical-side }
let default-h = if display-side == "left" { left } else if display-side == "right" { right } else { center }
    let default-v = if kind == "header" { bottom } else { top }
    let width-value = style.at(_width-key(parity, kind, side))
let field-width = if type(width-value) == dictionary and width-value.at("fancyhdr_fixed_width", default:
false) {
      let requested-width = width-value.at("value")
      let base-spec = width-value.at("base", default: auto)
      let base-delta = width-value.at("base-delta", default: 0pt)
let fixed-base = (if base-spec == auto { page-width } else if type(base-spec) == ratio { base-spec * page-width } else { base-spec }) + base-delta
if type(requested-width) == ratio { requested-width * fixed-base } else { requested-width }
} else if width-value == auto { auto } else if type(width-value) == ratio { width-value * row-width } else { width-value }
let content = if _is-justified(requested) { par(justify:
true)[#content] } else { content }
    let content = align(_horizontal-align(requested, default-h), content)
    if field-width == auto {
      box(baseline: _vertical-align(requested, default-v))[#content]
    } else {
      box(
        width: field-width,
        baseline: _vertical-align(requested, default-v),
      )[#content]
    }
  }
}

#let _render-row(style, parity, kind, conditions: ()) = layout(size => {
  let delta = style.at("headwidth_delta", default: 0pt)
let base-width = (if style.headwidth == auto { size.width } else if type(style.headwidth) == ratio { style.headwidth * size.width } else { style.headwidth }) + delta
let direction = conditions.at("direction", default:
style.at("direction", default: "ltr"))
  let logical-left-offset = style.at(_offset-key(parity, kind, "left"))
  let logical-right-offset = style.at(_offset-key(parity, kind, "right"))
let physical-left-offset = if direction == "rtl" { logical-right-offset } else { logical-left-offset }
let physical-right-offset = if direction == "rtl" { logical-left-offset } else { logical-right-offset }
  let logical-left = _logical-to-physical("left", direction)
  let logical-center = _logical-to-physical("center", direction)
  let logical-right = _logical-to-physical("right", direction)
let left-field = _render-field(style, parity, kind, "left", conditions:
conditions, row-width: base-width, page-width: size.width, physical-side:
logical-left)
let center-field = _render-field(style, parity, kind, "center", conditions:
conditions, row-width: base-width, page-width: size.width, physical-side:
logical-center)
let right-field = _render-field(style, parity, kind, "right", conditions:
conditions, row-width: base-width, page-width: size.width, physical-side:
logical-right)
let physical-fields = if direction == "rtl" { (right-field, center-field, left-field) } else { (left-field, center-field, right-field) }
  let row = box(width: base-width)[
    #grid(
      columns: (1fr, auto, 1fr),
      gutter: 0pt,
      align: (left, center, right),
      ..physical-fields,
    )
  ]
  let total-width = base-width + physical-left-offset + physical-right-offset
if physical-left-offset == 0pt and physical-right-offset == 0pt and base-width == size.width {
    row
  } else {
    box(width: total-width)[#h(-physical-left-offset)#row]
  }
})

#let _eval(value, conditions) = if type(value) == function { value(conditions) } else { value }

#let _hook-slot(name) = {
  if name == "fancyhdr/before" or name == "before" { "before" }
  else if name == "fancyhdr/after" or name == "after" { "after" }
else if name == "fancyhdr/head/begin" or name == "head-begin" { "head_begin" }
  else if name == "fancyhdr/head/end" or name == "head-end" { "head_end" }
else if name == "fancyhdr/foot/begin" or name == "foot-begin" { "foot_begin" }
  else if name == "fancyhdr/foot/end" or name == "foot-end" { "foot_end" }
else { panic("fancyhdr:
unknown hook; use fancyhdr/before, fancyhdr/after, fancyhdr/head/begin, fancyhdr/head/end, fancyhdr/foot/begin, or fancyhdr/foot/end") }
}
#let _hook-values(style, name, conditions) = {
  let slot = _hook-slot(name)
let hooks = style.at("hooks", default:
_default-style.hooks).at(slot, default: ())
  let values = ()
  for hook in hooks {
    let value = _eval(hook, conditions)
    if value != none { values.push(value) }
  }
  values
}
#let _hooks-content(style, name, conditions) = {
  let values = _hook-values(style, name, conditions)
if values.len() == 0 { [] } else { stack(dir: ttb, spacing:
0pt, ..values) }
}

#let _resolved-space-limit(limit, page-height) = {
  if type(limit) == length { limit }
else if type(limit) == ratio and page-height != auto { limit * page-height }
  else { none }
}
#let _guard-page-box(content, kind, limit, mode, page-number, page-height: auto) = layout(size => {
  let measured = measure(content, width: size.width).height
  let resolved-limit = _resolved-space-limit(limit, page-height)
let overflowing = resolved-limit != none and measured > resolved-limit + 0.5pt
  if overflowing {
    if mode == "error" {
panic("fancyhdr:
" + kind + " is too tall (" + repr(measured) + "); increase the reserved page space or use nocheck/compat-v3")
    } else if mode == "expand" {
metadata((fancyhdr_kind: "overflow", box: kind, page:
page-number, measured: measured, limit: resolved-limit))
      box(height: measured, clip: false)[#content]
    } else {
      content
    }
  } else {
    content
  }
})

#let fancy-overflow-events(page-number: auto) = context {
  query(metadata).filter(item => {
    let value = item.value
type(value) == dictionary and value.at("fancyhdr_kind", default:
none) == "overflow" and (page-number == auto or value.at("page", default:
none) == page-number)
  }).map(item => item.value)
}

#let _automatic-heading-mark(conditions) = context {
  let current-page = counter(page).get().first()
  let all-headings = query(heading.where(level: 1))
let on-page = all-headings.filter(item => counter(page).at(item.location()).first() == current-page)
let prior = all-headings.filter(item => counter(page).at(item.location()).first() < current-page)
let chosen = if on-page.len() > 0 { on-page.last() } else if prior.len() > 0 { prior.last() } else { none }
  if chosen == none { none } else { chosen.body }
}

#let _items-on-page(selector, page-number) = query(selector).filter(item => counter(page).at(item.location()).first() == page-number)
#let _figure-placement-name(item) = if item.placement == top { "top" } else if item.placement == bottom { "bottom" } else { "auto" }
#let _auto-page-condition(kind, page-number) = {
  if kind == "footnote" {
    _items-on-page(footnote, page-number).len() > 0
  } else if kind == "top-float" or kind == "bottom-float" {
    let wanted = if kind == "top-float" { "top" } else { "bottom" }
_items-on-page(figure, page-number).filter(item => _figure-placement-name(item) == wanted).len() > 0
  } else if kind == "float-page" {
    _items-on-page(metadata, page-number).filter(item => {
      let value = item.value
type(value) == dictionary and value.at("fancyhdr_kind", default:
none) == "float-page"
    }).len() > 0
  } else {
    false
  }
}
#let _resolve-page-condition(value, kind, page-number) = if value == auto { _auto-page-condition(kind, page-number) } else { value }

#let fancy-float-page-mark() = metadata((fancyhdr_kind: "float-page"))
#let fancy-float-page = fancy-float-page-mark
#let fancy-mark-float-page = fancy-float-page-mark

#let _effective-style(style) = {
  let page-number = conditions => context counter(page).display("1")
let out = if style.option_headings or style.option_myheadings { _clear-all-fields(style) } else { style }
  if style.option_headings {
    out.insert("odd_header_left", _default-mark-query("right", "first"))
    out.insert("odd_header_right", page-number)
    out.insert("even_header_left", page-number)
    out.insert("even_header_right", _default-mark-query("left", "last"))
    out.insert("head_rule", (width: 0.4pt, stroke: auto, body: none))
  } else if style.option_myheadings {
    out.insert("odd_header_left", _default-mark-query("right", "first"))
    out.insert("odd_header_right", page-number)
    out.insert("even_header_left", page-number)
    out.insert("even_header_right", _default-mark-query("left", "last"))
    out.insert("head_rule", (width: 0.4pt, stroke: auto, body: none))
  }
if style.at("compat_v3", default:
false) and not style.at("nocheck", default:
false) and style.at("overflow", default: "error") == "error" {
    out.insert("overflow", "expand")
  }
  out
}

#let _page-selection(fallback, page-number) = {
  let candidates = query(metadata).filter(item => {
    let value = item.value
    if type(value) != dictionary { false } else {
      let kind = value.at("fancyhdr_kind", default: none)
      let marker-page = counter(page).at(item.location()).first()
(kind == "page-style" and marker-page == page-number) or (kind == "page-style-switch" and marker-page <= page-number)
    }
  })
  if candidates.len() == 0 {
    (style: fallback, page-style: none, suppress: false)
  } else {
    let selected = candidates.last().value
(style: selected.at("style", default: fallback), page-style:
selected.at("page-style", default: none), suppress:
selected.at("suppress", default: false))
  }
}
#let _style-on-page(fallback, page-number) = _page-selection(fallback, page-number).style
#let _render-style-for-page(style, suppress) = if suppress { _clear-all-fields(style) } else { style }

#let fancy-page-style-scope(style, body, page-style: "fancy") = [
#metadata((fancyhdr_kind: "page-style", style: style, page-style:
page-style, suppress: page-style == "plain"))
  #body
]
#let fancy-this-page-style = fancy-page-style-scope

#let fancy-page-style-switch(style, body, page-style: "fancy") = [
#metadata((fancyhdr_kind: "page-style-switch", style: style, page-style:
page-style, suppress: page-style == "plain"))
  #body
]
#let pagestyle = fancy-page-style-switch
#let fancy-pagestyle = fancy-page-style-switch

#let fancy-float-page-style(style, body, page-style: "float") = [
  #metadata((fancyhdr_kind: "float-page"))
  #fancy-page-style-scope(style, body, page-style: page-style)
]
#let floatpagestyle = fancy-float-page-style

#let _render-rule(style, kind, conditions: ()) = layout(size => {
  let is-plain = conditions.at("plain", default: false)
  let rule = if is-plain {
if kind == "header" { style.plain_head_rule } else { style.plain_foot_rule }
  } else if kind == "header" { style.head_rule } else { style.foot_rule }
  let width = _eval(rule.width, conditions)
  let stroke = _eval(rule.stroke, conditions)
  let body = _eval(rule.body, conditions)
  let delta = style.at("headwidth_delta", default: 0pt)
let base-width = (if style.headwidth == auto { size.width } else if type(style.headwidth) == ratio { style.headwidth * size.width } else { style.headwidth }) + delta
let direction = conditions.at("direction", default:
style.at("direction", default: "ltr"))
let logical-left-offset = style.at(_offset-key(conditions.at("parity", default:
"odd"), kind, "left"))
let logical-right-offset = style.at(_offset-key(conditions.at("parity", default:
"odd"), kind, "right"))
let physical-left-offset = if direction == "rtl" { logical-right-offset } else { logical-left-offset }
let physical-right-offset = if direction == "rtl" { logical-left-offset } else { logical-right-offset }
  let total-width = base-width + physical-left-offset + physical-right-offset
  if body != none {
    box(width: total-width)[#h(-physical-left-offset)#body]
  } else if width == 0pt {
    none
  } else {
box(width: total-width)[#h(-physical-left-offset)#line(length:
total-width, stroke: if stroke == auto { width } else { stroke })]
  }
})

#let _render-header(style, parity, conditions: ()) = {
  let init = _eval(style.head_init, conditions)
  let content = [
    #_hooks-content(style, "fancyhdr/before", conditions)
    #_hooks-content(style, "fancyhdr/head/begin", conditions)
    #if init != none { init }
    #_render-row(style, parity, "header", conditions: conditions)
    #v(_eval(style.head_skip, conditions))
    #_render-rule(style, "header", conditions: conditions)
    #_hooks-content(style, "fancyhdr/head/end", conditions)
    #_hooks-content(style, "fancyhdr/after", conditions)
  ]
  _guard-page-box(
    content,
    "header",
conditions.at("header-ascent", default: style.at("header_ascent", default:
30%)),
    conditions.at("overflow", default: "error"),
    conditions.at("page", default: 0),
    page-height: conditions.at("page-height", default: auto),
  )
}

#let _render-footer(style, parity, conditions: ()) = {
  let init = _eval(style.foot_init, conditions)
  let footer-align = _eval(style.footer_align, conditions)
  let content = [
    #_hooks-content(style, "fancyhdr/before", conditions)
    #_hooks-content(style, "fancyhdr/foot/begin", conditions)
    #if init != none { init }
    #_render-rule(style, "footer", conditions: conditions)
    #v(_eval(style.foot_skip, conditions))
    #_render-row(style, parity, "footer", conditions: conditions)
    #if footer-align != auto { v(footer-align) }
    #_hooks-content(style, "fancyhdr/foot/end", conditions)
    #_hooks-content(style, "fancyhdr/after", conditions)
  ]
  _guard-page-box(
    content,
    "footer",
conditions.at("footer-descent", default:
style.at("footer_descent", default: 30%)),
    conditions.at("overflow", default: "error"),
    conditions.at("page", default: 0),
    page-height: conditions.at("page-height", default: auto),
  )
}

// --------------------------------------------------------------------------
// Style construction and application
// --------------------------------------------------------------------------

#let fancy-style(
  base: _default-style,
  two-sided: auto,
  headwidth: auto,
  footer-align: auto,
  nocheck: auto,
  compat-v3: auto,
  headings: auto,
  myheadings: auto,
  head-rule: auto,
  foot-rule: auto,
  head-skip: auto,
  foot-skip: auto,
  head-init: auto,
  foot-init: auto,
  header-ascent: auto,
  footer-descent: auto,
  direction: auto,
  overflow: auto,
) = {
  let out = base
  if two-sided != auto { out.insert("two_sided", two-sided) }
  if headwidth != auto { out.insert("headwidth", headwidth) }
  if footer-align != auto { out.insert("footer_align", footer-align) }
  if nocheck != auto { out.insert("nocheck", nocheck) }
  if compat-v3 != auto { out.insert("compat_v3", compat-v3) }
  if headings != auto { out.insert("option_headings", headings) }
  if myheadings != auto { out.insert("option_myheadings", myheadings) }
  if head-rule != auto { out.insert("head_rule", head-rule) }
  if foot-rule != auto { out.insert("foot_rule", foot-rule) }
  if head-skip != auto { out.insert("head_skip", head-skip) }
  if foot-skip != auto { out.insert("foot_skip", foot-skip) }
  if head-init != auto { out.insert("head_init", head-init) }
  if foot-init != auto { out.insert("foot_init", foot-init) }
  if header-ascent != auto { out.insert("header_ascent", header-ascent) }
  if footer-descent != auto { out.insert("footer_descent", footer-descent) }
  if direction != auto { out.insert("direction", direction) }
  if overflow != auto { out.insert("overflow", overflow) }
  out
}

#let fancy-apply(
  style,
  body,
  paper: "a4",
  margin: auto,
  two-sided: auto,
  header-ascent: auto,
  footer-descent: auto,
  direction: auto,
  overflow: auto,
  page-style: "fancy",
  float-page: auto,
  top-float: auto,
  bottom-float: auto,
  footnote: auto,
) = {
  let style = _effective-style(style)
let use-two-sided = if two-sided == auto { style.two_sided } else { two-sided }
let use-direction = if direction == auto { style.at("direction", default:
"ltr") } else { direction }
let use-overflow = if overflow == auto { if style.at("nocheck", default:
false) { "ignore" } else { style.at("overflow", default:
"error") } } else { overflow }
if not _direction-valid(use-direction) { panic("fancyhdr:
direction must be `ltr` or `rtl`") }
if use-overflow != "error" and use-overflow != "ignore" and use-overflow != "expand" { panic("fancyhdr:
overflow must be `error`, `ignore`, or `expand`") }
let conditions = (page-style: page-style, two-sided:
use-two-sided, direction: use-direction, overflow:
use-overflow, float-page: float-page, top-float: top-float, bottom-float:
bottom-float, footnote: footnote)
let use-ascent = if header-ascent == auto { style.header_ascent } else { header-ascent }
let use-descent = if footer-descent == auto { style.footer_descent } else { footer-descent }
  set page(
    paper: paper,
    margin: margin,
    header-ascent: use-ascent,
    footer-descent: use-descent,
    header: context {
      let page-number = counter(page).get().first()
let parity = if use-two-sided and calc.rem(page-number, 2) == 0 { "even" } else { "odd" }
      let selection = _page-selection(style, page-number)
let active-page-style = if selection.page-style == none { page-style } else { selection.page-style }
let active-style = _render-style-for-page(_effective-style(selection.style), selection.suppress)
let active-direction = if direction != auto { use-direction } else { active-style.at("direction", default:
use-direction) }
let active-overflow = if overflow != auto { use-overflow } else if active-style.at("nocheck", default:
false) { "ignore" } else { active-style.at("overflow", default:
use-overflow) }
      let current = conditions
      current.insert("page", page-number)
      current.insert("parity", parity)
      current.insert("plain", active-page-style == "plain")
      current.insert("page-style", active-page-style)
      current.insert("direction", active-direction)
      current.insert("overflow", active-overflow)
      current.insert("header-ascent", use-ascent)
      current.insert("footer-descent", use-descent)
      current.insert("page-height", page.height)
current.insert("float-page", _resolve-page-condition(float-page, "float-page", page-number))
current.insert("top-float", _resolve-page-condition(top-float, "top-float", page-number))
current.insert("bottom-float", _resolve-page-condition(bottom-float, "bottom-float", page-number))
current.insert("footnote", _resolve-page-condition(footnote, "footnote", page-number))
      _render-header(active-style, parity, conditions: current)
    },
    footer: context {
      let page-number = counter(page).get().first()
let parity = if use-two-sided and calc.rem(page-number, 2) == 0 { "even" } else { "odd" }
      let selection = _page-selection(style, page-number)
let active-page-style = if selection.page-style == none { page-style } else { selection.page-style }
let active-style = _render-style-for-page(_effective-style(selection.style), selection.suppress)
let active-direction = if direction != auto { use-direction } else { active-style.at("direction", default:
use-direction) }
let active-overflow = if overflow != auto { use-overflow } else if active-style.at("nocheck", default:
false) { "ignore" } else { active-style.at("overflow", default:
use-overflow) }
      let current = conditions
      current.insert("page", page-number)
      current.insert("parity", parity)
      current.insert("plain", active-page-style == "plain")
      current.insert("page-style", active-page-style)
      current.insert("direction", active-direction)
      current.insert("overflow", active-overflow)
      current.insert("header-ascent", use-ascent)
      current.insert("footer-descent", use-descent)
      current.insert("page-height", page.height)
current.insert("float-page", _resolve-page-condition(float-page, "float-page", page-number))
current.insert("top-float", _resolve-page-condition(top-float, "top-float", page-number))
current.insert("bottom-float", _resolve-page-condition(bottom-float, "bottom-float", page-number))
current.insert("footnote", _resolve-page-condition(footnote, "footnote", page-number))
      _render-footer(active-style, parity, conditions: current)
    },
  )
  body
}

#let fancy-setup(style, body, paper: "a4", margin: auto, two-sided: auto, header-ascent: auto, footer-descent: auto, page-style: "fancy", float-page: auto, top-float: auto, bottom-float: auto, footnote: auto) = fancy-apply(style, body, paper: paper, margin: margin, two-sided: two-sided, header-ascent: header-ascent, footer-descent: footer-descent, page-style: page-style, float-page: float-page, top-float: top-float, bottom-float: bottom-float, footnote: footnote)

#let _selector-valid(token) = {
  if type(token) != str { false }
  else {
    let invalid = token
      .replace(" ", "")
      .replace("\\t", "")
      .replace("\\n", "")
for letter in ("E", "O", "L", "C", "R", "H", "F", "e", "o", "l", "c", "r", "h", "f") {
      invalid = invalid.replace(letter, "")
    }
    invalid.len() == 0
  }
}
#let _validate-selectors(style, places, forbid-center: false) = {
  let raw = _tokens(places)
  let whole-empty = type(places) == str and places.trim() == ""
  for token in raw {
    if token == "" and whole-empty {
      continue
    }
    if not _selector-valid(token) {
panic("fancyhdr:
illegal field selector; allowed letters are E/O/L/C/R/H/F")
    }
    if forbid-center and _has(token, "C", "c") {
      panic("fancyhdr: the C selector is not allowed for offsets")
    }
  }
}
#let _set-fields(style, places, default-kind, content) = {
  _validate-selectors(style, places)
  let out = style
  let stored = if content == none { _empty-field } else { content }
  for token in _tokens(places) {
    for parity in _parity-list(token) {
      for kind in _kind-list(token, default-kind) {
        for side in _side-list(token) {
          out.insert(_field-key(parity, kind, side), stored)
        }
      }
    }
  }
  out
}

#let fancy-head(style, places, content) = _set-fields(style, places, "header", content)
#let fancy-foot(style, places, content) = _set-fields(style, places, "footer", content)
#let fancy-hf(style, places, content) = _set-fields(style, places, none, content)

#let fancy-head-clear(style, places: "") = _set-fields(style, places, "header", none)
#let fancy-foot-clear(style, places: "") = _set-fields(style, places, "footer", none)
#let fancy-hf-clear(style, places: "") = _set-fields(style, places, none, none)

#let _heading-page-number = conditions => context counter(page).display("1")
#let _fancy-heading-style(style, include-marks: true, rule-width: 0.4pt) = {
  let out = fancy-hf-clear(style)
  let out = fancy-head(out, "RO,LE", _heading-page-number)
  let out = if include-marks {
    let marked = fancy-head(out, "LO", _default-mark-query("right", "first"))
    fancy-head(marked, "RE", _default-mark-query("left", "last"))
  } else {
    out
  }
  let out = {
    let updated = out
    updated.insert("head_rule", (width: rule-width, stroke: auto, body: none))
    updated
  }
  fancy-style(base: out, headings: false, myheadings: false)
}
#let fancy-headings-style(style, rule-width: 0.4pt) = _fancy-heading-style(style, include-marks: true, rule-width: rule-width)
#let fancy-myheadings-style(style, rule-width: 0.4pt) = _fancy-heading-style(style, include-marks: true, rule-width: rule-width)

#let _set-widths(style, places, default-kind, width, alignment, fixed: false) = {
  _validate-selectors(style, places)
  if not _width-alignment-valid(alignment) {
    panic("fancyhdr: illegal alignment; use T/t/c/b/B/- with l/c/r/j")
  }
  let out = style
  for token in _tokens(places) {
    for parity in _parity-list(token) {
      for kind in _kind-list(token, default-kind) {
        for side in _side-list(token) {
          let stored-width = if fixed {
(fancyhdr_fixed_width: true, value: width, base:
style.at("headwidth", default: auto), base-delta:
style.at("headwidth_delta", default: 0pt))
          } else { width }
          out.insert(_width-key(parity, kind, side), stored-width)
          out.insert(_align-key(parity, kind, side), alignment)
        }
      }
    }
  }
  out
}

#let fancy-head-width(style, places, width, alignment: auto, fixed: false) = _set-widths(style, places, "header", width, alignment, fixed: fixed)
#let fancy-foot-width(style, places, width, alignment: auto, fixed: false) = _set-widths(style, places, "footer", width, alignment, fixed: fixed)
#let fancy-hf-width(style, places, width, alignment: auto, fixed: false) = _set-widths(style, places, none, width, alignment, fixed: fixed)

#let _set-offsets(style, places, default-kind, amount) = {
  _validate-selectors(style, places, forbid-center: true)
  let out = style
  out.insert("headwidth", auto)
  out.insert("headwidth_delta", 0pt)
  for token in _tokens(places) {
    for parity in _parity-list(token) {
      for kind in _kind-list(token, default-kind) {
        for side in _side-list(token) {
          if side != "center" {
            out.insert(_offset-key(parity, kind, side), amount)
          }
        }
      }
    }
  }
  out
}

#let fancy-head-offset(style, places, amount) = _set-offsets(style, places, "header", amount)
#let fancy-foot-offset(style, places, amount) = _set-offsets(style, places, "footer", amount)
#let fancy-hf-offset(style, places, amount) = _set-offsets(style, places, none, amount)

#let fancy-headwidth(style, width) = {
  let out = style
  out.insert("headwidth", width)
  out.insert("headwidth_delta", 0pt)
  out
}

#let fancy-headwidth-add(style, amount) = {
  let out = style
  let delta = style.at("headwidth_delta", default: 0pt)
  out.insert("headwidth_delta", delta + amount)
  out
}

#let fancy-headwidth-reset(style) = fancy-headwidth(style, auto)

#let _rule-width(style, rule-key, default: 0pt) = style.at(rule-key, default: (width: default, stroke: auto, body: none)).at("width", default: default)
#let fancy-register-get(style, name, default: auto) = {
  if name == "headwidth" { style.at("headwidth", default: default) }
else if name == "headwidth-delta" or name == "headwidth_delta" { style.at("headwidth_delta", default:
default) }
else if name == "headrulewidth" or name == "head-rule-width" { _rule-width(style, "head_rule", default:
default) }
else if name == "footrulewidth" or name == "foot-rule-width" { _rule-width(style, "foot_rule", default:
default) }
else if name == "plainheadrulewidth" or name == "plain-head-rule-width" { _rule-width(style, "plain_head_rule", default:
default) }
else if name == "plainfootrulewidth" or name == "plain-foot-rule-width" { _rule-width(style, "plain_foot_rule", default:
default) }
else if name == "headruleskip" or name == "head-rule-skip" { style.at("head_skip", default:
default) }
else if name == "footruleskip" or name == "foot-rule-skip" { style.at("foot_skip", default:
default) }
else if name == "header-ascent" or name == "header_ascent" { style.at("header_ascent", default:
default) }
else if name == "footer-descent" or name == "footer_descent" { style.at("footer_descent", default:
default) }
else if name == "footer-align" or name == "footer_align" { style.at("footer_align", default:
default) }
  else { default }
}
#let _set-rule-width(style, rule-key, value) = {
  let out = style
let rule = style.at(rule-key, default: (width: 0pt, stroke: auto, body:
none))
  let updated = rule
  updated.insert("width", value)
  out.insert(rule-key, updated)
  out
}
#let fancy-register-set(style, name, value) = {
  if name == "headwidth" { fancy-headwidth(style, value) }
  else if name == "headwidth-delta" or name == "headwidth_delta" {
    let out = style
    out.insert("headwidth_delta", value)
    out
} else if name == "headrulewidth" or name == "head-rule-width" { _set-rule-width(style, "head_rule", value) }
else if name == "footrulewidth" or name == "foot-rule-width" { _set-rule-width(style, "foot_rule", value) }
else if name == "plainheadrulewidth" or name == "plain-head-rule-width" { _set-rule-width(style, "plain_head_rule", value) }
else if name == "plainfootrulewidth" or name == "plain-foot-rule-width" { _set-rule-width(style, "plain_foot_rule", value) }
else if name == "headruleskip" or name == "head-rule-skip" { fancy-head-skip(style, value) }
else if name == "footruleskip" or name == "foot-rule-skip" { fancy-foot-skip(style, value) }
  else if name == "header-ascent" or name == "header_ascent" {
    let out = style
    out.insert("header_ascent", value)
    out
  } else if name == "footer-descent" or name == "footer_descent" {
    let out = style
    out.insert("footer_descent", value)
    out
} else if name == "footer-align" or name == "footer_align" { fancyfootalign(style, value) }
  else { style }
}
#let fancy-register-add(style, name, amount) = {
  if name == "headwidth" {
    fancy-headwidth-add(style, amount)
  } else {
    let current = fancy-register-get(style, name, default: 0pt)
    fancy-register-set(style, name, current + amount)
  }
}
#let fancy-register-reset(style, name) = {
  if name == "headwidth" { fancy-headwidth-reset(style) }
  else {
    let fallback = fancy-register-get(_default-style, name, default: auto)
if fallback == auto { style } else { fancy-register-set(style, name, fallback) }
  }
}
#let fancy-head-rule(style, width, stroke: auto, body: none) = {
  let out = style
  out.insert("head_rule", (width: width, stroke: stroke, body: body))
  out
}

#let fancy-foot-rule(style, width, stroke: auto, body: none) = {
  let out = style
  out.insert("foot_rule", (width: width, stroke: stroke, body: body))
  out
}

#let fancy-head-rule-content(style, body) = fancy-head-rule(style, 0pt, body: body)
#let fancy-foot-rule-content(style, body) = fancy-foot-rule(style, 0pt, body: body)

#let fancy-head-rule-width(style, width) = fancy-head-rule(style, width)
#let fancy-foot-rule-width(style, width) = fancy-foot-rule(style, width)
#let fancy-head-rule-skip(style, amount) = fancy-head-skip(style, amount)
#let fancy-foot-rule-skip(style, amount) = fancy-foot-skip(style, amount)
#let headrule(style, body) = fancy-head-rule-content(style, body)
#let footrule(style, body) = fancy-foot-rule-content(style, body)
#let headrulewidth(style, width) = fancy-head-rule(style, width)
#let footrulewidth(style, width) = fancy-foot-rule(style, width)
#let headruleskip(style, amount) = {
  let out = style
  out.insert("head_skip", amount)
  out
}
#let footruleskip(style, amount) = {
  let out = style
  out.insert("foot_skip", amount)
  out
}

#let fancy-head-skip(style, amount) = {
  let out = style
  out.insert("head_skip", amount)
  out
}

#let fancy-foot-skip(style, amount) = {
  let out = style
  out.insert("foot_skip", amount)
  out
}

#let fancy-head-init(style, body) = {
  let out = style
  out.insert("head_init", body)
  out
}

#let fancy-foot-init(style, body) = {
  let out = style
  out.insert("foot_init", body)
  out
}

#let fancy-hf-init(style, body) = {
  let out = fancy-head-init(style, body)
  fancy-foot-init(out, body)
}

#let fancy-add-hook(style, name, body) = {
  let slot = _hook-slot(name)
  let out = style
  let hook-map = style.at("hooks", default: _default-style.hooks)
  let values = hook-map.at(slot, default: ())
  values.push(body)
  hook-map.insert(slot, values)
  out.insert("hooks", hook-map)
  out
}
#let fancy-hook = fancy-add-hook
#let fancy-hook-reset(style, name) = {
  let slot = _hook-slot(name)
  let out = style
  let hook-map = style.at("hooks", default: _default-style.hooks)
  hook-map.insert(slot, ())
  out.insert("hooks", hook-map)
  out
}

#let fancyfootalign(style, amount) = {
  let out = style
  out.insert("footer_align", amount)
  out
}
#let fancyfootalign-reset(style) = fancyfootalign(style, auto)
#let fancy-footer-align = fancyfootalign
#let fancy-footer-align-reset = fancyfootalign-reset
#let fancyheadinit = fancy-head-init
#let fancyfootinit = fancy-foot-init
#let fancyhfinit = fancy-hf-init

#let fancy-options(style, twoside: auto, nocheck: auto, compat-v3: auto, compatV3: auto, headings: auto, myheadings: auto, direction: auto, overflow: auto) = {
  let compat = if compatV3 != auto { compatV3 } else { compat-v3 }
  fancy-style(
    base: style,
    two-sided: twoside,
    nocheck: nocheck,
    compat-v3: compat,
    headings: headings,
    myheadings: myheadings,
    direction: direction,
    overflow: overflow,
  )
}

#let fancy-open-style(style) = style
#let fancy-closed-style(style) = style
#let fancy-page-style(base, definition: none, closed: false) = {
  if definition == none { base } else { definition(base) }
}
#let fancy-page-style-closed(base, definition: none) = fancy-page-style(base, definition: definition, closed: true)
#let fancy-page-style-assign(style) = style

#let fancy-style-registry(base: _default-style) = (base: base)
#let _style-entry(kind, base, definition) = (__fancy_style_entry: true, kind: kind, base: base, definition: definition)
#let _resolve-style-entry(registry, entry, environment: _default-style) = {
if type(entry) != dictionary or not entry.at("__fancy_style_entry", default:
false) {
    entry
  } else {
    let base = if entry.kind == "open" {
if entry.base == "base" { registry.at("base", default:
environment) } else { _resolve-style-entry(registry, registry.at(entry.base, default:
environment), environment: environment) }
    } else {
if entry.base == "base" { registry.at("base", default:
environment) } else { _resolve-style-entry(registry, registry.at(entry.base, default:
environment), environment: environment) }
    }
    if entry.definition == none { base } else { (entry.definition)(base) }
  }
}
#let fancy-style-define(registry, name, base: "base", definition: none, closed: false) = {
  let out = registry
  if closed {
let parent = if base == "base" { registry.at("base", default:
_default-style) } else { _resolve-style-entry(registry, registry.at(base, default:
_default-style)) }
    let result = if definition == none { parent } else { definition(parent) }
    out.insert(name, result)
  } else {
    out.insert(name, _style-entry("open", base, definition))
  }
  out
}
#let fancy-style-define-closed(registry, name, base: "base", definition: none) = fancy-style-define(registry, name, base: base, definition: definition, closed: true)
#let fancy-style-assign(registry, name, source, environment: _default-style) = {
  let out = registry
out.insert(name, _resolve-style-entry(registry, registry.at(source, default:
environment), environment: environment))
  out
}
#let fancy-style-get(registry, name, environment: _default-style) = _resolve-style-entry(registry, registry.at(name, default: environment), environment: environment)

#let fancypagestyle = fancy-style-define
#let fancypagestyle-closed = fancy-style-define-closed
#let fancypagestyleassign = fancy-style-assign

#let fancy-apply-style(registry, name, body, environment: _default-style, ..args) = fancy-apply(fancy-style-get(registry, name, environment: environment), body, ..args)

#let fancy-style-copy(style) = style
#let fancy-assign-style(style) = style

#let _condition(conditions, key, yes, no) = {
  if conditions.at(key, default: false) { yes } else { no }
}

#let fancy-if-float-page(yes, no, conditions: (float-page: false)) = _condition(conditions, "float-page", yes, no)
#let fancy-if-top-float(yes, no, conditions: (top-float: false)) = _condition(conditions, "top-float", yes, no)
#let fancy-if-bottom-float(yes, no, conditions: (bottom-float: false)) = _condition(conditions, "bottom-float", yes, no)
#let fancy-if-footnote(yes, no, conditions: (footnote: false)) = _condition(conditions, "footnote", yes, no)

#let iffloatpage = fancy-if-float-page
#let iftopfloat = fancy-if-top-float
#let ifbotfloat = fancy-if-bottom-float
#let iffootnote = fancy-if-footnote

#let fancy-page-number(numbering: "1") = context counter(page).display(numbering)
#let fancy-page-of(numbering) = context counter(page).display(numbering, both: true)
#let fancy-nouppercase(body) = body
#let nouppercase = fancy-nouppercase
#let fancy-total-pages() = context counter(page).final().first()
#let thepage = fancy-page-number
#let fancy-default-style() = _default-style
#let fancydefault = fancy-default-style
#let fancy = fancy-style

#let fancy-center(left-content, center-content, right-content, distance: 1em, stretch: 3, width: auto) = layout(size => {
  let available = if width == auto { size.width } else { width }
  let gap-width = measure(box(width: distance)).width
  let left-width = measure(left-content, width: available).width
  let right-width = measure(right-content, width: available).width
  if center-content == none {
    grid(
      columns: (1fr, 1fr),
      gutter: distance,
      left-content,
      right-content,
    )
  } else {
    let center-width = measure(center-content, width: available).width
let limit = available - 2 * (stretch * gap-width + calc.max(left-width, right-width))
    if center-width <= limit {
      grid(
        columns: (1fr, auto, 1fr),
        gutter: distance,
        align: (left, center, right),
        left-content,
        center-content,
        right-content,
      )
    } else {
      grid(
        columns: (auto, 1fr, auto),
        gutter: distance,
        align: (left, center, right),
        left-content,
        center-content,
        right-content,
      )
    }
  }
})

#let _box-alignment-valid(alignment) = {
  if type(alignment) != str { false }
  else if alignment.len() == 1 {
alignment == "T" or alignment == "t" or alignment == "c" or alignment == "C" or alignment == "b" or alignment == "B" or alignment == "l" or alignment == "L" or alignment == "r" or alignment == "R"
  } else if alignment.len() == 2 {
    let vertical = alignment.first(default: "")
    let horizontal = alignment.at(1, default: "")
(vertical == "T" or vertical == "t" or vertical == "c" or vertical == "C" or vertical == "b" or vertical == "B") and (horizontal == "l" or horizontal == "L" or horizontal == "c" or horizontal == "C" or horizontal == "r" or horizontal == "R")
  } else { false }
}
#let _box-alignments(alignment) = {
  if not _box-alignment-valid(alignment) {
panic("fancyhdr-box:
illegal alignment; use T/t/c/b/B with l/c/r, or a single c/l/r")
  }
  let single-center = alignment == "c" or alignment == "C"
let vertical = if single-center or alignment.contains("c") or alignment.contains("C") { horizon }
    else if alignment.contains("T") or alignment.contains("t") { top }
    else if alignment.contains("B") or alignment.contains("b") { bottom }
    else { horizon }
  let horizontal = if single-center { center }
    else if alignment.contains("r") or alignment.contains("R") { right }
    else if alignment.contains("c") or alignment.contains("C") { center }
    else { left }
  (vertical: vertical, horizontal: horizontal)
}

#let _box-line-list(lines) = if type(lines) == array { lines } else { (lines,) }
#let _box-rule(rule) = {
  if rule == true {
    line(length: 100%, stroke: 0.4pt)
  } else if type(rule) == dictionary {
    line(
      length: rule.at("length", default: 100%),
      stroke: rule.at("stroke", default: 0.4pt),
    )
  } else {
    none
  }
}
#let _box-item(item, include-after: true) = {
  let is-dictionary = type(item) == dictionary
let content = if is-dictionary { item.at("content", default:
none) } else { item }
let rule = if is-dictionary { item.at("rule", default:
none) } else { none }
let before = if is-dictionary { item.at("before", default:
0pt) } else { 0pt }
let after = if is-dictionary { item.at("after", default:
0pt) } else { 0pt }
  let rule-content = _box-rule(rule)
  let core = if rule-content == none {
    content
  } else if content == none {
    [#v(before) #rule-content]
  } else {
    [#content #v(before) #rule-content]
  }
  if include-after and after != 0pt { [#core #v(after)] } else { core }
}
#let _box-with-strut(content, enabled) = if enabled {
  [#box(width: 0pt, height: 1em, baseline: 0.7em) #content]
} else { content }
#let fancyhdr-box(lines, alignment: "cl", width: auto, gap: 0pt, strut: false) = {
  let alignments = _box-alignments(alignment)
  let cells = ()
  for line-content in _box-line-list(lines) {
    let row = _box-with-strut(_box-item(line-content), strut)
    cells.push(align(alignments.horizontal, row))
  }
box(width: width, baseline: alignments.vertical)[#stack(dir: ttb, spacing:
gap, ..cells)]
}

#let fancyhdr-box-spaced(lines, alignment: "cl", width: auto, gap: 0pt, strut: false) = {
  let alignments = _box-alignments(alignment)
  let cells = ()
  for item in _box-line-list(lines) {
let after = if type(item) == dictionary { item.at("after", default:
0pt) } else { 0pt }
    let body = _box-item(item, include-after: false)
    let body = if after == 0pt { body } else { [#body #v(after)] }
    let body = _box-with-strut(body, strut)
    cells.push(align(alignments.horizontal, body))
  }
box(width: width, baseline: alignments.vertical)[#stack(dir: ttb, spacing:
gap, ..cells)]
}

#let fancy-layout-measure(body, width: auto) = context layout(size => {
  let target-width = if width == auto { size.width } else { width }
  measure(body, width: target-width).height
})
#let fancy-measure = fancy-layout-measure
#let fancyhdrsettoheight(body, width: auto) = fancy-measure(body, width: width)
#let fancy-measure-field(style, parity: "odd", kind: "header", side: "center", width: auto) = context {
  let value = style.at(_field-key(parity, kind, side), default: none)
let value = if type(value) == function { value((page:
counter(page).get().first(), parity: parity, page-style:
"fancy")) } else { value }
  fancy-measure(value, width: width)
}
#let fancy-measure-header(style, parity: "odd", side: "center", width: auto) = fancy-measure-field(style, parity: parity, kind: "header", side: side, width: width)
#let fancy-measure-footer(style, parity: "odd", side: "center", width: auto) = fancy-measure-field(style, parity: parity, kind: "footer", side: side, width: width)
#let fancyhdrsettoheight-field = fancy-measure-field
#let fancy-hdr-set-to-height = fancyhdrsettoheight

#let fancyplain(plain-value, fancy-value, conditions: (page-style: "fancy")) = if conditions.at("page-style", default: "fancy") == "plain" { plain-value } else { fancy-value }
#let fancy-plain = fancyplain
#let fancy-plain-style(style, head-rule: auto, foot-rule: auto, head-stroke: auto, foot-stroke: auto) = {
  let out = style
if head-rule != auto { out.insert("plain_head_rule", (width:
head-rule, stroke: head-stroke, body: none)) }
if foot-rule != auto { out.insert("plain_foot_rule", (width:
foot-rule, stroke: foot-stroke, body: none)) }
  out
}
#let fancy-plain-rules(style, head-width: auto, foot-width: auto, head-stroke: auto, foot-stroke: auto) = {
  let out = style
if head-width != auto { out.insert("plain_head_rule", (width:
head-width, stroke: head-stroke, body: none)) }
if foot-width != auto { out.insert("plain_foot_rule", (width:
foot-width, stroke: foot-stroke, body: none)) }
  out
}
#let fancy-plain-head-rule-width(style, width, stroke: auto) = fancy-plain-rules(style, head-width: width, head-stroke: stroke)
#let fancy-plain-foot-rule-width(style, width, stroke: auto) = fancy-plain-rules(style, foot-width: width, foot-stroke: stroke)

#let fancyhead = fancy-head
#let fancyfoot = fancy-foot
#let fancyhf = fancy-hf
#let fancyheadclear = fancy-head-clear
#let fancyfootclear = fancy-foot-clear
#let fancyhfclear = fancy-hf-clear
#let fancyheadwidth = fancy-head-width
#let fancyfootwidth = fancy-foot-width
#let fancyhfwidth = fancy-hf-width
#let fancy-head-width-fixed(style, places, width, alignment: auto) = fancy-head-width(style, places, width, alignment: alignment, fixed: true)
#let fancy-foot-width-fixed(style, places, width, alignment: auto) = fancy-foot-width(style, places, width, alignment: alignment, fixed: true)
#let fancy-hf-width-fixed(style, places, width, alignment: auto) = fancy-hf-width(style, places, width, alignment: alignment, fixed: true)
#let fancyheadwidthfixed = fancy-head-width-fixed
#let fancyfootwidthfixed = fancy-foot-width-fixed
#let fancyhfwidthfixed = fancy-hf-width-fixed
#let fancyheadoffset = fancy-head-offset
#let fancyfootoffset = fancy-foot-offset
#let fancyhfoffset = fancy-hf-offset
#let fancycenter = fancy-center
#let fancyplainstyle = fancy-plain-style
#let fancyheadingsstyle = fancy-headings-style
#let fancymyheadingsstyle = fancy-myheadings-style
#let fancyhdrbox = fancyhdr-box
#let fancyhdrbox-spaced = fancyhdr-box-spaced

#let _mark-entry(kind, mark-class: "default", left: none, right: none, value: none) = metadata((
  fancyhdr_kind: kind,
  fancyhdr_class: mark-class,
  left: left,
  right: right,
  value: value,
))

#let _mark-items(kind, mark-class: none) = query(metadata).filter(item => {
  let value = item.value
type(value) == dictionary and value.at("fancyhdr_kind", default:
none) == kind and (mark-class == none or value.at("fancyhdr_class", default:
"default") == mark-class)
})

#let _query-mark(kind, side, which, mark-class: none) = context {
  let current-page = counter(page).get().first()
let items = _mark-items(kind, mark-class:
mark-class).filter(item => item.value.at(side, default: none) != none)
let current = items.filter(item => counter(page).at(item.location()).first() == current-page)
let prior = items.filter(item => counter(page).at(item.location()).first() < current-page)
  let chosen = if which == "top" {
    if prior.len() > 0 { prior.last() } else { none }
  } else if current.len() > 0 {
    if which == "first" { current.first() } else { current.last() }
  } else if prior.len() > 0 {
    prior.last()
  } else {
    none
  }
  if chosen == none { none } else { chosen.value.at(side, default: none) }
}

#let fancy-extramarks(left, right) = _mark-entry("extra", mark-class: "extra", left: left, right: right)
#let fancy-extramarks-left(value) = _mark-entry("extra", mark-class: "extra", left: value)
#let fancy-extramarks-right(value) = _mark-entry("extra", mark-class: "extra", right: value)
#let fancy-markboth(left, right) = _mark-entry("standard", mark-class: "standard", left: left, right: if right == none { [] } else { right })
#let fancy-markright(right) = _mark-entry("standard", mark-class: "standard", right: right)

#let fancy-mark-class(mark-class, left, right) = _mark-entry("named", mark-class: mark-class, left: left, right: right)
#let fancy-mark-class-left(mark-class, value) = _mark-entry("named", mark-class: mark-class, left: value)
#let fancy-mark-class-right(mark-class, value) = _mark-entry("named", mark-class: mark-class, right: value)
#let fancy-new-mark-class(mark-class) = mark-class
#let fancy-insert-mark(mark-class, value) = _mark-entry("named", mark-class: mark-class, value: value)

#let fancy-chapter-mark(value) = fancy-markboth(value, none)
#let fancy-section-mark(value) = fancy-markright(value)
#let fancy-subsection-mark(value) = fancy-markright(value)
#let chaptermark = fancy-chapter-mark
#let sectionmark = fancy-section-mark
#let subsectionmark = fancy-subsection-mark

#let fancy-heading-marks(body, chapter-level: 1, section-level: 2, subsection-level: 3, transform: none) = [
  #let title(value) = if transform == none { value } else { transform(value) }
  #show heading.where(level: chapter-level): it => [#fancy-chapter-mark(title(it.body)) #it]
  #show heading.where(level: section-level): it => [#fancy-section-mark(title(it.body)) #it]
  #show heading.where(level: subsection-level): it => [#fancy-subsection-mark(title(it.body)) #it]
  #body
]
#let fancy-auto-heading-marks = fancy-heading-marks

#let fancy-first-left-mark() = _query-mark("standard", "left", "first", mark-class: "standard")
#let fancy-last-left-mark() = _default-mark-query("left", "last")
#let fancy-first-right-mark() = _default-mark-query("right", "first")
#let fancy-last-right-mark() = _query-mark("standard", "right", "last", mark-class: "standard")
#let fancy-top-left-x-mark() = _query-mark("extra", "left", "top", mark-class: "extra")
#let fancy-first-left-x-mark() = _query-mark("extra", "left", "first", mark-class: "extra")
#let fancy-last-left-x-mark() = _query-mark("extra", "left", "last", mark-class: "extra")
#let fancy-top-right-x-mark() = _query-mark("extra", "right", "top", mark-class: "extra")
#let fancy-first-right-x-mark() = _query-mark("extra", "right", "first", mark-class: "extra")
#let fancy-last-right-x-mark() = _query-mark("extra", "right", "last", mark-class: "extra")

#let fancy-firstmark() = fancy-first-right-mark()
#let fancy-topmark() = _query-mark("standard", "right", "top", mark-class: "standard")
#let fancy-lastmark() = fancy-last-right-mark()
#let fancy-firstxmark() = fancy-first-right-x-mark()
#let fancy-topxmark() = fancy-top-right-x-mark()
#let fancy-lastxmark() = fancy-last-right-x-mark()
#let firstmark = fancy-firstmark
#let topmark = fancy-topmark
#let lastmark = fancy-lastmark
#let firstxmark = fancy-firstxmark
#let topxmark = fancy-topxmark
#let lastxmark = fancy-lastxmark
#let FirstMark = fancy-firstmark
#let TopMark = fancy-topmark
#let LastMark = fancy-lastmark
#let FirstXMark = fancy-firstxmark
#let TopXMark = fancy-topxmark
#let LastXMark = fancy-lastxmark

#let fancy-first-class-left(mark-class) = _query-mark("named", "left", "first", mark-class: mark-class)
#let fancy-last-class-left(mark-class) = _query-mark("named", "left", "last", mark-class: mark-class)
#let fancy-top-class-left(mark-class) = _query-mark("named", "left", "top", mark-class: mark-class)
#let fancy-first-class-right(mark-class) = _query-mark("named", "right", "first", mark-class: mark-class)
#let fancy-last-class-right(mark-class) = _query-mark("named", "right", "last", mark-class: mark-class)
#let fancy-top-class-right(mark-class) = _query-mark("named", "right", "top", mark-class: mark-class)

#let _query-class-value(mark-class, which) = context {
  let current-page = counter(page).get().first()
let items = _mark-items("named", mark-class:
mark-class).filter(item => item.value.at("value", default: none) != none)
let current = items.filter(item => counter(page).at(item.location()).first() == current-page)
let prior = items.filter(item => counter(page).at(item.location()).first() < current-page)
  let chosen = if which == "top" {
    if prior.len() > 0 { prior.last() } else { none }
  } else if current.len() > 0 {
    if which == "first" { current.first() } else { current.last() }
  } else if prior.len() > 0 {
    prior.last()
  } else { none }
  if chosen == none { none } else { chosen.value.at("value", default: none) }
}
#let fancy-first-mark(mark-class) = _query-class-value(mark-class, "first")
#let fancy-last-mark(mark-class) = _query-class-value(mark-class, "last")
#let fancy-top-mark(mark-class) = _query-class-value(mark-class, "top")

#let fancy-left-mark() = fancy-last-left-mark()
#let fancy-right-mark() = fancy-first-right-mark()

#let fancy-page-context(callback) = context callback(counter(page).get().first())

#let fancy-heading-mark(level: 1, which: "last") = context {
  let current-page = counter(page).get().first()
  let all-headings = query(heading.where(level: level))
let on-page = all-headings.filter(item => counter(page).at(item.location()).first() == current-page)
let prior = all-headings.filter(item => counter(page).at(item.location()).first() < current-page)
  let chosen = if on-page.len() > 0 {
    if which == "first" { on-page.first() } else { on-page.last() }
  } else if prior.len() > 0 {
    prior.last()
  } else {
    none
  }
  if chosen == none { none } else { chosen.body }
}

#let firstleftmark = fancy-first-left-mark
#let lastleftmark = fancy-last-left-mark
#let firstrightmark = fancy-first-right-mark
#let lastrightmark = fancy-last-right-mark
#let firstleftxmark = fancy-first-left-x-mark
#let lastleftxmark = fancy-last-left-x-mark
#let firstrightxmark = fancy-first-right-x-mark
#let lastrightxmark = fancy-last-right-x-mark
#let topleftxmark = fancy-top-left-x-mark
#let toprightxmark = fancy-top-right-x-mark
#let firstxmark = fancy-first-left-x-mark
#let lastxmark = fancy-last-right-x-mark
#let topxmark = fancy-top-left-x-mark
#let extramarks = fancy-extramarks
#let extramarksleft = fancy-extramarks-left
#let extramarksright = fancy-extramarks-right
#let markboth = fancy-markboth
#let markright = fancy-markright
#let leftmark = fancy-left-mark
#let rightmark = fancy-right-mark

#let _legacy-field(style, side, kind, odd-content, even-content) = {
let out = if even-content == none { style } else { _set-fields(style, "E" + side + kind, none, even-content) }
  _set-fields(out, "O" + side + kind, none, odd-content)
}
#let _even-value(even-content, even) = if even != auto { even } else if even-content != auto { even-content } else { none }
#let lhead(style, odd-content, even-content: auto, even: auto) = _legacy-field(style, "L", "H", odd-content, _even-value(even-content, even))
#let chead(style, odd-content, even-content: auto, even: auto) = _legacy-field(style, "C", "H", odd-content, _even-value(even-content, even))
#let rhead(style, odd-content, even-content: auto, even: auto) = _legacy-field(style, "R", "H", odd-content, _even-value(even-content, even))
#let lfoot(style, odd-content, even-content: auto, even: auto) = _legacy-field(style, "L", "F", odd-content, _even-value(even-content, even))
#let cfoot(style, odd-content, even-content: auto, even: auto) = _legacy-field(style, "C", "F", odd-content, _even-value(even-content, even))
#let rfoot(style, odd-content, even-content: auto, even: auto) = _legacy-field(style, "R", "F", odd-content, _even-value(even-content, even))
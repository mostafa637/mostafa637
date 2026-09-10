// listings for Typst
// A Typst-native conversion of the observable listings interface.
// The implementation deliberately uses raw(...) and Typst layout primitives.

#import "listings-languages.typ": listings-language, listings-language-aliases, listings-driver-names, listings-language-registry, listings-add-language
#import "fix-bidi.typ": fix-bidi
#import "inkjet.typ": highlighted-code as _inkjet-highlighted-code

#let _none = none

#let listings-counter = counter("listings")
#let listings-chapter-counter = counter("listings-by-heading")
#let listings-layout-state = state("listings-layout", none)
#let listings-name-state = state("listings-named-last", (:))

#let listings-defaults = (
  language: none,
  language-aliases: (:),
  dialect: none,
  defaultdialect: none,
  alsolanguage: none,
  style: none,
  basicstyle: (font: "DejaVu Sans Mono", size: 8.5pt, fill: rgb("20242b")),
  keywordstyle: none,
  keywordstyles: (:),
  commentstyle: none,
  stringstyle: none,
  tagstyle: none,
  identifierstyle: none,
  showstringspaces: true,
  extendedchars: true,
  texcl: false,
  tag: false,
  podcomment: false,
  printpod: true,
  inputencoding: none,
  input-decoder: none,
  upquote: false,
  numbers: "none",
  numberstyle: (font: "DejaVu Sans Mono", size: 7pt, fill: rgb("7b8490")),
  numbersep: 0.8em,
  firstnumber: "auto",
  stepnumber: 1,
  numberfirstline: false,
  numberblanklines: true,
  consecutivenumbers: true,
  showspaces: false,
  showtabs: false,
  tabsize: 8,
  keepspaces: true,
  columns: "flexible",
  flexiblecolumns: true,
  basewidth: none,
  fontadjust: false,
  breaklines: true,
  breakatwhitespace: false,
  breakindent: 2em,
  prebreak: none,
  postbreak: none,
  breakautoindent: true,
  wrap-chars: none,
  firstline: none,
  lastline: none,
  showlines: false,
  emptylines: none,
  linerange: none,
  rangeprefix: "",
  rangesuffix: "",
  rangebeginprefix: "",
  rangebeginsuffix: "",
  rangeendprefix: "",
  rangeendsuffix: "",
  includerangemarker: true,
  gobble: 0,
  frame: "none",
  frameshape: none,
  framerule: 0.4pt,
  framesep: 3pt,
  rulesep: 2pt,
  framexleftmargin: 0pt,
  framexrightmargin: 0pt,
  framextopmargin: 0pt,
  framexbottommargin: 0pt,
  rulecolor: rgb("7b8490"),
  rulesepcolor: rgb("b6bec8"),
  backgroundcolor: none,
  fillcolor: none,
  xleftmargin: 0pt,
  xrightmargin: 0pt,
  resetmargins: false,
  caption: none,
  title: none,
  label: none,
  nolol: false,
  listing-name: "Listing",
  captionpos: "b",
  abovecaptionskip: 0.5em,
  belowcaptionskip: 0.5em,
  aboveskip: 0.5em,
  belowskip: 0.5em,
  lineskip: 0pt,
  everydisplay: none,
  boxpos: "c",
  escapeinside: none,
  escapechar: "",
  escapebegin: none,
  escapeend: none,
  escape-renderer: none,
  mathescape: false,
  literate: none,
  morekeywords: (),
  deletekeywords: (),
  comment: (),
  morecomment: (),
  deletecomment: (),
  string: (),
  morestring: (),
  deletestring: (),
  delim: (),
  moredelim: (),
  deletedelim: (),
  excludedelims: (),
  alsoletter: (),
  alsoother: (),
  emph: (),
  emphstyle: none,
  classoffset: 0,
  format: none,
  fmtindent: 20pt,
  fancyvrb: false,
  useoutput: 0,
  SelectCharTable: none,
  rescanchars: none,
  index: none,
  moreindex: (),
  deleteindex: (),
  indexstyle: none,
  numberbychapter: false,
  language-definition: none,
  name: none,
  inputpath: none,
  tab: none,
  formfeed: none,
  print: true,
  linewidth: auto,
  float: false,
  floatplacement: "htbp",
  multicols: none,
  multipage: false,
  show-continuation: false,
  continuation-label: "continued",
  continuation-symbol: "↪",
  continuation-style: (font: "DejaVu Sans Mono", size: 7pt, fill: rgb("68727e")),
  continuation-rule: 0.3pt + rgb("9aa3ad"),
  max-lines-per-page: none,
  title-style: (size: 8pt, weight: "bold", fill: rgb("3d4652")),
  arabic-digits: "arabic",
  bidi: "auto",
  bidi-default-dir: rtl,
  bidi-default-lang: "ar",
)

#let listings-config(options: (:)) = listings-defaults + options
#let lstset(config: listings-defaults, options: (:)) = config + options
#let listings-set = lstset
#let lstreset = listings-config()

#let _get(options, key, default: none) = options.at(key, default: default)

#let _lang(options, config) = {
  let chosen = _get(options, "language", default: none)
  if chosen != none { chosen } else { _get(config, "language", default: none) }
}

#let _canonical-language(name, options: (:)) = {
  let extra = _get(options, "language-aliases", default: (:))
  let registry = if type(extra) == dictionary { listings-language-registry(extra: extra) } else { listings-language-aliases }
  listings-language(name, registry: registry)
}

#let _custom-list(value) = {
  if value == none { () } else if type(value) == array { value } else { (value,) }
}

#let _registry-definition(registry, language) = {
  if type(registry) != dictionary or language == none or language == "" {
    none
  } else {
    let direct = registry.at(language, default: none)
    if direct != none {
      direct
    } else {
      let found = none
      for (defined-name, definition) in registry {
        let aliases = if type(definition) == dictionary { definition.at("aliases", default: ()) } else { () }
        if type(aliases) == array and language in aliases { found = definition }
      }
      found
    }
  }
}

#let _custom-apply-dialect(definition, options) = {
  if definition == none or type(definition) != dictionary { definition }
  else {
    let requested = _get(options, "dialect", default: none)
    let configured-default = _get(options, "defaultdialect", default: none)
    let definition-default = definition.at("defaultdialect", default: none)
    let dialect-name = if requested != none { requested } else if configured-default != none { configured-default } else { definition-default }
    let dialects = definition.at("dialects", default: none)
    if dialect-name == none or type(dialects) != dictionary {
      definition
    } else {
      let dialect = dialects.at(str(dialect-name), default: none)
      if type(dialect) != dictionary { definition } else {
        let keywords = _custom-list(definition.at("keywords", default: ())) + _custom-list(dialect.at("keywords", default: ()))
        let comments = _custom-list(definition.at("comments", default: ())) + _custom-list(dialect.at("comments", default: ()))
        let strings = _custom-list(definition.at("strings", default: ())) + _custom-list(dialect.at("strings", default: ()))
        definition + (keywords: keywords, comments: comments, strings: strings)
      }
    }
  }
}

#let _custom-language-definition(options, language) = {
  let registry = _get(options, "language-definition", default: none)
  let primary = _registry-definition(registry, language)
  let alternate-name = _get(options, "alsolanguage", default: none)
  let alternate = _registry-definition(registry, alternate-name)
  let combined = if primary == none or alternate == none {
    primary
  } else {
    let primary-keywords = _custom-list(primary.at("keywords", default: ()))
    let alternate-keywords = _custom-list(alternate.at("keywords", default: ()))
    let primary-comments = _custom-list(primary.at("comments", default: ()))
    let alternate-comments = _custom-list(alternate.at("comments", default: ()))
    let primary-strings = _custom-list(primary.at("strings", default: ()))
    let alternate-strings = _custom-list(alternate.at("strings", default: ()))
    primary + (
      keywords: primary-keywords + alternate-keywords,
      comments: primary-comments + alternate-comments,
      strings: primary-strings + alternate-strings,
    )
  }
  _custom-apply-dialect(combined, options)
}

#let _merge-options(config, options) = config + options

#let _resolve-style-options(options, styles) = {
  let name = _get(options, "style", default: none)
  if name == none or styles == none or type(styles) != dictionary {
    options
  } else {
    styles.at(name, default: (:)) + options
  }
}

#let _as-length(value, fallback) = {
  if type(value) == length { value } else { fallback }
}

#let _line-number-style(value, content) = {
  if value == none {
    text(font: "DejaVu Sans Mono", size: 7pt, fill: rgb("7b8490"), content)
  } else if type(value) == dictionary {
    text(..value, content)
  } else {
    content
  }
}

#let _arabic-shared-font(options) = {
  let basic = _get(options, "basicstyle", default: none)
  if type(basic) == dictionary {
    (
      font: basic.at("font", default: "DejaVu Sans Mono"),
      size: basic.at("size", default: 8.5pt),
    )
  } else {
    (font: "DejaVu Sans Mono", size: 8.5pt)
  }
}

#let _arabic-line-number-style(options, content) = {
  let shared = _arabic-shared-font(options)
  let configured = _get(options, "numberstyle", default: none)
  let style = if type(configured) == dictionary {
    configured + shared
  } else {
    shared + (fill: rgb("7b8490"))
  }
  text(..style, content)
}

#let _arabic-line-box(options, content) = {
  let shared = _arabic-shared-font(options)
  let line-height = shared.size * 1.45
  box(height: line-height)[#content]
}

#let _code-style(value, content) = {
  if value == none {
    text(font: "DejaVu Sans Mono", size: 8.5pt, content)
  } else if type(value) == dictionary {
    text(..value, content)
  } else {
    content
  }
}

#let _column-formatted(content, options) = {
  let mode = _get(options, "columns", default: "flexible")
  let flexible = _get(options, "flexiblecolumns", default: true)
  let fixed = type(mode) == str and mode.contains("fixed") or (mode == "flexible" and not flexible)
  let configured = _get(options, "basewidth", default: none)
  let has-width = type(configured) == length or (type(configured) == array and configured.len() > 0)
  let base = if type(configured) == length {
    configured
  } else if type(configured) == array and configured.len() > 0 and type(configured.at(0)) == length {
    if not fixed and configured.len() > 1 and type(configured.at(1)) == length { configured.at(1) } else { configured.at(0) }
  } else {
    0.6em
  }
  if fixed or (has-width and type(configured) == array) {
    text(tracking: base - 0.6em, content)
  } else {
    content
  }
}

#let _frame-stroke(options) = {
  let configured-shape = _get(options, "frameshape", default: none)
  let frame = if type(configured-shape) == dictionary { configured-shape } else { _get(options, "frame", default: "none") }
  let width = _get(options, "framerule", default: 0.4pt)
  let color = _get(options, "rulecolor", default: rgb("7b8490"))
  let stroke = width + color
  let double-stroke = (width * 2) + color
  if frame == none or frame == "none" { none }
  else if frame == "topline" { (top: stroke) }
  else if frame == "bottomline" { (bottom: stroke) }
  else if frame == "leftline" { (left: stroke) }
  else if frame == "rightline" { (right: stroke) }
  else if frame == "lines" { (top: stroke, bottom: stroke) }
  else if frame == "single" or frame == "shadowbox" { stroke }
  else if type(frame) == dictionary {
    let top = if frame.at("top", default: false) { stroke } else { none }
    let bottom = if frame.at("bottom", default: false) { stroke } else { none }
    let left = if frame.at("left", default: false) { stroke } else { none }
    let right = if frame.at("right", default: false) { stroke } else { none }
    (top: top, bottom: bottom, left: left, right: right)
  }
  else if type(frame) == str {
    let top = if frame.contains("t") { stroke } else if frame.contains("T") { double-stroke } else { none }
    let bottom = if frame.contains("b") { stroke } else if frame.contains("B") { double-stroke } else { none }
    let left = if frame.contains("l") { stroke } else if frame.contains("L") { double-stroke } else { none }
    let right = if frame.contains("r") { stroke } else if frame.contains("R") { double-stroke } else { none }
    (top: top, bottom: bottom, left: left, right: right)
  } else { stroke }
}

#let _background(options) = {
  let bg = _get(options, "backgroundcolor", default: none)
  if bg != none { bg } else { _get(options, "fillcolor", default: none) }
}

#let _normalize-code(code) = {
  if type(code) == str { code }
  else { none }
}

#let _split-lines(code, keep-final: false) = {
  let lines = code.split("\n")
  // A terminal newline is a source delimiter unless showlines requests it.
  if not keep-final and lines.len() > 1 and lines.last() == "" { lines.slice(0, -1) } else { lines }
}

#let _limit-empty-lines(lines, options) = {
  let limit = _get(options, "emptylines", default: none)
  if type(limit) != int or limit < 0 { lines }
  else {
    let result = ()
    let run = 0
    for line in lines {
      if line == "" {
        run += 1
        if run <= limit { result.push(line) }
      } else {
        run = 0
        result.push(line)
      }
    }
    result
  }
}

#let _apply-gobble(lines, amount) = {
  if type(amount) != int or amount <= 0 { lines }
  else {
    lines.map(line => if line.len() > amount { line.slice(amount) } else { "" })
  }
}

#let _apply-literate(lines, options) = {
  let rules = _get(options, "literate", default: none)
  if type(rules) != array { lines }
  else {
    let result = lines
    for rule in rules {
      if type(rule) == array and rule.len() >= 2 and type(rule.at(0)) == str {
        let replacement = str(rule.at(1))
        result = result.map(line => line.replace(rule.at(0), replacement))
      }
    }
    result
  }
}

#let _apply-format(code, options) = {
  let formatter = _get(options, "format", default: none)
  if formatter == none {
    code
  } else if type(formatter) == function {
    let result = formatter(code)
    if type(result) == str { result } else { code }
  } else if type(formatter) == array {
    let result = code
    for rule in formatter {
      if type(rule) == array and rule.len() >= 2 and type(rule.at(0)) == str {
        result = result.replace(rule.at(0), str(rule.at(1)))
      }
    }
    result
  } else if type(formatter) == dictionary {
    let result = code
    let replacements = formatter.at("replacements", default: ())
    let result = if type(replacements) == array {
      let replaced = result
      for rule in replacements {
        if type(rule) == array and rule.len() >= 2 and type(rule.at(0)) == str {
          replaced = replaced.replace(rule.at(0), str(rule.at(1)))
        }
      }
      replaced
    } else { result }
    let before = formatter.at("before", default: "")
    let after = formatter.at("after", default: "")
    let wrapped = str(before) + result + str(after)
    if formatter.at("indent", default: false) {
      let width = _get(options, "fmtindent", default: 20pt)
      let spaces = if type(width) == length { calc.max(1, int(width / 0.5em)) } else { 4 }
      let prefix = " " * spaces
      let formatted-lines = wrapped.split("\n")
      formatted-lines.enumerate().map(item => {
        let index = item.at(0)
        let line = item.at(1)
        if index == 0 { line } else { prefix + line }
      }).join("\n")
    } else {
      wrapped
    }
  } else {
    code
  }
}

#let _wrap-indent(line, options) = {
  let chars = line.codepoints()
  let leading = 0
  while leading < chars.len() and (chars.at(leading) == " " or chars.at(leading) == "\t") { leading += 1 }
  let inherited = if _get(options, "breakautoindent", default: true) { chars.slice(0, leading).join() } else { "" }
  let amount = _get(options, "breakindent", default: 2em)
  let extra = if type(amount) == length { calc.max(1, int(amount / 0.5em)) } else if type(amount) == int { calc.max(0, amount) } else { 4 }
  inherited + " " * extra
}

#let _wrap-one-line(line, options, width) = {
  let chars = line.codepoints()
  let pre = _get(options, "prebreak", default: none)
  let post = _get(options, "postbreak", default: none)
  let pre = if pre == none { "" } else { str(pre) }
  let post = if post == none { "" } else { str(post) }
  if width <= 0 or chars.len() <= width {
    (line,)
  } else {
    let result = ()
    let remaining = chars
    let first = true
    while remaining.len() > width {
      let room = if first { width } else { calc.max(1, width - _wrap-indent(line, options).len()) }
      let cut = if _get(options, "breakatwhitespace", default: false) {
        let candidate = room
        while candidate > 0 and remaining.at(candidate - 1) != " " and remaining.at(candidate - 1) != "\t" { candidate -= 1 }
        if candidate > 0 { candidate } else { room }
      } else { room }
      let actual = calc.max(1, cut)
      let head = remaining.slice(0, actual).join()
      let prefix = if first { "" } else { _wrap-indent(line, options) }
      let continuation-marker = if first { "" } else { post }
      result.push(prefix + continuation-marker + head + pre)
      remaining = remaining.slice(actual, remaining.len())
      first = false
    }
    let prefix = if first { "" } else { _wrap-indent(line, options) }
    result.push(prefix + post + remaining.join())
    result
  }
}

#let _wrap-source-lines(lines, options) = {
  let requested = _get(options, "wrap-chars", default: none)
  if type(requested) != int or requested <= 0 or not _get(options, "breaklines", default: true) {
    lines
  } else {
    let wrapped = ()
    for line in lines {
      for part in _wrap-one-line(line, options, requested) { wrapped.push(part) }
    }
    wrapped
  }
}

#let _prepare-source-lines(code, options) = {
  let formatted = _apply-format(code, options)
  let lines = _split-lines(formatted, keep-final: _get(options, "showlines", default: false))
  let lines = _apply-gobble(lines, _get(options, "gobble", default: 0))
  let lines = _apply-literate(lines, options)
  let lines = _limit-empty-lines(lines, options)
  _wrap-source-lines(lines, options)
}

#let _collapse-source-spaces(line) = {
  let result = ""
  let previous-space = false
  for c in line.codepoints() {
    if c == " " {
      if not previous-space { result += c }
      previous-space = true
    } else {
      result += c
      previous-space = false
    }
  }
  result
}

#let _visible-whitespace(line, options) = {
  let result = if _get(options, "keepspaces", default: true) { line } else { _collapse-source-spaces(line) }
  if _get(options, "upquote", default: false) {
    result = result.replace("‘", "'").replace("’", "'").replace("“", "\"").replace("”", "\"")
  }
  let configured-tab = _get(options, "tab", default: none)
  let tab-marker = if configured-tab == none { "→" } else { configured-tab }
  if _get(options, "showtabs", default: false) {
    result = result.replace("\t", str(tab-marker) + " " * _get(options, "tabsize", default: 8))
  } else {
    result = result.replace("\t", " " * _get(options, "tabsize", default: 8))
  }
  if _get(options, "showspaces", default: false) {
    result = result.replace(" ", "·")
  }
  result
}

#let _marker-range(lines, pair, options) = {
  let shared-prefix = _get(options, "rangeprefix", default: "")
  let shared-suffix = _get(options, "rangesuffix", default: "")
  let begin-prefix = _get(options, "rangebeginprefix", default: "")
  let begin-suffix = _get(options, "rangebeginsuffix", default: "")
  let end-prefix = _get(options, "rangeendprefix", default: "")
  let end-suffix = _get(options, "rangeendsuffix", default: "")
  let begin-prefix = if begin-prefix == "" { shared-prefix } else { begin-prefix }
  let begin-suffix = if begin-suffix == "" { shared-suffix } else { begin-suffix }
  let end-prefix = if end-prefix == "" { shared-prefix } else { end-prefix }
  let end-suffix = if end-suffix == "" { shared-suffix } else { end-suffix }
  let open-marker = begin-prefix + str(pair.at(0)) + begin-suffix
  let close-marker = end-prefix + str(pair.at(1)) + end-suffix
  let start = none
  let finish = none
  for (index, line) in lines.enumerate() {
    if start == none and line.contains(open-marker) { start = index }
    if start != none and finish == none and line.contains(close-marker) { finish = index }
  }
  if start == none {
    (start: 0, finish: 0)
  } else {
    let include-marker = _get(options, "includerangemarker", default: true)
    let a = if include-marker { start } else { start + 1 }
    let b = if finish == none { lines.len() } else if include-marker { finish + 1 } else { finish }
    (start: calc.min(lines.len(), a), finish: calc.max(calc.min(lines.len(), b), calc.min(lines.len(), a)))
  }
}

#let _line-selection(lines, options) = {
  let first = _get(options, "firstline", default: none)
  let last = _get(options, "lastline", default: none)
  let ranges = _get(options, "linerange", default: none)
  if ranges == none {
    let a = if type(first) == int { calc.max(0, first - 1) } else { 0 }
    let b = if type(last) == int { calc.min(lines.len(), last) } else { lines.len() }
    let selected = lines.slice(a, b)
    let source-indices = ()
    for (index, _) in selected.enumerate() { source-indices.push(a + index) }
    (lines: selected, offset: a, source-indices: source-indices, range-starts: (0,))
  } else {
    let pairs = if type(ranges) == array and ranges.len() == 2 and (type(ranges.at(0)) == int or type(ranges.at(0)) == str) { (ranges,) } else { ranges }
    let result = ()
    let source-indices = ()
    let range-starts = ()
    let first-pair = if pairs.len() > 0 { pairs.at(0) } else { (1, 0) }
    let offset = if type(first-pair.at(0)) == int { calc.max(0, first-pair.at(0) - 1) } else { 0 }
    for pair in pairs {
      let bounds = if type(pair) == array and pair.len() >= 2 and type(pair.at(0)) == int and type(pair.at(1)) == int {
        (start: calc.max(0, pair.at(0) - 1), finish: calc.min(lines.len(), pair.at(1)))
      } else if type(pair) == array and pair.len() >= 2 {
        _marker-range(lines, pair, options)
      } else {
        (start: 0, finish: 0)
      }
      range-starts.push(result.len())
      for (index, line) in lines.slice(bounds.start, bounds.finish).enumerate() {
        result.push(line)
        source-indices.push(bounds.start + index)
      }
    }
    (lines: result, offset: offset, source-indices: source-indices, range-starts: range-starts)
  }
}

#let _line-range(lines, options) = _line-selection(lines, options).lines

#let _line-number-value(index, options, offset: 0, source-index: none) = {
  let first = _get(options, "firstnumber", default: "auto")
  let consecutive = _get(options, "consecutivenumbers", default: true)
  if consecutive or source-index == none {
    let start = if type(first) == int { first } else { 1 + offset }
    start + index
  } else if type(first) == int {
    first + source-index - offset
  } else {
    source-index + 1
  }
}

#let _number-this-line(index, line, options, offset: 0, source-index: none) = {
  let numbers = _get(options, "numbers", default: "none")
  let step = _get(options, "stepnumber", default: 1)
  let logical = _line-number-value(index, options, offset: offset, source-index: source-index)
  if numbers == none or numbers == "none" or type(step) != int or step <= 0 { false }
  else if line == "" and not _get(options, "numberblanklines", default: true) { false }
  else if index == 0 and _get(options, "numberfirstline", default: false) { true }
  else { calc.rem(logical, step) == 0 }
}

#let _arabic-code-keywords = (
  "إذا", "وإلا", "طالما", "لكل", "في", "دالة", "أرجع", "اطبع",
  "دع", "متغير", "ثابت", "صنف", "استورد", "من", "جديد", "كسر",
  "تابع", "تحقق", "صحيح", "خطأ", "فارغ", "نص", "عدد", "قائمة", "منطق",
)

#let _contains-char(value, c) = {
  if type(value) == str {
    value.codepoints().contains(c)
  } else if type(value) == array {
    value.any(item => type(item) == str and item.codepoints().contains(c))
  } else {
    false
  }
}

#let _arabic-word-char(c, options: (:)) = {
  let letters = _get(options, "alsoletter", default: "")
  let digits = _get(options, "alsodigit", default: "")
  let other = _get(options, "alsoother", default: "")
  ((c >= "ا" and c <= "ي") or (c >= "A" and c <= "Z") or
   (c >= "a" and c <= "z") or (c >= "0" and c <= "9") or c == "_" or
   (c >= "\u{0660}" and c <= "\u{0669}") or _contains-char(letters, c) or
   _contains-char(digits, c) or _contains-char(other, c))
}

#let _arabic-digit-char(c) = {
  (c >= "0" and c <= "9") or (c >= "\u{0660}" and c <= "\u{0669}")
}

#let _arabic-digit-shape(token, options) = {
  let mode = _get(options, "arabic-digits", default: "arabic")
  if mode == "latin" {
    let out = ""
    for c in token.codepoints() {
      out += if c == "٠" { "0" } else if c == "١" { "1" } else if c == "٢" { "2" } else if c == "٣" { "3" } else if c == "٤" { "4" } else if c == "٥" { "5" } else if c == "٦" { "6" } else if c == "٧" { "7" } else if c == "٨" { "8" } else if c == "٩" { "9" } else { c }
    }
    out
  } else if mode == "arabic" {
    let out = ""
    for c in token.codepoints() {
      out += if c == "0" { "٠" } else if c == "1" { "١" } else if c == "2" { "٢" } else if c == "3" { "٣" } else if c == "4" { "٤" } else if c == "5" { "٥" } else if c == "6" { "٦" } else if c == "7" { "٧" } else if c == "8" { "٨" } else if c == "9" { "٩" } else { c }
    }
    out
  } else {
    token
  }
}

#let _arabic-number-isolate(token) = {
  let compound = token.codepoints().any(c => not _arabic-digit-char(c))
  if compound {
    "\u{202A}" + token + "\u{202C}"
  } else {
    "\u{2066}" + token + "\u{2069}"
  }
}

#let _arabic-index-match(value, token) = {
  if type(value) == str {
    value == token
  } else if type(value) == array {
    value.any(item => {
      if type(item) == str { item == token } else if type(item) == array and item.len() > 0 { item.at(0) == token } else { false }
    })
  } else {
    false
  }
}

#let _arabic-index-record(options, token) = {
  let base = _get(options, "index", default: none)
  let added = _get(options, "moreindex", default: ())
  let removed = _get(options, "deleteindex", default: ())
  let matched = _arabic-index-match(base, token) or _arabic-index-match(added, token)
  if matched and not _arabic-index-match(removed, token) {
    metadata((kind: "listing-index", term: token, target: none))
  } else {
    none
  }
}

#let _arabic-token-style(token, options, kind) = {
  let configured = if kind == "keyword" { _get(options, "keywordstyle", default: none) }
    else if kind == "comment" { _get(options, "commentstyle", default: none) }
    else if kind == "string" { _get(options, "stringstyle", default: none) }
    else { none }
  let fallback = if kind == "keyword" {
    (fill: rgb("1d4ed8"), weight: "bold")
  } else if kind == "comment" {
    (fill: rgb("64748b"), style: "italic")
  } else if kind == "string" {
    (fill: rgb("b45309"))
  } else {
    (fill: rgb("7c3aed"))
  }
  let style = if type(configured) == dictionary { configured } else { fallback }
  let shown-token = if kind == "string" and _get(options, "showstringspaces", default: true) and not _get(options, "showspaces", default: false) {
    token.replace(" ", "␠")
  } else { token }
  text(..(style + _arabic-shared-font(options)), shown-token)
}

#let _arabic-extra-prefix-at(chars, index, prefixes) = {
  let found = none
  if type(prefixes) == array {
    for prefix in prefixes {
      if type(prefix) == str {
        let codepoints = prefix.codepoints()
        if codepoints.len() > 0 and index + codepoints.len() <= chars.len() and chars.slice(index, index + codepoints.len()) == codepoints {
          found = prefix
        }
      }
    }
  }
  found
}

#let _arabic-plain-token(token, options) = {
  let emph = _get(options, "emph", default: ())
  let emph-style = _get(options, "emphstyle", default: none)
  let identifier-style = _get(options, "identifierstyle", default: none)
  let style = if type(emph) == array and token in emph and type(emph-style) == dictionary {
    emph-style
  } else if type(identifier-style) == dictionary {
    identifier-style
  } else {
    none
  }
  if style == none { token } else { text(..(style + _arabic-shared-font(options)), token) }
}

#let _arabic-delimiter-at(chars, index, rules) = {
  let found = none
  if type(rules) == array {
    for rule in rules {
      if type(rule) == array and rule.len() >= 2 and type(rule.at(0)) == str and type(rule.at(1)) == str {
        let opener = rule.at(0)
        let opener-chars = opener.codepoints()
        if opener-chars.len() > 0 and index + opener-chars.len() <= chars.len() and chars.slice(index, index + opener-chars.len()) == opener-chars {
          found = rule
        }
      }
    }
  }
  found
}

#let _arabic-code-line(line, options) = {
  let chars = line.codepoints()
  let deleted-comments = _get(options, "deletecomment", default: ())
  let deleted-strings = _get(options, "deletestring", default: ())
  let deleted-delimiters = _get(options, "deletedelim", default: ())
  let pieces = ()
  let plain = ""
  let i = 0
  while i < chars.len() {
    let c = chars.at(i)
    let excluded-delim = _arabic-delimiter-at(chars, i, _get(options, "excludedelims", default: ()))
    if excluded-delim != none {
      let opener = excluded-delim.at(0)
      let closer = excluded-delim.at(1)
      let opener-chars = opener.codepoints()
      let closer-chars = closer.codepoints()
      let j = i + opener-chars.len()
      while j < chars.len() {
        if j + closer-chars.len() <= chars.len() and chars.slice(j, j + closer-chars.len()) == closer-chars {
          j = j + closer-chars.len()
          break
        }
        j = j + 1
      }
      pieces.push(chars.slice(i, j).join())
      i = j
    } else {
    let configured-comment = _arabic-extra-prefix-at(chars, i, _get(options, "comment", default: ()))
    let extra-comment = _arabic-extra-prefix-at(chars, i, _get(options, "morecomment", default: ()))
    let deleted-comment = _arabic-extra-prefix-at(chars, i, deleted-comments)
    let configured-delim = _get(options, "delim", default: ())
    let configured-more-delim = _get(options, "moredelim", default: ())
    let extra-delim = _arabic-delimiter-at(chars, i, configured-delim + configured-more-delim)
    let deleted-delim = _arabic-delimiter-at(chars, i, deleted-delimiters)
    let base-hash-comment = c == "#" and _arabic-extra-prefix-at(chars, i, deleted-comments) != "#"
    let base-slash-comment = c == "/" and i + 1 < chars.len() and chars.at(i + 1) == "/" and deleted-comment != "//"
    let is-comment = base-hash-comment or base-slash-comment or (configured-comment != none and deleted-comment == none) or (extra-comment != none and deleted-comment == none)
    if is-comment {
      if plain != "" { pieces.push(plain); plain = "" }
      pieces.push(_arabic-token-style(chars.slice(i, chars.len()).join(), options, "comment"))
      i = chars.len()
    } else if extra-delim != none and deleted-delim == none {
      let opener = extra-delim.at(0)
      let closer = extra-delim.at(1)
      let opener-chars = opener.codepoints()
      let closer-chars = closer.codepoints()
      let j = i + opener-chars.len()
      while j < chars.len() {
        if j + closer-chars.len() <= chars.len() and chars.slice(j, j + closer-chars.len()) == closer-chars {
          j = j + closer-chars.len()
          break
        }
        j = j + 1
      }
      let token = chars.slice(i, j).join()
      let custom-style = extra-delim.at(2, default: none)
      if type(custom-style) == dictionary {
        pieces.push(text(..(custom-style + _arabic-shared-font(options)), token))
      } else {
        pieces.push(token)
      }
      i = j
    } else if (c == "\"" and _arabic-extra-prefix-at(chars, i, deleted-strings) != "\"") or (c == "'" and _arabic-extra-prefix-at(chars, i, deleted-strings) != "'") or (_arabic-extra-prefix-at(chars, i, _get(options, "string", default: ()) + _get(options, "morestring", default: ())) != none and _arabic-extra-prefix-at(chars, i, deleted-strings) == none) {
      if plain != "" { pieces.push(plain); plain = "" }
      let extra-string = _arabic-extra-prefix-at(chars, i, _get(options, "string", default: ()) + _get(options, "morestring", default: ()))
      let quote = if extra-string == none { c } else { extra-string }
      let quote-chars = quote.codepoints()
      let j = i + quote-chars.len()
      while j < chars.len() {
        let closes = if quote-chars.len() == 1 { chars.at(j) == quote } else { j + quote-chars.len() <= chars.len() and chars.slice(j, j + quote-chars.len()) == quote-chars }
        if closes and (j == i + quote-chars.len() or chars.at(j - 1) != "\\") {
          j = j + quote-chars.len()
          break
        }
        j = j + 1
      }
      pieces.push(_arabic-token-style(chars.slice(i, j).join(), options, "string"))
      i = j
    } else if _arabic-word-char(c, options: options) {
      let j = i + 1
      while j < chars.len() and _arabic-word-char(chars.at(j), options: options) { j = j + 1 }
      let token = chars.slice(i, j).join()
      let index-record = _arabic-index-record(options, token)
      if index-record != none { pieces.push(index-record) }
      let extra-keywords = _get(options, "morekeywords", default: ())
      let deleted-keywords = _get(options, "deletekeywords", default: ())
      let is-keyword = (token in _arabic-code-keywords or (type(extra-keywords) == array and token in extra-keywords)) and not (type(deleted-keywords) == array and token in deleted-keywords)
      if is-keyword {
        if plain != "" { pieces.push(plain); plain = "" }
        pieces.push(_arabic-token-style(token, options, "keyword"))
      } else if token.codepoints().len() > 0 and _arabic-digit-char(token.codepoints().at(0)) {
        if plain != "" { pieces.push(plain); plain = "" }
        let shaped = _arabic-digit-shape(token, options)
        pieces.push(_arabic-token-style(_arabic-number-isolate(shaped), options, "number"))
      } else {
        let styled = _arabic-plain-token(token, options)
        if styled == token { plain += token } else {
          if plain != "" { pieces.push(plain); plain = "" }
          pieces.push(styled)
        }
      }
      i = j
    } else {
      plain += c
      i = i + 1
    }
    }
  }
  if plain != "" { pieces.push(plain) }
  pieces.join()
}

#let _find-sequence(chars, needle, from: 0) = {
  if needle.len() == 0 { from } else {
    let found = none
    let index = from
    while index + needle.len() <= chars.len() {
      if chars.slice(index, index + needle.len()) == needle {
        found = index
        break
      }
      index += 1
    }
    found
  }
}

#let _custom-word-char(c, options: none) = {
  let opts = if type(options) == dictionary { options } else { (:)}
  let extra-letter = _get(opts, "alsoletter", default: ())
  let extra-digit = _get(opts, "alsodigit", default: ())
  let extra-other = _get(opts, "alsoother", default: ())
  let extended = _get(opts, "extendedchars", default: true)
  let ascii-word = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
  let in-extra = value => {
    if type(value) == array { c in value }
    else if type(value) == str { value.contains(c) }
    else { false }
  }
  let unicode-letter = (
    (c >= "À" and c <= "ÿ") or
    (c >= "Α" and c <= "ω") or
    (c >= "А" and c <= "я") or
    (c >= "\u{0590}" and c <= "\u{08FF}")
  )
  let unicode-digit = (
    (c >= "٠" and c <= "٩") or
    (c >= "۰" and c <= "۹")
  )
  let extended-byte = extended and c >= "\u{0080}" and c <= "\u{00FF}"
  (
    c == "_" or c == "-" or ascii-word.contains(c) or
      (extended and unicode-letter) or extended-byte or (extended and unicode-digit) or
      in-extra(extra-letter) or in-extra(extra-digit) or in-extra(extra-other)
  )
}

#let _custom-rule-at(chars, index, rules) = {
  let found = none
  if type(rules) == array {
    for rule in rules {
      let opener = if type(rule) == str { rule } else if type(rule) == array and rule.len() > 0 and type(rule.at(0)) == str { rule.at(0) } else { none }
      if opener != none {
        let codepoints = opener.codepoints()
        if codepoints.len() > 0 and index + codepoints.len() <= chars.len() and chars.slice(index, index + codepoints.len()) == codepoints {
          found = rule
        }
      }
    }
  } else if type(rules) == str {
    let codepoints = rules.codepoints()
    if codepoints.len() > 0 and index + codepoints.len() <= chars.len() and chars.slice(index, index + codepoints.len()) == codepoints { found = rules }
  }
  found
}

#let _custom-style(options, definition, key, fallback, content) = {
  let configured = _get(options, key, default: none)
  let defined = if type(definition) == dictionary { definition.at(key, default: none) } else { none }
  let style = if configured != none { configured } else if defined != none { defined } else { fallback }
  if type(style) == dictionary {
    text(..(style + _arabic-shared-font(options)), content)
  } else {
    content
  }
}

#let _custom-keyword-list(options, definition) = {
  let defined = if type(definition) == dictionary { definition.at("keywords", default: ()) } else { () }
  let added = _get(options, "morekeywords", default: ())
  let base = if type(defined) == array { defined } else if type(defined) == str { (defined,) } else { () }
  let extra = if type(added) == array { added } else if type(added) == str { (added,) } else { () }
  base + extra
}

#let _custom-in-list(value, token) = {
  if type(value) == array { token in value } else if type(value) == str { token == value } else { false }
}

#let _custom-keyword-entry(words, token) = {
  let found = none
  if type(words) == array {
    for item in words {
      if type(item) == str and item == token {
        found = (word: token, class: 1)
      } else if type(item) == array and item.len() > 0 and item.at(0) == token {
        let class = if item.len() > 1 and type(item.at(1)) == int { item.at(1) } else { 1 }
        found = (word: token, class: class)
      } else if type(item) == dictionary and item.at("word", default: item.at("name", default: none)) == token {
        found = (word: token, class: item.at("class", default: 1))
      }
    }
  } else if type(words) == str and words == token {
    found = (word: token, class: 1)
  }
  found
}

#let _custom-keyword-style(options, definition, entry, content) = {
  let offset = _get(options, "classoffset", default: 0)
  let class-id = entry.at("class", default: 1) + offset
  let styles = _get(options, "keywordstyles", default: (:))
  let class-style = if type(styles) == dictionary { styles.at(str(class-id), default: none) } else { none }
  let configured = _get(options, "keywordstyle", default: none)
  let defined = if type(definition) == dictionary { definition.at("keywordstyle", default: none) } else { none }
  let style = if class-style != none { class-style } else if configured != none { configured } else if defined != none { defined } else { (fill: rgb("245bdb"), weight: "bold") }
  if type(style) == dictionary { text(..(style + _arabic-shared-font(options)), content) } else { content }
}

#let _custom-scanner-options(options, definition) = {
  if type(definition) != dictionary {
    options
  } else {
    let local = key => _get(options, key, default: none)
    let definition-list = key => _custom-list(definition.at(key, default: ()))
    let local-list = key => _custom-list(local(key))
    let selected = key => if local(key) == none { definition.at(key, default: none) } else { local(key) }
    options + (
      alsoletter: definition-list("alsoletter") + local-list("alsoletter"),
      alsodigit: definition-list("alsodigit") + local-list("alsodigit"),
      alsoother: definition-list("alsoother") + local-list("alsoother"),
      emph: definition-list("emph") + local-list("emph"),
      emphstyle: selected("emphstyle"),
      identifierstyle: selected("identifierstyle"),
      index: selected("index"),
      tagstyle: selected("tagstyle"),
      tag: (_get(options, "tag", default: false) or definition.at("tag", default: false)),
      texcl: (_get(options, "texcl", default: false) or definition.at("texcl", default: false)),
      podcomment: (_get(options, "podcomment", default: false) or definition.at("podcomment", default: false)),
    )
  }
}

#let _custom-code-line(line, options, definition) = {
  let chars = line.codepoints()
  let scanner-options = _custom-scanner-options(options, definition)
  let comments = if type(definition) == dictionary { _custom-list(definition.at("comments", default: ())) } else { () }
  let strings = if type(definition) == dictionary { _custom-list(definition.at("strings", default: ())) } else { () }
  let definition-delims = if type(definition) == dictionary { _custom-list(definition.at("delim", default: ())) } else { () }
  let deleted-comments = _get(options, "deletecomment", default: ())
  let deleted-strings = _get(options, "deletestring", default: ())
  let deleted-delims = _get(options, "deletedelim", default: ())
  let words = _custom-keyword-list(options, definition)
  let deleted-words = _get(options, "deletekeywords", default: ())
  let tex-comments = if _get(scanner-options, "texcl", default: false) { ("%",) } else { () }
  let pod-line = _get(scanner-options, "podcomment", default: false) and chars.len() > 0 and chars.at(0) == "="
  let pieces = if pod-line and _get(scanner-options, "printpod", default: true) {
    (_custom-style(options, definition, "commentstyle", (fill: rgb("5b856b")), line),)
  } else { () }
  let plain = ""
  let i = if pod-line { chars.len() } else { 0 }
  while i < chars.len() {
    let comment-rule = _custom-rule-at(chars, i, comments + tex-comments + _custom-list(_get(options, "comment", default: ())) + _custom-list(_get(options, "morecomment", default: ())))
    let string-rule = _custom-rule-at(chars, i, strings + _custom-list(_get(options, "string", default: ())) + _custom-list(_get(options, "morestring", default: ())))
    let delim-rule = _custom-rule-at(chars, i, definition-delims + _custom-list(_get(options, "delim", default: ())) + _custom-list(_get(options, "moredelim", default: ())))
    let tag-enabled = _get(scanner-options, "tag", default: false) and chars.at(i) == "<"
    if comment-rule != none and not _custom-in-list(deleted-comments, if type(comment-rule) == str { comment-rule } else { comment-rule.at(0) }) {
      if plain != "" { pieces.push(plain); plain = "" }
      let opener = if type(comment-rule) == str { comment-rule } else { comment-rule.at(0) }
      let closer = if type(comment-rule) == array and comment-rule.len() > 1 and type(comment-rule.at(1)) == str { comment-rule.at(1) } else { none }
      let start = i + opener.codepoints().len()
      let finish = if closer == none { chars.len() } else {
        let close-chars = closer.codepoints()
        let cursor = start
        let found = none
        while cursor + close-chars.len() <= chars.len() {
          if chars.slice(cursor, cursor + close-chars.len()) == close-chars { found = cursor; break }
          cursor = cursor + 1
        }
        if found == none { chars.len() } else { found + close-chars.len() }
      }
      pieces.push(_custom-style(options, definition, "commentstyle", (fill: rgb("5b856b")), chars.slice(i, finish).join()))
      i = finish
    } else if string-rule != none and not _custom-in-list(deleted-strings, if type(string-rule) == str { string-rule } else { string-rule.at(0) }) {
      if plain != "" { pieces.push(plain); plain = "" }
      let opener = if type(string-rule) == str { string-rule } else { string-rule.at(0) }
      let closer = if type(string-rule) == array and string-rule.len() > 1 and type(string-rule.at(1)) == str { string-rule.at(1) } else { opener }
      let start = i + opener.codepoints().len()
      let close-chars = closer.codepoints()
      let cursor = start
      let found = none
      while cursor + close-chars.len() <= chars.len() {
        if chars.slice(cursor, cursor + close-chars.len()) == close-chars and (cursor == start or chars.at(cursor - 1) != "\\") { found = cursor; break }
        cursor = cursor + 1
      }
      let finish = if found == none { chars.len() } else { found + close-chars.len() }
      pieces.push(_custom-style(options, definition, "stringstyle", (fill: rgb("a15c38")), chars.slice(i, finish).join()))
      i = finish
    } else if tag-enabled {
      if plain != "" { pieces.push(plain); plain = "" }
      let cursor = i + 1
      while cursor < chars.len() and chars.at(cursor) != ">" { cursor = cursor + 1 }
      let finish = if cursor < chars.len() { cursor + 1 } else { chars.len() }
      let tag-style = _get(scanner-options, "tagstyle", default: none)
      if type(tag-style) == dictionary { pieces.push(text(..(tag-style + _arabic-shared-font(options)), chars.slice(i, finish).join())) } else { pieces.push(chars.slice(i, finish).join()) }
      i = finish
    } else if delim-rule != none and not _custom-in-list(deleted-delims, if type(delim-rule) == str { delim-rule } else { delim-rule.at(0) }) {
      if plain != "" { pieces.push(plain); plain = "" }
      let opener = if type(delim-rule) == str { delim-rule } else { delim-rule.at(0) }
      let closer = if type(delim-rule) == array and delim-rule.len() > 1 and type(delim-rule.at(1)) == str { delim-rule.at(1) } else { opener }
      let close-chars = closer.codepoints()
      let cursor = i + opener.codepoints().len()
      let found = none
      while cursor + close-chars.len() <= chars.len() {
        if chars.slice(cursor, cursor + close-chars.len()) == close-chars { found = cursor; break }
        cursor = cursor + 1
      }
      let finish = if found == none { chars.len() } else { found + close-chars.len() }
      let delim-style = if type(delim-rule) == array and delim-rule.len() > 2 and type(delim-rule.at(2)) == dictionary { delim-rule.at(2) } else { none }
      if delim-style == none { pieces.push(chars.slice(i, finish).join()) } else { pieces.push(text(..(delim-style + _arabic-shared-font(options)), chars.slice(i, finish).join())) }
      i = finish
    } else if _custom-word-char(chars.at(i), options: scanner-options) {
      let j = i + 1
      while j < chars.len() and _custom-word-char(chars.at(j), options: scanner-options) { j = j + 1 }
      let token = chars.slice(i, j).join()
      let index-record = _arabic-index-record(scanner-options, token)
      if index-record != none { pieces.push(index-record) }
      let keyword-entry = _custom-keyword-entry(words, token)
      if keyword-entry != none and not _custom-in-list(deleted-words, token) {
        if plain != "" { pieces.push(plain); plain = "" }
        pieces.push(_custom-keyword-style(options, definition, keyword-entry, token))
      } else {
        let emph = _get(scanner-options, "emph", default: ())
        let emph-style = _get(scanner-options, "emphstyle", default: none)
        let identifier-style = _get(scanner-options, "identifierstyle", default: none)
        let token-style = if type(emph) == array and token in emph and type(emph-style) == dictionary { emph-style } else if type(identifier-style) == dictionary { identifier-style } else { none }
        if token-style != none {
          if plain != "" { pieces.push(plain); plain = "" }
          pieces.push(text(..(token-style + _arabic-shared-font(options)), token))
        } else {
          plain += token
        }
      }
      i = j
    } else {
      plain += chars.at(i)
      i = i + 1
    }
  }
  if plain != "" { pieces.push(plain) }
  pieces.join()
}

#let _arabic-code-rendered-line(line, options) = {
  let mode = _get(options, "bidi", default: false)
  if mode == "algorithm" {
    let rendered = _arabic-code-line(_visible-whitespace(line, options), options)
    fix-bidi[#rendered]
  } else {
    let rendered = _arabic-code-line(line, options)
    if mode == true or mode == "auto" {
      fix-bidi[#rendered]
    } else {
      rendered
    }
  }
}

#let _render-raw-fragment(fragment, options, language) = {
  if language == "arabic-code" {
    _arabic-code-rendered-line(fragment, options)
  } else {
    let custom = _custom-language-definition(options, language)
    if custom != none {
      _custom-code-line(fragment, options, custom)
    } else if language == none or language == "" {
      raw(fragment)
    } else {
      _inkjet-highlighted-code(fragment, language: language)
    }
  }
}

#let _render-escaped-line(line, options, language) = {
  let configured = _get(options, "escapeinside", default: none)
  let escape-char = _get(options, "escapechar", default: "")
  let delimiters = if type(configured) == array and configured.len() >= 2 { configured } else if type(escape-char) == str and escape-char.len() > 0 { (escape-char, escape-char) } else if _get(options, "mathescape", default: false) { ("$", "$") } else { none }
  if type(delimiters) != array or delimiters.len() < 2 or type(delimiters.at(0)) != str or type(delimiters.at(1)) != str {
    none
  } else {
    let open = delimiters.at(0).codepoints()
    let close = delimiters.at(1).codepoints()
    let chars = line.codepoints()
    let pieces = ()
    let cursor = 0
    let escaped = false
    while cursor < chars.len() {
      let start = _find-sequence(chars, open, from: cursor)
      if start == none {
        if cursor < chars.len() { pieces.push(_render-raw-fragment(chars.slice(cursor, chars.len()).join(), options, language)) }
        cursor = chars.len()
      } else {
        if start > cursor { pieces.push(_render-raw-fragment(chars.slice(cursor, start).join(), options, language)) }
        let content-start = start + open.len()
        let finish = _find-sequence(chars, close, from: content-start)
        if finish == none {
          pieces.push(_render-raw-fragment(chars.slice(start, chars.len()).join(), options, language))
          cursor = chars.len()
        } else {
          let inner = chars.slice(content-start, finish).join()
          let renderer = if _get(options, "mathescape", default: false) and configured == none {
            _get(options, "math-renderer", default: none)
          } else {
            _get(options, "escape-renderer", default: none)
          }
          let begin = _get(options, "escapebegin", default: none)
          let end = _get(options, "escapeend", default: none)
          if begin != none { pieces.push(begin) }
          if type(renderer) == function {
            pieces.push(renderer(inner))
          } else {
            pieces.push(text(fill: rgb("b45309"), inner))
          }
          if end != none { pieces.push(end) }
          cursor = finish + close.len()
          escaped = true
        }
      }
    }
    if escaped { pieces.join() } else { none }
  }
}

#let _render-raw-line(line, options, language) = {
  let visible = _visible-whitespace(line, options)
  let escaped = _render-escaped-line(visible, options, language)
  let raw-content = if escaped != none {
    escaped
  } else {
    _render-raw-fragment(visible, options, language)
  }
  _column-formatted(_code-style(_get(options, "basicstyle", default: none), raw-content), options)
}

#let _line-cells(lines, options, language, offset: 0, source-indices: none) = {
  let numbers = _get(options, "numbers", default: "none")
  let cells = ()
  for (index, line) in lines.enumerate() {
    let source-index = if source-indices == none { none } else { source-indices.at(index, default: none) }
    let show-number = _number-this-line(index, line, options, offset: offset, source-index: source-index)
    let number = if show-number { str(_line-number-value(index, options, offset: offset, source-index: source-index)) } else { "" }
    let number-content = if language == "arabic-code" {
      _arabic-line-number-style(options, number)
    } else {
      _line-number-style(_get(options, "numberstyle", default: none), number)
    }
    let code-content = _render-raw-line(line, options, language)
    if language == "arabic-code" {
      number-content = _arabic-line-box(options, number-content)
      code-content = _arabic-line-box(options, code-content)
    }
    if numbers == "right" {
      cells = cells + (code-content, number-content)
    } else {
      cells = cells + (number-content, code-content)
    }
  }
  let columns = if numbers == "right" { (1fr, auto) } else { (auto, 1fr) }
  (columns, cells)
}

#let _render-numbered-lines(lines, options, language, offset: 0, source-indices: none) = {
  let prepared = _line-cells(lines, options, language, offset: offset, source-indices: source-indices)
  // This table is only the two-column line layout. Its cells must not inherit
  // the listing frame; the outer _listing-frame owns the single perimeter.
  table(columns: prepared.at(0), column-gutter: _get(options, "numbersep", default: 0.8em), row-gutter: _get(options, "lineskip", default: 0pt), inset: (x: 0pt, y: 0pt), stroke: none, ..prepared.at(1))
}

#let _render-code-lines(lines, options, language, offset: 0, source-indices: none) = {
  if _get(options, "numbers", default: "none") == none or _get(options, "numbers", default: "none") == "none" {
    if _get(options, "escapeinside", default: none) != none or _get(options, "escapechar", default: "") != "" or _get(options, "mathescape", default: false) {
      lines.map(line => _render-raw-line(line, options, language)).join("\n")
    } else if language == "arabic-code" {
      let shown = lines.map(line => _arabic-code-rendered-line(line, options)).join("\n")
      _column-formatted(_code-style(_get(options, "basicstyle", default: none), shown), options)
    } else if _custom-language-definition(options, language) != none {
      let shown = lines.map(line => _render-raw-line(line, options, language)).join("\n")
      _column-formatted(shown, options)
    } else {
      let shown = lines.map(line => _visible-whitespace(line, options)).join("\n")
      let raw-content = if language == none or language == "" { raw(shown, block: true) } else { raw(shown, lang: language, block: true) }
      _column-formatted(_code-style(_get(options, "basicstyle", default: none), raw-content), options)
    }
  } else {
    _render-numbered-lines(lines, options, language, offset: offset, source-indices: source-indices)
  }
}

#let _render-code(code, options, language) = {
  let normalized = _normalize-code(code)
  if normalized == none {
    // Content bodies are retained as-is. String input is recommended when
    // source slicing, line numbers, and whitespace markers are required.
    block(width: 100%, breakable: true)[#code]
  } else {
    let lines = _prepare-source-lines(normalized, options)
    let selection = _line-selection(lines, options)
    _render-code-lines(selection.lines, options, language, offset: selection.offset, source-indices: selection.source-indices)
  }
}

#let _title(options) = {
  let title = _get(options, "title", default: none)
  if title == none { none } else {
    block(
      width: 100%,
      above: _get(options, "abovecaptionskip", default: 0.5em),
      below: _get(options, "belowcaptionskip", default: 0.5em),
      text(.._get(options, "title-style", default: (size: 8pt, weight: "bold", fill: rgb("3d4652"))))[
        #title
      ],
    )
  }
}

#let _listing-number-display(options) = context {
  let by-heading = _get(options, "numberbychapter", default: false)
  let listing-number = if by-heading { listings-chapter-counter.display() } else { listings-counter.display() }
  if by-heading {
    let heading-number = str(counter(heading).get().first())
    [#heading-number.#listing-number]
  } else {
    listing-number
  }
}

#let _caption(options) = {
  let caption = _get(options, "caption", default: none)
  if caption == none { none } else {
    if _get(options, "numberbychapter", default: false) { listings-chapter-counter.step() } else { listings-counter.step() }
    let display-name = _get(options, "listing-name", default: "Listing")
    let logical-name = _get(options, "name", default: none)
    block(width: 100%)[
      #metadata((kind: "listing", caption: caption, label: _get(options, "label", default: none), title: _get(options, "title", default: none), name: logical-name, listed: not _get(options, "nolol", default: false)))
      #block(
        width: 100%,
        above: _get(options, "abovecaptionskip", default: 0.5em),
        below: _get(options, "belowcaptionskip", default: 0.5em),
        text(.._get(options, "title-style", default: (size: 8pt, weight: "bold", fill: rgb("3d4652"))))[
          #display-name #_listing-number-display(options): #caption
        ],
      )
    ]
  }
}

#let _listing-radius(options) = {
  let rounded = _get(options, "frameround", default: none)
  if rounded == none or rounded == "" or rounded == "ffff" { 0pt } else { 4pt }
}

#let _listing-width(options) = {
  let width = _get(options, "linewidth", default: auto)
  if width == auto { 100% } else { width }
}

#let _listing-frame(options, body) = {
  let framesep = _get(options, "framesep", default: 3pt)
  let inset = (top: framesep + _get(options, "framextopmargin", default: 0pt), bottom: framesep + _get(options, "framexbottommargin", default: 0pt), left: framesep, right: framesep)
  let bg = _background(options)
  let stroke = _frame-stroke(options)
  let radius = _listing-radius(options)
  let left = _get(options, "xleftmargin", default: 0pt) + _get(options, "framexleftmargin", default: 0pt)
  let right = _get(options, "xrightmargin", default: 0pt) + _get(options, "framexrightmargin", default: 0pt)
  let shadow = _get(options, "frame", default: "none") == "shadowbox" and type(_get(options, "frameshape", default: none)) != dictionary
  let framed = if shadow {
    let rulesep = _get(options, "rulesep", default: 2pt)
    let rulecolor = _get(options, "rulecolor", default: rgb("7b8490"))
    let gap-color = _get(options, "rulesepcolor", default: rgb("b6bec8"))
    let inner = block(width: _listing-width(options), inset: inset, fill: bg, stroke: _get(options, "framerule", default: 0.4pt) + rulecolor, radius: radius, breakable: true, above: _get(options, "aboveskip", default: 0.5em), below: _get(options, "belowskip", default: 0.5em))[#body]
    block(width: _listing-width(options), inset: (top: rulesep, bottom: rulesep, left: rulesep, right: rulesep), fill: gap-color, stroke: _get(options, "framerule", default: 0.4pt) + rulecolor, radius: radius + rulesep, breakable: true)[#inner]
  } else {
    block(width: _listing-width(options), inset: inset, fill: bg, stroke: stroke, radius: radius, breakable: true, above: _get(options, "aboveskip", default: 0.5em), below: _get(options, "belowskip", default: 0.5em))[#body]
  }
  if left != 0pt or right != 0pt {
    block(width: 100%, inset: (left: left, right: right))[#framed]
  } else { framed }
}

#let _listing-frame-natural(options, body) = {
  let framesep = _get(options, "framesep", default: 3pt)
  let inset = (top: framesep + _get(options, "framextopmargin", default: 0pt), bottom: framesep + _get(options, "framexbottommargin", default: 0pt), left: framesep, right: framesep)
  let bg = _background(options)
  let stroke = _frame-stroke(options)
  let radius = _listing-radius(options)
  let left = _get(options, "xleftmargin", default: 0pt) + _get(options, "framexleftmargin", default: 0pt)
  let right = _get(options, "xrightmargin", default: 0pt) + _get(options, "framexrightmargin", default: 0pt)
  let side-stroke = if stroke == none {
    none
  } else if type(stroke) == dictionary {
    (left: stroke.at("left", default: none), right: stroke.at("right", default: none))
  } else {
    (left: stroke, right: stroke)
  }
  let top-stroke = if stroke == none { none } else if type(stroke) == dictionary { stroke.at("top", default: none) } else { stroke }
  let bottom-stroke = if stroke == none { none } else if type(stroke) == dictionary { stroke.at("bottom", default: none) } else { stroke }
  let top-edge = if top-stroke == none { [] } else { box(width: 100%, height: 0pt, stroke: (top: top-stroke))[] }
  let bottom-edge = if bottom-stroke == none { [] } else { box(width: 100%, height: 0pt, stroke: (bottom: bottom-stroke))[] }
  let wrapped = block(width: _listing-width(options), inset: inset, fill: bg, stroke: side-stroke, radius: radius, breakable: true, above: _get(options, "aboveskip", default: 0.5em), below: _get(options, "belowskip", default: 0.5em))[
    #top-edge
    #body
    #bottom-edge
  ]
  if left != 0pt or right != 0pt {
    block(width: 100%, inset: (left: left, right: right))[#wrapped]
  } else {
    wrapped
  }
}

#let _continuation-intro(options) = {
  if not _get(options, "show-continuation", default: false) {
    []
  } else {
    let label = _get(options, "continuation-label", default: none)
    let symbol = _get(options, "continuation-symbol", default: none)
    let label-text = if label == none { "" } else { str(label) }
    let symbol-text = if symbol == none { "" } else { str(symbol) }
    let separator = if label-text != "" and symbol-text != "" { " " } else { "" }
    let marker = label-text + separator + symbol-text
    let marker-style = _get(options, "continuation-style", default: (font: "DejaVu Sans Mono", size: 7pt, fill: rgb("68727e")))
    let rule = _get(options, "continuation-rule", default: none)
    block(width: 100%, above: 0pt, below: 0.35em)[
      #if rule != none {
        box(width: 100%, height: 0pt, stroke: (bottom: rule))[]
      }
      #if marker != "" {
        align(end)[#text(..marker-style, marker)]
      }
    ]
  }
}

#let _chunk-lines(lines, chunk-size) = {
  if type(chunk-size) != int or chunk-size <= 0 or lines.len() <= chunk-size {
    (lines,)
  } else {
    let chunks = ()
    let start = 0
    while start < lines.len() {
      let finish = calc.min(lines.len(), start + chunk-size)
      chunks = chunks + (lines.slice(start, finish),)
      start = finish
    }
    chunks
  }
}

#let _render-natural-code-table(lines, options, language, offset: 0, source-indices: none) = {
  let numbers = _get(options, "numbers", default: "none")
  let prepared = if numbers == none or numbers == "none" {
    let cells = lines.map(line => _render-raw-line(line, options, language))
    (columns: (1fr,), cells: cells)
  } else {
    let line-cells = _line-cells(lines, options, language, offset: offset, source-indices: source-indices)
    (columns: line-cells.at(0), cells: line-cells.at(1))
  }
  table(
    columns: prepared.columns,
    column-gutter: _get(options, "numbersep", default: 0.8em),
    row-gutter: _get(options, "lineskip", default: 0pt),
    inset: (x: 0pt, y: 0pt),
    stroke: none,
    ..prepared.cells,
  )
}

#let listings-show(doc) = {
  // Place this at document level as `#show: listings-show` or
  // `#show: doc => listings-show-layout(doc)`. The layout probe
  // records the available page size; multipage listings use it on the next pass.
  show heading: it => {
    if it.level == 1 {
      context {
        listings-chapter-counter.update(0)
        it
      }
    } else {
      it
    }
  }
  layout(size => {
    listings-layout-state.update((width: size.width, height: size.height))
    none
  })
  doc
}

#let listings-show-layout(doc) = listings-show(doc)

#let _everydisplay(options, body) = {
  let hook = _get(options, "everydisplay", default: none)
  if hook == none {
    body
  } else if type(hook) == function {
    hook(body)
  } else {
    [#hook #body]
  }
}

#let _multipage-listing(code, options, language) = context {
  let normalized = _normalize-code(code)
  if normalized == none {
    _listing-body(code, options, language)
  } else {
    let lines = _prepare-source-lines(normalized, options)
    let selection = _line-selection(lines, options)
    let selected = selection.lines
    let source-indices = selection.source-indices
    let offset = selection.offset
    let explicit = _get(options, "max-lines-per-page", default: none)
    let title = _title(options)
    let caption = _caption(options)
    layout(size => {
      let line-height = measure(_render-raw-line("M", options, language)).height
      let usable = calc.max(24pt, size.height - 70pt)
      let computed = calc.max(1, int(usable / calc.max(1pt, line-height)))
      listings-layout-state.update((width: size.width, height: size.height, lines: computed))
      none
    })
    let measured = listings-layout-state.final()
    let measured-lines = if type(measured) == dictionary { measured.at("lines", default: 40) } else { 40 }

    if type(explicit) == int and explicit > 0 {
      // Opt-in deterministic chunks remain available for fixtures and layouts
      // that need a prescribed page budget. The default path below is natural.
      let chunk-size = explicit
      let chunks = _chunk-lines(selected, chunk-size)
      let total = chunks.len()
      [
        #for (index, chunk) in chunks.enumerate() {
          if index > 0 { pagebreak() }
          let intro = if index == 0 {
            if _get(options, "captionpos", default: "b") == "t" and caption != none {
              [#caption #title]
            } else {
              [#title]
            }
          } else {
            _continuation-intro(options)
          }
          let ending = if index + 1 < total {
            []
          } else if _get(options, "captionpos", default: "b") == "b" and caption != none {
            [#caption]
          } else {
            []
          }
          let chunk-start = index * chunk-size
          let chunk-indices = source-indices.slice(chunk-start, chunk-start + chunk.len())
          _everydisplay(options, _listing-frame(options, [#intro #_render-code-lines(chunk, options, language, offset: offset + index * chunk-size, source-indices: chunk-indices) #ending]))
        }
      ]
    } else {
      // Natural mode lets Typst split the breakable table exactly where the
      // current page runs out of room. No artificial pagebreak is emitted.
      let start-page = here().position().page
      let intro = if _get(options, "captionpos", default: "b") == "t" and caption != none {
        [#caption #title]
      } else {
        [#title]
      }
      let ending = if _get(options, "captionpos", default: "b") == "b" and caption != none {
        [#caption]
      } else {
        []
      }
      _everydisplay(options, _listing-frame-natural(options, [
        #intro
        #_render-natural-code-table(selected, options, language, offset: offset, source-indices: source-indices)
        #ending
      ]))
    }
  }
}

#let lstlisting-multipage(code, language: none, options: (:), config: listings-defaults, styles: (:)) = {
  let options = _resolve-style-options(_merge-options(config, options + (language: language,)), styles)
  let chosen-language = _canonical-language(_lang(options, config), options: options)
  _multipage-listing(code, options + (multipage: true,), chosen-language)
}

#let listing-multipage = lstlisting-multipage

#let _attach-label(body, name) = {
  if name == none { body } else { block(width: 100%)[#body #label(str(name))] }
}

#let _float-placement(options) = {
  let spec = _get(options, "floatplacement", default: "htbp")
  if type(spec) == str and spec.contains("h") {
    auto
  } else if type(spec) == str and spec.contains("t") {
    top
  } else if type(spec) == str and spec.contains("b") {
    bottom
  } else {
    auto
  }
}

#let _listing-body(code, options, language) = {
  let title = _title(options)
  let caption = _caption(options)
  let content = _render-code(code, options, language)
  let framed = _listing-frame(options, [#title #content])
  if _get(options, "captionpos", default: "b") == "t" and caption != none {
    _listing-frame(options, [#caption #title #content])
  } else if caption != none {
    _listing-frame(options, [#title #content #caption])
  } else { framed }
}

#let _listing-float-render(code, options, language) = {
  let caption = _get(options, "caption", default: none)
  let label-name = _get(options, "label", default: none)
  let body = _listing-body(code, options + (caption: none, label: none, float: false), language)
  let supplement = _get(options, "listing-name", default: "Listing")
  let listing-caption = if caption == none { none } else {
    if _get(options, "numberbychapter", default: false) { listings-chapter-counter.step() } else { listings-counter.step() }
    metadata((kind: "listing", caption: caption, label: label-name, title: _get(options, "title", default: none), name: _get(options, "name", default: none), listed: not _get(options, "nolol", default: false)))
    [#supplement #_listing-number-display(options): #caption]
  }
  let result = if listing-caption == none {
    figure(body, kind: "listing", supplement: none, numbering: none, placement: _float-placement(options))
  } else {
    figure(body, kind: "listing", supplement: none, numbering: none, caption: listing-caption, placement: _float-placement(options))
  }
  if label-name == none { result } else { [#result #label(str(label-name))] }
}

#let _listing-columns-render(code, language, options, config, styles, count) = {
  let normalized = _normalize-code(code)
  let chosen-language = _canonical-language(_lang(options, config), options: options)
  if normalized == none or type(count) != int or count <= 1 {
    lstlisting(code, language: language, options: options, config: config, styles: styles)
  } else {
    let lines = _prepare-source-lines(normalized, options)
    let selection = _line-selection(lines, options)
    let selected = selection.lines
    let source-indices = selection.source-indices
    let chunk-size = calc.max(1, int((selected.len() + count - 1) / count))
    let chunks = _chunk-lines(selected, chunk-size)
    let caption = _caption(options)
    let title = _title(options)
    let rendered = ()
    for (index, chunk) in chunks.enumerate() {
      let chunk-start = index * chunk-size
      let chunk-indices = source-indices.slice(chunk-start, chunk-start + chunk.len())
      let intro = if index == 0 {
        if _get(options, "captionpos", default: "b") == "t" and caption != none {
          [#caption #title]
        } else { [#title] }
      } else { [] }
      let ending = if index + 1 == chunks.len() and _get(options, "captionpos", default: "b") == "b" and caption != none {
        [#caption]
      } else { [] }
      rendered.push(_listing-frame(options, [#intro #_render-code-lines(chunk, options, chosen-language, offset: selection.offset + chunk-start, source-indices: chunk-indices) #ending]))
    }
    let tracks = ()
    for _ in range(count) { tracks.push(1fr) }
    grid(columns: tracks, gutter: 1em, ..rendered)
  }
}

#let _lstlisting-render(code, language: none, options: (:), config: listings-defaults, styles: (:)) = {
  let options = _merge-options(config, options + (language: language,))
  let options = _resolve-style-options(options, styles)
  let chosen-language = _canonical-language(_lang(options, config), options: options)
  if not _get(options, "print", default: true) {
    let caption = _caption(options)
    let name = _get(options, "label", default: none)
    block(width: 100%)[
      #caption
      #if name != none { label(str(name)) }
    ]
  } else if _get(options, "float", default: false) {
    _everydisplay(options, _listing-float-render(code, options, chosen-language))
  } else if _get(options, "multicols", default: none) != none {
    let count = _get(options, "multicols", default: 2)
    _listing-columns-render(code, language, options, config, styles, count)
  } else if _get(options, "multipage", default: false) {
    let body = _multipage-listing(code, options, chosen-language)
    _attach-label(body, _get(options, "label", default: none))
  } else {
    let body = _listing-body(code, options, chosen-language)
    _attach-label(_everydisplay(options, body), _get(options, "label", default: none))
  }
}

#let _listing-source-selection(code, options) = {
  let normalized = _normalize-code(code)
  if normalized == none {
    (lines: (), offset: 0, source-indices: ())
  } else {
    let lines = _prepare-source-lines(normalized, options)
    _line-selection(lines, options)
  }
}

#let _listing-source-count(code, options) = _listing-source-selection(code, options).lines.len()

#let _listing-last-number(code, options) = {
  let selection = _listing-source-selection(code, options)
  if selection.lines.len() == 0 { 0 } else {
    _line-number-value(selection.lines.len() - 1, options, offset: selection.offset, source-index: selection.source-indices.last())
  }
}

#let lstlisting(code, language: none, options: (:), config: listings-defaults, styles: (:)) = {
  let merged = _resolve-style-options(_merge-options(config, options + (language: language,)), styles)
  let first = _get(merged, "firstnumber", default: "auto")
  let logical-name = _get(merged, "name", default: none)
  if logical-name == none {
    _lstlisting-render(code, language: language, options: options, config: config, styles: styles)
  } else {
    context {
      let counts = listings-name-state.get()
      let previous = if type(counts) == dictionary { counts.at(str(logical-name), default: 0) } else { 0 }
      let resolved = if first == "last" { options + (firstnumber: previous + 1) } else { options }
      let rendered = _lstlisting-render(code, language: language, options: resolved, config: config, styles: styles)
      let effective = _merge-options(merged, if first == "last" { (firstnumber: previous + 1,) } else { (:) })
      let current-last = _listing-last-number(code, effective)
      listings-name-state.update(old => {
        let base = if type(old) == dictionary { old } else { (:)}
        base + (str(logical-name): current-last)
      })
      [#rendered]
    }
  }
}

#let lstlisting-formfeed(code, language: none, options: (:), config: listings-defaults, styles: (:)) = {
  let normalized = _normalize-code(code)
  if normalized == none or not normalized.contains("\u{000C}") {
    lstlisting(code, language: language, options: options, config: config, styles: styles)
  } else {
    let parts = normalized.split("\u{000C}")
    let formfeed = _get(options, "formfeed", default: none)
    let output = ()
    for (index, part) in parts.enumerate() {
      if index > 0 {
        if type(formfeed) == function { output.push(formfeed()) } else { output.push(pagebreak()) }
      }
      output.push(lstlisting(part, language: language, options: options, config: config, styles: styles))
    }
    output.join()
  }
}

#let listing-formfeed = lstlisting-formfeed

#let lstlistoflistings(title: [List of Listings]) = context {
  let records = query(metadata).filter(item => type(item.value) == dictionary and item.value.at("kind", default: none) == "listing" and item.value.at("listed", default: true))
  heading(level: 1)[#title]
  if records.len() == 0 {
    [No listings recorded.]
  } else {
    for (index, item) in records.enumerate() {
      let value = item.value
      let caption = value.at("caption", default: [Untitled listing])
      let label-name = value.at("label", default: none)
      let row = [#strong([#str(index + 1). #h(0.4em)#caption])]
      if label-name == none { block(row) } else { block(link(label-name)[#row]) }
    }
  }
}

#let listings = lstlisting
#let listing = lstlisting
#let lst = lstlisting
#let listoflistings = lstlistoflistings

#let lstlisting-wrapped(code, language: none, options: (:), config: listings-defaults, styles: (:), wrap-chars: 80) = {
  lstlisting(code, language: language, options: options + (wrap-chars: wrap-chars, breaklines: true), config: config, styles: styles)
}

#let listing-wrapped = lstlisting-wrapped
#let lstlisting-breakable = lstlisting-wrapped

#let _listing-float-placement(options, requested) = {
  if requested != auto { requested } else { _float-placement(options) }
}

#let lstlisting-figure(code, language: none, options: (:), config: listings-defaults, styles: (:), placement: auto) = {
  let merged = _merge-options(config, options + (language: language,))
  let merged = _resolve-style-options(merged, styles)
  let caption = _get(merged, "caption", default: none)
  let label-name = _get(merged, "label", default: none)
  let body = lstlisting(code, language: language, options: merged + (caption: none, label: none, float: false), config: config, styles: styles)
  let supplement = _get(merged, "listing-name", default: "Listing")
  let actual-placement = _listing-float-placement(merged, placement)
  let listing-caption = if caption == none { none } else {
    if _get(merged, "numberbychapter", default: false) { listings-chapter-counter.step() } else { listings-counter.step() }
    metadata((kind: "listing", caption: caption, label: label-name, title: _get(merged, "title", default: none), name: _get(merged, "name", default: none), listed: not _get(merged, "nolol", default: false)))
    [#supplement #_listing-number-display(merged): #caption]
  }
  let result = if listing-caption == none {
    figure(body, kind: "listing", supplement: none, numbering: none, placement: actual-placement)
  } else {
    figure(body, kind: "listing", supplement: none, numbering: none, caption: listing-caption, placement: actual-placement)
  }
  if label-name == none {
    result
  } else {
    [#result #label(str(label-name))]
  }
}

#let listing-figure = lstlisting-figure
#let lstlisting-float = lstlisting-figure
#let listing-float = lstlisting-figure

#let lstlisting-columns(code, language: none, options: (:), config: listings-defaults, styles: (:), count: 2) = {
  let options = _resolve-style-options(_merge-options(config, options + (language: language,)), styles)
  _listing-columns-render(code, language, options, config, styles, count)
}

#let listing-multicols = lstlisting-columns
#let lstlisting-multicols = lstlisting-columns

// Typst-native alternatives for fancyvrb/useoutput. These keep source reading
// explicit and do not execute arbitrary TeX or external programs.
#let lstlisting-verbatim(code, language: none, options: (:), config: listings-defaults, styles: (:)) = {
  lstlisting(code, language: language, options: options, config: config, styles: styles)
}

#let listing-verbatim = lstlisting-verbatim
#let lstlisting-fancyvrb = lstlisting-verbatim

#let lstlisting-with-output(code, output, language: none, options: (:), config: listings-defaults, styles: (:), output-title: [Output]) = {
  let listing = lstlisting(code, language: language, options: options + (caption: none, label: none), config: config, styles: styles)
  let shown-output = block(
    width: 100%,
    inset: _get(options, "framesep", default: 3pt),
    fill: _background(options),
    stroke: _frame-stroke(options),
  )[
    #text(weight: "bold", output-title)
    #output
  ]
  grid(columns: (1fr, 1fr), gutter: 1em, listing, shown-output)
}

#let listing-with-output = lstlisting-with-output
#let lstlisting-useoutput = lstlisting-with-output

#let lstindex(term, target: none, style: none) = {
  metadata((kind: "listing-index", term: term, target: target, style: style))
}

#let _listing-index-term(value, fallback-style) = {
  let term = value.at("term", default: [Untitled])
  let own-style = value.at("style", default: none)
  let style = if own-style != none { own-style } else { fallback-style }
  if type(style) == function {
    style(term)
  } else if type(style) == dictionary {
    text(..style, term)
  } else {
    term
  }
}

#let lstlistingindex(title: [Listing Index], indexstyle: none) = context {
  let records = query(metadata).filter(item => type(item.value) == dictionary and item.value.at("kind", default: none) == "listing-index")
  heading(level: 1)[#title]
  if records.len() == 0 {
    [No listing index entries.]
  } else {
    for item in records {
      let value = item.value
      let shown = _listing-index-term(value, indexstyle)
      let target = value.at("target", default: none)
      if target == none { block(shown) } else { block(link(target)[#shown]) }
    }
  }
}

#let listing-index = lstindex
#let listoflistingindex = lstlistingindex

#let _inline-baseline(options) = {
  let position = _get(options, "boxpos", default: "c")
  if position == "b" { 0pt } else if position == "t" { 1em } else { 0.5em }
}

#let lstinline(code, language: none, options: (:), config: listings-defaults, styles: (:)) = {
  let options = _merge-options(config, options + (language: language,))
  let options = _resolve-style-options(options, styles)
  let chosen-language = _canonical-language(_lang(options, config), options: options)
  let custom = _custom-language-definition(options, chosen-language)
  let inline-content = if chosen-language == "arabic-code" {
    _code-style(_get(options, "basicstyle", default: none), _arabic-code-rendered-line(str(code), options))
  } else if custom != none {
    _code-style(_get(options, "basicstyle", default: none), _custom-code-line(_visible-whitespace(str(code), options), options, custom))
  } else if chosen-language == none or chosen-language == "" {
    raw(str(code))
  } else {
    _inkjet-highlighted-code(str(code), language: chosen-language)
  }
  box(
    baseline: _inline-baseline(options),
    inset: (x: 0.28em, y: 0.08em),
    fill: _background(options),
    stroke: _frame-stroke(options),
    radius: 2pt,
  )[#inline-content]
}

#let _decode-input(source, options) = {
  let decoder = _get(options, "input-decoder", default: none)
  if type(decoder) == function {
    let decoded = decoder(source, _get(options, "inputencoding", default: none))
    if type(decoded) == str { decoded } else { source }
  } else {
    source
  }
}

#let lstinputlisting(path, language: none, options: (:), config: listings-defaults, styles: (:)) = {
  let merged = _merge-options(config, options + (language: language,))
  let inputpath = _get(merged, "inputpath", default: none)
  let source-path = if inputpath == none or inputpath == "" { path } else { inputpath + "/" + path }
  let source = _decode-input(read(source-path), merged)
  lstlisting(source, language: language, options: options, config: config, styles: styles)
}

#let lstset-language(config, language) = config + (language: language)
#let listings-style-registry(base: (:)) = base
#let lstdefinestyle(styles, name, options: (:)) = styles + ((name): options)
#let lststyle(styles, name) = styles.at(name, default: (:))

#let lstdefinelanguage(registry, name, aliases: (), keywords: (), comments: (), strings: (), delim: (), keywordstyle: none, keywordstyles: (:), commentstyle: none, stringstyle: none, tagstyle: none, identifierstyle: none, emph: (), emphstyle: none, index: none, alsoletter: (), alsodigit: (), alsoother: (), tag: false, texcl: false, podcomment: false, dialect: none, dialects: (:), defaultdialect: none) = {
  registry + ((name): (aliases: aliases, keywords: keywords, comments: comments, strings: strings, delim: delim, keywordstyle: keywordstyle, keywordstyles: keywordstyles, commentstyle: commentstyle, stringstyle: stringstyle, tagstyle: tagstyle, identifierstyle: identifierstyle, emph: emph, emphstyle: emphstyle, index: index, alsoletter: alsoletter, alsodigit: alsodigit, alsoother: alsoother, tag: tag, texcl: texcl, podcomment: podcomment, dialect: dialect, dialects: dialects, defaultdialect: defaultdialect))
}

#let lstnewenvironment(name, base-options: (:), config: listings-defaults, styles: (:)) = {
  (code, language: none, options: (:)) => lstlisting(code, language: language, options: base-options + options, config: config, styles: styles)
}

#let lstMakeShortInline() = {
  // Typst has no catcode-level short-inline delimiter. Use lstinline directly.
  none
}

#let lstDeleteShortInline() = none

#let lstinputpath = lstinputlisting
#let lstinline-code = lstinline
#let listing-config = listings-config
#let listing-set = listings-set
#let listing-input = lstinputlisting
#let listing-inline = lstinline

#let listings-version = "0.1.0"

// mode 1: مستويات ICU4X مع ترقيم يتبع الفقرة.
// mode 2: مستويات ICU4X مع السماح بالمحايدات.
// mode 3: محاكاة babel onchar=ids fonts: فقرة كاملة واتصال اللغة/الخط.
#let _icu = plugin("icu4x-typst-bidi.wasm")

#let _latex-paragraph(s) = {
  let chars = s.codepoints()
  let records = str(_icu.resolve_language_runs(bytes(s))).split("\n")
  let base = int(records.at(0))
  let runs = ()
  for record in records.slice(1) {
    if record == "" { continue }
    let fields = record.split("\t")
    let start = int(fields.at(0))
    let end = int(fields.at(1))
    let lang = fields.at(2)
    runs.push(text(lang: lang)[#chars.slice(start, end).join()])
  }
  text(dir: if calc.odd(base) { rtl } else { ltr })[#runs.join()]
}

#let _segmented-paragraph(s, allow-neutral) = {
  let chars = s.codepoints()
  let policy = if allow-neutral { "\u{0001}" } else { "\u{0000}" }
  let records = str(_icu.resolve_segments_policy(bytes(policy + s))).split("\n")
  let base = int(records.at(0))
  let inner = ()
  for record in records.slice(1) {
    if record == "" { continue }
    let fields = record.split("\t")
    let start = int(fields.at(0))
    let end = int(fields.at(1))
    let level = int(fields.at(2))
    let dir = if calc.odd(level) { rtl } else { ltr }
    let chunk = chars.slice(start, end).join()
    let marked = str(_icu.mark_segment(bytes(if calc.odd(level) { "\u{0001}" + chunk } else { "\u{0000}" + chunk })))
    inner.push(text(dir: dir)[#marked])
  }
  text(dir: if calc.odd(base) { rtl } else { ltr })[#inner.join()]
}

#let _resolve-paragraph(s, mode: 1) = {
  if s == "" { return s }
  if mode == 3 { _latex-paragraph(s) }
  else { _segmented-paragraph(s, mode == 2) }
}

#let fix-bidi(body, mode: 1) = {
  show regex("[^\\n]+") : it => _resolve-paragraph(it.text, mode: mode)
  body
}

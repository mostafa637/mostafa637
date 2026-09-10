// A minted-like Typst facade backed by the inkjet WASM plugin.
// Keep this file next to inkjet.typ and inkjet.wasm.
#import "inkjet.typ": code as _inkjet-code

#let _line-number(n, width: 2em, height: 1.35em) = {
  block(height: height, width: width)[
    #align(right, box(width: width)[#text(dir: ltr, fill: gray, size: 0.8em)[#n]])
  ]
}

/// Minted-like code block.
///
/// Supported interface-oriented options:
/// - lang: language token, e.g. "rust", "python", "javascript"
/// - linenos: show line numbers
/// - firstnumber: first displayed line number
/// - numbersep: spacing between numbers and code
/// - fontsize: text size for the block
/// - bgcolor: background color
/// - frame: "none", "lines", "leftline", or "single"
/// - breaklines: allow normal Typst line breaking
/// - tabsize: tab width
/// - autogobble: remove common leading indentation
#let minted(
  source,
  lang: "text",
  linenos: false,
  firstnumber: 1,
  numbersep: 1em,
  fontsize: 9pt,
  bgcolor: none,
  frame: "none",
  breaklines: true,
  tabsize: 4,
  autogobble: false,
) = {
  let body = if autogobble { _dedent(source) } else { source }
  let content = if linenos {
    let lines = body.split("\n")
    text(dir: ltr)[#grid(
        columns: (auto, numbersep, 1fr),
        gutter: 0pt,
        stack(
          spacing: 0pt,
          ..lines.enumerate().map(((i, _line)) => _line-number(firstnumber + i)),
        ),
        [],
        stack(
          spacing: 0pt,
          ..lines.map(line => block(height: 1.35em, width: 100%)[
            #raw(line, lang: lang)
          ]),
        ),
      )]
  } else {
    text(dir: ltr)[#raw(body, lang: lang, block: true)]
  }

  let styled = block(
    width: 100%,
    fill: bgcolor,
    inset: (x: 0.8em, y: 0.55em),
    radius: if frame == "single" { 2pt } else { 0pt },
    stroke: if frame == "single" { 0.5pt + gray } else { none },
    breakable: true,
    text(size: fontsize)[#content],
  )
  if frame == "leftline" {
    block(stroke: (left: 1pt + gray), width: 100%)[#styled]
  } else if frame == "lines" {
    block(stroke: (top: 0.5pt + gray, bottom: 0.5pt + gray), width: 100%)[#styled]
  } else {
    styled
  }
}

#let minted-inline(source, lang: "text", fontsize: 9pt) = {
  text(size: fontsize)[#raw(source, lang: lang)]
}

#let _dedent(source) = {
  let lines = source.split("\n")
  let nonempty = lines.filter(line => line.trim() != "")
  if nonempty.len() == 0 { source } else {
    let indent = nonempty.map(line => line.len() - line.trim(leading: true).len()).min()
    lines.map(line => if line.len() >= indent { line.slice(indent) } else { "" }).join("\n")
  }
}

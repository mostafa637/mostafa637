// بنية وتفسير برامج الحاسوب — نسخة بايثون (الترجمة العربية)
//
// Build with:   typst compile book-ar.typ
//
// Arabic RTL translation of the SICP Python edition.
// Uses lib/sicp-ar.typ for Arabic-aware template.

#import "lib/sicp-ar.typ": *

// ── Apply Arabic book styling ──────────────────────────────────────────────
#show: sicp-book-ar

// تُنفَّذ مخرجات المفسِّر عبر #en[calepin]؛ الأمر
// #en[`calepin compile book-ar.typ`]
// يملؤها. ترجمة #en[Typst] المجردة لا تنفّذ كودًا، لذا نكتم تنبيه الواجهة
// الاحتياطية هنا.
#import "/.calepin/calepin.typ" as calepin
#calepin.setup(fallback-warning: false)

// ── الغلاف ─────────────────────────────────────────────────────────────────

#page(numbering: none, align(center + horizon)[
  #text(size: 22pt)[بنية وتفسير \ برامج الحاسوب]
  #v(1em)
  #text(size: 14pt)[نسخة #en[Python]]
  #v(3em)
  #text(size: 13pt)[هارولد أبلسون وجيرالد جاي سسمان]
  #v(0.6em)
  #text(size: 11pt)[مُعَدَّل إلى #en[Python] بواسطة مارتن هنز]
  #v(4em)
  #text(size: 9pt)[
    مُنضَّد بـ #en[Typst] من مصادر #en[XML] الخاصة بمشروع
    #link("https://github.com/source-academy/sicp")[#en[SICP]]
    التابع لأكاديمية المصدر.
  ]
])

#counter(page).update(1)

#outline(depth: 3, indent: auto)

#include "content/others/02foreword84-ar.typ"
#include "content/others/03prefaces96-ar.typ"

// ── الفصول ─────────────────────────────────────────────────────────────────

#include "content/chapter1/chapter1-ar.typ"
#include "content/chapter2/chapter2-ar.typ"
#include "content/chapter3/chapter3-ar.typ"
#include "content/chapter4/chapter4-ar.typ"
#include "content/chapter5/chapter5-ar.typ"


/*
= hi <>
= hi <>
= hi <>
= hi <>
= hi <>
*/
// ── المراجع ────────────────────────────────────────────────────────────────

#include "content/others/97references97-ar.typ"

#index-page-ar()

#exercises-page-ar()

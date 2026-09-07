// Master Typst Document for xv6 RISC-V Arabic Translation
// Imports native templates: listings.typ and fancyhdr.typ

#import "template/listings.typ": *
#import "template/fancyhdr.typ": *

// Global document configuration
#set text(
  font: ("DejaVu Sans", "Amiri", "Noto Naskh Arabic"),
  size: 10pt,
  lang: "ar",
  dir: rtl,
)

#set page(
  paper: "a4",
  margin: (top: 2.5cm, bottom: 2.5cm, left: 2cm, right: 2cm),
)

#set par(
  justify: true,
  leading: 0.8em,
)

// Initialize fancyhdr page style
#let book-style = fancy-style(
  direction: "rtl",
  headwidth: 100%,
  two-sided: true,
)

#let book-style = fancy-head(book-style, "LO,RE", [xv6: نظام تشغيل تعليمي بسيط شبيه بـ Unix])
#let book-style = fancy-head(book-style, "RO,LE", fancy-heading-mark(level: 1))
#let book-style = fancy-foot(book-style, "C", fancy-page-number())

#show: doc => fancy-apply(book-style, doc)
#show: listings-show

// Title Page
#align(center + horizon)[
  #v(2cm)
  #text(size: 22pt, weight: "bold")[xv6: نظام تشغيل تعليمي بسيط شبيه بـ Unix]
  
  #v(0.5cm)
  #text(size: 14pt, style: "italic")[الترجمة العربية الأكاديمية المعتمدة وفق معايير الترجمة البرمجية]
  
  #v(2cm)
  #text(size: 12pt)[
    *تأليف:* روس كوكس ، فرانس كاشوك ، روبيرت موريس \
    _قسم الهندسة الكهربائية وعلم الحاسوب - معهد ماساتشوستس للتكنولوجيا (MIT)_
  ]
  #v(3cm)
]

#pagebreak()

// Table of Contents
#outline(
  title: [جدول المحتويات],
  indent: 1.5em,
)

#pagebreak()

// Include all chapters in order from chapters/
#include "chapters/00_acks.typ"
#include "chapters/01_unix.typ"
#include "chapters/02_first.typ"
#include "chapters/03_mem.typ"
#include "chapters/04_trap.typ"
#include "chapters/05_pgfault.typ"
#include "chapters/06_interrupt.typ"
#include "chapters/07_lock.typ"
#include "chapters/08_sched.typ"
#include "chapters/09_sleep.typ"
#include "chapters/10_fs.typ"
#include "chapters/11_log.typ"
#include "chapters/12_lock2.typ"
#include "chapters/13_sum.typ"

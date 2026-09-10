// Bootstrap facade for the Calepin runtime.
//
// `calepin compile book.typ` (or `calepin compile book-ar.typ`) regenerates
// this directory: the real runtime plus the stored transcript results land
// here, and this file is replaced. Until the first such run, this facade
// re-exports the published compatibility package, so plain
//
//     typst compile book.typ
//
// works on a fresh checkout: every file compiles, and the interpreter
// transcripts simply render empty, because plain Typst cannot execute code.
//
// This file must stay committed — it is what makes the one-command build work
// before Calepin has ever run here. Do not edit it; it is regenerated.
#import "@preview/calepin:0.1.0": chunk, document, elements, inline, pages, results, setup, store

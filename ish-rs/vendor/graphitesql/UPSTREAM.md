# Vendored `graphitesql`

This directory vendors the pure-Rust SQLite implementation from
[`KarpelesLab/graphitesql` v0.1.7](https://github.com/KarpelesLab/graphitesql),
commit `4b5b7b0d51de1c89171292f3539148f356f84053`. It is released under the
SQLite Blessing / public-domain-style license; see `LICENSE`, `NOTICE`, and
`ATTRIBUTION.md`.

`graphitesql` is a from-scratch, safe Rust implementation of SQLite's SQL
surface and version-3 database-file format. It does **not** compile, link, or
call native SQLite. This profile retains its file and in-memory Rust engine and
has zero external Cargo dependencies.

The upstream optional `capi` and WebAssembly binding modules are intentionally
removed, along with their registry-only dependencies. The optional FTS5 and
Unicode integrations remain disabled. The retained profile has
`#![forbid(unsafe_code)]` and exposes only the native Rust library API.

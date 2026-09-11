# Vendored `redb`

This directory is the library source from
[`cberner/redb` v3.1.2](https://github.com/cberner/redb/tree/v3.1.2), commit
`c67e8ea634dfd4c83897641afa494e186fd255ea`, licensed under MIT OR Apache-2.0
(see `LICENSE-MIT` and `LICENSE-APACHE`).

It is intentionally vendored because this conversion environment cannot reach
the crates.io sparse index reliably. The profile retains redb's database engine
and its `std` file/in-memory backends, but removes optional `log`, `chrono`, and
`uuid` integrations and all development-only dependencies. Those changes leave
this copy with **zero external crate dependencies and no C/C++ database
library**. The database package is therefore buildable offline as a pure-Rust
library on the Rust 1.89+ toolchain required by upstream redb v3.

The upstream source is otherwise retained verbatim. The conditional code for
the removed optional integrations is compiled out with `#[cfg(any())]`; the
portable standard-library file backend remains unchanged.

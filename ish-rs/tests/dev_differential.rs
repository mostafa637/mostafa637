//! Differential test of `dev.rs` against unmodified `fs/dev.h` and
//! `fs/devices.h`.
//!
//! `tools/dev-dump.c` includes the real headers, so `dev_make`, `dev_major` and
//! `dev_minor` in `tests/fixtures/dev_reference.txt` are the header's own
//! arithmetic, and its `H`/`R` records are the *host libc's* `makedev`/`major`/
//! `minor` answers for the same numbers. That is what makes this file worth
//! replaying: the port transcribes the host encoding instead of calling it, and
//! the corpus is the host's opinion of the transcription.
//!
//! The corpus is every device `fs/devices.h` names plus the boundaries of the
//! encoding — a minor of `0x100000` (dropped), a major of `0x1000` (leaked into
//! the minor's high half) and `-1` for both, which wraps.
//!
//! Regenerate with:
//!
//! ```text
//! ISH_SRC=/path/to/ish ./tools/gen_dev_reference.sh
//! ```

use std::collections::BTreeMap;

use ish_emu::dev::{
    dev_fake_from_real, dev_major, dev_make, dev_minor, dev_real_from_fake, DevT, DEV_BLOCK,
    DEV_CHAR, DEV_CLIPBOARD_MINOR, DEV_CONSOLE_MINOR, DEV_FULL_MINOR, DEV_LOCATION_MINOR,
    DEV_NULL_MINOR, DEV_PTMX_MINOR, DEV_RANDOM_MINOR, DEV_TTY_MINOR, DEV_URANDOM_MINOR,
    DEV_ZERO_MINOR, DYN_DEV_MAJOR, MEM_MAJOR, TTY_ALTERNATE_MAJOR, TTY_CONSOLE_MAJOR,
    TTY_PSEUDO_MASTER_MAJOR, TTY_PSEUDO_SLAVE_MAJOR,
};

const FIXTURE: &str = concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/tests/fixtures/dev_reference.txt"
);

fn hex(text: &str) -> u64 {
    let digits = text.strip_prefix("0x").unwrap_or(text);
    u64::from_str_radix(digits, 16).unwrap_or_else(|err| panic!("invalid hex `{text}`: {err}"))
}

fn decimal(text: &str) -> i32 {
    text.parse()
        .unwrap_or_else(|err| panic!("invalid integer `{text}`: {err}"))
}

#[test]
fn every_device_encoding_matches_the_c_header() {
    let fixture = std::fs::read_to_string(FIXTURE)
        .unwrap_or_else(|err| panic!("cannot read {FIXTURE}: {err}"));

    // The fixture is a table keyed by dev number: how it decodes, what it
    // encodes back to, and what the host libc makes of it in both directions.
    let mut makes: Vec<(i32, i32, DevT)> = Vec::new();
    let mut decodes: BTreeMap<DevT, (i32, i32)> = BTreeMap::new();
    let mut backs: BTreeMap<DevT, DevT> = BTreeMap::new();
    let mut from_fake: BTreeMap<DevT, u64> = BTreeMap::new();
    let mut to_fake: BTreeMap<u64, DevT> = BTreeMap::new();
    let mut constants: BTreeMap<String, i32> = BTreeMap::new();
    let mut sizes: BTreeMap<String, usize> = BTreeMap::new();

    for line in fixture.lines() {
        if line.is_empty() || line.starts_with('#') {
            continue;
        }
        let fields: Vec<&str> = line.split(' ').collect();
        match fields[0] {
            "S" => {
                assert_eq!(fields.len(), 3, "malformed size record: {line}");
                sizes.insert(
                    fields[1].to_owned(),
                    fields[2]
                        .strip_prefix("size=")
                        .expect("size")
                        .parse()
                        .expect("size"),
                );
            }
            "M" => {
                assert_eq!(fields.len(), 4, "malformed make record: {line}");
                makes.push((
                    decimal(fields[1]),
                    decimal(fields[2]),
                    hex(fields[3].strip_prefix("dev=").expect("dev")) as DevT,
                ));
            }
            "D" => {
                assert_eq!(fields.len(), 4, "malformed decode record: {line}");
                let dev = hex(fields[1]) as DevT;
                let major = decimal(fields[2].strip_prefix("major=").expect("major"));
                let minor = decimal(fields[3].strip_prefix("minor=").expect("minor"));
                decodes.insert(dev, (major, minor));
            }
            "T" => {
                assert_eq!(fields.len(), 3, "malformed round-trip record: {line}");
                let dev = hex(fields[1]) as DevT;
                let back = hex(fields[2].strip_prefix("back=").expect("back")) as DevT;
                backs.insert(dev, back);
            }
            "H" => {
                assert_eq!(fields.len(), 5, "malformed fake-to-real record: {line}");
                let fake = hex(fields[3].strip_prefix("fake=").expect("fake")) as DevT;
                let real = hex(fields[4].strip_prefix("real=").expect("real"));
                from_fake.insert(fake, real);
            }
            "R" => {
                assert_eq!(fields.len(), 5, "malformed real-to-fake record: {line}");
                let real = hex(fields[3].strip_prefix("real=").expect("real"));
                let fake = hex(fields[4].strip_prefix("fake=").expect("fake")) as DevT;
                to_fake.insert(real, fake);
            }
            "K" => {
                assert_eq!(fields.len(), 2, "malformed constant record: {line}");
                let (name, value) = fields[1].split_once('=').expect("constant");
                constants.insert(name.to_owned(), decimal(value));
            }
            // The host macros' own arguments, which the port exposes only
            // through the two conversions above (so they are replayed, as far as
            // they can be, by the `H` and `R` records). The unit test beside the
            // transcription compares the `X` and `Y` records themselves.
            "X" | "Y" => {}
            other => panic!("unknown fixture record `{other}`: {line}"),
        }
    }

    assert_eq!(
        sizes.get("dev_t_"),
        Some(&std::mem::size_of::<DevT>()),
        "dev_t_ is not the width the fixture says"
    );
    assert_eq!(std::mem::size_of::<DevT>(), 4);

    // ---- the encoding -----------------------------------------------------
    assert_eq!(makes.len(), 28, "the corpus changed size");
    for (major, minor, expected) in &makes {
        assert_eq!(
            dev_make(*major, *minor),
            *expected,
            "dev_make({major}, {minor})"
        );
    }
    for (dev, (major, minor)) in &decodes {
        assert_eq!(dev_major(*dev), *major, "dev_major({dev:#x})");
        assert_eq!(dev_minor(*dev), *minor, "dev_minor({dev:#x})");
    }
    for (dev, back) in &backs {
        assert_eq!(
            dev_make(dev_major(*dev), dev_minor(*dev)),
            *back,
            "the round trip of {dev:#x}"
        );
    }

    // ---- the host encoding ------------------------------------------------
    assert_eq!(from_fake.len(), 28, "the fake-to-real corpus changed size");
    for (fake, real) in &from_fake {
        assert_eq!(
            dev_real_from_fake(*fake),
            *real,
            "dev_real_from_fake({fake:#x})"
        );
    }
    assert_eq!(to_fake.len(), 28, "the real-to-fake corpus changed size");
    for (real, fake) in &to_fake {
        assert_eq!(
            dev_fake_from_real(*real),
            *fake,
            "dev_fake_from_real({real:#x})"
        );
    }

    // ---- the constants ----------------------------------------------------
    let expected_constants: [(&str, i32); 18] = [
        ("DEV_BLOCK", DEV_BLOCK),
        ("DEV_CHAR", DEV_CHAR),
        ("MEM_MAJOR", MEM_MAJOR),
        ("DEV_NULL_MINOR", DEV_NULL_MINOR),
        ("DEV_ZERO_MINOR", DEV_ZERO_MINOR),
        ("DEV_FULL_MINOR", DEV_FULL_MINOR),
        ("DEV_RANDOM_MINOR", DEV_RANDOM_MINOR),
        ("DEV_URANDOM_MINOR", DEV_URANDOM_MINOR),
        ("TTY_CONSOLE_MAJOR", TTY_CONSOLE_MAJOR),
        ("TTY_ALTERNATE_MAJOR", TTY_ALTERNATE_MAJOR),
        ("DEV_TTY_MINOR", DEV_TTY_MINOR),
        ("DEV_CONSOLE_MINOR", DEV_CONSOLE_MINOR),
        ("DEV_PTMX_MINOR", DEV_PTMX_MINOR),
        ("TTY_PSEUDO_MASTER_MAJOR", TTY_PSEUDO_MASTER_MAJOR),
        ("TTY_PSEUDO_SLAVE_MAJOR", TTY_PSEUDO_SLAVE_MAJOR),
        ("DYN_DEV_MAJOR", DYN_DEV_MAJOR),
        ("DEV_CLIPBOARD_MINOR", DEV_CLIPBOARD_MINOR),
        ("DEV_LOCATION_MINOR", DEV_LOCATION_MINOR),
    ];
    assert_eq!(
        constants.len(),
        expected_constants.len(),
        "the fixture's constant list changed"
    );
    for (name, value) in expected_constants {
        assert_eq!(
            constants.get(name),
            Some(&value),
            "{name} differs from the C header"
        );
    }

    println!(
        "{} encodings, {} host conversions and {} constants matched the C",
        makes.len(),
        from_fake.len() + to_fake.len(),
        constants.len()
    );
}

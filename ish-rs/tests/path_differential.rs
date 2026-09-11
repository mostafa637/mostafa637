//! Differential test of `path.rs` against unmodified `fs/path.c`.
//!
//! `tools/path-dump.c` includes the real `fs/path.c`, so both functions in
//! `tests/fixtures/path_reference.txt` are the C's own, and the constants are
//! read off the real `kernel/fs.h` and `fs/path.h` macros. The `C` records are
//! one record per call to `path_next_component`, in the order the C's own
//! calling convention — `while (path_next_component(&path, component, &err))` —
//! makes them. Replaying is therefore threading: each record names the input it
//! was given, a step leaves the `rest` the next record will be given, and a
//! `ret=0` record is the call that ended the walk, either at the end of the
//! path or with the name that did not fit.
//!
//! The corpus is every shape a normalized path can have, the countershapels the
//! predicate has to reject, the bytes that are not text, and every component
//! length around `MAX_NAME` — the boundary where a name stops fitting is the one
//! piece of off-by-one arithmetic in the file.
//!
//! Regenerate with:
//!
//! ```text
//! ISH_SRC=/path/to/ish ./tools/gen_path_reference.sh
//! ```

use ish_emu::path::{
    path_is_normalized, path_next_component, ENAMETOOLONG, MAX_NAME, MAX_PATH, N_PARENT_DIR_WRITE,
    N_SYMLINK_FOLLOW, N_SYMLINK_NOFOLLOW,
};

const FIXTURE: &str = concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/tests/fixtures/path_reference.txt"
);

/// The corpus's shape, so a regenerated fixture that lost records is a failure
/// rather than a test that checks less.
const PREDICATES: usize = 69;
const STEPS: usize = 107;
const WALKS: usize = 64;
const LONG_NAMES: usize = 9;
const CONSTANTS: usize = 5;

/// The fixture's hex encoding, where `-` is the empty byte string.
fn bytes(field: &str) -> Vec<u8> {
    if field == "-" {
        return Vec::new();
    }
    assert_eq!(field.len() % 2, 0, "odd hex length in `{field}`");
    (0..field.len() / 2)
        .map(|index| u8::from_str_radix(&field[index * 2..index * 2 + 2], 16).expect("hex"))
        .collect()
}

fn quoted(raw: &[u8]) -> String {
    format!("{:?}", String::from_utf8_lossy(raw))
}

#[test]
fn the_path_helpers_match_the_c() {
    let fixture = std::fs::read_to_string(FIXTURE)
        .unwrap_or_else(|err| panic!("cannot read {FIXTURE}: {err}"));

    let mut constants: Vec<(String, i32)> = Vec::new();
    let mut predicates: Vec<(Vec<u8>, bool)> = Vec::new();
    let mut long_names: Vec<(usize, usize)> = Vec::new();
    let mut steps = 0usize;
    let mut walks = 0usize;
    // The path the current walk is positioned at. `None` between walks.
    let mut walking: Option<Vec<u8>> = None;

    for line in fixture.lines() {
        if line.is_empty() || line.starts_with('#') {
            continue;
        }
        let fields: Vec<&str> = line.split(' ').collect();
        match fields[0] {
            "K" => {
                assert_eq!(fields.len(), 2, "malformed constant record: {line}");
                let (name, value) = fields[1].split_once('=').expect("name=value");
                constants.push((name.to_owned(), value.parse().expect("integer")));
            }
            "P" => {
                assert_eq!(fields.len(), 3, "malformed predicate record: {line}");
                let path = bytes(fields[1]);
                let expected = match fields[2] {
                    "normalized=1" => true,
                    "normalized=0" => false,
                    other => panic!("unknown predicate `{other}`: {line}"),
                };
                assert_eq!(
                    path_is_normalized(&path),
                    expected,
                    "path_is_normalized({})",
                    quoted(&path)
                );
                predicates.push((path, expected));
            }
            "L" => {
                assert_eq!(fields.len(), 3, "malformed long name record: {line}");
                let length: usize = fields[1].parse().expect("length");
                let size: usize = fields[2]
                    .strip_prefix("bytes=")
                    .expect("bytes=")
                    .parse()
                    .expect("size");
                // The path is a slash and `length` characters, plus a terminator.
                assert_eq!(size, length + 1, "long name record: {line}");
                long_names.push((length, size));
            }
            "C" => {
                // The record's input is where the previous record left the walk.
                let input = bytes(fields[1]);
                match (&walking, fields.get(2)) {
                    (Some(position), _) => assert_eq!(
                        *position,
                        input,
                        "a walk's records do not chain: {} after {}",
                        quoted(&input),
                        quoted(position)
                    ),
                    // A walk starts at the first record after the last ret=0,
                    // and is counted when it ends (below), so that a walk whose
                    // first call already fails still counts once.
                    (None, Some(_)) => {}
                    (None, None) => panic!("malformed record: {line}"),
                }
                assert!(
                    input.is_empty() || input[0] == b'/',
                    "a walk started on a relative path: {}",
                    quoted(&input)
                );

                let returned = path_next_component(&input);
                match fields.get(2) {
                    Some(&"ret=1") => {
                        assert_eq!(fields.len(), 5, "malformed step record: {line}");
                        let name = bytes(fields[3].strip_prefix("component=").expect("component="));
                        let rest = bytes(fields[4].strip_prefix("rest=").expect("rest="));
                        let component =
                            returned
                                .expect("a step is not an error")
                                .unwrap_or_else(|| {
                                    panic!("the C took a component from {}", quoted(&input))
                                });
                        assert_eq!(component.name, name, "the component of {}", quoted(&input));
                        assert_eq!(component.rest, rest, "what follows in {}", quoted(&input));
                        // What the C's buffer size buys, and what the caller
                        // relies on: a step's name is always shorter than the
                        // buffer it would have been copied into.
                        assert!(name.len() < MAX_NAME, "a component is longer than MAX_NAME");
                        assert!(
                            rest.is_empty() || rest[0] == b'/',
                            "rest does not start at a slash: {}",
                            quoted(&rest)
                        );
                        assert!(!name.contains(&b'/'), "a component contains a slash");
                        steps += 1;
                        walking = Some(rest);
                    }
                    Some(&"ret=0") => {
                        assert_eq!(fields.len(), 4, "malformed end record: {line}");
                        let err: i32 = fields[3]
                            .strip_prefix("err=")
                            .expect("err=")
                            .parse()
                            .expect("errno");
                        match returned {
                            Ok(Some(component)) => panic!(
                                "the C stopped but the port took {} from {}",
                                quoted(component.name),
                                quoted(&input)
                            ),
                            Ok(None) => assert_eq!(err, 0, "the end of {}", quoted(&input)),
                            Err(code) => {
                                assert_eq!(code, err, "the error for {}", quoted(&input));
                                assert_eq!(code, ENAMETOOLONG);
                                // The only error in the corpus is a first
                                // component at least MAX_NAME bytes long, which
                                // is where the C's check has to fire.
                                let first = input[1..]
                                    .iter()
                                    .position(|&byte| byte == b'/')
                                    .unwrap_or(input.len() - 1);
                                assert!(
                                    first >= MAX_NAME,
                                    "a component of {first} bytes was rejected in {}",
                                    quoted(&input)
                                );
                            }
                        }
                        walking = None;
                        walks += 1;
                    }
                    other => panic!("unknown record `{other:?}`: {line}"),
                }
            }
            other => panic!("unknown fixture record `{other}`: {line}"),
        }
    }

    assert!(
        walking.is_none(),
        "the fixture ends inside a walk at {}",
        quoted(&walking.unwrap_or_default())
    );

    // ---- the constants ----------------------------------------------------
    assert_eq!(constants.len(), CONSTANTS, "the constant list changed");
    let expected: [(&str, i32); CONSTANTS] = [
        ("MAX_PATH", MAX_PATH as i32),
        ("MAX_NAME", MAX_NAME as i32),
        ("N_SYMLINK_FOLLOW", N_SYMLINK_FOLLOW),
        ("N_SYMLINK_NOFOLLOW", N_SYMLINK_NOFOLLOW),
        ("N_PARENT_DIR_WRITE", N_PARENT_DIR_WRITE),
    ];
    for (name, value) in expected {
        assert!(
            constants.iter().any(|(n, v)| n == name && *v == value),
            "{name} differs from the C"
        );
    }

    // ---- the corpus -------------------------------------------------------
    assert_eq!(predicates.len(), PREDICATES, "the predicate corpus changed");
    assert_eq!(long_names.len(), LONG_NAMES, "the long name corpus changed");
    assert_eq!(steps, STEPS, "the fixture lost step records");
    assert_eq!(walks, WALKS, "the fixture lost walks");

    // The boundary: the longest name that fits is MAX_NAME - 1, and the corpus
    // is built to straddle it.
    assert!(long_names.iter().any(|(length, _)| *length == MAX_NAME - 1));
    assert!(long_names.iter().any(|(length, _)| *length == MAX_NAME));
    let mut longest = vec![b'/'];
    longest.extend(std::iter::repeat_n(b'x', MAX_NAME - 1));
    assert_eq!(
        path_next_component(&longest)
            .expect("in range")
            .expect("a component")
            .name
            .len(),
        MAX_NAME - 1
    );

    println!(
        "{} predicates, {} steps over {} walks and {} constants matched the C",
        predicates.len(),
        steps,
        walks,
        constants.len()
    );
}

//! `fs/path.h` + `fs/path.c` — the part of the path layer that needs nothing
//! but its input.
//!
//! A guest path arrives as a `char *` from `user_read_string`, and every
//! filesystem operation starts by turning it into an absolute path with no
//! `.`, `..` or repeated slashes and no symlink left in it. That is
//! [`crate::path`]'s two larger functions — `__path_normalize` and
//! `path_normalize` — and they are *not* ported yet: they walk the mount tree
//! (`find_mount_and_trim_path`), call into the mounted filesystem
//! (`mount->fs->readlink` and `->stat`), check permissions (`access_check`) and
//! read `current->fs`'s root and working directory under its lock. Those are
//! `fs/mount.c`, `fs/fd.c`, `fs/generic.c` and `kernel/fs.c`, none of which
//! exists in this crate yet.
//!
//! What is ported here is what those functions, and the filesystems that
//! implement `lookup`, are allowed to assume: every path they see is
//! [normalized](path_is_normalized), and paths are consumed a component at a
//! time by [`path_next_component`]. The two are ports of the C's only two pure
//! functions, and they are checked against the real `fs/path.c` by
//! `tests/path_differential.rs`.
//!
//! # The one C behaviour the port cannot have
//!
//! `path_next_component` writes into a caller-supplied `char component` buffer
//! of `MAX_NAME + 1` bytes and, on `_ENAMETOOLONG`, returns false *after* having
//! copied `MAX_NAME` bytes with no terminator. Every caller aborts its walk when
//! the function returns false, so those bytes are never read and the quirk is
//! unobservable — but it is why the port returns a borrowed slice rather than
//! filling a buffer: there is no half-written buffer to leave behind. The corpus in
//! `tests/fixtures/path_reference.txt` records the error and not the partial
//! copy, which is also all a caller can see.
//!
//! A path that does not begin with `/` is an assertion failure in the C, and
//! [`path_next_component`] panics in the same place for the same reason: it is
//! called on paths [`path_is_normalized`] has accepted and on nothing else.
//!
//! # Strings and slices
//!
//! The C reads these functions' input as a `char *` that ends at its first NUL.
//! The port takes a `&[u8]` that ends where it ends, so bytes past a NUL would
//! be seen where the C stops — which cannot happen, because a path's bytes come
//! from `user_read_string`, and that is where the C and the port both stop at
//! the terminator.

/// `MAX_PATH` from `kernel/fs.h`: how big the buffer `path_normalize` writes
/// into has to be.
pub const MAX_PATH: usize = 4096;

/// `MAX_NAME` from `kernel/fs.h`: the longest single path component *including*
/// its terminator, so the longest name that fits is `MAX_NAME - 1` bytes.
pub const MAX_NAME: usize = 256;

/// `N_SYMLINK_FOLLOW` from `fs/path.h`: resolve a symlink in the last component.
pub const N_SYMLINK_FOLLOW: i32 = 1;
/// `N_SYMLINK_NOFOLLOW` from `fs/path.h`: leave the last component alone. The C
/// asserts that exactly one of the two is set.
pub const N_SYMLINK_NOFOLLOW: i32 = 2;
/// `N_PARENT_DIR_WRITE` from `fs/path.h`: also require write permission on the
/// containing directory, for calls that create or remove an entry.
pub const N_PARENT_DIR_WRITE: i32 = 4;

/// `_ENAMETOOLONG` from `kernel/errno.h`.
pub const ENAMETOOLONG: i32 = -36;

/// `path_is_normalized` — whether a path is of the form `path_normalize`
/// produces: a leading slash followed by plain components, one slash between
/// each pair.
///
/// The empty path is normalized (`path_normalize` can produce it and then
/// treats it as an error a level up), `"/"` is normalized, and `"/a/"` is too:
/// a trailing slash is a component of its own, which [`path_next_component`]
/// yields as an empty name.
#[must_use]
pub fn path_is_normalized(path: &[u8]) -> bool {
    let mut index = 0;
    while index < path.len() {
        if path[index] != b'/' {
            return false;
        }
        index += 1;
        if index < path.len() && path[index] == b'/' {
            return false;
        }
        while index < path.len() && path[index] != b'/' {
            index += 1;
        }
    }
    true
}

/// One component of a normalized path, and what follows it.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct NextComponent<'a> {
    /// the component's name, without the slash in front of it
    pub name: &'a [u8],
    /// the rest of the path: it starts at the next component's slash, or is
    /// empty when this was the last component
    pub rest: &'a [u8],
}

/// `path_next_component` — take the next component off a normalized path.
///
/// Returns `Ok(None)` at the end of the path, and `Err(ENAMETOOLONG)` when the
/// component is at least [`MAX_NAME`] bytes long (a name of `MAX_NAME - 1`
/// bytes fits, because the C's buffer holds a terminator too).
///
/// Two shapes are worth knowing about, because they come up when a caller is
/// looking for the last component:
///
/// * `"/"` yields one component with an empty name and an empty rest, not
///   `Ok(None)`;
/// * every component but the last is followed by `rest` starting with `/`, and
///   the last component of `"/a"` is followed by an empty `rest` — so `rest`
///   being empty, not the method returning `Ok(None)`, is what says "there is
///   nothing after this name". The C reads that off `*path == '\0'`.
///
/// # Panics
///
/// If `path` is nonempty and does not start with `/`, which in the C is an
/// `assert` on the contract that only normalized paths are walked.
pub fn path_next_component(path: &[u8]) -> Result<Option<NextComponent<'_>>, i32> {
    let Some(&first) = path.first() else {
        return Ok(None);
    };
    assert_eq!(
        first, b'/',
        "path_next_component needs a normalized path, got {path:?}"
    );

    let rest = &path[1..];
    let end = rest
        .iter()
        .position(|&byte| byte == b'/')
        .unwrap_or(rest.len());
    if end >= MAX_NAME {
        return Err(ENAMETOOLONG);
    }

    Ok(Some(NextComponent {
        name: &rest[..end],
        rest: &rest[end..],
    }))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn normalization_is_about_slashes_only() {
        for path in [
            &b""[..],
            b"/",
            b"/a",
            b"/a/b",
            b"/a/",
            b"/a b",
            b"/.",
            b"/..",
            b"/...",
            b"/a\tb",
            b"/ /",
        ] {
            assert!(
                path_is_normalized(path),
                "{:?} is normalized",
                String::from_utf8_lossy(path)
            );
        }

        for path in [&b"a"[..], b"a/", b"//", b"//a", b"/a//b", b"/a//", b" /a"] {
            assert!(
                !path_is_normalized(path),
                "{:?} is not normalized",
                String::from_utf8_lossy(path)
            );
        }
    }

    #[test]
    fn a_walk_visits_every_component_and_stops_at_the_end() {
        let mut rest = &b"/a/bb/ccc"[..];
        let mut names: Vec<&[u8]> = Vec::new();
        while let Some(component) = path_next_component(rest).expect("in range") {
            names.push(component.name);
            rest = component.rest;
        }
        assert_eq!(names, [&b"a"[..], b"bb", b"ccc"]);
        assert_eq!(rest, b"");
    }

    #[test]
    fn the_last_component_is_the_one_with_an_empty_rest() {
        // `*path == '\0'` in the C, and `rest.is_empty()` here, is how a caller
        // tells the last component from a middle one — `Ok(None)` never comes
        // up until the walk is over.
        let component = path_next_component(b"/a")
            .expect("in range")
            .expect("a component");
        assert_eq!(component.name, b"a");
        assert!(component.rest.is_empty());
        assert_eq!(path_next_component(component.rest), Ok(None));

        // A trailing slash is a component of its own, with an empty name.
        let component = path_next_component(b"/a/")
            .expect("in range")
            .expect("a component");
        assert_eq!(component.name, b"a");
        assert_eq!(component.rest, b"/");
        let component = path_next_component(component.rest)
            .expect("in range")
            .expect("a component");
        assert_eq!(component.name, b"");
        assert_eq!(component.rest, b"");
    }

    #[test]
    fn a_component_that_cannot_fit_is_named_too_long() {
        // MAX_NAME counts the terminator, so the longest name is one less.
        let mut longest = vec![b'/'];
        longest.extend(std::iter::repeat_n(b'x', MAX_NAME - 1));
        let component = path_next_component(&longest)
            .expect("the longest name fits")
            .expect("a component");
        assert_eq!(component.name.len(), MAX_NAME - 1);
        assert!(component.rest.is_empty());

        let mut too_long = vec![b'/'];
        too_long.extend(std::iter::repeat_n(b'x', MAX_NAME));
        assert_eq!(path_next_component(&too_long), Err(ENAMETOOLONG));
        assert_eq!(ENAMETOOLONG, -36);
    }

    #[test]
    #[should_panic(expected = "needs a normalized path")]
    fn a_relative_path_is_a_contract_violation() {
        let _ = path_next_component(b"a/b");
    }

    #[test]
    fn the_corpus_of_the_differential_test_agrees_with_these_examples() {
        // Guard against the fixture and these tests drifting apart.
        let fixture = std::fs::read_to_string(concat!(
            env!("CARGO_MANIFEST_DIR"),
            "/tests/fixtures/path_reference.txt"
        ))
        .expect("the fixture is part of the repository");
        assert!(fixture.contains("P 2f normalized=1"), "/ is normalized");
        assert!(fixture.contains("P 2f2f normalized=0"), "// is not");
        assert!(fixture.contains("K MAX_NAME=256"));
    }
}

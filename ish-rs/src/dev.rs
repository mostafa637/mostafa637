//! `fs/dev.h` + `fs/devices.h` — the device numbers iSH invents for the guest.
//!
//! Linux hands a userspace filesystem one 64-bit `dev_t` whose layout the kernel
//! chose; iSH instead keeps its own 32-bit encoding, "`mmmMMMmm`" in the
//! header's words, and only converts to the host's encoding at the two places
//! that have to talk to the host filesystem (`fs/real.c` creating a device node,
//! and its `stat` translations). Everything inside the emulator — `stat`
//! results, inodes, `mknod`, the device tables — speaks the 32-bit form.
//!
//! ```text
//!  31              20 19         8 7      0
//! +-----------------+------------+--------+
//! |   minor[19:8]   |  major     | minor[7:0] |
//! +-----------------+------------+--------+
//! ```
//!
//! Two consequences of that layout are load-bearing and are pinned by the
//! differential corpus rather than described here:
//!
//! * the major gets twelve bits and the minor twenty, so a major of `0x1000`
//!   leaks into the minor's high half and a minor of `0x100000` is dropped;
//! * the encoding is a bijection on 32 bits, so a [`DevT`] is an opaque value:
//!   anything the guest passes in can be decoded and re-encoded without loss.
//!
//! [`DevT`] is `uint32_t` in C, but `dev_make` does its arithmetic in `int` and
//! can overflow it — a negative minor shifts into the sign bit. The port
//! computes in `u32` and wraps, which is what the machine did; the corpus
//! records `dev_make(-1, -1) == 0xffffffff` for exactly that reason.
//!
//! # Deliberate differences
//!
//! * **The host's encoding is transcribed, not called.** C's
//!   `dev_real_from_fake` calls the host libc's `makedev`, and
//!   `dev_fake_from_real` calls `major`/`minor`. This crate declares no host
//!   functions, so the glibc macros' arithmetic is written out in
//!   [`dev_real_from_fake`] and [`dev_fake_from_real`] — and the oracle prints
//!   the host macros' answers next to them, including the truncating cases, so
//!   the transcription is checked against the host rather than against a
//!   reading of it. Two limbs of those macros are unreachable from the fake
//!   encoding and so cannot be checked through them: `dev_major` never returns
//!   more than twelve bits, so the major `makedev` puts at bit 32 is always
//!   zero, and the bits a wider `major` mask would add land where the minor's
//!   high half already is. The fixture therefore also records the host macros'
//!   answers for arguments of their own (its `X` and `Y` records), which the
//!   unit test below replays. A host whose `makedev` differs (macOS encodes
//!   device numbers another way) needs its own pair here.
//! * **`dev_open` and the device tables are not ported yet.** `struct dev_ops`
//!   holds an `fd_ops`, `dev_open` assigns one to a `struct fd`, and both live
//!   in `fs/dev.c` with the fd layer; that also means the `char_devs` table
//!   this header's file declares cannot be filled in until `mem`, `tty` and
//!   `dyndev` exist.

/// `dev_t_`: iSH's 32-bit device number.
pub type DevT = u32;

/// `dev_make(major, minor)` — the C computes this in `int` and overflows for
/// large or negative inputs; the port wraps in `u32`, which is what the machine
/// did and what the corpus records.
#[must_use]
pub fn dev_make(major: i32, minor: i32) -> DevT {
    ((minor as u32 & 0xfff00) << 12) | ((major as u32) << 8) | (minor as u32 & 0xff)
}

/// `dev_major(dev)` — the middle twelve bits.
#[must_use]
pub fn dev_major(dev: DevT) -> i32 {
    ((dev & 0xfff00) >> 8) as i32
}

/// `dev_minor(dev)` — the low eight bits and the top twelve.
#[must_use]
pub fn dev_minor(dev: DevT) -> i32 {
    (((dev & 0xfff00000) >> 12) | (dev & 0xff)) as i32
}

/// The host libc's `makedev`, as glibc spells it on a 64-bit host: the minor
/// keeps its low eight bits and gets bits 12 and up, the major keeps its low
/// twelve bits and gets bits 8..19, and the major's upper bits go to bit 32.
///
/// The arguments are `int` here where the macro takes an `unsigned int`; the
/// two agree because every operation is truncated to the width the macro's
/// `unsigned long long` arithmetic leaves, which is what the `X` records check.
fn host_makedev(major: i32, minor: i32) -> u64 {
    let major = major as u32 as u64;
    let minor = minor as u32 as u64;
    (minor & 0xff) | ((major & 0xfff) << 8) | ((minor & !0xff) << 12) | ((major & !0xfff) << 32)
}

/// The host libc's `major(dev)`, including the truncation to `unsigned int` the
/// glibc macro performs on the high half.
fn host_major(dev: u64) -> i32 {
    (((dev >> 8) & 0xfff) as u32 | ((dev >> 32) & !0xfff_u64) as u32) as i32
}

/// The host libc's `minor(dev)`, same truncation.
fn host_minor(dev: u64) -> i32 {
    ((dev & 0xff) as u32 | ((dev >> 12) & !0xff_u64) as u32) as i32
}

/// `dev_real_from_fake(dev)`: iSH's device number in the host's encoding, for
/// the one place that has to hand one to the host filesystem.
#[must_use]
pub fn dev_real_from_fake(dev: DevT) -> u64 {
    host_makedev(dev_major(dev), dev_minor(dev))
}

/// `dev_fake_from_real(dev)`: a host device number in iSH's encoding. A host
/// number with more than twenty minor bits loses them, exactly as in C.
#[must_use]
pub fn dev_fake_from_real(dev: u64) -> DevT {
    dev_make(host_major(dev), host_minor(dev))
}

/// `DEV_BLOCK`: the block-device half of `dev_open`'s dispatch.
pub const DEV_BLOCK: i32 = 0;
/// `DEV_CHAR`: the character-device half.
pub const DEV_CHAR: i32 = 1;

/// `MEM_MAJOR`: `/dev/null`, `/dev/zero`, `/dev/full`, `/dev/random`,
/// `/dev/urandom`.
pub const MEM_MAJOR: i32 = 1;
/// `DEV_NULL_MINOR`.
pub const DEV_NULL_MINOR: i32 = 3;
/// `DEV_ZERO_MINOR`.
pub const DEV_ZERO_MINOR: i32 = 5;
/// `DEV_FULL_MINOR`.
pub const DEV_FULL_MINOR: i32 = 7;
/// `DEV_RANDOM_MINOR`.
pub const DEV_RANDOM_MINOR: i32 = 8;
/// `DEV_URANDOM_MINOR`.
pub const DEV_URANDOM_MINOR: i32 = 9;

/// `TTY_CONSOLE_MAJOR`: `/dev/ttyX`, where X is the minor.
pub const TTY_CONSOLE_MAJOR: i32 = 4;

/// `TTY_ALTERNATE_MAJOR`: `/dev/tty`, `/dev/console`, `/dev/ptmx`.
pub const TTY_ALTERNATE_MAJOR: i32 = 5;
/// `DEV_TTY_MINOR`.
pub const DEV_TTY_MINOR: i32 = 0;
/// `DEV_CONSOLE_MINOR`.
pub const DEV_CONSOLE_MINOR: i32 = 1;
/// `DEV_PTMX_MINOR`.
pub const DEV_PTMX_MINOR: i32 = 2;

/// `TTY_PSEUDO_MASTER_MAJOR`.
pub const TTY_PSEUDO_MASTER_MAJOR: i32 = 128;
/// `TTY_PSEUDO_SLAVE_MAJOR`.
pub const TTY_PSEUDO_SLAVE_MAJOR: i32 = 136;

/// `DYN_DEV_MAJOR`: the devices iOS adds, `/dev/clipboard` and `/dev/gps`.
pub const DYN_DEV_MAJOR: i32 = 240;
/// `DEV_CLIPBOARD_MINOR`.
pub const DEV_CLIPBOARD_MINOR: i32 = 0;
/// `DEV_LOCATION_MINOR`.
pub const DEV_LOCATION_MINOR: i32 = 1;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn the_encoding_is_the_documented_layout() {
        // /dev/null: major 1 in bits 8..19, minor 3 in bits 0..7.
        assert_eq!(dev_make(MEM_MAJOR, DEV_NULL_MINOR), 0x103);
        assert_eq!(dev_major(0x103), MEM_MAJOR);
        assert_eq!(dev_minor(0x103), DEV_NULL_MINOR);

        // A pseudo-terminal master puts the minor across both halves.
        assert_eq!(dev_make(TTY_PSEUDO_MASTER_MAJOR, 0), 0x8000);
        assert_eq!(dev_make(TTY_PSEUDO_MASTER_MAJOR, 0x1ff), 0x1080ff);
        assert_eq!(dev_major(0x1080ff), TTY_PSEUDO_MASTER_MAJOR);
        assert_eq!(dev_minor(0x1080ff), 0x1ff);
    }

    #[test]
    fn the_boundaries_drop_and_leak_as_the_c_does() {
        // A minor past bit 19 is masked away, so it encodes as no minor at all.
        assert_eq!(dev_make(1, 0x100000), 0x100);
        assert_eq!(dev_minor(0x100000), 0x100);

        // A major past bit 11 is not masked in `dev_make`, so it leaks into the
        // minor's high half: 0x1000 << 8 is bit 20.
        assert_eq!(dev_make(0x1000, 0), 0x100000);
        assert_eq!(dev_minor(dev_make(0x1000, 0)), 0x100);
        assert_eq!(dev_major(dev_make(0x1000, 0)), 0);
    }

    #[test]
    fn negative_fields_wrap_like_the_machine_did() {
        assert_eq!(dev_make(-1, -1), 0xffffffff);
        assert_eq!(dev_major(0xffffffff), 0xfff);
        assert_eq!(dev_minor(0xffffffff), 0xfffff);
        assert_eq!(dev_make(-1, 0), 0xffffff00);
        assert_eq!(dev_make(0, -1), 0xfff000ff);
    }

    #[test]
    fn decoding_always_encodes_back_to_the_same_number() {
        // The two halves cover every bit, so a `dev_t_` survives a trip through
        // `dev_major`/`dev_minor` — which is why iSH can keep one opaque.
        let mut state = 0x1234_5678u32;
        for _ in 0..4096 {
            // xorshift, so the sample is not all in one field
            state ^= state << 13;
            state ^= state >> 17;
            state ^= state << 5;
            assert_eq!(dev_make(dev_major(state), dev_minor(state)), state);
        }
    }

    #[test]
    fn the_host_encoding_agrees_with_the_libc() {
        // The values the corpus pins, so a change to the transcription shows up
        // in a unit test as well as in the differential replay.
        assert_eq!(dev_real_from_fake(0x103), 0x103);
        assert_eq!(dev_fake_from_real(0x103), 0x103);
        assert_eq!(dev_real_from_fake(0xfff001ff), 0xfff001ff);
        assert_eq!(dev_fake_from_real(0xfff001ff), 0xfff001ff);

        // The host's minor has room the fake one does not: makedev(1, 0x100000)
        // is 0x100000100, and coming back through the fake encoding drops the
        // bit that has nowhere to go.
        assert_eq!(dev_real_from_fake(dev_make(1, 0x100000)), 0x100);
        assert_eq!(dev_fake_from_real(0x1_0000_0100), 0x100);
        assert_eq!(dev_real_from_fake(0x100), 0x100);

        // The host's `major`/`minor` truncate the high half to 32 bits, so an
        // all-ones device number comes back as all ones.
        assert_eq!(dev_fake_from_real(u64::MAX), 0xffffffff);
    }

    #[test]
    fn the_host_macros_match_the_host_even_where_the_corpus_cannot_see_them() {
        // `dev_real_from_fake` only ever hands `makedev` a major of twelve bits,
        // and `dev_fake_from_real` drops whatever the host's extra fields hold,
        // so the two conversions above cannot check those limbs. The fixture's
        // `X` and `Y` records do: they are the host libc's own answers for
        // arguments of its own, and this test is where the private
        // transcription can be compared with them.
        let fixture = std::fs::read_to_string(concat!(
            env!("CARGO_MANIFEST_DIR"),
            "/tests/fixtures/dev_reference.txt"
        ))
        .expect("the fixture is part of the repository");

        let mut makedevs = 0;
        let mut decodes = 0;
        for line in fixture.lines() {
            let fields: Vec<&str> = line.split(' ').collect();
            match fields[0] {
                "X" => {
                    assert_eq!(fields.len(), 4, "malformed makedev record: {line}");
                    let major: i32 = fields[1].parse().expect("major");
                    let minor: i32 = fields[2].parse().expect("minor");
                    // `%#x` prints zero as `0`, so the `0x` is optional.
                    let hex = fields[3].strip_prefix("makedev=").expect("makedev=");
                    let expected = u64::from_str_radix(hex.strip_prefix("0x").unwrap_or(hex), 16)
                        .expect("hex");
                    assert_eq!(
                        host_makedev(major, minor),
                        expected,
                        "makedev({major}, {minor})"
                    );
                    makedevs += 1;
                }
                "Y" => {
                    assert_eq!(fields.len(), 4, "malformed host decode record: {line}");
                    let hex = fields[1];
                    let dev = u64::from_str_radix(hex.strip_prefix("0x").unwrap_or(hex), 16)
                        .expect("hex");
                    let major: i32 = fields[2]
                        .strip_prefix("major=")
                        .expect("major=")
                        .parse()
                        .expect("major");
                    let minor: i32 = fields[3]
                        .strip_prefix("minor=")
                        .expect("minor=")
                        .parse()
                        .expect("minor");
                    assert_eq!(host_major(dev), major, "major({dev:#x})");
                    assert_eq!(host_minor(dev), minor, "minor({dev:#x})");
                    decodes += 1;
                }
                _ => {}
            }
        }

        // The records that make this test worth having: a device number whose
        // major's bits 12..15 are set, which a twelve-bit mask drops and a wider
        // one would not.
        assert!(makedevs >= 14, "the makedev corpus shrank to {makedevs}");
        assert!(decodes >= 28, "the host decode corpus shrank to {decodes}");
        assert_eq!(
            host_major(0x00f0_0000),
            0,
            "bits 20..23 of dev are not the major"
        );
        assert_eq!(
            host_minor(0x00f0_0000),
            0xf00,
            "they are the minor's middle"
        );
    }
}

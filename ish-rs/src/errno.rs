//! `kernel/errno.c` — the host→guest errno translation.
//!
//! Every syscall that fails ends here. `err_map` turns the number the *host*
//! kernel returned into the number the *guest* expects, and `errno_map` is the
//! same thing plus the one side effect the C attaches to it: a failed write to a
//! closed pipe raises `SIGPIPE` in the guest before the error is returned.
//!
//! The table is generated, not transcribed — see `tools/gen_errno_table.py`. The
//! guest half is the i386 Linux ABI and never moves; the host half is whatever
//! the compiling platform calls these, so it is asked for rather than assumed.
//! On the Linux host this table was generated on, all 82 entries happen to be
//! the identity (`host == -guest`); on a Darwin host they would not be, which is
//! the whole reason the C does the mapping at all.
//!
//! What is left over, and what most of the behaviour actually is, is the
//! fallback: an errno the C does not know becomes `-(err | 0x1000)`, which is a
//! negative number far outside the guest's errno range, so the guest sees an
//! error it cannot name rather than a plausible-looking one.

use crate::errno_table;

pub use crate::errno_table::HOST_EPIPE;

/// The `SIGPIPE_` vector the guest expects, from `kernel/signal.h`.
pub const SIGPIPE: i32 = 13;

/// `err_map` — translate a host errno into the negated guest errno.
///
/// Unknown values take the fallback described in the module documentation, which
/// in the C is accompanied by a `printk("unknown error %d")`. Use
/// [`is_unknown`] to find out whether that path was taken.
pub fn err_map(err: i32) -> i32 {
    for &(host, guest) in errno_table::HOST_TO_GUEST {
        if host == err {
            return guest;
        }
    }
    -(err | 0x1000)
}

/// Whether [`err_map`] would fall through to its `printk` path for `err`.
pub fn is_unknown(err: i32) -> bool {
    !errno_table::HOST_TO_GUEST.iter().any(|&(host, _)| host == err)
}

/// What [`errno_map`] did.
///
/// The C performs the signal delivery itself, through the thread-local `current`
/// task. Signal delivery is not ported, so the fact is returned instead of acted
/// on: `sigpipe` is exactly the condition under which the C would have called
/// `send_signal(current, SIGPIPE_, SIGINFO_NIL)`.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Mapped {
    /// the guest errno to hand back to the caller
    pub guest: i32,
    /// whether the guest should also be sent `SIGPIPE`
    pub sigpipe: bool,
}

/// `errno_map` — translate the host's `errno`, raising `SIGPIPE` on `EPIPE`.
pub fn errno_map(host_errno: i32) -> Mapped {
    Mapped {
        guest: err_map(host_errno),
        sigpipe: host_errno == HOST_EPIPE,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn the_table_covers_every_case_the_c_lists() {
        // kernel/errno.c has 83 ERRCASEs; ENOTSUP and EOPNOTSUPP are the same
        // number on Linux, so the switch has 82 distinct labels
        assert_eq!(errno_table::HOST_TO_GUEST.len(), 82);
        let mut hosts: Vec<i32> = errno_table::HOST_TO_GUEST.iter().map(|&(h, _)| h).collect();
        hosts.sort_unstable();
        hosts.dedup();
        assert_eq!(hosts.len(), 82, "no host number appears twice");
    }

    #[test]
    fn known_errnos_map_to_the_guest_numbers() {
        assert_eq!(err_map(1), -1, "EPERM");
        assert_eq!(err_map(2), -2, "ENOENT");
        assert_eq!(err_map(11), -11, "EAGAIN");
        assert_eq!(err_map(95), -95, "EOPNOTSUPP");
        assert_eq!(err_map(122), -122, "EDQUOT");
        assert!(!is_unknown(1));
    }

    #[test]
    fn an_unknown_errno_becomes_something_the_guest_cannot_mistake() {
        // -(err | 0x1000): a negative number well outside the errno range
        assert_eq!(err_map(200), -(200 | 0x1000));
        assert_eq!(err_map(200), -4296);
        assert!(is_unknown(200));
        // and the bit is an or, not an add, so an already-set bit is left alone
        assert_eq!(err_map(0x1000), -0x1000);
        assert_eq!(err_map(0x1001), -0x1001);
    }

    #[test]
    fn zero_is_not_a_known_errno_either() {
        // nothing maps 0, so it takes the fallback like anything else: -4096,
        // which does *not* read as success to the guest
        assert_eq!(err_map(0), -4096);
        assert!(is_unknown(0));
    }

    #[test]
    fn only_epipe_raises_sigpipe() {
        assert_eq!(errno_map(HOST_EPIPE), Mapped { guest: -32, sigpipe: true });
        assert_eq!(errno_map(1), Mapped { guest: -1, sigpipe: false });
        assert_eq!(errno_map(200), Mapped { guest: -(200 | 0x1000), sigpipe: false });
    }
}

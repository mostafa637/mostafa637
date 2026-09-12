//! `kernel/time.h` — time structures and conversions.
//!
//! Leaf header plus the pure conversion helpers from `time.c`. The syscall
//! implementations themselves depend on `fs/poll.h`, `resource`, and host
//! timers, so they remain for a later layer. This module ports the guest ABI
//! structures and the `clock_from_timeval` / `convert_timespec` helpers.

/// `CLOCK_*` constants from `time.h`.
pub const CLOCK_REALTIME: u32 = 0;
pub const CLOCK_MONOTONIC: u32 = 1;
pub const CLOCK_PROCESS_CPUTIME_ID: u32 = 2;
pub const CLOCK_REALTIME_COARSE: u32 = 5;

/// `ITIMER_*`
pub const ITIMER_REAL: u32 = 0;
pub const ITIMER_VIRTUAL: u32 = 1;
pub const ITIMER_PROF: u32 = 2;

/// `struct timeval_` — guest ABI.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct Timeval {
    pub sec: u32,
    pub usec: u32,
}

/// `struct timespec_` — guest ABI.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct Timespec {
    pub sec: u32,
    pub nsec: u32,
}

/// `struct timezone_`
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct Timezone {
    pub minuteswest: u32,
    pub dsttime: u32,
}

/// `struct itimerval_`
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct Itimerval {
    pub interval: Timeval,
    pub value: Timeval,
}

/// `struct itimerspec_`
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct Itimerspec {
    pub interval: Timespec,
    pub value: Timespec,
}

/// `struct tms_`
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct Tms {
    pub tms_utime: u32,
    pub tms_stime: u32,
    pub tms_cutime: u32,
    pub tms_cstime: u32,
}

/// `clock_t_` from `misc.h` — iSH defines it as 32-bit.
pub type ClockT = u32;

/// `clock_from_timeval` — `timeval.sec * 100 + usec / 10000`
pub fn clock_from_timeval(tv: Timeval) -> ClockT {
    tv.sec * 100 + tv.usec / 10000
}

/// Host `timespec` conversion helpers, mirroring C's `convert_timespec` and
/// `convert_timeval` that turn guest structs into host `struct timespec`.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct HostTimespec {
    pub sec: i64,
    pub nsec: i64,
}

pub fn convert_timespec(t: Timespec) -> HostTimespec {
    HostTimespec {
        sec: t.sec as i64,
        nsec: t.nsec as i64,
    }
}

pub fn convert_timeval(t: Timeval) -> HostTimespec {
    HostTimespec {
        sec: t.sec as i64,
        nsec: t.usec as i64 * 1000,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn clock_from_timeval_matches_c() {
        let tv = Timeval { sec: 1, usec: 50000 };
        assert_eq!(clock_from_timeval(tv), 105);
        let tv2 = Timeval { sec: 0, usec: 9999 };
        assert_eq!(clock_from_timeval(tv2), 0);
    }

    #[test]
    fn convert_timespec_and_timeval_match_c() {
        let ts = Timespec { sec: 5, nsec: 123 };
        let host = convert_timespec(ts);
        assert_eq!(host.sec, 5);
        assert_eq!(host.nsec, 123);

        let tv = Timeval { sec: 2, usec: 3000 };
        let host2 = convert_timeval(tv);
        assert_eq!(host2.sec, 2);
        assert_eq!(host2.nsec, 3_000_000);
    }

    #[test]
    fn constants_match_c_header() {
        assert_eq!(CLOCK_REALTIME, 0);
        assert_eq!(CLOCK_MONOTONIC, 1);
        assert_eq!(ITIMER_REAL, 0);
    }
}

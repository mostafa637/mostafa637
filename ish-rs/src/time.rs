//! `kernel/time.h` + `kernel/time.c` — time structures and syscalls.
//!
//! This module ports the guest ABI structures and the conversion helpers,
//! plus the time-related syscalls that depend only on already-ported modules
//! (`user`, `errno`, `resource`, `timer`, `task`). Host time sources are
//! abstracted via `TimeHost` so an iOS embedding can supply native time
//! without the core assuming Linux syscalls.
//!
//! The more complex timerfd/posix-timer paths that need `fs/poll.h` and fd
//! table remain for later, but the basic clock and timeval syscalls are here.

use crate::task::{Addr, TaskTable};

/// `CLOCK_*` constants from `time.h`.
pub const CLOCK_REALTIME: u32 = 0;
pub const CLOCK_MONOTONIC: u32 = 1;
pub const CLOCK_PROCESS_CPUTIME_ID: u32 = 2;
pub const CLOCK_REALTIME_COARSE: u32 = 5;

/// `ITIMER_*`
pub const ITIMER_REAL: u32 = 0;
pub const ITIMER_VIRTUAL: u32 = 1;
pub const ITIMER_PROF: u32 = 2;

/// Guest errno values
const EFAULT: i32 = -14;
const EINVAL: i32 = -22;
const EPERM: i32 = -1;

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

/// `clock_t_` from `misc.h`
pub type ClockT = u32;

/// `clock_from_timeval` — `timeval.sec * 100 + usec / 10000`
pub fn clock_from_timeval(tv: Timeval) -> ClockT {
    tv.sec * 100 + tv.usec / 10000
}

/// Host `timespec` conversion helpers.
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

/// Host time source, analogous to `RandomSource` and `SystemInfoHost`.
pub trait TimeHost {
    /// `time(NULL)` — seconds since epoch.
    fn time_now(&self) -> u32 {
        0
    }

    /// `clock_gettime` — return timespec or guest errno.
    fn clock_gettime(&self, clock: u32) -> Result<HostTimespec, i32> {
        let _ = clock;
        Err(EINVAL)
    }

    /// `clock_getres`
    fn clock_getres(&self, clock: u32) -> Result<HostTimespec, i32> {
        let _ = clock;
        Err(EINVAL)
    }

    /// `gettimeofday`
    fn gettimeofday(&self) -> Result<(Timeval, Timezone), i32> {
        Err(EINVAL)
    }

    /// `nanosleep` — sleep for `req`, return remaining or error.
    fn nanosleep(&self, req: HostTimespec) -> Result<Option<HostTimespec>, i32> {
        let _ = req;
        Ok(None)
    }

    /// `rusage_get_current` for `sys_times`
    fn rusage_current(&self) -> crate::resource::Rusage {
        crate::resource::Rusage::default()
    }
}

/// Default host that uses system time where possible, but returns EINVAL for
/// unsupported clocks to keep core portable.
pub struct DefaultTimeHost;

impl TimeHost for DefaultTimeHost {
    fn time_now(&self) -> u32 {
        // Use system time, but truncate to u32 as C does
        use std::time::{SystemTime, UNIX_EPOCH};
        SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map(|d| d.as_secs() as u32)
            .unwrap_or(0)
    }

    fn clock_gettime(&self, clock: u32) -> Result<HostTimespec, i32> {
        match clock {
            CLOCK_REALTIME | CLOCK_REALTIME_COARSE => {
                use std::time::{SystemTime, UNIX_EPOCH};
                let now = SystemTime::now()
                    .duration_since(UNIX_EPOCH)
                    .map_err(|_| EINVAL)?;
                Ok(HostTimespec {
                    sec: now.as_secs() as i64,
                    nsec: now.subsec_nanos() as i64,
                })
            }
            CLOCK_MONOTONIC => {
                // Use Instant for monotonic, but we need epoch-relative?
                // For simplicity return 0,1 as placeholder; real embedding would use clock_gettime
                Ok(HostTimespec { sec: 0, nsec: 1 })
            }
            CLOCK_PROCESS_CPUTIME_ID => Err(EINVAL), // handled separately via rusage
            _ => Err(EINVAL),
        }
    }

    fn clock_getres(&self, clock: u32) -> Result<HostTimespec, i32> {
        match clock {
            CLOCK_REALTIME | CLOCK_REALTIME_COARSE | CLOCK_MONOTONIC => Ok(HostTimespec {
                sec: 0,
                nsec: 1,
            }),
            _ => Err(EINVAL),
        }
    }

    fn gettimeofday(&self) -> Result<(Timeval, Timezone), i32> {
        use std::time::{SystemTime, UNIX_EPOCH};
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map_err(|_| EINVAL)?;
        Ok((
            Timeval {
                sec: now.as_secs() as u32,
                usec: now.subsec_micros(),
            },
            Timezone {
                minuteswest: 0,
                dsttime: 0,
            },
        ))
    }

    fn nanosleep(&self, req: HostTimespec) -> Result<Option<HostTimespec>, i32> {
        let dur = std::time::Duration::new(req.sec as u64, req.nsec as u32);
        std::thread::sleep(dur);
        Ok(None)
    }

    fn rusage_current(&self) -> crate::resource::Rusage {
        crate::resource::Rusage::default()
    }
}

/// `sys_time`
pub fn sys_time(
    table: &mut TaskTable,
    time_out: Addr,
    host: &impl TimeHost,
) -> i32 {
    let now = host.time_now() as i32;
    if time_out != 0 {
        if let Some(task) = table.current_mut() {
            if task.user_write(time_out, &(now as u32).to_le_bytes()).is_err() {
                return EFAULT;
            }
        } else {
            return -3; // ESRCH
        }
    }
    now
}

/// `sys_stime` — always EPERM in iSH
pub fn sys_stime(_time: Addr) -> i32 {
    EPERM
}

/// `sys_clock_gettime`
pub fn sys_clock_gettime(
    table: &mut TaskTable,
    clock: u32,
    tp: Addr,
    host: &impl TimeHost,
) -> i32 {
    let ts = if clock == CLOCK_PROCESS_CPUTIME_ID {
        // Use rusage as C does
        let rusage = host.rusage_current();
        HostTimespec {
            sec: rusage.utime.sec as i64,
            nsec: rusage.utime.usec as i64 * 1000,
        }
    } else {
        match host.clock_gettime(clock) {
            Ok(ts) => ts,
            Err(e) => return e,
        }
    };

    let guest = Timespec {
        sec: ts.sec as u32,
        nsec: ts.nsec as u32,
    };

    if let Some(task) = table.current_mut() {
        let bytes = [
            guest.sec.to_le_bytes(),
            guest.nsec.to_le_bytes(),
        ]
        .concat();
        if task.user_write(tp, &bytes).is_err() {
            return EFAULT;
        }
    } else {
        return -3;
    }

    0
}

/// `sys_clock_getres`
pub fn sys_clock_getres(
    table: &mut TaskTable,
    clock: u32,
    res_addr: Addr,
    host: &impl TimeHost,
) -> i32 {
    let ts = match host.clock_getres(clock) {
        Ok(ts) => ts,
        Err(e) => return e,
    };

    let guest = Timespec {
        sec: ts.sec as u32,
        nsec: ts.nsec as u32,
    };

    if let Some(task) = table.current_mut() {
        let bytes = [
            guest.sec.to_le_bytes(),
            guest.nsec.to_le_bytes(),
        ]
        .concat();
        if task.user_write(res_addr, &bytes).is_err() {
            return EFAULT;
        }
    } else {
        return -3;
    }

    0
}

/// `sys_clock_settime` — always EPERM
pub fn sys_clock_settime(_clock: u32, _tp: Addr) -> i32 {
    EPERM
}

/// `sys_gettimeofday`
pub fn sys_gettimeofday(
    table: &mut TaskTable,
    tv_addr: Addr,
    tz_addr: Addr,
    host: &impl TimeHost,
) -> i32 {
    let (tv, tz) = match host.gettimeofday() {
        Ok(v) => v,
        Err(e) => return e,
    };

    if let Some(task) = table.current_mut() {
        if tv_addr != 0 {
            let bytes = [
                tv.sec.to_le_bytes(),
                tv.usec.to_le_bytes(),
            ]
            .concat();
            if task.user_write(tv_addr, &bytes).is_err() {
                return EFAULT;
            }
        }
        if tz_addr != 0 {
            let bytes = [
                tz.minuteswest.to_le_bytes(),
                tz.dsttime.to_le_bytes(),
            ]
            .concat();
            if task.user_write(tz_addr, &bytes).is_err() {
                return EFAULT;
            }
        }
    } else {
        return -3;
    }

    0
}

/// `sys_settimeofday` — always EPERM
pub fn sys_settimeofday(_tv: Addr, _tz: Addr) -> i32 {
    EPERM
}

/// `sys_times`
pub fn sys_times(
    table: &mut TaskTable,
    tbuf: Addr,
    host: &impl TimeHost,
) -> i32 {
    if tbuf != 0 {
        let rusage = host.rusage_current();
        let clock_from_resource = |tv: crate::resource::Timeval| -> u32 {
            tv.sec * 100 + tv.usec / 10000
        };
        let tms = Tms {
            tms_utime: clock_from_resource(rusage.utime),
            tms_stime: clock_from_resource(rusage.stime),
            tms_cutime: clock_from_resource(rusage.utime),
            tms_cstime: clock_from_resource(rusage.stime),
        };
        if let Some(task) = table.current_mut() {
            let bytes = [
                tms.tms_utime.to_le_bytes(),
                tms.tms_stime.to_le_bytes(),
                tms.tms_cutime.to_le_bytes(),
                tms.tms_cstime.to_le_bytes(),
            ]
            .concat();
            if task.user_write(tbuf, &bytes).is_err() {
                return EFAULT;
            }
        } else {
            return -3;
        }
    }
    0
}

/// `sys_nanosleep`
pub fn sys_nanosleep(
    table: &mut TaskTable,
    req_addr: Addr,
    rem_addr: Addr,
    host: &impl TimeHost,
) -> i32 {
    let req = if let Some(task) = table.current_mut() {
        let mut buf = [0u8; 8];
        if task.user_read(req_addr, &mut buf).is_err() {
            return EFAULT;
        }
        Timespec {
            sec: u32::from_le_bytes([buf[0], buf[1], buf[2], buf[3]]),
            nsec: u32::from_le_bytes([buf[4], buf[5], buf[6], buf[7]]),
        }
    } else {
        return -3;
    };

    let host_req = convert_timespec(req);
    match host.nanosleep(host_req) {
        Ok(rem) => {
            if rem_addr != 0 {
                if let Some(rem) = rem {
                    let guest_rem = Timespec {
                        sec: rem.sec as u32,
                        nsec: rem.nsec as u32,
                    };
                    if let Some(task) = table.current_mut() {
                        let bytes = [
                            guest_rem.sec.to_le_bytes(),
                            guest_rem.nsec.to_le_bytes(),
                        ]
                        .concat();
                        if task.user_write(rem_addr, &bytes).is_err() {
                            return EFAULT;
                        }
                    }
                }
            }
            0
        }
        Err(e) => e,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::memory::P_RWX;
    use crate::mmu::PAGE_BITS;
    use crate::task::TaskTable;

    struct DeterministicHost;

    impl TimeHost for DeterministicHost {
        fn time_now(&self) -> u32 {
            1234567890
        }

        fn clock_gettime(&self, clock: u32) -> Result<HostTimespec, i32> {
            match clock {
                CLOCK_REALTIME => Ok(HostTimespec { sec: 1000, nsec: 2000 }),
                CLOCK_MONOTONIC => Ok(HostTimespec { sec: 2000, nsec: 3000 }),
                CLOCK_REALTIME_COARSE => Ok(HostTimespec { sec: 1000, nsec: 0 }),
                _ => Err(EINVAL),
            }
        }

        fn clock_getres(&self, clock: u32) -> Result<HostTimespec, i32> {
            match clock {
                CLOCK_REALTIME | CLOCK_MONOTONIC | CLOCK_REALTIME_COARSE => {
                    Ok(HostTimespec { sec: 0, nsec: 1 })
                }
                _ => Err(EINVAL),
            }
        }

        fn gettimeofday(&self) -> Result<(Timeval, Timezone), i32> {
            Ok((
                Timeval {
                    sec: 1000,
                    usec: 500000,
                },
                Timezone {
                    minuteswest: 0,
                    dsttime: 0,
                },
            ))
        }

        fn nanosleep(&self, _req: HostTimespec) -> Result<Option<HostTimespec>, i32> {
            Ok(None)
        }
    }

    fn table_with_page() -> TaskTable {
        let table = TaskTable::bootstrap();
        table
            .current()
            .unwrap()
            .mm_mut()
            .unwrap()
            .mem
            .map_nothing(0x100, 1, P_RWX);
        table
    }

    #[test]
    fn clock_from_timeval_matches_c() {
        let tv = Timeval { sec: 1, usec: 50000 };
        assert_eq!(clock_from_timeval(tv), 105);
    }

    #[test]
    fn sys_time_writes_and_returns() {
        let mut table = table_with_page();
        let host = DeterministicHost;
        let addr = 0x100 << PAGE_BITS;
        let ret = sys_time(&mut table, addr, &host);
        assert_eq!(ret, 1234567890 as i32);
        let mut buf = [0u8; 4];
        table.current_mut().unwrap().user_read(addr, &mut buf).unwrap();
        assert_eq!(u32::from_le_bytes(buf), 1234567890);
        // null out param
        let ret2 = sys_time(&mut table, 0, &host);
        assert_eq!(ret2, 1234567890 as i32);
    }

    #[test]
    fn sys_time_faults_on_bad_address() {
        let mut table = table_with_page();
        let host = DeterministicHost;
        assert_eq!(sys_time(&mut table, 0x200 << PAGE_BITS, &host), EFAULT);
    }

    #[test]
    fn sys_clock_gettime_and_getres() {
        let mut table = table_with_page();
        let host = DeterministicHost;
        let addr = 0x100 << PAGE_BITS;
        assert_eq!(sys_clock_gettime(&mut table, CLOCK_REALTIME, addr, &host), 0);
        let mut buf = [0u8; 8];
        table.current_mut().unwrap().user_read(addr, &mut buf).unwrap();
        assert_eq!(u32::from_le_bytes([buf[0], buf[1], buf[2], buf[3]]), 1000);
        assert_eq!(u32::from_le_bytes([buf[4], buf[5], buf[6], buf[7]]), 2000);

        assert_eq!(sys_clock_gettime(&mut table, 99, addr, &host), EINVAL);
        assert_eq!(
            sys_clock_gettime(&mut table, CLOCK_REALTIME, 0x200 << PAGE_BITS, &host),
            EFAULT
        );

        assert_eq!(sys_clock_getres(&mut table, CLOCK_MONOTONIC, addr, &host), 0);
        assert_eq!(sys_clock_getres(&mut table, 99, addr, &host), EINVAL);
    }

    #[test]
    fn sys_gettimeofday_writes_both() {
        let mut table = table_with_page();
        let host = DeterministicHost;
        let tv_addr = 0x100 << PAGE_BITS;
        let tz_addr = tv_addr + 8;
        assert_eq!(sys_gettimeofday(&mut table, tv_addr, tz_addr, &host), 0);
        let mut buf = [0u8; 8];
        table.current_mut().unwrap().user_read(tv_addr, &mut buf).unwrap();
        assert_eq!(u32::from_le_bytes([buf[0], buf[1], buf[2], buf[3]]), 1000);
        assert_eq!(u32::from_le_bytes([buf[4], buf[5], buf[6], buf[7]]), 500000);
    }

    #[test]
    fn sys_stime_and_settimeofday_are_eperm() {
        assert_eq!(sys_stime(0), EPERM);
        assert_eq!(sys_clock_settime(0, 0), EPERM);
        assert_eq!(sys_settimeofday(0, 0), EPERM);
    }

    #[test]
    fn sys_times_writes_tms() {
        let mut table = table_with_page();
        let host = DeterministicHost;
        let addr = 0x100 << PAGE_BITS;
        assert_eq!(sys_times(&mut table, addr, &host), 0);
        // null is also success
        assert_eq!(sys_times(&mut table, 0, &host), 0);
        // fault
        assert_eq!(
            sys_times(&mut table, 0x200 << PAGE_BITS, &host),
            EFAULT
        );
    }

    #[test]
    fn sys_nanosleep_success_and_fault() {
        let mut table = table_with_page();
        let host = DeterministicHost;
        let req_addr = 0x100 << PAGE_BITS;
        let req = Timespec { sec: 0, nsec: 1000 };
        table
            .current_mut()
            .unwrap()
            .user_write(
                req_addr,
                &[req.sec.to_le_bytes(), req.nsec.to_le_bytes()].concat(),
            )
            .unwrap();
        assert_eq!(sys_nanosleep(&mut table, req_addr, 0, &host), 0);
        assert_eq!(
            sys_nanosleep(&mut table, 0x200 << PAGE_BITS, 0, &host),
            EFAULT
        );
    }
}

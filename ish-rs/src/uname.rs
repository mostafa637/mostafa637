//! `kernel/uname.c` — guest uname and sysinfo layouts over explicit host data.
//!
//! The C source combines fixed iSH identity strings with three host-dependent
//! observations: `uname(2)`'s node name, platform uptime/load values, and host
//! memory statistics. [`SystemInfoHost`] makes those observations explicit so
//! the portable Rust core neither fabricates iOS data nor reaches a Linux-only
//! API directly.

use crate::group::{EPERM, ESRCH};
use crate::task::{Addr, TaskTable};

/// `_EFAULT`.
pub const EFAULT: i32 = -14;
/// `UNAME_LENGTH` from `kernel/calls.h`, including the terminating NUL.
pub const UNAME_LENGTH: usize = 65;

/// The six fixed-length C strings in `struct uname`.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Uname {
    /// `system`
    pub system: [u8; UNAME_LENGTH],
    /// `hostname`
    pub hostname: [u8; UNAME_LENGTH],
    /// `release`
    pub release: [u8; UNAME_LENGTH],
    /// `version`
    pub version: [u8; UNAME_LENGTH],
    /// `arch`
    pub arch: [u8; UNAME_LENGTH],
    /// `domain`
    pub domain: [u8; UNAME_LENGTH],
}

impl Default for Uname {
    fn default() -> Self {
        Self {
            system: [0; UNAME_LENGTH],
            hostname: [0; UNAME_LENGTH],
            release: [0; UNAME_LENGTH],
            version: [0; UNAME_LENGTH],
            arch: [0; UNAME_LENGTH],
            domain: [0; UNAME_LENGTH],
        }
    }
}

impl Uname {
    /// Guest ABI size of `struct uname`.
    pub const SIZE: usize = UNAME_LENGTH * 6;

    /// Encode the exact guest `struct uname` field order.
    pub fn to_le_bytes(self) -> [u8; Self::SIZE] {
        let mut bytes = [0; Self::SIZE];
        for (index, field) in [
            self.system,
            self.hostname,
            self.release,
            self.version,
            self.arch,
            self.domain,
        ]
        .into_iter()
        .enumerate()
        {
            let start = index * UNAME_LENGTH;
            bytes[start..start + UNAME_LENGTH].copy_from_slice(&field);
        }
        bytes
    }
}

/// Data returned by the platform's uptime/load query.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct UptimeInfo {
    /// `uptime_ticks` from `platform/platform.h`.
    pub uptime_ticks: u64,
    /// `load_1m`.
    pub load_1m: u64,
    /// `load_5m`.
    pub load_5m: u64,
    /// `load_15m`.
    pub load_15m: u64,
}

/// The host `sysinfo` values that iSH copies into its guest ABI.
///
/// `kernel/uname.c` intentionally leaves guest `bufferram` zero, so it is not
/// a field here even on hosts whose native `sysinfo` reports it.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct HostSysInfo {
    /// Native `totalram`.
    pub totalram: u64,
    /// Native `freeram`.
    pub freeram: u64,
    /// Native `sharedram`.
    pub sharedram: u64,
    /// Native `totalswap`.
    pub totalswap: u64,
    /// Native `freeswap`.
    pub freeswap: u64,
    /// Native process count.
    pub procs: u16,
    /// Native `totalhigh`.
    pub totalhigh: u64,
    /// Native `freehigh`.
    pub freehigh: u64,
    /// Native `mem_unit`.
    pub mem_unit: u32,
}

/// The i386-compatible `struct sys_info` written by `sys_sysinfo`.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct SysInfo {
    /// `uptime`.
    pub uptime: u32,
    /// `loads[3]`.
    pub loads: [u32; 3],
    /// `totalram`.
    pub totalram: u32,
    /// `freeram`.
    pub freeram: u32,
    /// `sharedram`.
    pub sharedram: u32,
    /// `bufferram`, deliberately zeroed by iSH's C implementation.
    pub bufferram: u32,
    /// `totalswap`.
    pub totalswap: u32,
    /// `freeswap`.
    pub freeswap: u32,
    /// `procs`.
    pub procs: u16,
    /// `totalhigh`.
    pub totalhigh: u32,
    /// `freehigh`.
    pub freehigh: u32,
    /// `mem_unit`.
    pub mem_unit: u32,
}

fn put_u32(bytes: &mut [u8], offset: usize, value: u32) {
    bytes[offset..offset + 4].copy_from_slice(&value.to_le_bytes());
}

impl SysInfo {
    /// The C structure has two alignment bytes after `procs` and three trailing
    /// alignment bytes after `pad`: `sizeof(struct sys_info) == 60`.
    pub const SIZE: usize = 60;

    /// Construct C's zero-initialized `struct sys_info` plus its assignments.
    pub fn from_host(uptime: UptimeInfo, host: HostSysInfo) -> Self {
        Self {
            uptime: uptime.uptime_ticks as u32,
            loads: [
                uptime.load_1m as u32,
                uptime.load_5m as u32,
                uptime.load_15m as u32,
            ],
            totalram: host.totalram as u32,
            freeram: host.freeram as u32,
            sharedram: host.sharedram as u32,
            // sysinfo_specific never copies host bufferram.
            bufferram: 0,
            totalswap: host.totalswap as u32,
            freeswap: host.freeswap as u32,
            procs: host.procs,
            totalhigh: host.totalhigh as u32,
            freehigh: host.freehigh as u32,
            mem_unit: host.mem_unit,
        }
    }

    /// Encode `struct sys_info`, including C's zero-filled padding bytes.
    pub fn to_le_bytes(self) -> [u8; Self::SIZE] {
        let mut bytes = [0; Self::SIZE];
        put_u32(&mut bytes, 0, self.uptime);
        put_u32(&mut bytes, 4, self.loads[0]);
        put_u32(&mut bytes, 8, self.loads[1]);
        put_u32(&mut bytes, 12, self.loads[2]);
        put_u32(&mut bytes, 16, self.totalram);
        put_u32(&mut bytes, 20, self.freeram);
        put_u32(&mut bytes, 24, self.sharedram);
        put_u32(&mut bytes, 28, self.bufferram);
        put_u32(&mut bytes, 32, self.totalswap);
        put_u32(&mut bytes, 36, self.freeswap);
        bytes[40..42].copy_from_slice(&self.procs.to_le_bytes());
        // bytes 42..44 remain the first C alignment pad.
        put_u32(&mut bytes, 44, self.totalhigh);
        put_u32(&mut bytes, 48, self.freehigh);
        put_u32(&mut bytes, 52, self.mem_unit);
        // byte 56 (`pad`) and bytes 57..60 (tail alignment) remain zero.
        bytes
    }
}

/// Values corresponding to the C globals `uname_version` and
/// `uname_hostname_override`, plus the build macros used in C's version field.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct UnameConfig {
    /// C's mutable `uname_version` global; upstream initializes it to
    /// [`DEFAULT_UNAME_VERSION`].
    pub version: String,
    /// C's nullable `uname_hostname_override` global.
    pub hostname_override: Option<String>,
    /// The C compilation's `__DATE__` value.
    pub build_date: String,
    /// The C compilation's `__TIME__` value.
    pub build_time: String,
}

/// Upstream's initial `uname_version` string.
pub const DEFAULT_UNAME_VERSION: &str = "SUPER AWESOME";

impl UnameConfig {
    /// Make the configuration that upstream `uname.c` has at startup, with an
    /// explicit C build date/time for reproducible embeddings and tests.
    pub fn ish_default(build_date: impl Into<String>, build_time: impl Into<String>) -> Self {
        Self {
            version: DEFAULT_UNAME_VERSION.to_owned(),
            hostname_override: None,
            build_date: build_date.into(),
            build_time: build_time.into(),
        }
    }
}

/// The host observations consumed by `kernel/uname.c`.
///
/// The C source calls `uname`, platform `get_uptime`, and (on Linux) `sysinfo`.
/// Their values are supplied explicitly rather than obtained through a
/// Linux-only API inside the portable Rust core.
pub trait SystemInfoHost {
    /// Return the node-name string that host `uname(2)` would report.
    fn hostname(&self) -> &str;

    /// Return the platform uptime and three load values.
    fn uptime(&self) -> UptimeInfo;

    /// Return host memory/process values used by the Linux `sysinfo` branch.
    fn sysinfo(&self) -> HostSysInfo;
}

fn c_string_prefix(value: &str) -> &str {
    value.split_once('\0').map_or(value, |(prefix, _)| prefix)
}

fn copy_c_string(dst: &mut [u8; UNAME_LENGTH], value: &str) {
    // C begins with memset(uts, 0, sizeof *uts), then strcpy or snprintf. Its
    // valid source strings fit their fields; clamp a malformed Rust embedding
    // safely instead of reproducing strcpy's out-of-bounds write.
    let source = c_string_prefix(value).as_bytes();
    let count = source.len().min(UNAME_LENGTH - 1);
    dst[..count].copy_from_slice(&source[..count]);
}

/// `do_uname`.
///
/// As in C, the host uname lookup happens even when the override is set; an
/// observer can therefore retain the same host-call timing. Strings longer
/// than 64 bytes are safely truncated here, whereas C's `strcpy` override path
/// would overflow the fixed guest field.
pub fn do_uname(host: &impl SystemInfoHost, config: &UnameConfig) -> Uname {
    let host_hostname = host.hostname();
    let hostname = config.hostname_override.as_deref().unwrap_or(host_hostname);
    let mut uts = Uname::default();
    copy_c_string(&mut uts.system, "Linux");
    copy_c_string(&mut uts.hostname, hostname);
    copy_c_string(&mut uts.release, "4.20.69-ish");
    let version = format!(
        "{} {} {}",
        c_string_prefix(&config.version),
        c_string_prefix(&config.build_date),
        c_string_prefix(&config.build_time)
    );
    copy_c_string(&mut uts.version, &version);
    copy_c_string(&mut uts.arch, "i686");
    copy_c_string(&mut uts.domain, "(none)");
    uts
}

/// `sys_uname`.
pub fn sys_uname(
    table: &mut TaskTable,
    host: &impl SystemInfoHost,
    config: &UnameConfig,
    uts_addr: Addr,
) -> i32 {
    let uts = do_uname(host, config);
    let Some(task) = table.current_mut() else {
        return ESRCH;
    };
    task.user_write(uts_addr, &uts.to_le_bytes())
        .map_or(EFAULT, |_| 0)
}

/// `sys_sethostname` is an iSH permission-denied stub.
pub fn sys_sethostname(_hostname_addr: Addr, _hostname_len: u32) -> i32 {
    EPERM
}

/// `sys_sysinfo`.
pub fn sys_sysinfo(table: &mut TaskTable, host: &impl SystemInfoHost, info_addr: Addr) -> i32 {
    // C calls get_uptime before its Linux sysinfo-specific helper.
    let info = SysInfo::from_host(host.uptime(), host.sysinfo());
    let Some(task) = table.current_mut() else {
        return ESRCH;
    };
    task.user_write(info_addr, &info.to_le_bytes())
        .map_or(EFAULT, |_| 0)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::memory::P_RWX;
    use crate::mmu::PAGE_BITS;

    struct Host;

    impl SystemInfoHost for Host {
        fn hostname(&self) -> &str {
            "host-name"
        }

        fn uptime(&self) -> UptimeInfo {
            UptimeInfo {
                uptime_ticks: 0x1_0000_0001,
                load_1m: 2,
                load_5m: 3,
                load_15m: 4,
            }
        }

        fn sysinfo(&self) -> HostSysInfo {
            HostSysInfo {
                totalram: 0x1_0000_0005,
                freeram: 6,
                sharedram: 7,
                totalswap: 8,
                freeswap: 9,
                procs: 10,
                totalhigh: 11,
                freehigh: 12,
                mem_unit: 13,
            }
        }
    }

    #[test]
    fn uname_and_sysinfo_keep_the_c_layout_and_truncations() {
        let host = Host;
        let config = UnameConfig::ish_default("Jan  1 1970", "00:00:00");
        let uts = do_uname(&host, &config);
        assert_eq!(&uts.system[..6], b"Linux\0");
        assert_eq!(&uts.hostname[..10], b"host-name\0");
        assert_eq!(&uts.release[..12], b"4.20.69-ish\0");
        assert_eq!(&uts.version[..34], b"SUPER AWESOME Jan  1 1970 00:00:00");
        assert_eq!(Uname::SIZE, 390);

        let info = SysInfo::from_host(host.uptime(), host.sysinfo());
        assert_eq!(info.uptime, 1);
        assert_eq!(info.totalram, 5);
        let bytes = info.to_le_bytes();
        assert_eq!(bytes.len(), 60);
        assert_eq!(&bytes[40..42], &10u16.to_le_bytes());
        assert_eq!(&bytes[42..44], &[0, 0]);
        assert_eq!(&bytes[56..60], &[0, 0, 0, 0]);

        let mut table = TaskTable::bootstrap();
        table
            .current()
            .unwrap()
            .mm_mut()
            .unwrap()
            .mem
            .map_nothing(0x100, 1, P_RWX);
        let addr = 0x100 << PAGE_BITS;
        assert_eq!(sys_uname(&mut table, &host, &config, addr), 0);
        assert_eq!(sys_sysinfo(&mut table, &host, 0x9000), EFAULT);
        assert_eq!(sys_sethostname(addr, 5), EPERM);
    }
}

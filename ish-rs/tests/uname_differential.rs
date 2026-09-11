//! Differential test of `uname.rs` against unmodified `kernel/uname.c`.
//!
//! The C fixture linker-wraps only `uname(2)` and Linux `sysinfo(2)`, and
//! supplies platform `get_uptime()`. `SOURCE_DATE_EPOCH=0` makes the C build
//! macros in iSH's version string deterministic. This leaves the original C
//! field construction, host-call ordering, guest writes, and error handling in
//! the executable that produces the reference.
//!
//! Regenerate with:
//!
//! ```text
//! ISH_SRC=/path/to/ish ./tools/gen_uname_reference.sh
//! ```

use std::cell::Cell;

use ish_emu::memory::P_RWX;
use ish_emu::mmu::PAGE_BITS;
use ish_emu::task::{Addr, TaskTable};
use ish_emu::uname::{self, HostSysInfo, SystemInfoHost, UnameConfig, UptimeInfo};

const FIXTURE: &str = concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/tests/fixtures/uname_reference.txt"
);
const GUEST_BASE: Addr = 0x001000;
const GUEST_SIZE: usize = 0x002000;

struct ControlledHost {
    hostname: String,
    uptime: Cell<UptimeInfo>,
    sysinfo: Cell<HostSysInfo>,
    uname_calls: Cell<u32>,
    uptime_calls: Cell<u32>,
    sysinfo_calls: Cell<u32>,
}

impl ControlledHost {
    fn new() -> Self {
        Self {
            hostname: String::new(),
            uptime: Cell::new(UptimeInfo::default()),
            sysinfo: Cell::new(HostSysInfo::default()),
            uname_calls: Cell::new(0),
            uptime_calls: Cell::new(0),
            sysinfo_calls: Cell::new(0),
        }
    }
}

impl SystemInfoHost for ControlledHost {
    fn hostname(&self) -> &str {
        self.uname_calls.set(self.uname_calls.get() + 1);
        &self.hostname
    }

    fn uptime(&self) -> UptimeInfo {
        self.uptime_calls.set(self.uptime_calls.get() + 1);
        self.uptime.get()
    }

    fn sysinfo(&self) -> HostSysInfo {
        self.sysinfo_calls.set(self.sysinfo_calls.get() + 1);
        self.sysinfo.get()
    }
}

#[derive(Debug)]
struct Pending {
    description: String,
    result: i32,
    direct_uname: Option<Vec<u8>>,
    saw_return: bool,
}

fn hex32(text: &str) -> u32 {
    u32::from_str_radix(text, 16).unwrap_or_else(|err| panic!("invalid u32 `{text}`: {err}"))
}

fn hex64(text: &str) -> u64 {
    u64::from_str_radix(text, 16).unwrap_or_else(|err| panic!("invalid u64 `{text}`: {err}"))
}

fn decode_bytes(text: &str) -> Vec<u8> {
    assert_eq!(text.len() % 2, 0, "odd-length hex string `{text}`");
    (0..text.len())
        .step_by(2)
        .map(|index| u8::from_str_radix(&text[index..index + 2], 16).unwrap())
        .collect()
}

fn decode_text(text: &str) -> String {
    String::from_utf8(decode_bytes(text))
        .unwrap_or_else(|err| panic!("invalid fixture UTF-8: {err}"))
}

fn table_with_guest_window() -> TaskTable {
    let mut table = TaskTable::bootstrap();
    {
        let task = table.current().expect("bootstrap task");
        let mut mm = task.mm_mut().expect("bootstrap mm");
        assert_eq!(mm.mem.map_nothing(GUEST_BASE >> PAGE_BITS, 2, P_RWX), 0);
    }
    let seed: Vec<u8> = (0..GUEST_SIZE)
        .map(|index| (index as u32).wrapping_mul(13).wrapping_add(3) as u8)
        .collect();
    table
        .current_mut()
        .unwrap()
        .user_write(GUEST_BASE, &seed)
        .unwrap();
    table
}

fn guest_hash(table: &mut TaskTable) -> u64 {
    let mut bytes = vec![0; GUEST_SIZE];
    table
        .current_mut()
        .unwrap()
        .user_read(GUEST_BASE, &mut bytes)
        .expect("fixture leaves both guest pages mapped");
    bytes
        .into_iter()
        .fold(0xcbf2_9ce4_8422_2325u64, |hash, byte| {
            (hash ^ u64::from(byte)).wrapping_mul(0x0000_0100_0000_01b3)
        })
}

fn apply_prep(fields: &[&str], host: &mut ControlledHost, config: &mut UnameConfig) {
    assert!(fields.len() >= 3, "short preparation record: {fields:?}");
    match fields[1] {
        "HOSTNAME" => {
            assert_eq!(fields.len(), 3);
            host.hostname = decode_text(fields[2]);
        }
        "OVERRIDE" => {
            assert_eq!(fields.len(), 3);
            config.hostname_override = (fields[2] != "-").then(|| decode_text(fields[2]));
        }
        "VERSION" => {
            assert_eq!(fields.len(), 3);
            config.version = decode_text(fields[2]);
        }
        "UPTIME" => {
            assert_eq!(fields.len(), 6);
            host.uptime.set(UptimeInfo {
                uptime_ticks: hex64(fields[2]),
                load_1m: hex64(fields[3]),
                load_5m: hex64(fields[4]),
                load_15m: hex64(fields[5]),
            });
        }
        "SYS" => {
            assert_eq!(fields.len(), 11);
            host.sysinfo.set(HostSysInfo {
                totalram: hex64(fields[2]),
                freeram: hex64(fields[3]),
                sharedram: hex64(fields[4]),
                totalswap: hex64(fields[5]),
                freeswap: hex64(fields[6]),
                procs: hex32(fields[7]) as u16,
                totalhigh: hex64(fields[8]),
                freehigh: hex64(fields[9]),
                mem_unit: hex32(fields[10]),
            });
        }
        other => panic!("unknown uname preparation `{other}`"),
    }
}

fn run_op(
    fields: &[&str],
    table: &mut TaskTable,
    host: &ControlledHost,
    config: &UnameConfig,
) -> Pending {
    assert!(fields.len() >= 2, "short operation record: {fields:?}");
    let description = fields[1..].join(" ");
    match fields[1] {
        "DU" => {
            assert_eq!(fields.len(), 2);
            let result = uname::do_uname(host, config).to_le_bytes().to_vec();
            Pending {
                description,
                result: 0,
                direct_uname: Some(result),
                saw_return: false,
            }
        }
        "UN" => {
            assert_eq!(fields.len(), 3);
            Pending {
                description,
                result: uname::sys_uname(table, host, config, hex32(fields[2])),
                direct_uname: None,
                saw_return: false,
            }
        }
        "SH" => {
            assert_eq!(fields.len(), 4);
            Pending {
                description,
                result: uname::sys_sethostname(hex32(fields[2]), hex32(fields[3])),
                direct_uname: None,
                saw_return: false,
            }
        }
        "SI" => {
            assert_eq!(fields.len(), 3);
            Pending {
                description,
                result: uname::sys_sysinfo(table, host, hex32(fields[2])),
                direct_uname: None,
                saw_return: false,
            }
        }
        other => panic!("unknown uname operation `{other}`"),
    }
}

#[test]
fn uname_and_sysinfo_match_the_c_reference() {
    let fixture = std::fs::read_to_string(FIXTURE)
        .unwrap_or_else(|err| panic!("cannot read {FIXTURE}: {err}"));
    let mut table = table_with_guest_window();
    let mut host = ControlledHost::new();
    let mut config = UnameConfig::ish_default("Jan  1 1970", "00:00:00");
    let mut pending: Option<Pending> = None;
    let mut operations = 0usize;
    let mut snapshots = 0usize;

    for line in fixture.lines() {
        if line.is_empty() || line.starts_with('#') {
            continue;
        }
        let fields: Vec<&str> = line.split(' ').collect();
        match fields[0] {
            "P" => {
                assert!(
                    pending.is_none(),
                    "preparation while an operation is pending"
                );
                apply_prep(&fields, &mut host, &mut config);
            }
            "O" => {
                assert!(pending.is_none(), "operation before the prior snapshot");
                pending = Some(run_op(&fields, &mut table, &host, &config));
            }
            "R" => {
                assert_eq!(fields.len(), 2, "malformed return: {line}");
                let operation = pending.as_mut().expect("return without an operation");
                assert!(
                    !operation.saw_return,
                    "duplicate return for {}",
                    operation.description
                );
                assert_eq!(
                    operation.result as u32,
                    hex32(fields[1]),
                    "{}: raw i386 return",
                    operation.description
                );
                operation.saw_return = true;
            }
            "V" => {
                assert_eq!(fields.len(), 2, "malformed direct uname output: {line}");
                let operation = pending.as_mut().expect("value without an operation");
                assert!(
                    operation.saw_return,
                    "value before return for {}",
                    operation.description
                );
                let got = operation
                    .direct_uname
                    .take()
                    .expect("V record for a non-do_uname operation");
                assert_eq!(
                    got,
                    decode_bytes(fields[1]),
                    "{}: struct uname bytes",
                    operation.description
                );
            }
            "S" => {
                assert_eq!(fields.len(), 5, "malformed state: {line}");
                let context = pending.as_ref().map_or_else(
                    || "initial deterministic uname state".to_owned(),
                    |operation| operation.description.clone(),
                );
                assert_eq!(
                    host.uname_calls.get(),
                    hex32(fields[1]),
                    "{context}: host uname calls"
                );
                assert_eq!(
                    host.uptime_calls.get(),
                    hex32(fields[2]),
                    "{context}: platform uptime calls"
                );
                assert_eq!(
                    host.sysinfo_calls.get(),
                    hex32(fields[3]),
                    "{context}: host sysinfo calls"
                );
                assert_eq!(
                    guest_hash(&mut table),
                    hex64(fields[4]),
                    "{context}: guest bytes"
                );
                if let Some(operation) = pending.take() {
                    assert!(
                        operation.saw_return,
                        "missing return for {}",
                        operation.description
                    );
                    assert!(
                        operation.direct_uname.is_none(),
                        "missing direct uname bytes for {}",
                        operation.description
                    );
                    operations += 1;
                } else {
                    assert_eq!(snapshots, 0, "unexpected standalone state snapshot");
                }
                snapshots += 1;
            }
            other => panic!("unknown uname fixture record `{other}`: {line}"),
        }
    }

    assert!(pending.is_none(), "fixture ended before a state snapshot");
    assert_eq!(operations, 10);
    assert_eq!(snapshots, operations + 1);
    println!("{operations} uname/sysinfo operations and all C-derived guest bytes matched exactly");
}

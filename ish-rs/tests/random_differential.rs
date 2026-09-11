//! Differential test of `random.rs` against unmodified `kernel/random.c`.
//!
//! The C fixture links `kernel/random.c` unchanged. Its Linux
//! `syscall(SYS_getrandom, ...)` boundary is linker-wrapped with a deterministic
//! source, so the C and Rust executions can compare output bytes, source-call
//! ordering, and error paths locally without relying on host entropy.
//!
//! Regenerate with:
//!
//! ```text
//! ISH_SRC=/path/to/ish ./tools/gen_random_reference.sh
//! ```

use std::cell::Cell;

use ish_emu::memory::P_RWX;
use ish_emu::mmu::PAGE_BITS;
use ish_emu::random::{self, RandomError, RandomSource};
use ish_emu::task::{Addr, TaskTable};

const FIXTURE: &str = concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/tests/fixtures/random_reference.txt"
);
const GUEST_BASE: Addr = 0x001000;
const GUEST_SIZE: usize = 0x002000;

struct Pending {
    description: String,
    result: i32,
    saw_return: bool,
}

struct PatternSource {
    fail: Cell<bool>,
    calls: Cell<u32>,
}

impl PatternSource {
    fn new() -> Self {
        Self {
            fail: Cell::new(false),
            calls: Cell::new(0),
        }
    }
}

impl RandomSource for PatternSource {
    fn fill_random(&self, out: &mut [u8]) -> Result<(), RandomError> {
        let call = self.calls.get();
        self.calls.set(call + 1);
        if self.fail.get() {
            return Err(RandomError::Failed);
        }
        let len = out.len() as u32;
        for (index, byte) in out.iter_mut().enumerate() {
            *byte = (index as u32)
                .wrapping_mul(29)
                .wrapping_add(len)
                .wrapping_add(call.wrapping_mul(17)) as u8;
        }
        Ok(())
    }
}

fn hex32(text: &str) -> u32 {
    u32::from_str_radix(text, 16).unwrap_or_else(|err| panic!("invalid u32 `{text}`: {err}"))
}

fn hex64(text: &str) -> u64 {
    u64::from_str_radix(text, 16).unwrap_or_else(|err| panic!("invalid u64 `{text}`: {err}"))
}

fn table_with_guest_window() -> TaskTable {
    let mut table = TaskTable::bootstrap();
    {
        let task = table.current().expect("bootstrap task");
        let mut mm = task.mm_mut().expect("bootstrap mm");
        assert_eq!(mm.mem.map_nothing(GUEST_BASE >> PAGE_BITS, 2, P_RWX), 0);
    }
    let seed: Vec<u8> = (0..GUEST_SIZE)
        .map(|index| (index as u32).wrapping_mul(31).wrapping_add(7) as u8)
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

#[test]
fn getrandom_matches_the_c_reference() {
    let fixture = std::fs::read_to_string(FIXTURE)
        .unwrap_or_else(|err| panic!("cannot read {FIXTURE}: {err}"));
    let mut table = table_with_guest_window();
    let source = PatternSource::new();
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
                assert_eq!(fields.len(), 3, "malformed preparation: {line}");
                assert!(
                    pending.is_none(),
                    "preparation while an operation is pending"
                );
                assert_eq!(fields[1], "FAIL", "unknown preparation: {line}");
                source.fail.set(match fields[2] {
                    "0" => false,
                    "1" => true,
                    other => panic!("invalid failure switch `{other}`"),
                });
            }
            "O" => {
                assert_eq!(fields.len(), 5, "malformed operation: {line}");
                assert!(pending.is_none(), "operation before the prior snapshot");
                assert_eq!(fields[1], "GR", "unknown operation: {line}");
                let result = random::sys_getrandom(
                    &mut table,
                    &source,
                    hex32(fields[2]),
                    hex32(fields[3]),
                    hex32(fields[4]),
                );
                pending = Some(Pending {
                    description: fields[1..].join(" "),
                    result,
                    saw_return: false,
                });
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
            "S" => {
                assert_eq!(fields.len(), 3, "malformed state: {line}");
                let context = pending.as_ref().map_or_else(
                    || "initial deterministic random state".to_owned(),
                    |op| op.description.clone(),
                );
                assert_eq!(
                    source.calls.get(),
                    hex32(fields[1]),
                    "{context}: platform source calls"
                );
                assert_eq!(
                    guest_hash(&mut table),
                    hex64(fields[2]),
                    "{context}: guest bytes"
                );
                if let Some(operation) = pending.take() {
                    assert!(
                        operation.saw_return,
                        "missing return for {}",
                        operation.description
                    );
                    operations += 1;
                } else {
                    assert_eq!(snapshots, 0, "unexpected standalone state snapshot");
                }
                snapshots += 1;
            }
            other => panic!("unknown random fixture record `{other}`: {line}"),
        }
    }

    assert!(pending.is_none(), "fixture ended before a state snapshot");
    assert_eq!(operations, 9);
    assert_eq!(snapshots, operations + 1);
    println!("{operations} getrandom operations and every deterministic output byte matched C");
}

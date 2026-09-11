//! Differential replay of unmodified `kernel/log.c` and `util/fifo.c`.
//!
//! `tools/log-dump.c` links both original C files. It selects the dprintf log
//! handler and linker-wraps only `writev(2)`, turning complete host-visible log
//! lines into a deterministic count/hash; its guest-memory harness records C's
//! `sys_syslog` output. The large generated-line operation reaches the real
//! one-MiB FIFO wrap without committing megabytes of fixture input.
//!
//! Regenerate locally with:
//!
//! ```text
//! ISH_SRC=/path/to/ish ./tools/gen_log_reference.sh
//! ```

use ish_emu::log::{self, KernelLog, LogSink, PrintkBuffer};
use ish_emu::memory::P_RWX;
use ish_emu::mmu::PAGE_BITS;
use ish_emu::task::{Addr, TaskTable};

const FIXTURE: &str = concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/tests/fixtures/log_reference.txt"
);
const GUEST_BASE: Addr = 0x001000;
const GUEST_SIZE: usize = 0x008000;

struct HashSink {
    calls: u32,
    hash: u64,
}

impl Default for HashSink {
    fn default() -> Self {
        Self {
            calls: 0,
            hash: 0xcbf2_9ce4_8422_2325,
        }
    }
}

impl HashSink {
    fn add_bytes(&mut self, bytes: &[u8]) {
        for byte in bytes {
            self.hash ^= u64::from(*byte);
            self.hash = self.hash.wrapping_mul(0x0000_0100_0000_01b3);
        }
    }
}

impl LogSink for HashSink {
    fn line(&mut self, line: &[u8]) {
        self.calls += 1;
        self.add_bytes(line);
        self.add_bytes(b"\n");
    }
}

#[derive(Debug)]
struct Pending {
    description: String,
    result: i32,
    output_addr: Option<Addr>,
    expects_value: bool,
    saw_return: bool,
    saw_value: bool,
}

fn hex32(text: &str) -> u32 {
    u32::from_str_radix(text, 16).unwrap_or_else(|err| panic!("invalid u32 `{text}`: {err}"))
}

fn hex64(text: &str) -> u64 {
    u64::from_str_radix(text, 16).unwrap_or_else(|err| panic!("invalid u64 `{text}`: {err}"))
}

fn decode_bytes(text: &str) -> Vec<u8> {
    assert_eq!(text.len() % 2, 0, "odd-length hex record `{text}`");
    (0..text.len())
        .step_by(2)
        .map(|index| u8::from_str_radix(&text[index..index + 2], 16).unwrap())
        .collect()
}

fn table_with_guest_window() -> TaskTable {
    let mut table = TaskTable::bootstrap();
    {
        let task = table.current().expect("bootstrap task");
        let mut mm = task.mm_mut().expect("bootstrap mm");
        assert_eq!(mm.mem.map_nothing(GUEST_BASE >> PAGE_BITS, 8, P_RWX), 0);
    }
    let seed: Vec<u8> = (0..GUEST_SIZE)
        .map(|index| (index as u32).wrapping_mul(17).wrapping_add(9) as u8)
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
        .expect("fixture keeps the guest window mapped");
    bytes
        .into_iter()
        .fold(0xcbf2_9ce4_8422_2325u64, |hash, byte| {
            (hash ^ u64::from(byte)).wrapping_mul(0x0000_0100_0000_01b3)
        })
}

fn generated_line(line_len: usize, row: u32, seed: u32) -> Vec<u8> {
    assert!(line_len >= 2);
    let mut line = Vec::with_capacity(line_len);
    for index in 0..line_len - 1 {
        line.push(b'A' + ((row * 7 + index as u32 * 11 + seed) % 26) as u8);
    }
    line.push(b'\n');
    line
}

fn is_read_action(action: i32) -> bool {
    matches!(
        action,
        log::SYSLOG_ACTION_READ | log::SYSLOG_ACTION_READ_ALL | log::SYSLOG_ACTION_READ_CLEAR
    )
}

fn run_op(
    fields: &[&str],
    log_state: &mut KernelLog,
    printk_buffer: &mut PrintkBuffer,
    sink: &mut HashSink,
    table: &mut TaskTable,
) -> Pending {
    assert!(fields.len() >= 2, "short operation record: {fields:?}");
    let description = fields[1..].join(" ");
    match fields[1] {
        "T" => {
            assert_eq!(fields.len(), 3);
            printk_buffer.printk(log_state, &decode_bytes(fields[2]), sink);
            Pending {
                description,
                result: 0,
                output_addr: None,
                expects_value: false,
                saw_return: false,
                saw_value: false,
            }
        }
        "G" => {
            assert_eq!(fields.len(), 5);
            let line_len = hex32(fields[2]) as usize;
            let count = hex32(fields[3]);
            let seed = hex32(fields[4]);
            for row in 0..count {
                printk_buffer.printk(log_state, &generated_line(line_len, row, seed), sink);
            }
            Pending {
                description,
                result: 0,
                output_addr: None,
                expects_value: false,
                saw_return: false,
                saw_value: false,
            }
        }
        "SL" => {
            assert_eq!(fields.len(), 5);
            let action = hex32(fields[2]) as i32;
            let addr = hex32(fields[3]);
            let result = log::sys_syslog(log_state, table, action, addr, hex32(fields[4]) as i32);
            Pending {
                description,
                result,
                output_addr: Some(addr),
                expects_value: is_read_action(action) && result >= 0,
                saw_return: false,
                saw_value: false,
            }
        }
        other => panic!("unknown log operation `{other}`"),
    }
}

#[test]
fn log_and_syslog_match_the_c_reference() {
    let fixture = std::fs::read_to_string(FIXTURE)
        .unwrap_or_else(|err| panic!("cannot read {FIXTURE}: {err}"));
    let mut log_state = KernelLog::new();
    let mut printk_buffer = PrintkBuffer::new();
    let mut sink = HashSink::default();
    let mut table = table_with_guest_window();
    let mut pending: Option<Pending> = None;
    let mut operations = 0usize;
    let mut values = 0usize;
    let mut snapshots = 0usize;

    for line in fixture.lines() {
        if line.is_empty() || line.starts_with('#') {
            continue;
        }
        let fields: Vec<&str> = line.split(' ').collect();
        match fields[0] {
            "O" => {
                assert!(pending.is_none(), "operation before the prior snapshot");
                pending = Some(run_op(
                    &fields,
                    &mut log_state,
                    &mut printk_buffer,
                    &mut sink,
                    &mut table,
                ));
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
                assert_eq!(fields.len(), 2, "malformed guest bytes: {line}");
                let operation = pending.as_mut().expect("bytes without an operation");
                assert!(
                    operation.saw_return,
                    "bytes before return for {}",
                    operation.description
                );
                assert!(
                    operation.expects_value,
                    "unexpected bytes for {}",
                    operation.description
                );
                assert!(
                    !operation.saw_value,
                    "duplicate bytes for {}",
                    operation.description
                );
                let expected = decode_bytes(fields[1]);
                let addr = operation
                    .output_addr
                    .expect("bytes without a guest address");
                let mut got = vec![0; expected.len()];
                table
                    .current_mut()
                    .unwrap()
                    .user_read(addr, &mut got)
                    .expect("C only emits V after a successful mapped write");
                assert_eq!(got, expected, "{}: C guest output", operation.description);
                operation.saw_value = true;
                values += 1;
            }
            "S" => {
                assert_eq!(fields.len(), 5, "malformed state: {line}");
                let context = pending.as_ref().map_or_else(
                    || "initial deterministic log state".to_owned(),
                    |operation| operation.description.clone(),
                );
                assert_eq!(
                    sink.calls,
                    hex32(fields[1]),
                    "{context}: emitted line count"
                );
                assert_eq!(sink.hash, hex64(fields[2]), "{context}: emitted line bytes");
                assert_eq!(
                    log_state.unread_len(),
                    hex32(fields[3]) as usize,
                    "{context}: unread FIFO size"
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
                    assert_eq!(
                        operation.saw_value, operation.expects_value,
                        "missing or unexpected guest bytes for {}",
                        operation.description
                    );
                    operations += 1;
                } else {
                    assert_eq!(snapshots, 0, "unexpected standalone state snapshot");
                }
                snapshots += 1;
            }
            other => panic!("unknown log fixture record `{other}`: {line}"),
        }
    }

    assert!(pending.is_none(), "fixture ended before a state snapshot");
    assert_eq!(operations, 31);
    assert_eq!(values, 9);
    assert_eq!(snapshots, operations + 1);
    println!("{operations} log/syslog operations and all C-derived output bytes matched exactly");
}

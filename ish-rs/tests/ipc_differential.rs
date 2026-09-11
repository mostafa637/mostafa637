//! Differential replay of unmodified `kernel/ipc.c`.
//!
//! The source has no host boundary to replace: its sole guest-visible behavior
//! is to return `_ENOSYS` after accepting the six raw i386 argument words.
//! The committed fixture is generated locally with:
//!
//! ```text
//! ISH_SRC=/path/to/ish ./tools/gen_ipc_reference.sh
//! ```

use ish_emu::ipc;

const FIXTURE: &str = concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/tests/fixtures/ipc_reference.txt"
);

fn hex32(text: &str) -> u32 {
    u32::from_str_radix(text, 16).unwrap_or_else(|err| panic!("invalid u32 `{text}`: {err}"))
}

#[test]
fn ipc_stub_matches_the_c_reference_for_all_argument_bits() {
    let fixture = std::fs::read_to_string(FIXTURE)
        .unwrap_or_else(|err| panic!("cannot read {FIXTURE}: {err}"));
    let mut pending: Option<(u32, i32, i32, i32, u32, i32)> = None;
    let mut operations = 0usize;

    for line in fixture.lines() {
        if line.is_empty() || line.starts_with('#') {
            continue;
        }
        let fields: Vec<&str> = line.split(' ').collect();
        match fields[0] {
            "O" => {
                assert!(pending.is_none(), "operation before the preceding return");
                assert_eq!(fields.len(), 8, "malformed C operation: {line}");
                assert_eq!(fields[1], "IPC", "unexpected C operation: {line}");
                pending = Some((
                    hex32(fields[2]),
                    hex32(fields[3]) as i32,
                    hex32(fields[4]) as i32,
                    hex32(fields[5]) as i32,
                    hex32(fields[6]),
                    hex32(fields[7]) as i32,
                ));
            }
            "R" => {
                assert_eq!(fields.len(), 2, "malformed C return: {line}");
                let (call, first, second, third, ptr, fifth) =
                    pending.take().expect("return without a C operation");
                assert_eq!(
                    ipc::sys_ipc(call, first, second, third, ptr, fifth) as u32,
                    hex32(fields[1]),
                    "IPC({call:#x}, {first:#x}, {second:#x}, {third:#x}, {ptr:#x}, {fifth:#x})"
                );
                operations += 1;
            }
            other => panic!("unknown ipc fixture record `{other}`: {line}"),
        }
    }

    assert!(pending.is_none(), "fixture ended without a return");
    assert_eq!(operations, 5);
    println!("{operations} raw ipc calls matched unmodified C exactly");
}

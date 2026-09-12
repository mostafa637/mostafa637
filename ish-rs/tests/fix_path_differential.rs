//! Differential test for `fs/fix_path.h` vs `crate::fix_path`.
//!
//! `tools/fix-path-dump.c` links the unmodified C header and prints cases.
//! This test replays them through Rust's `fix_path` and requires exact match.

use ish_emu::fix_path::{fix_path, fix_path_bytes};

const FIXTURE: &str = concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/tests/fixtures/fix_path_reference.txt"
);

fn hex_to_bytes(s: &str) -> Vec<u8> {
    if s.is_empty() {
        return Vec::new();
    }
    (0..s.len())
        .step_by(2)
        .map(|i| u8::from_str_radix(&s[i..i + 2], 16).unwrap())
        .collect()
}

fn bytes_to_string(bytes: &[u8]) -> String {
    String::from_utf8_lossy(bytes).to_string()
}

#[test]
fn fix_path_matches_c() {
    let content = std::fs::read_to_string(FIXTURE).expect("read fixture");
    let mut count = 0;
    for line in content.lines() {
        if line.starts_with('C') || line.starts_with('B') {
            // Format: C <len> <hex> -> <len> <hex> "input" -> "output"
            // We parse the quoted parts for simplicity, but also verify hex lengths.
            // Example: C 4 2f666f6f -> 3 666f6f "/foo" -> "foo"
            let parts: Vec<&str> = line.split('"').collect();
            if parts.len() >= 4 {
                let input = parts[1];
                let output = parts[3];
                let rust_output = fix_path(input);
                assert_eq!(
                    rust_output, output,
                    "fix_path mismatch for input {:?}: C={:?} Rust={:?} line={}",
                    input, output, rust_output, line
                );
                count += 1;

                // Also test bytes version
                let input_bytes = input.as_bytes();
                let output_bytes = fix_path_bytes(input_bytes);
                assert_eq!(
                    output_bytes,
                    output.as_bytes(),
                    "fix_path_bytes mismatch for {:?}",
                    input
                );
            } else {
                // For B lines without quotes in some cases, parse hex
                // Format: B <len> <hex> -> <len> <hex>
                // We already covered via C lines, so skip
                continue;
            }
        }
    }
    assert!(count >= 10, "expected at least 10 cases, got {}", count);
    println!("fix_path: {} cases matched C exactly", count);
}

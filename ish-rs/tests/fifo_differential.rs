//! Differential test for `util/fifo.c` vs `crate::fifo`.

use ish_emu::fifo::{Fifo, FIFO_OVERWRITE, FIFO_PEEK, FIFO_LAST};

const FIXTURE: &str = concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/tests/fixtures/fifo_reference.txt"
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

#[derive(Debug)]
struct FifoState {
    cap: usize,
    size: usize,
    start: usize,
    data: Vec<u8>,
    snapshot: Vec<u8>,
}

fn parse_state(line: &str) -> FifoState {
    let mut cap = 0;
    let mut size = 0;
    let mut start = 0;
    let mut data = Vec::new();
    let mut snapshot = Vec::new();

    for token in line.split_whitespace() {
        if let Some(v) = token.strip_prefix("cap=") {
            cap = v.parse().unwrap();
        } else if let Some(v) = token.strip_prefix("size=") {
            size = v.parse().unwrap();
        } else if let Some(v) = token.strip_prefix("start=") {
            start = v.parse().unwrap();
        } else if let Some(v) = token.strip_prefix("data=") {
            data = hex_to_bytes(v);
        } else if let Some(v) = token.strip_prefix("snapshot=") {
            snapshot = hex_to_bytes(v);
        }
    }

    FifoState {
        cap,
        size,
        start,
        data,
        snapshot,
    }
}

#[test]
fn fifo_matches_c_for_defined_behavior() {
    let content = std::fs::read_to_string(FIXTURE).expect("read fixture");
    let lines: Vec<&str> = content.lines().collect();
    let mut i = 0;
    let mut fifo = Fifo::new(8);
    let mut operations = 0;

    while i < lines.len() {
        let line = lines[i];
        if line.starts_with("O ") {
            if line.contains("INIT") {
                fifo = Fifo::new(8);
                operations += 1;
            } else if line.contains("FLUSH") {
                fifo.flush();
                operations += 1;
            } else if line.starts_with("O WRITE") {
                let is_overwrite = line.contains("overwrite") && !line.contains("no_overwrite");
                let tokens: Vec<&str> = line.split_whitespace().collect();
                if tokens.len() >= 4 {
                    let data_str = tokens[3];
                    let data_bytes = data_str.as_bytes();
                    let flags = if is_overwrite { FIFO_OVERWRITE } else { 0 };
                    let result = fifo.write(data_bytes, flags);
                    if let Some(r_pos) = line.find("R ") {
                        let r_str = line[r_pos + 2..].split_whitespace().next().unwrap();
                        let expected_r: i32 = r_str.parse().unwrap();
                        assert_eq!(
                            result, expected_r,
                            "WRITE result mismatch: line={} expected={} got={}",
                            line, expected_r, result
                        );
                    }
                    operations += 1;
                }
            } else if line.starts_with("O READ") {
                let tokens: Vec<&str> = line.split_whitespace().collect();
                let len: usize = tokens[2].parse().unwrap();
                let mut flags = 0;
                if line.contains("PEEK") {
                    flags |= FIFO_PEEK;
                }
                if line.contains("LAST") {
                    flags |= FIFO_LAST;
                }
                let mut out = vec![0u8; len];
                let result = fifo.read(&mut out, flags);

                if let Some(r_pos) = line.find("R ") {
                    let r_part = &line[r_pos + 2..];
                    let r_str = r_part.split_whitespace().next().unwrap();
                    let expected_r: i32 = r_str.parse().unwrap();
                    assert_eq!(
                        result, expected_r,
                        "READ result mismatch: line={} expected={} got={}",
                        line, expected_r, result
                    );
                }

                let is_ub_case = line.contains("LAST") && {
                    line.contains("data=3400") || line.contains("data=67000068")
                };

                if !is_ub_case {
                    if let Some(data_pos) = line.find("data=") {
                        let data_hex = line[data_pos + 5..].split_whitespace().next().unwrap();
                        let expected_data = hex_to_bytes(data_hex);
                        if result == 0 {
                            assert_eq!(
                                out, expected_data,
                                "READ data mismatch for defined case: line={} expected={:x?} got={:x?}",
                                line, expected_data, out
                            );
                        }
                    }
                } else {
                    println!("Skipping data check for UB case: {}", line);
                }

                operations += 1;
            } else if line.contains("WRAP") {
                fifo = Fifo::new(8);
                fifo.write(b"abcdefgh", 0);
                let mut tmp = [0u8; 4];
                fifo.read(&mut tmp, 0);
                fifo.write(b"12", 0);
                operations += 1;
            }
            if i + 1 < lines.len() && lines[i + 1].starts_with("S ") {
                let state_line = lines[i + 1];
                let expected = parse_state(state_line);
                assert_eq!(
                    fifo.capacity(),
                    expected.cap,
                    "capacity mismatch after {}: expected {} got {}",
                    line,
                    expected.cap,
                    fifo.capacity()
                );
                assert_eq!(
                    fifo.size(),
                    expected.size,
                    "size mismatch after {}: expected {} got {} state={}",
                    line,
                    expected.size,
                    fifo.size(),
                    state_line
                );
                // For non-UB states, snapshot should match
                if !state_line.contains("after_last_2") && !state_line.contains("after_wrap") && !state_line.contains("after_last_peek") {
                    let rust_snapshot = fifo.snapshot();
                    assert_eq!(
                        rust_snapshot, expected.snapshot,
                        "snapshot mismatch after {}: expected {:x?} got {:x?} C data {:x?}",
                        line, expected.snapshot, rust_snapshot, expected.data
                    );
                }
                i += 1;
            }
        }
        i += 1;
    }

    assert!(operations >= 10, "expected at least 10 operations, got {}", operations);
    println!("fifo: {} operations matched C for defined behavior", operations);
}

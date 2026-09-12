//! Differential test for path_is_normalized and path_next_component
//! vs C implementation in `fs/path.c` (leaf helpers).

use ish_emu::path::{path_is_normalized, path_next_component};

const FIXTURE: &str = concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/tests/fixtures/path_reference.txt"
);

#[test]
fn path_is_normalized_matches_c() {
    let content = std::fs::read_to_string(FIXTURE).expect("read fixture");
    let mut norm_cases = 0;
    for line in content.lines() {
        if line.starts_with("NORM") {
            // NORM "<path>" -> <0|1>
            // Parse: NORM "<path>" -> <result>
            if let Some(start) = line.find('"') {
                if let Some(end) = line[start + 1..].find('"') {
                    let path = &line[start + 1..start + 1 + end];
                    let result_part = &line[start + 1 + end..];
                    let result: i32 = result_part
                        .split("->")
                        .nth(1)
                        .unwrap()
                        .trim()
                        .parse()
                        .unwrap();
                    let rust_result = if path_is_normalized(path) { 1 } else { 0 };
                    assert_eq!(
                        rust_result, result,
                        "path_is_normalized mismatch for {:?}: C={} Rust={}",
                        path, result, rust_result
                    );
                    norm_cases += 1;
                }
            }
        }
    }
    assert!(norm_cases >= 10, "expected at least 10 norm cases");
    println!("path_is_normalized: {} cases matched C", norm_cases);
}

#[test]
fn path_next_component_matches_c() {
    let content = std::fs::read_to_string(FIXTURE).expect("read fixture");
    let lines: Vec<&str> = content.lines().collect();
    let mut i = 0;
    let mut iter_cases = 0;

    while i < lines.len() {
        let line = lines[i];
        if line.starts_with("ITER") {
            // ITER "<path>":
            let start = line.find('"').unwrap();
            let end = line[start + 1..].find('"').unwrap();
            let path_str = &line[start + 1..start + 1 + end];
            let mut rust_path = path_str.to_string();
            let mut c_components: Vec<String> = Vec::new();
            let mut c_remaining = String::new();

            // Collect C components until END or ERR
            i += 1;
            while i < lines.len() {
                let l = lines[i];
                if l.trim().starts_with("COMP") {
                    //   COMP "<comp>" remaining="<rem>"
                    let first_quote = l.find('"').unwrap();
                    let second_quote = l[first_quote + 1..].find('"').unwrap();
                    let comp = &l[first_quote + 1..first_quote + 1 + second_quote];
                    c_components.push(comp.to_string());
                } else if l.trim().starts_with("END") {
                    //   END remaining="<rem>"
                    if let Some(s) = l.find("remaining=\"") {
                        let rest = &l[s + 11..];
                        if let Some(e) = rest.find('"') {
                            c_remaining = rest[..e].to_string();
                        }
                    }
                    break;
                } else if l.trim().starts_with("ERR") {
                    // Error case
                    break;
                }
                i += 1;
            }

            // Now replay with Rust
            let mut rust_components: Vec<String> = Vec::new();
            let mut rust_path_mut: &str = &rust_path;
            // We need to handle path_next_component which takes &mut &str
            let mut path_ref: &str = path_str;
            loop {
                match path_next_component(&mut path_ref) {
                    Ok(Some(comp)) => rust_components.push(comp),
                    Ok(None) => break,
                    Err(_) => break,
                }
            }

            assert_eq!(
                rust_components, c_components,
                "path_next_component components mismatch for {:?}: C={:?} Rust={:?}",
                path_str, c_components, rust_components
            );
            assert_eq!(
                path_ref, c_remaining,
                "path_next_component remaining mismatch for {:?}: C={:?} Rust={:?}",
                path_str, c_remaining, path_ref
            );

            iter_cases += 1;
        }
        i += 1;
    }

    assert!(iter_cases >= 4, "expected at least 4 iter cases");
    println!("path_next_component: {} cases matched C", iter_cases);
}

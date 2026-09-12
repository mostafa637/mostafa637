fn main() {
    // Online only - build Slint UI as iOS does for terminal
    // This compiles ui/appwindow.slint which is pure Rust SlintUi + Servo
    slint_build::compile("ui/appwindow.slint").expect("Slint build failed - requires online crates.io (no offline)");
}

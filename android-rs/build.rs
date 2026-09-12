fn main() {
    // Online only - build Slint UI as iOS does for terminal
    // This compiles ui/appwindow.slint which is pure Rust SlintUi + Servo
    // Only compile when slint-ui feature is enabled (fast-test skips heavy compile)
    #[cfg(feature = "slint-ui")]
    {
        slint_build::compile("ui/appwindow.slint").expect("Slint build failed - requires online crates.io (no offline)");
    }
    #[cfg(not(feature = "slint-ui"))]
    {
        println!("cargo:warning=Slint UI feature disabled (fast-test) - skipping Slint compilation ONLINE ONLY");
    }
}

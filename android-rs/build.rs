fn main() {
    #[cfg(feature = "desktop")]
    {
        slint_build::compile("ui/appwindow.slint").expect("Slint build failed");
    }
    #[cfg(not(feature = "desktop"))]
    {
        println!("cargo:rerun-if-changed=ui/appwindow.slint");
        println!("Skipping slint_build (requires --features desktop)");
    }
}

# AVD research — 2026-08-15

## Sources and findings

1. Android Developers, [Configure hardware acceleration for the Android Emulator](https://developer.android.com/studio/run/emulator-acceleration): VM acceleration requires a host/system-image architecture match. The documented table says x86_64 hosts use x86/x86_64 images and ARM64 hosts use arm64-v8a images; ARM images on Intel/AMD cannot use the documented VM acceleration. The same page documents `-gpu software` and `-gpu swiftshader` for software graphics, and marks `swiftshader_indirect` deprecated in newer emulator releases.

2. ReactiveCircus issue [Support for M1 Silicon — HVF error: HV_UNSUPPORTED](https://github.com/ReactiveCircus/android-emulator-runner/issues/350): an arm64-v8a AVD on GitHub macOS can fail with `HVF error: HV_UNSUPPORTED` and `qemu-system-aarch64-headless: failed to initialize HVF: Invalid argument`, matching this project's AVD log.

3. GitHub Docs, [Larger runners reference](https://docs.github.com/en/actions/reference/runners/larger-runners): macOS larger runners are available as Intel x64 (`macos-latest-large`) and Apple Silicon arm64 (`macos-latest-xlarge`); macOS larger runners do not support nested virtualization.

## Project implication

The current workflow used `runs-on: macos-latest` with an `arm64-v8a` image. The log showed `qemu-system-aarch64-headless` and `HV_UNSUPPORTED`. To preserve the requested macOS AVD smoke test on the likely x64 standard runner, the next workflow revision should detect `uname -m`, select `x86_64`/`google_apis;x86_64` on x64 and `arm64-v8a` on arm64, create a matching AVD, and download the corresponding APK artifact. Use the non-deprecated `-gpu swiftshader` where supported, with a fallback only if necessary.

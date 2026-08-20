# iSH iOS reference build notes

Reference repository: https://github.com/ish-app/ish

The upstream tree contains the original iSH runtime sources under `kernel/`, `fs/`, `emu/`, `asbestos/`, `util/`, `vdso/`, and platform glue under `app/`. The original UIKit terminal implementation is in `app/TerminalViewController.m`, `app/TerminalView.m`, `app/Terminal.m`, and `app/terminal/term.js`, `term.html`, and `term.css`.

The upstream app includes Linux-oriented C glue such as `app/LinuxInterop.c`, `app/LinuxPTY.c`, `app/LinuxRoot.c`, `app/LinuxTTY.c`, and `app/PasteboardDeviceLinux.c`. The original build is Xcode/xcconfig based (`app/xcode-meson.sh`, `app/xcode-ninja.sh`, `app/*.xcconfig`), not a ready-made Qt/CMake build.

The Qt port therefore needs a deliberate platform-neutral bridge around the upstream C runtime: a Qt `IshSession` object for PTY/input/output, rootfs and preferences models, and WebChannel/WebSocket transport for the preserved terminal web UI. GitHub Actions must run `ci/check-source-completeness.sh` before CMake so missing bridge files fail explicitly rather than generating a misleading partial APK.

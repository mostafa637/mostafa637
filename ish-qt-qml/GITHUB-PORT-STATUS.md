# iSH Qt/QML GitHub port status

## Scope

This directory contains the Qt 6.11.1/QML presentation layer for the iSH iOS port. The intended targets are Android (`arm64-v8a` and `x86_64`) and Linux Desktop. The port keeps iSH core, PTY/session behavior, UTF-8 terminal I/O, and the original `term.js` flow separate from the QML presentation layer.

## Available build outputs

The workspace contains previously built outputs under `dist-final/qt6111-iosstyle/`:

- `ish-qt-arm64-v8a-iosstyle-release.apk`
- `ish-qt-x86_64-iosstyle-release.apk`
- `linux/ish_qt_linux-x86_64`

The APKs are signed with the development/debug keystore used for testing, not a production release certificate.

## Current source snapshot limitation

The current restored source snapshot contains the Qt/QML layer and CMake integration, but it does not contain the C++ bridge files referenced by `android-qt/CMakeLists.txt` (for example `IshSession.cpp`, `RootfsManager.cpp`, `main.cpp`, and the WebChannel bridge). The original iSH upstream repository is used as the reference for core behavior; it is not silently copied into this presentation-layer snapshot.

Before rebuilding from a clean clone, restore the complete C++ bridge and iSH core sources, then configure Qt 6.11.1 for the selected target.

## Reproducibility

Git history and the project ZIP archive are maintained locally. Each source update should produce a new ZIP named from the Git commit and should include a SHA-256 checksum.

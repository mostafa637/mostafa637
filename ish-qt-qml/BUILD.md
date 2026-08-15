# Building the iSH Qt/QML port

## Source layout

`android-qt/` contains the Qt 6.11.1/QML presentation layer and its CMake entry point. `upstream/ish-ios/` contains the complete upstream iSH iOS source used as the behavioral and core reference. The local QML layer is intended to replace UIKit while preserving the iSH core, PTY/session flow, UTF-8 terminal I/O, and original terminal behavior.

## Linux Desktop

Configure with Qt 6.11.1 desktop and the host C/C++ toolchain, then build `android-qt/CMakeLists.txt` with `CMAKE_BUILD_TYPE=Release`. The complete Qt bridge sources must be present under `android-qt/src/` and the headless iSH libraries must be available at the paths configured by CMake.

## Android

Configure separately for `android_arm64_v8a` and `android_x86_64` using Qt 6.11.1, Android SDK platform 36, build-tools 36.0.0, JDK 17, and the Android NDK configured for the project. Build Release APKs through the Qt Android deployment target. Sign final APKs only with a release keystore appropriate for distribution.

## Important source note

This GitHub snapshot includes the full upstream iSH reference tree and the Qt/QML layer. The restored workspace used to create it does not currently contain every custom C++ bridge file referenced by `android-qt/CMakeLists.txt`; these bridge files must be restored or regenerated before a clean clone can be rebuilt from source.

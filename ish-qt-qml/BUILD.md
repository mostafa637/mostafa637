# Building the iSH Qt/QML port

## Source layout

`android-qt/` contains the Qt 6.11.1/QML presentation layer and its CMake entry point. `upstream/ish-ios/` contains the complete upstream iSH iOS source used as the behavioral and core reference. The local QML layer is intended to replace UIKit while preserving the iSH core, PTY/session flow, UTF-8 terminal I/O, and original terminal behavior.

## Linux Desktop

Configure with Qt 6.11.1 desktop and the host C/C++ toolchain, then build `android-qt/CMakeLists.txt` with `CMAKE_BUILD_TYPE=Release`. The complete Qt bridge sources must be present under `android-qt/src/` and the headless iSH libraries must be available at the paths configured by CMake.

## Android

Configure separately for `android_arm64_v8a` and `android_x86_64` using Qt 6.11.1, Android SDK platform 36, build-tools 36.0.0, JDK 17, and the Android NDK configured for the project. Build Release APKs through the Qt Android deployment target. Sign final APKs only with a release keystore appropriate for distribution.

## GitHub Actions and aqtinstall

The workflow at `.github/workflows/build-qt.yml` installs Qt 6.11.1 with `aqtinstall`. It runs `qmllint`, then prepares Linux Desktop and Android jobs for `arm64-v8a` and `x86_64`. The workflow installs Android SDK platform 36, build-tools 36.0.0, CMake 3.22.1, NDK 27.2.12479018, and JDK 17.

The workflow intentionally runs `ci/check-source-completeness.sh` before CMake. This makes a missing bridge fail with a useful list of files instead of producing a misleading partial APK.

## Important source note

This GitHub snapshot includes the full upstream iSH reference tree and the Qt/QML layer. The restored workspace used to create it does not currently contain every custom C++ bridge file referenced by `android-qt/CMakeLists.txt`; these bridge files must be restored or regenerated before a clean clone can be rebuilt from source. The GitHub workflow is ready for those files and will stop clearly until they are present.

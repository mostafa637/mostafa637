#!/usr/bin/env bash
set -euo pipefail

platform=${1:-unknown}
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cmake_file="$root/android-qt/CMakeLists.txt"

required=(
  "$root/android-qt/src/main.cpp"
  "$root/android-qt/src/IshSession.cpp"
  "$root/android-qt/src/IshSession.h"
  "$root/android-qt/src/CoreSession.cpp"
  "$root/android-qt/src/CoreSession.h"
  "$root/android-qt/tests/CoreHostSmoke.cpp"
  "$root/android-qt/src/RootfsManager.cpp"
  "$root/android-qt/src/RootfsManager.h"
  "$root/android-qt/src/ThemeManager.cpp"
  "$root/android-qt/src/ThemeManager.h"
  "$root/android-qt/src/UserPreferencesQt.cpp"
  "$root/android-qt/src/UserPreferencesQt.h"
  "$root/android-qt/src/WebSocketTransport.cpp"
  "$root/android-qt/src/WebSocketTransport.h"
  "$root/android-qt/src/WebChannelServer.cpp"
  "$root/android-qt/src/WebChannelServer.h"
  "$root/android-qt/src/PlatformServicesQt.cpp"
  "$root/android-qt/src/PlatformServicesQt.h"
  "$root/android-qt/src/FontCatalog.cpp"
  "$root/android-qt/src/FontCatalog.h"
  "$root/android-qt/src/RootModel.cpp"
  "$root/android-qt/src/RootModel.h"
  "$root/android-qt/src/RootFilesModel.cpp"
  "$root/android-qt/src/RootFilesModel.h"
  "$root/android-qt/src/RootfsUpgradeController.cpp"
  "$root/android-qt/src/RootfsUpgradeController.h"
  "$root/android-qt/assets/repositories.txt"
  "$root/android-qt/android/AndroidManifest.xml"
  "$root/android-qt/android/res/xml/network_security_config.xml"
  "$root/ci/rasterize-icons.py"
  "$root/upstream/ish-ios/tools/fakefs.c"
  "$root/upstream/ish-ios/tools/fakefs.h"
  "$root/upstream/ish-ios/app/core/CoreSession.c"
  "$root/upstream/ish-ios/app/core/CoreSession.h"
  "$root/upstream/ish-ios/app/core/CoreClipboard.c"
  "$root/upstream/ish-ios/app/core/CoreLocation.c"
  "$root/upstream/ish-ios/deps/sqlite/sqlite3.c"
  "$root/android-qt/assets/ui/icons/arrow-down-dark.png"
  "$root/android-qt/assets/ui/icons/arrow-down-light.png"
  "$root/android-qt/assets/ui/icons/arrow-left-dark.png"
  "$root/android-qt/assets/ui/icons/arrow-left-light.png"
  "$root/android-qt/assets/ui/icons/arrow-right-dark.png"
  "$root/android-qt/assets/ui/icons/arrow-right-light.png"
  "$root/android-qt/assets/ui/icons/arrow-up-dark.png"
  "$root/android-qt/assets/ui/icons/arrow-up-light.png"
  "$root/android-qt/assets/ui/icons/checkbox-dark.png"
  "$root/android-qt/assets/ui/icons/checkbox-light.png"
  "$root/android-qt/assets/ui/icons/hide-keyboard-dark.png"
  "$root/android-qt/assets/ui/icons/hide-keyboard-light.png"
  "$root/android-qt/assets/ui/icons/paste-dark.png"
  "$root/android-qt/assets/ui/icons/paste-light.png"
  "$root/android-qt/assets/ui/icons/xmark-dark.png"
  "$root/android-qt/assets/ui/icons/xmark-light.png"
)

missing=()
for path in "${required[@]}"; do
  [[ -f "$path" ]] || missing+=("${path#"$root"/}")
done

if ((${#missing[@]})); then
  printf 'ERROR: incomplete Qt/core source for %s build. Missing files:\n' "$platform" >&2
  printf '  - %s\n' "${missing[@]}" >&2
  printf '\nThe full upstream iSH reference is present under upstream/ish-ios, but the custom Qt bridge must be restored before this workflow can build.\n' >&2
  exit 2
fi

if [[ ! -f "$cmake_file" ]]; then
  echo "ERROR: missing android-qt/CMakeLists.txt" >&2
  exit 2
fi

echo "Complete Qt/core source detected for $platform."

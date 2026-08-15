#!/usr/bin/env bash
set -euo pipefail

apk_root="apk/ish-qt-android-x86_64"
apk="$(find "$apk_root" -type f -name '*.apk' ! -name '*unsigned.apk' | head -n 1 || true)"
if [[ -z "$apk" ]]; then
  echo "No signed APK found under $apk_root" >&2
  find apk -maxdepth 4 -type f -print >&2 || true
  exit 1
fi

sdk_root="${ANDROID_HOME:-${ANDROID_SDK_ROOT:?ANDROID_SDK_ROOT is not set}}"
"$sdk_root/build-tools/36.0.0/apksigner" verify "$apk"
echo "Installing signed APK: $apk"
adb install -r "$apk"
adb shell monkey -p com.mostafa637.ishqt 1
sleep 15
adb shell dumpsys activity activities | grep -F com.mostafa637.ishqt
adb exec-out screencap -p > avd-linux-screenshot.png
adb logcat -d > avd-linux-logcat.txt

if grep -E "FATAL EXCEPTION|UnsatisfiedLinkError|QQmlApplicationEngine failed|module .* is not installed" avd-linux-logcat.txt; then
  echo "Application startup errors detected" >&2
  exit 1
fi

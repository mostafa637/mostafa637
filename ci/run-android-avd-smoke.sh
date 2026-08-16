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

# Pixel Launcher can show an ANR dialog while the Qt app is already visible.
# Dismiss that system dialog in the cloud AVD before collecting app diagnostics.
for _ in {1..8}; do
  ui_xml="$(adb shell uiautomator dump /sdcard/window.xml >/dev/null 2>&1 || true; adb shell cat /sdcard/window.xml 2>/dev/null || true)"
  wait_bounds="$(printf '%s' "$ui_xml" | sed -n 's/.*text="Wait"[^>]*bounds="\[\([0-9]*\),\([0-9]*\)\]\[\([0-9]*\),\([0-9]*\)\]".*/\1 \2 \3 \4/p' | head -n 1)"
  if [[ -n "$wait_bounds" ]]; then
    read -r x1 y1 x2 y2 <<< "$wait_bounds"
    adb shell input tap "$(( (x1 + x2) / 2 ))" "$(( (y1 + y2) / 2 ))"
    sleep 3
  else
    sleep 3
  fi
done

adb shell dumpsys activity activities | grep -F com.mostafa637.ishqt
adb exec-out screencap -p > avd-linux-screenshot.png
adb logcat -d > avd-linux-logcat.txt
adb exec-out cat /sdcard/Android/data/com.mostafa637.ishqt/files/ish-qt-errors.log > ish-qt-errors-device.log 2>/dev/null || true

if grep -E "FATAL EXCEPTION|UnsatisfiedLinkError|QQmlApplicationEngine failed|module .* is not installed" avd-linux-logcat.txt; then
  echo "Application startup errors detected" >&2
  exit 1
fi

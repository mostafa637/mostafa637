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

mkdir -p avd-linux-captures avd-linux-ui avd-linux-logcat

capture_state() {
  local label="$1"
  adb exec-out screencap -p > "avd-linux-captures/${label}.png" 2>/dev/null || true
  adb shell uiautomator dump /sdcard/window.xml >/dev/null 2>&1 || true
  adb shell cat /sdcard/window.xml > "avd-linux-ui/${label}.xml" 2>/dev/null || true
  adb logcat -d -v threadtime > "avd-linux-logcat/${label}.txt" 2>/dev/null || true
  printf 'Captured %s\n' "$label"
}

# Capture immediately and once per second so text printed by QML/Android just
# before a delayed crash remains visible instead of being overwritten by an
# ANR dialog or a final black frame.
capture_state "t00-start"
(
  for second in {1..15}; do
    sleep 1
    capture_state "t$(printf '%02d' "$second")"
  done
) &
capture_pid=$!

# Pixel Launcher can show an ANR dialog while the Qt app is already visible.
# Dismiss the dialog as soon as it appears, while the capture loop keeps the
# original screen and the text printed by the app at each second.
for _ in {1..15}; do
  ui_xml="$(adb shell uiautomator dump /sdcard/window.xml >/dev/null 2>&1 || true; adb shell cat /sdcard/window.xml 2>/dev/null || true)"
  wait_bounds="$(printf '%s' "$ui_xml" | sed -n 's/.*text="Wait"[^>]*bounds="\[\([0-9]*\),\([0-9]*\)\]\[\([0-9]*\),\([0-9]*\)\]".*/\1 \2 \3 \4/p' | head -n 1)"
  if [[ -n "$wait_bounds" ]]; then
    read -r x1 y1 x2 y2 <<< "$wait_bounds"
    adb shell input tap "$(( (x1 + x2) / 2 ))" "$(( (y1 + y2) / 2 ))" || true
  fi
  sleep 1
done
wait "$capture_pid" || true

adb shell dumpsys activity activities > avd-linux-activity.txt 2>&1 || true
adb exec-out cat /sdcard/Android/data/com.mostafa637.ishqt/files/ish-qt-errors.log > ish-qt-errors-device.log 2>/dev/null || true

latest="$(find avd-linux-captures -maxdepth 1 -type f -name '*.png' | sort | tail -n 1 || true)"
if [[ -n "$latest" && -f "$latest" ]]; then
  cp "$latest" avd-linux-screenshot.png
fi
adb logcat -d -v threadtime > avd-linux-logcat.txt

if grep -E "FATAL EXCEPTION|UnsatisfiedLinkError|QQmlApplicationEngine failed|module .* is not installed|Fatal signal|Fatal signal [0-9]+" avd-linux-logcat.txt; then
  echo "Application startup/runtime errors detected; see timed screenshots and logs." >&2
  exit 1
fi

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

# Give Qt/WebView and the native session time to become interactive, then send
# the requested command through the focused terminal surface. Android's input
# utility encodes spaces as %s.
sleep 10
adb shell input tap 420 520 || true
sleep 1
adb shell input text 'apk%ssadd%spython' || true
adb shell input keyevent KEYCODE_ENTER || true
printf 'Sent command: apk add python\n'

mkdir -p avd-linux-captures avd-linux-ui

capture_state() {
  local label="$1"
  # Keep uiautomator out of this background capture function. A second
  # uiautomator instance cannot register the same automation service and would
  # create a misleading AndroidRuntime FATAL EXCEPTION in logcat.
  adb exec-out screencap -p > "avd-linux-captures/${label}.png" 2>/dev/null || true
  printf 'Captured %s\n' "$label"
}

dump_window_state() {
  local label="$1"
  # Use dumpsys instead of uiautomator. It records the focused window without
  # registering a UiAutomationService on every second and avoids polluting
  # logcat with instrumentation exceptions during a long smoke test.
  adb shell dumpsys window windows > "avd-linux-ui/${label}.txt" 2>/dev/null || true
  adb shell dumpsys activity activities >> "avd-linux-ui/${label}.txt" 2>/dev/null || true
}

# Capture immediately and once per second for 90 seconds so text printed by
# QML/Android just before a delayed crash remains visible instead of being
# overwritten by an ANR dialog or a final black frame.
capture_state "t00-start"
dump_window_state "t00-start"
(
  for second in {1..90}; do
    sleep 1
    capture_state "t$(printf '%02d' "$second")"
  done
) &
capture_pid=$!

# Pixel Launcher can show an ANR dialog while the Qt app is already visible.
# Dismiss the dialog as soon as it appears, while the capture loop keeps the
# original screen and the text printed by the app at each second.
for second in {1..90}; do
  label="t$(printf '%02d' "$second")"
  dump_window_state "$label"
  window_dump="$(adb shell dumpsys window windows 2>/dev/null || true)"
  if printf '%s' "$window_dump" | grep -q 'Application Not Responding: com.google.android.apps.nexuslauncher'; then
    # The ANR window is outside the Qt activity's uiautomator hierarchy. Its
    # stable Wait-row center on the Pixel 6 AVD is approximately (220, 1360).
    adb shell input tap 220 1360 || true
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

if grep -E "UnsatisfiedLinkError|QQmlApplicationEngine failed|module .* is not installed|Fatal signal|SIG(SEGV|ABRT|FPE|ILL|BUS)" avd-linux-logcat.txt; then
  echo "Application startup/runtime errors detected; see timed screenshots and logs." >&2
  exit 1
fi

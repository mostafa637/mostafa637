#!/usr/bin/env bash
set -euo pipefail

apk_root="${APK_ROOT:-apk/ish-qt-android-x86_64-debug}"
apk="$(find "$apk_root" -type f -name '*.apk' ! -name '*unsigned.apk' | head -n 1 || true)"
if [[ -z "$apk" ]]; then
  echo "No debug APK found under $apk_root" >&2
  find apk -maxdepth 5 -type f -print >&2 || true
  exit 1
fi

sdk_root="${ANDROID_HOME:-${ANDROID_SDK_ROOT:?ANDROID_SDK_ROOT is not set}}"
ndk_root="${ANDROID_NDK_ROOT:-${ANDROID_NDK_HOME:-${sdk_root}/ndk/27.2.12479018}}"
lldb_root="$ndk_root/toolchains/llvm/prebuilt/linux-x86_64"
lldb="$(find "$lldb_root/bin" -maxdepth 1 -type f -name lldb -print -quit 2>/dev/null || true)"
lldb_server="$(find "$lldb_root" -type f -name lldb-server -print -quit 2>/dev/null || true)"

if [[ -z "$lldb" || -z "$lldb_server" ]]; then
  echo "Unable to find host lldb or device lldb-server under $lldb_root" >&2
  find "$ndk_root/toolchains/llvm/prebuilt" -type f \( -name lldb -o -name lldb-server \) -print 2>/dev/null || true
  exit 2
fi

"$sdk_root/build-tools/36.0.0/apksigner" verify "$apk"
adb install -r "$apk"
adb shell am force-stop com.mostafa637.ishqt || true
adb logcat -c
adb shell monkey -p com.mostafa637.ishqt 1

mkdir -p avd-linux-captures avd-linux-ui avd-linux-logcat
adb logcat -v threadtime > avd-linux-logcat/full-runtime.log 2>&1 &
logcat_pid=$!
cleanup() {
  kill "$logcat_pid" 2>/dev/null || true
  kill "${lldb_pid:-0}" 2>/dev/null || true
  kill "${capture_pid:-0}" 2>/dev/null || true
  adb shell pkill -f '/data/local/tmp/ish-lldb-server' 2>/dev/null || true
}
trap cleanup EXIT

capture_state() {
  local label="$1"
  adb exec-out screencap -p > "avd-linux-captures/${label}.png" 2>/dev/null || true
  adb shell dumpsys window windows > "avd-linux-ui/${label}.txt" 2>/dev/null || true
  adb shell dumpsys activity activities >> "avd-linux-ui/${label}.txt" 2>/dev/null || true
  adb logcat -d -v threadtime -b crash > "avd-linux-logcat/${label}-crash.txt" 2>/dev/null || true
  printf 'Captured %s\n' "$label"
}

# Capture the complete lifecycle: launch, rootfs import, session start, prompt,
# command execution, and the final steady state. A 180-second interval remains
# independent of whether LLDB reports an exception.
capture_state t00-start
(
  for second in $(seq 1 180); do
    sleep 1
    capture_state "t$(printf '%03d' "$second")"
  done
) &
capture_pid=$!

# Wait for the Qt process and native session to appear.
pid=""
for _ in $(seq 1 60); do
  pid="$(adb shell pidof com.mostafa637.ishqt 2>/dev/null | tr -d '\r' | awk '{print $1}')"
  [[ -n "$pid" ]] && break
  sleep 1
done
if [[ -z "$pid" ]]; then
  echo "Application process did not start" >&2
  exit 3
fi
echo "Android application pid=$pid"

# Push a device-side LLDB server and attach without stopping the normal smoke
# sequence. LLDB output is recorded for the whole interval, not only crashes.
adb push "$lldb_server" /data/local/tmp/ish-lldb-server >/dev/null
adb shell chmod 700 /data/local/tmp/ish-lldb-server
adb forward tcp:5039 tcp:5039
adb shell "/data/local/tmp/ish-lldb-server g :5039 --attach $pid" > avd-linux-logcat/lldb-server.log 2>&1 &
server_pid=$!
sleep 2
(
  {
    echo '=== LLDB attach ==='
    echo 'process connect/attach: application pid='"$pid"
    timeout --signal=SIGINT 175s "$lldb" --batch \
      -o 'gdb-remote 127.0.0.1:5039' \
      -o 'process status' \
      -o 'thread list' \
      -o 'image list' \
      -o 'thread backtrace all' \
      -o 'process continue' \
      -o 'process status' \
      -o 'thread backtrace all' \
      -o 'process detach' \
      2>&1
    echo '=== LLDB session complete ==='
  } > avd-linux-logcat/lldb-full-runtime.log
) &
lldb_pid=$!

# Allow the UI and WebView to initialize, then run the same user-level command
# used by the smoke test while LLDB and the lifecycle capture continue running.
sleep 15
adb shell input tap 420 520 || true
sleep 1
adb shell input text 'apk%ssadd%spython' || true
adb shell input keyevent KEYCODE_ENTER || true
echo 'Sent command: apk add python'

# Keep monitoring until the full interval completes, including activity state
# and crash buffer, even if the process exits or LLDB detaches early.
for second in $(seq 1 165); do
  adb shell dumpsys activity activities > "avd-linux-ui/post-$(printf '%03d' "$second").txt" 2>/dev/null || true
  sleep 1
done
wait "$capture_pid" || true
wait "$lldb_pid" || true
kill "$server_pid" 2>/dev/null || true
adb shell dumpsys activity activities > avd-linux-activity.txt 2>&1 || true
adb exec-out cat /sdcard/Android/data/com.mostafa637.ishqt/files/ish-qt-errors.log > ish-qt-errors-device.log 2>/dev/null || true
adb exec-out screencap -p > avd-linux-screenshot.png 2>/dev/null || true
adb logcat -d -v threadtime > avd-linux-logcat.txt

# Do not fail merely because LLDB observed a normal stop; fail for actual
# native/runtime errors while retaining every diagnostic artifact.
if grep -E 'Fatal signal|SIG(SEGV|ABRT|FPE|ILL|BUS)|UnsatisfiedLinkError|QQmlApplicationEngine failed|module .* is not installed' avd-linux-logcat.txt; then
  echo 'Application startup/runtime errors detected; see complete LLDB and lifecycle logs.' >&2
  exit 4
fi

echo 'ANDROID_LLDB_FULL_RUNTIME=PASS'

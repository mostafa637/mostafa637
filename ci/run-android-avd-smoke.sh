#!/usr/bin/env bash
set -euo pipefail

apk_root="apk/ish-qt-android-x86_64"
apk="$(find "$apk_root" -type f -name '*.apk' ! -name '*unsigned.apk' | grep -E '/debug/|debug.*\.apk$' | head -n 1 || true)"
if [[ -z "$apk" ]]; then
  apk="$(find "$apk_root" -type f -name '*.apk' ! -name '*unsigned.apk' | head -n 1 || true)"
fi
if [[ -z "$apk" ]]; then
  apk="$(find "$apk_root" -type f -name '*unsigned.apk' | head -n 1 || true)"
fi
if [[ -z "$apk" ]]; then
  echo "No APK found under $apk_root" >&2
  find apk -maxdepth 4 -type f -print >&2 || true
  exit 1
fi

if [[ "$apk" == *-unsigned.apk ]]; then
  echo "Signing unsigned APK for emulator smoke test: $apk"
  sdk_root="${ANDROID_HOME:-${ANDROID_SDK_ROOT:?ANDROID_SDK_ROOT is not set}}"
  build_tools="$(find "$sdk_root/build-tools" -mindepth 1 -maxdepth 1 -type d | sort -V | tail -n 1)"
  keystore="${RUNNER_TEMP:-/tmp}/ish-avd-debug.keystore"
  aligned="${RUNNER_TEMP:-/tmp}/ish-qt-avd-aligned.apk"
  if [[ ! -f "$keystore" ]]; then
    keytool -genkeypair -v \
      -keystore "$keystore" -storepass android -keypass android \
      -alias androiddebugkey -keyalg RSA -keysize 2048 -validity 10000 \
      -dname "CN=Android Debug,O=Android,C=US"
  fi
  "$build_tools/zipalign" -f 4 "$apk" "$aligned"
  "$build_tools/apksigner" sign \
    --ks "$keystore" --ks-key-alias androiddebugkey \
    --ks-pass pass:android --key-pass pass:android "$aligned"
  "$build_tools/apksigner" verify "$aligned"
  apk="$aligned"
fi

echo "Installing APK: $apk"
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

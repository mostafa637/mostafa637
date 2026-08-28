#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="${ROOT}/dist"
mkdir -p "${OUT}"

build_native() {
  local build_dir="$1"
  shift
  rm -rf "${build_dir}" "${ROOT}/native/lib"
  cmake -S "${ROOT}/native" -B "${build_dir}" -G Ninja -DCMAKE_BUILD_TYPE=Release "$@"
  cmake --build "${build_dir}" --target ish_core_session ish_kernel ish_emu ish_fakefs ish_sqlite -j2
  mkdir -p "${ROOT}/native/lib"
  cp "${build_dir}"/lib/libish_*.a "${ROOT}/native/lib/"
  test -s "${ROOT}/native/lib/libish_core_session.a"
}

case "${1:-linux}" in
  linux)
    build_native "${ROOT}/native/build-linux"
    CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -trimpath -tags ishcore \
      -ldflags "-s -w -X gioui.org/app.ID=org.ish.go" \
      -o "${OUT}/ish-go-linux-x86_64" "${ROOT}/cmd/ish-go-gui"
    ;;
  android)
    command -v gogio >/dev/null 2>&1 || { echo "gogio is required for Android packaging" >&2; exit 1; }
    : "${ANDROID_NDK_ROOT:=${ANDROID_NDK_HOME:-}}"
    test -n "${ANDROID_NDK_ROOT}" || { echo "ANDROID_NDK_ROOT or ANDROID_NDK_HOME is required" >&2; exit 1; }
    for item in "arm64-v8a:arm64" "x86_64:amd64"; do
      abi="${item%%:*}"
      arch="${item##*:}"
      build_native "${ROOT}/native/build-${abi}" \
        -DCMAKE_TOOLCHAIN_FILE="${ANDROID_NDK_ROOT}/build/cmake/android.toolchain.cmake" \
        -DANDROID_ABI="${abi}" \
        -DANDROID_PLATFORM="android-${ANDROID_MIN_SDK:-28}"
      gogio -target android -arch "${arch}" -tags ishcore \
        -appid org.ish.go -o "${OUT}/ish-go-${abi}.apk" "${ROOT}/cmd/ish-go-gui"
    done
    ;;
  all)
    "${BASH_SOURCE[0]}" linux
    "${BASH_SOURCE[0]}" android
    ;;
  *)
    echo "usage: $0 {linux|android|all}" >&2
    exit 2
    ;;
esac

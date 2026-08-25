#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="${ROOT}/dist"
mkdir -p "${OUT}"

case "${1:-linux}" in
  linux)
    GOOS=linux GOARCH=amd64 go build -trimpath -o "${OUT}/ish-go-linux-x86_64" "${ROOT}/cmd/ish-go-gui"
    ;;
  android)
    command -v gogio >/dev/null 2>&1 || { echo "gogio is required for Android packaging" >&2; exit 1; }
    gogio -target android -arch arm64 -o "${OUT}/ish-go-arm64.apk" "${ROOT}/cmd/ish-go-gui"
    gogio -target android -arch amd64 -o "${OUT}/ish-go-x86_64.apk" "${ROOT}/cmd/ish-go-gui"
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

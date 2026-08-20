#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
assets="$root/android-qt/assets"
qrc="$assets/assets.qrc"

fail() {
    printf 'ERROR: %s\n' "$*" >&2
    exit 1
}

[[ -f "$qrc" ]] || fail "missing Qt resource file: ${qrc#$root/}"
[[ -f "$assets/rootfs/root.tar.gz" ]] || fail "missing bundled rootfs archive"
gzip -t "$assets/rootfs/root.tar.gz" || fail "bundled rootfs is not a valid gzip stream"

tar_listing=$(mktemp)
trap 'rm -f "$tar_listing"' EXIT
tar -tzf "$assets/rootfs/root.tar.gz" > "$tar_listing"
grep -Eq '(^|/)bin/sh$' "$tar_listing" \
    || fail "bundled rootfs does not contain bin/sh"
grep -Eq '(^|/)etc/apk/repositories$' "$tar_listing" \
    || fail "bundled rootfs does not contain etc/apk/repositories"

while IFS= read -r resource; do
    [[ -n "$resource" ]] || continue
    [[ -f "$assets/$resource" ]] || fail "qrc entry is missing: android-qt/assets/$resource"
done < <(sed -n 's#^[[:space:]]*<file[^>]*>\(.*\)</file>[[:space:]]*$#\1#p' "$qrc")

for original in "$root"/upstream/ish-ios/app/Icons/*.png; do
    name=$(basename "$original")
    [[ -f "$assets/ios-icons/$name" ]] \
        || fail "iSH original icon was not copied: $name"
done

for icon in \
    arrow-up-light arrow-up-dark arrow-down-light arrow-down-dark \
    arrow-left-light arrow-left-dark arrow-right-light arrow-right-dark \
    xmark-light xmark-dark paste-light paste-dark \
    hide-keyboard-light hide-keyboard-dark checkbox-light checkbox-dark; do
    [[ -f "$assets/ui/icons/$icon.svg" ]] || fail "missing QML icon: $icon.svg"
done

printf 'Bundled assets and iSH original icon contract are valid.\n'

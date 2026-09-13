#!/usr/bin/env bash
#
# Prepare the hterm terminal assets the way iSH iOS does, without making CI depend
# on it succeeding.
#
# iSH iOS regenerates hterm_all.js in an Xcode build phase:
#     cd $SRCROOT/deps/libapps && ./hterm/bin/mkdist
# The Android equivalent clones libapps on the runner and rebuilds. That is nice to
# prove, but it is a network + node dependency in the middle of an APK build, so a
# flaky clone must not take the whole pipeline down: the repo also carries a
# committed, already-built hterm_all.js, and we fall back to it.
#
# What is mandatory either way is the packaging invariant: cargo-apk copies this whole
# folder into the APK via [package.metadata.android] assets = "assets", so a file
# missing here ships a broken page with no way to notice at runtime. Note the honest
# scope: the Slint shell renders its own terminal today and does not yet host a
# WebView over these files - this checks what gets packaged, not what is drawn.
set -u -o pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
ASSET_DIRS="android-rs/assets/terminal"
REQUIRED="hterm_all.js term.js term.css term.html"
REBUILT=false

echo "=== 1. online regeneration (as the iOS Xcode build phase does) ==="
if command -v node >/dev/null 2>&1; then
  rm -rf /tmp/libapps
  if timeout 180 git clone --depth 1 https://github.com/ish-app/libapps /tmp/libapps >/dev/null 2>&1; then
    if (cd /tmp/libapps && timeout 600 ./hterm/bin/mkdist > /tmp/mkdist.log 2>&1); then
      BUILT=/tmp/libapps/hterm/dist/js/hterm_all.js
      if [ -s "$BUILT" ]; then
        echo "mkdist produced $BUILT ($(wc -c < "$BUILT") bytes)"
        for d in $ASSET_DIRS; do
          mkdir -p "$ROOT/$d"
          cp "$BUILT" "$ROOT/$d/hterm_all.js"
        done
        REBUILT=true
      else
        echo "::warning::mkdist produced no hterm_all.js; keeping the committed copy"
      fi
    else
      echo "::warning::mkdist failed; keeping the committed copy. Tail of its log:"
      tail -n 20 /tmp/mkdist.log || true
    fi
  else
    echo "::warning::could not clone ish-app/libapps (network?); keeping the committed copy"
  fi
else
  echo "::warning::node is not installed on this runner; skipping mkdist"
fi

echo "=== 2. verify the assets term.html expects (hard requirement) ==="
STATUS=0
for d in $ASSET_DIRS; do
  dir="$ROOT/$d"
  [ -d "$dir" ] || { echo "::error::$d does not exist"; STATUS=1; continue; }
  printf '%-46s %s\n' "$d" "$([ "$REBUILT" = true ] && echo 'rebuilt online' || echo 'committed copy')"
  for f in $REQUIRED; do
    if [ -s "$dir/$f" ]; then
      printf '   ok   %-16s %8s bytes\n' "$f" "$(wc -c < "$dir/$f")"
    else
      echo "   MISSING $f"
      STATUS=1
    fi
  done
  # The page is only functional if it actually wires the three scripts together.
  if grep -q 'hterm_all.js' "$dir/term.html" && grep -q 'term.js' "$dir/term.html"; then
    echo "   ok   term.html loads hterm_all.js + term.js"
  else
    echo "   FAIL term.html does not reference hterm_all.js / term.js"
    STATUS=1
  fi
  if grep -q 'hterm.Terminal' "$dir/hterm_all.js"; then
    echo "   ok   hterm_all.js defines hterm.Terminal"
  else
    echo "   FAIL hterm_all.js does not look like an hterm bundle"
    STATUS=1
  fi
  # Android must bridge to JavascriptInterface, not to webkit.messageHandlers (iOS),
  # and it has to reach the exact entry point MainActivity.onLoad() logs from.
  if grep -q 'window.Android.onLoad' "$dir/term.js"; then
    echo "   ok   term.js calls window.Android.onLoad (drives Terminal.loaded)"
  else
    echo "   FAIL term.js has no window.Android bridge - native.load() will never fire"
    STATUS=1
  fi
done

if [ "$STATUS" -ne 0 ]; then
  echo "::error::terminal assets are incomplete - the app would show a blank terminal"
fi
exit "$STATUS"

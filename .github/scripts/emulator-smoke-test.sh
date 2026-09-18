#!/usr/bin/env bash
#
# Emulator smoke test for the iSH Android APK.
#
# WHY THIS LIVES IN A FILE INSTEAD OF INLINE IN THE WORKFLOW
# -----------------------------------------------------------
# `reactivecircus/android-emulator-runner` does NOT hand your `script:` input to
# one shell. It splits it on newlines (src/script-parser.ts) and runs EVERY LINE
# as its own `sh -c '<line>'` (src/main.ts), after stripping `#`-comments and
# blank lines. Inside `script:` that means:
#
#   * multi-line `for` / `if` / `case` blocks are shell syntax errors
#   * variables, `cd` and `set -e` never survive from one line to the next
#   * any single line exiting non-zero fails the whole action
#
# The previous inline scripts in this repo did all three, which is why
# `APK_RUST=...` was always empty two lines later and the boot-check `for` loop
# died on its first line. Real logic belongs here; the workflow calls it as one
# line:  bash .github/scripts/emulator-smoke-test.sh
#
# The action exports ANDROID_SERIAL=emulator-$EMULATOR_PORT and EMULATOR_PORT,
# so we never hardcode a device serial.
#
# Exit status: 0 = every assertion passed, 1 = at least one failed. Evidence
# (screenshots, logcat, window/activity dumps, markdown report) is always written
# to $SCREENSHOT_DIR before exiting, so a red run stays debuggable.
#
# Optional env:
#   APK_PATH       explicit APK to install (skips the search)
#   SCREENSHOT_DIR where evidence goes (default android-rs/screenshots)
#   BOOT_SETTLE    seconds to let System UI idle (default 15)
#   LOG_TIMEOUT    seconds to wait for the hterm log markers (default 90)
#   UI_COMMANDS    'command:label' pairs typed into the terminal, ';' separated
#   FRAME_TIMEOUT  seconds to wait for the first *painted* frame (default 90) -
#                  swiftshader_indirect needs seconds per frame on a 2-core runner

set -u -o pipefail

OUT_DIR="${SCREENSHOT_DIR:-android-rs/screenshots}"
REPO_ROOT="${REPO_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
BOOT_SETTLE="${BOOT_SETTLE:-15}"
LOG_TIMEOUT="${LOG_TIMEOUT:-90}"
PKG=""
SELECTED_APK=""
EVIDENCE_COLLECTED=false
PASS=()
FAIL=()
INFO=()

note() { INFO+=("$1"); printf '>>> %s\n' "$1"; }
ok()   { PASS+=("$1"); printf '  [ok]   %s\n' "$1"; }
bad()  { FAIL+=("$1"); printf '  [FAIL] %s\n' "$1"; }

# `try` = run, echo, never abort. Failures are decided by explicit assertions.
# `< /dev/null` matters: adb forwards stdin to the device, so a child left attached
# to the loop's input pipe would swallow the remaining iterations.
try() {
  printf '  $ %s\n' "$*"
  sh -c "$*" < /dev/null 2>&1 | sed 's/^/  | /' || true
}
adb_sh() { adb shell "$@" < /dev/null 2>&1 | tr -d '\r'; }

shot() { # <name>
  local f="$OUT_DIR/$1.png"
  if adb exec-out screencap -p < /dev/null > "$f" 2>/dev/null && [ "$(wc -c < "$f")" -gt 1024 ]; then
    ok "screenshot $1.png ($(wc -c < "$f") bytes)"
  else
    # A missing screencap is an emulator/GPU flake, not an app failure: warn, do not redden the run.
    note "screenshot $1.png is missing or empty ($(wc -c < "$f" 2>/dev/null || echo 0) bytes)"
  fi
}

# Pixels, not log lines, are what separates "the app painted" from "the window exists".
# This check was added after a run reported 19/19 green while screen-boot.png was the
# black window plus the system gesture pill: android_main had run, the markers had been
# logged, and nothing had been drawn yet. Counting colours alone is not enough either -
# that frame measures 2 colours - so the test is colours *and* the share of lit pixels.
frame_detail() { # <png> -> 0 rendered, 1 blank, 2 cannot judge
  [ -f "$1" ] || { echo "missing file"; return 1; }
  if ! command -v python3 > /dev/null 2>&1; then
    echo "no python3 to judge the frame"
    return 2
  fi
  python3 "$REPO_ROOT/.github/scripts/png-detail-check.py" "$1"
}

# Capture repeatedly until a frame contains UI, and keep that frame as the boot shot.
shot_rendered() { # <name>
  local deadline=$(( SECONDS + ${FRAME_TIMEOUT:-90} )) last="" f="$OUT_DIR/$1.png" rc
  while :; do
    adb exec-out screencap -p < /dev/null > "$f" 2>/dev/null || true
    last="$(frame_detail "$f")"; rc=$?
    if [ "$rc" = "0" ]; then
      ok "screenshot $1.png shows a painted frame ($last)"
      return 0
    elif [ "$rc" = "2" ]; then
      # Not a pass: say out loud that the frame was never judged.
      note "screenshot $1.png was captured but not judged: $last"
      return 0
    fi
    [ "$SECONDS" -lt "$deadline" ] || break
    sleep 3
  done
  bad "no visible frame within ${FRAME_TIMEOUT:-90}s (last reading: $last) - markers alone are not UI"
  return 1
}

collect_evidence() {
  [ "$EVIDENCE_COLLECTED" = "true" ] && return 0
  EVIDENCE_COLLECTED=true
  [ "${1:-}" = "--with-interaction-shots" ] && shot screen-final
  adb logcat -d -v threadtime > "$OUT_DIR/logcat.txt" 2>/dev/null || true
  adb logcat -d -b crash -v threadtime > "$OUT_DIR/logcat-crash.txt" 2>/dev/null || true
  adb_sh dumpsys window windows > "$OUT_DIR/windows.txt" 2>/dev/null || true
  adb_sh dumpsys activity activities > "$OUT_DIR/activities.txt" 2>/dev/null || true
  [ -n "$PKG" ] && adb_sh dumpsys package "$PKG" > "$OUT_DIR/package.txt" 2>/dev/null
  try "adb shell uiautomator dump /sdcard/window.xml"
  adb exec-out cat /sdcard/window.xml < /dev/null > "$OUT_DIR/window.xml" 2>/dev/null || true
  ls -l "$OUT_DIR" | sed 's/^/  /'
}

report_and_exit() {
  collect_evidence
  {
    printf '\n## Android emulator smoke test\n\n'
    printf '| check | result |\n|---|---|\n'
    for c in "${PASS[@]}"; do printf '| %s | pass |\n' "$c"; done
    for c in "${FAIL[@]}"; do printf '| %s | **fail** |\n' "$c"; done
    printf '\n```\n'
    for c in "${INFO[@]}"; do printf '%s\n' "$c"; done
    printf '```\n\n'
    printf 'Full evidence (screenshots, logcat, dumpsys): workflow artifact `%s/`.\n' "$OUT_DIR"
  } >> "${GITHUB_STEP_SUMMARY:-/dev/null}"

  printf '\n=== summary ===\npassed: %s\nfailed: %s\n' "${#PASS[@]}" "${#FAIL[@]}"
  # Two extra channels for the verdict, because the step log is not always readable from
  # outside the run: an annotation (shown on the run page and through the API), and a text
  # copy inside the evidence directory, which the workflow publishes to a branch. Without
  # these, a run that died before any screenshot looked identical to a build failure.
  {
    printf 'apk: %s (%s bytes)\npackage: %s\npassed: %s\nfailed: %s\n\nFAILURES:\n' \
      "${SELECTED_APK:-none}" "$( [ -n "${SELECTED_APK:-}" ] && wc -c < "$SELECTED_APK" 2>/dev/null || echo 0 )" \
      "${PKG:-unresolved}" "${#PASS[@]}" "${#FAIL[@]}"
    if [ "${#FAIL[@]}" -gt 0 ]; then
      for c in "${FAIL[@]}"; do printf '  - %s\n' "$c"; done
    else
      printf '  (none)\n'
    fi
    printf '\nNOTES:\n'
    for c in "${INFO[@]}"; do printf '  %s\n' "$c"; done
  } > "$OUT_DIR/smoke-result.txt" 2>/dev/null || true
  if [ "${#FAIL[@]}" -gt 0 ]; then
    MSG="$(printf '%s\n' "${FAIL[@]}" | sed 's/%/%25/g' | awk '{printf "%s%s", (NR>1?"%0A":""), $0}')"
    # %%0A, not %0A: printf eats the latter as a hex-float conversion (it printed
    # "0X0P+0" and swallowed the message in testing), while GitHub needs the literal
    # four characters %0A to render a newline inside an annotation.
    printf '\n::error::emulator smoke test failed %d of %d checks (apk=%s)%%0A%s\n' \
      "${#FAIL[@]}" "$(( ${#PASS[@]} + ${#FAIL[@]} ))" "${SELECTED_APK:-none}" "$MSG"
  fi
  local status=0
  if [ "${#FAIL[@]}" -gt 0 ]; then
    printf 'FAILURES:\n'
    for c in "${FAIL[@]}"; do printf '  - %s\n' "$c"; done
    status=1
  fi
  # Feeds later steps; harmless outside Actions.
  if [ -n "${GITHUB_OUTPUT:-}" ]; then
    {
      echo "apk=$SELECTED_APK"
      echo "package=$PKG"
      echo "screenshot_dir=$OUT_DIR"
      echo "failures=${#FAIL[@]}"
    } >> "$GITHUB_OUTPUT"
  fi
  exit "$status"
}

# ---------------------------------------------------------------------------
# 0. device
# ---------------------------------------------------------------------------
if ! command -v adb >/dev/null 2>&1; then
  printf 'ERROR: adb is not on PATH; the emulator action installs platform-tools.\n' >&2
  exit 1
fi
mkdir -p "$OUT_DIR"
note "ANDROID_SERIAL=${ANDROID_SERIAL:-<unset>}  EMULATOR_PORT=${EMULATOR_PORT:-<unset>}"
adb wait-for-device
try "adb devices -l"

# ---------------------------------------------------------------------------
# 1. silence the emulator so a slow System UI cannot masquerade as our failure.
#    `hide_error_dialogs` replaces the old six-times-repeated
#    "uiautomator dump + tap 540,1060" ANR clicking loop with the real switch.
# ---------------------------------------------------------------------------
try "adb shell settings put global hide_error_dialogs 1"
try "adb shell settings put global anr_show_background 0"
try "adb shell settings put secure show_ime_with_hard_keyboard 0"
try "adb shell svc power stayon true"
try "adb shell dumpsys deviceidle disable"
note "letting System UI settle for ${BOOT_SETTLE}s"
sleep "$BOOT_SETTLE"

if [ "$(adb_sh getprop sys.boot_completed | tail -1)" = "1" ]; then
  ok "emulator finished booting"
else
  bad "emulator did not report sys.boot_completed=1"
  report_and_exit
fi

# ---------------------------------------------------------------------------
# 2. pick the APK: pure-Rust cargo-apk output first, then the Kotlin build
# ---------------------------------------------------------------------------
# cargo-apk writes target/android-artifacts/<profile>/apks/<apk_name>-<profile>.apk.
# The exact layout has moved between cargo-apk versions, so this is a preference
# list with a find-based fallback rather than one hardcoded path.
# Where cargo-apk actually writes the APK. It is target/<profile>/apk/<apk_name>.apk for
# the version this job installs (its own log says so: "Signing
# .../android-rs/target/debug/apk/ish-android-rs.apk"); the android-artifacts/... layout
# belongs to an older cargo-apk and is kept only as a fallback, because a wrong list here
# is indistinguishable from "the build produced nothing".
SELECTED_APK="${APK_PATH:-}"
if [ -n "$SELECTED_APK" ] && [ ! -f "$SELECTED_APK" ]; then
  bad "APK_PATH=$SELECTED_APK does not exist"
  report_and_exit
fi
if [ -z "$SELECTED_APK" ]; then
  for dir in android-rs/target/debug/apk android-rs/target/release/apk \
             android-rs/target/android-artifacts/debug/apks android-rs/target/android-artifacts/release/apks; do
    [ -d "$dir" ] || continue
    # *-unaligned.apk is cargo-apk's *intermediate*, before zipalign and apksigner: it is
    # written last, so a newest-first search picks it and adb answers
    # INSTALL_PARSE_FAILED_NO_CERTIFICATES. Only the signed APK is installable.
    SELECTED_APK="$(find "$dir" -name '*.apk' ! -name '*-unaligned.apk' -type f -printf '%T@ %p\n' 2>/dev/null | sort -rn | sed 's/^[^ ]* //' | head -n 1)"
    [ -n "$SELECTED_APK" ] && break
  done
fi
[ -z "$SELECTED_APK" ] && SELECTED_APK="$(find . -name '*.apk' -type f 2>/dev/null | head -1)"
if [ -z "$SELECTED_APK" ]; then
  note "no APK found in: android-rs/target/{debug,release}/apk and android-rs/target/android-artifacts/{debug,release}/apks (APK_PATH was: ${APK_PATH:-unset})"
  bad "no APK was produced by the build steps - nothing to install"
  report_and_exit
fi
ok "APK found: $SELECTED_APK ($(wc -c < "$SELECTED_APK") bytes)"
cp -f "$SELECTED_APK" "$OUT_DIR/app.apk" 2>/dev/null || true

# ---------------------------------------------------------------------------
# 3. install (allow test-only/unsigned), grant runtime permissions at install
# ---------------------------------------------------------------------------
note "installing with: adb install -r -g -t $SELECTED_APK"
INSTALL_OUT="$(adb install -r -g -t "$SELECTED_APK" < /dev/null 2>&1 | tr -d '\r')"
printf '%s\n' "$INSTALL_OUT" | sed 's/^/  | /'
if grep -q 'Success' <<<"$INSTALL_OUT"; then
  ok "adb install succeeded"
else
  # Signature mismatch against a leftover install: wipe it and retry once.
  PKG_LIST="$(adb_sh pm list packages)"
  note "retrying after uninstalling matching packages"
  for p in $(grep -E 'ish|gtk' <<<"$PKG_LIST" | sed -n 's/^package://p' || true); do
    try "adb shell pm uninstall $p"
  done
  INSTALL_OUT="$(adb install -r -g -t "$SELECTED_APK" < /dev/null 2>&1 | tr -d '\r')"
  printf '%s\n' "$INSTALL_OUT" | sed 's/^/  | /'
  if grep -q 'Success' <<<"$INSTALL_OUT"; then ok "adb install succeeded on retry"; else bad "adb install failed: $(printf '%s' "$INSTALL_OUT" | tr '\n' ' ' | cut -c1-200)"; report_and_exit; fi
fi

# Resolve the package name from the APK itself (build-tools/aapt) instead of
# grepping the device - the old `grep ish` fallback silently matched anything.
AAPT="$(ls "${ANDROID_HOME:-$HOME/Android/Sdk}"/build-tools/*/aapt 2>/dev/null | head -1 || true)"
[ -n "$AAPT" ] && PKG="$("$AAPT" dump packagename "$SELECTED_APK" 2>/dev/null | tr -d '\r' || true)"
PKG_LIST="$(adb_sh pm list packages)"
if [ -z "$PKG" ]; then
  for guess in com.ish.emulator.rust; do
    if grep -qx "package:$guess" <<<"$PKG_LIST"; then PKG="$guess"; break; fi
  done
fi
if [ -n "$PKG" ] && grep -qx "package:$PKG" <<<"$PKG_LIST"; then
  ok "package installed: $PKG"
elif [ -n "$PKG" ]; then
  bad "aapt says package $PKG but it is not installed"
  report_and_exit
else
  bad "could not resolve an installed package name"
  report_and_exit
fi

# ---------------------------------------------------------------------------
# 4. launch, then wait for the hterm bridge instead of sleeping blindly
# ---------------------------------------------------------------------------
try "adb shell pm clear '$PKG'"
adb logcat -c 2>/dev/null || true

ACTIVITY="$(adb_sh cmd package resolve-activity --brief -c android.intent.category.LAUNCHER "$PKG" | tail -1)"
case "$ACTIVITY" in
  "$PKG"/*) note "launcher activity: $ACTIVITY" ;;
  *)        ACTIVITY=""; note "resolve-activity found no launcher, falling back to monkey" ;;
esac

if [ -n "$ACTIVITY" ]; then
  START_OUT="$(adb_sh am start -W -n "$ACTIVITY")"
else
  START_OUT="$(adb_sh monkey -p "$PKG" -c android.intent.category.LAUNCHER 1)"
fi
printf '%s\n' "$START_OUT" | sed 's/^/  | /'
if grep -Eqi 'Status: ok|Events injected: 1' <<<"$START_OUT"; then
  ok "activity start accepted"
else
  bad "could not start $PKG: $(printf '%s' "$START_OUT" | tr '\n' ' ' | cut -c1-200)"
fi

wait_for_log() { # <pattern> <timeout-seconds>
  local pattern="$1" deadline=$(( SECONDS + $2 ))
  while [ "$SECONDS" -lt "$deadline" ]; do
    grep -q -- "$pattern" <<<"$(adb logcat -d 2>/dev/null || true)" && return 0
    sleep 2
  done
  return 1
}
if wait_for_log "iSH Android Pure Rust" "$LOG_TIMEOUT"; then
  ok "app logged its boot marker (android_main ran, so NativeActivity resolved the symbol)"
else
  bad "app never logged its boot marker within ${LOG_TIMEOUT}s (logcat tail follows in $OUT_DIR/logcat.txt)"
fi
# build_ui() reports this on Android through android_logger, i.e. only after
# AppWindow::new() and the AppDelegate wiring both succeeded. A missing symbol or a
# panic inside build_ui never prints it.
if wait_for_log "TerminalBuffer ready" 60; then
  ok "Slint UI built and wired (AppWindow + AppDelegate ready)"
else
  FAIL+=("the app never reported a ready UI - AppWindow::new or the wiring did not complete")
fi
adb logcat -d -b crash -v threadtime 2>/dev/null | grep -A 12 'FATAL EXCEPTION' > "$OUT_DIR/crash.txt" 2>/dev/null || true
adb logcat -d 2>/dev/null | grep -A 12 'FATAL EXCEPTION' >> "$OUT_DIR/crash.txt" 2>/dev/null || true
if [ -s "$OUT_DIR/crash.txt" ]; then
  bad "app crashed with a FATAL EXCEPTION - see $OUT_DIR/crash.txt"
else
  rm -f "$OUT_DIR/crash.txt"
  ok "no FATAL EXCEPTION logged"
fi

# ---------------------------------------------------------------------------
# 5. drive the terminal and screenshot every state (the visual proof)
# ---------------------------------------------------------------------------
shot_rendered screen-boot
UI_COMMANDS="${UI_COMMANDS:-apk add python3:apk;python3 --version:python;help:help;about:about;roots:roots;theme:theme;prefs:prefs}"
mapfile -t UI_PAIRS < <(printf '%s\n' "$UI_COMMANDS" | tr ';' '\n')
for pair in "${UI_PAIRS[@]}"; do
  [ -n "$pair" ] || continue
  cmd="${pair%%:*}"
  label="${pair##*:}"
  # `input text` has no spaces: encode them as %s, quote for the shell.
  encoded="$(printf '%s' "$cmd" | tr -d "\\" | sed 's/ /%s/g')"
  printf '  $ adb shell input text "%s" ; keyevent 66 (ENTER)\n' "$cmd"
  try "adb shell input text '$encoded'"
  try "adb shell input keyevent 66"
  # on_input_submitted reports every command through android_logger, so this is what
  # distinguishes "the UI processed the keys" from "nothing had focus" - seven identical
  # screenshots could not tell the two apart, which is how the previous run stayed quiet.
  # Anchored with $ on purpose.  Without it an input line that is never cleared would
  # still satisfy every check after the first, because the accumulated buffer *ends*
  # with the newest command ("command: apk add python3python3 --version" contains
  # "command: python3 --version").  The anchor makes "the app saw exactly this
  # command" the assertion, which is the one the screenshots cannot make for us.
  if wait_for_log "\[Android-RS\] command: $cmd\$" 20; then
    ok "typed input reached the focused input: '$cmd'"
  else
    bad "the app never logged exactly '$cmd' on its own line - either the keys reached no focused input, or the input line is not cleared and ENTER re-submitted the accumulated buffer (that is what the previous run did: 'command: apk add python3python3 --version')"
  fi
  sleep 2
  shot "screen-$label"
done

# ---------------------------------------------------------------------------
# 6. final verdict: the app must still be the live foreground UI
# ---------------------------------------------------------------------------
if grep -q "$PKG" <<<"$(adb_sh ps -A)"; then ok "app process alive"; else bad "app process $PKG died"; fi
RESUMED_LINE="$(adb_sh dumpsys activity activities | grep -E 'ResumedActivity|mResumedActivity' || true)"
if grep -q "$PKG" <<<"$RESUMED_LINE"; then
  ok "$PKG is the resumed (foreground) activity"
else
  bad "$PKG is not the resumed activity - its UI never stayed in front"
fi

collect_evidence
# This check is about evidence existing, nothing more: the *meaning* of the log was
# already asserted above (boot markers, per-command echoes, no FATAL EXCEPTION).  It
# used to demand 2000 bytes and then call anything shorter "empty", which is simply
# untrue of a short but genuine log.
log_bytes="$(wc -c < "$OUT_DIR/logcat.txt" 2>/dev/null | tr -d ' ' || echo 0)"
log_lines="$(grep -c . "$OUT_DIR/logcat.txt" 2>/dev/null || echo 0)"
case "$log_bytes" in ''|*[!0-9]*) log_bytes=0 ;; esac
if [ "${log_bytes:-0}" -gt 0 ]; then
  ok "logcat captured as evidence ($log_bytes bytes, $log_lines lines)"
else
  bad "logcat.txt is empty - the device logged nothing, so this run proves nothing"
fi
report_and_exit

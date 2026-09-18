#!/usr/bin/env python3
"""Differential test runner for the Go port.

Builds the AArch64 test corpus with mkelf.py, runs it under the C emulator the
port was made from (the reference) and under the Go build, and compares the
result blocks word by word. Any word that disagrees is reported by group and
index, so a failure points at the decoder path that produced it.

Usage:
    tests/asm/diffrun.py [group ...]

Each group is run as its own guest program, so one trapping snippet costs one
group's results and the report names the group that produced every word.

Environment:
    ARM64EMU_ORACLE  path to the reference arm64chroot (default
                     ./.oracle/arm64emu-user/arm64chroot, else
                     /tmp/arm64emu-user/arm64chroot)
    ARM64EMU_GO      path to the Go binary under test (default
                     ./arm64chroot-go, i.e. `go build -o arm64chroot-go .`)
    ARM64EMU_ROOTFS  directory used as the guest rootfs (a temp dir by default)
"""

import os
import struct
import subprocess
import sys
import tempfile

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import mkelf  # noqa: E402

# The reference build: the C emulator this port was made from, cloned and
# built by `.oracle/`-side instructions in docs/PORT.md.
DEFAULT_ORACLE = "/tmp/arm64emu-user/arm64chroot"
if os.path.exists("./.oracle/arm64emu-user/arm64chroot"):
    DEFAULT_ORACLE = "./.oracle/arm64emu-user/arm64chroot"
ORACLE = os.environ.get("ARM64EMU_ORACLE", DEFAULT_ORACLE)
GO = os.environ.get("ARM64EMU_GO", "./arm64chroot-go")


def run(emu, rootfs, guest_path, extra=(), timeout=120):
    try:
        p = subprocess.run([emu] + list(extra) + [rootfs, guest_path],
                           stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=timeout)
    except subprocess.TimeoutExpired:
        return None, None, "TIMEOUT"
    err = p.stderr.decode("utf-8", "replace").strip()
    err = "" if err in ("", "0") else err.splitlines()[0]
    return p.stdout, p.returncode, err


def main():
    wanted = sys.argv[1:]
    groups = [g for g in mkelf.SNIPPETS if not wanted or g in wanted]
    if not groups:
        sys.exit("diffrun: no such group: " + " ".join(wanted))
    for emu in (ORACLE, GO):
        if not os.path.exists(emu):
            sys.exit("diffrun: %s does not exist; set ARM64EMU_ORACLE / ARM64EMU_GO" % emu)

    rootfs = os.environ.get("ARM64EMU_ROOTFS")
    tmp = None
    if not rootfs:
        tmp = tempfile.mkdtemp(prefix="arm64emu-rootfs-")
        rootfs = tmp
    guest = "/corpus.elf"
    path = os.path.join(rootfs, "corpus.elf")
    emu_flags = os.environ.get("ARM64EMU_FLAGS", "").split()

    total = bad_total = known_total = 0
    for grp in groups:
        # One program per group: a snippet that traps or runs away then
        # costs one group's worth of results instead of the whole corpus, and
        # the report names the group that owns every word.
        with open(path, "wb") as f:
            f.write(mkelf.build(mkelf.SNIPPETS[grp]))
        os.chmod(path, 0o755)

        out_c, rc_c, err_c = run(ORACLE, rootfs, guest)
        out_g, rc_g, err_g = run(GO, rootfs, guest, emu_flags)
        if tmp is None:
            os.unlink(path)

        if out_c is None or out_g is None:
            print("  %-14s TIMEOUT (%s)" % (grp, "oracle" if out_c is None else "go"))
            bad_total += 1
            continue
        if len(out_c) != len(out_g) or rc_c != rc_g:
            print("  %-14s rc/length: oracle rc=%s %d bytes, go rc=%s %d bytes" %
                  (grp, rc_c, len(out_c), rc_g, len(out_g)))
            print("                 oracle: %s" % err_c)
            print("                 go:     %s" % err_g)
            bad_total += 1
            continue

        n = len(out_c) // 8
        total += n
        bad = known = 0
        for i in range(n):
            a, = struct.unpack_from("<Q", out_c, i * 8)
            b, = struct.unpack_from("<Q", out_g, i * 8)
            if a == b:
                continue
            nm = "%s[%d]" % (grp, i)
            if nm in mkelf.KNOWN:
                known += 1
                print("  known %-16s %s" % (nm, mkelf.KNOWN[nm]))
                continue
            if grp in mkelf.VOLATILE:
                print("  skip %-16s volatile: 0x%x vs 0x%x" % (nm, a, b))
                continue
            bad += 1
            print("  FAIL %-16s oracle=0x%016x go=0x%016x" % (nm, a, b))
        bad_total += bad
        known_total += known
        print("  %-14s %d/%d" % (grp, n - bad - known, n))

    if tmp:
        try:
            os.unlink(path)
        except OSError:
            pass
        os.rmdir(rootfs)

    print("%d/%d results match%s%s" % (total - bad_total - known_total, total,
                                       "" if not bad_total else "  -- %d mismatched" % bad_total,
                                       "" if not known_total else "  (%d known differences)" % known_total))
    return 1 if bad_total else 0


if __name__ == "__main__":
    sys.exit(main())

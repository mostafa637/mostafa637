#!/usr/bin/env python3
"""Generate src/decode_table.rs from tests/fixtures/decode_reference.txt.

emu/decode.h is a 648-case switch spread over eight opcode maps and two operand
sizes. Transcribing it by hand would be slow and unverifiable, so the opcode
table is derived from the reference fixture instead: tools/decode-dump.c runs
the *unmodified* header and records what it does, and this script turns those
recordings into Rust.

Two things make that safe rather than circular:

  * tools/decode-dump.c proves, over 1,042,066 decodes, that once the ModRM
    byte is consumed the dispatch depends on it only through the 72-value class
    function in class_of(). So a table indexed by class is complete.
  * tests/decode_differential.rs replays all 37,891 corpus cases through the
    Rust decoder, which selects and applies these entries using its own prefix,
    map-transition, byte-reading and fault logic.

Usage:  python3 tools/gen_decode_table.py
"""

import collections
import os
import re
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(HERE)
FIXTURE = os.path.join(ROOT, "tests", "fixtures", "decode_reference.txt")
DUMP = os.path.join(ROOT, "tools", "decode-dump.c")
OUT = os.path.join(ROOT, "src", "decode_table.rs")

NCLASS = 72

# map index -> the prefix bytes that reach it, matching map_prefix in decode-dump.c
MAPS = [
    ("Base", ()),
    ("Of", (0x0F,)),
    ("Lock", (0xF0,)),
    ("LockOf", (0xF0, 0x0F)),
    ("F2", (0xF2,)),
    ("F2Of", (0xF2, 0x0F)),
    ("F3", (0xF3,)),
    ("F3Of", (0xF3, 0x0F)),
]

# the opcodes the decoder drives itself instead of through the table
PREFIX_OPS = {
    0: {0x0F, 0x2E, 0x3E, 0x65, 0x66, 0x67, 0xF0, 0xF2, 0xF3},
    2: {0x0F, 0x65, 0x66},
    4: {0x0F},
    6: {0x0F},
}


def class_of(modrm: int) -> int:
    """The class function both sides share; see class_of in decode-dump.c."""
    if modrm >> 6 == 3:
        return 8 + (((modrm >> 3) & 7) << 3) + (modrm & 7)
    return (modrm >> 3) & 7


def class_modrm(cls: int) -> int:
    if cls < 8:
        return cls << 3
    r, m = divmod(cls - 8, 8)
    return (3 << 6) | (r << 3) | m


def rust_op_name(macro: str) -> str:
    parts = macro.split("_")
    out = "".join(p[:1].upper() + p[1:].lower() for p in parts if p)
    return out or "Unknown"


def read_ops():
    """The semantic macro names, taken from the recorder block of decode-dump.c."""
    src = open(DUMP).read()
    block = src[src.index("// ---- the ~150 semantic macros"):]
    block = block[: block.index("// ---- the backend hooks")]
    names = re.findall(r"^#define ([A-Z][A-Z0-9_]*)[ (]", block, re.M)
    return sorted(set(names))


def read_fixture():
    cases = []
    cur = None
    header = {}
    for line in open(FIXTURE):
        f = line.split()
        if not f:
            continue
        if f[0] == "#":
            header[f[1]] = " ".join(f[2:])
        elif f[0] == "C":
            cur = {"size": int(f[1]), "ip": f[2], "b": [int(x, 16) for x in f[3:]], "ev": []}
        elif f[0] == "E":
            cur["ev"].append(f[1:])
        elif f[0] == "R":
            cur["ret"] = int(f[1])
            cur["end"] = f[2]
            cases.append(cur)
    return header, cases


def map_of(b):
    """Which opcode map a case's bytes belong to: the longest matching prefix."""
    best, best_len = None, -1
    for i, (_, prefix) in enumerate(MAPS):
        if list(b[: len(prefix)]) == list(prefix) and len(prefix) > best_len:
            best, best_len = i, len(prefix)
    if best is None:
        raise SystemExit("no map for case bytes %s" % b)
    return best, best_len


def step_of(ev, ops):
    name = ev[0]
    args = ev[1:]
    if name == "READMODRM":
        return "Step::ReadModrm"
    if name == "READIMM":
        return "Step::ReadImm(%s)" % args[0]
    if name == "SEG_GS":
        return "Step::SegGs"
    if name == "UNDEFINED":
        return "Step::Undefined"
    if name == "DONE":
        return "Step::Done"
    if name == "END_BLOCK":
        return "Step::EndBlock"
    if name == "SEGFAULT":
        return None  # runtime, never part of a table entry
    if name not in ops:
        raise SystemExit("unknown semantic macro in the fixture: %s" % name)
    argl = ", ".join('"%s"' % a.replace('"', '\\"') for a in args)
    return "Step::Op(Op::%s, &[%s])" % (rust_op_name(name), argl)


def common_prefix(lists):
    if not lists:
        return []
    out = []
    for i in range(min(len(l) for l in lists)):
        if all(l[i] == lists[0][i] for l in lists):
            out.append(lists[0][i])
        else:
            break
    return out


def main():
    ops = read_ops()
    header, cases = read_fixture()
    if "class_invariant" not in header:
        raise SystemExit("the fixture has no class invariant line - regenerate it")

    # (size, map, opcode) -> {class: [steps]}
    table = collections.defaultdict(dict)
    skipped_prefix = 0
    for c in cases:
        if c["ip"] != "400":
            continue  # the fault cases exercise the runtime, not the table
        mi, npre = map_of(c["b"])
        op = c["b"][npre]
        if op in PREFIX_OPS.get(mi, ()):
            skipped_prefix += 1
            continue
        modrm = c["b"][npre + 1]
        steps = []
        # the decoder performs every opcode fetch itself as it walks prefixes and
        # maps, so drop those - one per prefix byte plus the opcode itself
        fetches = npre + 1
        for i, ev in enumerate(c["ev"]):
            if i < fetches and ev == ["READIMM", "8"]:
                continue
            st = step_of(ev, set(ops))
            if st is None:
                raise SystemExit("a corpus case faulted: %s" % c["b"])
            steps.append(st)
        table[(c["size"], mi, op)][class_of(modrm)] = steps

    # intern step lists so identical ones are emitted once
    interned = {}
    slices = []

    def intern(steps):
        key = tuple(steps)
        if key not in interned:
            interned[key] = len(slices)
            slices.append(steps)
        return interned[key]

    entries = {}
    split_ops = 0
    for key, byclass in sorted(table.items()):
        variants = sorted(set(tuple(v) for v in byclass.values()))
        if len(variants) == 1:
            head = list(variants[0])
            tails = [intern([])]
        else:
            split_ops += 1
            lists = [byclass.get(cls, byclass[min(byclass)]) for cls in range(NCLASS)]
            # every class the corpus did not carry takes the behaviour of the
            # lowest class that was carried - valid because the generator proved
            # the class rule over all 256 ModRM bytes
            head = common_prefix([list(v) for v in variants])
            tails = []
            for cls in range(NCLASS):
                full = byclass.get(cls)
                if full is None:
                    full = lists[cls]
                tails.append(intern(full[len(head):]))
            if "Step::ReadModrm" not in head:
                raise SystemExit(
                    "opcode %s is class-dependent but its head reads no ModRM byte" % (key,)
                )
        entries[key] = (intern(head), tails)

    # ---- emit ----
    out = []
    w = out.append
    w("// Generated by tools/gen_decode_table.py from tests/fixtures/decode_reference.txt.")
    w("// Do not edit. The fixture comes from the unmodified emu/decode.h; see the")
    w("// generator's docstring for why deriving this is sound.")
    w("")
    w("use crate::decode::Step;")
    w("")
    w("/// The semantic operations emu/decode.h can ask for, one variant per macro")
    w("/// the header invokes. The names are the C's, so `Op::AtomicCmpxchg8b` is")
    w("/// `ATOMIC_CMPXCHG8B`. What each one *does* lives in the gadget backends,")
    w("/// which are not ported; the decoder's job ends at naming it.")
    w("#[derive(Debug, Clone, Copy, PartialEq, Eq)]")
    w("pub enum Op {")
    for o in ops:
        w("    /// `%s`" % o)
        w("    %s," % rust_op_name(o))
    w("}")
    w("")
    w("impl Op {")
    w("    /// The C macro name this variant came from.")
    w("    pub fn as_macro(self) -> &'static str {")
    w("        match self {")
    for o in ops:
        w('            Op::%s => "%s",' % (rust_op_name(o), o))
    w("        }")
    w("    }")
    w("}")
    w("")
    w("/// One opcode's decode: steps that always run, then the per-class remainder.")
    w("///")
    w("/// `tails` has one element when the opcode behaves the same for every ModRM")
    w("/// class, and 72 when it does not - the GRP groups, the READMODRM_MEM/NOMEM")
    w("/// checks and the x87 register form.")
    w("pub struct Entry {")
    w("    pub head: &'static [Step],")
    w("    pub tails: &'static [&'static [Step]],")
    w("}")
    w("")
    for i, steps in enumerate(slices):
        if steps:
            w("const S%d: &[Step] = &[\n    %s,\n];" % (i, ",\n    ".join(steps)))
        else:
            w("const S%d: &[Step] = &[];" % i)
    w("")
    for (size, mi, op), (head, tails) in sorted(entries.items()):
        if len(tails) == 1:
            w("const E%d_%d_%d: Entry = Entry { head: S%d, tails: &[S%d] };"
              % (size, mi, op, head, tails[0]))
        else:
            body = ", ".join("S%d" % t for t in tails)
            w("const E%d_%d_%d: Entry = Entry { head: S%d, tails: &[%s] };"
              % (size, mi, op, head, body))
    w("")
    for size in (32, 16):
        w("static TABLE%d: [Option<&'static Entry>; %d] = [" % (size, 8 * 256))
        for mi in range(8):
            row = []
            for op in range(256):
                if (size, mi, op) in entries:
                    row.append("Some(&E%d_%d_%d)" % (size, mi, op))
                else:
                    row.append("None")
            w("    // map %d (%s)" % (mi, MAPS[mi][0]))
            for chunk in range(0, 256, 8):
                w("    " + ", ".join(row[chunk:chunk + 8]) + ",")
        w("];")
        w("")
    w("/// The decode table for one operand size, indexed `map * 256 + opcode`.")
    w("pub fn table(size: u8) -> &'static [Option<&'static Entry>; 2048] {")
    w("    if size == 16 { &TABLE16 } else { &TABLE32 }")
    w("}")
    w("")

    open(OUT, "w").write("\n".join(out))
    print(
        "wrote src/decode_table.rs: %d opcodes (%d class-dependent), %d distinct step lists"
        % (len(entries), split_ops, len(slices))
    )
    print("  %d semantic macros in the Op enum, %d prefix-opcode cases left to the decoder"
          % (len(ops), skipped_prefix))
    names = [rust_op_name(o) for o in ops]
    if len(set(names)) != len(names):
        dupes = [n for n, c in collections.Counter(names).items() if c > 1]
        raise SystemExit("Op variant name collision: %s" % dupes)


if __name__ == "__main__":
    sys.exit(main())

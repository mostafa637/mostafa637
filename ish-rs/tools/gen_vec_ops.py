#!/usr/bin/env python3
"""Generate the op table for the vec/mmx differential test from emu/vec.h.

`emu/vec.h` declares 166 functions in 30 distinct signatures. Hand-copying
166 call sites into a C driver *and* a Rust match would be 332 chances to typo
a name, so both sides are derived from the header instead:

    tools/gen_vec_ops.py /path/to/ish > tools/vec-ops.inc
    tools/gen_vec_ops.py /path/to/ish --rust > /tmp/rust_arms.txt

The C driver includes vec-ops.inc; the Rust test's dispatch match is the
--rust output, pasted in. Re-run both if vec.h ever changes: an op that is
missing on the Rust side makes the test panic on its name.
"""
import re
import sys

# normalized parameter list (cpu first arg stripped) -> (kind, fn member)
KINDS = {
    'const union xmm_reg *src, union xmm_reg *dst':
        ('K_XMM_XMM_C', 'xmm_xmm_c', 'struct cpu_state *, const union xmm_reg *, union xmm_reg *'),
    'union xmm_reg *src, union xmm_reg *dst':
        ('K_XMM_XMM', 'xmm_xmm', 'struct cpu_state *, union xmm_reg *, union xmm_reg *'),
    'const union mm_reg *src, union mm_reg *dst':
        ('K_MM_MM_C', 'mm_mm_c', 'struct cpu_state *, const union mm_reg *, union mm_reg *'),
    'union mm_reg *src, union mm_reg *dst':
        ('K_MM_MM', 'mm_mm', 'struct cpu_state *, union mm_reg *, union mm_reg *'),
    'const void *src, void *dst':
        ('K_VOID_VOID', 'void_void', 'struct cpu_state *, const void *, void *'),
    'const uint8_t amount, union xmm_reg *dst':
        ('K_IMM_XMM', 'imm_xmm', 'struct cpu_state *, const uint8_t, union xmm_reg *'),
    'uint8_t amount, union xmm_reg *dst':
        ('K_IMM_XMM_NC', 'imm_xmm_nc', 'struct cpu_state *, uint8_t, union xmm_reg *'),
    'const uint8_t amount, union mm_reg *dst':
        ('K_IMM_MM', 'imm_mm', 'struct cpu_state *, const uint8_t, union mm_reg *'),
    'const double *src, double *dst':
        ('K_F64_F64', 'f64_f64', 'struct cpu_state *, const double *, double *'),
    'const float *src, float *dst':
        ('K_F32_F32', 'f32_f32', 'struct cpu_state *, const float *, float *'),
    'const union xmm_reg *src, union xmm_reg *dst, uint8_t encoding':
        ('K_XMM_XMM_ENC', 'xmm_xmm_enc', 'struct cpu_state *, const union xmm_reg *, union xmm_reg *, uint8_t'),
    'const union mm_reg *src, union mm_reg *dst, uint8_t encoding':
        ('K_MM_MM_ENC', 'mm_mm_enc', 'struct cpu_state *, const union mm_reg *, union mm_reg *, uint8_t'),
    'const uint64_t *src, union xmm_reg *dst':
        ('K_U64_XMM', 'u64_xmm', 'struct cpu_state *, const uint64_t *, union xmm_reg *'),
    'const union xmm_reg *src, uint64_t *dst':
        ('K_XMM_U64', 'xmm_u64', 'struct cpu_state *, const union xmm_reg *, uint64_t *'),
    'const union xmm_reg *src, uint32_t *dst':
        ('K_XMM_U32', 'xmm_u32', 'struct cpu_state *, const union xmm_reg *, uint32_t *'),
    'const union mm_reg *src, uint32_t *dst':
        ('K_MM_U32', 'mm_u32', 'struct cpu_state *, const union mm_reg *, uint32_t *'),
    'const float *src, const float *dst':
        ('K_F32_F32_CC', 'f32_f32_cc', 'struct cpu_state *, const float *, const float *'),
    'const double *src, const double *dst':
        ('K_F64_F64_CC', 'f64_f64_cc', 'struct cpu_state *, const double *, const double *'),
    'const double *src, union xmm_reg *dst, uint8_t type':
        ('K_F64_XMM_TYPE', 'f64_xmm_type', 'struct cpu_state *, const double *, union xmm_reg *, uint8_t'),
    'const float *src, union xmm_reg *dst, uint8_t type':
        ('K_F32_XMM_TYPE', 'f32_xmm_type', 'struct cpu_state *, const float *, union xmm_reg *, uint8_t'),
    'const union xmm_reg *src, union xmm_reg *dst, uint8_t type':
        ('K_XMM_XMM_TYPE', 'xmm_xmm_type', 'struct cpu_state *, const union xmm_reg *, union xmm_reg *, uint8_t'),
    'const int32_t *src, double *dst':
        ('K_I32_F64', 'i32_f64', 'struct cpu_state *, const int32_t *, double *'),
    'const double *src, int32_t *dst':
        ('K_F64_I32', 'f64_i32', 'struct cpu_state *, const double *, int32_t *'),
    'const double *src, float *dst':
        ('K_F64_F32', 'f64_f32', 'struct cpu_state *, const double *, float *'),
    'const int32_t *src, float *dst':
        ('K_I32_F32', 'i32_f32', 'struct cpu_state *, const int32_t *, float *'),
    'const float *src, int32_t *dst':
        ('K_F32_I32', 'f32_i32', 'struct cpu_state *, const float *, int32_t *'),
    'const float *src, double *dst':
        ('K_F32_F64', 'f32_f64', 'struct cpu_state *, const float *, double *'),
    'const uint32_t *src, union mm_reg *dst, uint8_t index':
        ('K_U32_MM_IDX', 'u32_mm_idx', 'struct cpu_state *, const uint32_t *, union mm_reg *, uint8_t'),
    'const uint32_t *src, union xmm_reg *dst, uint8_t index':
        ('K_U32_XMM_IDX', 'u32_xmm_idx', 'struct cpu_state *, const uint32_t *, union xmm_reg *, uint8_t'),
    'const union xmm_reg *src, uint32_t *dst, uint8_t index':
        ('K_XMM_U32_IDX', 'xmm_u32_idx', 'struct cpu_state *, const union xmm_reg *, uint32_t *, uint8_t'),
}

# The Rust call shape for each kind, as a complete match arm body. `src`/`dst`
# are XmmReg, `out` is a 16-byte side buffer standing in for the ops whose C
# destination is a bare scalar, `cpu` is the CpuState, and `c` holds the case
# parameters. Each arm is self-contained: the mm and void kinds build a view of
# the same bytes and write it back, which is what the C does by casting the
# pointer.
RUST_CALL = {
    'K_XMM_XMM_C': 'vec::{n}(&src, &mut dst)',
    'K_XMM_XMM': 'vec::{n}(&mut src, &mut dst)',
    # the mm/void kinds get their operands from views of the same bytes and
    # write the result back, exactly as the C does through a cast pointer
    'K_MM_MM_C': '{{ let sm = MmReg(src.qw(0)); let mut dm = MmReg(dst.qw(0)); '
                 'vec::{n}(&sm, &mut dm); dst.set_qw(0, dm.qw()); }}',
    'K_MM_MM': '{{ let mut sm = MmReg(src.qw(0)); let mut dm = MmReg(dst.qw(0)); '
                'vec::{n}(&mut sm, &mut dm); dst.set_qw(0, dm.qw()); }}',
    'K_VOID_VOID': '{{ let sb = src.bytes(); let mut db = dst.bytes(); '
                   'vec::{n}(&sb, &mut db); dst = XmmReg::from_bytes(db); }}',
    'K_IMM_XMM': 'vec::{n}(c.amount, &mut dst)',
    'K_IMM_XMM_NC': 'vec::{n}(c.amount, &mut dst)',
    'K_IMM_MM': '{{ let mut dm = MmReg(dst.qw(0)); vec::{n}(c.amount, &mut dm); '
                'dst.set_qw(0, dm.qw()); }}',
    'K_F64_F64': '{{ let s = src.f64(0); let mut d = dst.f64(0); '
                 'vec::{n}(&s, &mut d); dst.set_f64(0, d); }}',
    'K_F32_F32': '{{ let s = src.f32(0); let mut d = dst.f32(0); '
                 'vec::{n}(&s, &mut d); dst.set_f32(0, d); }}',
    'K_XMM_XMM_ENC': 'vec::{n}(&src, &mut dst, c.encoding)',
    'K_MM_MM_ENC': '{{ let sm = MmReg(src.qw(0)); let mut dm = MmReg(dst.qw(0)); '
                   'vec::{n}(&sm, &mut dm, c.encoding); dst.set_qw(0, dm.qw()); }}',
    'K_U64_XMM': '{{ let s = src.qw(0); vec::{n}(&s, &mut dst); }}',
    'K_XMM_U64': '{{ let mut o = 0u64; vec::{n}(&src, &mut o); '
                 'out[..8].copy_from_slice(&o.to_le_bytes()); }}',
    'K_XMM_U32': '{{ let mut o = 0u32; vec::{n}(&src, &mut o); '
                 'out[..4].copy_from_slice(&o.to_le_bytes()); }}',
    'K_MM_U32': '{{ let sm = MmReg(src.qw(0)); let mut o = 0u32; vec::{n}(&sm, &mut o); '
                 'out[..4].copy_from_slice(&o.to_le_bytes()); }}',
    'K_F32_F32_CC': '{{ let s = src.f32(0); let d = dst.f32(0); vec::{n}(&mut cpu, &s, &d); }}',
    'K_F64_F64_CC': '{{ let s = src.f64(0); let d = dst.f64(0); vec::{n}(&mut cpu, &s, &d); }}',
    'K_F64_XMM_TYPE': '{{ let s = src.f64(0); vec::{n}(&s, &mut dst, c.type_); }}',
    'K_F32_XMM_TYPE': '{{ let s = src.f32(0); vec::{n}(&s, &mut dst, c.type_); }}',
    'K_XMM_XMM_TYPE': 'vec::{n}(&src, &mut dst, c.type_)',
    'K_I32_F64': '{{ let s = src.u32(0) as i32; let mut d = dst.f64(0); '
                 'vec::{n}(&s, &mut d); dst.set_f64(0, d); }}',
    'K_F64_I32': '{{ let s = src.f64(0); let mut d = dst.u32(0) as i32; '
                 'vec::{n}(&s, &mut d); dst.set_u32(0, d as u32); }}',
    'K_F64_F32': '{{ let s = src.f64(0); let mut d = dst.f32(0); '
                 'vec::{n}(&s, &mut d); dst.set_f32(0, d); }}',
    'K_I32_F32': '{{ let s = src.u32(0) as i32; let mut d = dst.f32(0); '
                 'vec::{n}(&s, &mut d); dst.set_f32(0, d); }}',
    'K_F32_I32': '{{ let s = src.f32(0); let mut d = dst.u32(0) as i32; '
                 'vec::{n}(&s, &mut d); dst.set_u32(0, d as u32); }}',
    'K_F32_F64': '{{ let s = src.f32(0); let mut d = dst.f64(0); '
                 'vec::{n}(&s, &mut d); dst.set_f64(0, d); }}',
    'K_U32_MM_IDX': '{{ let s = src.u32(0); let mut dm = MmReg(dst.qw(0)); '
                    'vec::{n}(&s, &mut dm, c.index); dst.set_qw(0, dm.qw()); }}',
    'K_U32_XMM_IDX': '{{ let s = src.u32(0); vec::{n}(&s, &mut dst, c.index); }}',
    'K_XMM_U32_IDX': '{{ let mut o = 0u32; vec::{n}(&src, &mut o, c.index); '
                     'out[..4].copy_from_slice(&o.to_le_bytes()); }}',
}


def normalize(args):
    args = args.replace('NO_CPU', 'struct cpu_state *cpu').replace('UNUSED(cpu)', 'cpu')
    args = re.sub(r'\s+', ' ', args).strip()
    parts = [p.strip() for p in args.split(',')]
    assert parts[0].startswith('struct cpu_state'), parts[0]
    return ', '.join(parts[1:])


def parse(header_path):
    src = open(header_path).read()
    out = []
    for name, args in re.findall(r'^(?:void|bool)\s+(vec_\w+)\s*\(([^;]*)\)\s*;', src, re.M):
        key = normalize(args)
        if key not in KINDS:
            raise SystemExit(f'{name}: unrecognised signature: {key}')
        out.append((name, KINDS[key]))
    return out


def emit_c(ops):
    print('// Generated by tools/gen_vec_ops.py from emu/vec.h - do not edit.')
    print('// %d operations in %d signature classes.'
          % (len(ops), len({k[0] for _, k in ops})))
    print('enum vec_kind {')
    seen = []
    for _, (kind, _, _) in ops:
        if kind not in seen:
            seen.append(kind)
    print(',\n'.join('    ' + k for k in seen) + ',')
    print('    K_COUNT')
    print('};')
    print('union vec_fn {')
    for kind in seen:
        member = next(m for _, (k, m, _) in ops if k == kind)
        proto = next(p for _, (k, _, p) in ops if k == kind)
        print(f'    void (*{member})({proto});')
    print('};')
    print('struct vec_op { const char *name; enum vec_kind kind; union vec_fn fn; };')
    print('static const struct vec_op vec_ops[] = {')
    for name, (kind, member, _) in ops:
        print(f'    {{ "{name}", {kind}, .fn.{member} = {name} }},')
    print('};')
    print('#define N_VEC_OPS ((int) (sizeof(vec_ops) / sizeof(vec_ops[0])))')


def emit_rust(ops):
    print('        // Generated by tools/gen_vec_ops.py from emu/vec.h - do not edit.')
    for name, (kind, _, _) in ops:
        print(f'        "{name}" => {RUST_CALL[kind].format(n=name)},')


def main():
    if len(sys.argv) < 2:
        raise SystemExit(__doc__)
    ish = sys.argv[1]
    rust = '--rust' in sys.argv
    ops = parse(f'{ish}/emu/vec.h')
    if rust:
        emit_rust(ops)
    else:
        emit_c(ops)


if __name__ == '__main__':
    main()

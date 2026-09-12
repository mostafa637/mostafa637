//! `kernel/vdso.{h,c}` — vdso image and symbol lookup.
//!
//! The C version embeds `vdso/libvdso.so.elf` via inline assembly and parses
//! its `PT_DYNAMIC` segment to resolve symbols. This Rust port implements the
//! same parsing over an explicit byte slice, so an embedding can supply the
//! vdso image via `include_bytes!` or a host-provided buffer.
//!
//! The on-disk format is ELF32 i386 little-endian; parsing is done with
//! `crate::elf`.

use crate::elf::{DynEnt, ElfHeader, ElfSym, PrgHeader, DT_HASH, DT_NULL, DT_STRTAB, DT_SYMTAB, PT_DYNAMIC};

/// `VDSO_PAGES` from `tools/ptraceomatic-config.h` and `kernel/vdso.h`.
pub const VDSO_PAGES: usize = 2;
pub const VDSO_SIZE: usize = VDSO_PAGES * 4096;
/// `VVAR_PAGES` from the same header, kept for completeness.
pub const VVAR_PAGES: usize = 4;

/// Look up a symbol's `st_value` in a vdso ELF image.
///
/// Mirrors `vdso_symbol(const char *name)` in `kernel/vdso.c`.
pub fn vdso_symbol(image: &[u8], name: &str) -> Option<u32> {
    let hdr = ElfHeader::from_le_bytes(image.get(0..52)?)?;
    if !hdr.is_valid_i386() {
        panic!("invalid vdso: bad ELF header");
    }

    let ph_off = hdr.prghead_off as usize;
    let ph_entsz = hdr.phent_size as usize;
    let ph_count = hdr.phent_count as usize;
    let mut dyn_offset: Option<usize> = None;

    for i in 0..ph_count {
        let off = ph_off + i * ph_entsz;
        let ph_bytes = image.get(off..off + 32)?;
        let ph = PrgHeader::from_le_bytes(ph_bytes)?;
        if ph.type_ == PT_DYNAMIC {
            dyn_offset = Some(ph.offset as usize);
            break;
        }
    }

    let dyn_off = dyn_offset.unwrap_or_else(|| panic!("invalid vdso: no PT_DYNAMIC"));

    let mut strtab: Option<usize> = None;
    let mut symtab: Option<usize> = None;
    let mut hash: Option<usize> = None;

    let mut cur = dyn_off;
    loop {
        let ent_bytes = image.get(cur..cur + 8)?;
        let ent = DynEnt::from_le_bytes(ent_bytes)?;
        if ent.tag == DT_NULL {
            break;
        }
        match ent.tag {
            DT_STRTAB => strtab = Some(ent.val as usize),
            DT_SYMTAB => symtab = Some(ent.val as usize),
            DT_HASH => hash = Some(ent.val as usize),
            _ => {}
        }
        cur += 8;
        if cur + 8 > image.len() {
            panic!("invalid vdso: truncated dynamic");
        }
    }

    let strtab_off = strtab.unwrap_or_else(|| panic!("invalid vdso: no DT_STRTAB"));
    let symtab_off = symtab.unwrap_or_else(|| panic!("invalid vdso: no DT_SYMTAB"));
    let hash_off = hash.unwrap_or_else(|| panic!("invalid vdso: no DT_HASH"));

    let hash_bytes = image.get(hash_off..hash_off + 8)?;
    let nbucket = u32::from_le_bytes([hash_bytes[0], hash_bytes[1], hash_bytes[2], hash_bytes[3]]);
    let nchain = u32::from_le_bytes([hash_bytes[4], hash_bytes[5], hash_bytes[6], hash_bytes[7]]);
    let _ = nbucket;
    let num_syms = nchain as usize;

    for i in 0..num_syms {
        let sym_off = symtab_off + i * 16;
        let sym_bytes = image.get(sym_off..sym_off + 16)?;
        let sym = ElfSym::from_le_bytes(sym_bytes)?;
        let name_off = strtab_off + sym.name as usize;
        let mut end = name_off;
        while end < image.len() && image[end] != 0 {
            end += 1;
        }
        if end >= image.len() {
            continue;
        }
        let sym_name = core::str::from_utf8(&image[name_off..end]).unwrap_or("");
        if sym_name == name {
            return Some(sym.value);
        }
    }

    None
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::elf::{ELF_MAGIC, ELF_32BIT, ELF_LITTLEENDIAN, ELF_X86, PT_DYNAMIC, DT_STRTAB, DT_SYMTAB, DT_HASH, DT_NULL};

    fn build_minimal_vdso_with_symbols(symbols: &[(&str, u32)]) -> Vec<u8> {
        let mut img = vec![0u8; 4096];
        img[0..4].copy_from_slice(&ELF_MAGIC);
        img[4] = ELF_32BIT;
        img[5] = ELF_LITTLEENDIAN;
        img[18..20].copy_from_slice(&ELF_X86.to_le_bytes());
        img[28..32].copy_from_slice(&52u32.to_le_bytes());
        img[40..42].copy_from_slice(&52u16.to_le_bytes());
        img[42..44].copy_from_slice(&32u16.to_le_bytes());
        img[44..46].copy_from_slice(&1u16.to_le_bytes());
        let ph_off = 52;
        img[ph_off..ph_off + 4].copy_from_slice(&PT_DYNAMIC.to_le_bytes());
        img[ph_off + 4..ph_off + 8].copy_from_slice(&0x100u32.to_le_bytes());
        let dyn_off = 0x100;
        let mut cur = dyn_off;
        let mut write_dyn = |tag: u32, val: u32| {
            img[cur..cur + 4].copy_from_slice(&tag.to_le_bytes());
            img[cur + 4..cur + 8].copy_from_slice(&val.to_le_bytes());
            cur += 8;
        };
        write_dyn(DT_STRTAB, 0x200);
        write_dyn(DT_SYMTAB, 0x300);
        write_dyn(DT_HASH, 0x400);
        write_dyn(DT_NULL, 0);
        img[0x400..0x404].copy_from_slice(&1u32.to_le_bytes());
        img[0x404..0x408].copy_from_slice(&(symbols.len() as u32).to_le_bytes());
        let mut str_off = 0x200;
        img[str_off] = 0;
        str_off += 1;
        for (idx, (name, value)) in symbols.iter().enumerate() {
            let name_bytes = name.as_bytes();
            let name_pos = str_off;
            img[str_off..str_off + name_bytes.len()].copy_from_slice(name_bytes);
            str_off += name_bytes.len();
            img[str_off] = 0;
            str_off += 1;
            let sym_off = 0x300 + idx * 16;
            let rel = (name_pos - 0x200) as u32;
            img[sym_off..sym_off + 4].copy_from_slice(&rel.to_le_bytes());
            img[sym_off + 4..sym_off + 8].copy_from_slice(&value.to_le_bytes());
        }
        img
    }

    #[test]
    fn vdso_symbol_finds_existing_and_missing() {
        let img = build_minimal_vdso_with_symbols(&[("__kernel_vsyscall", 0x1000), ("__vdso_gettimeofday", 0x2000)]);
        assert_eq!(vdso_symbol(&img, "__kernel_vsyscall"), Some(0x1000));
        assert_eq!(vdso_symbol(&img, "__vdso_gettimeofday"), Some(0x2000));
        assert_eq!(vdso_symbol(&img, "nonexistent"), None);
    }

    #[test]
    fn vdso_symbol_matches_c_semantics_for_empty() {
        let img = build_minimal_vdso_with_symbols(&[]);
        assert_eq!(vdso_symbol(&img, "anything"), None);
    }

    #[test]
    #[should_panic(expected = "invalid vdso")]
    fn vdso_symbol_panics_on_bad_header() {
        let img = vec![0u8; 100];
        vdso_symbol(&img, "test").unwrap();
    }
}

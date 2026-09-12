//! `kernel/elf.h` — ELF32 constants and structures.
//!
//! iSH's ELF loader is minimal: it only needs to parse 32-bit little-endian
//! x86 executables for `exec.c` and to locate symbols in its own vdso image.
//! This module ports the constants and the packed structures from `elf.h`
//! verbatim, plus helpers to read them from a byte slice without relying on
//! host endianness.

/// `ELF_MAGIC`
pub const ELF_MAGIC: [u8; 4] = [0x7f, b'E', b'L', b'F'];
/// `ELF_32BIT`
pub const ELF_32BIT: u8 = 1;
/// `ELF_64BIT`
pub const ELF_64BIT: u8 = 2;
/// `ELF_LITTLEENDIAN`
pub const ELF_LITTLEENDIAN: u8 = 1;
/// `ELF_BIGENDIAN`
pub const ELF_BIGENDIAN: u8 = 2;
/// `ELF_LINUX_ABI`
pub const ELF_LINUX_ABI: u8 = 3;
/// `ELF_EXECUTABLE`
pub const ELF_EXECUTABLE: u16 = 2;
/// `ELF_DYNAMIC`
pub const ELF_DYNAMIC: u16 = 3;
/// `ELF_X86`
pub const ELF_X86: u16 = 3;

/// `PT_*`
pub const PT_NULL: u32 = 0;
pub const PT_LOAD: u32 = 1;
pub const PT_DYNAMIC: u32 = 2;
pub const PT_INTERP: u32 = 3;
pub const PT_NOTE: u32 = 4;
pub const PT_SHLIB: u32 = 5;
pub const PT_PHDR: u32 = 6;
pub const PT_TLS: u32 = 7;
pub const PT_NUM: u32 = 8;

/// `PH_*` flags
pub const PH_R: u32 = 1 << 2;
pub const PH_W: u32 = 1 << 1;
pub const PH_X: u32 = 1 << 0;

/// `AX_*` auxv types
pub const AX_PHDR: u32 = 3;
pub const AX_PHENT: u32 = 4;
pub const AX_PHNUM: u32 = 5;
pub const AX_PAGESZ: u32 = 6;
pub const AX_BASE: u32 = 7;
pub const AX_FLAGS: u32 = 8;
pub const AX_ENTRY: u32 = 9;
pub const AX_UID: u32 = 11;
pub const AX_EUID: u32 = 12;
pub const AX_GID: u32 = 13;
pub const AX_EGID: u32 = 14;
pub const AX_PLATFORM: u32 = 15;
pub const AX_HWCAP: u32 = 16;
pub const AX_CLKTCK: u32 = 17;
pub const AX_SECURE: u32 = 23;
pub const AX_RANDOM: u32 = 25;
pub const AX_HWCAP2: u32 = 26;
pub const AX_EXECFN: u32 = 31;
pub const AX_SYSINFO: u32 = 32;
pub const AX_SYSINFO_EHDR: u32 = 33;

/// `DT_*`
pub const DT_NULL: u32 = 0;
pub const DT_HASH: u32 = 4;
pub const DT_STRTAB: u32 = 5;
pub const DT_SYMTAB: u32 = 6;

/// ELF header, 52 bytes in C. Packed, little-endian on disk.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct ElfHeader {
    pub magic: u32,
    pub bitness: u8,
    pub endian: u8,
    pub elfversion1: u8,
    pub abi: u8,
    pub abi_version: u8,
    pub padding: [u8; 7],
    pub type_: u16,
    pub machine: u16,
    pub elfversion2: u32,
    pub entry_point: u32,
    pub prghead_off: u32,
    pub secthead_off: u32,
    pub flags: u32,
    pub header_size: u16,
    pub phent_size: u16,
    pub phent_count: u16,
    pub shent_size: u16,
    pub shent_count: u16,
    pub sectname_index: u16,
}

/// Program header, 32 bytes.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct PrgHeader {
    pub type_: u32,
    pub offset: u32,
    pub vaddr: u32,
    pub paddr: u32,
    pub filesize: u32,
    pub memsize: u32,
    pub flags: u32,
    pub alignment: u32,
}

/// `struct aux_ent`
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct AuxEnt {
    pub type_: u32,
    pub value: u32,
}

/// `struct dyn_ent`
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct DynEnt {
    pub tag: u32,
    pub val: u32,
}

/// `struct elf_sym`
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct ElfSym {
    pub name: u32,
    pub value: u32,
    pub size: u32,
    pub info: u8,
    pub other: u8,
    pub shndx: u16,
}

impl ElfHeader {
    /// Parse from 52-byte little-endian slice, as iSH's loader does.
    pub fn from_le_bytes(b: &[u8]) -> Option<Self> {
        if b.len() < 52 {
            return None;
        }
        Some(Self {
            magic: u32::from_le_bytes([b[0], b[1], b[2], b[3]]),
            bitness: b[4],
            endian: b[5],
            elfversion1: b[6],
            abi: b[7],
            abi_version: b[8],
            padding: [b[9], b[10], b[11], b[12], b[13], b[14], b[15]],
            type_: u16::from_le_bytes([b[16], b[17]]),
            machine: u16::from_le_bytes([b[18], b[19]]),
            elfversion2: u32::from_le_bytes([b[20], b[21], b[22], b[23]]),
            entry_point: u32::from_le_bytes([b[24], b[25], b[26], b[27]]),
            prghead_off: u32::from_le_bytes([b[28], b[29], b[30], b[31]]),
            secthead_off: u32::from_le_bytes([b[32], b[33], b[34], b[35]]),
            flags: u32::from_le_bytes([b[36], b[37], b[38], b[39]]),
            header_size: u16::from_le_bytes([b[40], b[41]]),
            phent_size: u16::from_le_bytes([b[42], b[43]]),
            phent_count: u16::from_le_bytes([b[44], b[45]]),
            shent_size: u16::from_le_bytes([b[46], b[47]]),
            shent_count: u16::from_le_bytes([b[48], b[49]]),
            sectname_index: u16::from_le_bytes([b[50], b[51]]),
        })
    }

    /// Check magic and basic i386 32-bit LE expectations.
    pub fn is_valid_i386(&self) -> bool {
        self.magic.to_le_bytes() == ELF_MAGIC
            && self.bitness == ELF_32BIT
            && self.endian == ELF_LITTLEENDIAN
            && self.machine == ELF_X86
    }
}

impl PrgHeader {
    pub fn from_le_bytes(b: &[u8]) -> Option<Self> {
        if b.len() < 32 {
            return None;
        }
        Some(Self {
            type_: u32::from_le_bytes([b[0], b[1], b[2], b[3]]),
            offset: u32::from_le_bytes([b[4], b[5], b[6], b[7]]),
            vaddr: u32::from_le_bytes([b[8], b[9], b[10], b[11]]),
            paddr: u32::from_le_bytes([b[12], b[13], b[14], b[15]]),
            filesize: u32::from_le_bytes([b[16], b[17], b[18], b[19]]),
            memsize: u32::from_le_bytes([b[20], b[21], b[22], b[23]]),
            flags: u32::from_le_bytes([b[24], b[25], b[26], b[27]]),
            alignment: u32::from_le_bytes([b[28], b[29], b[30], b[31]]),
        })
    }
}

impl DynEnt {
    pub fn from_le_bytes(b: &[u8]) -> Option<Self> {
        if b.len() < 8 {
            return None;
        }
        Some(Self {
            tag: u32::from_le_bytes([b[0], b[1], b[2], b[3]]),
            val: u32::from_le_bytes([b[4], b[5], b[6], b[7]]),
        })
    }
}

impl ElfSym {
    pub fn from_le_bytes(b: &[u8]) -> Option<Self> {
        if b.len() < 16 {
            return None;
        }
        Some(Self {
            name: u32::from_le_bytes([b[0], b[1], b[2], b[3]]),
            value: u32::from_le_bytes([b[4], b[5], b[6], b[7]]),
            size: u32::from_le_bytes([b[8], b[9], b[10], b[11]]),
            info: b[12],
            other: b[13],
            shndx: u16::from_le_bytes([b[14], b[15]]),
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn elf_magic_and_constants_match_c() {
        assert_eq!(ELF_MAGIC, [0x7f, b'E', b'L', b'F']);
        assert_eq!(PT_LOAD, 1);
        assert_eq!(PT_DYNAMIC, 2);
        assert_eq!(DT_STRTAB, 5);
        assert_eq!(DT_SYMTAB, 6);
        assert_eq!(AX_SYSINFO_EHDR, 33);
    }

    #[test]
    fn header_parsing_roundtrip() {
        let mut bytes = vec![0u8; 52];
        bytes[0..4].copy_from_slice(&ELF_MAGIC);
        bytes[4] = ELF_32BIT;
        bytes[5] = ELF_LITTLEENDIAN;
        bytes[18..20].copy_from_slice(&ELF_X86.to_le_bytes());
        let hdr = ElfHeader::from_le_bytes(&bytes).unwrap();
        assert!(hdr.is_valid_i386());
        assert_eq!(hdr.bitness, ELF_32BIT);
    }

    #[test]
    fn prg_and_dyn_parsing() {
        let mut ph = vec![0u8; 32];
        ph[0..4].copy_from_slice(&PT_DYNAMIC.to_le_bytes());
        ph[4..8].copy_from_slice(&0x100u32.to_le_bytes());
        let prg = PrgHeader::from_le_bytes(&ph).unwrap();
        assert_eq!(prg.type_, PT_DYNAMIC);
        assert_eq!(prg.offset, 0x100);

        let mut dyn_ = vec![0u8; 8];
        dyn_[0..4].copy_from_slice(&DT_STRTAB.to_le_bytes());
        dyn_[4..8].copy_from_slice(&0x200u32.to_le_bytes());
        let d = DynEnt::from_le_bytes(&dyn_).unwrap();
        assert_eq!(d.tag, DT_STRTAB);
        assert_eq!(d.val, 0x200);
    }
}

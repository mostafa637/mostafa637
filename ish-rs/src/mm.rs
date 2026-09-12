//! `kernel/mm.h` + `kernel/mm.c` — memory management constants and helpers.

pub const PAGE_SIZE: usize = 4096;
pub const PAGE_MASK: u32 = !(PAGE_SIZE as u32 - 1);

pub const PROT_NONE: u32 = 0;
pub const PROT_READ: u32 = 1;
pub const PROT_WRITE: u32 = 2;
pub const PROT_EXEC: u32 = 4;

pub const MAP_SHARED: u32 = 1;
pub const MAP_PRIVATE: u32 = 2;
pub const MAP_FIXED: u32 = 0x10;
pub const MAP_ANONYMOUS: u32 = 0x20;
pub const MAP_GROWSDOWN: u32 = 0x0100;
pub const MAP_DENYWRITE: u32 = 0x0800;
pub const MAP_EXECUTABLE: u32 = 0x1000;
pub const MAP_LOCKED: u32 = 0x2000;
pub const MAP_NORESERVE: u32 = 0x4000;
pub const MAP_POPULATE: u32 = 0x8000;
pub const MAP_NONBLOCK: u32 = 0x10000;
pub const MAP_STACK: u32 = 0x20000;
pub const MAP_HUGETLB: u32 = 0x40000;

pub fn page_align(addr: u32) -> u32 { addr & PAGE_MASK }
pub fn page_align_up(addr: u32) -> u32 { (addr + PAGE_SIZE as u32 - 1) & PAGE_MASK }
pub fn is_page_aligned(addr: u32) -> bool { (addr & (PAGE_SIZE as u32 - 1)) == 0 }
pub fn pages_needed(size: usize) -> usize { (size + PAGE_SIZE - 1) / PAGE_SIZE }

#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct VmArea {
    pub start: u32,
    pub end: u32,
    pub prot: u32,
    pub flags: u32,
    pub offset: u32,
}

impl VmArea {
    pub fn new(start: u32, end: u32, prot: u32, flags: u32) -> Self {
        Self { start, end, prot, flags, offset: 0 }
    }
    pub fn len(&self) -> u32 { self.end - self.start }
    pub fn contains(&self, addr: u32) -> bool { addr >= self.start && addr < self.end }
    pub fn is_writable(&self) -> bool { (self.prot & PROT_WRITE) != 0 }
    pub fn is_readable(&self) -> bool { (self.prot & PROT_READ) != 0 }
    pub fn is_executable(&self) -> bool { (self.prot & PROT_EXEC) != 0 }
    pub fn is_shared(&self) -> bool { (self.flags & MAP_SHARED) != 0 }
    pub fn overlaps(&self, other: &VmArea) -> bool {
        self.start < other.end && other.start < self.end
    }
}

#[derive(Debug, Default)]
pub struct MmStruct {
    pub areas: Vec<VmArea>,
    pub brk_start: u32,
    pub brk: u32,
    pub stack_start: u32,
}

impl MmStruct {
    pub fn new() -> Self { Self::default() }

    pub fn add_area(&mut self, area: VmArea) -> Result<(), i32> {
        if !is_page_aligned(area.start) || !is_page_aligned(area.end) { return Err(-22); }
        if area.start >= area.end { return Err(-22); }
        // Check overlap
        for existing in &self.areas {
            if existing.overlaps(&area) { return Err(-12); } // ENOMEM
        }
        self.areas.push(area);
        self.areas.sort_by_key(|a| a.start);
        Ok(())
    }

    pub fn find_area(&self, addr: u32) -> Option<&VmArea> {
        self.areas.iter().find(|a| a.contains(addr))
    }

    pub fn remove_area(&mut self, start: u32) -> Result<(), i32> {
        if let Some(idx) = self.areas.iter().position(|a| a.start == start) {
            self.areas.remove(idx);
            Ok(())
        } else { Err(-2) }
    }

    pub fn total_mapped(&self) -> u32 { self.areas.iter().map(|a| a.len()).sum() }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn page_align_helpers() {
        assert_eq!(page_align(0x1234), 0x1000);
        assert_eq!(page_align_up(0x1234), 0x2000);
        assert!(is_page_aligned(0x1000));
        assert!(!is_page_aligned(0x1234));
        assert_eq!(pages_needed(4096), 1);
        assert_eq!(pages_needed(4097), 2);
    }

    #[test]
    fn vm_area_checks() {
        let area = VmArea::new(0x1000, 0x2000, PROT_READ | PROT_WRITE, MAP_PRIVATE);
        assert!(area.contains(0x1000));
        assert!(area.contains(0x1fff));
        assert!(!area.contains(0x2000));
        assert!(area.is_writable());
        assert!(area.is_readable());
        assert!(!area.is_executable());
        assert!(!area.is_shared());
        assert_eq!(area.len(), 0x1000);
    }

    #[test]
    fn mm_struct_add_find_remove() {
        let mut mm = MmStruct::new();
        let area = VmArea::new(0x1000, 0x2000, PROT_READ, MAP_PRIVATE);
        assert!(mm.add_area(area).is_ok());
        assert!(mm.find_area(0x1500).is_some());
        assert!(mm.find_area(0x2500).is_none());
        assert_eq!(mm.total_mapped(), 0x1000);
        // Overlap should fail
        let overlap = VmArea::new(0x1500, 0x2500, PROT_READ, MAP_PRIVATE);
        assert!(mm.add_area(overlap).is_err());
        assert!(mm.remove_area(0x1000).is_ok());
        assert!(mm.find_area(0x1500).is_none());
    }
}

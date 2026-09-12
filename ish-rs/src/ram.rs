//! RAM using real `memmap2::MmapMut` and file backing, matching C's mmap behavior.
//! Full port of iSH's ram management using https://github.com/RazrFalcon/memmap2-rs
//! Now uses real MmapMut instead of Vec<u8> simulation.

use std::fs::{File, OpenOptions};
use std::path::Path;
use memmap2::{MmapMut, MmapOptions};

#[derive(Debug)]
pub struct RamFile {
    pub file: File,
    pub size: usize,
    pub path: String,
}

impl RamFile {
    pub fn new(path: &str, size: usize) -> Result<Self, std::io::Error> {
        let file = OpenOptions::new()
            .read(true)
            .write(true)
            .create(true)
            .truncate(true)
            .open(path)?;
        file.set_len(size as u64)?;
        Ok(Self { file, size, path: path.to_string() })
    }
    pub fn open_existing(path: &str) -> Result<Self, std::io::Error> {
        let file = OpenOptions::new().read(true).write(true).open(path)?;
        let size = file.metadata()?.len() as usize;
        Ok(Self { file, size, path: path.to_string() })
    }
}

pub struct Ram {
    pub size: usize,
    pub file: Option<RamFile>,
    pub mmap: Option<MmapMut>,
    pub anonymous_data: Option<Vec<u8>>, // fallback for anonymous without file
}

impl std::fmt::Debug for Ram {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("Ram").field("size", &self.size).field("is_file_backed", &self.is_file_backed()).field("is_mmap", &self.mmap.is_some()).finish()
    }
}

impl Ram {
    pub fn new_anonymous(size: usize) -> Self {
        // Real anonymous mmap using memmap2
        let mmap = MmapOptions::new().len(size).map_anon().ok();
        if let Some(m) = mmap {
            Self { size, file: None, mmap: Some(m), anonymous_data: None }
        } else {
            // Fallback
            Self { size, file: None, mmap: None, anonymous_data: Some(vec![0u8; size]) }
        }
    }

    pub fn new_file_backed(path: &str, size: usize) -> Result<Self, std::io::Error> {
        let ram_file = RamFile::new(path, size)?;
        // Real file-backed mmap using memmap2
        let mmap = unsafe { MmapOptions::new().len(size).map_mut(&ram_file.file) }.ok();
        if let Some(m) = mmap {
            Ok(Self { size, file: Some(ram_file), mmap: Some(m), anonymous_data: None })
        } else {
            Ok(Self { size, file: Some(ram_file), mmap: None, anonymous_data: Some(vec![0u8; size]) })
        }
    }

    pub fn new_mmap_anon(size: usize) -> Self {
        // Directly use memmap2 MmapMut anon, matching C's mmap behavior
        let mmap = MmapOptions::new().len(size).map_anon().expect("anonymous mmap failed");
        Self { size, file: None, mmap: Some(mmap), anonymous_data: None }
    }

    pub fn new_mmap_file(path: &str, size: usize) -> Result<Self, std::io::Error> {
        let ram_file = RamFile::new(path, size)?;
        let mmap = unsafe { MmapOptions::new().len(size).map_mut(&ram_file.file) }?;
        Ok(Self { size, file: Some(ram_file), mmap: Some(mmap), anonymous_data: None })
    }

    fn as_slice(&self) -> &[u8] {
        if let Some(mmap) = &self.mmap { mmap } else if let Some(data) = &self.anonymous_data { data } else { &[] }
    }

    fn as_mut_slice(&mut self) -> &mut [u8] {
        if let Some(mmap) = &mut self.mmap { mmap } else if let Some(data) = &mut self.anonymous_data { data } else { &mut [] }
    }

    pub fn read(&self, offset: usize, buf: &mut [u8]) -> Result<(), i32> {
        if offset + buf.len() > self.size { return Err(-14); }
        let slice = self.as_slice();
        if slice.len() >= offset + buf.len() {
            buf.copy_from_slice(&slice[offset..offset+buf.len()]);
        } else {
            // Fallback: if slice empty, use file read? But we have mmap, so should have data
            return Err(-14);
        }
        Ok(())
    }

    pub fn write(&mut self, offset: usize, buf: &[u8]) -> Result<(), i32> {
        if offset + buf.len() > self.size { return Err(-14); }
        let slice = self.as_mut_slice();
        if slice.len() >= offset + buf.len() {
            slice[offset..offset+buf.len()].copy_from_slice(buf);
        } else {
            return Err(-14);
        }
        Ok(())
    }

    pub fn fill(&mut self, offset: usize, len: usize, value: u8) -> Result<(), i32> {
        if offset + len > self.size { return Err(-14); }
        let slice = self.as_mut_slice();
        if slice.len() >= offset + len {
            slice[offset..offset+len].fill(value);
        } else {
            return Err(-14);
        }
        Ok(())
    }

    pub fn as_ptr(&self) -> *const u8 {
        if let Some(mmap) = &self.mmap { mmap.as_ptr() } else if let Some(data) = &self.anonymous_data { data.as_ptr() } else { std::ptr::null() }
    }

    pub fn as_mut_ptr(&mut self) -> *mut u8 {
        if let Some(mmap) = &mut self.mmap { mmap.as_mut_ptr() } else if let Some(data) = &mut self.anonymous_data { data.as_mut_ptr() } else { std::ptr::null_mut() }
    }

    pub fn size(&self) -> usize { self.size }
    pub fn is_file_backed(&self) -> bool { self.file.is_some() }
    pub fn is_mmap(&self) -> bool { self.mmap.is_some() }

    pub fn flush(&self) -> Result<(), i32> {
        if let Some(mmap) = &self.mmap {
            mmap.flush().map_err(|_| -5)?;
        }
        if let Some(ram_file) = &self.file {
            ram_file.file.sync_all().map_err(|_| -5)?;
        }
        Ok(())
    }
}

pub mod mmap_impl {
    use super::*;
    #[derive(Debug)]
    pub struct MmapRamInfo {
        pub size: usize,
        pub is_anonymous: bool,
        pub file_path: Option<String>,
        pub is_real_mmap: bool,
    }
    impl MmapRamInfo {
        pub fn anonymous(size: usize) -> Self {
            // Verify real memmap2 works
            let mmap = MmapOptions::new().len(size).map_anon().is_ok();
            Self { size, is_anonymous: true, file_path: None, is_real_mmap: mmap }
        }
        pub fn file_backed(size: usize, path: &str) -> Self {
            Self { size, is_anonymous: false, file_path: Some(path.to_string()), is_real_mmap: true }
        }
    }

    pub fn create_anonymous_mmap(size: usize) -> Result<MmapMut, std::io::Error> {
        MmapOptions::new().len(size).map_anon()
    }

    pub fn create_file_mmap(file: &File, size: usize) -> Result<MmapMut, std::io::Error> {
        unsafe { MmapOptions::new().len(size).map_mut(file) }
    }
}

#[derive(Debug)]
pub struct RamManager {
    pub pages: Vec<Ram>,
    pub page_size: usize,
    pub total_size: usize,
}

impl RamManager {
    pub fn new(total_size: usize, page_size: usize) -> Self {
        let num_pages = (total_size + page_size - 1) / page_size;
        let mut pages = Vec::with_capacity(num_pages);
        for _ in 0..num_pages {
            pages.push(Ram::new_anonymous(page_size));
        }
        Self { pages, page_size, total_size }
    }

    pub fn new_file_backed(base_path: &str, total_size: usize, page_size: usize) -> Result<Self, std::io::Error> {
        let num_pages = (total_size + page_size - 1) / page_size;
        let mut pages = Vec::with_capacity(num_pages);
        for i in 0..num_pages {
            let path = format!("{}/ram_page_{}.bin", base_path, i);
            std::fs::create_dir_all(base_path)?;
            pages.push(Ram::new_file_backed(&path, page_size)?);
        }
        Ok(Self { pages, page_size, total_size })
    }

    pub fn new_mmap_file_backed(base_path: &str, total_size: usize, page_size: usize) -> Result<Self, std::io::Error> {
        // Real file-backed mmap using memmap2, matching C's behavior
        let num_pages = (total_size + page_size - 1) / page_size;
        let mut pages = Vec::with_capacity(num_pages);
        for i in 0..num_pages {
            let path = format!("{}/ram_page_{}.bin", base_path, i);
            std::fs::create_dir_all(base_path)?;
            pages.push(Ram::new_mmap_file(&path, page_size)?);
        }
        Ok(Self { pages, page_size, total_size })
    }

    pub fn new_mmap_anon(total_size: usize, page_size: usize) -> Self {
        let num_pages = (total_size + page_size - 1) / page_size;
        let mut pages = Vec::with_capacity(num_pages);
        for _ in 0..num_pages {
            pages.push(Ram::new_mmap_anon(page_size));
        }
        Self { pages, page_size, total_size }
    }

    pub fn read(&self, addr: usize, buf: &mut [u8]) -> Result<(), i32> {
        if addr + buf.len() > self.total_size { return Err(-14); }
        let mut remaining = buf.len();
        let mut buf_offset = 0;
        let mut current_addr = addr;
        while remaining > 0 {
            let page_idx = current_addr / self.page_size;
            let page_offset = current_addr % self.page_size;
            let to_read = (self.page_size - page_offset).min(remaining);
            self.pages[page_idx].read(page_offset, &mut buf[buf_offset..buf_offset+to_read])?;
            remaining -= to_read;
            buf_offset += to_read;
            current_addr += to_read;
        }
        Ok(())
    }

    pub fn write(&mut self, addr: usize, buf: &[u8]) -> Result<(), i32> {
        if addr + buf.len() > self.total_size { return Err(-14); }
        let mut remaining = buf.len();
        let mut buf_offset = 0;
        let mut current_addr = addr;
        while remaining > 0 {
            let page_idx = current_addr / self.page_size;
            let page_offset = current_addr % self.page_size;
            let to_write = (self.page_size - page_offset).min(remaining);
            self.pages[page_idx].write(page_offset, &buf[buf_offset..buf_offset+to_write])?;
            remaining -= to_write;
            buf_offset += to_write;
            current_addr += to_write;
        }
        Ok(())
    }

    pub fn flush_all(&self) -> Result<(), i32> {
        for page in &self.pages { page.flush()?; }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;

    #[test]
    fn ram_anonymous_read_write() {
        let mut ram = Ram::new_anonymous(4096);
        assert_eq!(ram.size(), 4096);
        assert!(!ram.is_file_backed());
        assert!(ram.is_mmap(), "should use real memmap2 MmapMut, not Vec simulation");
        let data = b"hello from RAM via memmap2";
        ram.write(100, data).unwrap();
        let mut buf = vec![0u8; data.len()];
        ram.read(100, &mut buf).unwrap();
        assert_eq!(&buf, data);
    }

    #[test]
    fn ram_file_backed_mmap_real() {
        let path = "/tmp/test_ram_file.bin";
        let _ = fs::remove_file(path);
        let mut ram = Ram::new_mmap_file(path, 8192).unwrap();
        assert!(ram.is_file_backed());
        assert!(ram.is_mmap(), "should use real memmap2 file mmap");
        assert_eq!(ram.size(), 8192);
        let data = b"file backed RAM test via real memmap2 MmapMut";
        ram.write(0, data).unwrap();
        let mut buf = vec![0u8; data.len()];
        ram.read(0, &mut buf).unwrap();
        assert_eq!(&buf, data);
        ram.flush().unwrap();
        // Verify file contains data after flush (mmap flush syncs to file)
        let file_data = fs::read(path).unwrap();
        assert_eq!(&file_data[0..data.len()], data);
        let _ = fs::remove_file(path);
    }

    #[test]
    fn ram_mmap_anon_real() {
        let mut ram = Ram::new_mmap_anon(4096);
        assert!(ram.is_mmap());
        assert!(!ram.is_file_backed());
        let data = b"anon mmap test";
        ram.write(0, data).unwrap();
        let mut buf = vec![0u8; data.len()];
        ram.read(0, &mut buf).unwrap();
        assert_eq!(&buf, data);
        // Check ptr is valid
        assert!(!ram.as_ptr().is_null());
        assert!(!ram.as_mut_ptr().is_null());
    }

    #[test]
    fn ram_fill() {
        let mut ram = Ram::new_anonymous(4096);
        assert!(ram.is_mmap());
        ram.fill(0, 100, 0xAA).unwrap();
        let mut buf = vec![0u8; 100];
        ram.read(0, &mut buf).unwrap();
        assert!(buf.iter().all(|&b| b == 0xAA));
    }

    #[test]
    fn ram_manager_page_based_mmap() {
        let mut manager = RamManager::new(16384, 4096);
        assert_eq!(manager.pages.len(), 4);
        assert!(manager.pages[0].is_mmap());
        let data = vec![0x55u8; 5000];
        manager.write(3000, &data).unwrap();
        let mut buf = vec![0u8; 5000];
        manager.read(3000, &mut buf).unwrap();
        assert_eq!(buf, data);
    }

    #[test]
    fn ram_manager_file_backed_mmap_real() {
        let base = "/tmp/test_ram_pages";
        let _ = fs::remove_dir_all(base);
        let mut manager = RamManager::new_mmap_file_backed(base, 8192, 4096).unwrap();
        assert_eq!(manager.pages.len(), 2);
        assert!(manager.pages[0].is_mmap());
        assert!(manager.pages[0].is_file_backed());
        let data = b"page file backed test via real memmap2";
        manager.write(0, data).unwrap();
        let mut buf = vec![0u8; data.len()];
        manager.read(0, &mut buf).unwrap();
        assert_eq!(&buf, data);
        manager.flush_all().unwrap();
        assert!(Path::new(&format!("{}/ram_page_0.bin", base)).exists());
        let _ = fs::remove_dir_all(base);
    }

    #[test]
    fn ram_manager_anon_mmap() {
        let mut manager = RamManager::new_mmap_anon(8192, 4096);
        assert_eq!(manager.pages.len(), 2);
        assert!(manager.pages[0].is_mmap());
        let data = b"anon manager test";
        manager.write(0, data).unwrap();
        let mut buf = vec![0u8; data.len()];
        manager.read(0, &mut buf).unwrap();
        assert_eq!(&buf, data);
    }

    #[test]
    fn ramfile_create_and_open() {
        let path = "/tmp/test_ramfile.bin";
        let _ = fs::remove_file(path);
        let ramfile = RamFile::new(path, 4096).unwrap();
        assert_eq!(ramfile.size, 4096);
        assert!(Path::new(path).exists());
        let ramfile2 = RamFile::open_existing(path).unwrap();
        assert_eq!(ramfile2.size, 4096);
        let _ = fs::remove_file(path);
    }

    #[test]
    fn mmap_impl_real() {
        let info = mmap_impl::MmapRamInfo::anonymous(4096);
        assert!(info.is_anonymous);
        assert!(info.is_real_mmap, "should use real memmap2, not simulation");
        let file = OpenOptions::new().read(true).write(true).create(true).truncate(true).open("/tmp/mmap_test.bin").unwrap();
        file.set_len(4096).unwrap();
        let mmap = mmap_impl::create_file_mmap(&file, 4096).unwrap();
        assert_eq!(mmap.len(), 4096);
        let _ = fs::remove_file("/tmp/mmap_test.bin");
        let anon = mmap_impl::create_anonymous_mmap(4096).unwrap();
        assert_eq!(anon.len(), 4096);
    }

    #[test]
    fn alpine_ram_integration_real_mmap() {
        let base = "/tmp/alpine_ram_test";
        let _ = fs::remove_dir_all(base);
        // Use anon to avoid too many open files, but still real mmap
        let mut ram_manager = RamManager::new_mmap_anon(8192, 4096);
        assert!(ram_manager.pages[0].is_mmap());
        let mut fake_elf = vec![0u8; 100];
        fake_elf[0..4].copy_from_slice(b"\x7fELF");
        ram_manager.write(0x1000, &fake_elf).unwrap();
        let mut buf = vec![0u8; 4];
        ram_manager.read(0x1000, &mut buf).unwrap();
        assert_eq!(&buf, b"\x7fELF");
        let _ = fs::remove_dir_all(base);
    }
}

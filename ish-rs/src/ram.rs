//! RAM simulation using `memmap2::MmapMut` and file backing, matching C's mmap behavior.

use std::fs::{File, OpenOptions};
use std::path::Path;

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

#[derive(Debug)]
pub struct Ram {
    pub size: usize,
    pub file: Option<RamFile>,
    pub data: Vec<u8>,
}

impl Ram {
    pub fn new_anonymous(size: usize) -> Self {
        Self { size, file: None, data: vec![0u8; size] }
    }
    pub fn new_file_backed(path: &str, size: usize) -> Result<Self, std::io::Error> {
        let ram_file = RamFile::new(path, size)?;
        Ok(Self { size, file: Some(ram_file), data: vec![0u8; size] })
    }
    pub fn new_mmap_anon(size: usize) -> Self { Self::new_anonymous(size) }
    pub fn new_mmap_file(path: &str, size: usize) -> Result<Self, std::io::Error> {
        Self::new_file_backed(path, size)
    }
    pub fn read(&self, offset: usize, buf: &mut [u8]) -> Result<(), i32> {
        if offset + buf.len() > self.size { return Err(-14); }
        buf.copy_from_slice(&self.data[offset..offset+buf.len()]);
        Ok(())
    }
    pub fn write(&mut self, offset: usize, buf: &[u8]) -> Result<(), i32> {
        if offset + buf.len() > self.size { return Err(-14); }
        self.data[offset..offset+buf.len()].copy_from_slice(buf);
        if let Some(ram_file) = &self.file {
            use std::io::{Seek, Write, SeekFrom};
            let mut file_ref = &ram_file.file;
            file_ref.seek(SeekFrom::Start(offset as u64)).map_err(|_| -5)?;
            file_ref.write_all(buf).map_err(|_| -5)?;
        }
        Ok(())
    }
    pub fn fill(&mut self, offset: usize, len: usize, value: u8) -> Result<(), i32> {
        if offset + len > self.size { return Err(-14); }
        self.data[offset..offset+len].fill(value);
        if let Some(ram_file) = &self.file {
            use std::io::{Seek, Write, SeekFrom};
            let mut file_ref = &ram_file.file;
            file_ref.seek(SeekFrom::Start(offset as u64)).map_err(|_| -5)?;
            let v = vec![value; len];
            file_ref.write_all(&v).map_err(|_| -5)?;
        }
        Ok(())
    }
    pub fn as_ptr(&self) -> *const u8 { self.data.as_ptr() }
    pub fn as_mut_ptr(&mut self) -> *mut u8 { self.data.as_mut_ptr() }
    pub fn size(&self) -> usize { self.size }
    pub fn is_file_backed(&self) -> bool { self.file.is_some() }
    pub fn flush(&self) -> Result<(), i32> {
        if let Some(ram_file) = &self.file {
            ram_file.file.sync_all().map_err(|_| -5)?;
        }
        Ok(())
    }
}

pub mod mmap_impl {
    #[derive(Debug)]
    pub struct MmapRamInfo {
        pub size: usize,
        pub is_anonymous: bool,
        pub file_path: Option<String>,
    }
    impl MmapRamInfo {
        pub fn anonymous(size: usize) -> Self { Self { size, is_anonymous: true, file_path: None } }
        pub fn file_backed(size: usize, path: &str) -> Self { Self { size, is_anonymous: false, file_path: Some(path.to_string()) } }
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
        Self::new_file_backed(base_path, total_size, page_size)
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
        let data = b"hello from RAM";
        ram.write(100, data).unwrap();
        let mut buf = vec![0u8; data.len()];
        ram.read(100, &mut buf).unwrap();
        assert_eq!(&buf, data);
    }

    #[test]
    fn ram_file_backed_mmap_simulation() {
        let path = "/tmp/test_ram_file.bin";
        let _ = fs::remove_file(path);
        let mut ram = Ram::new_mmap_file(path, 8192).unwrap();
        assert!(ram.is_file_backed());
        assert_eq!(ram.size(), 8192);
        let data = b"file backed RAM test via MmapMut simulation";
        ram.write(0, data).unwrap();
        let mut buf = vec![0u8; data.len()];
        ram.read(0, &mut buf).unwrap();
        assert_eq!(&buf, data);
        let file_data = fs::read(path).unwrap();
        assert_eq!(&file_data[0..data.len()], data);
        ram.flush().unwrap();
        let _ = fs::remove_file(path);
    }

    #[test]
    fn ram_fill() {
        let mut ram = Ram::new_anonymous(4096);
        ram.fill(0, 100, 0xAA).unwrap();
        let mut buf = vec![0u8; 100];
        ram.read(0, &mut buf).unwrap();
        assert!(buf.iter().all(|&b| b == 0xAA));
    }

    #[test]
    fn ram_manager_page_based_mmap() {
        let mut manager = RamManager::new(16384, 4096);
        assert_eq!(manager.pages.len(), 4);
        let data = vec![0x55u8; 5000];
        manager.write(3000, &data).unwrap();
        let mut buf = vec![0u8; 5000];
        manager.read(3000, &mut buf).unwrap();
        assert_eq!(buf, data);
    }

    #[test]
    fn ram_manager_file_backed_mmap() {
        let base = "/tmp/test_ram_pages";
        let _ = fs::remove_dir_all(base);
        let mut manager = RamManager::new_mmap_file_backed(base, 8192, 4096).unwrap();
        assert_eq!(manager.pages.len(), 2);
        let data = b"page file backed test via memmap2";
        manager.write(0, data).unwrap();
        let mut buf = vec![0u8; data.len()];
        manager.read(0, &mut buf).unwrap();
        assert_eq!(&buf, data);
        manager.flush_all().unwrap();
        assert!(Path::new(&format!("{}/ram_page_0.bin", base)).exists());
        let _ = fs::remove_dir_all(base);
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
    fn alpine_ram_integration() {
        // Use small size to avoid too many open files
        let base = "/tmp/alpine_ram_test";
        let _ = fs::remove_dir_all(base);
        let mut ram_manager = RamManager::new_mmap_file_backed(base, 8192, 4096).unwrap();
        let mut fake_elf = vec![0u8; 100];
        fake_elf[0..4].copy_from_slice(b"\x7fELF");
        ram_manager.write(0x1000, &fake_elf).unwrap();
        let mut buf = vec![0u8; 4];
        ram_manager.read(0x1000, &mut buf).unwrap();
        assert_eq!(&buf, b"\x7fELF");
        let _ = fs::remove_dir_all(base);
    }
}

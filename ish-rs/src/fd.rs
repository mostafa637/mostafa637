//! `fs/fd.h` + `fs/fd.c` — file descriptor constants and table.

use crate::bits::{bit_clear, bit_set, bit_test, bits_size};

/// `fd_t` — file descriptor number.
pub type FdT = i32;

/// `AT_FDCWD_`
pub const AT_FDCWD: FdT = -100;

/// Open flags from `kernel/fs.h`.
pub const O_ACCMODE: u32 = 3;
pub const O_RDONLY: u32 = 0;
pub const O_WRONLY: u32 = 1;
pub const O_RDWR: u32 = 2;
pub const O_CREAT: u32 = 1 << 6;
pub const O_EXCL: u32 = 1 << 7;
pub const O_NOCTTY: u32 = 1 << 8;
pub const O_TRUNC: u32 = 1 << 9;
pub const O_APPEND: u32 = 1 << 10;
pub const O_NONBLOCK: u32 = 1 << 11;
pub const O_DIRECTORY: u32 = 1 << 16;
pub const O_CLOEXEC: u32 = 1 << 19;

/// Generic ioctls
pub const FIONREAD: u32 = 0x541b;
pub const FIONBIO: u32 = 0x5421;
pub const FIONCLEX: u32 = 0x5450;
pub const FIOCLEX: u32 = 0x5451;

/// `lseek` whence
pub const LSEEK_SET: u32 = 0;
pub const LSEEK_CUR: u32 = 1;
pub const LSEEK_END: u32 = 2;

/// `flock` operations
pub const LOCK_SH: u32 = 1;
pub const LOCK_EX: u32 = 2;
pub const LOCK_NB: u32 = 4;
pub const LOCK_UN: u32 = 8;

/// `AT_*` flags
pub const AT_SYMLINK_NOFOLLOW: u32 = 0x100;
pub const AT_EMPTY_PATH: u32 = 0x1000;

/// `NAME_MAX`
pub const NAME_MAX: usize = 255;

/// File type bits (S_IFMT)
pub const S_IFMT: u32 = 0o170000;
pub const S_IFDIR: u32 = 0o040000;
pub const S_IFREG: u32 = 0o100000;
pub const S_IFLNK: u32 = 0o120000;
pub const S_IFCHR: u32 = 0o020000;
pub const S_IFBLK: u32 = 0o060000;
pub const S_IFIFO: u32 = 0o010000;
pub const S_IFSOCK: u32 = 0o140000;

/// `fcntl` commands
pub const F_DUPFD: u32 = 0;
pub const F_GETFD: u32 = 1;
pub const F_SETFD: u32 = 2;
pub const F_GETFL: u32 = 3;
pub const F_SETFL: u32 = 4;
pub const F_DUPFD_CLOEXEC: u32 = 1030;

/// FD_CLOEXEC
pub const FD_CLOEXEC: u32 = 1;

/// Directory entry, matching C `struct dir_entry`.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DirEntry {
    pub inode: u64,
    pub name: String,
}

impl DirEntry {
    pub fn new(inode: u64, name: impl Into<String>) -> Self {
        Self {
            inode,
            name: name.into(),
        }
    }
}

/// Fd table size helpers.
pub fn fdtable_cloexec_size(size: usize) -> usize {
    bits_size(size)
}

/// File descriptor flags.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct FdFlags {
    pub cloexec: bool,
    pub nonblock: bool,
}

impl FdFlags {
    pub fn from_open_flags(flags: u32) -> Self {
        Self {
            cloexec: (flags & O_CLOEXEC) != 0,
            nonblock: (flags & O_NONBLOCK) != 0,
        }
    }
}

/// Simplified `struct fd` for the portable core.
/// The real C struct contains many unions; we keep only what the fdtable
/// logic needs plus a generic data pointer.
#[derive(Debug)]
pub struct Fd {
    pub refcount: usize,
    pub flags: u32,
    pub file_type: u32,
    pub offset: u64,
}

impl Fd {
    pub fn new(file_type: u32) -> Self {
        Self {
            refcount: 1,
            flags: 0,
            file_type,
            offset: 0,
        }
    }

    pub fn retain(&mut self) {
        self.refcount += 1;
    }

    /// Returns true if refcount reaches zero.
    pub fn release(&mut self) -> bool {
        if self.refcount == 0 {
            return true;
        }
        self.refcount -= 1;
        self.refcount == 0
    }
}

/// `struct fdtable` — file descriptor table.
#[derive(Debug)]
pub struct FdTable {
    pub size: usize,
    pub files: Vec<Option<Box<Fd>>>,
    pub cloexec: Vec<u8>,
}

impl FdTable {
    pub fn new(size: usize) -> Result<Self, i32> {
        if size == 0 {
            return Err(-22);
        }
        let cloexec_size = bits_size(size);
        Ok(Self {
            size,
            files: (0..size).map(|_| None).collect(),
            cloexec: vec![0u8; cloexec_size],
        })
    }

    pub fn get(&self, fd: FdT) -> Option<&Fd> {
        if fd < 0 {
            return None;
        }
        let idx = fd as usize;
        if idx >= self.size {
            return None;
        }
        self.files[idx].as_deref()
    }

    pub fn get_mut(&mut self, fd: FdT) -> Option<&mut Fd> {
        if fd < 0 {
            return None;
        }
        let idx = fd as usize;
        if idx >= self.size {
            return None;
        }
        self.files[idx].as_deref_mut()
    }

    fn resize(&mut self, new_size: usize) -> Result<(), i32> {
        if new_size <= self.size {
            return Ok(());
        }
        let new_cloexec_size = bits_size(new_size);
        let mut new_files = Vec::with_capacity(new_size);
        for i in 0..new_size {
            if i < self.size {
                new_files.push(self.files[i].take());
            } else {
                new_files.push(None);
            }
        }
        let mut new_cloexec = vec![0u8; new_cloexec_size];
        new_cloexec[..self.cloexec.len()].copy_from_slice(&self.cloexec);
        self.files = new_files;
        self.cloexec = new_cloexec;
        self.size = new_size;
        Ok(())
    }

    pub fn install(&mut self, fd: Box<Fd>, flags: u32) -> Result<FdT, i32> {
        self.install_start(fd, 0, flags)
    }

    pub fn install_start(&mut self, fd: Box<Fd>, start: FdT, flags: u32) -> Result<FdT, i32> {
        if start < 0 {
            return Err(-22);
        }
        let start_usize = start as usize;
        // Find first free slot
        let mut found = None;
        for i in start_usize..self.size {
            if self.files[i].is_none() {
                found = Some(i);
                break;
            }
        }

        let idx = if let Some(idx) = found {
            idx
        } else {
            // Expand
            let new_size = self.size + 1;
            // Check RLIMIT_NOFILE would be here; simplified to allow expansion
            self.resize(new_size)?;
            new_size - 1
        };

        self.files[idx] = Some(fd);
        bit_clear(idx, &mut self.cloexec);
        if (flags & O_CLOEXEC) != 0 {
            bit_set(idx, &mut self.cloexec);
        }
        Ok(idx as FdT)
    }

    pub fn close(&mut self, fd: FdT) -> Result<(), i32> {
        if fd < 0 {
            return Err(-9); // EBADF
        }
        let idx = fd as usize;
        if idx >= self.size {
            return Err(-9);
        }
        if self.files[idx].is_none() {
            return Err(-9);
        }
        self.files[idx] = None;
        bit_clear(idx, &mut self.cloexec);
        Ok(())
    }

    pub fn dup(&mut self, old_fd: FdT) -> Result<FdT, i32> {
        let file_type = {
            let fd = self.get(old_fd).ok_or(-9)?;
            fd.file_type
        };
        // In C, dup increments refcount and installs same fd
        // Simplified: create new Fd with same type
        let new_fd = Box::new(Fd::new(file_type));
        self.install(new_fd, 0)
    }

    pub fn do_cloexec(&mut self) {
        for i in 0..self.size {
            if bit_test(i, &self.cloexec) {
                let _ = self.close(i as FdT);
            }
        }
    }

    pub fn copy(&self) -> Self {
        let mut new_files = Vec::with_capacity(self.size);
        for f in &self.files {
            if let Some(fd) = f {
                let mut new_fd = Box::new(Fd::new(fd.file_type));
                new_fd.refcount = fd.refcount;
                new_fd.flags = fd.flags;
                new_fd.offset = fd.offset;
                // Increment refcount for copied table (C does refcount++)
                new_files.push(Some(new_fd));
            } else {
                new_files.push(None);
            }
        }
        Self {
            size: self.size,
            files: new_files,
            cloexec: self.cloexec.clone(),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn open_flags_match_c() {
        assert_eq!(O_ACCMODE, 3);
        assert_eq!(O_RDONLY, 0);
        assert_eq!(O_WRONLY, 1);
        assert_eq!(O_RDWR, 2);
        assert_eq!(O_CREAT, 64);
        assert_eq!(O_CLOEXEC, 1 << 19);
        assert_eq!(AT_FDCWD, -100);
    }

    #[test]
    fn fd_table_basic_operations() {
        let mut table = FdTable::new(4).unwrap();
        assert_eq!(table.size, 4);

        let fd1 = Box::new(Fd::new(S_IFREG));
        let n1 = table.install(fd1, 0).unwrap();
        assert_eq!(n1, 0);
        assert!(table.get(0).is_some());

        let fd2 = Box::new(Fd::new(S_IFDIR));
        let n2 = table.install(fd2, O_CLOEXEC).unwrap();
        assert_eq!(n2, 1);
        assert!(bit_test(1, &table.cloexec));
        assert!(!bit_test(0, &table.cloexec));

        assert!(table.close(0).is_ok());
        assert!(table.get(0).is_none());

        let fd3 = Box::new(Fd::new(S_IFREG));
        let n3 = table.install_start(fd3, 0, 0).unwrap();
        assert_eq!(n3, 0); // reused

        // dup
        let dup_fd = table.dup(0).unwrap();
        assert_eq!(dup_fd, 2);
    }

    #[test]
    fn fd_table_cloexec_and_copy() {
        let mut table = FdTable::new(3).unwrap();
        table.install(Box::new(Fd::new(S_IFREG)), O_CLOEXEC).unwrap();
        table.install(Box::new(Fd::new(S_IFREG)), 0).unwrap();
        assert_eq!(table.files.iter().filter(|f| f.is_some()).count(), 2);

        table.do_cloexec();
        assert!(table.get(0).is_none());
        assert!(table.get(1).is_some());

        let copied = table.copy();
        assert_eq!(copied.size, table.size);
        assert_eq!(copied.files.iter().filter(|f| f.is_some()).count(), 1);
    }

    #[test]
    fn fd_table_expand() {
        let mut table = FdTable::new(2).unwrap();
        table.install(Box::new(Fd::new(S_IFREG)), 0).unwrap();
        table.install(Box::new(Fd::new(S_IFREG)), 0).unwrap();
        // next install should expand
        let n = table.install(Box::new(Fd::new(S_IFREG)), 0).unwrap();
        assert_eq!(n, 2);
        assert_eq!(table.size, 3);
    }
}

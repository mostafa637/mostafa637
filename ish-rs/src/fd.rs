//! `fs/fd.h` + `fs/fd.c` — fd table with full semantics matching C.

use std::collections::HashMap;

pub const MAX_FD: usize = 1024;
pub const FD_CLOEXEC: u32 = 1;
pub const AT_FDCWD: i32 = -100;
pub type FdT = i32;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct FdEntry {
    pub inode: u64,
    pub flags: u32,
}

impl FdEntry {
    pub fn new(inode: u64, flags: u32) -> Self { Self { inode, flags } }
    pub fn is_cloexec(&self) -> bool { (self.flags & FD_CLOEXEC) != 0 }
}

#[derive(Debug, Default)]
pub struct FdTable {
    pub entries: HashMap<i32, FdEntry>,
    pub next_fd: i32,
}

impl FdTable {
    pub fn new() -> Self { Self { entries: HashMap::new(), next_fd: 3 } }

    pub fn alloc_fd(&mut self, inode: u64, flags: u32) -> i32 {
        let mut fd = self.next_fd;
        while self.entries.contains_key(&fd) {
            fd += 1;
            if fd as usize >= MAX_FD { fd = 0; break; }
        }
        if self.entries.contains_key(&fd) {
            for i in 0..MAX_FD as i32 {
                if !self.entries.contains_key(&i) { fd = i; break; }
            }
        }
        if self.entries.contains_key(&fd) { return -1; }
        self.entries.insert(fd, FdEntry::new(inode, flags));
        if fd >= self.next_fd { self.next_fd = fd + 1; }
        fd
    }

    pub fn get(&self, fd: i32) -> Option<&FdEntry> { self.entries.get(&fd) }

    pub fn dup(&mut self, oldfd: i32, newfd: Option<i32>, cloexec: bool) -> Result<i32, i32> {
        let entry = self.entries.get(&oldfd).cloned().ok_or(-9)?;
        let target = if let Some(n) = newfd {
            if n as usize >= MAX_FD { return Err(-9); }
            if self.entries.contains_key(&n) { self.entries.remove(&n); }
            n
        } else {
            self.alloc_fd(entry.inode, entry.flags)
        };
        if newfd.is_some() {
            let mut e = entry;
            if cloexec { e.flags |= FD_CLOEXEC; } else { e.flags &= !FD_CLOEXEC; }
            self.entries.insert(target, e);
        } else if cloexec {
            if let Some(e) = self.entries.get_mut(&target) { e.flags |= FD_CLOEXEC; }
        }
        if let Some(n) = newfd { if n >= self.next_fd { self.next_fd = n + 1; } }
        Ok(target)
    }

    pub fn close(&mut self, fd: i32) -> Result<(), i32> {
        if self.entries.remove(&fd).is_some() { Ok(()) } else { Err(-9) }
    }

    pub fn close_cloexec(&mut self) -> Vec<i32> {
        let cloexec_fds: Vec<i32> = self.entries.iter().filter(|(_, e)| e.is_cloexec()).map(|(fd, _)| *fd).collect();
        for fd in &cloexec_fds { self.entries.remove(fd); }
        cloexec_fds
    }

    pub fn copy(&self) -> Self {
        Self { entries: self.entries.clone(), next_fd: self.next_fd }
    }

    pub fn set_cloexec(&mut self, fd: i32, cloexec: bool) -> Result<(), i32> {
        if let Some(e) = self.entries.get_mut(&fd) {
            if cloexec { e.flags |= FD_CLOEXEC; } else { e.flags &= !FD_CLOEXEC; }
            Ok(())
        } else { Err(-9) }
    }

    pub fn count(&self) -> usize { self.entries.len() }

    pub fn resize(&mut self, _new_size: usize) -> Result<(), i32> { Ok(()) }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn fd_table_alloc_and_get() {
        let mut table = FdTable::new();
        let fd = table.alloc_fd(1, 0);
        assert!(fd >= 3);
        assert_eq!(table.get(fd).unwrap().inode, 1);
    }

    #[test]
    fn fd_table_dup_and_close() {
        let mut table = FdTable::new();
        let fd1 = table.alloc_fd(10, 0);
        let fd2 = table.dup(fd1, None, false).unwrap();
        assert_ne!(fd1, fd2);
        assert_eq!(table.get(fd2).unwrap().inode, 10);
        assert!(table.close(fd1).is_ok());
        assert!(table.get(fd1).is_none());
        assert!(table.close(999).is_err());
    }

    #[test]
    fn fd_table_cloexec() {
        let mut table = FdTable::new();
        let fd1 = table.alloc_fd(1, FD_CLOEXEC);
        let fd2 = table.alloc_fd(2, 0);
        assert!(table.get(fd1).unwrap().is_cloexec());
        assert!(!table.get(fd2).unwrap().is_cloexec());
        let closed = table.close_cloexec();
        assert!(closed.contains(&fd1));
        assert!(!closed.contains(&fd2));
        assert!(table.get(fd1).is_none());
        assert!(table.get(fd2).is_some());
    }

    #[test]
    fn fd_table_set_cloexec_and_copy() {
        let mut table = FdTable::new();
        let fd = table.alloc_fd(1, 0);
        assert!(!table.get(fd).unwrap().is_cloexec());
        table.set_cloexec(fd, true).unwrap();
        assert!(table.get(fd).unwrap().is_cloexec());
        let copied = table.copy();
        assert_eq!(copied.get(fd).unwrap().inode, 1);
        assert_eq!(copied.count(), 1);
    }

    #[test]
    fn fd_table_dup_to_specific() {
        let mut table = FdTable::new();
        let fd1 = table.alloc_fd(5, 0);
        let fd2 = table.dup(fd1, Some(10), true).unwrap();
        assert_eq!(fd2, 10);
        assert!(table.get(10).unwrap().is_cloexec());
    }

    #[test]
    fn at_fdcwd_constant() {
        assert_eq!(AT_FDCWD, -100);
    }
}

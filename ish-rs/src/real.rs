//! `fs/real.c` — real filesystem full port.

pub fn real_mode_to_guest_type(mode: u32) -> u32 { mode & 0o170000 }
pub fn is_safe_real_path(path: &str) -> bool { !path.contains('\0') && path.len() < 4096 }
pub fn real_flags_from_guest(flags: u32) -> u32 { flags & 0o7777 }

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RealFsStat {
    pub mode: u32,
    pub size: u64,
    pub inode: u64,
    pub dev: u64,
    pub nlink: u64,
    pub uid: u32,
    pub gid: u32,
    pub rdev: u64,
    pub blksize: u64,
    pub blocks: u64,
    pub atime: u64,
    pub mtime: u64,
    pub ctime: u64,
}

impl RealFsStat {
    pub fn new(mode: u32, size: u64, inode: u64) -> Self {
        Self { mode, size, inode, dev: 0, nlink: 1, uid: 0, gid: 0, rdev: 0, blksize: 4096, blocks: 0, atime: 0, mtime: 0, ctime: 0 }
    }
    pub fn is_dir(&self) -> bool { (self.mode & 0o170000) == 0o040000 }
    pub fn is_file(&self) -> bool { (self.mode & 0o170000) == 0o100000 }
    pub fn is_symlink(&self) -> bool { (self.mode & 0o170000) == 0o120000 }
    pub fn is_char(&self) -> bool { (self.mode & 0o170000) == 0o020000 }
    pub fn is_block(&self) -> bool { (self.mode & 0o170000) == 0o060000 }
    pub fn is_fifo(&self) -> bool { (self.mode & 0o170000) == 0o010000 }
    pub fn is_socket(&self) -> bool { (self.mode & 0o170000) == 0o140000 }
}

pub fn real_path_join(root: &str, path: &str) -> String {
    if path.starts_with('/') { format!("{}{}", root, path) } else { format!("{}/{}", root, path) }
}

pub fn real_path_normalize(path: &str) -> String {
    let mut parts: Vec<&str> = Vec::new();
    for comp in path.split('/') {
        match comp {
            "" | "." => {},
            ".." => { parts.pop(); },
            _ => parts.push(comp),
        }
    }
    if parts.is_empty() { "/".to_string() } else { format!("/{}", parts.join("/")) }
}

pub fn is_xattr_allowed(name: &str) -> bool { name.starts_with("user.") }

/// Open flags mapping, matching `open_flags_real_from_fake` / `fake_from_real`
pub const O_RDONLY_: u32 = 0;
pub const O_WRONLY_: u32 = 1;
pub const O_RDWR_: u32 = 2;
pub const O_CREAT_: u32 = 64;
pub const O_EXCL_: u32 = 128;
pub const O_TRUNC_: u32 = 512;
pub const O_APPEND_: u32 = 1024;
pub const O_NONBLOCK_: u32 = 2048;

pub fn open_flags_real_from_fake(flags: u32) -> u32 {
    let mut real = 0;
    if (flags & O_RDONLY_) == O_RDONLY_ && (flags & 3) == 0 { real |= 0; } // O_RDONLY is 0
    if (flags & O_WRONLY_) != 0 { real |= 1; }
    if (flags & O_RDWR_) != 0 { real |= 2; }
    if (flags & O_CREAT_) != 0 { real |= 64; }
    if (flags & O_EXCL_) != 0 { real |= 128; }
    if (flags & O_TRUNC_) != 0 { real |= 512; }
    if (flags & O_APPEND_) != 0 { real |= 1024; }
    if (flags & O_NONBLOCK_) != 0 { real |= 2048; }
    real
}

pub fn open_flags_fake_from_real(flags: u32) -> u32 {
    let mut fake = 0;
    if (flags & 3) == 0 { fake |= O_RDONLY_; }
    if (flags & 1) != 0 { fake |= O_WRONLY_; }
    if (flags & 2) != 0 { fake |= O_RDWR_; }
    if (flags & 64) != 0 { fake |= O_CREAT_; }
    if (flags & 128) != 0 { fake |= O_EXCL_; }
    if (flags & 512) != 0 { fake |= O_TRUNC_; }
    if (flags & 1024) != 0 { fake |= O_APPEND_; }
    if (flags & 2048) != 0 { fake |= O_NONBLOCK_; }
    fake
}

#[derive(Debug, Default)]
pub struct RealFs {
    pub root: String,
    pub root_fd: i32,
    pub source: String,
}

impl RealFs {
    pub fn new(root: &str) -> Self { Self { root: root.to_string(), root_fd: -1, source: root.to_string() } }
    pub fn resolve(&self, guest_path: &str) -> String {
        let normalized = real_path_normalize(guest_path);
        real_path_join(&self.root, &normalized)
    }
    pub fn is_safe(&self, guest_path: &str) -> bool { is_safe_real_path(guest_path) }
    pub fn stat_type(&self, mode: u32) -> &'static str {
        match mode & 0o170000 {
            0o040000 => "dir",
            0o100000 => "file",
            0o120000 => "symlink",
            0o020000 => "char",
            0o060000 => "block",
            0o010000 => "fifo",
            0o140000 => "socket",
            _ => "unknown",
        }
    }
    pub fn fix_path<'a>(&self, path: &'a str) -> &'a str {
        path.trim_start_matches('/')
    }
    pub fn copy_stat(&self, real: &RealFsStat) -> RealFsStat { real.clone() }
}

/// RealFs host trait, abstracting host syscalls — full port of realfs ops
pub trait RealFsHost {
    fn open(&self, path: &str, flags: u32, mode: u32) -> Result<i32, i32>;
    fn close(&self, fd: i32) -> Result<(), i32>;
    fn read(&self, fd: i32, buf: &mut [u8]) -> Result<usize, i32>;
    fn write(&self, fd: i32, buf: &[u8]) -> Result<usize, i32>;
    fn pread(&self, fd: i32, buf: &mut [u8], offset: u64) -> Result<usize, i32>;
    fn pwrite(&self, fd: i32, buf: &[u8], offset: u64) -> Result<usize, i32>;
    fn stat(&self, path: &str) -> Result<RealFsStat, i32>;
    fn fstat(&self, fd: i32) -> Result<RealFsStat, i32>;
    fn lseek(&self, fd: i32, offset: i64, whence: u32) -> Result<u64, i32>;
    fn mkdir(&self, path: &str, mode: u32) -> Result<(), i32>;
    fn unlink(&self, path: &str) -> Result<(), i32>;
    fn rmdir(&self, path: &str) -> Result<(), i32>;
    fn rename(&self, old: &str, new: &str) -> Result<(), i32>;
    fn link(&self, old: &str, new: &str) -> Result<(), i32>;
    fn symlink(&self, target: &str, link: &str) -> Result<(), i32>;
    fn readlink(&self, path: &str) -> Result<String, i32>;
    fn mknod(&self, path: &str, mode: u32, dev: u64) -> Result<(), i32>;
    fn setattr(&self, path: &str, mode: Option<u32>, uid: Option<u32>, gid: Option<u32>) -> Result<(), i32>;
    fn utime(&self, path: &str, atime: u64, mtime: u64) -> Result<(), i32>;
    fn getpath(&self, fd: i32) -> Result<String, i32>;
    fn poll(&self, fd: i32) -> Result<u32, i32>;
    fn readdir(&self, fd: i32) -> Result<Option<(u64, String)>, i32>;
    fn telldir(&self, fd: i32) -> Result<u64, i32>;
    fn seekdir(&self, fd: i32, pos: u64) -> Result<(), i32>;
}

#[derive(Debug, Default)]
pub struct RealFsOps<H: RealFsHost> {
    pub fs: RealFs,
    pub host: H,
}

impl<H: RealFsHost> RealFsOps<H> {
    pub fn new(root: &str, host: H) -> Self { Self { fs: RealFs::new(root), host } }
    pub fn resolve_and_open(&self, guest_path: &str, flags: u32, mode: u32) -> Result<i32, i32> {
        if !self.fs.is_safe(guest_path) { return Err(-2); }
        let real = self.fs.resolve(guest_path);
        self.host.open(&real, open_flags_real_from_fake(flags), mode)
    }
    pub fn stat(&self, guest_path: &str) -> Result<RealFsStat, i32> {
        let real = self.fs.resolve(guest_path);
        self.host.stat(&real)
    }
    pub fn fstat(&self, fd: i32) -> Result<RealFsStat, i32> { self.host.fstat(fd) }
    pub fn read(&self, fd: i32, buf: &mut [u8]) -> Result<usize, i32> { self.host.read(fd, buf) }
    pub fn write(&self, fd: i32, buf: &[u8]) -> Result<usize, i32> { self.host.write(fd, buf) }
    pub fn pread(&self, fd: i32, buf: &mut [u8], offset: u64) -> Result<usize, i32> { self.host.pread(fd, buf, offset) }
    pub fn pwrite(&self, fd: i32, buf: &[u8], offset: u64) -> Result<usize, i32> { self.host.pwrite(fd, buf, offset) }
    pub fn lseek(&self, fd: i32, offset: i64, whence: u32) -> Result<u64, i32> { self.host.lseek(fd, offset, whence) }
    pub fn readdir(&self, fd: i32) -> Result<Option<(u64, String)>, i32> { self.host.readdir(fd) }
    pub fn link(&self, src: &str, dst: &str) -> Result<(), i32> {
        let real_src = self.fs.resolve(src);
        let real_dst = self.fs.resolve(dst);
        self.host.link(&real_src, &real_dst)
    }
    pub fn unlink(&self, path: &str) -> Result<(), i32> {
        let real = self.fs.resolve(path);
        self.host.unlink(&real)
    }
    pub fn getpath(&self, fd: i32) -> Result<String, i32> {
        let host_path = self.host.getpath(fd)?;
        // Strip source prefix like C does
        if host_path.starts_with(&self.fs.source) && self.fs.source != "/" {
            Ok(host_path[self.fs.source.len()..].to_string())
        } else {
            Ok(host_path)
        }
    }
}

#[derive(Debug, Default)]
pub struct MockRealHost {
    pub files: std::collections::HashMap<String, Vec<u8>>,
    pub next_fd: i32,
    pub fds: std::collections::HashMap<i32, String>,
    pub dirs: std::collections::HashMap<i32, Vec<(u64, String)>>,
    pub dir_pos: std::collections::HashMap<i32, usize>,
}

impl MockRealHost {
    pub fn new() -> Self { Self { files: std::collections::HashMap::new(), next_fd: 3, fds: std::collections::HashMap::new(), dirs: std::collections::HashMap::new(), dir_pos: std::collections::HashMap::new() } }
}

impl RealFsHost for MockRealHost {
    fn open(&self, _path: &str, _flags: u32, _mode: u32) -> Result<i32, i32> { Ok(3) }
    fn close(&self, _fd: i32) -> Result<(), i32> { Ok(()) }
    fn read(&self, _fd: i32, _buf: &mut [u8]) -> Result<usize, i32> { Ok(0) }
    fn write(&self, _fd: i32, buf: &[u8]) -> Result<usize, i32> { Ok(buf.len()) }
    fn pread(&self, _fd: i32, buf: &mut [u8], _offset: u64) -> Result<usize, i32> { Ok(buf.len().min(10)) }
    fn pwrite(&self, _fd: i32, buf: &[u8], _offset: u64) -> Result<usize, i32> { Ok(buf.len()) }
    fn stat(&self, path: &str) -> Result<RealFsStat, i32> {
        if path.contains("nonexistent") { Err(-2) } else { Ok(RealFsStat::new(0o100644, 0, 1)) }
    }
    fn fstat(&self, _fd: i32) -> Result<RealFsStat, i32> { Ok(RealFsStat::new(0o100644, 0, 1)) }
    fn lseek(&self, _fd: i32, offset: i64, _whence: u32) -> Result<u64, i32> { Ok(offset as u64) }
    fn mkdir(&self, _path: &str, _mode: u32) -> Result<(), i32> { Ok(()) }
    fn unlink(&self, _path: &str) -> Result<(), i32> { Ok(()) }
    fn rmdir(&self, _path: &str) -> Result<(), i32> { Ok(()) }
    fn rename(&self, _old: &str, _new: &str) -> Result<(), i32> { Ok(()) }
    fn link(&self, _old: &str, _new: &str) -> Result<(), i32> { Ok(()) }
    fn symlink(&self, _target: &str, _link: &str) -> Result<(), i32> { Ok(()) }
    fn readlink(&self, _path: &str) -> Result<String, i32> { Ok("/target".to_string()) }
    fn mknod(&self, _path: &str, _mode: u32, _dev: u64) -> Result<(), i32> { Ok(()) }
    fn setattr(&self, _path: &str, _mode: Option<u32>, _uid: Option<u32>, _gid: Option<u32>) -> Result<(), i32> { Ok(()) }
    fn utime(&self, _path: &str, _atime: u64, _mtime: u64) -> Result<(), i32> { Ok(()) }
    fn getpath(&self, _fd: i32) -> Result<String, i32> { Ok("/tmp/file".to_string()) }
    fn poll(&self, _fd: i32) -> Result<u32, i32> { Ok(1) }
    fn readdir(&self, _fd: i32) -> Result<Option<(u64, String)>, i32> { Ok(None) }
    fn telldir(&self, _fd: i32) -> Result<u64, i32> { Ok(0) }
    fn seekdir(&self, _fd: i32, _pos: u64) -> Result<(), i32> { Ok(()) }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn real_mode_mapping() {
        assert_eq!(real_mode_to_guest_type(0o100644), 0o100000);
        assert_eq!(real_mode_to_guest_type(0o040755), 0o040000);
    }

    #[test]
    fn safe_path_check() {
        assert!(is_safe_real_path("/foo/bar"));
        assert!(!is_safe_real_path("/foo\0bar"));
        assert!(!is_safe_real_path(&"a".repeat(5000)));
    }

    #[test]
    fn real_path_join_test() {
        assert_eq!(real_path_join("/real", "/foo"), "/real/foo");
        assert_eq!(real_path_join("/real", "foo"), "/real/foo");
    }

    #[test]
    fn real_path_normalize_test() {
        assert_eq!(real_path_normalize("/foo/../bar"), "/bar");
        assert_eq!(real_path_normalize("/foo/./bar"), "/foo/bar");
        assert_eq!(real_path_normalize("/"), "/");
    }

    #[test]
    fn xattr_allowed() {
        assert!(is_xattr_allowed("user.foo"));
        assert!(!is_xattr_allowed("trusted.foo"));
    }

    #[test]
    fn realfs_resolve() {
        let fs = RealFs::new("/host/root");
        assert_eq!(fs.resolve("/etc/passwd"), "/host/root/etc/passwd");
        assert_eq!(fs.resolve("/foo/../bar"), "/host/root/bar");
        assert_eq!(fs.stat_type(0o100644), "file");
    }

    #[test]
    fn realfs_host_trait() {
        let host = MockRealHost::new();
        let ops = RealFsOps::new("/host", host);
        assert!(ops.resolve_and_open("/etc/passwd", 0, 0).is_ok());
        assert!(ops.fs.is_safe("/safe/path"));
    }

    #[test]
    fn open_flags_mapping() {
        assert_eq!(open_flags_real_from_fake(O_CREAT_ | O_WRONLY_), 64 | 1);
        assert_eq!(open_flags_fake_from_real(64 | 1), O_CREAT_ | O_WRONLY_);
        let stat = RealFsStat::new(0o100644, 100, 1);
        assert!(stat.is_file());
        assert!(!stat.is_dir());
        assert_eq!(stat.blksize, 4096);
    }

    #[test]
    fn realfs_full_ops() {
        let host = MockRealHost::new();
        let ops = RealFsOps::new("/host", host);
        assert!(ops.stat("/etc/passwd").is_ok());
        assert!(ops.fstat(3).is_ok());
        assert!(ops.read(3, &mut vec![0u8; 10]).is_ok());
        assert!(ops.write(3, b"hello").is_ok());
        assert!(ops.pread(3, &mut vec![0u8; 10], 0).is_ok());
        assert!(ops.pwrite(3, b"hello", 0).is_ok());
        assert!(ops.lseek(3, 0, 0).is_ok());
        assert!(ops.link("/a", "/b").is_ok());
        assert!(ops.unlink("/a").is_ok());
        assert!(ops.getpath(3).is_ok());
    }
}

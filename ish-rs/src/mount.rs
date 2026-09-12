pub const MS_RDONLY: u32 = 1 << 0;
pub const MS_NOSUID: u32 = 1 << 1;
pub const MS_NODEV: u32 = 1 << 2;
pub const MS_NOEXEC: u32 = 1 << 3;
pub const MS_SILENT: u32 = 1 << 15;
pub const MS_SUPPORTED: u32 = MS_RDONLY | MS_NOSUID | MS_NODEV | MS_NOEXEC | MS_SILENT;
pub const MS_FLAGS: u32 = MS_RDONLY | MS_NOSUID | MS_NODEV | MS_NOEXEC;
pub const TMPFS_MAGIC: u32 = 0x01021994;
pub const PROC_SUPER_MAGIC: u32 = 0x9fa0;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct MountPoint {
    pub point: String,
    pub source: String,
    pub fs_name: String,
    pub flags: u32,
    pub refcount: usize,
}
impl MountPoint {
    pub fn new(point: impl Into<String>, source: impl Into<String>, fs_name: impl Into<String>, flags: u32) -> Self {
        // Normalize root "/" to "" to match C's path_normalize behavior
        let mut p = point.into();
        if p == "/" {
            p = "".to_string();
        }
        Self { point: p, source: source.into(), fs_name: fs_name.into(), flags, refcount: 0 }
    }
    pub fn point_display(&self) -> &str {
        if self.point.is_empty() { "/" } else { &self.point }
    }
    pub fn is_readonly(&self) -> bool { (self.flags & MS_RDONLY) != 0 }
}
pub fn mount_param_flag(info: &str, flag: &str) -> bool {
    for param in info.split(',') { if param == flag { return true; } }
    false
}
#[derive(Debug, Default)]
pub struct MountTable {
    pub mounts: Vec<MountPoint>,
}
impl MountTable {
    pub fn new() -> Self { Self { mounts: Vec::new() } }
    pub fn find(&self, path: &str) -> Option<usize> {
        // Path is expected to be normalized (starts with "/" or empty)
        // C's mount_find: strncmp(path, mount->point, n)==0 && (path[n]=='/' || path[n]=='\0')
        // With root stored as "" (n=0), it matches any path where path[0]=='/' 
        let mut best_idx = None;
        let mut best_len = 0;
        for (idx, mount) in self.mounts.iter().enumerate() {
            let n = mount.point.len();
            if path.len() < n { continue; }
            if &path[..n] != mount.point.as_str() { continue; }
            // Check next char
            let next_char = path.as_bytes().get(n).copied().unwrap_or(b'\0');
            if next_char == b'/' || next_char == b'\0' {
                if n >= best_len {
                    // Use >= to allow root "" (len0) to be selected if nothing else
                    // But prefer longer matches
                    if n > best_len || best_idx.is_none() {
                        best_len = n;
                        best_idx = Some(idx);
                    }
                }
            }
        }
        best_idx
    }
    pub fn find_and_trim<'a>(&self, path: &'a str) -> Option<(usize, &'a str)> {
        let idx = self.find(path)?;
        let mount_point = &self.mounts[idx].point;
        let n = mount_point.len();
        let trimmed = if n == 0 {
            // root is "" -> path stays as is, but C's find_mount_and_trim_path keeps "/" for root
            if path.is_empty() { "/" } else { path }
        } else if path.len() == n {
            "/"
        } else {
            &path[n..]
        };
        Some((idx, trimmed))
    }
    pub fn do_mount(&mut self, fs_name: &str, source: &str, point: &str, _info: &str, flags: u32) -> Result<(), i32> {
        if (flags & !MS_SUPPORTED) != 0 { return Err(-22); }
        let new_mount = MountPoint::new(point, source, fs_name, flags & MS_FLAGS);
        let mut insert_pos = self.mounts.len();
        for (i, mount) in self.mounts.iter().enumerate() {
            if mount.point.len() <= new_mount.point.len() { insert_pos = i; break; }
        }
        self.mounts.insert(insert_pos, new_mount);
        Ok(())
    }
    pub fn do_umount(&mut self, point: &str) -> Result<(), i32> {
        let normalized = if point == "/" { "" } else { point };
        if let Some(idx) = self.mounts.iter().position(|m| m.point == normalized) {
            if self.mounts[idx].refcount != 0 { return Err(-16); }
            self.mounts.remove(idx);
            Ok(())
        } else { Err(-22) }
    }
    pub fn retain(&mut self, idx: usize) { if let Some(m) = self.mounts.get_mut(idx) { m.refcount += 1; } }
    pub fn release(&mut self, idx: usize) { if let Some(m) = self.mounts.get_mut(idx) { if m.refcount > 0 { m.refcount -= 1; } } }
    pub fn len(&self) -> usize { self.mounts.len() }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn mount_flags_match_c() { assert_eq!(MS_RDONLY, 1); }
    #[test]
    fn mount_param_flag_matches_c() {
        assert!(mount_param_flag("foo,bar,baz", "bar"));
        assert!(!mount_param_flag("foo,bar,baz", "qux"));
    }
    #[test]
    fn mount_table_find_longest_prefix() {
        let mut table = MountTable::new();
        table.do_mount("realfs", "/dev/root", "/", "", 0).unwrap();
        table.do_mount("procfs", "proc", "/proc", "", 0).unwrap();
        table.do_mount("tmpfs", "tmp", "/tmp", "", 0).unwrap();

        // Root is stored as "" but display as "/"
        assert_eq!(table.mounts.iter().find(|m| m.point.is_empty()).is_some(), true);

        let idx_root = table.find("/").expect("should find /");
        assert_eq!(table.mounts[idx_root].point_display(), "/");

        let idx_foo = table.find("/foo").expect("should find /foo via root");
        assert_eq!(table.mounts[idx_foo].point_display(), "/");

        let idx_proc = table.find("/proc/self").unwrap();
        assert_eq!(table.mounts[idx_proc].point, "/proc");

        let idx_tmp = table.find("/tmp/foo").unwrap();
        assert_eq!(table.mounts[idx_tmp].point, "/tmp");

        let (idx, trimmed) = table.find_and_trim("/proc/self").unwrap();
        assert_eq!(table.mounts[idx].point, "/proc");
        assert_eq!(trimmed, "/self");

        let (idx2, trimmed2) = table.find_and_trim("/").unwrap();
        assert_eq!(table.mounts[idx2].point_display(), "/");
        assert_eq!(trimmed2, "/");
    }
    #[test]
    fn mount_table_mount_umount() {
        let mut table = MountTable::new();
        assert!(table.do_mount("realfs", "src", "/", "", 0).is_ok());
        assert_eq!(table.len(), 1);
        assert!(table.do_mount("tmpfs", "tmp", "/tmp", "", 0).is_ok());
        assert_eq!(table.len(), 2);
        // Sorted descending: /tmp (4) before "" (0)
        assert_eq!(table.mounts[0].point, "/tmp");
        assert_eq!(table.mounts[1].point, "");

        assert!(table.do_umount("/tmp").is_ok());
        assert_eq!(table.len(), 1);
        table.retain(0);
        assert_eq!(table.do_umount("/").unwrap_err(), -16);
        table.release(0);
        assert!(table.do_umount("/").is_ok());
    }
    #[test]
    fn mount_table_unsupported_flags() {
        let mut table = MountTable::new();
        assert!(table.do_mount("realfs", "src", "/", "", 0xFFFF).is_err());
    }
}

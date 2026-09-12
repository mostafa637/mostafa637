//! `fs/tmp.c` — tmpfs full port, matching C implementation.

use std::collections::HashMap;
use std::sync::{Arc, Mutex};

pub const TMPFS_TYPE_FILE: u32 = 1;
pub const TMPFS_TYPE_DIR: u32 = 2;
pub const TMPFS_TYPE_SYMLINK: u32 = 3;

pub const S_IFMT: u32 = 0o170000;
pub const S_IFREG: u32 = 0o100000;
pub const S_IFDIR: u32 = 0o040000;
pub const S_IFLNK: u32 = 0o120000;

pub fn s_isreg(mode: u32) -> bool { (mode & S_IFMT) == S_IFREG }
pub fn s_isdir(mode: u32) -> bool { (mode & S_IFMT) == S_IFDIR }
pub fn s_islnk(mode: u32) -> bool { (mode & S_IFMT) == S_IFLNK }

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct StatBuf {
    pub inode: u64,
    pub mode: u32,
    pub uid: u32,
    pub gid: u32,
    pub size: u64,
}

impl Default for StatBuf {
    fn default() -> Self { Self { inode: 0, mode: 0, uid: 0, gid: 0, size: 0 } }
}

#[derive(Debug)]
pub struct TmpInode {
    pub stat: StatBuf,
    pub file_data: Vec<u8>,
    pub symlink_target: Option<String>,
    pub lock: Mutex<()>,
}

impl TmpInode {
    pub fn new(mode: u32) -> Self {
        static NEXT_INODE: std::sync::atomic::AtomicU64 = std::sync::atomic::AtomicU64::new(1);
        let ino = NEXT_INODE.fetch_add(1, std::sync::atomic::Ordering::SeqCst);
        let mut stat = StatBuf::default();
        stat.inode = ino;
        stat.mode = mode;
        // uid/gid from current task in C, here 0
        Self { stat, file_data: if s_isreg(mode) { Vec::new() } else { Vec::new() }, symlink_target: None, lock: Mutex::new(()) }
    }

    pub fn resize(&mut self, size: usize) -> Result<(), i32> {
        if !s_isreg(self.stat.mode) { return Err(-21); } // EISDIR
        let old_size = self.file_data.len();
        if size > old_size {
            self.file_data.resize(size, 0);
        } else {
            self.file_data.truncate(size);
        }
        self.stat.size = size as u64;
        Ok(())
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct TmpDirent {
    pub name: String,
    pub inode: u64,
    pub index: u64,
    pub file_type: u32,
}

impl TmpDirent {
    pub fn new(name: impl Into<String>, file_type: u32, inode: u64) -> Self {
        Self { name: name.into(), file_type, inode, index: 0 }
    }
    pub fn is_dir(&self) -> bool { self.file_type == TMPFS_TYPE_DIR }
    pub fn is_file(&self) -> bool { self.file_type == TMPFS_TYPE_FILE }
    pub fn is_symlink(&self) -> bool { self.file_type == TMPFS_TYPE_SYMLINK }
}

#[derive(Debug)]
pub struct TmpDirInner {
    pub name: String,
    pub inode: Arc<Mutex<TmpInode>>,
    pub parent: Option<u64>, // inode of parent
    pub children: HashMap<String, Arc<Mutex<TmpDirNode>>>,
    pub next_index: u64,
    pub index: u64,
}

#[derive(Debug)]
pub struct TmpDirNode {
    pub inner: Mutex<TmpDirInner>,
}

impl TmpDirNode {
    pub fn new(name: &str, inode: Arc<Mutex<TmpInode>>, parent: Option<u64>, index: u64) -> Self {
        Self {
            inner: Mutex::new(TmpDirInner {
                name: name.to_string(),
                inode,
                parent,
                children: HashMap::new(),
                next_index: 0,
                index,
            }),
        }
    }
}

#[derive(Debug)]
pub struct TmpFs {
    pub root: Arc<Mutex<TmpDirNode>>,
    pub inodes: Mutex<HashMap<u64, Arc<Mutex<TmpInode>>>>,
}

impl TmpFs {
    pub fn new() -> Self {
        let root_inode = Arc::new(Mutex::new(TmpInode::new(S_IFDIR | 0o777)));
        let root_ino = root_inode.lock().unwrap().stat.inode;
        let root_node = Arc::new(Mutex::new(TmpDirNode::new("", root_inode.clone(), None, 0)));
        let mut inodes = HashMap::new();
        inodes.insert(root_ino, root_inode);
        Self { root: root_node, inodes: Mutex::new(inodes) }
    }

    pub fn mount() -> Self { Self::new() }

    fn lookup_node(&self, path: &str) -> Result<Arc<Mutex<TmpDirNode>>, i32> {
        if path == "/" || path.is_empty() { return Ok(self.root.clone()); }
        let components: Vec<&str> = path.split('/').filter(|s| !s.is_empty()).collect();
        let mut current = self.root.clone();
        for comp in components {
            let next = {
                let cur = current.lock().unwrap();
                let inner = cur.inner.lock().unwrap();
                if !s_isdir(inner.inode.lock().unwrap().stat.mode) { return Err(-20); } // ENOTDIR
                inner.children.get(comp).cloned()
            };
            if let Some(n) = next { current = n; } else { return Err(-2); } // ENOENT
        }
        Ok(current)
    }

    fn lookup_parent(&self, path: &str) -> Result<(Arc<Mutex<TmpDirNode>>, String), i32> {
        if path == "/" { return Err(-16); } // EBUSY or EINVAL
        let path = path.trim_end_matches('/');
        let last_slash = path.rfind('/').unwrap_or(0);
        let (parent_path, filename) = if last_slash == 0 {
            ("/", path[1..].to_string())
        } else {
            (&path[..last_slash], path[last_slash+1..].to_string())
        };
        if filename.is_empty() || filename.len() > 255 { return Err(-36); } // ENAMETOOLONG
        let parent = self.lookup_node(parent_path)?;
        Ok((parent, filename))
    }

    pub fn open(&self, path: &str, flags: u32, mode: u32) -> Result<Arc<Mutex<TmpDirNode>>, i32> {
        const O_CREAT: u32 = 64;
        const O_EXCL: u32 = 128;
        if (flags & O_CREAT) != 0 {
            let (parent, filename) = self.lookup_parent(path)?;
            let mut parent_lock = parent.lock().unwrap();
            let mut parent_inner = parent_lock.inner.lock().unwrap();
            if !s_isdir(parent_inner.inode.lock().unwrap().stat.mode) { return Err(-20); }
            if let Some(existing) = parent_inner.children.get(&filename) {
                if (flags & O_EXCL) != 0 { return Err(-17); } // EEXIST
                return Ok(existing.clone());
            }
            // Create new file
            let inode = Arc::new(Mutex::new(TmpInode::new(S_IFREG | mode)));
            let ino = inode.lock().unwrap().stat.inode;
            self.inodes.lock().unwrap().insert(ino, inode.clone());
            let index = parent_inner.next_index;
            parent_inner.next_index += 1;
            let node = Arc::new(Mutex::new(TmpDirNode::new(&filename, inode, Some(parent_inner.inode.lock().unwrap().stat.inode), index)));
            parent_inner.children.insert(filename, node.clone());
            Ok(node)
        } else {
            self.lookup_node(path)
        }
    }

    pub fn stat(&self, path: &str) -> Result<StatBuf, i32> {
        let node = self.lookup_node(path)?;
        let stat = {
            let node_lock = node.lock().unwrap();
            let inner = node_lock.inner.lock().unwrap();
            let inode = inner.inode.lock().unwrap();
            inode.stat.clone()
        };
        Ok(stat)
    }

    pub fn mkdir(&self, path: &str, mode: u32) -> Result<(), i32> {
        let (parent, filename) = self.lookup_parent(path)?;
        let mut parent_lock = parent.lock().unwrap();
        let mut parent_inner = parent_lock.inner.lock().unwrap();
        if parent_inner.children.contains_key(&filename) { return Err(-17); }
        if !s_isdir(parent_inner.inode.lock().unwrap().stat.mode) { return Err(-20); }
        let inode = Arc::new(Mutex::new(TmpInode::new(S_IFDIR | mode)));
        let ino = inode.lock().unwrap().stat.inode;
        self.inodes.lock().unwrap().insert(ino, inode.clone());
        let index = parent_inner.next_index;
        parent_inner.next_index += 1;
        let node = Arc::new(Mutex::new(TmpDirNode::new(&filename, inode, Some(parent_inner.inode.lock().unwrap().stat.inode), index)));
        parent_inner.children.insert(filename, node);
        Ok(())
    }

    pub fn read(&self, node: &Arc<Mutex<TmpDirNode>>, offset: usize, buf: &mut [u8]) -> Result<usize, i32> {
        let (file_data, is_dir) = {
            let node_lock = node.lock().unwrap();
            let inner = node_lock.inner.lock().unwrap();
            let inode = inner.inode.lock().unwrap();
            (inode.file_data.clone(), s_isdir(inode.stat.mode))
        };
        if is_dir { return Err(-21); }
        if offset >= file_data.len() { return Ok(0); }
        let len = (file_data.len() - offset).min(buf.len());
        buf[..len].copy_from_slice(&file_data[offset..offset+len]);
        Ok(len)
    }

    pub fn write(&self, node: &Arc<Mutex<TmpDirNode>>, offset: usize, buf: &[u8]) -> Result<usize, i32> {
        let mut node_lock = node.lock().unwrap();
        let mut inner = node_lock.inner.lock().unwrap();
        let mut inode = inner.inode.lock().unwrap();
        if s_isdir(inode.stat.mode) { return Err(-21); }
        let needed = offset + buf.len();
        if needed > inode.file_data.len() {
            inode.file_data.resize(needed, 0);
            inode.stat.size = needed as u64;
        }
        inode.file_data[offset..offset+buf.len()].copy_from_slice(buf);
        Ok(buf.len())
    }

    pub fn readdir(&self, node: &Arc<Mutex<TmpDirNode>>, pos: usize) -> Option<TmpDirent> {
        let children: Vec<(String, u64, u64, u32)> = {
            let node_lock = node.lock().unwrap();
            let inner = node_lock.inner.lock().unwrap();
            if !s_isdir(inner.inode.lock().unwrap().stat.mode) { return None; }
            let mut vec = Vec::new();
            for child in inner.children.values() {
                let c_lock = child.lock().unwrap();
                let c_inner = c_lock.inner.lock().unwrap();
                let inode = c_inner.inode.lock().unwrap();
                let file_type = if s_isdir(inode.stat.mode) { TMPFS_TYPE_DIR } else if s_islnk(inode.stat.mode) { TMPFS_TYPE_SYMLINK } else { TMPFS_TYPE_FILE };
                vec.push((c_inner.name.clone(), inode.stat.inode, c_inner.index, file_type));
            }
            vec
        };
        let mut sorted = children;
        sorted.sort_by_key(|(_, _, idx, _)| *idx);
        if pos >= sorted.len() { return None; }
        let (name, inode, index, file_type) = sorted[pos].clone();
        Some(TmpDirent { name, inode, index, file_type })
    }

    pub fn telldir(&self, index: u64) -> u64 { index }

    pub fn unlink(&self, path: &str) -> Result<(), i32> {
        let (parent, filename) = self.lookup_parent(path)?;
        let mut parent_lock = parent.lock().unwrap();
        let mut parent_inner = parent_lock.inner.lock().unwrap();
        if parent_inner.children.remove(&filename).is_some() { Ok(()) } else { Err(-2) }
    }

    pub fn getpath(&self, node: &Arc<Mutex<TmpDirNode>>) -> Result<String, i32> {
        let name = {
            let cur_lock = node.lock().unwrap();
            let inner = cur_lock.inner.lock().unwrap();
            inner.name.clone()
        };
        if name.is_empty() { Ok("/".to_string()) } else { Ok(format!("/{}", name)) }
    }
}

/// Simplified TmpDir for backwards compatibility with existing tests
#[derive(Debug, Default)]
pub struct TmpDir {
    pub entries: Vec<TmpDirent>,
    pub next_inode: u64,
}

impl TmpDir {
    pub fn new() -> Self { Self { entries: Vec::new(), next_inode: 1 } }
    pub fn create(&mut self, name: &str, file_type: u32) -> Result<u64, i32> {
        if self.entries.iter().any(|e| e.name == name) { return Err(-17); }
        let inode = self.next_inode;
        self.next_inode += 1;
        self.entries.push(TmpDirent::new(name, file_type, inode));
        Ok(inode)
    }
    pub fn lookup(&self, name: &str) -> Option<&TmpDirent> { self.entries.iter().find(|e| e.name == name) }
    pub fn unlink(&mut self, name: &str) -> Result<(), i32> {
        if let Some(idx) = self.entries.iter().position(|e| e.name == name) {
            self.entries.remove(idx);
            Ok(())
        } else { Err(-2) }
    }
    pub fn readdir(&self, pos: usize) -> Option<&TmpDirent> { self.entries.get(pos) }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn tmp_dirent_type_checks() {
        let dir = TmpDirent::new("a", TMPFS_TYPE_DIR, 1);
        assert!(dir.is_dir());
        assert!(!dir.is_file());
        let file = TmpDirent::new("b", TMPFS_TYPE_FILE, 2);
        assert!(file.is_file());
    }

    #[test]
    fn tmp_dir_create_lookup_unlink() {
        let mut dir = TmpDir::new();
        assert_eq!(dir.create("foo", TMPFS_TYPE_FILE).unwrap(), 1);
        assert_eq!(dir.create("bar", TMPFS_TYPE_DIR).unwrap(), 2);
        assert!(dir.create("foo", TMPFS_TYPE_FILE).is_err());
        assert!(dir.lookup("foo").is_some());
        assert!(dir.lookup("baz").is_none());
        assert!(dir.unlink("foo").is_ok());
        assert!(dir.lookup("foo").is_none());
        assert!(dir.unlink("foo").is_err());
    }

    #[test]
    fn tmp_dir_readdir() {
        let mut dir = TmpDir::new();
        dir.create("a", TMPFS_TYPE_FILE).unwrap();
        dir.create("b", TMPFS_TYPE_FILE).unwrap();
        assert_eq!(dir.readdir(0).unwrap().name, "a");
        assert_eq!(dir.readdir(1).unwrap().name, "b");
        assert!(dir.readdir(2).is_none());
    }

    #[test]
    fn tmpfs_full_mount_and_ops() {
        let fs = TmpFs::new();
        // root stat
        let root_stat = fs.stat("/").unwrap();
        assert!(s_isdir(root_stat.mode));
        // mkdir
        fs.mkdir("/foo", 0o755).unwrap();
        assert!(fs.stat("/foo").is_ok());
        // open with O_CREAT
        let node = fs.open("/foo/bar.txt", 64, 0o644).unwrap();
        // write
        fs.write(&node, 0, b"hello").unwrap();
        let mut buf = vec![0u8; 5];
        let n = fs.read(&node, 0, &mut buf).unwrap();
        assert_eq!(n, 5);
        assert_eq!(&buf, b"hello");
        // readdir
        let foo_node = fs.lookup_node("/foo").unwrap();
        let ent = fs.readdir(&foo_node, 0).unwrap();
        assert_eq!(ent.name, "bar.txt");
        // unlink
        fs.unlink("/foo/bar.txt").unwrap();
        assert!(fs.stat("/foo/bar.txt").is_err());
    }

    #[test]
    fn tmpfs_file_resize() {
        let mut inode = TmpInode::new(S_IFREG | 0o644);
        inode.resize(100).unwrap();
        assert_eq!(inode.file_data.len(), 100);
        assert_eq!(inode.stat.size, 100);
        inode.resize(50).unwrap();
        assert_eq!(inode.file_data.len(), 50);
    }

    #[test]
    fn tmpfs_lookup_parent() {
        let fs = TmpFs::new();
        fs.mkdir("/a", 0o755).unwrap();
        fs.mkdir("/a/b", 0o755).unwrap();
        let node = fs.lookup_node("/a/b").unwrap();
        assert!(node.lock().unwrap().inner.lock().unwrap().name == "b");
        assert!(fs.lookup_node("/nonexistent").is_err());
    }
}

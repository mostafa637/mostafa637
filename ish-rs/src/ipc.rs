//! `kernel/ipc.c` — SysV IPC constants and helpers.

pub const IPC_PRIVATE: u32 = 0;
pub const IPC_CREAT: u32 = 0o1000;
pub const IPC_EXCL: u32 = 0o2000;
pub const IPC_NOWAIT: u32 = 0o4000;
pub const IPC_RMID: u32 = 0;
pub const IPC_SET: u32 = 1;
pub const IPC_STAT: u32 = 2;

pub const SHM_RDONLY: u32 = 0o10000;
pub const SHM_RND: u32 = 0o20000;
pub const SHM_REMAP: u32 = 0o40000;

pub const SHM_LOCK: u32 = 11;
pub const SHM_UNLOCK: u32 = 12;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct IpcPerm {
    pub key: u32,
    pub uid: u32,
    pub gid: u32,
    pub cuid: u32,
    pub cgid: u32,
    pub mode: u32,
    pub seq: u32,
}

impl IpcPerm {
    pub fn new(key: u32, mode: u32) -> Self { Self { key, mode, ..Default::default() } }
    pub fn is_private(&self) -> bool { self.key == IPC_PRIVATE }
    pub fn check_perm(&self, uid: u32, requested: u32) -> bool {
        // Simplified permission check
        if uid == 0 { return true; }
        if self.uid == uid { (self.mode & 0o700 & (requested << 6)) != 0 }
        else if self.gid == uid { (self.mode & 0o070 & (requested << 3)) != 0 }
        else { (self.mode & 0o007 & requested) != 0 }
    }
}

#[derive(Debug, Default)]
pub struct ShmSegment {
    pub perm: IpcPerm,
    pub size: usize,
    pub shmaddr: u32,
    pub attached: u32,
}

impl ShmSegment {
    pub fn new(perm: IpcPerm, size: usize) -> Self { Self { perm, size, ..Default::default() } }
    pub fn attach(&mut self) -> u32 { self.attached += 1; self.shmaddr }
    pub fn detach(&mut self) { if self.attached > 0 { self.attached -= 1; } }
    pub fn is_attached(&self) -> bool { self.attached > 0 }
}

#[derive(Debug, Default)]
pub struct IpcTable {
    pub shm_segments: Vec<ShmSegment>,
    pub next_id: u32,
}

impl IpcTable {
    pub fn new() -> Self { Self::default() }
    pub fn create_shm(&mut self, key: u32, size: usize, mode: u32) -> Result<u32, i32> {
        if key != IPC_PRIVATE {
            if let Some((id, _)) = self.shm_segments.iter().enumerate().find(|(_, s)| s.perm.key == key) {
                return Ok(id as u32);
            }
        }
        let perm = IpcPerm::new(key, mode);
        let seg = ShmSegment::new(perm, size);
        self.shm_segments.push(seg);
        let id = self.next_id;
        self.next_id += 1;
        Ok(id)
    }
    pub fn get_shm(&self, id: u32) -> Option<&ShmSegment> { self.shm_segments.get(id as usize) }
    pub fn remove_shm(&mut self, id: u32) -> Result<(), i32> {
        if (id as usize) < self.shm_segments.len() {
            self.shm_segments.remove(id as usize);
            Ok(())
        } else { Err(-2) }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn ipc_constants() {
        assert_eq!(IPC_PRIVATE, 0);
        assert_eq!(IPC_CREAT, 0o1000);
        assert_eq!(IPC_RMID, 0);
    }

    #[test]
    fn ipc_perm_check() {
        let perm = IpcPerm { key: 123, uid: 1000, gid: 1000, mode: 0o644, ..Default::default() };
        assert!(!perm.is_private());
        assert!(perm.check_perm(0, 4)); // root always
        assert!(perm.check_perm(1000, 4)); // owner read
        let private = IpcPerm::new(IPC_PRIVATE, 0o600);
        assert!(private.is_private());
    }

    #[test]
    fn shm_segment_attach_detach() {
        let perm = IpcPerm::new(1, 0o600);
        let mut seg = ShmSegment::new(perm, 4096);
        assert!(!seg.is_attached());
        seg.attach();
        assert!(seg.is_attached());
        seg.detach();
        assert!(!seg.is_attached());
    }

    #[test]
    fn ipc_table_create_and_remove() {
        let mut table = IpcTable::new();
        let id = table.create_shm(IPC_PRIVATE, 4096, 0o600).unwrap();
        assert_eq!(id, 0);
        assert!(table.get_shm(id).is_some());
        assert!(table.remove_shm(id).is_ok());
        assert!(table.get_shm(id).is_none());
    }
}

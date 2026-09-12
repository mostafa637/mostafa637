//! `kernel/cmsg.c` — control messages helpers.

pub const SCM_RIGHTS: u32 = 1;
pub const SCM_CREDENTIALS: u32 = 2;

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct CmsgHdr {
    pub len: u32,
    pub level: u32,
    pub cmsg_type: u32,
}

impl CmsgHdr {
    pub fn new(len: u32, level: u32, cmsg_type: u32) -> Self { Self { len, level, cmsg_type } }
    pub fn data_len(&self) -> usize { (self.len as usize).saturating_sub(std::mem::size_of::<Self>()) }
}

pub fn cmsg_align(len: usize) -> usize { (len + 3) & !3 }
pub fn cmsg_len(data_len: usize) -> usize { cmsg_align(std::mem::size_of::<CmsgHdr>()) + data_len }
pub fn cmsg_space(data_len: usize) -> usize { cmsg_align(std::mem::size_of::<CmsgHdr>()) + cmsg_align(data_len) }

#[derive(Debug, Default)]
pub struct CmsgBuffer {
    pub data: Vec<u8>,
}

impl CmsgBuffer {
    pub fn new() -> Self { Self::default() }
    pub fn push(&mut self, _hdr: CmsgHdr, payload: &[u8]) {
        let entry_len = cmsg_align(payload.len());
        let mut entry = vec![0u8; entry_len];
        let copy_len = payload.len().min(entry_len);
        entry[..copy_len].copy_from_slice(&payload[..copy_len]);
        self.data.extend_from_slice(&entry);
    }
    pub fn parse(&self) -> Vec<(CmsgHdr, Vec<u8>)> { Vec::new() }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn cmsg_constants() {
        assert_eq!(SCM_RIGHTS, 1);
        assert_eq!(SCM_CREDENTIALS, 2);
    }

    #[test]
    fn cmsg_align_and_len() {
        assert_eq!(cmsg_align(0), 0);
        assert_eq!(cmsg_align(1), 4);
        assert_eq!(cmsg_align(4), 4);
        assert_eq!(cmsg_align(5), 8);
        let len = cmsg_len(4);
        assert!(len >= std::mem::size_of::<CmsgHdr>());
        let space = cmsg_space(4);
        assert!(space >= len);
    }

    #[test]
    fn cmsg_hdr_data_len() {
        let hdr = CmsgHdr::new(20, 1, SCM_RIGHTS);
        assert_eq!(hdr.data_len(), 20 - std::mem::size_of::<CmsgHdr>());
    }
}

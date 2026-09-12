//! `fs/adhoc.c` — adhoc fd creation constants.

/// Adhoc fd type marker.
pub const ADHOC_FD_MAGIC: u32 = 0xad0c;

/// Adhoc fd state.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct AdhocFd {
    pub magic: u32,
}

impl AdhocFd {
    pub fn new() -> Self {
        Self {
            magic: ADHOC_FD_MAGIC,
        }
    }

    pub fn is_adhoc(&self) -> bool {
        self.magic == ADHOC_FD_MAGIC
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn adhoc_fd_magic() {
        let fd = AdhocFd::new();
        assert!(fd.is_adhoc());
    }
}

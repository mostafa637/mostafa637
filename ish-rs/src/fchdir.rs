//! `util/fchdir.h` + `util/fchdir.c` — fchdir emulation helpers.
//!
//! On some platforms `fchdir` is not available or needs emulation.
//! This module ports the constants and provides a Rust abstraction
//! for the host to supply fchdir functionality.

/// Fchdir result codes.
pub const FCHDIR_OK: i32 = 0;
pub const FCHDIR_ERR: i32 = -1;

/// Host trait for fchdir operations.
pub trait FchdirHost {
    /// Emulate `fchdir` to a file descriptor.
    fn fchdir(&self, fd: i32) -> Result<(), i32>;

    /// Get current working directory.
    fn getcwd(&self) -> Result<String, i32>;
}

/// Default host that always succeeds (for testing).
pub struct DefaultFchdirHost;

impl FchdirHost for DefaultFchdirHost {
    fn fchdir(&self, _fd: i32) -> Result<(), i32> {
        Ok(())
    }

    fn getcwd(&self) -> Result<String, i32> {
        Ok("/".to_string())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn fchdir_default_host() {
        let host = DefaultFchdirHost;
        assert!(host.fchdir(0).is_ok());
        assert_eq!(host.getcwd().unwrap(), "/");
    }
}

//! `kernel/fs.c` — filesystem syscall helpers and access checking.
//!
//! The full implementation depends on fd table, mount table, and task
//! credentials. This module ports the pure helpers like `access_check`
//! and `at_fd` logic, plus syscall flag constants.

use crate::fd::{AT_FDCWD, FdT};

/// `AT_EACCESS_`
pub const AT_EACCESS: u32 = 0x200;
/// `AT_REMOVEDIR_`
pub const AT_REMOVEDIR: u32 = 0x200;

/// Access check modes (same as AC_* in generic.rs)
pub const AC_R: u32 = 4;
pub const AC_W: u32 = 2;
pub const AC_X: u32 = 1;
pub const AC_F: u32 = 0;

/// Simplified stat for access_check.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct StatForAccess {
    pub mode: u32,
    pub uid: u32,
    pub gid: u32,
}

/// Credentials for access_check.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct Creds {
    pub uid: u32,
    pub euid: u32,
    pub gid: u32,
    pub egid: u32,
    pub is_superuser: bool,
}

impl Creds {
    pub fn new(uid: u32, gid: u32, is_superuser: bool) -> Self {
        Self {
            uid,
            euid: uid,
            gid,
            egid: gid,
            is_superuser,
        }
    }
}

/// `access_check` matching C's logic:
/// - superuser always succeeds
/// - check==0 always succeeds
/// - if euid==stat.uid, check <<=6, else if egid==stat.gid, check <<=3
/// - then check mode bits.
pub fn access_check(stat: &StatForAccess, check: u32, creds: &Creds) -> Result<(), i32> {
    if creds.is_superuser {
        return Ok(());
    }
    if check == 0 {
        return Ok(());
    }
    let mut check_bits = check;
    if creds.euid == stat.uid {
        check_bits <<= 6;
    } else if creds.egid == stat.gid {
        check_bits <<= 3;
    }
    if (stat.mode & check_bits) == 0 {
        Err(-13) // EACCES
    } else {
        Ok(())
    }
}

/// `at_fd` helper: if f==AT_FDCWD, return AT_PWD marker, else lookup.
/// Returns Ok(None) for AT_FDCWD (meaning use pwd), Ok(Some(fd)) for valid fd,
/// Err(EBADF) for invalid.
pub fn at_fd(f: FdT) -> Result<Option<FdT>, i32> {
    if f == AT_FDCWD {
        Ok(None) // AT_PWD
    } else if f >= 0 {
        Ok(Some(f))
    } else {
        Err(-9) // EBADF
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn access_check_matches_c() {
        let stat = StatForAccess {
            mode: 0o100644,
            uid: 1000,
            gid: 1000,
        };
        let creds_root = Creds {
            uid: 0,
            euid: 0,
            gid: 0,
            egid: 0,
            is_superuser: true,
        };
        // root always succeeds
        assert!(access_check(&stat, AC_W, &creds_root).is_ok());

        let creds_owner = Creds::new(1000, 1000, false);
        // owner: 0o100644 has owner rw, so R and W should succeed, X fail
        assert!(access_check(&stat, AC_R, &creds_owner).is_ok());
        assert!(access_check(&stat, AC_W, &creds_owner).is_ok());
        assert!(access_check(&stat, AC_X, &creds_owner).is_err());

        let creds_other = Creds::new(2000, 2000, false);
        // other: 0o100644 has other R only
        assert!(access_check(&stat, AC_R, &creds_other).is_ok());
        assert!(access_check(&stat, AC_W, &creds_other).is_err());

        // check==0 always succeeds
        assert!(access_check(&stat, AC_F, &creds_other).is_ok());
    }

    #[test]
    fn at_fd_helper() {
        assert_eq!(at_fd(AT_FDCWD).unwrap(), None);
        assert_eq!(at_fd(5).unwrap(), Some(5));
        assert!(at_fd(-5).is_err());
    }
}

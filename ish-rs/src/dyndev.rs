//! `fs/dyndev.h` — dynamic device registration.

use crate::dev::{DEV_CHAR, DevT};

/// Dynamic device registration result.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum DynDevError {
    Exists,
    Invalid,
}

/// Dynamic device entry.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DynDev {
    pub major: u32,
    pub minor: u32,
    pub dev_type: u32,
}

impl DynDev {
    pub fn new(major: u32, minor: u32, dev_type: u32) -> Result<Self, DynDevError> {
        if major != 240 {
            return Err(DynDevError::Invalid);
        }
        if minor > 255 {
            return Err(DynDevError::Invalid);
        }
        if dev_type != DEV_CHAR {
            return Err(DynDevError::Invalid);
        }
        Ok(Self {
            major,
            minor,
            dev_type,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn dyndev_registration() {
        assert!(DynDev::new(240, 0, DEV_CHAR).is_ok());
        assert_eq!(DynDev::new(1, 0, DEV_CHAR).unwrap_err(), DynDevError::Invalid);
        assert_eq!(DynDev::new(240, 300, DEV_CHAR).unwrap_err(), DynDevError::Invalid);
    }
}

//! `kernel/ipc.c` — the legacy System V IPC multiplexor stub.
//!
//! iSH deliberately implements none of the old i386 `ipc(2)` subcalls. The C
//! function logs its six arguments when syscall tracing is enabled, then always
//! returns `_ENOSYS`; keeping the full signature documents the ABI while making
//! the no-state behavior explicit.

use crate::task::Addr;

/// `_ENOSYS` from `kernel/errno.h`.
pub const ENOSYS: i32 = -38;

/// `sys_ipc`.
///
/// Every legacy IPC operation, including unknown selector values and arbitrary
/// pointer/argument bit patterns, is rejected by upstream iSH with `_ENOSYS`.
pub fn sys_ipc(_call: u32, _first: i32, _second: i32, _third: i32, _ptr: Addr, _fifth: i32) -> i32 {
    ENOSYS
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn every_legacy_ipc_selector_is_an_enosys_stub() {
        assert_eq!(sys_ipc(0, 0, 0, 0, 0, 0), ENOSYS);
        assert_eq!(
            sys_ipc(u32::MAX, i32::MIN, i32::MAX, -1, u32::MAX, i32::MIN),
            ENOSYS
        );
    }
}

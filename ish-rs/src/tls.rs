//! `kernel/tls.c` — i386 TLS setup.
//!
//! Linux's `set_thread_area` normally creates an x86 segment descriptor.  iSH
//! has always used a smaller emulation: it records the descriptor's base in
//! `cpu.tls_ptr`, and its GS-addressing gadgets add that value to a guest
//! memory reference.  This module keeps that exact ABI boundary, including the
//! write-back of an automatically selected entry number.

use crate::getset::EFAULT;
use crate::task::{Addr, Pid, Task};

/// The i386 `struct user_desc` as iSH lays it out in guest memory.
///
/// The final C bitfield is an `unsigned int`; none of its seven named bits is
/// interpreted by iSH, but its 32-bit storage must survive the read/modify/write
/// round trip.  Making that word explicit also pins the guest structure size at
/// 16 bytes on every host Rust supports.
#[repr(C)]
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct UserDesc {
    /// `entry_number`
    pub entry_number: u32,
    /// `base_addr`
    pub base_addr: u32,
    /// `limit`
    pub limit: u32,
    /// Storage for `seg_32bit`, `contents`, `read_exec_only`,
    /// `limit_in_pages`, `seg_not_present`, and `useable` plus the unused bits.
    pub flags: u32,
}

impl UserDesc {
    /// The exact i386 little-endian guest encoding, independent of the host.
    pub const SIZE: usize = 16;

    pub fn from_le_bytes(bytes: [u8; Self::SIZE]) -> Self {
        Self {
            entry_number: u32::from_le_bytes(bytes[0..4].try_into().unwrap()),
            base_addr: u32::from_le_bytes(bytes[4..8].try_into().unwrap()),
            limit: u32::from_le_bytes(bytes[8..12].try_into().unwrap()),
            flags: u32::from_le_bytes(bytes[12..16].try_into().unwrap()),
        }
    }

    pub fn to_le_bytes(self) -> [u8; Self::SIZE] {
        let mut bytes = [0u8; Self::SIZE];
        bytes[0..4].copy_from_slice(&self.entry_number.to_le_bytes());
        bytes[4..8].copy_from_slice(&self.base_addr.to_le_bytes());
        bytes[8..12].copy_from_slice(&self.limit.to_le_bytes());
        bytes[12..16].copy_from_slice(&self.flags.to_le_bytes());
        bytes
    }
}

/// `task_set_thread_area`.
///
/// The C writes `cpu.tls_ptr` immediately after the guest read and before its
/// descriptor write-back.  Keep that sequencing: a read-only mapping can let
/// the descriptor be read, update TLS, then make the final write fault.
pub fn task_set_thread_area(task: &mut Task, u_info: Addr) -> i32 {
    let mut bytes = [0u8; UserDesc::SIZE];
    if task.user_read(u_info, &mut bytes).is_err() {
        return EFAULT;
    }
    let mut info = UserDesc::from_le_bytes(bytes);
    task.cpu.tls_ptr = info.base_addr;
    if info.entry_number == u32::MAX {
        info.entry_number = 0x0c;
    }
    if task.user_write(u_info, &info.to_le_bytes()).is_err() {
        return EFAULT;
    }
    0
}

/// `sys_set_thread_area`.
pub fn sys_set_thread_area(task: &mut Task, u_info: Addr) -> i32 {
    task_set_thread_area(task, u_info)
}

/// `sys_set_tid_address`.
///
/// The return is `sys_getpid()`, i.e. the task's thread-group ID rather than
/// its individual thread ID.
pub fn sys_set_tid_address(task: &mut Task, tid: Addr) -> Pid {
    task.clear_tid = tid;
    task.tgid
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::memory::{P_READ, P_RWX};
    use crate::mmu::PAGE_BITS;
    use crate::task::TaskTable;

    fn task_with_page(flags: u32) -> Task {
        let mut table = TaskTable::bootstrap();
        let task = table.current_mut().unwrap();
        task.mm_mut().unwrap().mem.map_nothing(0x100, 1, flags);
        // Clone the state manually via a child-free local task. The Rc-owned mm
        // lets this isolated syscall test keep the same mapped memory.
        Task {
            cpu: task.cpu.clone(),
            mm: task.mm.clone(),
            pid: task.pid,
            tgid: task.tgid,
            group_id: task.group_id,
            credentials: task.credentials,
            ngroups: task.ngroups,
            groups: task.groups,
            comm: task.comm,
            did_exec: task.did_exec,
            parent: task.parent,
            children: task.children.clone(),
            blocked: task.blocked,
            pending: task.pending,
            waiting: task.waiting,
            saved_mask: task.saved_mask,
            has_saved_mask: task.has_saved_mask,
            clear_tid: task.clear_tid,
            robust_list: task.robust_list,
            exit_code: task.exit_code,
            zombie: task.zombie,
            exiting: task.exiting,
        }
    }

    #[test]
    fn user_desc_is_the_16_byte_little_endian_guest_layout() {
        assert_eq!(core::mem::size_of::<UserDesc>(), UserDesc::SIZE);
        let desc = UserDesc {
            entry_number: u32::MAX,
            base_addr: 0x1122_3344,
            limit: 0x5566_7788,
            flags: 0xaabb_ccdd,
        };
        assert_eq!(
            desc.to_le_bytes(),
            [
                0xff, 0xff, 0xff, 0xff, 0x44, 0x33, 0x22, 0x11, 0x88, 0x77, 0x66, 0x55, 0xdd, 0xcc,
                0xbb, 0xaa,
            ]
        );
        assert_eq!(UserDesc::from_le_bytes(desc.to_le_bytes()), desc);
    }

    #[test]
    fn automatic_entry_number_is_written_back_and_base_becomes_tls() {
        let mut task = task_with_page(P_RWX);
        let addr = 0x100 << PAGE_BITS;
        let desc = UserDesc {
            entry_number: u32::MAX,
            base_addr: 0x7f00_1000,
            limit: 0x1234_5678,
            flags: 0xffff_ff80,
        };
        task.user_write(addr, &desc.to_le_bytes()).unwrap();
        assert_eq!(sys_set_thread_area(&mut task, addr), 0);
        assert_eq!(task.cpu.tls_ptr, 0x7f00_1000);
        let mut bytes = [0u8; UserDesc::SIZE];
        task.user_read(addr, &mut bytes).unwrap();
        let got = UserDesc::from_le_bytes(bytes);
        assert_eq!(got.entry_number, 0x0c);
        assert_eq!(got.base_addr, desc.base_addr);
        assert_eq!(got.limit, desc.limit);
        assert_eq!(got.flags, desc.flags);
    }

    #[test]
    fn explicit_entry_number_is_not_rewritten() {
        let mut task = task_with_page(P_RWX);
        let addr = 0x100 << PAGE_BITS;
        let desc = UserDesc {
            entry_number: 9,
            base_addr: 0xabc,
            limit: 0,
            flags: 0,
        };
        task.user_write(addr, &desc.to_le_bytes()).unwrap();
        assert_eq!(task_set_thread_area(&mut task, addr), 0);
        let mut bytes = [0; UserDesc::SIZE];
        task.user_read(addr, &mut bytes).unwrap();
        assert_eq!(UserDesc::from_le_bytes(bytes).entry_number, 9);
    }

    #[test]
    fn a_read_only_descriptor_still_updates_tls_before_writeback_faults() {
        let mut task = task_with_page(P_RWX);
        let addr = 0x100 << PAGE_BITS;
        let desc = UserDesc {
            entry_number: u32::MAX,
            base_addr: 0xdead_beef,
            limit: 0,
            flags: 0,
        };
        task.user_write(addr, &desc.to_le_bytes()).unwrap();
        // Change the mapping after initialization. Unlike a ptrace write this
        // leaves the page truly read-only for the syscall's final user_write.
        assert_eq!(task.mm_mut().unwrap().mem.set_flags(0x100, 1, P_READ), 0);
        assert_eq!(task_set_thread_area(&mut task, addr), EFAULT);
        assert_eq!(task.cpu.tls_ptr, 0xdead_beef);
        let mut bytes = [0; UserDesc::SIZE];
        task.user_read(addr, &mut bytes).unwrap();
        assert_eq!(
            UserDesc::from_le_bytes(bytes),
            desc,
            "write-back did not happen"
        );
    }

    #[test]
    fn set_tid_address_keeps_clear_tid_and_returns_tgid() {
        let mut task = task_with_page(P_RWX);
        task.pid = 44;
        task.tgid = 12;
        assert_eq!(sys_set_tid_address(&mut task, 0xbeef_0000), 12);
        assert_eq!(task.clear_tid, 0xbeef_0000);
    }
}

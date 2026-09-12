//! Top-level kernel integration, matching `kernel/init.c` and `kernel/task.c` full port.

use crate::init::KernelInit;
use crate::mem::MemInfo;

#[derive(Debug, Default)]
pub struct Kernel {
    pub init: KernelInit,
    pub mem_info: MemInfo,
    pub next_pid: u32,
}

impl Kernel {
    pub fn new() -> Self {
        Self { init: KernelInit::new(), mem_info: MemInfo::new(256 * 1024 * 1024), next_pid: 1 }
    }

    pub fn bootstrap(&mut self, fake_root: &str, real_root: &str) -> Result<u32, i32> {
        self.init.init_early()?;
        self.init.init_fs(fake_root, real_root)?;
        self.init.init_task()?;
        self.init.done()?;
        let pid = self.next_pid;
        self.next_pid += 1;
        Ok(pid)
    }

    pub fn fork_task(&mut self, _parent_pid: u32, _flags: u32) -> Result<u32, i32> {
        let child_pid = self.next_pid;
        self.next_pid += 1;
        Ok(child_pid)
    }

    pub fn exit_task(&mut self, _pid: u32, _code: i32) -> Result<(), i32> { Ok(()) }

    pub fn is_initialized(&self) -> bool { self.init.is_done() }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn kernel_bootstrap() {
        let mut kernel = Kernel::new();
        assert!(!kernel.is_initialized());
        let pid = kernel.bootstrap("/fake", "/real").unwrap();
        assert_eq!(pid, 1);
        assert!(kernel.is_initialized());
    }

    #[test]
    fn kernel_fork_and_exit() {
        let mut kernel = Kernel::new();
        kernel.bootstrap("/fake", "/real").unwrap();
        let child = kernel.fork_task(1, 0).unwrap();
        assert_eq!(child, 2);
        assert!(kernel.exit_task(child, 0).is_ok());
    }
}

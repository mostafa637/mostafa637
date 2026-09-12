//! Alpine rootfs runner — integration of all modules needed to boot Alpine.

use crate::exec::{ExecArgs, ExecContext};
use crate::fake::FakeFs;
use crate::mm::{MmStruct, VmArea, PROT_READ, PROT_WRITE, PROT_EXEC, MAP_PRIVATE};
use crate::kernel::Kernel;

#[derive(Debug)]
pub struct AlpineRunner {
    pub fakefs: FakeFs,
    pub mm: MmStruct,
    pub kernel: Kernel,
    pub exec_ctx: ExecContext,
}

impl Default for AlpineRunner {
    fn default() -> Self {
        Self { fakefs: FakeFs::new(), mm: MmStruct::new(), kernel: Kernel::new(), exec_ctx: ExecContext::new() }
    }
}

impl AlpineRunner {
    pub fn new() -> Self { Self::default() }

    pub fn init(&mut self, fake_root: &str, real_root: &str) -> Result<u32, i32> {
        self.fakefs.load_alpine_mock();
        self.kernel.bootstrap(fake_root, real_root)
    }

    pub fn load_binary(&mut self, path: &str) -> Result<(), i32> {
        self.exec_ctx.load_from_fakefs(&self.fakefs, path)
    }

    pub fn setup_memory(&mut self) -> Result<(), i32> {
        for ph in &self.exec_ctx.phdrs {
            if ph.is_load() {
                let start = ph.vaddr & !0xfff;
                let end = (ph.vaddr + ph.memsz + 0xfff) & !0xfff;
                let prot = {
                    let mut p = 0;
                    if (ph.flags & 4) != 0 { p |= PROT_READ; }
                    if (ph.flags & 2) != 0 { p |= PROT_WRITE; }
                    if (ph.flags & 1) != 0 { p |= PROT_EXEC; }
                    p
                };
                let area = VmArea::new(start, end, prot, MAP_PRIVATE);
                let _ = self.mm.add_area(area);
            }
        }
        self.mm.stack_start = 0xbffff000;
        self.mm.brk_start = 0x08000000;
        self.mm.brk = self.mm.brk_start;
        Ok(())
    }

    pub fn exec(&mut self, path: &str, args: Vec<String>, env: Vec<String>) -> Result<u32, i32> {
        self.load_binary(path)?;
        self.setup_memory()?;
        let exec_args = ExecArgs::new(args);
        let env_args = ExecArgs::new(env);
        let _stack = self.exec_ctx.setup_stack(&exec_args, &env_args, self.mm.stack_start)?;
        Ok(self.exec_ctx.entry)
    }

    pub fn run_alpine_sh(&mut self) -> Result<String, i32> {
        self.init("/fake", "/real")?;
        let entry = self.exec("/bin/busybox", vec!["/bin/sh".to_string(), "-c".to_string(), "echo hello from Alpine".to_string()], vec!["PATH=/bin".to_string()])?;
        Ok(format!("Alpine /bin/sh loaded at entry {:#x}, stack at {:#x}, {} pages mapped", entry, self.exec_ctx.stack_top, self.mm.areas.len()))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn alpine_runner_init() {
        let mut runner = AlpineRunner::new();
        let pid = runner.init("/fake", "/real").unwrap();
        assert_eq!(pid, 1);
        assert!(runner.kernel.is_initialized());
        assert!(runner.fakefs.path_get_inode("/bin/sh") != 0);
    }

    #[test]
    fn alpine_runner_load_binary() {
        let mut runner = AlpineRunner::new();
        runner.fakefs.load_alpine_mock();
        assert!(runner.load_binary("/bin/busybox").is_ok());
        assert!(runner.exec_ctx.header.is_executable());
        assert!(runner.load_binary("/nonexistent").is_err());
    }

    #[test]
    fn alpine_runner_setup_memory() {
        let mut runner = AlpineRunner::new();
        runner.fakefs.load_alpine_mock();
        runner.load_binary("/bin/busybox").unwrap();
        runner.setup_memory().unwrap();
    }

    #[test]
    fn alpine_runner_exec() {
        let mut runner = AlpineRunner::new();
        runner.fakefs.load_alpine_mock();
        let _entry = runner.exec("/bin/busybox", vec!["/bin/sh".to_string()], vec!["PATH=/bin".to_string()]).unwrap();
    }

    #[test]
    fn alpine_run_sh() {
        let mut runner = AlpineRunner::new();
        let result = runner.run_alpine_sh().unwrap();
        assert!(result.contains("Alpine"));
        assert!(result.contains("entry"));
        println!("{}", result);
    }
}

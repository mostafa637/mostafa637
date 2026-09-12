//! Alpine rootfs runner — integration of all modules needed to boot Alpine.
//! Supports both mock and real Alpine rootfs 3.24.1 x86.

use crate::exec::{ExecArgs, ExecContext};
use crate::fake::{FakeFs, IshStat};
use crate::mm::{MmStruct, VmArea, PROT_READ, PROT_WRITE, PROT_EXEC, MAP_PRIVATE};
use crate::kernel::Kernel;
use std::path::Path;

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

    pub fn load_real_alpine(&mut self, host_path: &str) -> Result<usize, i32> {
        let path = Path::new(host_path);
        if !path.exists() { return Err(-2); }
        let mut count = 0;
        self.load_dir_recursive(path, "", &mut count)?;
        Ok(count)
    }

    fn load_dir_recursive(&mut self, base: &Path, relative: &str, count: &mut usize) -> Result<(), i32> {
        let full_path = if relative.is_empty() { base.to_path_buf() } else { base.join(relative) };
        let entries = std::fs::read_dir(&full_path).map_err(|_| -2)?;
        for entry in entries {
            let entry = entry.map_err(|_| -2)?;
            let file_name = entry.file_name().to_string_lossy().to_string();
            let rel_path = if relative.is_empty() { format!("/{}", file_name) } else { format!("{}/{}", relative, file_name) };
            let guest_path = if rel_path.starts_with('/') { rel_path.clone() } else { format!("/{}", rel_path) };
            let file_type = entry.file_type().map_err(|_| -2)?;
            if file_type.is_symlink() {
                let target = std::fs::read_link(entry.path()).map(|p| p.to_string_lossy().to_string()).unwrap_or_default();
                self.fakefs.path_create_symlink(&guest_path, &target);
                *count += 1;
            } else if file_type.is_dir() {
                self.fakefs.path_create(&guest_path, IshStat::new(0o040755, 0, 0, 0));
                self.load_dir_recursive(base, &rel_path.trim_start_matches('/'), count)?;
            } else if file_type.is_file() {
                let data = std::fs::read(entry.path()).unwrap_or_default();
                let mode = if guest_path.contains("/bin/") || guest_path.contains("/sbin/") { 0o100755 } else { 0o100644 };
                self.fakefs.path_create_file(&guest_path, IshStat::new(mode, 0, 0, 0), data);
                *count += 1;
            }
        }
        Ok(())
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
        Ok(format!("Alpine /bin/sh loaded at entry {:#x}, stack at {:#x}, {} pages mapped, {} files in fakefs", entry, self.exec_ctx.stack_top, self.mm.areas.len(), self.fakefs.inodes.len()))
    }

    pub fn boot_alpine(&mut self, real_rootfs_path: Option<&str>) -> Result<String, i32> {
        if let Some(path) = real_rootfs_path {
            match self.load_real_alpine(path) {
                Ok(count) => println!("Loaded {} files from real Alpine rootfs at {}", count, path),
                Err(_) => {
                    println!("Real Alpine rootfs not found at {}, using mock", path);
                    self.fakefs.load_alpine_mock();
                }
            }
        } else {
            self.fakefs.load_alpine_mock();
        }
        self.kernel.bootstrap("/fake", "/real")?;
        for sh_path in &["/bin/sh", "/bin/busybox", "/bin/ash"] {
            if self.fakefs.path_get_inode(sh_path) != 0 {
                let entry = self.exec(sh_path, vec![sh_path.to_string()], vec!["PATH=/bin:/usr/bin".to_string(), "HOME=/root".to_string()])?;
                return Ok(format!("Alpine boot successful: {} at {:#x}, {} files, {} memory areas", sh_path, entry, self.fakefs.inodes.len(), self.mm.areas.len()));
            }
        }
        Err(-2)
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

    #[test]
    fn alpine_boot_mock() {
        let mut runner = AlpineRunner::new();
        let result = runner.boot_alpine(None).unwrap();
        assert!(result.contains("Alpine boot successful"));
        println!("{}", result);
    }

    #[test]
    fn alpine_load_real_if_exists() {
        let mut runner = AlpineRunner::new();
        // Test with real Alpine 3.24.1 if available
        if std::path::Path::new("/tmp/alpine_real").exists() {
            let count = runner.load_real_alpine("/tmp/alpine_real").unwrap();
            println!("Loaded {} files from /tmp/alpine_real", count);
            assert!(count > 50);
            assert!(runner.fakefs.path_get_inode("/bin/sh") != 0);
            assert!(runner.fakefs.path_get_inode("/bin/busybox") != 0);
            // Test symlink resolution
            let sh_data = runner.fakefs.read_file("/bin/sh").unwrap();
            assert_eq!(&sh_data[0..4], b"\x7fELF");
            // Test ELF loading of real busybox
            assert!(runner.load_binary("/bin/busybox").is_ok());
            assert_eq!(runner.exec_ctx.header.machine, 3); // EM_386
            println!("Real Alpine busybox: entry {:#x}, {} PHDRs, {} bytes total", runner.exec_ctx.entry, runner.exec_ctx.phdrs.len(), runner.exec_ctx.total_memory());
        }
    }
}

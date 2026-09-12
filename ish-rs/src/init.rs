//! `kernel/init.c` — kernel initialization.

#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub enum InitStage {
    #[default]
    None,
    Early,
    Fs,
    Task,
    Done,
}

#[derive(Debug, Default)]
pub struct KernelInit {
    pub stage: InitStage,
    pub fs_initialized: bool,
    pub task_initialized: bool,
    pub fakefs_root: String,
    pub realfs_root: String,
}

impl KernelInit {
    pub fn new() -> Self { Self::default() }

    pub fn init_early(&mut self) -> Result<(), i32> {
        if self.stage != InitStage::None { return Err(-22); }
        self.stage = InitStage::Early;
        Ok(())
    }

    pub fn init_fs(&mut self, fake_root: &str, real_root: &str) -> Result<(), i32> {
        if self.stage != InitStage::Early { return Err(-22); }
        self.fakefs_root = fake_root.to_string();
        self.realfs_root = real_root.to_string();
        self.fs_initialized = true;
        self.stage = InitStage::Fs;
        Ok(())
    }

    pub fn init_task(&mut self) -> Result<(), i32> {
        if self.stage != InitStage::Fs { return Err(-22); }
        self.task_initialized = true;
        self.stage = InitStage::Task;
        Ok(())
    }

    pub fn done(&mut self) -> Result<(), i32> {
        if self.stage != InitStage::Task { return Err(-22); }
        self.stage = InitStage::Done;
        Ok(())
    }

    pub fn is_done(&self) -> bool { self.stage == InitStage::Done }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn init_stages() {
        let mut init = KernelInit::new();
        assert!(!init.is_done());
        assert!(init.init_early().is_ok());
        assert!(init.init_early().is_err()); // already early
        assert!(init.init_fs("/fake", "/real").is_ok());
        assert!(init.fs_initialized);
        assert!(init.init_task().is_ok());
        assert!(init.task_initialized);
        assert!(init.done().is_ok());
        assert!(init.is_done());
    }

    #[test]
    fn init_wrong_order_fails() {
        let mut init = KernelInit::new();
        assert!(init.init_fs("/fake", "/real").is_err());
        assert!(init.init_task().is_err());
        assert!(init.done().is_err());
    }
}

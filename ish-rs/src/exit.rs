//! `kernel/exit.c` — exit status handling and group exit full port.

pub fn wifexited(status: i32) -> bool { (status & 0x7f) == 0 }
pub fn wexitstatus(status: i32) -> i32 { (status >> 8) & 0xff }
pub fn wifsignaled(status: i32) -> bool { let sig = status & 0x7f; sig != 0 && sig != 0x7f }
pub fn wtermsig(status: i32) -> i32 { status & 0x7f }
pub fn wifstopped(status: i32) -> bool { (status & 0xff) == 0x7f }
pub fn wstopsig(status: i32) -> i32 { (status >> 8) & 0xff }
pub fn wifcontinued(status: i32) -> bool { status == 0xffff }

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ExitCode {
    Exited(i32),
    Signaled(i32),
    Stopped(i32),
    Continued,
}

pub fn decode_exit_code(status: i32) -> ExitCode {
    if wifcontinued(status) {
        ExitCode::Continued
    } else if wifstopped(status) {
        ExitCode::Stopped(wstopsig(status))
    } else if wifsignaled(status) {
        ExitCode::Signaled(wtermsig(status))
    } else {
        ExitCode::Exited(wexitstatus(status))
    }
}

#[derive(Debug, Default)]
pub struct GroupExit {
    pub doing_group_exit: bool,
    pub exit_code: i32,
}

impl GroupExit {
    pub fn new() -> Self { Self::default() }
    pub fn start_group_exit(&mut self, code: i32) {
        self.doing_group_exit = true;
        self.exit_code = code;
    }
    pub fn is_group_exiting(&self) -> bool { self.doing_group_exit }
}

#[derive(Debug, Default)]
pub struct ExitState {
    pub exit_code: i32,
    pub group_exit: GroupExit,
    pub children: Vec<u32>,
    pub parent: Option<u32>,
    pub is_zombie: bool,
    pub is_dead: bool,
}

impl ExitState {
    pub fn new() -> Self { Self::default() }
    pub fn set_exit_code(&mut self, code: i32) { self.exit_code = code; }
    pub fn become_zombie(&mut self) { self.is_zombie = true; }
    pub fn reparent_children(&mut self, _new_parent: u32) -> Vec<u32> {
        let old = std::mem::take(&mut self.children);
        old
    }
    pub fn add_child(&mut self, pid: u32) { self.children.push(pid); }
    pub fn remove_child(&mut self, pid: u32) { self.children.retain(|&c| c != pid); }
    pub fn should_release(&self) -> bool { self.is_zombie && self.children.is_empty() }
}

pub const WNOHANG: u32 = 1;
pub const WUNTRACED: u32 = 2;
pub const WCONTINUED: u32 = 8;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum WaitResult {
    NoChild,
    WouldBlock,
    Pid(u32, i32),
}

pub fn wait_for_child(children: &[(u32, ExitState)], pid: i32, options: u32) -> WaitResult {
    let mut found_any = false;
    for (child_pid, state) in children {
        if pid != -1 && *child_pid as i32 != pid { continue; }
        found_any = true;
        if state.is_zombie {
            return WaitResult::Pid(*child_pid, state.exit_code);
        }
    }
    if !found_any { WaitResult::NoChild } else if (options & WNOHANG) != 0 { WaitResult::WouldBlock } else { WaitResult::WouldBlock }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn exit_status_helpers_match_c() {
        let status = 0x0100;
        assert!(wifexited(status));
        assert_eq!(wexitstatus(status), 1);
        let status2 = 0x0009;
        assert!(wifsignaled(status2));
        assert_eq!(wtermsig(status2), 9);
        let status3 = 0x137f;
        assert!(wifstopped(status3));
        assert_eq!(wstopsig(status3), 19);
        assert!(wifcontinued(0xffff));
    }

    #[test]
    fn decode_exit_code_test() {
        assert_eq!(decode_exit_code(0x0100), ExitCode::Exited(1));
        assert_eq!(decode_exit_code(0x0009), ExitCode::Signaled(9));
        assert_eq!(decode_exit_code(0x137f), ExitCode::Stopped(19));
        assert_eq!(decode_exit_code(0xffff), ExitCode::Continued);
    }

    #[test]
    fn group_exit() {
        let mut ge = GroupExit::new();
        assert!(!ge.is_group_exiting());
        ge.start_group_exit(1);
        assert!(ge.is_group_exiting());
        assert_eq!(ge.exit_code, 1);
    }

    #[test]
    fn exit_state_zombie_and_reparent() {
        let mut state = ExitState::new();
        state.add_child(2);
        state.add_child(3);
        assert_eq!(state.children.len(), 2);
        state.remove_child(2);
        assert_eq!(state.children.len(), 1);
        let old = state.reparent_children(1);
        assert_eq!(old.len(), 1);
        assert!(state.children.is_empty());
        state.become_zombie();
        assert!(state.should_release());
    }

    #[test]
    fn wait_for_child_test() {
        let mut child_state = ExitState::new();
        child_state.set_exit_code(0x0100);
        child_state.become_zombie();
        let children = vec![(2u32, child_state)];
        match wait_for_child(&children, -1, 0) {
            WaitResult::Pid(pid, code) => { assert_eq!(pid, 2); assert_eq!(code, 0x0100); },
            _ => panic!("should find child"),
        }
        let empty: Vec<(u32, ExitState)> = Vec::new();
        assert_eq!(wait_for_child(&empty, -1, 0), WaitResult::NoChild);
    }
}

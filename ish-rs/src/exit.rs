//! `kernel/exit.c` — exit status handling and group exit.

pub fn wifexited(status: i32) -> bool { (status & 0x7f) == 0 }
pub fn wexitstatus(status: i32) -> i32 { (status >> 8) & 0xff }
pub fn wifsignaled(status: i32) -> bool { let sig = status & 0x7f; sig != 0 && sig != 0x7f }
pub fn wtermsig(status: i32) -> i32 { status & 0x7f }
pub fn wifstopped(status: i32) -> bool { (status & 0xff) == 0x7f }
pub fn wstopsig(status: i32) -> i32 { (status >> 8) & 0xff }
pub fn wifcontinued(status: i32) -> bool { status == 0xffff }

/// Exit code from wait
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

/// Group exit state
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
}

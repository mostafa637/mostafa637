//! `kernel/exit.c` — exit status handling.

/// Exit code helpers, matching C's `WIFEXITED`, `WEXITSTATUS`, etc.

pub fn wifexited(status: i32) -> bool {
    (status & 0x7f) == 0
}

pub fn wexitstatus(status: i32) -> i32 {
    (status >> 8) & 0xff
}

pub fn wifsignaled(status: i32) -> bool {
    let sig = status & 0x7f;
    sig != 0 && sig != 0x7f
}

pub fn wtermsig(status: i32) -> i32 {
    status & 0x7f
}

pub fn wifstopped(status: i32) -> bool {
    (status & 0xff) == 0x7f
}

pub fn wstopsig(status: i32) -> i32 {
    (status >> 8) & 0xff
}

/// Exit hook type, matching C's `void (*exit_hook)(struct task *, int)`.
pub type ExitHook = fn(pid: u32, code: i32);

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn exit_status_helpers_match_c() {
        let status = 0x0100; // exit 1
        assert!(wifexited(status));
        assert_eq!(wexitstatus(status), 1);

        let status2 = 0x0009; // killed by SIGKILL (9)
        assert!(wifsignaled(status2));
        assert_eq!(wtermsig(status2), 9);

        let status3 = 0x137f; // stopped by SIGSTOP (19)
        assert!(wifstopped(status3));
        assert_eq!(wstopsig(status3), 19);
    }
}

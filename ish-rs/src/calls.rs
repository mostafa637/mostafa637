//! `kernel/calls.h` + `kernel/calls.c` — syscall numbers and dispatch.

/// i386 syscall numbers from `calls.c` table.
pub const SYS_EXIT: u32 = 1;
pub const SYS_FORK: u32 = 2;
pub const SYS_READ: u32 = 3;
pub const SYS_WRITE: u32 = 4;
pub const SYS_OPEN: u32 = 5;
pub const SYS_CLOSE: u32 = 6;
pub const SYS_WAITPID: u32 = 7;
pub const SYS_LINK: u32 = 9;
pub const SYS_UNLINK: u32 = 10;
pub const SYS_EXECVE: u32 = 11;
pub const SYS_CHDIR: u32 = 12;
pub const SYS_TIME: u32 = 13;
pub const SYS_MKNOD: u32 = 14;
pub const SYS_CHMOD: u32 = 15;
pub const SYS_LSEEK: u32 = 19;
pub const SYS_GETPID: u32 = 20;
pub const SYS_MOUNT: u32 = 21;
pub const SYS_SETUID: u32 = 23;
pub const SYS_GETUID: u32 = 24;
pub const SYS_STIME: u32 = 25;
pub const SYS_PTRACE: u32 = 26;
pub const SYS_ALARM: u32 = 27;
pub const SYS_PAUSE: u32 = 29;
pub const SYS_UTIME: u32 = 30;
pub const SYS_ACCESS: u32 = 33;
pub const SYS_KILL: u32 = 37;
pub const SYS_RENAME: u32 = 38;
pub const SYS_MKDIR: u32 = 39;
pub const SYS_RMDIR: u32 = 40;
pub const SYS_DUP: u32 = 41;
pub const SYS_PIPE: u32 = 42;
pub const SYS_TIMES: u32 = 43;
pub const SYS_BRK: u32 = 45;
pub const SYS_SETGID: u32 = 46;
pub const SYS_GETGID: u32 = 47;
pub const SYS_GETEUID: u32 = 49;
pub const SYS_GETEGID: u32 = 50;
pub const SYS_UMOUNT2: u32 = 52;
pub const SYS_IOCTL: u32 = 54;
pub const SYS_FCNTL32: u32 = 55;
pub const SYS_SETPGID: u32 = 57;
pub const SYS_UMASK: u32 = 60;
pub const SYS_CHROOT: u32 = 61;
pub const SYS_DUP2: u32 = 63;
pub const SYS_GETPPID: u32 = 64;
pub const SYS_GETPGRP: u32 = 65;
pub const SYS_SETSID: u32 = 66;
pub const SYS_SETHOSTNAME: u32 = 74;
pub const SYS_SETRLIMIT32: u32 = 75;
pub const SYS_OLD_GETRLIMIT32: u32 = 76;
pub const SYS_GETRUSAGE: u32 = 77;
pub const SYS_GETTIMEOFDAY: u32 = 78;
pub const SYS_SETTIMEOFDAY: u32 = 79;
pub const SYS_GETGROUPS_OLD: u32 = 80;
pub const SYS_SETGROUPS_OLD: u32 = 81;
pub const SYS_SYMLINK: u32 = 83;
pub const SYS_READLINK: u32 = 85;
pub const SYS_REBOOT: u32 = 88;
pub const SYS_MMAP: u32 = 90;
pub const SYS_MUNMAP: u32 = 91;
pub const SYS_FCHMOD: u32 = 94;
pub const SYS_GETPRIORITY: u32 = 96;
pub const SYS_SETPRIORITY: u32 = 97;
pub const SYS_STATFS: u32 = 99;
pub const SYS_FSTATFS: u32 = 100;
pub const SYS_SOCKETCALL: u32 = 102;
pub const SYS_SYSLOG: u32 = 103;
pub const SYS_SETITIMER: u32 = 104;
pub const SYS_WAIT4: u32 = 114;
pub const SYS_SYSINFO: u32 = 116;
pub const SYS_IPC: u32 = 117;
pub const SYS_FSYNC: u32 = 118;
pub const SYS_SIGRETURN: u32 = 119;
pub const SYS_CLONE: u32 = 120;
pub const SYS_UNAME: u32 = 122;
pub const SYS_MPROTECT: u32 = 125;
pub const SYS_GETPGID: u32 = 132;
pub const SYS_FCHDIR: u32 = 133;
pub const SYS_PERSONALITY: u32 = 136;
pub const SYS_LLSEEK: u32 = 140;
pub const SYS_GETDENTS: u32 = 141;
pub const SYS_SELECT: u32 = 142;
pub const SYS_FLOCK: u32 = 143;
pub const SYS_MSYNC: u32 = 144;
pub const SYS_READV: u32 = 145;
pub const SYS_WRITEV: u32 = 146;
pub const SYS_GETSID: u32 = 147;
pub const SYS_FDATASYNC: u32 = 148;
pub const SYS_MLOCK: u32 = 150;
pub const SYS_SCHED_GETPARAM: u32 = 155;
pub const SYS_SCHED_SETSCHEDULER: u32 = 156;
pub const SYS_SCHED_GETSCHEDULER: u32 = 157;
pub const SYS_SCHED_YIELD: u32 = 158;
pub const SYS_SCHED_GET_PRIORITY_MAX: u32 = 159;
pub const SYS_NANOSLEEP: u32 = 162;
pub const SYS_MREMAP: u32 = 163;
pub const SYS_POLL: u32 = 168;
pub const SYS_PRCTL: u32 = 172;
pub const SYS_RT_SIGRETURN: u32 = 173;
pub const SYS_RT_SIGACTION: u32 = 174;
pub const SYS_RT_SIGPROCMASK: u32 = 175;
pub const SYS_RT_SIGPENDING: u32 = 176;
pub const SYS_RT_SIGTIMEDWAIT: u32 = 177;
pub const SYS_RT_SIGSUSPEND: u32 = 179;
pub const SYS_PREAD: u32 = 180;
pub const SYS_PWRITE: u32 = 181;
pub const SYS_GETCWD: u32 = 183;
pub const SYS_CAPGET: u32 = 184;
pub const SYS_CAPSET: u32 = 185;
pub const SYS_SIGALTSTACK: u32 = 186;
pub const SYS_SENDFILE: u32 = 187;
pub const SYS_VFORK: u32 = 190;
pub const SYS_GETRLIMIT32: u32 = 191;
pub const SYS_MMAP2: u32 = 192;
pub const SYS_TRUNCATE64: u32 = 193;
pub const SYS_FTRUNCATE64: u32 = 194;
pub const SYS_STAT64: u32 = 195;
pub const SYS_LSTAT64: u32 = 196;
pub const SYS_FSTAT64: u32 = 197;
pub const SYS_LCHOWN: u32 = 198;
pub const SYS_GETUID32: u32 = 199;
pub const SYS_GETGID32: u32 = 200;
pub const SYS_GETEUID32: u32 = 201;
pub const SYS_GETEGID32: u32 = 202;
pub const SYS_SETREUID: u32 = 203;
pub const SYS_SETREGID: u32 = 204;
pub const SYS_GETGROUPS32: u32 = 205;
pub const SYS_SETGROUPS32: u32 = 206;
pub const SYS_FCHOWN32: u32 = 207;
pub const SYS_SETRESUID: u32 = 208;
pub const SYS_GETRESUID: u32 = 209;
pub const SYS_SETRESGID: u32 = 210;
pub const SYS_GETRESGID: u32 = 211;
pub const SYS_CHOWN32: u32 = 212;
pub const SYS_SETUID32: u32 = 213;
pub const SYS_SETGID32: u32 = 214;
pub const SYS_MADVISE: u32 = 219;
pub const SYS_GETDENTS64: u32 = 220;
pub const SYS_FCNTL64: u32 = 221;
pub const SYS_GETTID: u32 = 224;
pub const SYS_TKILL: u32 = 238;
pub const SYS_SENDFILE64: u32 = 239;
pub const SYS_FUTEX: u32 = 240;
pub const SYS_SCHED_SETAFFINITY: u32 = 241;
pub const SYS_SCHED_GETAFFINITY: u32 = 242;
pub const SYS_SET_THREAD_AREA: u32 = 243;
pub const SYS_EXIT_GROUP: u32 = 252;
pub const SYS_EPOLL_CREATE: u32 = 254;
pub const SYS_EPOLL_CTL: u32 = 255;
pub const SYS_EPOLL_WAIT: u32 = 256;
pub const SYS_SET_TID_ADDRESS: u32 = 258;
pub const SYS_TIMER_CREATE: u32 = 259;
pub const SYS_TIMER_SETTIME: u32 = 260;
pub const SYS_TIMER_DELETE: u32 = 263;
pub const SYS_CLOCK_SETTIME: u32 = 264;
pub const SYS_CLOCK_GETTIME: u32 = 265;
pub const SYS_CLOCK_GETRES: u32 = 266;
pub const SYS_STATFS64: u32 = 268;
pub const SYS_FSTATFS64: u32 = 269;
pub const SYS_TGKILL: u32 = 270;
pub const SYS_UTIMES: u32 = 271;
pub const SYS_MBIND: u32 = 274;
pub const SYS_WAITID: u32 = 284;
pub const SYS_IOPRIO_SET: u32 = 289;
pub const SYS_IOPRIO_GET: u32 = 290;
pub const SYS_OPENAT: u32 = 295;
pub const SYS_MKDIRAT: u32 = 296;
pub const SYS_MKNODAT: u32 = 297;
pub const SYS_FCHOWNAT: u32 = 298;
pub const SYS_FSTATAT64: u32 = 300;
pub const SYS_UNLINKAT: u32 = 301;
pub const SYS_RENAMEAT: u32 = 302;
pub const SYS_LINKAT: u32 = 303;
pub const SYS_SYMLINKAT: u32 = 304;
pub const SYS_READLINKAT: u32 = 305;
pub const SYS_FCHMODAT: u32 = 306;
pub const SYS_FACCESSAT: u32 = 307;
pub const SYS_PSELECT: u32 = 308;
pub const SYS_PPOLL: u32 = 309;
pub const SYS_SET_ROBUST_LIST: u32 = 311;
pub const SYS_GET_ROBUST_LIST: u32 = 312;
pub const SYS_SPLICE: u32 = 313;
pub const SYS_EPOLL_PWAIT: u32 = 319;
pub const SYS_UTIMENSAT: u32 = 320;
pub const SYS_TIMERFD_CREATE: u32 = 322;
pub const SYS_EVENTFD: u32 = 323;
pub const SYS_FALLOCATE: u32 = 324;
pub const SYS_TIMERFD_SETTIME: u32 = 325;
pub const SYS_EVENTFD2: u32 = 328;
pub const SYS_EPOLL_CREATE1: u32 = 329;
pub const SYS_DUP3: u32 = 330;
pub const SYS_PIPE2: u32 = 331;
pub const SYS_PRLIMIT64: u32 = 340;
pub const SYS_SENDMMSG: u32 = 345;
pub const SYS_RENAMEAT2: u32 = 353;
pub const SYS_GETRANDOM: u32 = 355;
pub const SYS_SOCKET: u32 = 359;
pub const SYS_SOCKETPAIR: u32 = 360;
pub const SYS_BIND: u32 = 361;

pub const SYS_GETRLIMIT: u32 = SYS_GETRLIMIT32;
pub const SYS_GETGROUPS: u32 = SYS_GETGROUPS32;
pub const SYS_SETGROUPS: u32 = SYS_SETGROUPS32;

/// Syscall dispatch result.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Syscall {
    Unknown(u32),
    Exit,
    Fork,
    Read,
    Write,
    Open,
    Close,
    WaitPid,
    Execve,
    Time,
    GetPid,
    Brk,
    Mmap,
    Mmap2,
    Munmap,
    Mprotect,
    Uname,
    GetTimeOfDay,
    GetRandom,
}

pub fn syscall_from_number(nr: u32) -> Syscall {
    match nr {
        SYS_EXIT => Syscall::Exit,
        SYS_FORK => Syscall::Fork,
        SYS_READ => Syscall::Read,
        SYS_WRITE => Syscall::Write,
        SYS_OPEN => Syscall::Open,
        SYS_CLOSE => Syscall::Close,
        SYS_WAITPID => Syscall::WaitPid,
        SYS_EXECVE => Syscall::Execve,
        SYS_TIME => Syscall::Time,
        SYS_GETPID => Syscall::GetPid,
        SYS_BRK => Syscall::Brk,
        SYS_MMAP => Syscall::Mmap,
        SYS_MMAP2 => Syscall::Mmap2,
        SYS_MUNMAP => Syscall::Munmap,
        SYS_MPROTECT => Syscall::Mprotect,
        SYS_UNAME => Syscall::Uname,
        SYS_GETTIMEOFDAY => Syscall::GetTimeOfDay,
        SYS_GETRANDOM => Syscall::GetRandom,
        _ => Syscall::Unknown(nr),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn syscall_numbers_match_c_table() {
        assert_eq!(SYS_EXIT, 1);
        assert_eq!(SYS_READ, 3);
        assert_eq!(SYS_WRITE, 4);
        assert_eq!(SYS_OPEN, 5);
        assert_eq!(SYS_CLOSE, 6);
        assert_eq!(SYS_TIME, 13);
        assert_eq!(SYS_GETPID, 20);
        assert_eq!(SYS_BRK, 45);
        assert_eq!(SYS_MMAP, 90);
        assert_eq!(SYS_MMAP2, 192);
        assert_eq!(SYS_GETRANDOM, 355);
    }

    #[test]
    fn syscall_dispatch_known() {
        assert_eq!(syscall_from_number(1), Syscall::Exit);
        assert_eq!(syscall_from_number(4), Syscall::Write);
        assert!(matches!(syscall_from_number(9999), Syscall::Unknown(9999)));
    }
}

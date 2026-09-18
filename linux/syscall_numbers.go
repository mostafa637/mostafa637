// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Generated from src/guest_abi.h — the AArch64 syscall numbers (the generic
// asm-generic/unistd.h table, which is what arm64 uses).

package linux

// AArch64 syscall numbers. The guest passes the number in x8 and the arguments
// in x0..x5; see Dispatch.
const (
	Syssetxattr                = 5
	Syslsetxattr               = 6
	Sysfsetxattr               = 7
	Sysgetxattr                = 8
	Syslgetxattr               = 9
	Sysfgetxattr               = 10
	Syslistxattr               = 11
	Sysllistxattr              = 12
	Sysflistxattr              = 13
	Sysremovexattr             = 14
	Syslremovexattr            = 15
	Sysfremovexattr            = 16
	Sysgetcwd                  = 17
	Syseventfd2                = 19
	Sysepoll_create1           = 20
	Sysepoll_ctl               = 21
	Sysepoll_pwait             = 22
	Sysdup                     = 23
	Sysdup3                    = 24
	Sysfcntl                   = 25
	Sysinotify_init1           = 26
	Sysinotify_add_watch       = 27
	Sysinotify_rm_watch        = 28
	Sysioctl                   = 29
	Sysflock                   = 32
	Sysmknodat                 = 33
	Sysmkdirat                 = 34
	Sysunlinkat                = 35
	Syssymlinkat               = 36
	Syslinkat                  = 37
	Sysrenameat                = 38
	Sysumount2                 = 39
	Sysmount                   = 40
	Syspivot_root              = 41
	Sysstatfs                  = 43
	Sysfstatfs                 = 44
	Systruncate                = 45
	Sysftruncate               = 46
	Sysfallocate               = 47
	Sysfaccessat               = 48
	Syschdir                   = 49
	Sysfchdir                  = 50
	Syschroot                  = 51
	Sysfchmod                  = 52
	Sysfchmodat                = 53
	Sysfchownat                = 54
	Sysfchown                  = 55
	Sysopenat                  = 56
	Sysclose                   = 57
	Sysvhangup                 = 58
	Syspipe2                   = 59
	Sysquotactl                = 60
	Sysgetdents64              = 61
	Syslseek                   = 62
	Sysread                    = 63
	Syswrite                   = 64
	Sysreadv                   = 65
	Syswritev                  = 66
	Syspread64                 = 67
	Syspwrite64                = 68
	Syspreadv                  = 69
	Syspwritev                 = 70
	Syssendfile                = 71
	Syspselect6                = 72
	Sysppoll                   = 73
	Syssignalfd4               = 74
	Sysvmsplice                = 75
	Syssplice                  = 76
	Systee                     = 77
	Sysreadlinkat              = 78
	Sysnewfstatat              = 79
	Sysfstat                   = 80
	Syssync                    = 81
	Sysfsync                   = 82
	Sysfdatasync               = 83
	Syssync_file_range         = 84
	Systimerfd_create          = 85
	Systimerfd_settime         = 86
	Systimerfd_gettime         = 87
	Sysutimensat               = 88
	Sysacct                    = 89
	Syscapget                  = 90
	Syscapset                  = 91
	Syspersonality             = 92
	Sysexit                    = 93
	Sysexit_group              = 94
	Syswaitid                  = 95
	Sysset_tid_address         = 96
	Sysunshare                 = 97
	Sysfutex                   = 98
	Sysset_robust_list         = 99
	Sysget_robust_list         = 100
	Sysnanosleep               = 101
	Sysgetitimer               = 102
	Syssetitimer               = 103
	Syskexec_load              = 104
	Sysinit_module             = 105
	Sysdelete_module           = 106
	Systimer_create            = 107
	Systimer_gettime           = 108
	Systimer_getoverrun        = 109
	Systimer_settime           = 110
	Systimer_delete            = 111
	Sysclock_settime           = 112
	Sysclock_gettime           = 113
	Sysclock_getres            = 114
	Sysclock_nanosleep         = 115
	Syssyslog                  = 116
	Sysptrace                  = 117
	Syssched_setparam          = 118
	Syssched_setscheduler      = 119
	Syssched_getscheduler      = 120
	Syssched_getparam          = 121
	Syssched_setaffinity       = 122
	Syssched_getaffinity       = 123
	Syssched_yield             = 124
	Syssched_get_priority_max  = 125
	Syssched_get_priority_min  = 126
	Syssched_rr_get_interval   = 127
	Sysrestart_syscall         = 128
	Syskill                    = 129
	Systkill                   = 130
	Systgkill                  = 131
	Syssigaltstack             = 132
	Sysrt_sigsuspend           = 133
	Sysrt_sigaction            = 134
	Sysrt_sigprocmask          = 135
	Sysrt_sigpending           = 136
	Sysrt_sigtimedwait         = 137
	Sysrt_sigqueueinfo         = 138
	Sysrt_sigreturn            = 139
	Syssetpriority             = 140
	Sysgetpriority             = 141
	Sysreboot                  = 142
	Syssetregid                = 143
	Syssetgid                  = 144
	Syssetreuid                = 145
	Syssetuid                  = 146
	Syssetresuid               = 147
	Sysgetresuid               = 148
	Syssetresgid               = 149
	Sysgetresgid               = 150
	Syssetfsuid                = 151
	Syssetfsgid                = 152
	Systimes                   = 153
	Syssetpgid                 = 154
	Sysgetpgid                 = 155
	Sysgetsid                  = 156
	Syssetsid                  = 157
	Sysgetgroups               = 158
	Syssetgroups               = 159
	Sysuname                   = 160
	Syssethostname             = 161
	Syssetdomainname           = 162
	Sysgetrlimit               = 163
	Syssetrlimit               = 164
	Sysgetrusage               = 165
	Sysumask                   = 166
	Sysprctl                   = 167
	Sysgetcpu                  = 168
	Sysgettimeofday            = 169
	Syssettimeofday            = 170
	Sysadjtimex                = 171
	Sysgetpid                  = 172
	Sysgetppid                 = 173
	Sysgetuid                  = 174
	Sysgeteuid                 = 175
	Sysgetgid                  = 176
	Sysgetegid                 = 177
	Sysgettid                  = 178
	Syssysinfo                 = 179
	Sysmq_open                 = 180
	Sysmsgget                  = 186
	Sysmsgctl                  = 187
	Sysmsgrcv                  = 188
	Sysmsgsnd                  = 189
	Syssemget                  = 190
	Syssemctl                  = 191
	Syssemtimedop              = 192
	Syssemop                   = 193
	Sysshmget                  = 194
	Sysshmctl                  = 195
	Sysshmat                   = 196
	Sysshmdt                   = 197
	Syssocket                  = 198
	Syssocketpair              = 199
	Sysbind                    = 200
	Syslisten                  = 201
	Sysaccept                  = 202
	Sysconnect                 = 203
	Sysgetsockname             = 204
	Sysgetpeername             = 205
	Syssendto                  = 206
	Sysrecvfrom                = 207
	Syssetsockopt              = 208
	Sysgetsockopt              = 209
	Sysshutdown                = 210
	Syssendmsg                 = 211
	Sysrecvmsg                 = 212
	Sysreadahead               = 213
	Sysbrk                     = 214
	Sysmunmap                  = 215
	Sysmremap                  = 216
	Sysadd_key                 = 217
	Sysrequest_key             = 218
	Syskeyctl                  = 219
	Sysclone                   = 220
	Sysexecve                  = 221
	Sysmmap                    = 222
	Sysfadvise64               = 223
	Sysswapon                  = 224
	Sysswapoff                 = 225
	Sysmprotect                = 226
	Sysmsync                   = 227
	Sysmlock                   = 228
	Sysmunlock                 = 229
	Sysmlockall                = 230
	Sysmunlockall              = 231
	Sysmincore                 = 232
	Sysmadvise                 = 233
	Sysremap_file_pages        = 234
	Sysmbind                   = 235
	Sysget_mempolicy           = 236
	Sysset_mempolicy           = 237
	Sysaccept4                 = 242
	Sysrecvmmsg                = 243
	Syswait4                   = 260
	Sysprlimit64               = 261
	Sysfanotify_init           = 262
	Sysname_to_handle_at       = 264
	Sysclock_adjtime           = 266
	Syssyncfs                  = 267
	Syssetns                   = 268
	Syssendmmsg                = 269
	Sysprocess_vm_readv        = 270
	Sysprocess_vm_writev       = 271
	Syskcmp                    = 272
	Sysfinit_module            = 273
	Syssched_setattr           = 274
	Syssched_getattr           = 275
	Sysrenameat2               = 276
	Sysseccomp                 = 277
	Sysgetrandom               = 278
	Sysmemfd_create            = 279
	Sysbpf                     = 280
	Sysexecveat                = 281
	Sysuserfaultfd             = 282
	Sysmembarrier              = 283
	Sysmlock2                  = 284
	Syscopy_file_range         = 285
	Syspreadv2                 = 286
	Syspwritev2                = 287
	Syspkey_mprotect           = 288
	Sysstatx                   = 291
	Sysio_pgetevents           = 292
	Sysrseq                    = 293
	Syskexec_file_load         = 294
	Syspidfd_send_signal       = 424
	Sysio_uring_setup          = 425
	Sysio_uring_enter          = 426
	Sysio_uring_register       = 427
	Sysopen_tree               = 428
	Sysmove_mount              = 429
	Sysfsopen                  = 430
	Sysfsconfig                = 431
	Sysfsmount                 = 432
	Sysfspick                  = 433
	Syspidfd_open              = 434
	Sysclone3                  = 435
	Sysclose_range             = 436
	Sysopenat2                 = 437
	Syspidfd_getfd             = 438
	Sysfaccessat2              = 439
	Sysprocess_madvise         = 440
	Sysepoll_pwait2            = 441
	Sysmount_setattr           = 442
	Syslandlock_create_ruleset = 444
	Sysmemfd_secret            = 447
	Sysprocess_mrelease        = 448
	Sysfutex_waitv             = 449
	Sysset_mempolicy_home_node = 450
	Syscachestat               = 451
	Sysfchmodat2               = 452
	Sysmap_shadow_stack        = 453
	Sysfutex_wake              = 454
	Sysfutex_wait              = 455
	Sysfutex_requeue           = 456
	Sysstatmount               = 457
	Syslistmount               = 458
	Syslsm_get_self_attr       = 459
	Syslsm_set_self_attr       = 460
	Syslsm_list_modules        = 461
	Sysmseal                   = 462
	SysMAX                     = 512
)

// syscallNames maps a number back to its name for strace and for the
// one-shot "unimplemented" warning.
var syscallNames = map[uint64]string{
	5:   "setxattr",
	6:   "lsetxattr",
	7:   "fsetxattr",
	8:   "getxattr",
	9:   "lgetxattr",
	10:  "fgetxattr",
	11:  "listxattr",
	12:  "llistxattr",
	13:  "flistxattr",
	14:  "removexattr",
	15:  "lremovexattr",
	16:  "fremovexattr",
	17:  "getcwd",
	19:  "eventfd2",
	20:  "epoll_create1",
	21:  "epoll_ctl",
	22:  "epoll_pwait",
	23:  "dup",
	24:  "dup3",
	25:  "fcntl",
	26:  "inotify_init1",
	27:  "inotify_add_watch",
	28:  "inotify_rm_watch",
	29:  "ioctl",
	32:  "flock",
	33:  "mknodat",
	34:  "mkdirat",
	35:  "unlinkat",
	36:  "symlinkat",
	37:  "linkat",
	38:  "renameat",
	39:  "umount2",
	40:  "mount",
	41:  "pivot_root",
	43:  "statfs",
	44:  "fstatfs",
	45:  "truncate",
	46:  "ftruncate",
	47:  "fallocate",
	48:  "faccessat",
	49:  "chdir",
	50:  "fchdir",
	51:  "chroot",
	52:  "fchmod",
	53:  "fchmodat",
	54:  "fchownat",
	55:  "fchown",
	56:  "openat",
	57:  "close",
	58:  "vhangup",
	59:  "pipe2",
	60:  "quotactl",
	61:  "getdents64",
	62:  "lseek",
	63:  "read",
	64:  "write",
	65:  "readv",
	66:  "writev",
	67:  "pread64",
	68:  "pwrite64",
	69:  "preadv",
	70:  "pwritev",
	71:  "sendfile",
	72:  "pselect6",
	73:  "ppoll",
	74:  "signalfd4",
	75:  "vmsplice",
	76:  "splice",
	77:  "tee",
	78:  "readlinkat",
	79:  "newfstatat",
	80:  "fstat",
	81:  "sync",
	82:  "fsync",
	83:  "fdatasync",
	84:  "sync_file_range",
	85:  "timerfd_create",
	86:  "timerfd_settime",
	87:  "timerfd_gettime",
	88:  "utimensat",
	89:  "acct",
	90:  "capget",
	91:  "capset",
	92:  "personality",
	93:  "exit",
	94:  "exit_group",
	95:  "waitid",
	96:  "set_tid_address",
	97:  "unshare",
	98:  "futex",
	99:  "set_robust_list",
	100: "get_robust_list",
	101: "nanosleep",
	102: "getitimer",
	103: "setitimer",
	104: "kexec_load",
	105: "init_module",
	106: "delete_module",
	107: "timer_create",
	108: "timer_gettime",
	109: "timer_getoverrun",
	110: "timer_settime",
	111: "timer_delete",
	112: "clock_settime",
	113: "clock_gettime",
	114: "clock_getres",
	115: "clock_nanosleep",
	116: "syslog",
	117: "ptrace",
	118: "sched_setparam",
	119: "sched_setscheduler",
	120: "sched_getscheduler",
	121: "sched_getparam",
	122: "sched_setaffinity",
	123: "sched_getaffinity",
	124: "sched_yield",
	125: "sched_get_priority_max",
	126: "sched_get_priority_min",
	127: "sched_rr_get_interval",
	128: "restart_syscall",
	129: "kill",
	130: "tkill",
	131: "tgkill",
	132: "sigaltstack",
	133: "rt_sigsuspend",
	134: "rt_sigaction",
	135: "rt_sigprocmask",
	136: "rt_sigpending",
	137: "rt_sigtimedwait",
	138: "rt_sigqueueinfo",
	139: "rt_sigreturn",
	140: "setpriority",
	141: "getpriority",
	142: "reboot",
	143: "setregid",
	144: "setgid",
	145: "setreuid",
	146: "setuid",
	147: "setresuid",
	148: "getresuid",
	149: "setresgid",
	150: "getresgid",
	151: "setfsuid",
	152: "setfsgid",
	153: "times",
	154: "setpgid",
	155: "getpgid",
	156: "getsid",
	157: "setsid",
	158: "getgroups",
	159: "setgroups",
	160: "uname",
	161: "sethostname",
	162: "setdomainname",
	163: "getrlimit",
	164: "setrlimit",
	165: "getrusage",
	166: "umask",
	167: "prctl",
	168: "getcpu",
	169: "gettimeofday",
	170: "settimeofday",
	171: "adjtimex",
	172: "getpid",
	173: "getppid",
	174: "getuid",
	175: "geteuid",
	176: "getgid",
	177: "getegid",
	178: "gettid",
	179: "sysinfo",
	180: "mq_open",
	186: "msgget",
	187: "msgctl",
	188: "msgrcv",
	189: "msgsnd",
	190: "semget",
	191: "semctl",
	192: "semtimedop",
	193: "semop",
	194: "shmget",
	195: "shmctl",
	196: "shmat",
	197: "shmdt",
	198: "socket",
	199: "socketpair",
	200: "bind",
	201: "listen",
	202: "accept",
	203: "connect",
	204: "getsockname",
	205: "getpeername",
	206: "sendto",
	207: "recvfrom",
	208: "setsockopt",
	209: "getsockopt",
	210: "shutdown",
	211: "sendmsg",
	212: "recvmsg",
	213: "readahead",
	214: "brk",
	215: "munmap",
	216: "mremap",
	217: "add_key",
	218: "request_key",
	219: "keyctl",
	220: "clone",
	221: "execve",
	222: "mmap",
	223: "fadvise64",
	224: "swapon",
	225: "swapoff",
	226: "mprotect",
	227: "msync",
	228: "mlock",
	229: "munlock",
	230: "mlockall",
	231: "munlockall",
	232: "mincore",
	233: "madvise",
	234: "remap_file_pages",
	235: "mbind",
	236: "get_mempolicy",
	237: "set_mempolicy",
	242: "accept4",
	243: "recvmmsg",
	260: "wait4",
	261: "prlimit64",
	262: "fanotify_init",
	264: "name_to_handle_at",
	266: "clock_adjtime",
	267: "syncfs",
	268: "setns",
	269: "sendmmsg",
	270: "process_vm_readv",
	271: "process_vm_writev",
	272: "kcmp",
	273: "finit_module",
	274: "sched_setattr",
	275: "sched_getattr",
	276: "renameat2",
	277: "seccomp",
	278: "getrandom",
	279: "memfd_create",
	280: "bpf",
	281: "execveat",
	282: "userfaultfd",
	283: "membarrier",
	284: "mlock2",
	285: "copy_file_range",
	286: "preadv2",
	287: "pwritev2",
	288: "pkey_mprotect",
	291: "statx",
	292: "io_pgetevents",
	293: "rseq",
	294: "kexec_file_load",
	424: "pidfd_send_signal",
	425: "io_uring_setup",
	426: "io_uring_enter",
	427: "io_uring_register",
	428: "open_tree",
	429: "move_mount",
	430: "fsopen",
	431: "fsconfig",
	432: "fsmount",
	433: "fspick",
	434: "pidfd_open",
	435: "clone3",
	436: "close_range",
	437: "openat2",
	438: "pidfd_getfd",
	439: "faccessat2",
	440: "process_madvise",
	441: "epoll_pwait2",
	442: "mount_setattr",
	444: "landlock_create_ruleset",
	447: "memfd_secret",
	448: "process_mrelease",
	449: "futex_waitv",
	450: "set_mempolicy_home_node",
	451: "cachestat",
	452: "fchmodat2",
	453: "map_shadow_stack",
	454: "futex_wake",
	455: "futex_wait",
	456: "futex_requeue",
	457: "statmount",
	458: "listmount",
	459: "lsm_get_self_attr",
	460: "lsm_set_self_attr",
	461: "lsm_list_modules",
	462: "mseal",
	512: "MAX",
}

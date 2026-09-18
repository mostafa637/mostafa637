// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Sylirre
// Ported to Go from the errno values the C emulator's guest-visible ABI uses
// (src/sys.h and the target_errno table in src/syscall.c).

// Package abi holds the guest-visible ABI definitions the syscall layer and
// the memory layer share: Linux errno values and the struct layouts a guest
// passes by pointer (see guest_abi.h in the C sources).
package abi

import "strconv"

// Errno is a Linux error number. The emulator is its own kernel, so it returns
// these to the guest as negative return values, exactly as Linux does.
type Errno uint32

// Error implements the error interface so the Go layers above mem/ can use
// them directly while still handing the number to the guest.
func (e Errno) Error() string {
	if s, ok := names[e]; ok {
		return s
	}
	return "errno " + strconv.Itoa(int(e))
}

// The generic Linux errno values (asm-generic/errno-base.h and errno.h).
const (
	EPERM           Errno = 1
	ENOENT          Errno = 2
	ESRCH           Errno = 3
	EINTR           Errno = 4
	EIO             Errno = 5
	ENXIO           Errno = 6
	E2BIG           Errno = 7
	ENOEXEC         Errno = 8
	EBADF           Errno = 9
	ECHILD          Errno = 10
	EAGAIN          Errno = 11
	ENOMEM          Errno = 12
	EACCES          Errno = 13
	EFAULT          Errno = 14
	ENOTBLK         Errno = 15
	EBUSY           Errno = 16
	EEXIST          Errno = 17
	EXDEV           Errno = 18
	ENODEV          Errno = 19
	ENOTDIR         Errno = 20
	EISDIR          Errno = 21
	EINVAL          Errno = 22
	ENFILE          Errno = 23
	EMFILE          Errno = 24
	ENOTTY          Errno = 25
	ETXTBSY         Errno = 26
	EFBIG           Errno = 27
	ENOSPC          Errno = 28
	ESPIPE          Errno = 29
	EROFS           Errno = 30
	EMLINK          Errno = 31
	EPIPE           Errno = 32
	EDOM            Errno = 33
	ERANGE          Errno = 34
	EDEADLK         Errno = 35
	ENAMETOOLONG    Errno = 36
	ENOLCK          Errno = 37
	ENOSYS          Errno = 38
	ENOTEMPTY       Errno = 39
	ELOOP           Errno = 40
	EWOULDBLOCK           = EAGAIN
	ENOMSG          Errno = 42
	EIDRM           Errno = 43
	ECHRNG          Errno = 44
	EL3HLT          Errno = 46
	EL3RST          Errno = 47
	ELNRNG          Errno = 48
	EUNATCH         Errno = 49
	ENOCSI          Errno = 50
	EL2HLT          Errno = 51
	EBADE           Errno = 52
	EBADR           Errno = 53
	EXFULL          Errno = 54
	ENOANO          Errno = 55
	EBADRQC         Errno = 56
	EBADSLT         Errno = 57
	EDEADLOCK             = EDEADLK
	EBFONT          Errno = 59
	ENOSTR          Errno = 60
	ENODATA         Errno = 61
	ETIME           Errno = 62
	ENOSR           Errno = 63
	ENONET          Errno = 64
	ENOPKG          Errno = 65
	EREMOTE         Errno = 66
	ENOLINK         Errno = 67
	EADV            Errno = 68
	ESRMNT          Errno = 69
	ECOMM           Errno = 70
	EPROTO          Errno = 71
	EMULTIHOP       Errno = 72
	EDOTDOT         Errno = 73
	EBADMSG         Errno = 74
	EOVERFLOW       Errno = 75
	ENOTUNIQ        Errno = 76
	EBADFD          Errno = 77
	EREMCHG         Errno = 78
	ELIBACC         Errno = 79
	ELIBBAD         Errno = 80
	ELIBSCN         Errno = 81
	ELIBMAX         Errno = 82
	ELIBEXEC        Errno = 83
	EILSEQ          Errno = 84
	ERESTART        Errno = 85
	ESTRPIPE        Errno = 86
	EUSERS          Errno = 87
	ENOTSOCK        Errno = 88
	EDESTADDRREQ    Errno = 89
	EMSGSIZE        Errno = 90
	EPROTOTYPE      Errno = 91
	ENOPROTOOPT     Errno = 92
	EPROTONOSUPPORT Errno = 93
	ESOCKTNOSUPPORT Errno = 94
	EOPNOTSUPP      Errno = 95
	EPFNOSUPPORT    Errno = 96
	EAFNOSUPPORT    Errno = 97
	EADDRINUSE      Errno = 98
	EADDRNOTAVAIL   Errno = 99
	ENETDOWN        Errno = 100
	ENETUNREACH     Errno = 101
	ENETRESET       Errno = 102
	ECONNABORTED    Errno = 103
	ECONNRESET      Errno = 104
	ENOBUFS         Errno = 105
	EISCONN         Errno = 106
	ENOTCONN        Errno = 107
	ESHUTDOWN       Errno = 108
	ETOOMANYREFS    Errno = 109
	ETIMEDOUT       Errno = 110
	ECONNREFUSED    Errno = 111
	EHOSTDOWN       Errno = 112
	EHOSTUNREACH    Errno = 113
	EALREADY        Errno = 114
	EINPROGRESS     Errno = 115
	ESTALE          Errno = 116
	EUCLEAN         Errno = 117
	ENOTNAM         Errno = 118
	ENAVAIL         Errno = 119
	EISNAM          Errno = 120
	EREMOTEIO       Errno = 121
	EDQUOT          Errno = 122
	ENOMEDIUM       Errno = 123
	EMEDIUMTYPE     Errno = 124
	ECANCELED       Errno = 125
	ENOKEY          Errno = 126
	EKEYEXPIRED     Errno = 127
	EKEYREVOKED     Errno = 128
	EKEYREJECTED    Errno = 129
	EOWNERDEAD      Errno = 130
	ENOTRECOVERABLE Errno = 131
	ERFKILL         Errno = 132
	EHWPOISON       Errno = 133
)

var names = map[Errno]string{
	EPERM: "EPERM", ENOENT: "ENOENT", ESRCH: "ESRCH", EINTR: "EINTR",
	EIO: "EIO", ENXIO: "ENXIO", E2BIG: "E2BIG", ENOEXEC: "ENOEXEC",
	EBADF: "EBADF", ECHILD: "ECHILD", EAGAIN: "EAGAIN", ENOMEM: "ENOMEM",
	EACCES: "EACCES", EFAULT: "EFAULT", ENOTBLK: "ENOTBLK", EBUSY: "EBUSY",
	EEXIST: "EEXIST", EXDEV: "EXDEV", ENODEV: "ENODEV", ENOTDIR: "ENOTDIR",
	EISDIR: "EISDIR", EINVAL: "EINVAL", ENFILE: "ENFILE", EMFILE: "EMFILE",
	ENOTTY: "ENOTTY", ETXTBSY: "ETXTBSY", EFBIG: "EFBIG", ENOSPC: "ENOSPC",
	ESPIPE: "ESPIPE", EROFS: "EROFS", EMLINK: "EMLINK", EPIPE: "EPIPE",
	EDOM: "EDOM", ERANGE: "ERANGE", ENAMETOOLONG: "ENAMETOOLONG",
	ENOLCK: "ENOLCK", ENOSYS: "ENOSYS", ENOTEMPTY: "ENOTEMPTY", ELOOP: "ELOOP",
	ENOMSG: "ENOMSG", EBADMSG: "EBADMSG", EOVERFLOW: "EOVERFLOW",
	EILSEQ: "EILSEQ", ENOTSOCK: "ENOTSOCK", EDESTADDRREQ: "EDESTADDRREQ",
	EMSGSIZE: "EMSGSIZE", EPROTOTYPE: "EPROTOTYPE", ENOPROTOOPT: "ENOPROTOOPT",
	EPROTONOSUPPORT: "EPROTONOSUPPORT", ESOCKTNOSUPPORT: "ESOCKTNOSUPPORT",
	EOPNOTSUPP: "EOPNOTSUPP", EAFNOSUPPORT: "EAFNOSUPPORT",
	EADDRINUSE: "EADDRINUSE", EADDRNOTAVAIL: "EADDRNOTAVAIL",
	ENETDOWN: "ENETDOWN", ENETUNREACH: "ENETUNREACH", ENETRESET: "ENETRESET",
	ECONNABORTED: "ECONNABORTED", ECONNRESET: "ECONNRESET", ENOBUFS: "ENOBUFS",
	EISCONN: "EISCONN", ENOTCONN: "ENOTCONN", ETIMEDOUT: "ETIMEDOUT",
	ECONNREFUSED: "ECONNREFUSED", EHOSTUNREACH: "EHOSTUNREACH",
	EALREADY: "EALREADY", EINPROGRESS: "EINPROGRESS", ESTALE: "ESTALE",
	EDQUOT: "EDQUOT", ECANCELED: "ECANCELED",
}

// ENOTSUP is EOPNOTSUPP under a second name (the two are the same value in
// the generic table; the guest sees one number either way).
const ENOTSUP = EOPNOTSUPP

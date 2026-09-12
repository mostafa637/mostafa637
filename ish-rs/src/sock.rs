//! `fs/sock.h` + `fs/sock.c` — socket constants, ABIs, and helpers.

pub const AF_UNSPEC: u32 = 0;
pub const AF_UNIX: u32 = 1;
pub const AF_LOCAL: u32 = 1;
pub const AF_INET: u32 = 2;
pub const AF_INET6: u32 = 10;

pub const SOCK_STREAM: u32 = 1;
pub const SOCK_DGRAM: u32 = 2;
pub const SOCK_RAW: u32 = 3;
pub const SOCK_SEQPACKET: u32 = 5;

pub const IPPROTO_IP: u32 = 0;
pub const IPPROTO_ICMP: u32 = 1;
pub const IPPROTO_TCP: u32 = 6;
pub const IPPROTO_UDP: u32 = 17;
pub const IPPROTO_RAW: u32 = 255;

pub const SOL_SOCKET: u32 = 1;
pub const SO_REUSEADDR: u32 = 2;
pub const SO_TYPE: u32 = 3;
pub const SO_ERROR: u32 = 4;
pub const SO_KEEPALIVE: u32 = 9;

pub const SOCKADDR_DATA_MAX: usize = 108;
pub const SOCKET_TYPE_MASK: u32 = 0xf;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Sockaddr {
    pub family: u16,
    pub data: [u8; 14],
}
impl Default for Sockaddr {
    fn default() -> Self { Self { family: 0, data: [0; 14] } }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SockaddrMax {
    pub family: u16,
    pub data: [u8; SOCKADDR_DATA_MAX],
}
impl Default for SockaddrMax {
    fn default() -> Self { Self { family: 0, data: [0; SOCKADDR_DATA_MAX] } }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct Msghdr {
    pub msg_name: u32,
    pub msg_namelen: u32,
    pub msg_iov: u32,
    pub msg_iovlen: u32,
    pub msg_control: u32,
    pub msg_controllen: u32,
    pub msg_flags: u32,
}

pub fn sockaddr_size(family: u16) -> usize {
    match family as u32 {
        AF_UNIX => 2 + 108,
        AF_INET => 16,
        AF_INET6 => 28,
        _ => 14,
    }
}

pub fn is_valid_sock_type(sock_type: u32) -> bool {
    let base = sock_type & SOCKET_TYPE_MASK;
    matches!(base, SOCK_STREAM | SOCK_DGRAM | SOCK_RAW | SOCK_SEQPACKET)
}

pub fn sock_family_to_real(family: u32) -> Result<i32, i32> {
    match family {
        AF_UNSPEC => Ok(0),
        AF_UNIX => Ok(1),
        AF_INET => Ok(2),
        AF_INET6 => Ok(10),
        _ => Err(-22),
    }
}

pub fn sock_type_to_real(sock_type: u32, _protocol: u32) -> Result<i32, i32> {
    let base = sock_type & SOCKET_TYPE_MASK;
    match base {
        SOCK_STREAM => Ok(1),
        SOCK_DGRAM => Ok(2),
        SOCK_RAW => Ok(3),
        SOCK_SEQPACKET => Ok(5),
        _ => Err(-22),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn sock_constants_match_c() {
        assert_eq!(AF_UNIX, 1);
        assert_eq!(AF_INET, 2);
        assert_eq!(SOCK_STREAM, 1);
        assert_eq!(SOCKADDR_DATA_MAX, 108);
        assert_eq!(SOL_SOCKET, 1);
        assert_eq!(SOCKET_TYPE_MASK, 0xf);
    }

    #[test]
    fn sockaddr_size_matches_c() {
        assert_eq!(sockaddr_size(AF_INET as u16), 16);
        assert_eq!(sockaddr_size(AF_UNIX as u16), 110);
    }

    #[test]
    fn valid_sock_type() {
        assert!(is_valid_sock_type(SOCK_STREAM));
        assert!(is_valid_sock_type(SOCK_DGRAM | 0x80000)); // with flags
        assert!(is_valid_sock_type(99)); // 99 & 0xf = 3 = SOCK_RAW, valid per C masking
        assert!(!is_valid_sock_type(4)); // base 4 invalid
        assert!(!is_valid_sock_type(100)); // 100 & 0xf = 4 invalid
    }

    #[test]
    fn family_and_type_to_real() {
        assert_eq!(sock_family_to_real(AF_INET).unwrap(), 2);
        assert!(sock_family_to_real(99).is_err());
        assert_eq!(sock_type_to_real(SOCK_STREAM, 0).unwrap(), 1);
        assert_eq!(sock_type_to_real(99, 0).unwrap(), 3); // 99 masked to 3 = RAW
        assert!(sock_type_to_real(4, 0).is_err());
    }
}

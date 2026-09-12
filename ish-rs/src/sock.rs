//! `fs/sock.h` — socket constants, guest ABIs, and address helpers.

/// Socket families
pub const AF_UNSPEC: u32 = 0;
pub const AF_UNIX: u32 = 1;
pub const AF_INET: u32 = 2;
pub const AF_INET6: u32 = 10;

/// Socket types
pub const SOCK_STREAM: u32 = 1;
pub const SOCK_DGRAM: u32 = 2;
pub const SOCK_RAW: u32 = 3;
pub const SOCK_SEQPACKET: u32 = 5;

/// Protocols
pub const IPPROTO_IP: u32 = 0;
pub const IPPROTO_TCP: u32 = 6;
pub const IPPROTO_UDP: u32 = 17;

/// Socket options
pub const SOL_SOCKET: u32 = 1;
pub const SO_REUSEADDR: u32 = 2;
pub const SO_TYPE: u32 = 3;
pub const SO_ERROR: u32 = 4;
pub const SO_KEEPALIVE: u32 = 9;

/// `SOCKADDR_DATA_MAX`
pub const SOCKADDR_DATA_MAX: usize = 108;

/// `struct sockaddr_` guest ABI.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Sockaddr {
    pub family: u16,
    pub data: [u8; 14],
}

impl Default for Sockaddr {
    fn default() -> Self {
        Self { family: 0, data: [0; 14] }
    }
}

/// `struct sockaddr_max_` guest ABI.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SockaddrMax {
    pub family: u16,
    pub data: [u8; SOCKADDR_DATA_MAX],
}

impl Default for SockaddrMax {
    fn default() -> Self {
        Self { family: 0, data: [0; SOCKADDR_DATA_MAX] }
    }
}

/// `struct msghdr_` guest ABI
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

/// Socket address size helper, matching C `sockaddr_size`
pub fn sockaddr_size(family: u16) -> usize {
    match family as u32 {
        AF_UNIX => 2 + 108, // sun_path max
        AF_INET => 16,
        AF_INET6 => 28,
        _ => 14,
    }
}

/// Check if socket type is valid.
pub fn is_valid_sock_type(sock_type: u32) -> bool {
    matches!(sock_type, SOCK_STREAM | SOCK_DGRAM | SOCK_RAW | SOCK_SEQPACKET)
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
    }

    #[test]
    fn sockaddr_size_matches_c() {
        assert_eq!(sockaddr_size(AF_INET as u16), 16);
        assert_eq!(sockaddr_size(AF_UNIX as u16), 110);
    }

    #[test]
    fn valid_sock_type() {
        assert!(is_valid_sock_type(SOCK_STREAM));
        assert!(is_valid_sock_type(SOCK_DGRAM));
        assert!(!is_valid_sock_type(99));
    }
}

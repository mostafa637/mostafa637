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
pub const SO_SNDBUF: u32 = 7;
pub const SO_RCVBUF: u32 = 8;
pub const SO_LINGER: u32 = 13;
pub const SO_RCVTIMEO: u32 = 20;
pub const SO_SNDTIMEO: u32 = 21;

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

/// Unix socket address helpers
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct UnixAddr {
    pub path: String,
    pub is_abstract: bool,
}

impl UnixAddr {
    pub fn new(path: impl Into<String>) -> Self {
        let p = path.into();
        let is_abstract = p.starts_with('\0');
        Self { path: p, is_abstract }
    }
    pub fn is_unnamed(&self) -> bool { self.path.is_empty() }
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

/// Socket state, matching C's sock state machine
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub enum SocketState {
    #[default]
    Unbound,
    Bound,
    Listening,
    Connecting,
    Connected,
    Closed,
}

#[derive(Debug, Default)]
pub struct Socket {
    pub family: u32,
    pub sock_type: u32,
    pub protocol: u32,
    pub state: SocketState,
    pub bound_addr: Option<UnixAddr>,
    pub sndbuf: u32,
    pub rcvbuf: u32,
    pub reuseaddr: bool,
}

impl Socket {
    pub fn new(family: u32, sock_type: u32, protocol: u32) -> Result<Self, i32> {
        sock_family_to_real(family)?;
        sock_type_to_real(sock_type, protocol)?;
        Ok(Self {
            family,
            sock_type: sock_type & SOCKET_TYPE_MASK,
            protocol,
            state: SocketState::Unbound,
            bound_addr: None,
            sndbuf: 212992,
            rcvbuf: 212992,
            reuseaddr: false,
        })
    }

    pub fn bind(&mut self, addr: UnixAddr) -> Result<(), i32> {
        if self.state != SocketState::Unbound {
            return Err(-22); // EINVAL
        }
        self.bound_addr = Some(addr);
        self.state = SocketState::Bound;
        Ok(())
    }

    pub fn listen(&mut self, _backlog: i32) -> Result<(), i32> {
        if self.state != SocketState::Bound {
            return Err(-22);
        }
        self.state = SocketState::Listening;
        Ok(())
    }

    pub fn connect(&mut self, addr: UnixAddr) -> Result<(), i32> {
        if self.state == SocketState::Listening {
            return Err(-22);
        }
        self.bound_addr = Some(addr);
        self.state = SocketState::Connected;
        Ok(())
    }

    pub fn setsockopt(&mut self, level: u32, optname: u32, optval: u32) -> Result<(), i32> {
        if level != SOL_SOCKET {
            return Err(-92); // ENOPROTOOPT
        }
        match optname {
            SO_REUSEADDR => { self.reuseaddr = optval != 0; Ok(()) },
            SO_SNDBUF => { self.sndbuf = optval; Ok(()) },
            SO_RCVBUF => { self.rcvbuf = optval; Ok(()) },
            _ => Err(-92),
        }
    }

    pub fn getsockopt(&self, level: u32, optname: u32) -> Result<u32, i32> {
        if level != SOL_SOCKET {
            return Err(-92);
        }
        match optname {
            SO_TYPE => Ok(self.sock_type),
            SO_ERROR => Ok(0),
            SO_REUSEADDR => Ok(if self.reuseaddr { 1 } else { 0 }),
            SO_SNDBUF => Ok(self.sndbuf),
            SO_RCVBUF => Ok(self.rcvbuf),
            _ => Err(-92),
        }
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
        assert!(is_valid_sock_type(SOCK_DGRAM | 0x80000));
        assert!(is_valid_sock_type(99));
        assert!(!is_valid_sock_type(4));
        assert!(!is_valid_sock_type(100));
    }

    #[test]
    fn family_and_type_to_real() {
        assert_eq!(sock_family_to_real(AF_INET).unwrap(), 2);
        assert!(sock_family_to_real(99).is_err());
        assert_eq!(sock_type_to_real(SOCK_STREAM, 0).unwrap(), 1);
        assert_eq!(sock_type_to_real(99, 0).unwrap(), 3);
        assert!(sock_type_to_real(4, 0).is_err());
    }

    #[test]
    fn unix_addr_helpers() {
        let addr = UnixAddr::new("/tmp/socket");
        assert!(!addr.is_abstract);
        assert!(!addr.is_unnamed());
        let abstract_addr = UnixAddr::new("\0abstract");
        assert!(abstract_addr.is_abstract);
        let unnamed = UnixAddr::new("");
        assert!(unnamed.is_unnamed());
    }

    #[test]
    fn socket_state_machine() {
        let mut sock = Socket::new(AF_UNIX, SOCK_STREAM, 0).unwrap();
        assert_eq!(sock.state, SocketState::Unbound);
        sock.bind(UnixAddr::new("/tmp/test")).unwrap();
        assert_eq!(sock.state, SocketState::Bound);
        sock.listen(5).unwrap();
        assert_eq!(sock.state, SocketState::Listening);
        assert!(sock.connect(UnixAddr::new("/tmp/other")).is_err());
    }

    #[test]
    fn socket_options() {
        let mut sock = Socket::new(AF_INET, SOCK_STREAM, 0).unwrap();
        assert_eq!(sock.getsockopt(SOL_SOCKET, SO_TYPE).unwrap(), SOCK_STREAM);
        assert_eq!(sock.getsockopt(SOL_SOCKET, SO_ERROR).unwrap(), 0);
        sock.setsockopt(SOL_SOCKET, SO_REUSEADDR, 1).unwrap();
        assert_eq!(sock.getsockopt(SOL_SOCKET, SO_REUSEADDR).unwrap(), 1);
        sock.setsockopt(SOL_SOCKET, SO_SNDBUF, 4096).unwrap();
        assert_eq!(sock.getsockopt(SOL_SOCKET, SO_SNDBUF).unwrap(), 4096);
        assert!(sock.getsockopt(SOL_SOCKET, 9999).is_err());
    }
}

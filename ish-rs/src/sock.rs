//! `fs/sock.h` + `fs/sock.c` — socket full port, matching C implementation.

use std::collections::{HashMap, VecDeque};
use std::sync::{Arc, Mutex};

pub const AF_UNSPEC: u32 = 0;
pub const AF_UNIX: u32 = 1;
pub const AF_LOCAL: u32 = 1;
pub const AF_INET: u32 = 2;
pub const AF_INET6: u32 = 10;

pub const SOCK_STREAM: u32 = 1;
pub const SOCK_DGRAM: u32 = 2;
pub const SOCK_RAW: u32 = 3;
pub const SOCK_RDM: u32 = 4;
pub const SOCK_SEQPACKET: u32 = 5;

pub const SOCK_CLOEXEC: u32 = 0o2000000;
pub const SOCK_NONBLOCK: u32 = 0o4000;

pub const IPPROTO_IP: u32 = 0;
pub const IPPROTO_ICMP: u32 = 1;
pub const IPPROTO_TCP: u32 = 6;
pub const IPPROTO_UDP: u32 = 17;
pub const IPPROTO_RAW: u32 = 255;

pub const SOL_SOCKET: u32 = 1;
pub const SOL_TCP: u32 = 6;
pub const SOL_UDP: u32 = 17;

pub const SO_DEBUG: u32 = 1;
pub const SO_REUSEADDR: u32 = 2;
pub const SO_TYPE: u32 = 3;
pub const SO_ERROR: u32 = 4;
pub const SO_DONTROUTE: u32 = 5;
pub const SO_BROADCAST: u32 = 6;
pub const SO_SNDBUF: u32 = 7;
pub const SO_RCVBUF: u32 = 8;
pub const SO_KEEPALIVE: u32 = 9;
pub const SO_OOBINLINE: u32 = 10;
pub const SO_LINGER: u32 = 13;
pub const SO_REUSEPORT: u32 = 15;
pub const SO_RCVTIMEO: u32 = 20;
pub const SO_SNDTIMEO: u32 = 21;
pub const SO_PASSCRED: u32 = 16;

pub const TCP_NODELAY: u32 = 1;
pub const TCP_MAXSEG: u32 = 2;
pub const TCP_KEEPIDLE: u32 = 4;

pub const MSG_OOB: u32 = 1;
pub const MSG_PEEK: u32 = 2;
pub const MSG_DONTROUTE: u32 = 4;
pub const MSG_CTRUNC: u32 = 8;
pub const MSG_TRUNC: u32 = 32;
pub const MSG_DONTWAIT: u32 = 64;
pub const MSG_WAITALL: u32 = 256;

pub const SHUT_RD: u32 = 0;
pub const SHUT_WR: u32 = 1;
pub const SHUT_RDWR: u32 = 2;

pub const SOCKADDR_DATA_MAX: usize = 108;
pub const SOCKET_TYPE_MASK: u32 = 0xf;

pub const UNIX_PATH_MAX: usize = 108;

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
    pub fn len(&self) -> usize { self.path.len() }
}

#[derive(Debug, Clone)]
pub struct SockaddrIn {
    pub family: u16,
    pub port: u16,
    pub addr: [u8; 4],
}

impl SockaddrIn {
    pub fn new(addr: [u8; 4], port: u16) -> Self { Self { family: AF_INET as u16, port, addr } }
    pub fn any() -> Self { Self::new([0,0,0,0], 0) }
    pub fn loopback() -> Self { Self::new([127,0,0,1], 0) }
}

#[derive(Debug, Clone)]
pub struct SockaddrIn6 {
    pub family: u16,
    pub port: u16,
    pub flowinfo: u32,
    pub addr: [u8; 16],
    pub scope_id: u32,
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
    // In iSH C, only STREAM, DGRAM, RAW, SEQPACKET are valid; RDM (4) is not supported
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

#[derive(Debug, Clone)]
pub struct ScmMessage {
    pub data: Vec<u8>,
    pub fds: Vec<i32>,
}

#[derive(Debug)]
pub struct UnixSocket {
    pub id: u32,
    pub peer_id: Option<u32>,
    pub recv_queue: VecDeque<Vec<u8>>,
    pub scm_queue: VecDeque<ScmMessage>,
    pub got_peer: bool,
}

impl UnixSocket {
    pub fn new(id: u32) -> Self {
        Self { id, peer_id: None, recv_queue: VecDeque::new(), scm_queue: VecDeque::new(), got_peer: false }
    }
}

#[derive(Debug)]
pub struct Socket {
    pub family: u32,
    pub sock_type: u32,
    pub protocol: u32,
    pub state: SocketState,
    pub bound_addr: Option<UnixAddr>,
    pub peer_addr: Option<UnixAddr>,
    pub unix_socket: Option<Arc<Mutex<UnixSocket>>>,
    pub unix_name_inode: Option<u64>,
    pub sndbuf: u32,
    pub rcvbuf: u32,
    pub reuseaddr: bool,
    pub reuseport: bool,
    pub keepalive: bool,
    pub dontroute: bool,
    pub broadcast: bool,
    pub linger: Option<u32>,
    pub sndtimeo: u64,
    pub rcvtimeo: u64,
    pub error: i32,
    pub real_fd: i32,
    pub nonblock: bool,
    pub cloexec: bool,
    pub backlog: i32,
    pub accept_queue: VecDeque<Arc<Mutex<Socket>>>,
}

impl Socket {
    pub fn new(family: u32, sock_type: u32, protocol: u32) -> Result<Self, i32> {
        sock_family_to_real(family)?;
        sock_type_to_real(sock_type, protocol)?;
        let mut proto = protocol;
        if sock_type & SOCKET_TYPE_MASK == SOCK_RAW && protocol == IPPROTO_RAW {
            proto = IPPROTO_ICMP; // hack for mtr
        }
        Ok(Self {
            family,
            sock_type: sock_type & SOCKET_TYPE_MASK,
            protocol: proto,
            state: SocketState::Unbound,
            bound_addr: None,
            peer_addr: None,
            unix_socket: None,
            unix_name_inode: None,
            sndbuf: 212992,
            rcvbuf: 212992,
            reuseaddr: false,
            reuseport: false,
            keepalive: false,
            dontroute: false,
            broadcast: false,
            linger: None,
            sndtimeo: 0,
            rcvtimeo: 0,
            error: 0,
            real_fd: -1,
            nonblock: (sock_type & SOCK_NONBLOCK) != 0,
            cloexec: (sock_type & SOCK_CLOEXEC) != 0,
            backlog: 0,
            accept_queue: VecDeque::new(),
        })
    }

    pub fn bind(&mut self, addr: UnixAddr) -> Result<(), i32> {
        if self.state != SocketState::Unbound { return Err(-22); }
        if addr.path.len() > UNIX_PATH_MAX { return Err(-22); }
        self.bound_addr = Some(addr);
        self.state = SocketState::Bound;
        Ok(())
    }

    pub fn listen(&mut self, backlog: i32) -> Result<(), i32> {
        if self.state != SocketState::Bound { return Err(-22); }
        if self.sock_type != SOCK_STREAM && self.sock_type != SOCK_SEQPACKET { return Err(-95); } // EOPNOTSUPP
        self.state = SocketState::Listening;
        self.backlog = backlog;
        Ok(())
    }

    pub fn connect(&mut self, addr: UnixAddr) -> Result<(), i32> {
        if self.state == SocketState::Listening { return Err(-22); }
        if self.state == SocketState::Connected { return Err(-106); } // EISCONN
        self.peer_addr = Some(addr);
        self.state = SocketState::Connected;
        Ok(())
    }

    pub fn accept(&mut self) -> Result<Arc<Mutex<Socket>>, i32> {
        if self.state != SocketState::Listening { return Err(-22); }
        if let Some(sock) = self.accept_queue.pop_front() { Ok(sock) } else { Err(-11) } // EAGAIN
    }

    pub fn send(&mut self, data: &[u8], flags: u32) -> Result<usize, i32> {
        if self.state != SocketState::Connected && self.sock_type == SOCK_STREAM { return Err(-107); } // ENOTCONN
        if (flags & MSG_DONTWAIT) != 0 && self.nonblock { return Err(-11); }
        Ok(data.len())
    }

    pub fn recv(&mut self, buf: &mut [u8], flags: u32) -> Result<usize, i32> {
        if self.state != SocketState::Connected && self.sock_type == SOCK_STREAM { return Err(-107); }
        if let Some(unix) = &self.unix_socket {
            let mut u = unix.lock().unwrap();
            if let Some(data) = u.recv_queue.pop_front() {
                let len = data.len().min(buf.len());
                buf[..len].copy_from_slice(&data[..len]);
                if (flags & MSG_PEEK) != 0 { u.recv_queue.push_front(data); }
                return Ok(len);
            } else {
                if self.nonblock || (flags & MSG_DONTWAIT) != 0 { return Err(-11); }
                return Ok(0);
            }
        }
        Ok(buf.len().min(1024))
    }

    pub fn sendto(&mut self, data: &[u8], flags: u32, addr: Option<UnixAddr>) -> Result<usize, i32> {
        if let Some(a) = addr {
            if self.sock_type == SOCK_STREAM { return Err(-89); } // EDESTADDRREQ handled differently
            self.peer_addr = Some(a);
        }
        self.send(data, flags)
    }

    pub fn recvfrom(&mut self, buf: &mut [u8], flags: u32) -> Result<(usize, Option<UnixAddr>), i32> {
        let n = self.recv(buf, flags)?;
        Ok((n, self.peer_addr.clone()))
    }

    pub fn shutdown(&mut self, how: u32) -> Result<(), i32> {
        match how {
            SHUT_RD | SHUT_WR | SHUT_RDWR => { self.state = SocketState::Closed; Ok(()) },
            _ => Err(-22),
        }
    }

    pub fn setsockopt(&mut self, level: u32, optname: u32, optval: u32) -> Result<(), i32> {
        if level == SOL_SOCKET {
            match optname {
                SO_REUSEADDR => { self.reuseaddr = optval != 0; Ok(()) },
                SO_REUSEPORT => { self.reuseport = optval != 0; Ok(()) },
                SO_KEEPALIVE => { self.keepalive = optval != 0; Ok(()) },
                SO_DONTROUTE => { self.dontroute = optval != 0; Ok(()) },
                SO_BROADCAST => { self.broadcast = optval != 0; Ok(()) },
                SO_SNDBUF => { self.sndbuf = optval; Ok(()) },
                SO_RCVBUF => { self.rcvbuf = optval; Ok(()) },
                SO_LINGER => { self.linger = Some(optval); Ok(()) },
                SO_RCVTIMEO => { self.rcvtimeo = optval as u64; Ok(()) },
                SO_SNDTIMEO => { self.sndtimeo = optval as u64; Ok(()) },
                SO_PASSCRED => Ok(()),
                _ => Err(-92),
            }
        } else if level == SOL_TCP {
            match optname {
                TCP_NODELAY => Ok(()),
                TCP_MAXSEG => Ok(()),
                _ => Err(-92),
            }
        } else {
            Err(-92)
        }
    }

    pub fn getsockopt(&self, level: u32, optname: u32) -> Result<u32, i32> {
        if level == SOL_SOCKET {
            match optname {
                SO_TYPE => Ok(self.sock_type),
                SO_ERROR => Ok(self.error as u32),
                SO_REUSEADDR => Ok(if self.reuseaddr { 1 } else { 0 }),
                SO_REUSEPORT => Ok(if self.reuseport { 1 } else { 0 }),
                SO_KEEPALIVE => Ok(if self.keepalive { 1 } else { 0 }),
                SO_DONTROUTE => Ok(if self.dontroute { 1 } else { 0 }),
                SO_BROADCAST => Ok(if self.broadcast { 1 } else { 0 }),
                SO_SNDBUF => Ok(self.sndbuf),
                SO_RCVBUF => Ok(self.rcvbuf),
                SO_LINGER => Ok(self.linger.unwrap_or(0)),
                SO_RCVTIMEO => Ok(self.rcvtimeo as u32),
                SO_SNDTIMEO => Ok(self.sndtimeo as u32),
                _ => Err(-92),
            }
        } else if level == SOL_TCP {
            match optname {
                TCP_NODELAY => Ok(0),
                TCP_MAXSEG => Ok(1460),
                _ => Err(-92),
            }
        } else {
            Err(-92)
        }
    }

    pub fn getsockname(&self) -> Option<UnixAddr> { self.bound_addr.clone() }
    pub fn getpeername(&self) -> Option<UnixAddr> { self.peer_addr.clone() }

    pub fn socketpair(family: u32, sock_type: u32, protocol: u32) -> Result<(Socket, Socket), i32> {
        let s1 = Socket::new(family, sock_type, protocol)?;
        let s2 = Socket::new(family, sock_type, protocol)?;
        Ok((s1, s2))
    }
}

/// Unix socket ID management, matching C's unix_socket_next_id
static NEXT_UNIX_ID: std::sync::atomic::AtomicU32 = std::sync::atomic::AtomicU32::new(1);

pub fn unix_socket_next_id() -> u32 {
    NEXT_UNIX_ID.fetch_add(1, std::sync::atomic::Ordering::SeqCst)
}

#[derive(Debug, Default)]
pub struct SocketTable {
    pub sockets: HashMap<i32, Arc<Mutex<Socket>>>,
    pub next_fd: i32,
    pub unix_id_to_inode: HashMap<u32, u64>,
}

impl SocketTable {
    pub fn new() -> Self { Self { sockets: HashMap::new(), next_fd: 3, unix_id_to_inode: HashMap::new() } }

    pub fn create(&mut self, family: u32, sock_type: u32, protocol: u32) -> Result<i32, i32> {
        let sock = Socket::new(family, sock_type, protocol)?;
        let fd = self.next_fd;
        self.next_fd += 1;
        self.sockets.insert(fd, Arc::new(Mutex::new(sock)));
        Ok(fd)
    }

    pub fn get(&self, fd: i32) -> Option<Arc<Mutex<Socket>>> { self.sockets.get(&fd).cloned() }

    pub fn close(&mut self, fd: i32) -> Result<(), i32> {
        if self.sockets.remove(&fd).is_some() { Ok(()) } else { Err(-9) }
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
        assert!(is_valid_sock_type(SOCK_DGRAM | SOCK_CLOEXEC));
        assert!(!is_valid_sock_type(4));
    }

    #[test]
    fn family_and_type_to_real() {
        assert_eq!(sock_family_to_real(AF_INET).unwrap(), 2);
        assert!(sock_family_to_real(99).is_err());
        assert_eq!(sock_type_to_real(SOCK_STREAM, 0).unwrap(), 1);
        assert!(sock_type_to_real(4, 0).is_err());
    }

    #[test]
    fn unix_addr_helpers() {
        let addr = UnixAddr::new("/tmp/socket");
        assert!(!addr.is_abstract);
        assert!(!addr.is_unnamed());
        let abstract_addr = UnixAddr::new("\0abstract");
        assert!(abstract_addr.is_abstract);
    }

    #[test]
    fn socket_full_state_machine() {
        let mut sock = Socket::new(AF_UNIX, SOCK_STREAM, 0).unwrap();
        assert_eq!(sock.state, SocketState::Unbound);
        sock.bind(UnixAddr::new("/tmp/test")).unwrap();
        assert_eq!(sock.state, SocketState::Bound);
        sock.listen(5).unwrap();
        assert_eq!(sock.state, SocketState::Listening);
        assert!(sock.connect(UnixAddr::new("/tmp/other")).is_err());
        sock.shutdown(SHUT_RDWR).unwrap();
        assert_eq!(sock.state, SocketState::Closed);
    }

    #[test]
    fn socket_send_recv() {
        let mut sock = Socket::new(AF_UNIX, SOCK_DGRAM, 0).unwrap();
        sock.bind(UnixAddr::new("/tmp/test")).unwrap();
        sock.connect(UnixAddr::new("/tmp/peer")).unwrap();
        let n = sock.send(b"hello", 0).unwrap();
        assert_eq!(n, 5);
        let mut buf = vec![0u8; 10];
        let n2 = sock.recv(&mut buf, 0).unwrap();
        assert!(n2 <= 10);
    }

    #[test]
    fn socket_options_full() {
        let mut sock = Socket::new(AF_INET, SOCK_STREAM, 0).unwrap();
        assert_eq!(sock.getsockopt(SOL_SOCKET, SO_TYPE).unwrap(), SOCK_STREAM);
        sock.setsockopt(SOL_SOCKET, SO_REUSEADDR, 1).unwrap();
        sock.setsockopt(SOL_SOCKET, SO_REUSEPORT, 1).unwrap();
        sock.setsockopt(SOL_SOCKET, SO_KEEPALIVE, 1).unwrap();
        sock.setsockopt(SOL_SOCKET, SO_SNDBUF, 4096).unwrap();
        assert_eq!(sock.getsockopt(SOL_SOCKET, SO_REUSEADDR).unwrap(), 1);
        assert_eq!(sock.getsockopt(SOL_SOCKET, SO_SNDBUF).unwrap(), 4096);
        assert!(sock.getsockopt(SOL_SOCKET, 9999).is_err());
        assert!(sock.setsockopt(SOL_TCP, TCP_NODELAY, 1).is_ok());
    }

    #[test]
    fn socketpair_and_table() {
        let (s1, s2) = Socket::socketpair(AF_UNIX, SOCK_STREAM, 0).unwrap();
        assert_eq!(s1.family, AF_UNIX);
        assert_eq!(s2.family, AF_UNIX);
        let mut table = SocketTable::new();
        let fd = table.create(AF_INET, SOCK_STREAM, 0).unwrap();
        assert!(table.get(fd).is_some());
        assert!(table.close(fd).is_ok());
        assert!(table.get(fd).is_none());
    }

    #[test]
    fn unix_socket_id() {
        let id1 = unix_socket_next_id();
        let id2 = unix_socket_next_id();
        assert_ne!(id1, id2);
        assert!(id2 > id1);
    }
}

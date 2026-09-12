//! `fs/sockrestart.c` — socket restart full port for interrupted syscalls.

use std::collections::HashMap;
use std::sync::{Arc, Mutex};

#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub enum SockRestartType {
    #[default]
    None,
    Connect,
    Accept,
    Recv,
    Send,
}

#[derive(Debug, Clone, Default)]
pub struct SockRestart {
    pub restart_type: SockRestartType,
    pub fd: i32,
    pub addr: u32,
    pub addrlen: u32,
    pub flags: u32,
}

impl SockRestart {
    pub fn new(restart_type: SockRestartType, fd: i32) -> Self { Self { restart_type, fd, ..Default::default() } }
    pub fn is_restarting(&self) -> bool { !matches!(self.restart_type, SockRestartType::None) }
    pub fn clear(&mut self) { self.restart_type = SockRestartType::None; }
}

pub fn should_restart_syscall(error: i32, signal_pending: bool) -> bool { error == -4 && !signal_pending }

/// Saved socket for suspend/resume, matching `struct saved_socket` in C
#[derive(Debug, Clone)]
pub struct SavedSocket {
    pub sock_fd: i32,
    pub type_: i32,
    pub proto: i32,
    pub name: Vec<u8>, // sockaddr
    pub name_len: u32,
}

impl SavedSocket {
    pub fn new(sock_fd: i32, type_: i32, proto: i32, name: Vec<u8>) -> Self {
        let name_len = name.len() as u32;
        Self { sock_fd, type_, proto, name, name_len }
    }
}

#[derive(Debug, Default)]
pub struct SockRestartState {
    pub listen_fds: Vec<i32>,
    pub listen_tasks: HashMap<u32, u32>, // pid -> count
    pub punt_flags: HashMap<u32, bool>, // pid -> punt
    pub saved_sockets: Vec<SavedSocket>,
    pub lock: Mutex<()>,
}

impl SockRestartState {
    pub fn new() -> Self { Self { listen_fds: Vec::new(), listen_tasks: HashMap::new(), punt_flags: HashMap::new(), saved_sockets: Vec::new(), lock: Mutex::new(()) } }

    pub fn begin_listen(&mut self, sock_fd: i32) {
        let _guard = self.lock.lock().unwrap();
        if !self.listen_fds.contains(&sock_fd) { self.listen_fds.push(sock_fd); }
    }

    pub fn end_listen(&mut self, sock_fd: i32) {
        let _guard = self.lock.lock().unwrap();
        self.listen_fds.retain(|&fd| fd != sock_fd);
    }

    pub fn begin_listen_wait(&mut self, pid: u32) {
        let _guard = self.lock.lock().unwrap();
        let count = self.listen_tasks.entry(pid).or_insert(0);
        *count += 1;
    }

    pub fn end_listen_wait(&mut self, pid: u32) {
        let _guard = self.lock.lock().unwrap();
        if let Some(count) = self.listen_tasks.get_mut(&pid) {
            *count = count.saturating_sub(1);
            if *count == 0 { self.listen_tasks.remove(&pid); }
        }
    }

    pub fn should_restart_listen_wait(&mut self, pid: u32) -> bool {
        let _guard = self.lock.lock().unwrap();
        let punt = self.punt_flags.get(&pid).copied().unwrap_or(false);
        self.punt_flags.insert(pid, false);
        punt
    }

    pub fn on_suspend(&mut self, sockets: &HashMap<i32, (i32, i32, Vec<u8>)>) {
        let _guard = self.lock.lock().unwrap();
        assert!(self.saved_sockets.is_empty());
        for &fd in &self.listen_fds {
            if let Some((type_, proto, name)) = sockets.get(&fd) {
                let saved = SavedSocket::new(fd, *type_, *proto, name.clone());
                self.saved_sockets.push(saved);
            }
        }
    }

    pub fn on_resume(&mut self) -> Vec<SavedSocket> {
        let _guard = self.lock.lock().unwrap();
        let saved = std::mem::take(&mut self.saved_sockets);
        for &pid in self.listen_tasks.keys() {
            self.punt_flags.insert(pid, true);
        }
        saved
    }

    pub fn is_listening(&self, fd: i32) -> bool { self.listen_fds.contains(&fd) }
}

#[derive(Debug, Default)]
pub struct TaskSockRestart {
    pub count: u32,
    pub punt: bool,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn sock_restart_type() {
        let mut r = SockRestart::new(SockRestartType::Connect, 3);
        assert!(r.is_restarting());
        assert_eq!(r.fd, 3);
        r.clear();
        assert!(!r.is_restarting());
    }

    #[test]
    fn should_restart() {
        assert!(should_restart_syscall(-4, false));
        assert!(!should_restart_syscall(-4, true));
        assert!(!should_restart_syscall(-22, false));
    }

    #[test]
    fn sock_restart_full_listen() {
        let mut state = SockRestartState::new();
        state.begin_listen(3);
        state.begin_listen(4);
        assert!(state.is_listening(3));
        assert!(state.is_listening(4));
        state.end_listen(3);
        assert!(!state.is_listening(3));
        assert!(state.is_listening(4));
    }

    #[test]
    fn sock_restart_listen_wait() {
        let mut state = SockRestartState::new();
        state.begin_listen_wait(100);
        state.begin_listen_wait(100);
        assert_eq!(state.listen_tasks.get(&100), Some(&2));
        state.end_listen_wait(100);
        assert_eq!(state.listen_tasks.get(&100), Some(&1));
        state.end_listen_wait(100);
        assert!(state.listen_tasks.get(&100).is_none());
        assert!(!state.should_restart_listen_wait(100));
    }

    #[test]
    fn sock_restart_suspend_resume() {
        let mut state = SockRestartState::new();
        state.begin_listen(3);
        state.begin_listen(4);
        let mut sockets = HashMap::new();
        sockets.insert(3, (1, 0, vec![1,2,3,4]));
        sockets.insert(4, (1, 0, vec![5,6,7,8]));
        state.on_suspend(&sockets);
        assert_eq!(state.saved_sockets.len(), 2);
        state.begin_listen_wait(100);
        let saved = state.on_resume();
        assert_eq!(saved.len(), 2);
        assert!(state.saved_sockets.is_empty());
        assert!(state.should_restart_listen_wait(100));
    }

    #[test]
    fn saved_socket() {
        let saved = SavedSocket::new(3, 1, 0, vec![1,2,3]);
        assert_eq!(saved.sock_fd, 3);
        assert_eq!(saved.name_len, 3);
    }
}

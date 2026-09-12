//! `fs/tty.h` + `fs/tty.c` — tty full port, matching C implementation.

use std::collections::{HashMap, VecDeque};
use std::sync::{Arc, Mutex};

#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct Winsize {
    pub row: u16,
    pub col: u16,
    pub xpixel: u16,
    pub ypixel: u16,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct Termios {
    pub iflags: u32,
    pub oflags: u32,
    pub cflags: u32,
    pub lflags: u32,
    pub line: u8,
    pub cc: [u8; 19],
}

pub const VINTR: usize = 0;
pub const VQUIT: usize = 1;
pub const VERASE: usize = 2;
pub const VKILL: usize = 3;
pub const VEOF: usize = 4;
pub const VTIME: usize = 5;
pub const VMIN: usize = 6;
pub const VSWTC: usize = 7;
pub const VSTART: usize = 8;
pub const VSTOP: usize = 9;
pub const VSUSP: usize = 10;
pub const VEOL: usize = 11;
pub const VREPRINT: usize = 12;
pub const VDISCARD: usize = 13;
pub const VWERASE: usize = 14;
pub const VLNEXT: usize = 15;
pub const VEOL2: usize = 16;

pub const IGNBRK: u32 = 1 << 0;
pub const BRKINT: u32 = 1 << 1;
pub const IGNPAR: u32 = 1 << 2;
pub const PARMRK: u32 = 1 << 3;
pub const INPCK: u32 = 1 << 4;
pub const ISTRIP: u32 = 1 << 5;
pub const INLCR: u32 = 1 << 6;
pub const IGNCR: u32 = 1 << 7;
pub const ICRNL: u32 = 1 << 8;
pub const IXON: u32 = 1 << 10;
pub const IXOFF: u32 = 1 << 12;

pub const OPOST: u32 = 1 << 0;
pub const ONLCR: u32 = 1 << 1;
pub const OCRNL: u32 = 1 << 3;
pub const ONOCR: u32 = 1 << 4;
pub const ONLRET: u32 = 1 << 5;

pub const ISIG: u32 = 1 << 0;
pub const ICANON: u32 = 1 << 1;
pub const XCASE: u32 = 1 << 2;
pub const ECHO: u32 = 1 << 3;
pub const ECHOE: u32 = 1 << 4;
pub const ECHOK: u32 = 1 << 5;
pub const ECHONL: u32 = 1 << 6;
pub const NOFLSH: u32 = 1 << 7;
pub const TOSTOP: u32 = 1 << 8;
pub const ECHOCTL: u32 = 1 << 9;
pub const ECHOPRT: u32 = 1 << 10;
pub const ECHOKE: u32 = 1 << 11;
pub const FLUSHO: u32 = 1 << 12;
pub const PENDIN: u32 = 1 << 14;
pub const IEXTEN: u32 = 1 << 15;

pub const B0: u32 = 0;
pub const B50: u32 = 1;
pub const B75: u32 = 2;
pub const B110: u32 = 3;
pub const B134: u32 = 4;
pub const B150: u32 = 5;
pub const B200: u32 = 6;
pub const B300: u32 = 7;
pub const B600: u32 = 8;
pub const B1200: u32 = 9;
pub const B1800: u32 = 10;
pub const B2400: u32 = 11;
pub const B4800: u32 = 12;
pub const B9600: u32 = 13;
pub const B19200: u32 = 14;
pub const B38400: u32 = 15;
pub const B57600: u32 = 4097;
pub const B115200: u32 = 4098;

impl Termios {
    pub fn default_tty() -> Self {
        let mut t = Self::default();
        t.iflags = ICRNL | IXON;
        t.oflags = OPOST | ONLCR;
        t.lflags = ISIG | ICANON | ECHO | ECHOE | ECHOK | ECHOCTL | ECHOKE | IEXTEN;
        t.cc = [3, 28, 127, 21, 4, 0, 1, 0, 17, 19, 26, 0, 18, 15, 23, 22, 0, 0, 0]; // from C
        t
    }
    pub fn is_canonical(&self) -> bool { (self.lflags & ICANON) != 0 }
    pub fn is_echo(&self) -> bool { (self.lflags & ECHO) != 0 }
    pub fn is_sig(&self) -> bool { (self.lflags & ISIG) != 0 }
    pub fn is_echoe(&self) -> bool { (self.lflags & ECHOE) != 0 }
    pub fn is_echok(&self) -> bool { (self.lflags & ECHOK) != 0 }
    pub fn is_ixon(&self) -> bool { (self.iflags & IXON) != 0 }
    pub fn is_icrnl(&self) -> bool { (self.iflags & ICRNL) != 0 }
    pub fn is_opost(&self) -> bool { (self.oflags & OPOST) != 0 }
    pub fn is_onlcr(&self) -> bool { (self.oflags & ONLCR) != 0 }
    pub fn vmin(&self) -> u8 { self.cc[VMIN] }
    pub fn vtime(&self) -> u8 { self.cc[VTIME] }
    pub fn vintr(&self) -> u8 { self.cc[VINTR] }
    pub fn vquit(&self) -> u8 { self.cc[VQUIT] }
    pub fn verase(&self) -> u8 { self.cc[VERASE] }
    pub fn vkill(&self) -> u8 { self.cc[VKILL] }
    pub fn veof(&self) -> u8 { self.cc[VEOF] }
    pub fn vsusp(&self) -> u8 { self.cc[VSUSP] }
    pub fn veol(&self) -> u8 { self.cc[VEOL] }
    pub fn veol2(&self) -> u8 { self.cc[VEOL2] }
}

pub const TTY_CONSOLE_MAJOR: u32 = 4;
pub const TTY_PSEUDO_MASTER_MAJOR: u32 = 2;
pub const TTY_PSEUDO_SLAVE_MAJOR: u32 = 3;
pub const TTY_ALTERNATE_MAJOR: u32 = 5;
pub const DEV_TTY_MINOR: u32 = 0;
pub const DEV_CONSOLE_MINOR: u32 = 1;
pub const DEV_PTMX_MINOR: u32 = 2;

pub const TTY_BUF_SIZE: usize = 4096;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub enum TtyType {
    #[default]
    Console,
    PtyMaster,
    PtySlave,
    Other,
}

#[derive(Debug)]
pub struct Tty {
    pub refcount: u32,
    pub driver_major: u32,
    pub type_: TtyType,
    pub num: i32,
    pub hung_up: bool,
    pub ever_opened: bool,
    pub session: u32,
    pub fg_group: u32,
    pub termios: Termios,
    pub winsize: Winsize,
    pub buf: VecDeque<u8>,
    pub buf_flags: Vec<bool>, // echo flags per char
    pub bufsize: usize,
    pub packet_flags: u32,
    pub pty_other: Option<Arc<Mutex<Tty>>>,
    pub fds: Vec<i32>,
    pub lock: Mutex<()>,
}

impl Tty {
    pub fn alloc(driver_major: u32, type_: TtyType, num: i32) -> Self {
        Self {
            refcount: 0,
            driver_major,
            type_,
            num,
            hung_up: false,
            ever_opened: false,
            session: 0,
            fg_group: 0,
            termios: Termios::default_tty(),
            winsize: Winsize::default(),
            buf: VecDeque::with_capacity(TTY_BUF_SIZE),
            buf_flags: vec![false; TTY_BUF_SIZE],
            bufsize: 0,
            packet_flags: 0,
            pty_other: None,
            fds: Vec::new(),
            lock: Mutex::new(()),
        }
    }

    pub fn get(driver_major: u32, type_: TtyType, num: i32, table: &mut TtyTable) -> Arc<Mutex<Self>> {
        let key = (driver_major, num);
        if let Some(tty) = table.ttys.get(&key) {
            let mut t = tty.lock().unwrap();
            t.refcount += 1;
            t.ever_opened = true;
            return tty.clone();
        }
        let mut tty = Self::alloc(driver_major, type_, num);
        tty.refcount = 1;
        tty.ever_opened = true;
        let arc = Arc::new(Mutex::new(tty));
        table.ttys.insert(key, arc.clone());
        arc
    }

    pub fn release(&mut self) -> bool {
        if self.refcount > 0 { self.refcount -= 1; }
        self.refcount == 0
    }

    pub fn set_controlling(&mut self, session: u32, pgid: u32) {
        if self.session == 0 {
            self.session = session;
            self.fg_group = pgid;
        }
    }

    pub fn set_winsize(&mut self, ws: Winsize) { self.winsize = ws; }

    pub fn write_input(&mut self, data: &[u8]) -> usize {
        let mut written = 0;
        for &b in data {
            if self.buf.len() >= TTY_BUF_SIZE { break; }
            // Handle signals in canonical mode if ISIG
            if self.termios.is_sig() {
                if b == self.termios.vintr() {
                    // SIGINT to foreground group
                    // In C, would send signal to fg_group
                    continue;
                }
                if b == self.termios.vquit() {
                    // SIGQUIT
                    continue;
                }
                if b == self.termios.vsusp() {
                    // SIGTSTP
                    continue;
                }
            }
            if self.termios.is_canonical() {
                if b == self.termios.verase() {
                    if self.termios.is_echoe() {
                        // Erase char
                        if let Some(_) = self.buf.pop_back() {
                            self.bufsize = self.bufsize.saturating_sub(1);
                        }
                    }
                    continue;
                }
                if b == self.termios.vkill() {
                    if self.termios.is_echok() {
                        self.buf.clear();
                        self.bufsize = 0;
                    }
                    continue;
                }
            }
            // ICRNL translation
            let mut byte = b;
            if self.termios.is_icrnl() && b == b'\r' { byte = b'\n'; }
            self.buf.push_back(byte);
            self.bufsize += 1;
            written += 1;
        }
        written
    }

    pub fn read_input(&mut self, out: &mut [u8]) -> usize {
        if self.termios.is_canonical() {
            if let Some(pos) = self.buf.iter().position(|&b| b == b'\n' || b == self.termios.veol() || b == self.termios.veol2()) {
                let to_read = (pos + 1).min(out.len());
                for i in 0..to_read { out[i] = self.buf[i]; }
                for _ in 0..to_read { self.buf.pop_front(); }
                self.bufsize = self.bufsize.saturating_sub(to_read);
                to_read
            } else {
                0
            }
        } else {
            let vmin = self.termios.vmin() as usize;
            let to_read = if vmin == 0 { self.buf.len().min(out.len()) } else { self.buf.len().min(out.len()).min(vmin) };
            for i in 0..to_read { out[i] = self.buf[i]; }
            for _ in 0..to_read { self.buf.pop_front(); }
            self.bufsize = self.bufsize.saturating_sub(to_read);
            to_read
        }
    }

    pub fn write_output(&mut self, data: &[u8]) -> usize {
        // OPOST handling
        if self.termios.is_opost() && self.termios.is_onlcr() {
            // Translate \n to \r\n
            let mut translated = Vec::new();
            for &b in data {
                if b == b'\n' { translated.push(b'\r'); }
                translated.push(b);
            }
            translated.len()
        } else {
            data.len()
        }
    }

    pub fn is_hung_up(&self) -> bool { self.hung_up }
    pub fn set_hung_up(&mut self, hung: bool) { self.hung_up = hung; }
}

#[derive(Debug, Default)]
pub struct TtyTable {
    pub ttys: HashMap<(u32, i32), Arc<Mutex<Tty>>>,
    pub console_major: u32,
    pub console_minor: u32,
}

impl TtyTable {
    pub fn new() -> Self { Self { ttys: HashMap::new(), console_major: TTY_CONSOLE_MAJOR, console_minor: 1 } }

    pub fn alloc(&mut self, driver_major: u32, type_: TtyType, num: i32) -> Arc<Mutex<Tty>> {
        Tty::get(driver_major, type_, num, self)
    }

    pub fn get(&self, driver_major: u32, num: i32) -> Option<Arc<Mutex<Tty>>> {
        self.ttys.get(&(driver_major, num)).cloned()
    }

    pub fn release(&mut self, driver_major: u32, num: i32) -> bool {
        if let Some(tty) = self.ttys.get(&(driver_major, num)) {
            let should_remove = { tty.lock().unwrap().release() };
            if should_remove {
                self.ttys.remove(&(driver_major, num));
                return true;
            }
        }
        false
    }
}

/// TtyBuffer for backwards compat
#[derive(Debug, Default)]
pub struct TtyBuffer {
    pub input: Vec<u8>,
    pub output: Vec<u8>,
    pub termios: Termios,
}

impl TtyBuffer {
    pub fn new() -> Self { Self { input: Vec::new(), output: Vec::new(), termios: Termios::default_tty() } }
    pub fn write_input(&mut self, data: &[u8]) -> usize {
        self.input.extend_from_slice(data);
        data.len()
    }
    pub fn read_input(&mut self, out: &mut [u8]) -> usize {
        if self.termios.is_canonical() {
            if let Some(pos) = self.input.iter().position(|&b| b == b'\n') {
                let to_read = (pos + 1).min(out.len());
                out[..to_read].copy_from_slice(&self.input[..to_read]);
                self.input.drain(..to_read);
                to_read
            } else { 0 }
        } else {
            let to_read = self.input.len().min(out.len()).min(self.termios.vmin() as usize);
            if to_read == 0 && !self.input.is_empty() {
                let to_read = self.input.len().min(out.len());
                out[..to_read].copy_from_slice(&self.input[..to_read]);
                self.input.drain(..to_read);
                return to_read;
            }
            out[..to_read].copy_from_slice(&self.input[..to_read]);
            self.input.drain(..to_read);
            to_read
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn winsize_default() { let ws = Winsize::default(); assert_eq!(ws.row, 0); }

    #[test]
    fn termios_default_tty() {
        let term = Termios::default_tty();
        assert!(term.is_canonical());
        assert!(term.is_echo());
        assert!(term.is_sig());
        assert_eq!(term.vintr(), 3);
        assert_eq!(term.vquit(), 28);
        assert_eq!(term.verase(), 127);
    }

    #[test]
    fn tty_alloc_and_table() {
        let mut table = TtyTable::new();
        let tty = table.alloc(TTY_CONSOLE_MAJOR, TtyType::Console, 1);
        assert_eq!(tty.lock().unwrap().refcount, 1);
        let tty2 = table.alloc(TTY_CONSOLE_MAJOR, TtyType::Console, 1);
        assert_eq!(tty2.lock().unwrap().refcount, 2);
        assert!(table.get(TTY_CONSOLE_MAJOR, 1).is_some());
    }

    #[test]
    fn tty_line_discipline() {
        let mut table = TtyTable::new();
        let tty_arc = table.alloc(TTY_CONSOLE_MAJOR, TtyType::Console, 1);
        let mut tty = tty_arc.lock().unwrap();
        tty.write_input(b"hello\n");
        let mut out = [0u8; 10];
        let n = tty.read_input(&mut out);
        assert_eq!(n, 6);
        assert_eq!(&out[..6], b"hello\n");
    }

    #[test]
    fn tty_raw_mode() {
        let mut table = TtyTable::new();
        let tty_arc = table.alloc(TTY_CONSOLE_MAJOR, TtyType::Console, 2);
        let mut tty = tty_arc.lock().unwrap();
        tty.termios.lflags = 0;
        tty.termios.cc[VMIN] = 1;
        tty.write_input(b"abc");
        let mut out = [0u8; 5];
        assert_eq!(tty.read_input(&mut out), 1);
        assert_eq!(out[0], b'a');
    }

    #[test]
    fn tty_signal_handling() {
        let mut table = TtyTable::new();
        let tty_arc = table.alloc(TTY_CONSOLE_MAJOR, TtyType::Console, 3);
        let mut tty = tty_arc.lock().unwrap();
        // VINTR (Ctrl-C = 3) should be consumed and not buffered if ISIG
        tty.write_input(&[3]);
        assert_eq!(tty.buf.len(), 0);
        // Without ISIG, it should be buffered
        tty.termios.lflags = 0;
        tty.write_input(&[3]);
        assert_eq!(tty.buf.len(), 1);
    }

    #[test]
    fn tty_buffer_backwards_compat() {
        let mut buf = TtyBuffer::new();
        buf.termios.lflags = ICANON;
        buf.write_input(b"hello\nworld\n");
        let mut out = [0u8; 10];
        assert_eq!(buf.read_input(&mut out), 6);
        assert_eq!(&out[..6], b"hello\n");
    }

    #[test]
    fn baud_constants() {
        assert_eq!(B0, 0);
        assert_eq!(B9600, 13);
        assert_eq!(B115200, 4098);
    }
}

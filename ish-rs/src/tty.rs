//! `fs/tty.h` + `fs/tty.c` — tty constants, termios, and line discipline.

/// `winsize_` guest ABI.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct Winsize {
    pub row: u16,
    pub col: u16,
    pub xpixel: u16,
    pub ypixel: u16,
}

/// `termios_` guest ABI
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

pub const ISIG: u32 = 1 << 0;
pub const ICANON: u32 = 1 << 1;
pub const ECHO: u32 = 1 << 3;
pub const ECHOE: u32 = 1 << 4;
pub const ECHOK: u32 = 1 << 5;
pub const ECHOKE: u32 = 1 << 6;
pub const NOFLSH: u32 = 1 << 7;
pub const ECHOCTL: u32 = 1 << 9;
pub const IEXTEN: u32 = 1 << 15;

/// Baud rates
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

/// Line discipline helpers.
impl Termios {
    pub fn is_canonical(&self) -> bool {
        (self.lflags & ICANON) != 0
    }

    pub fn is_echo(&self) -> bool {
        (self.lflags & ECHO) != 0
    }

    pub fn vmin(&self) -> u8 {
        self.cc[VMIN]
    }

    pub fn vtime(&self) -> u8 {
        self.cc[VTIME]
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn winsize_default() {
        let ws = Winsize::default();
        assert_eq!(ws.row, 0);
    }

    #[test]
    fn termios_canonical_and_echo() {
        let mut term = Termios::default();
        term.lflags = ICANON | ECHO;
        assert!(term.is_canonical());
        assert!(term.is_echo());
        term.lflags = 0;
        assert!(!term.is_canonical());
    }

    #[test]
    fn baud_constants() {
        assert_eq!(B0, 0);
        assert_eq!(B9600, 13);
    }
}

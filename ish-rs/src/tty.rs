//! `fs/tty.h` — tty constants and structures.

/// `winsize_` guest ABI.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct Winsize {
    pub row: u16,
    pub col: u16,
    pub xpixel: u16,
    pub ypixel: u16,
}

/// `termios_` guest ABI (simplified __kernel_termios).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct Termios {
    pub iflags: u32,
    pub oflags: u32,
    pub cflags: u32,
    pub lflags: u32,
    pub line: u8,
    pub cc: [u8; 19],
}

/// `V*` constants
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

/// `lflags` bits
pub const ISIG: u32 = 1 << 0;
pub const ICANON: u32 = 1 << 1;
pub const ECHO: u32 = 1 << 3;
pub const ECHOE: u32 = 1 << 4;
pub const ECHOK: u32 = 1 << 5;
pub const ECHOKE: u32 = 1 << 6;
pub const NOFLSH: u32 = 1 << 7;
pub const ECHOCTL: u32 = 1 << 9;
pub const IEXTEN: u32 = 1 << 15;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn winsize_default() {
        let ws = Winsize::default();
        assert_eq!(ws.row, 0);
    }

    #[test]
    fn termios_cc_constants() {
        assert_eq!(VINTR, 0);
        assert_eq!(VEOF, 4);
        assert_eq!(ISIG, 1);
        assert_eq!(ICANON, 2);
    }
}

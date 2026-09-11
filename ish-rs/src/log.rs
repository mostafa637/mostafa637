//! `kernel/log.c` — iSH's kernel log ring and `sys_syslog` ABI.
//!
//! The C implementation collects complete `printk` lines in a one-MiB FIFO and
//! offers the old Linux `syslog(2)` actions over guest memory. [`KernelLog`]
//! owns the process-global ring while [`PrintkBuffer`] represents C's
//! per-thread formatting tail; an embedding supplies an explicit [`LogSink`]
//! for complete host-visible lines. This keeps the Rust core portable rather
//! than assuming iOS's `NSLog`, Linux syslog, or a particular stderr descriptor.

use crate::group::ESRCH;
use crate::task::{Addr, TaskTable};

/// `_EFAULT` from `kernel/errno.h`.
pub const EFAULT: i32 = -14;
/// `_EINVAL` from `kernel/errno.h`.
pub const EINVAL: i32 = -22;

/// `LOG_BUF_SHIFT` in `kernel/log.c`.
pub const LOG_BUF_SHIFT: usize = 20;
/// `1 << LOG_BUF_SHIFT`, the byte capacity of iSH's circular kernel log.
pub const LOG_CAPACITY: usize = 1 << LOG_BUF_SHIFT;
/// The fixed per-thread formatting buffer in `ish_vprintk`.
pub const PRINTK_BUFFER_CAPACITY: usize = 16_384;

/// `SYSLOG_ACTION_CLOSE_`.
pub const SYSLOG_ACTION_CLOSE: i32 = 0;
/// `SYSLOG_ACTION_OPEN_`.
pub const SYSLOG_ACTION_OPEN: i32 = 1;
/// `SYSLOG_ACTION_READ_`.
pub const SYSLOG_ACTION_READ: i32 = 2;
/// `SYSLOG_ACTION_READ_ALL_`.
pub const SYSLOG_ACTION_READ_ALL: i32 = 3;
/// `SYSLOG_ACTION_READ_CLEAR_`.
pub const SYSLOG_ACTION_READ_CLEAR: i32 = 4;
/// `SYSLOG_ACTION_CLEAR_`.
pub const SYSLOG_ACTION_CLEAR: i32 = 5;
/// `SYSLOG_ACTION_CONSOLE_OFF_`.
pub const SYSLOG_ACTION_CONSOLE_OFF: i32 = 6;
/// `SYSLOG_ACTION_CONSOLE_ON_`.
pub const SYSLOG_ACTION_CONSOLE_ON: i32 = 7;
/// `SYSLOG_ACTION_CONSOLE_LEVEL_`.
pub const SYSLOG_ACTION_CONSOLE_LEVEL: i32 = 8;
/// `SYSLOG_ACTION_SIZE_UNREAD_`.
pub const SYSLOG_ACTION_SIZE_UNREAD: i32 = 9;
/// `SYSLOG_ACTION_SIZE_BUFFER_`.
pub const SYSLOG_ACTION_SIZE_BUFFER: i32 = 10;

const FIFO_PEEK: u8 = 1;
const FIFO_LAST: u8 = 2;

/// A destination for each complete `printk` line.
///
/// C invokes its configured `log_line` handler before adding the same line and
/// newline to the internal FIFO. [`PrintkBuffer::printk`] preserves that order.
pub trait LogSink {
    /// Receive a complete line without its terminating newline.
    fn line(&mut self, line: &[u8]);
}

impl<F> LogSink for F
where
    F: FnMut(&[u8]),
{
    fn line(&mut self, line: &[u8]) {
        self(line);
    }
}

/// The process-global mutable log state from `kernel/log.c`.
///
/// This corresponds to C's `log_buffer`, `log_buf`, and
/// `log_max_since_clear`; [`PrintkBuffer`] holds the separate `__thread`
/// formatting state. The object itself is intentionally not synchronized;
/// synchronize it at the embedding boundary when multiple guest threads enter
/// `printk` or [`sys_syslog`] concurrently.
pub struct KernelLog {
    fifo: Box<[u8]>,
    fifo_start: usize,
    fifo_size: usize,
    max_since_clear: usize,
}

/// The per-thread `ish_vprintk` buffer from `kernel/log.c`.
///
/// Each guest execution thread needs its own buffer, while all of them share a
/// [`KernelLog`]. It holds only bytes after the most recent newline.
pub struct PrintkBuffer {
    tail: Vec<u8>,
}

impl Default for KernelLog {
    fn default() -> Self {
        Self::new()
    }
}

impl KernelLog {
    /// Construct C's initially empty one-MiB log FIFO.
    #[must_use]
    pub fn new() -> Self {
        Self {
            fifo: vec![0; LOG_CAPACITY].into_boxed_slice(),
            fifo_start: 0,
            fifo_size: 0,
            max_since_clear: 0,
        }
    }

    /// Bytes currently available to `SYSLOG_ACTION_READ`.
    #[must_use]
    pub fn unread_len(&self) -> usize {
        self.fifo_size
    }

    /// C's `log_max_since_clear`, useful to an embedding for diagnostics.
    #[must_use]
    pub fn max_since_clear(&self) -> usize {
        self.max_since_clear
    }

    fn append_log_bytes(&mut self, bytes: &[u8]) {
        // Every call from printk is at most 16 KiB, so this is within the
        // defined operating range of util/fifo.c's FIFO_OVERWRITE path.
        assert!(bytes.len() <= LOG_CAPACITY);
        let remaining = LOG_CAPACITY - self.fifo_size;
        if bytes.len() > remaining {
            let excess = bytes.len() - remaining;
            self.fifo_start = (self.fifo_start + excess) % LOG_CAPACITY;
            self.fifo_size -= excess;
        }

        let tail = (self.fifo_start + self.fifo_size) % LOG_CAPACITY;
        let first = (LOG_CAPACITY - tail).min(bytes.len());
        self.fifo[tail..tail + first].copy_from_slice(&bytes[..first]);
        self.fifo[..bytes.len() - first].copy_from_slice(&bytes[first..]);
        self.fifo_size += bytes.len();
        self.max_since_clear = (self.max_since_clear + bytes.len()).min(LOG_CAPACITY);
    }

    /// Literal `util/fifo.c::fifo_read` behavior for defined reads.
    ///
    /// In particular, upstream uses `fifo->start` rather than the locally
    /// adjusted `start` when deciding the first split length. That oddity is
    /// observable for a wrapped `FIFO_LAST` read and is retained here.
    fn fifo_read(&mut self, count: usize, flags: u8) -> Option<Vec<u8>> {
        if count > self.fifo_size {
            return None;
        }

        let start = if flags & FIFO_LAST != 0 {
            (self.fifo_start + self.fifo_size - count) % LOG_CAPACITY
        } else {
            self.fifo_start
        };
        // This deliberately uses fifo_start, not start: see util/fifo.c.
        let first = (LOG_CAPACITY - self.fifo_start).min(count);
        let mut out = Vec::with_capacity(count);
        for offset in 0..first {
            // The C source has undefined behavior if this reaches beyond the
            // backing array. Modulo makes the Rust invalid-input behavior safe;
            // all C-derived cases remain within C's defined range.
            out.push(self.fifo[(start + offset) % LOG_CAPACITY]);
        }
        for offset in first..count {
            out.push(self.fifo[offset - first]);
        }

        if flags & FIFO_PEEK == 0 {
            self.fifo_start = (start + count) % LOG_CAPACITY;
            self.fifo_size -= count;
        }
        Some(out)
    }

    fn syslog_read(&mut self, table: &mut TaskTable, buf_addr: Addr, len: i32, flags: u8) -> i32 {
        if len < 0 {
            return EINVAL;
        }
        let limit = if flags & FIFO_LAST != 0 {
            self.max_since_clear
        } else {
            LOG_CAPACITY
        };
        let count = (len as usize).min(limit);

        // fifo_read's failure is ignored by the C caller. In that undefined
        // C-output case malloc's uninitialized bytes reach user_write; retain
        // the defined return/state behavior while using zeroes safely in Rust.
        let bytes = self
            .fifo_read(count, flags)
            .unwrap_or_else(|| vec![0; count]);
        let Some(task) = table.current_mut() else {
            return ESRCH;
        };
        task.user_write(buf_addr, &bytes)
            .map_or(EFAULT, |_| count as i32)
    }
}

impl Default for PrintkBuffer {
    fn default() -> Self {
        Self::new()
    }
}

impl PrintkBuffer {
    /// Construct C's initially empty thread-local formatting tail.
    #[must_use]
    pub fn new() -> Self {
        Self { tail: Vec::new() }
    }

    /// Bytes buffered after the final newline in this thread's `printk` stream.
    #[must_use]
    pub fn pending_len(&self) -> usize {
        self.tail.len()
    }

    /// Feed an already-rendered C `ish_vprintk` string into the log.
    ///
    /// `rendered` is the output a valid `vsprintf` call would have produced;
    /// bytes after an embedded NUL are therefore ignored. C has an unchecked
    /// 16-KiB thread-local formatting array. This safe port rejects input that
    /// would overflow that C array rather than reproducing undefined behavior.
    /// A completed line is delivered to `sink`, then appended with its newline
    /// to `log` exactly as `output_line` and `log_buf_append` do.
    pub fn printk(&mut self, log: &mut KernelLog, rendered: &[u8], sink: &mut impl LogSink) {
        let visible = rendered
            .iter()
            .position(|&byte| byte == 0)
            .map_or(rendered, |end| &rendered[..end]);
        assert!(
            self.tail.len() + visible.len() < PRINTK_BUFFER_CAPACITY,
            "printk input exceeds kernel/log.c's 16 KiB formatting buffer"
        );
        self.tail.extend_from_slice(visible);

        while let Some(newline) = self.tail.iter().position(|&byte| byte == b'\n') {
            let line = self.tail[..newline].to_vec();
            sink.line(&line);
            log.append_log_bytes(&line);
            log.append_log_bytes(b"\n");
            self.tail.drain(..=newline);
        }
    }
}

/// `sys_syslog`.
///
/// The mutable log state is supplied explicitly instead of residing in a
/// process-global static. `READ_CLEAR` intentionally returns zero after a
/// successful read: this is the C fallthrough from `READ_CLEAR` to `CLEAR`, not
/// the byte count that its internal `syslog_read` returned.
pub fn sys_syslog(
    log: &mut KernelLog,
    table: &mut TaskTable,
    action: i32,
    buf_addr: Addr,
    len: i32,
) -> i32 {
    match action {
        SYSLOG_ACTION_READ => log.syslog_read(table, buf_addr, len, 0),
        SYSLOG_ACTION_READ_ALL => log.syslog_read(table, buf_addr, len, FIFO_LAST | FIFO_PEEK),
        SYSLOG_ACTION_READ_CLEAR => {
            let result = log.syslog_read(table, buf_addr, len, FIFO_LAST | FIFO_PEEK);
            if result < 0 {
                result
            } else {
                log.max_since_clear = 0;
                0
            }
        }
        SYSLOG_ACTION_CLEAR => {
            log.max_since_clear = 0;
            0
        }
        SYSLOG_ACTION_SIZE_UNREAD => log.fifo_size as i32,
        SYSLOG_ACTION_SIZE_BUFFER => LOG_CAPACITY as i32,
        SYSLOG_ACTION_CLOSE
        | SYSLOG_ACTION_OPEN
        | SYSLOG_ACTION_CONSOLE_OFF
        | SYSLOG_ACTION_CONSOLE_ON
        | SYSLOG_ACTION_CONSOLE_LEVEL => 0,
        _ => EINVAL,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::memory::P_RWX;
    use crate::mmu::PAGE_BITS;

    #[derive(Default)]
    struct VecSink(Vec<Vec<u8>>);

    impl LogSink for VecSink {
        fn line(&mut self, line: &[u8]) {
            self.0.push(line.to_vec());
        }
    }

    struct NullSink;

    impl LogSink for NullSink {
        fn line(&mut self, _line: &[u8]) {}
    }

    fn table_with_page() -> TaskTable {
        let table = TaskTable::bootstrap();
        table
            .current()
            .unwrap()
            .mm_mut()
            .unwrap()
            .mem
            .map_nothing(0x100, 1, P_RWX);
        table
    }

    #[test]
    fn printk_buffers_partial_lines_and_syslog_preserves_c_read_rules() {
        let mut log = KernelLog::new();
        let mut buffer = PrintkBuffer::new();
        let mut lines = VecSink::default();
        buffer.printk(&mut log, b"alpha", &mut lines);
        assert!(lines.0.is_empty());
        assert_eq!(buffer.pending_len(), 5);
        assert_eq!(log.unread_len(), 0);

        buffer.printk(&mut log, b" beta\nnext\ntrail", &mut lines);
        assert_eq!(lines.0, [b"alpha beta".to_vec(), b"next".to_vec()]);
        assert_eq!(buffer.pending_len(), 5);
        assert_eq!(log.unread_len(), 16);
        assert_eq!(log.max_since_clear(), 16);

        let mut table = table_with_page();
        let addr = 0x100 << PAGE_BITS;
        assert_eq!(
            sys_syslog(&mut log, &mut table, SYSLOG_ACTION_READ_ALL, addr, 12),
            12
        );
        let mut bytes = [0; 12];
        table
            .current_mut()
            .unwrap()
            .user_read(addr, &mut bytes)
            .unwrap();
        assert_eq!(&bytes, b"a beta\nnext\n");
        assert_eq!(log.unread_len(), 16);

        assert_eq!(
            sys_syslog(&mut log, &mut table, SYSLOG_ACTION_READ_CLEAR, addr, 6),
            0
        );
        assert_eq!(log.max_since_clear(), 0);
        assert_eq!(log.unread_len(), 16);
    }

    #[test]
    fn threads_keep_separate_printk_tails_but_share_the_kernel_fifo() {
        let mut log = KernelLog::new();
        let mut first = PrintkBuffer::new();
        let mut second = PrintkBuffer::new();
        let mut lines = VecSink::default();
        first.printk(&mut log, b"first", &mut lines);
        second.printk(&mut log, b"second\n", &mut lines);
        first.printk(&mut log, b" line\n", &mut lines);
        assert_eq!(lines.0, [b"second".to_vec(), b"first line".to_vec()]);
        assert_eq!(log.unread_len(), b"second\nfirst line\n".len());
    }

    #[test]
    fn read_consumes_before_a_guest_fault_but_read_clear_does_not_clear_on_one() {
        let mut log = KernelLog::new();
        let mut buffer = PrintkBuffer::new();
        buffer.printk(&mut log, b"abcdef\n", &mut NullSink);
        let mut table = table_with_page();
        assert_eq!(
            sys_syslog(&mut log, &mut table, SYSLOG_ACTION_READ, 0x9000, 3),
            EFAULT
        );
        assert_eq!(log.unread_len(), 4);
        assert_eq!(log.max_since_clear(), 7);
        assert_eq!(
            sys_syslog(&mut log, &mut table, SYSLOG_ACTION_READ_CLEAR, 0x9000, 4),
            EFAULT
        );
        assert_eq!(log.max_since_clear(), 7);
        assert_eq!(log.unread_len(), 4);
    }
}

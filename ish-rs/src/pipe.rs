//! `fs/pipe.c` — pipe constants and buffer logic.
//!
//! The full pipe implementation depends on fd table and poll, but the
//! buffer size constants and pure read/write logic can be ported.

/// Default pipe buffer size in iSH (from pipe.c).
pub const PIPE_BUF_SIZE: usize = 4096;

/// Pipe buffer, simplified circular buffer matching C's pipe logic.
#[derive(Debug, Clone)]
pub struct PipeBuffer {
    buf: Vec<u8>,
    start: usize,
    size: usize,
    capacity: usize,
}

impl PipeBuffer {
    pub fn new(capacity: usize) -> Self {
        Self {
            buf: vec![0; capacity],
            start: 0,
            size: 0,
            capacity,
        }
    }

    pub fn size(&self) -> usize {
        self.size
    }

    pub fn capacity(&self) -> usize {
        self.capacity
    }

    pub fn is_empty(&self) -> bool {
        self.size == 0
    }

    pub fn is_full(&self) -> bool {
        self.size == self.capacity
    }

    pub fn write(&mut self, data: &[u8]) -> usize {
        let mut written = 0;
        for &b in data {
            if self.is_full() {
                break;
            }
            let idx = (self.start + self.size) % self.capacity;
            self.buf[idx] = b;
            self.size += 1;
            written += 1;
        }
        written
    }

    pub fn read(&mut self, out: &mut [u8]) -> usize {
        let mut read = 0;
        for b in out.iter_mut() {
            if self.is_empty() {
                break;
            }
            *b = self.buf[self.start];
            self.start = (self.start + 1) % self.capacity;
            self.size -= 1;
            read += 1;
        }
        read
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn pipe_buffer_basic() {
        let mut pipe = PipeBuffer::new(8);
        assert!(pipe.is_empty());
        assert_eq!(pipe.write(b"abc"), 3);
        assert_eq!(pipe.size(), 3);
        let mut out = [0u8; 2];
        assert_eq!(pipe.read(&mut out), 2);
        assert_eq!(&out, b"ab");
        assert_eq!(pipe.size(), 1);
    }

    #[test]
    fn pipe_buffer_wrap_and_full() {
        let mut pipe = PipeBuffer::new(4);
        assert_eq!(pipe.write(b"abcd"), 4);
        assert!(pipe.is_full());
        assert_eq!(pipe.write(b"e"), 0);
        let mut out = [0u8; 4];
        assert_eq!(pipe.read(&mut out), 4);
        assert_eq!(&out, b"abcd");
        assert!(pipe.is_empty());
    }
}

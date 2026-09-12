//! `fs/pipe.c` — pipe constants and buffer logic with fd ops.

/// Default pipe buffer size
pub const PIPE_BUF_SIZE: usize = 4096;
pub const PIPE_MAX_SIZE: usize = 65536;

/// Pipe buffer, circular matching C
#[derive(Debug, Clone)]
pub struct PipeBuffer {
    buf: Vec<u8>,
    start: usize,
    size: usize,
    capacity: usize,
}

impl PipeBuffer {
    pub fn new(capacity: usize) -> Self {
        Self { buf: vec![0; capacity], start: 0, size: 0, capacity }
    }
    pub fn size(&self) -> usize { self.size }
    pub fn capacity(&self) -> usize { self.capacity }
    pub fn is_empty(&self) -> bool { self.size == 0 }
    pub fn is_full(&self) -> bool { self.size == self.capacity }
    pub fn available(&self) -> usize { self.capacity - self.size }

    pub fn write(&mut self, data: &[u8]) -> usize {
        let mut written = 0;
        for &b in data {
            if self.is_full() { break; }
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
            if self.is_empty() { break; }
            *b = self.buf[self.start];
            self.start = (self.start + 1) % self.capacity;
            self.size -= 1;
            read += 1;
        }
        read
    }
}

/// Pipe with two ends, matching C's pipe creation
#[derive(Debug)]
pub struct Pipe {
    pub buffer: PipeBuffer,
    pub read_closed: bool,
    pub write_closed: bool,
}

impl Pipe {
    pub fn new() -> Self {
        Self {
            buffer: PipeBuffer::new(PIPE_BUF_SIZE),
            read_closed: false,
            write_closed: false,
        }
    }

    pub fn new_with_capacity(cap: usize) -> Self {
        Self {
            buffer: PipeBuffer::new(cap),
            read_closed: false,
            write_closed: false,
        }
    }

    pub fn write(&mut self, data: &[u8]) -> Result<usize, i32> {
        if self.read_closed {
            return Err(-32); // EPIPE
        }
        Ok(self.buffer.write(data))
    }

    pub fn read(&mut self, out: &mut [u8]) -> Result<usize, i32> {
        if self.buffer.is_empty() && self.write_closed {
            return Ok(0); // EOF
        }
        Ok(self.buffer.read(out))
    }

    pub fn close_read(&mut self) { self.read_closed = true; }
    pub fn close_write(&mut self) { self.write_closed = true; self.buffer.size = 0; }
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

    #[test]
    fn pipe_read_write_close() {
        let mut pipe = Pipe::new();
        assert_eq!(pipe.write(b"hello").unwrap(), 5);
        let mut out = [0u8; 5];
        assert_eq!(pipe.read(&mut out).unwrap(), 5);
        assert_eq!(&out, b"hello");

        pipe.close_write();
        let mut out2 = [0u8; 5];
        assert_eq!(pipe.read(&mut out2).unwrap(), 0); // EOF after write closed

        let mut pipe2 = Pipe::new();
        pipe2.close_read();
        assert_eq!(pipe2.write(b"hi").unwrap_err(), -32);
    }
}

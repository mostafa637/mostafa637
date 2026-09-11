//! `kernel/random.{h,c}` — guest `getrandom(2)` over an explicit host source.
//!
//! iSH obtains entropy from `CCRandomGenerateBytes` on iOS and the host
//! `getrandom` syscall on Linux. The portable Rust core deliberately does not
//! replace either with a predictable pseudo-random generator: an embedding
//! supplies its platform's cryptographic source through [`RandomSource`].

use crate::group::ESRCH;
use crate::task::{Addr, TaskTable};

/// `_EIO` — iSH maps both host entropy failures and over-large requests here.
pub const EIO: i32 = -5;
/// `_EFAULT` — the output buffer could not be written to guest memory.
pub const EFAULT: i32 = -14;
/// `1 << 20`, the largest request iSH accepts in one syscall.
pub const MAX_RANDOM_BYTES: u32 = 1 << 20;

/// A platform entropy-source failure.
///
/// iSH deliberately exposes no host failure detail: [`sys_getrandom`] maps all
/// variants to the guest's `_EIO`. The named type keeps [`RandomSource`] useful
/// to embeddings without using an uninformative `Result<(), ()>`.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum RandomError {
    /// The platform could not fill the requested bytes.
    Failed,
}

/// The platform entropy boundary used by [`sys_getrandom`].
///
/// Linux iSH implements this with `syscall(SYS_getrandom, ...)`; iOS uses
/// `CCRandomGenerateBytes`. Implementations must fill all bytes or return an
/// error. A failure intentionally becomes the guest's `_EIO`, exactly as the C
/// `get_random` wrapper does.
pub trait RandomSource {
    /// Fill `out` with random bytes, or report that the platform source failed.
    fn fill_random(&self, out: &mut [u8]) -> Result<(), RandomError>;
}

/// `sys_getrandom`.
///
/// The `flags` argument is intentionally ignored, as it is by iSH's C source.
/// Entropy is obtained before the guest output is written, so an output fault
/// still consumes source bytes just like `kernel/random.c`.
pub fn sys_getrandom(
    table: &mut TaskTable,
    source: &impl RandomSource,
    buf_addr: Addr,
    len: u32,
    _flags: u32,
) -> i32 {
    if len > MAX_RANDOM_BYTES {
        return EIO;
    }

    // C allocates exactly `len` bytes before asking its platform source. The
    // accepted bound keeps this allocation at one MiB or below.
    let mut bytes = vec![0; len as usize];
    if source.fill_random(&mut bytes).is_err() {
        return EIO;
    }

    let Some(task) = table.current_mut() else {
        // The C call runs only with its thread-local current task set. Retain
        // the TaskTable convention used by the other current-task syscalls for
        // an explicit Rust caller that has not selected one.
        return ESRCH;
    };
    if task.user_write(buf_addr, &bytes).is_err() {
        return EFAULT;
    }
    len as i32
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::memory::P_RWX;
    use crate::mmu::PAGE_BITS;
    use std::cell::Cell;

    struct PatternSource {
        calls: Cell<u32>,
        fail: Cell<bool>,
    }

    impl RandomSource for PatternSource {
        fn fill_random(&self, out: &mut [u8]) -> Result<(), RandomError> {
            let call = self.calls.get();
            self.calls.set(call + 1);
            if self.fail.get() {
                return Err(RandomError::Failed);
            }
            let len = out.len() as u32;
            for (index, byte) in out.iter_mut().enumerate() {
                *byte = (index as u32)
                    .wrapping_mul(29)
                    .wrapping_add(len)
                    .wrapping_add(call.wrapping_mul(17)) as u8;
            }
            Ok(())
        }
    }

    #[test]
    fn entropy_precedes_the_guest_write_and_the_bound_precedes_entropy() {
        let mut table = TaskTable::bootstrap();
        table
            .current()
            .unwrap()
            .mm_mut()
            .unwrap()
            .mem
            .map_nothing(0x100, 1, P_RWX);
        let source = PatternSource {
            calls: Cell::new(0),
            fail: Cell::new(false),
        };
        let page = 0x100 << PAGE_BITS;

        assert_eq!(sys_getrandom(&mut table, &source, page, 4, 0), 4);
        assert_eq!(source.calls.get(), 1);
        let mut bytes = [0; 4];
        table
            .current_mut()
            .unwrap()
            .user_read(page, &mut bytes)
            .unwrap();
        assert_eq!(bytes, [4, 33, 62, 91]);

        // A guest-memory fault comes after the entropy source has been called.
        assert_eq!(sys_getrandom(&mut table, &source, 0x9000, 4, 0), EFAULT);
        assert_eq!(source.calls.get(), 2);

        // An over-large request is rejected before either allocation/source or
        // a guest-memory lookup.
        assert_eq!(
            sys_getrandom(&mut table, &source, 0x9000, MAX_RANDOM_BYTES + 1, 0),
            EIO
        );
        assert_eq!(source.calls.get(), 2);

        source.fail.set(true);
        assert_eq!(sys_getrandom(&mut table, &source, page, 4, 0), EIO);
        assert_eq!(source.calls.get(), 3);
    }
}

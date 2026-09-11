// Line-by-line translation of emu/interrupt.h
//
// "Intel standard interrupts. Any interrupt not handled specially becomes a
// SIGSEGV."

/// `#define INT_NONE -1`
pub const INT_NONE: i32 = -1;
/// `#define INT_DIV 0`
pub const INT_DIV: i32 = 0;
/// `#define INT_DEBUG 1`
pub const INT_DEBUG: i32 = 1;
/// `#define INT_NMI 2`
pub const INT_NMI: i32 = 2;
/// `#define INT_BREAKPOINT 3`
pub const INT_BREAKPOINT: i32 = 3;
/// `#define INT_OVERFLOW 4`
pub const INT_OVERFLOW: i32 = 4;
/// `#define INT_BOUND 5`
pub const INT_BOUND: i32 = 5;
/// `#define INT_UNDEFINED 6`
pub const INT_UNDEFINED: i32 = 6;
/// `#define INT_FPU 7` — "do not try to use the fpu. instead, try to realize
/// the truth: there is no fpu."
pub const INT_FPU: i32 = 7;
/// `#define INT_DOUBLE 8` — "interrupt during interrupt, i.e. interruptception"
pub const INT_DOUBLE: i32 = 8;
/// `#define INT_GPF 13`
pub const INT_GPF: i32 = 13;
/// `#define INT_TIMER 32`
pub const INT_TIMER: i32 = 32;
/// `#define INT_SYSCALL 0x80`
pub const INT_SYSCALL: i32 = 0x80;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn the_vector_numbers_are_the_x86_ones() {
        // pinned so a reordering cannot silently move a vector
        assert_eq!(INT_NONE, -1);
        assert_eq!(
            [
                INT_DIV,
                INT_DEBUG,
                INT_NMI,
                INT_BREAKPOINT,
                INT_OVERFLOW,
                INT_BOUND,
                INT_UNDEFINED,
                INT_FPU,
                INT_DOUBLE
            ],
            [0, 1, 2, 3, 4, 5, 6, 7, 8]
        );
        assert_eq!(INT_GPF, 13);
        assert_eq!(INT_TIMER, 32);
        assert_eq!(INT_SYSCALL, 0x80);
    }
}

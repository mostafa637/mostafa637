//! `util/timer.{h,c}` — interval timers.
//!
//! The C implementation spawns a pthread per active timer that sleeps until
//! expiry and invokes a callback. This Rust port provides the same API with
//! safe types, using `std::time` for arithmetic and an explicit host
//! callback mechanism. The actual threading is left to the embedding, but the
//! timer spec and state machine are fully ported.

use std::time::{Duration, Instant};

/// `struct timer_spec` — value + interval.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct TimerSpec {
    pub value: Duration,
    pub interval: Duration,
}

impl TimerSpec {
    pub fn is_zero(&self) -> bool {
        self.value.is_zero()
    }
}

/// Host clock id, mirroring `clockid_t`.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ClockId {
    Realtime,
    Monotonic,
}

impl ClockId {
    pub fn now(&self) -> Instant {
        Instant::now()
    }
}

/// Timer callback type.
pub type TimerCallback = Box<dyn FnMut() + Send>;

/// `struct timer` — timer state.
pub struct Timer {
    pub clockid: ClockId,
    pub start: Option<Instant>,
    pub end: Option<Instant>,
    pub interval: Duration,
    pub active: bool,
    pub callback: Option<TimerCallback>,
    pub dead: bool,
}

impl Timer {
    pub fn new(clockid: ClockId, callback: TimerCallback) -> Self {
        Self {
            clockid,
            start: None,
            end: None,
            interval: Duration::ZERO,
            active: false,
            callback: Some(callback),
            dead: false,
        }
    }

    /// `timer_set` — set spec, return old spec.
    pub fn set(&mut self, spec: TimerSpec) -> TimerSpec {
        let old = TimerSpec {
            value: self
                .end
                .and_then(|end| end.checked_duration_since(self.start.unwrap_or_else(Instant::now)))
                .unwrap_or(Duration::ZERO),
            interval: self.interval,
        };

        let now = self.clockid.now();
        self.start = Some(now);
        self.end = Some(now + spec.value);
        self.interval = spec.interval;
        self.active = !spec.value.is_zero();

        old
    }

    /// Check if timer has expired (for single-threaded polling).
    pub fn is_expired(&self) -> bool {
        if !self.active {
            return false;
        }
        if let Some(end) = self.end {
            Instant::now() >= end
        } else {
            false
        }
    }

    /// Fire callback if expired, handling interval re-arm.
    pub fn poll(&mut self) {
        if self.is_expired() {
            if let Some(mut cb) = self.callback.take() {
                cb();
                self.callback = Some(cb);
            }
            if !self.interval.is_zero() {
                let now = self.clockid.now();
                self.start = Some(now);
                self.end = Some(now + self.interval);
            } else {
                self.active = false;
            }
        }
    }

    pub fn free(&mut self) {
        self.active = false;
        self.dead = true;
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::{Arc, Mutex};

    #[test]
    fn timer_spec_zero_check() {
        let spec = TimerSpec {
            value: Duration::ZERO,
            interval: Duration::ZERO,
        };
        assert!(spec.is_zero());
        let spec2 = TimerSpec {
            value: Duration::from_secs(1),
            interval: Duration::ZERO,
        };
        assert!(!spec2.is_zero());
    }

    #[test]
    fn timer_set_and_expire() {
        let fired = Arc::new(Mutex::new(false));
        let fired_clone = fired.clone();
        let mut timer = Timer::new(ClockId::Monotonic, Box::new(move || {
            *fired_clone.lock().unwrap() = true;
        }));

        let spec = TimerSpec {
            value: Duration::from_millis(1),
            interval: Duration::ZERO,
        };
        timer.set(spec);
        assert!(timer.active);
        std::thread::sleep(Duration::from_millis(5));
        assert!(timer.is_expired());
        timer.poll();
        assert!(*fired.lock().unwrap());
        assert!(!timer.active);
    }

    #[test]
    fn timer_interval_rearms() {
        let mut timer = Timer::new(ClockId::Monotonic, Box::new(|| {}));
        let spec = TimerSpec {
            value: Duration::from_millis(1),
            interval: Duration::from_millis(10),
        };
        timer.set(spec);
        std::thread::sleep(Duration::from_millis(2));
        timer.poll();
        assert!(timer.active);
        assert_eq!(timer.interval, Duration::from_millis(10));
    }
}

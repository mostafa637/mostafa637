package syscall

import "math"

// ResourceLimit64 is the Linux rlimit pair used by the native x86-64 ABI.
type ResourceLimit64 struct {
	Cur uint64
	Max uint64
}

func defaultResourceLimits64() map[uint64]ResourceLimit64 {
	unlimited := ResourceLimit64{Cur: math.MaxUint64, Max: math.MaxUint64}
	limits := make(map[uint64]ResourceLimit64, 16)
	for resource := uint64(0); resource <= uint64(rlimitRTTIME); resource++ {
		limits[resource] = unlimited
	}
	limits[uint64(rlimitSTACK)] = ResourceLimit64{Cur: 8 << 20, Max: 8 << 20}
	limits[uint64(rlimitCORE)] = ResourceLimit64{}
	limits[uint64(rlimitNOFILE)] = ResourceLimit64{Cur: 1024, Max: 4096}
	limits[uint64(rlimitMEMLOCK)] = ResourceLimit64{Cur: 64 << 10, Max: 64 << 10}
	limits[uint64(rlimitMSGQUEUE)] = ResourceLimit64{Cur: 819200, Max: 819200}
	limits[uint64(rlimitNICE)] = ResourceLimit64{}
	limits[uint64(rlimitRTPRIO)] = ResourceLimit64{}
	return limits
}

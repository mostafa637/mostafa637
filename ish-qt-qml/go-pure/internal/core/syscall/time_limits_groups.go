package syscall

import (
	"encoding/binary"
	"math"
	"time"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

// ResourceLimit is the Linux rlimit pair stored in guest-process state.
type ResourceLimit struct {
	Cur uint64
	Max uint64
}

const (
	rlimitCPU uint32 = iota
	rlimitFSIZE
	rlimitDATA
	rlimitSTACK
	rlimitCORE
	rlimitRSS
	rlimitNPROC
	rlimitNOFILE
	rlimitMEMLOCK
	rlimitAS
	rlimitLOCKS
	rlimitSIGPENDING
	rlimitMSGQUEUE
	rlimitNICE
	rlimitRTPRIO
	rlimitRTTIME
)

const (
	clockRealtime  uint32 = 0
	clockMonotonic uint32 = 1
)

func defaultResourceLimits() map[uint32]ResourceLimit {
	unlimited := ResourceLimit{Cur: math.MaxUint64, Max: math.MaxUint64}
	limits := make(map[uint32]ResourceLimit, 16)
	for resource := uint32(0); resource <= rlimitRTTIME; resource++ {
		limits[resource] = unlimited
	}
	limits[rlimitSTACK] = ResourceLimit{Cur: 8 << 20, Max: 8 << 20}
	limits[rlimitCORE] = ResourceLimit{}
	limits[rlimitNOFILE] = ResourceLimit{Cur: 1024, Max: 4096}
	limits[rlimitMEMLOCK] = ResourceLimit{Cur: 64 << 10, Max: 64 << 10}
	limits[rlimitMSGQUEUE] = ResourceLimit{Cur: 819200, Max: 819200}
	limits[rlimitNICE] = ResourceLimit{}
	limits[rlimitRTPRIO] = ResourceLimit{}
	return limits
}

func clockGettime(context *Context, _ *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || context.Memory == nil || args[1] == 0 {
		return EFAULT
	}
	if args[0] != clockRealtime && args[0] != clockMonotonic {
		return EINVAL
	}
	now := time.Now()
	seconds := now.Unix()
	nanoseconds := now.Nanosecond()
	if args[0] == clockMonotonic {
		elapsed := time.Since(context.StartTime)
		seconds = int64(elapsed / time.Second)
		nanoseconds = int(elapsed % time.Second)
	}
	var value [8]byte
	binary.LittleEndian.PutUint32(value[0:4], uint32(seconds))
	binary.LittleEndian.PutUint32(value[4:8], uint32(nanoseconds))
	if err := context.Memory.Write(corecpu.Address(args[1]), value[:]); err != nil {
		return EFAULT
	}
	return 0
}

func gettimeofday(context *Context, _ *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || context.Memory == nil {
		return EFAULT
	}
	if args[0] != 0 {
		now := time.Now()
		var value [8]byte
		binary.LittleEndian.PutUint32(value[0:4], uint32(now.Unix()))
		binary.LittleEndian.PutUint32(value[4:8], uint32(now.Nanosecond()/1000))
		if err := context.Memory.Write(corecpu.Address(args[0]), value[:]); err != nil {
			return EFAULT
		}
	}
	if args[1] != 0 {
		var timezone [8]byte
		if err := context.Memory.Write(corecpu.Address(args[1]), timezone[:]); err != nil {
			return EFAULT
		}
	}
	return 0
}

func nanosleep(context *Context, _ *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || context.Memory == nil || args[0] == 0 {
		return EFAULT
	}
	var request [8]byte
	if err := context.Memory.Read(corecpu.Address(args[0]), request[:]); err != nil {
		return EFAULT
	}
	seconds := binary.LittleEndian.Uint32(request[0:4])
	nanoseconds := binary.LittleEndian.Uint32(request[4:8])
	if nanoseconds >= 1_000_000_000 {
		return EINVAL
	}
	if args[1] != 0 {
		var remaining [8]byte
		if err := context.Memory.Write(corecpu.Address(args[1]), remaining[:]); err != nil {
			return EFAULT
		}
	}
	duration := time.Duration(seconds)*time.Second + time.Duration(nanoseconds)
	time.Sleep(duration)
	return 0
}

func getrlimit(context *Context, _ *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || context.Memory == nil || args[1] == 0 {
		return EFAULT
	}
	if context.RLimits == nil {
		context.RLimits = defaultResourceLimits()
	}
	limit, ok := context.RLimits[args[0]]
	if !ok {
		return EINVAL
	}
	var value [16]byte
	binary.LittleEndian.PutUint64(value[0:8], limit.Cur)
	binary.LittleEndian.PutUint64(value[8:16], limit.Max)
	if err := context.Memory.Write(corecpu.Address(args[1]), value[:]); err != nil {
		return EFAULT
	}
	return 0
}

func setrlimit(context *Context, _ *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || context.Memory == nil || args[1] == 0 {
		return EFAULT
	}
	if context.RLimits == nil {
		context.RLimits = defaultResourceLimits()
	}
	if _, ok := context.RLimits[args[0]]; !ok {
		return EINVAL
	}
	var value [16]byte
	if err := context.Memory.Read(corecpu.Address(args[1]), value[:]); err != nil {
		return EFAULT
	}
	limit := ResourceLimit{Cur: binary.LittleEndian.Uint64(value[0:8]), Max: binary.LittleEndian.Uint64(value[8:16])}
	if limit.Cur > limit.Max {
		return EINVAL
	}
	context.RLimits[args[0]] = limit
	return 0
}

func getgroups32(context *Context, _ *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || context.Memory == nil {
		return EFAULT
	}
	groups := context.Groups
	if groups == nil {
		groups = []uint32{0}
	}
	if args[0] == 0 {
		return int32(len(groups))
	}
	if args[0] < uint32(len(groups)) || args[1] == 0 {
		return EINVAL
	}
	buffer := make([]byte, len(groups)*4)
	for index, group := range groups {
		binary.LittleEndian.PutUint32(buffer[index*4:], group)
	}
	if err := context.Memory.Write(corecpu.Address(args[1]), buffer); err != nil {
		return EFAULT
	}
	return int32(len(groups))
}

func setgroups32(context *Context, _ *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || context.Memory == nil {
		return EFAULT
	}
	if args[0] > 65536 {
		return EINVAL
	}
	groups := make([]uint32, args[0])
	if args[0] > 0 {
		if args[1] == 0 {
			return EFAULT
		}
		buffer := make([]byte, args[0]*4)
		if err := context.Memory.Read(corecpu.Address(args[1]), buffer); err != nil {
			return EFAULT
		}
		for index := range groups {
			groups[index] = binary.LittleEndian.Uint32(buffer[index*4:])
		}
	}
	context.Groups = groups
	return 0
}

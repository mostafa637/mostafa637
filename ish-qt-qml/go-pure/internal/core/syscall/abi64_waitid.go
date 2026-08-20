package syscall

import (
	"encoding/binary"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

const (
	waitIDAll64            uint64 = 0
	waitIDPID64            uint64 = 1
	waitIDPGID64           uint64 = 2
	waitIDPIDFD64          uint64 = 3
	waitIDNoHang64         uint32 = WaitNoHang
	waitIDExited64         uint32 = WaitExited
	waitIDNoWait64         uint32 = WaitNoWait
	waitIDSigCHLD64               = 17
	waitIDCLDExited64             = 1
	waitIDSiginfoSize64           = 128
	waitIDSiSignoOffset64         = 0
	waitIDSiCodeOffset64          = 8
	waitIDSiPIDOffset64           = 16
	waitIDSiUIDOffset64           = 20
	waitIDSiStatusOffset64        = 24
)

func waitid64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Children == nil {
		return int64(ECHILD)
	}
	if ctx.Memory == nil || args[2] == 0 {
		return int64(EFAULT)
	}
	if args[4] != 0 {
		return int64(EINVAL)
	}
	options := uint32(args[3])
	if options&waitIDExited64 == 0 || options&^(waitIDNoHang64|waitIDExited64|waitIDNoWait64) != 0 {
		return int64(EINVAL)
	}
	if args[0] != waitIDAll64 && args[0] != waitIDPID64 && args[0] != waitIDPGID64 {
		return int64(EINVAL)
	}
	pid, exitCode, exited, err := ctx.Children.WaitID(uint32(ctx.PID), uint32(args[0]), uint32(args[1]), options)
	if err != 0 {
		return int64(err)
	}
	var info [waitIDSiginfoSize64]byte
	if exited {
		binary.LittleEndian.PutUint32(info[waitIDSiSignoOffset64:], waitIDSigCHLD64)
		binary.LittleEndian.PutUint32(info[waitIDSiCodeOffset64:], waitIDCLDExited64)
		binary.LittleEndian.PutUint32(info[waitIDSiPIDOffset64:], pid)
		binary.LittleEndian.PutUint32(info[waitIDSiUIDOffset64:], 0)
		binary.LittleEndian.PutUint32(info[waitIDSiStatusOffset64:], uint32(exitCode))
	}
	if err := ctx.Memory.Write(corecpu.Address64(args[2]), info[:]); err != nil {
		return int64(EFAULT)
	}
	return 0
}

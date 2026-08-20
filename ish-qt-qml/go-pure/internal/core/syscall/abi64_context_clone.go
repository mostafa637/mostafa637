package syscall

import corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"

// CloneForChild64 copies process-visible state for a fork/clone child. Guest
// memory and the descriptor table are duplicated through their respective
// clone helpers; shared kernel registries remain shared, while signal and
// child-wait state are private to the new process.
func (c *Context64) CloneForChild64(memory *corecpu.Memory64, pid uint64) *Context64 {
	if c == nil || memory == nil || pid == 0 {
		return nil
	}
	child := NewContext64(memory)
	child.FS = c.FS
	child.CWD = c.CWD
	child.WinCols = c.WinCols
	child.WinRows = c.WinRows
	child.Termios = c.Termios
	child.PID = pid
	child.ParentPID = c.PID
	child.TID = pid
	child.PGID = c.PGID
	child.SID = c.SID
	child.RealUID = c.RealUID
	child.EffectiveUID = c.EffectiveUID
	child.SavedUID = c.SavedUID
	child.RealGID = c.RealGID
	child.EffectiveGID = c.EffectiveGID
	child.SavedGID = c.SavedGID
	child.CapEffective = c.CapEffective
	child.CapPermitted = c.CapPermitted
	child.CapInheritable = c.CapInheritable
	child.TIDAddress = c.TIDAddress
	child.RobustListHead = c.RobustListHead
	child.RobustListLen = c.RobustListLen
	child.Brk = c.Brk
	child.Futexes = c.Futexes
	child.RseqAddress = c.RseqAddress
	child.RseqLength = c.RseqLength
	child.RseqSignature = c.RseqSignature
	child.SignalMask = c.SignalMask
	child.PendingSignals = c.PendingSignals
	child.StartTime = c.StartTime
	child.FSBase = c.FSBase
	child.GSBase = c.GSBase
	child.CPUIDEnabled = c.CPUIDEnabled
	child.ProcessName = c.ProcessName
	child.Dumpable = c.Dumpable
	child.NoNewPrivs = c.NoNewPrivs
	child.KeepCaps = c.KeepCaps
	child.SecureBits = c.SecureBits
	child.ChildSubreaper = c.ChildSubreaper
	child.TimerSlack = c.TimerSlack
	child.ParentDeathSig = c.ParentDeathSig
	child.AffinityMask = c.AffinityMask
	child.Umask = c.Umask
	child.SharedMemory = c.SharedMemory
	child.Semaphores = c.Semaphores
	child.Timers = c.Timers
	child.MembarrierEpoch = c.MembarrierEpoch
	child.MembarrierGlobalRegistered = c.MembarrierGlobalRegistered
	child.MembarrierPrivateRegistered = c.MembarrierPrivateRegistered
	child.FDs = c.FDs.Clone()
	child.Mappings = append([]GuestMapping64(nil), c.Mappings...)
	child.Groups = append([]uint32(nil), c.Groups...)
	child.RLimits = make(map[uint64]ResourceLimit64, len(c.RLimits))
	for resource, limit := range c.RLimits {
		child.RLimits[resource] = limit
	}
	child.SignalActions = make(map[uint64][32]byte, len(c.SignalActions))
	for signal, action := range c.SignalActions {
		child.SignalActions[signal] = action
	}
	child.Machine = nil
	child.SignalRestored = false
	child.Exited = false
	child.ExitCode = 0
	child.Execve = nil
	child.ProcessFactory = nil
	child.ChildStarter = nil
	return child
}

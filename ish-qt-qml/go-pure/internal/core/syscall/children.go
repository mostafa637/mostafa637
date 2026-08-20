package syscall

import "sync"

const (
	WaitNoHang uint32 = 1
	WaitExited uint32 = 4
	WaitNoWait uint32 = 0x01000000
)

type ChildState struct {
	PID       uint32
	ParentPID uint32
	Exited    bool
	ExitCode  int32
	Reaped    bool
}

// ChildRegistry is the process-lifecycle seam between the i386 syscall ABI
// and the guest64 kernel runtime. The condition variable lets blocking wait4
// and waitid sleep until a child publishes an exit instead of returning EINTR
// and forcing the executor to spin.
type ChildRegistry struct {
	mu       sync.Mutex
	cond     *sync.Cond
	children map[uint32]*ChildState
}

func NewChildRegistry() *ChildRegistry {
	registry := &ChildRegistry{children: make(map[uint32]*ChildState)}
	registry.cond = sync.NewCond(&registry.mu)
	return registry
}

func (r *ChildRegistry) initLocked() {
	if r.cond == nil {
		r.cond = sync.NewCond(&r.mu)
	}
	if r.children == nil {
		r.children = make(map[uint32]*ChildState)
	}
}

func (r *ChildRegistry) AddChild(parentPID, childPID uint32) bool {
	if r == nil || childPID == 0 {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.initLocked()
	if _, exists := r.children[childPID]; exists {
		return false
	}
	r.children[childPID] = &ChildState{PID: childPID, ParentPID: parentPID}
	return true
}

func (r *ChildRegistry) MarkExited(childPID uint32, exitCode int32) bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.initLocked()
	child := r.children[childPID]
	if child == nil || child.Reaped {
		return false
	}
	child.Exited = true
	child.ExitCode = exitCode
	r.cond.Broadcast()
	return true
}

func (r *ChildRegistry) findChildLocked(parentPID uint32, wantedPID int32) *ChildState {
	for _, child := range r.children {
		if child.ParentPID != parentPID || child.Reaped {
			continue
		}
		if wantedPID > 0 && child.PID != uint32(wantedPID) {
			continue
		}
		if child.Exited {
			return child
		}
	}
	for _, child := range r.children {
		if child.ParentPID != parentPID || child.Reaped {
			continue
		}
		if wantedPID > 0 && child.PID != uint32(wantedPID) {
			continue
		}
		return child
	}
	return nil
}

func (r *ChildRegistry) Wait(parentPID uint32, wantedPID int32, options uint32) (pid int32, status int32, err int32) {
	if r == nil || options&^WaitNoHang != 0 {
		return 0, 0, EINVAL
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.initLocked()
	for {
		candidate := r.findChildLocked(parentPID, wantedPID)
		if candidate == nil {
			return 0, 0, ECHILD
		}
		if !candidate.Exited {
			if options&WaitNoHang != 0 {
				return 0, 0, 0
			}
			r.cond.Wait()
			continue
		}
		candidate.Reaped = true
		return int32(candidate.PID), candidate.ExitCode << 8, 0
	}
}

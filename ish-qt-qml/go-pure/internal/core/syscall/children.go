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
// and the future kernel ProcessManager. It is deliberately independent of
// kernel.Process so syscall tests and lightweight embeddings can use it.
type ChildRegistry struct {
	mu       sync.Mutex
	children map[uint32]*ChildState
}

func NewChildRegistry() *ChildRegistry {
	return &ChildRegistry{children: make(map[uint32]*ChildState)}
}

func (r *ChildRegistry) AddChild(parentPID, childPID uint32) bool {
	if r == nil || childPID == 0 {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.children == nil {
		r.children = make(map[uint32]*ChildState)
	}
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
	child := r.children[childPID]
	if child == nil || child.Reaped {
		return false
	}
	child.Exited = true
	child.ExitCode = exitCode
	return true
}

func (r *ChildRegistry) Wait(parentPID uint32, wantedPID int32, options uint32) (pid int32, status int32, err int32) {
	if r == nil || options&^WaitNoHang != 0 {
		return 0, 0, EINVAL
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	var candidate *ChildState
	for _, child := range r.children {
		if child.ParentPID != parentPID || child.Reaped {
			continue
		}
		if wantedPID > 0 && child.PID != uint32(wantedPID) {
			continue
		}
		candidate = child
		if child.Exited {
			break
		}
	}
	if candidate == nil {
		return 0, 0, ECHILD
	}
	if !candidate.Exited {
		if options&WaitNoHang != 0 {
			return 0, 0, 0
		}
		// Blocking is owned by the future ProcessManager. Returning EINTR here
		// avoids spinning inside the guest executor while preserving the ABI.
		return 0, 0, EINTR
	}
	candidate.Reaped = true
	return int32(candidate.PID), candidate.ExitCode << 8, 0
}

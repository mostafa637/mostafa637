package syscall

const (
	waitIDTypeAll64  uint32 = 0
	waitIDTypePID64  uint32 = 1
	waitIDTypePGRP64 uint32 = 2
)

// WaitID selects a child for waitid(2). It returns the raw exit code because
// waitid reports it through siginfo rather than the wait4 status encoding.
func (r *ChildRegistry) WaitID(parentPID, idType, wantedPID uint32, options uint32) (pid uint32, exitCode int32, exited bool, err int32) {
	if r == nil || options&(WaitNoHang|WaitExited|WaitNoWait) == 0 || options&^(WaitNoHang|WaitExited|WaitNoWait) != 0 {
		return 0, 0, false, EINVAL
	}
	if idType != waitIDTypeAll64 && idType != waitIDTypePID64 && idType != waitIDTypePGRP64 {
		return 0, 0, false, EINVAL
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	var candidate *ChildState
	for _, child := range r.children {
		if child.ParentPID != parentPID || child.Reaped {
			continue
		}
		switch idType {
		case waitIDTypePID64:
			if child.PID != wantedPID {
				continue
			}
		case waitIDTypePGRP64:
			// ChildRegistry has no process-group field yet. The only safe
			// representable group is the child PID itself.
			if wantedPID != 0 && child.PID != wantedPID {
				continue
			}
		}
		candidate = child
		if child.Exited {
			break
		}
	}
	if candidate == nil {
		return 0, 0, false, ECHILD
	}
	if !candidate.Exited {
		if options&WaitNoHang != 0 {
			return 0, 0, false, 0
		}
		return 0, 0, false, EINTR
	}
	if options&WaitNoWait == 0 {
		candidate.Reaped = true
	}
	return candidate.PID, candidate.ExitCode, true, 0
}

package kernel

import (
	"fmt"
	"sync"

	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"
	coresyscall "github.com/mostafa637/mostafa637/go-pure/internal/core/syscall"
)

// ProcessManager owns the process graph around the single-process kernel
// object. Child creation currently allocates an independent Process; address
// space and descriptor cloning remain explicit follow-up work rather than an
// unsafe partial fork implementation.
type ProcessManager struct {
	mu      sync.Mutex
	nextPID uint32
	fake    *corefs.FS
	tasks   map[uint32]*Process
	parents map[uint32]*Process
}

func NewProcessManager(fake *corefs.FS, firstPID uint32) *ProcessManager {
	if firstPID == 0 {
		firstPID = 1
	}
	return &ProcessManager{
		nextPID: firstPID,
		fake:    fake,
		tasks:   make(map[uint32]*Process),
		parents: make(map[uint32]*Process),
	}
}

func (m *ProcessManager) NewRoot() *Process {
	m.mu.Lock()
	defer m.mu.Unlock()
	pid := m.allocatePIDLocked()
	process := NewProcess(pid, m.fake)
	m.tasks[pid] = process
	return process
}

// CreateChild creates a child task record and registers it with the parent's
// wait4 registry. It does not claim fork semantics until memory and FD cloning
// are implemented by the executor/kernel layer.
func (m *ProcessManager) CreateChild(parent *Process) (*Process, error) {
	if m == nil || parent == nil {
		return nil, fmt.Errorf("kernel: nil process manager or parent")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.tasks[parent.PID]; !exists {
		return nil, fmt.Errorf("kernel: parent %d is not registered", parent.PID)
	}
	pid := m.allocatePIDLocked()
	child := NewProcess(pid, parent.FS)
	child.Context.ParentPID = parent.PID
	if parent.Context.Children == nil {
		parent.Context.Children = coresyscall.NewChildRegistry()
	}
	if !parent.Context.Children.AddChild(parent.PID, child.PID) {
		_ = child.Close()
		return nil, fmt.Errorf("kernel: child %d already exists", child.PID)
	}
	m.tasks[pid] = child
	m.parents[pid] = parent
	return child, nil
}

func (m *ProcessManager) Lookup(pid uint32) (*Process, bool) {
	if m == nil {
		return nil, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	process, ok := m.tasks[pid]
	return process, ok
}

// Run executes a registered process and publishes its exit result to its
// parent registry as soon as the guest reaches exit/exit_group.
func (m *ProcessManager) Run(process *Process, maxSteps int) error {
	if m == nil || process == nil {
		return fmt.Errorf("kernel: nil process manager or process")
	}
	if _, ok := m.Lookup(process.PID); !ok {
		return fmt.Errorf("kernel: process %d is not registered", process.PID)
	}
	err := process.Run(maxSteps)
	m.MarkExited(process)
	return err
}

func (m *ProcessManager) MarkExited(process *Process) bool {
	if m == nil || process == nil {
		return false
	}
	code, exited := process.ExitCode()
	if !exited {
		return false
	}
	m.mu.Lock()
	parent := m.parents[process.PID]
	m.mu.Unlock()
	if parent == nil || parent.Context == nil || parent.Context.Children == nil {
		return true
	}
	return parent.Context.Children.MarkExited(process.PID, code)
}

func (m *ProcessManager) Close() error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	processes := make([]*Process, 0, len(m.tasks))
	for _, process := range m.tasks {
		processes = append(processes, process)
	}
	m.tasks = make(map[uint32]*Process)
	m.parents = make(map[uint32]*Process)
	m.mu.Unlock()
	var first error
	for _, process := range processes {
		if err := process.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (m *ProcessManager) allocatePIDLocked() uint32 {
	for {
		pid := m.nextPID
		m.nextPID++
		if m.nextPID == 0 {
			m.nextPID = 1
		}
		if pid != 0 {
			if _, exists := m.tasks[pid]; !exists {
				return pid
			}
		}
	}
}

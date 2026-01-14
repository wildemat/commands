package orchestrator

import (
	"sync"
	"time"
)

// ProcessStatus represents the current status of a process
type ProcessStatus string

const (
	StatusStopped  ProcessStatus = "stopped"
	StatusStarting ProcessStatus = "starting"
	StatusRunning  ProcessStatus = "running"
	StatusReady    ProcessStatus = "ready"
	StatusError    ProcessStatus = "error"
)

// ProcessMetrics holds resource usage metrics for a process
type ProcessMetrics struct {
	CPUPercent    float64
	MemoryMB      float64
	MemoryPercent float64
	NumThreads    int32
	NumFDs        int32
}

// ChildProcessInfo holds information about a child process
type ChildProcessInfo struct {
	PID        int
	Name       string
	CPUPercent float64
	MemoryMB   float64
	Command    string
}

// ProcessState holds the complete state of a process
type ProcessState struct {
	Name         string
	PID          int
	Status       ProcessStatus
	LogFile      string
	Command      string
	Dependencies []string
	Dependents   []string
	Metrics      *ProcessMetrics
	ChildPIDs    []int
	LastUpdated  time.Time
	Error        string
	mu           sync.RWMutex
}

// NewProcessState creates a new ProcessState instance
func NewProcessState(name string, logFile string, dependencies []string) *ProcessState {
	return &ProcessState{
		Name:         name,
		Status:       StatusStopped,
		LogFile:      logFile,
		Dependencies: dependencies,
		Dependents:   []string{},
		LastUpdated:  time.Now(),
	}
}

// SetStatus updates the process status
func (ps *ProcessState) SetStatus(status ProcessStatus) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.Status = status
	ps.LastUpdated = time.Now()
}

// GetStatus returns the current process status
func (ps *ProcessState) GetStatus() ProcessStatus {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.Status
}

// SetPID updates the process PID
func (ps *ProcessState) SetPID(pid int) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.PID = pid
	ps.LastUpdated = time.Now()
}

// GetPID returns the current process PID
func (ps *ProcessState) GetPID() int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.PID
}

// SetError sets an error message for the process
func (ps *ProcessState) SetError(err string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.Error = err
	ps.Status = StatusError
	ps.LastUpdated = time.Now()
}

// SetMetrics updates the process metrics
func (ps *ProcessState) SetMetrics(metrics *ProcessMetrics) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.Metrics = metrics
	ps.LastUpdated = time.Now()
}

// GetMetrics returns the current process metrics
func (ps *ProcessState) GetMetrics() *ProcessMetrics {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.Metrics
}

// SetChildPIDs updates the list of child PIDs
func (ps *ProcessState) SetChildPIDs(pids []int) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.ChildPIDs = pids
	ps.LastUpdated = time.Now()
}

// GetChildPIDs returns the list of child PIDs
func (ps *ProcessState) GetChildPIDs() []int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.ChildPIDs
}

// IsRunning returns true if the process is in a running or ready state
func (ps *ProcessState) IsRunning() bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.Status == StatusRunning || ps.Status == StatusReady || ps.Status == StatusStarting
}

// Clone creates a copy of the ProcessState (for safe reading)
func (ps *ProcessState) Clone() ProcessState {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	clone := ProcessState{
		Name:         ps.Name,
		PID:          ps.PID,
		Status:       ps.Status,
		LogFile:      ps.LogFile,
		Command:      ps.Command,
		Dependencies: make([]string, len(ps.Dependencies)),
		Dependents:   make([]string, len(ps.Dependents)),
		ChildPIDs:    make([]int, len(ps.ChildPIDs)),
		LastUpdated:  ps.LastUpdated,
		Error:        ps.Error,
	}

	copy(clone.Dependencies, ps.Dependencies)
	copy(clone.Dependents, ps.Dependents)
	copy(clone.ChildPIDs, ps.ChildPIDs)

	if ps.Metrics != nil {
		clone.Metrics = &ProcessMetrics{
			CPUPercent:    ps.Metrics.CPUPercent,
			MemoryMB:      ps.Metrics.MemoryMB,
			MemoryPercent: ps.Metrics.MemoryPercent,
			NumThreads:    ps.Metrics.NumThreads,
			NumFDs:        ps.Metrics.NumFDs,
		}
	}

	return clone
}


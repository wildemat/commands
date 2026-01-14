package orchestrator

import (
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

// MetricsCollector collects process metrics using gopsutil
type MetricsCollector struct {
	updateInterval time.Duration
}

// NewMetricsCollector creates a new MetricsCollector
func NewMetricsCollector(interval time.Duration) *MetricsCollector {
	return &MetricsCollector{
		updateInterval: interval,
	}
}

// CollectMetrics gathers CPU, memory, thread, and FD information for a process
func (mc *MetricsCollector) CollectMetrics(pid int) (*ProcessMetrics, error) {
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return nil, err
	}

	metrics := &ProcessMetrics{}

	// CPU percent
	cpuPercent, err := proc.CPUPercent()
	if err == nil {
		metrics.CPUPercent = cpuPercent
	}

	// Memory info
	memInfo, err := proc.MemoryInfo()
	if err == nil {
		metrics.MemoryMB = float64(memInfo.RSS) / 1024 / 1024
	}

	memPercent, err := proc.MemoryPercent()
	if err == nil {
		metrics.MemoryPercent = float64(memPercent)
	}

	// Thread count
	numThreads, err := proc.NumThreads()
	if err == nil {
		metrics.NumThreads = numThreads
	}

	// File descriptors (may not be available on all platforms)
	numFDs, err := proc.NumFDs()
	if err == nil {
		metrics.NumFDs = numFDs
	}

	return metrics, nil
}

// GetChildProcesses returns all child process PIDs (recursively)
func (mc *MetricsCollector) GetChildProcesses(pid int) ([]int, error) {
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return nil, err
	}

	var allChildren []int
	err = mc.collectChildrenRecursive(proc, &allChildren)
	if err != nil {
		return nil, err
	}

	return allChildren, nil
}

func (mc *MetricsCollector) collectChildrenRecursive(proc *process.Process, result *[]int) error {
	children, err := proc.Children()
	if err != nil {
		// No children or error - that's okay
		return nil
	}

	for _, child := range children {
		*result = append(*result, int(child.Pid))
		mc.collectChildrenRecursive(child, result)
	}

	return nil
}

// GetChildProcessDetails returns detailed information about all child processes
func (mc *MetricsCollector) GetChildProcessDetails(pid int) ([]ChildProcessInfo, error) {
	childPIDs, err := mc.GetChildProcesses(pid)
	if err != nil {
		return nil, err
	}

	var details []ChildProcessInfo
	for _, childPID := range childPIDs {
		info, err := mc.getProcessInfo(childPID)
		if err != nil {
			continue // Skip processes that have exited
		}
		details = append(details, info)
	}

	return details, nil
}

func (mc *MetricsCollector) getProcessInfo(pid int) (ChildProcessInfo, error) {
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return ChildProcessInfo{}, err
	}

	info := ChildProcessInfo{
		PID: pid,
	}

	// Get process name
	name, err := proc.Name()
	if err == nil {
		info.Name = name
	}

	// Get CPU percent
	cpuPercent, err := proc.CPUPercent()
	if err == nil {
		info.CPUPercent = cpuPercent
	}

	// Get memory info
	memInfo, err := proc.MemoryInfo()
	if err == nil {
		info.MemoryMB = float64(memInfo.RSS) / 1024 / 1024
	}

	// Get command line
	cmdline, err := proc.Cmdline()
	if err == nil {
		info.Command = cmdline
	}

	return info, nil
}

// IsProcessRunning checks if a process with the given PID is still running
func (mc *MetricsCollector) IsProcessRunning(pid int) bool {
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return false
	}

	running, err := proc.IsRunning()
	if err != nil {
		return false
	}

	return running
}

// GetUpdateInterval returns the configured update interval
func (mc *MetricsCollector) GetUpdateInterval() time.Duration {
	return mc.updateInterval
}


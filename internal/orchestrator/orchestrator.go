package orchestrator

import (
	"fmt"
	"sync"
	"time"

	"github.com/wildmat/commands/internal/config"
)

// Orchestrator coordinates all process management
type Orchestrator struct {
	config           *config.Config
	processManager   *ProcessManager
	logManager       *LogManager
	metricsCollector *MetricsCollector
	depManager       *DependencyManager
	states           map[string]*ProcessState
	mu               sync.RWMutex
	stopChan         chan struct{}
	wg               sync.WaitGroup
}

// New creates a new Orchestrator instance
func New(cfg *config.Config) *Orchestrator {
	logManager := NewLogManager(cfg.LogDir, cfg.LogOpenCommand)
	metricsCollector := NewMetricsCollector(3 * time.Second)

	o := &Orchestrator{
		config:           cfg,
		logManager:       logManager,
		metricsCollector: metricsCollector,
		depManager:       NewDependencyManager(),
		states:           make(map[string]*ProcessState),
		stopChan:         make(chan struct{}),
	}

	o.processManager = NewProcessManager(cfg, logManager)

	// Initialize process states
	for name, procConfig := range cfg.Processes {
		state := NewProcessState(name, procConfig.LogFile, procConfig.Dependencies)
		state.Dependents = o.depManager.GetDependents(name)
		o.states[name] = state
		logManager.RegisterProcess(name, procConfig.LogFile)
	}

	return o
}

// StartAll starts all processes in dependency order
func (o *Orchestrator) StartAll() error {
	processNames := o.config.GetProcessNames()

	// Filter out Chrome processes - they're started after Kibana is ready
	var toStart []string
	for _, name := range processNames {
		if name != "sls_chrome" && name != "stack_chrome" {
			toStart = append(toStart, name)
		}
	}

	// Get start order
	startOrder, err := o.depManager.ResolveStartOrder(toStart)
	if err != nil {
		return fmt.Errorf("failed to resolve start order: %w", err)
	}

	// Kill any processes on ports we need
	ports := []int{9200, 9201, 9300, 9301, 5601, 5611}
	for _, port := range ports {
		o.processManager.KillProcessOnPort(port)
	}

	// Start processes in groups (processes in the same group can start concurrently)
	groups := o.depManager.GetStartGroups(startOrder)

	for _, group := range groups {
		var groupWg sync.WaitGroup
		var groupErrs []error
		var errMu sync.Mutex

		for _, name := range group {
			groupWg.Add(1)
			go func(processName string) {
				defer groupWg.Done()

				state := o.states[processName]
				if state == nil {
					return
				}

				// Create log file
				if err := o.logManager.CreateLogFile(processName); err != nil {
					errMu.Lock()
					groupErrs = append(groupErrs, fmt.Errorf("%s: %w", processName, err))
					errMu.Unlock()
					return
				}

				// Start the process
				if err := o.processManager.StartProcess(processName, state); err != nil {
					errMu.Lock()
					groupErrs = append(groupErrs, fmt.Errorf("%s: %w", processName, err))
					errMu.Unlock()
					return
				}

				// Wait for ready if it has a ready pattern
				if o.config.Processes[processName].ReadyPattern != "" {
					if err := o.processManager.WaitForReady(processName, 10*time.Minute); err != nil {
						errMu.Lock()
						groupErrs = append(groupErrs, fmt.Errorf("%s: %w", processName, err))
						errMu.Unlock()
						return
					}
				}
			}(name)
		}

		groupWg.Wait()

		if len(groupErrs) > 0 {
			return groupErrs[0]
		}
	}

	// Start Chrome processes after Kibana is ready
	go o.startChromeWhenReady("kbnsls", "sls_chrome")
	go o.startChromeWhenReady("kbnstack", "stack_chrome")

	// Start background monitoring
	o.startMonitoring()

	return nil
}

func (o *Orchestrator) startChromeWhenReady(kibanaProcess, chromeProcess string) {
	// Wait for Kibana to be ready
	for i := 0; i < 300; i++ { // Max 10 minutes
		select {
		case <-o.stopChan:
			return
		default:
		}

		state := o.GetState(kibanaProcess)
		if state != nil && state.Status == StatusReady {
			// Start Chrome
			chromeState := o.states[chromeProcess]
			if chromeState != nil {
				o.processManager.StartProcess(chromeProcess, chromeState)
			}
			return
		}

		time.Sleep(2 * time.Second)
	}
}

// StopAll stops all processes in reverse dependency order
func (o *Orchestrator) StopAll() error {
	// Signal monitoring to stop
	close(o.stopChan)

	processNames := o.config.GetProcessNames()

	// Get stop order
	stopOrder, err := o.depManager.ResolveStopOrder(processNames)
	if err != nil {
		return fmt.Errorf("failed to resolve stop order: %w", err)
	}

	// Stop processes in order
	for _, name := range stopOrder {
		if err := o.processManager.StopProcess(name); err != nil {
			// Log error but continue stopping other processes
			fmt.Printf("Error stopping %s: %v\n", name, err)
		}
	}

	// Cleanup Docker containers
	o.processManager.CleanupDockerContainers()

	// Wait for monitoring goroutines to finish
	o.wg.Wait()

	return nil
}

// Stop stops a single process and its dependents
func (o *Orchestrator) Stop(name string) error {
	// Get all dependents
	dependents := o.depManager.GetAllDependents(name)

	// Stop dependents first
	for _, dep := range dependents {
		if err := o.processManager.StopProcess(dep); err != nil {
			fmt.Printf("Error stopping %s: %v\n", dep, err)
		}
	}

	// Stop the target process
	return o.processManager.StopProcess(name)
}

// Restart restarts a process and its dependents
func (o *Orchestrator) Restart(name string) error {
	// Get running processes
	var running []string
	for pName := range o.states {
		if o.processManager.IsRunning(pName) {
			running = append(running, pName)
		}
	}

	// Get stop and start orders
	stopOrder, startOrder, err := o.depManager.ResolveRestartOrder(name, running)
	if err != nil {
		return fmt.Errorf("failed to resolve restart order: %w", err)
	}

	// Stop processes
	for _, pName := range stopOrder {
		if err := o.processManager.StopProcess(pName); err != nil {
			fmt.Printf("Error stopping %s: %v\n", pName, err)
		}
	}

	// Wait a moment for ports to be released
	time.Sleep(1 * time.Second)

	// Start processes
	for _, pName := range startOrder {
		state := o.states[pName]
		if state == nil {
			continue
		}

		// Create log file
		o.logManager.CreateLogFile(pName)

		// Start the process
		if err := o.processManager.StartProcess(pName, state); err != nil {
			return fmt.Errorf("failed to start %s: %w", pName, err)
		}

		// Wait for ready if applicable
		if o.config.Processes[pName].ReadyPattern != "" {
			if err := o.processManager.WaitForReady(pName, 10*time.Minute); err != nil {
				return fmt.Errorf("timeout waiting for %s: %w", pName, err)
			}
		}
	}

	return nil
}

// GetStatus returns the status of all processes
func (o *Orchestrator) GetStatus() map[string]ProcessState {
	o.mu.RLock()
	defer o.mu.RUnlock()

	result := make(map[string]ProcessState)
	for name, state := range o.states {
		result[name] = state.Clone()
	}
	return result
}

// GetState returns the state of a specific process
func (o *Orchestrator) GetState(name string) *ProcessState {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if state, ok := o.states[name]; ok {
		clone := state.Clone()
		return &clone
	}
	return nil
}

// GetProcessDetails returns detailed information about a process
func (o *Orchestrator) GetProcessDetails(name string) (*ProcessState, error) {
	state := o.GetState(name)
	if state == nil {
		return nil, fmt.Errorf("unknown process: %s", name)
	}

	// Update metrics if process is running
	if state.IsRunning() && state.PID > 0 {
		metrics, err := o.metricsCollector.CollectMetrics(state.PID)
		if err == nil {
			state.Metrics = metrics
		}

		children, err := o.metricsCollector.GetChildProcesses(state.PID)
		if err == nil {
			state.ChildPIDs = children
		}
	}

	return state, nil
}

// GetChildProcesses returns detailed info about child processes
func (o *Orchestrator) GetChildProcesses(name string) ([]ChildProcessInfo, error) {
	state := o.GetState(name)
	if state == nil {
		return nil, fmt.Errorf("unknown process: %s", name)
	}

	if !state.IsRunning() || state.PID <= 0 {
		return nil, nil
	}

	return o.metricsCollector.GetChildProcessDetails(state.PID)
}

// OpenLogFile opens the log file for a process
func (o *Orchestrator) OpenLogFile(name string) error {
	return o.logManager.OpenLogFile(name)
}

// GetLogTail returns the last n lines of a process's log
func (o *Orchestrator) GetLogTail(name string, lines int) ([]string, error) {
	return o.logManager.TailLogFile(name, lines)
}

// ListLogFiles returns all available log files
func (o *Orchestrator) ListLogFiles() ([]string, error) {
	return o.logManager.ListLogFiles()
}

// GetConfig returns the orchestrator configuration
func (o *Orchestrator) GetConfig() *config.Config {
	return o.config
}

// GetProcessNames returns all process names
func (o *Orchestrator) GetProcessNames() []string {
	return o.config.GetProcessNames()
}

func (o *Orchestrator) startMonitoring() {
	o.wg.Add(1)
	go func() {
		defer o.wg.Done()
		ticker := time.NewTicker(o.metricsCollector.GetUpdateInterval())
		defer ticker.Stop()

		for {
			select {
			case <-o.stopChan:
				return
			case <-ticker.C:
				o.updateAllMetrics()
			}
		}
	}()
}

func (o *Orchestrator) updateAllMetrics() {
	o.mu.Lock()
	defer o.mu.Unlock()

	for name, state := range o.states {
		if !state.IsRunning() {
			continue
		}

		pid := state.GetPID()
		if pid <= 0 {
			continue
		}

		// Check if process is still running
		if !o.metricsCollector.IsProcessRunning(pid) {
			state.SetStatus(StatusStopped)
			continue
		}

		// Update metrics
		metrics, err := o.metricsCollector.CollectMetrics(pid)
		if err == nil {
			state.SetMetrics(metrics)
		}

		// Update child processes
		children, err := o.metricsCollector.GetChildProcesses(pid)
		if err == nil {
			state.SetChildPIDs(children)
		}

		// Check if process became ready
		if state.GetStatus() == StatusRunning {
			procConfig := o.config.Processes[name]
			if procConfig != nil && procConfig.ReadyPattern != "" {
				found, _ := o.logManager.CheckLogForPattern(name, procConfig.ReadyPattern)
				if found {
					state.SetStatus(StatusReady)
				}
			}
		}
	}
}

// IsAnyRunning returns true if any process is running
func (o *Orchestrator) IsAnyRunning() bool {
	for _, state := range o.states {
		if state.IsRunning() {
			return true
		}
	}
	return false
}


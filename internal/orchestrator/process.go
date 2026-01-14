package orchestrator

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/wildmat/commands/internal/config"
)

// ProcessManager handles starting, stopping, and monitoring processes
type ProcessManager struct {
	config     *config.Config
	logManager *LogManager
	processes  map[string]*managedProcess
}

type managedProcess struct {
	cmd      *exec.Cmd
	state    *ProcessState
	logFile  *os.File
	stopChan chan struct{}
}

// NewProcessManager creates a new ProcessManager
func NewProcessManager(cfg *config.Config, logManager *LogManager) *ProcessManager {
	return &ProcessManager{
		config:     cfg,
		logManager: logManager,
		processes:  make(map[string]*managedProcess),
	}
}

// StartProcess starts a process and tracks its PID
func (pm *ProcessManager) StartProcess(name string, state *ProcessState) error {
	procConfig, ok := pm.config.Processes[name]
	if !ok {
		return fmt.Errorf("unknown process: %s", name)
	}

	// Handle Chrome processes specially
	if name == "sls_chrome" || name == "stack_chrome" {
		return pm.startChromeProcess(name, state, procConfig)
	}

	// Build the command
	cmdStr := pm.buildCommandString(procConfig)

	// Create log file
	logFile, err := os.OpenFile(procConfig.LogFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}

	// Build the full shell command with NVM setup
	fullCmd := pm.buildFullCommand(cmdStr, procConfig)

	cmd := exec.Command("bash", "-c", fullCmd)
	cmd.Dir = procConfig.WorkingDir
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range procConfig.EnvVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Set process group so we can kill all children
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Update state
	state.SetStatus(StatusStarting)
	state.Command = cmdStr

	// Start the process
	if err := cmd.Start(); err != nil {
		logFile.Close()
		state.SetError(err.Error())
		return fmt.Errorf("failed to start process: %w", err)
	}

	state.SetPID(cmd.Process.Pid)
	state.SetStatus(StatusRunning)

	// Store managed process
	pm.processes[name] = &managedProcess{
		cmd:      cmd,
		state:    state,
		logFile:  logFile,
		stopChan: make(chan struct{}),
	}

	// Start goroutine to wait for process and update state
	go pm.waitForProcess(name)

	return nil
}

func (pm *ProcessManager) startChromeProcess(name string, state *ProcessState, procConfig *config.ProcessConfig) error {
	port := procConfig.Port
	profileDir := fmt.Sprintf("/tmp/chrome-profile-%d", port)
	url := fmt.Sprintf("http://localhost:%d", port)

	cmd := exec.Command(
		pm.config.BrowserPath,
		"--incognito",
		fmt.Sprintf("--user-data-dir=%s", profileDir),
		url,
	)

	// Redirect stdout/stderr to /dev/null for Chrome
	cmd.Stdout = nil
	cmd.Stderr = nil

	state.SetStatus(StatusStarting)
	state.Command = fmt.Sprintf("%s --incognito --user-data-dir=%s %s", pm.config.BrowserPath, profileDir, url)

	if err := cmd.Start(); err != nil {
		state.SetError(err.Error())
		return fmt.Errorf("failed to start Chrome: %w", err)
	}

	state.SetPID(cmd.Process.Pid)
	state.SetStatus(StatusRunning)

	pm.processes[name] = &managedProcess{
		cmd:      cmd,
		state:    state,
		stopChan: make(chan struct{}),
	}

	go pm.waitForProcess(name)

	return nil
}

func (pm *ProcessManager) buildCommandString(procConfig *config.ProcessConfig) string {
	cmd := procConfig.Command

	// Append extra args if any
	if len(procConfig.ExtraArgs) > 0 {
		cmd = cmd + " " + strings.Join(procConfig.ExtraArgs, " ")
	}

	return cmd
}

func (pm *ProcessManager) buildFullCommand(cmdStr string, procConfig *config.ProcessConfig) string {
	var parts []string

	// Setup NVM
	parts = append(parts, fmt.Sprintf(`export NVM_DIR="%s"`, pm.config.NVMDir))
	parts = append(parts, `[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"`)
	parts = append(parts, "nvm use")

	// Apply memory limit if set
	if procConfig.MemoryLimit != "" {
		limitKB := pm.parseMemoryLimit(procConfig.MemoryLimit)
		if limitKB > 0 {
			parts = append(parts, fmt.Sprintf("ulimit -v %d", limitKB))
		}
	}

	// Add the actual command
	parts = append(parts, cmdStr)

	return strings.Join(parts, " && ")
}

func (pm *ProcessManager) parseMemoryLimit(limit string) int64 {
	limit = strings.ToUpper(strings.TrimSpace(limit))
	if limit == "" {
		return 0
	}

	var multiplier int64 = 1
	var numStr string

	if strings.HasSuffix(limit, "G") {
		multiplier = 1024 * 1024 // KB in a GB
		numStr = strings.TrimSuffix(limit, "G")
	} else if strings.HasSuffix(limit, "M") {
		multiplier = 1024 // KB in a MB
		numStr = strings.TrimSuffix(limit, "M")
	} else if strings.HasSuffix(limit, "K") {
		multiplier = 1
		numStr = strings.TrimSuffix(limit, "K")
	} else {
		// Assume it's already in KB
		numStr = limit
	}

	num, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return 0
	}

	return num * multiplier
}

func (pm *ProcessManager) waitForProcess(name string) {
	mp, ok := pm.processes[name]
	if !ok {
		return
	}

	// Wait for process to exit
	err := mp.cmd.Wait()

	// Close log file if open
	if mp.logFile != nil {
		mp.logFile.Close()
	}

	// Update state based on exit
	select {
	case <-mp.stopChan:
		// Process was intentionally stopped
		mp.state.SetStatus(StatusStopped)
	default:
		// Process exited on its own
		if err != nil {
			mp.state.SetError(fmt.Sprintf("process exited with error: %v", err))
		} else {
			mp.state.SetStatus(StatusStopped)
		}
	}
}

// StopProcess stops a process by name
func (pm *ProcessManager) StopProcess(name string) error {
	mp, ok := pm.processes[name]
	if !ok {
		return nil // Process not managed
	}

	// Signal that we're intentionally stopping
	close(mp.stopChan)

	// Kill the process group to ensure all children are killed
	if mp.cmd.Process != nil {
		pgid, err := syscall.Getpgid(mp.cmd.Process.Pid)
		if err == nil {
			syscall.Kill(-pgid, syscall.SIGTERM)
			// Give it a moment to terminate gracefully
			time.Sleep(500 * time.Millisecond)
			syscall.Kill(-pgid, syscall.SIGKILL)
		} else {
			// Fallback to killing just the process
			mp.cmd.Process.Kill()
		}
	}

	mp.state.SetStatus(StatusStopped)
	delete(pm.processes, name)

	return nil
}

// IsRunning checks if a process is currently running
func (pm *ProcessManager) IsRunning(name string) bool {
	mp, ok := pm.processes[name]
	if !ok {
		return false
	}
	return mp.state.IsRunning()
}

// GetPID returns the PID of a running process
func (pm *ProcessManager) GetPID(name string) int {
	mp, ok := pm.processes[name]
	if !ok {
		return 0
	}
	return mp.state.GetPID()
}

// WaitForReady waits for a process to be ready based on log pattern
func (pm *ProcessManager) WaitForReady(name string, timeout time.Duration) error {
	procConfig, ok := pm.config.Processes[name]
	if !ok {
		return fmt.Errorf("unknown process: %s", name)
	}

	if procConfig.ReadyPattern == "" {
		// No ready pattern, consider it ready immediately
		mp, ok := pm.processes[name]
		if ok {
			mp.state.SetStatus(StatusReady)
		}
		return nil
	}

	re, err := regexp.Compile(procConfig.ReadyPattern)
	if err != nil {
		return fmt.Errorf("invalid ready pattern: %w", err)
	}

	deadline := time.Now().Add(timeout)
	checkInterval := 500 * time.Millisecond

	for time.Now().Before(deadline) {
		// Check if process is still running
		if !pm.IsRunning(name) {
			return fmt.Errorf("process %s is not running", name)
		}

		// Check log file for ready pattern
		if pm.checkLogPattern(procConfig.LogFile, re) {
			mp, ok := pm.processes[name]
			if ok {
				mp.state.SetStatus(StatusReady)
			}
			return nil
		}

		time.Sleep(checkInterval)
	}

	return fmt.Errorf("timeout waiting for %s to be ready", name)
}

func (pm *ProcessManager) checkLogPattern(logPath string, pattern *regexp.Regexp) bool {
	file, err := os.Open(logPath)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		if pattern.MatchString(scanner.Text()) {
			return true
		}
	}

	return false
}

// KillProcessOnPort kills any process listening on the specified port
func (pm *ProcessManager) KillProcessOnPort(port int) error {
	// Use lsof to find the process
	cmd := exec.Command("lsof", "-ti", fmt.Sprintf("tcp:%d", port))
	output, err := cmd.Output()
	if err != nil {
		// No process found on port
		return nil
	}

	pids := strings.Fields(string(output))
	for _, pidStr := range pids {
		pid, err := strconv.Atoi(strings.TrimSpace(pidStr))
		if err != nil {
			continue
		}
		syscall.Kill(pid, syscall.SIGKILL)
	}

	return nil
}

// GetState returns the state of a managed process
func (pm *ProcessManager) GetState(name string) *ProcessState {
	mp, ok := pm.processes[name]
	if !ok {
		return nil
	}
	return mp.state
}

// CleanupDockerContainers stops ES-related Docker containers
func (pm *ProcessManager) CleanupDockerContainers() error {
	// Find ES-related containers
	cmd := exec.Command("docker", "ps", "-q", "--filter", "name=es", "--filter", "name=elasticsearch")
	output, err := cmd.Output()
	if err != nil {
		return nil // Docker not available or no containers
	}

	containerIDs := strings.Fields(string(output))
	for _, containerID := range containerIDs {
		exec.Command("docker", "stop", containerID).Run()
		exec.Command("docker", "rm", containerID).Run()
	}

	return nil
}


package orchestrator

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// LogManager handles log file operations
type LogManager struct {
	logDir      string
	openCommand string
	processes   map[string]string // process name -> log file path
}

// NewLogManager creates a new LogManager
func NewLogManager(logDir, openCommand string) *LogManager {
	return &LogManager{
		logDir:      logDir,
		openCommand: openCommand,
		processes:   make(map[string]string),
	}
}

// RegisterProcess registers a process and its log file
func (lm *LogManager) RegisterProcess(name, logFile string) {
	lm.processes[name] = logFile
}

// GetLogFile returns the log file path for a process
func (lm *LogManager) GetLogFile(processName string) (string, error) {
	if logFile, ok := lm.processes[processName]; ok {
		return logFile, nil
	}
	// Fall back to default path
	return filepath.Join(lm.logDir, processName+".log"), nil
}

// OpenLogFile opens a log file using the configured command
func (lm *LogManager) OpenLogFile(processName string) error {
	logPath, err := lm.GetLogFile(processName)
	if err != nil {
		return err
	}

	// Check if file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return fmt.Errorf("log file does not exist: %s", logPath)
	}

	// Parse and execute the open command
	return lm.executeOpenCommand(logPath)
}

func (lm *LogManager) executeOpenCommand(logPath string) error {
	cmdStr := lm.openCommand

	// Replace {logfile} placeholder if present
	if strings.Contains(cmdStr, "{logfile}") {
		cmdStr = strings.ReplaceAll(cmdStr, "{logfile}", logPath)
	} else {
		// Append log path as argument
		cmdStr = cmdStr + " " + logPath
	}

	// Parse command and arguments
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return fmt.Errorf("empty open command")
	}

	// Handle commands like "tail -f" that need shell execution
	if strings.Contains(cmdStr, " ") && !strings.HasPrefix(parts[0], "/") {
		cmd := exec.Command("sh", "-c", cmdStr)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Start()
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Start()
}

// TailLogFile streams the log file content (like tail -f)
func (lm *LogManager) TailLogFile(processName string, lines int) ([]string, error) {
	logPath, err := lm.GetLogFile(processName)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(logPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read all lines
	var allLines []string
	scanner := bufio.NewScanner(file)
	// Increase buffer size for long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Return last n lines
	if len(allLines) <= lines {
		return allLines, nil
	}
	return allLines[len(allLines)-lines:], nil
}

// WatchLogForPattern monitors a log file for a specific pattern
func (lm *LogManager) WatchLogForPattern(processName string, pattern string, stopChan <-chan struct{}) (bool, error) {
	logPath, err := lm.GetLogFile(processName)
	if err != nil {
		return false, err
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, fmt.Errorf("invalid pattern: %w", err)
	}

	file, err := os.Open(logPath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Seek to end to start watching new content
	_, err = file.Seek(0, 2)
	if err != nil {
		return false, err
	}

	reader := bufio.NewReader(file)

	for {
		select {
		case <-stopChan:
			return false, nil
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				// No new content, wait a bit
				continue
			}

			if re.MatchString(line) {
				return true, nil
			}
		}
	}
}

// CheckLogForPattern checks if a pattern exists in the log file
func (lm *LogManager) CheckLogForPattern(processName string, pattern string) (bool, error) {
	logPath, err := lm.GetLogFile(processName)
	if err != nil {
		return false, err
	}

	// Check if file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return false, nil
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, fmt.Errorf("invalid pattern: %w", err)
	}

	file, err := os.Open(logPath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		if re.MatchString(scanner.Text()) {
			return true, nil
		}
	}

	return false, scanner.Err()
}

// ListLogFiles returns all available log files
func (lm *LogManager) ListLogFiles() ([]string, error) {
	var logFiles []string

	// Add registered process log files
	for name := range lm.processes {
		logFile, _ := lm.GetLogFile(name)
		if _, err := os.Stat(logFile); err == nil {
			logFiles = append(logFiles, logFile)
		}
	}

	// Also include stack.log if it exists
	stackLog := filepath.Join(lm.logDir, "stack.log")
	if _, err := os.Stat(stackLog); err == nil {
		logFiles = append(logFiles, stackLog)
	}

	return logFiles, nil
}

// ClearLogFile truncates a log file
func (lm *LogManager) ClearLogFile(processName string) error {
	logPath, err := lm.GetLogFile(processName)
	if err != nil {
		return err
	}

	return os.Truncate(logPath, 0)
}

// CreateLogFile creates an empty log file (or truncates existing)
func (lm *LogManager) CreateLogFile(processName string) error {
	logPath, err := lm.GetLogFile(processName)
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(logPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.Create(logPath)
	if err != nil {
		return err
	}
	return file.Close()
}

// GetLogDir returns the log directory path
func (lm *LogManager) GetLogDir() string {
	return lm.logDir
}


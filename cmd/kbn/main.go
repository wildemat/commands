package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/wildmat/commands/internal/config"
	"github.com/wildmat/commands/internal/orchestrator"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Create orchestrator
	orch := orchestrator.New(cfg)

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n\nReceived interrupt signal...")
		handleShutdown(orch)
	}()

	// Run interactive menu
	runMenu(orch)
}

func handleShutdown(orch *orchestrator.Orchestrator) {
	if !orch.IsAnyRunning() {
		fmt.Println("No processes running. Exiting.")
		os.Exit(0)
	}

	prompt := promptui.Select{
		Label: "Kill all processes?",
		Items: []string{"Yes - Stop all processes", "No - Leave processes running"},
	}

	_, result, err := prompt.Run()
	if err != nil {
		fmt.Println("Leaving processes running. Exiting.")
		os.Exit(0)
	}

	if strings.HasPrefix(result, "Yes") {
		fmt.Println("Stopping all processes...")
		if err := orch.StopAll(); err != nil {
			fmt.Printf("Error stopping processes: %v\n", err)
		}
		fmt.Println("All processes stopped.")
	} else {
		fmt.Println("Leaving processes running. Exiting.")
	}

	os.Exit(0)
}

func runMenu(orch *orchestrator.Orchestrator) {
	for {
		// Display current status
		fmt.Println("\n" + strings.Repeat("=", 60))
		fmt.Println("  Kibana Process Orchestrator")
		fmt.Println(strings.Repeat("=", 60))
		displayStatus(orch)
		fmt.Println(strings.Repeat("-", 60))

		prompt := promptui.Select{
			Label: "Select action",
			Items: []string{
				"Start all processes",
				"Stop all processes",
				"Restart process",
				"Stop process",
				"View process details",
				"View logs",
				"Refresh status",
				"Exit",
			},
			Size: 10,
		}

		_, result, err := prompt.Run()
		if err != nil {
			if err == promptui.ErrInterrupt {
				handleShutdown(orch)
			}
			continue
		}

		switch result {
		case "Start all processes":
			startAllProcesses(orch)
		case "Stop all processes":
			stopAllProcesses(orch)
		case "Restart process":
			restartProcess(orch)
		case "Stop process":
			stopProcess(orch)
		case "View process details":
			viewProcessDetails(orch)
		case "View logs":
			viewLogs(orch)
		case "Refresh status":
			// Just redraw the menu
			continue
		case "Exit":
			handleShutdown(orch)
		}
	}
}

func displayStatus(orch *orchestrator.Orchestrator) {
	status := orch.GetStatus()
	processNames := orch.GetProcessNames()

	fmt.Printf("\n  %-15s %-10s %-10s %s\n", "Process", "Status", "PID", "Info")
	fmt.Println("  " + strings.Repeat("-", 55))

	for _, name := range processNames {
		state, ok := status[name]
		if !ok {
			continue
		}

		statusStr := colorStatus(state.Status)
		pidStr := "-"
		if state.PID > 0 {
			pidStr = fmt.Sprintf("%d", state.PID)
		}

		infoStr := ""
		if state.Metrics != nil {
			infoStr = fmt.Sprintf("CPU: %.1f%% MEM: %.0fMB", state.Metrics.CPUPercent, state.Metrics.MemoryMB)
		}
		if state.Error != "" {
			infoStr = truncate(state.Error, 30)
		}

		fmt.Printf("  %-15s %-10s %-10s %s\n", name, statusStr, pidStr, infoStr)
	}
}

func colorStatus(status orchestrator.ProcessStatus) string {
	switch status {
	case orchestrator.StatusReady:
		return "\033[32m" + string(status) + "\033[0m" // Green
	case orchestrator.StatusRunning, orchestrator.StatusStarting:
		return "\033[33m" + string(status) + "\033[0m" // Yellow
	case orchestrator.StatusError:
		return "\033[31m" + string(status) + "\033[0m" // Red
	default:
		return string(status)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func startAllProcesses(orch *orchestrator.Orchestrator) {
	fmt.Println("\nStarting all processes...")
	fmt.Println("This may take several minutes while processes initialize.")
	fmt.Println()

	if err := orch.StartAll(); err != nil {
		fmt.Printf("\033[31mError: %v\033[0m\n", err)
		return
	}

	fmt.Println("\033[32mAll processes started successfully!\033[0m")
}

func stopAllProcesses(orch *orchestrator.Orchestrator) {
	prompt := promptui.Select{
		Label: "Are you sure you want to stop all processes?",
		Items: []string{"Yes", "No"},
	}

	_, result, err := prompt.Run()
	if err != nil || result != "Yes" {
		return
	}

	fmt.Println("\nStopping all processes...")
	if err := orch.StopAll(); err != nil {
		fmt.Printf("\033[31mError: %v\033[0m\n", err)
		return
	}

	fmt.Println("\033[32mAll processes stopped.\033[0m")
}

func selectProcess(orch *orchestrator.Orchestrator, filterRunning bool) (string, bool) {
	processNames := orch.GetProcessNames()
	status := orch.GetStatus()

	var items []string
	for _, name := range processNames {
		if filterRunning {
			state := status[name]
			if !state.IsRunning() {
				continue
			}
		}
		items = append(items, name)
	}

	if len(items) == 0 {
		fmt.Println("No processes available.")
		return "", false
	}

	items = append(items, "← Back to main menu")

	prompt := promptui.Select{
		Label: "Select process",
		Items: items,
		Size:  10,
	}

	_, result, err := prompt.Run()
	if err != nil || result == "← Back to main menu" {
		return "", false
	}

	return result, true
}

func restartProcess(orch *orchestrator.Orchestrator) {
	name, ok := selectProcess(orch, false)
	if !ok {
		return
	}

	fmt.Printf("\nRestarting %s (and its dependents)...\n", name)
	if err := orch.Restart(name); err != nil {
		fmt.Printf("\033[31mError: %v\033[0m\n", err)
		return
	}

	fmt.Printf("\033[32m%s restarted successfully!\033[0m\n", name)
}

func stopProcess(orch *orchestrator.Orchestrator) {
	name, ok := selectProcess(orch, true)
	if !ok {
		return
	}

	fmt.Printf("\nStopping %s...\n", name)
	if err := orch.Stop(name); err != nil {
		fmt.Printf("\033[31mError: %v\033[0m\n", err)
		return
	}

	fmt.Printf("\033[32m%s stopped.\033[0m\n", name)
}

func viewProcessDetails(orch *orchestrator.Orchestrator) {
	name, ok := selectProcess(orch, false)
	if !ok {
		return
	}

	for {
		details, err := orch.GetProcessDetails(name)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		// Display details
		fmt.Println("\n" + strings.Repeat("=", 50))
		fmt.Printf("  Process: %s\n", details.Name)
		fmt.Println(strings.Repeat("=", 50))
		fmt.Printf("  Status:       %s\n", colorStatus(details.Status))
		fmt.Printf("  PID:          %d\n", details.PID)
		fmt.Printf("  Log File:     %s\n", details.LogFile)
		fmt.Printf("  Dependencies: %s\n", strings.Join(details.Dependencies, ", "))
		fmt.Printf("  Dependents:   %s\n", strings.Join(details.Dependents, ", "))
		fmt.Printf("  Last Updated: %s\n", details.LastUpdated.Format(time.RFC3339))

		if details.Error != "" {
			fmt.Printf("  Error:        \033[31m%s\033[0m\n", details.Error)
		}

		if details.Metrics != nil {
			fmt.Println("\n  Metrics:")
			fmt.Printf("    CPU:         %.2f%%\n", details.Metrics.CPUPercent)
			fmt.Printf("    Memory:      %.2f MB (%.2f%%)\n", details.Metrics.MemoryMB, details.Metrics.MemoryPercent)
			fmt.Printf("    Threads:     %d\n", details.Metrics.NumThreads)
			fmt.Printf("    File Desc:   %d\n", details.Metrics.NumFDs)
		}

		if len(details.ChildPIDs) > 0 {
			fmt.Printf("\n  Child Processes: %d\n", len(details.ChildPIDs))
		}

		fmt.Println(strings.Repeat("-", 50))

		// Actions menu
		prompt := promptui.Select{
			Label: "Action",
			Items: []string{
				"Restart process",
				"Stop process",
				"View child processes",
				"Open log file",
				"View log tail",
				"Refresh",
				"← Back to main menu",
			},
		}

		_, action, err := prompt.Run()
		if err != nil || action == "← Back to main menu" {
			return
		}

		switch action {
		case "Restart process":
			fmt.Printf("\nRestarting %s...\n", name)
			if err := orch.Restart(name); err != nil {
				fmt.Printf("\033[31mError: %v\033[0m\n", err)
			} else {
				fmt.Println("\033[32mRestarted successfully!\033[0m")
			}
		case "Stop process":
			fmt.Printf("\nStopping %s...\n", name)
			if err := orch.Stop(name); err != nil {
				fmt.Printf("\033[31mError: %v\033[0m\n", err)
			} else {
				fmt.Println("\033[32mStopped.\033[0m")
			}
			return
		case "View child processes":
			viewChildProcesses(orch, name)
		case "Open log file":
			if err := orch.OpenLogFile(name); err != nil {
				fmt.Printf("\033[31mError opening log: %v\033[0m\n", err)
			} else {
				fmt.Println("Log file opened.")
			}
		case "View log tail":
			viewLogTail(orch, name)
		case "Refresh":
			// Just redraw
		}
	}
}

func viewChildProcesses(orch *orchestrator.Orchestrator, name string) {
	children, err := orch.GetChildProcesses(name)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if len(children) == 0 {
		fmt.Println("\nNo child processes.")
		return
	}

	fmt.Println("\n" + strings.Repeat("-", 70))
	fmt.Printf("  %-10s %-15s %-10s %-10s %s\n", "PID", "Name", "CPU%", "MEM (MB)", "Command")
	fmt.Println("  " + strings.Repeat("-", 65))

	for _, child := range children {
		cmdTrunc := truncate(child.Command, 30)
		fmt.Printf("  %-10d %-15s %-10.1f %-10.1f %s\n",
			child.PID, truncate(child.Name, 15), child.CPUPercent, child.MemoryMB, cmdTrunc)
	}

	fmt.Println()
	fmt.Println("Press Enter to continue...")
	fmt.Scanln()
}

func viewLogTail(orch *orchestrator.Orchestrator, name string) {
	lines, err := orch.GetLogTail(name, 50)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Printf("  Last 50 lines of %s.log:\n", name)
	fmt.Println(strings.Repeat("=", 70))

	for _, line := range lines {
		// Truncate very long lines
		if len(line) > 120 {
			line = line[:117] + "..."
		}
		fmt.Println(line)
	}

	fmt.Println()
	fmt.Println("Press Enter to continue...")
	fmt.Scanln()
}

func viewLogs(orch *orchestrator.Orchestrator) {
	logFiles, err := orch.ListLogFiles()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	items := make([]string, len(logFiles))
	for i, f := range logFiles {
		items[i] = f
	}
	items = append(items, "← Back to main menu")

	prompt := promptui.Select{
		Label: "Select log file to open",
		Items: items,
		Size:  10,
	}

	_, result, err := prompt.Run()
	if err != nil || result == "← Back to main menu" {
		return
	}

	// Extract process name from log file path
	processNames := orch.GetProcessNames()
	for _, name := range processNames {
		if strings.Contains(result, name) {
			if err := orch.OpenLogFile(name); err != nil {
				fmt.Printf("\033[31mError opening log: %v\033[0m\n", err)
			} else {
				fmt.Println("Log file opened.")
			}
			return
		}
	}
}


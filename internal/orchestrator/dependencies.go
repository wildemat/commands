package orchestrator

import (
	"fmt"

	"github.com/wildmat/commands/internal/config"
)

// DependencyManager handles process dependency resolution
type DependencyManager struct {
	dependencies map[string][]string // process -> what it depends on
	dependents   map[string][]string // process -> what depends on it
}

// NewDependencyManager creates a new DependencyManager
func NewDependencyManager() *DependencyManager {
	dm := &DependencyManager{
		dependencies: make(map[string][]string),
		dependents:   make(map[string][]string),
	}

	// Initialize from config
	for process, deps := range config.ProcessDependencies {
		dm.dependencies[process] = deps
		// Build reverse map
		for _, dep := range deps {
			dm.dependents[dep] = append(dm.dependents[dep], process)
		}
	}

	return dm
}

// GetDependencies returns all processes that the given process depends on
func (dm *DependencyManager) GetDependencies(name string) []string {
	if deps, ok := dm.dependencies[name]; ok {
		return deps
	}
	return []string{}
}

// GetDependents returns all processes that depend on the given process
func (dm *DependencyManager) GetDependents(name string) []string {
	if deps, ok := dm.dependents[name]; ok {
		return deps
	}
	return []string{}
}

// GetAllDependents returns all processes that depend on the given process (recursively)
func (dm *DependencyManager) GetAllDependents(name string) []string {
	visited := make(map[string]bool)
	result := []string{}
	dm.collectDependents(name, visited, &result)
	return result
}

func (dm *DependencyManager) collectDependents(name string, visited map[string]bool, result *[]string) {
	for _, dep := range dm.dependents[name] {
		if !visited[dep] {
			visited[dep] = true
			*result = append(*result, dep)
			dm.collectDependents(dep, visited, result)
		}
	}
}

// GetAllDependencies returns all processes that the given process depends on (recursively)
func (dm *DependencyManager) GetAllDependencies(name string) []string {
	visited := make(map[string]bool)
	result := []string{}
	dm.collectDependencies(name, visited, &result)
	return result
}

func (dm *DependencyManager) collectDependencies(name string, visited map[string]bool, result *[]string) {
	for _, dep := range dm.dependencies[name] {
		if !visited[dep] {
			visited[dep] = true
			dm.collectDependencies(dep, visited, result)
			*result = append(*result, dep)
		}
	}
}

// ResolveStartOrder returns processes in the order they should be started
// Uses topological sort to ensure dependencies are started first
func (dm *DependencyManager) ResolveStartOrder(processes []string) ([]string, error) {
	// Build in-degree map for the requested processes
	inDegree := make(map[string]int)
	graph := make(map[string][]string)

	processSet := make(map[string]bool)
	for _, p := range processes {
		processSet[p] = true
		inDegree[p] = 0
	}

	// Build graph for only the requested processes
	for _, p := range processes {
		for _, dep := range dm.dependencies[p] {
			if processSet[dep] {
				graph[dep] = append(graph[dep], p)
				inDegree[p]++
			}
		}
	}

	// Kahn's algorithm for topological sort
	var queue []string
	for p, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, p)
		}
	}

	var result []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		for _, neighbor := range graph[node] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if len(result) != len(processes) {
		return nil, fmt.Errorf("circular dependency detected")
	}

	return result, nil
}

// ResolveStopOrder returns processes in the order they should be stopped
// (reverse of start order - stop dependents first)
func (dm *DependencyManager) ResolveStopOrder(processes []string) ([]string, error) {
	startOrder, err := dm.ResolveStartOrder(processes)
	if err != nil {
		return nil, err
	}

	// Reverse the order
	result := make([]string, len(startOrder))
	for i, p := range startOrder {
		result[len(startOrder)-1-i] = p
	}
	return result, nil
}

// ResolveRestartOrder returns processes in the order they should be restarted
// First stop dependents, then the process, then restart all
func (dm *DependencyManager) ResolveRestartOrder(name string, runningProcesses []string) (stopOrder []string, startOrder []string, err error) {
	// Get all dependents that need to be stopped
	allDependents := dm.GetAllDependents(name)

	// Filter to only include running processes
	runningSet := make(map[string]bool)
	for _, p := range runningProcesses {
		runningSet[p] = true
	}

	var toStop []string
	toStop = append(toStop, name) // Always include the target process
	for _, dep := range allDependents {
		if runningSet[dep] {
			toStop = append(toStop, dep)
		}
	}

	// Get stop order (dependents first)
	stopOrder, err = dm.ResolveStopOrder(toStop)
	if err != nil {
		return nil, nil, err
	}

	// Get start order (dependencies first)
	startOrder, err = dm.ResolveStartOrder(toStop)
	if err != nil {
		return nil, nil, err
	}

	return stopOrder, startOrder, nil
}

// CanStart checks if all dependencies of a process are ready
func (dm *DependencyManager) CanStart(name string, readyProcesses map[string]bool) bool {
	for _, dep := range dm.dependencies[name] {
		if !readyProcesses[dep] {
			return false
		}
	}
	return true
}

// GetStartGroups returns processes grouped by when they can be started
// (processes in the same group can be started concurrently)
func (dm *DependencyManager) GetStartGroups(processes []string) [][]string {
	remaining := make(map[string]bool)
	for _, p := range processes {
		remaining[p] = true
	}

	ready := make(map[string]bool)
	var groups [][]string

	for len(remaining) > 0 {
		var group []string
		for p := range remaining {
			if dm.CanStart(p, ready) {
				group = append(group, p)
			}
		}

		if len(group) == 0 {
			// This shouldn't happen if there are no circular dependencies
			// but add remaining to avoid infinite loop
			for p := range remaining {
				group = append(group, p)
			}
		}

		for _, p := range group {
			delete(remaining, p)
			ready[p] = true
		}
		groups = append(groups, group)
	}

	return groups
}


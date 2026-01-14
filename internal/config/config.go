package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// ProcessConfig holds configuration for a single process
type ProcessConfig struct {
	Name         string
	Command      string
	WorkingDir   string
	LogFile      string
	ReadyPattern string
	Dependencies []string
	EnvVars      map[string]string
	MemoryLimit  string
	ExtraArgs    []string
	Port         int
}

// Config holds all configuration for the orchestrator
type Config struct {
	KibanaDir      string
	LogDir         string
	NVMDir         string
	BrowserPath    string
	ScriptsDir     string
	LogOpenCommand string
	Processes      map[string]*ProcessConfig
}

// DefaultPorts for each process
var DefaultPorts = map[string]int{
	"essls":       9200,
	"esstack":     9201,
	"kbnsls":      5601,
	"kbnstack":    5611,
	"sls_chrome":  5601,
	"stack_chrome": 5611,
}

// ProcessDependencies defines which processes depend on which
var ProcessDependencies = map[string][]string{
	"optimizer":    {},
	"essls":        {},
	"esstack":      {},
	"kbnsls":       {"optimizer", "essls"},
	"kbnstack":     {"optimizer", "esstack"},
	"sls_chrome":   {"kbnsls"},
	"stack_chrome": {"kbnstack"},
}

// ProcessReadyPatterns defines the log patterns indicating process is ready
var ProcessReadyPatterns = map[string]string{
	"optimizer": `succ.*bundles compiled successfully|succ all bundles cached`,
	"essls":     `succ Serverless ES cluster running`,
	"esstack":   `succ ES cluster is ready`,
	"kbnsls":    `\[INFO \]\[status\] Kibana is now available`,
	"kbnstack":  `\[INFO \]\[status\] Kibana is now available`,
}

// LoadConfig loads configuration from .env file and environment variables
func LoadConfig() (*Config, error) {
	// Try to load .env file (silently ignore if missing)
	_ = godotenv.Load(".env")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	config := &Config{
		KibanaDir:      getEnvOrDefault("KIBANA_DIR", filepath.Join(homeDir, "workplace", "kibana")),
		LogDir:         getEnvOrDefault("LOG_DIR", filepath.Join(homeDir, "workplace", "local_dev_logs")),
		NVMDir:         getEnvOrDefault("NVM_DIR", filepath.Join(homeDir, ".nvm")),
		BrowserPath:    getEnvOrDefault("BROWSER_PATH", "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"),
		ScriptsDir:     getEnvOrDefault("SCRIPTS_DIR", "./scripts"),
		LogOpenCommand: getEnvOrDefault("LOG_OPEN_CMD", "open"),
		Processes:      make(map[string]*ProcessConfig),
	}

	// Ensure log directory exists
	if err := os.MkdirAll(config.LogDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Initialize process configurations
	config.initProcessConfigs()

	return config, nil
}

func (c *Config) initProcessConfigs() {
	processNames := []string{"optimizer", "essls", "esstack", "kbnsls", "kbnstack", "sls_chrome", "stack_chrome"}

	for _, name := range processNames {
		upperName := strings.ToUpper(name)

		// Get port
		port := DefaultPorts[name]
		if portStr := os.Getenv(fmt.Sprintf("PORT_%s", upperName)); portStr != "" {
			if p, err := strconv.Atoi(portStr); err == nil {
				port = p
			}
		}

		// Get ready pattern
		readyPattern := ProcessReadyPatterns[name]
		if pattern := os.Getenv(fmt.Sprintf("%s_READY_PATTERN", upperName)); pattern != "" {
			readyPattern = pattern
		}

		// Get custom log file path
		logFile := filepath.Join(c.LogDir, name+".log")
		if customLog := os.Getenv(fmt.Sprintf("%s_LOG_FILE", upperName)); customLog != "" {
			logFile = expandPath(customLog)
		}

		// Get memory limit
		memoryLimit := os.Getenv(fmt.Sprintf("%s_MEMORY_LIMIT", upperName))

		// Get extra args
		var extraArgs []string
		if args := os.Getenv(fmt.Sprintf("%s_EXTRA_ARGS", upperName)); args != "" {
			extraArgs = parseArgs(args)
		}

		// Get environment variables
		envVars := make(map[string]string)
		prefix := fmt.Sprintf("%s_ENV_", upperName)
		for _, env := range os.Environ() {
			if strings.HasPrefix(env, prefix) {
				parts := strings.SplitN(env, "=", 2)
				if len(parts) == 2 {
					varName := strings.TrimPrefix(parts[0], prefix)
					envVars[varName] = parts[1]
				}
			}
		}

		c.Processes[name] = &ProcessConfig{
			Name:         name,
			WorkingDir:   c.KibanaDir,
			LogFile:      logFile,
			ReadyPattern: readyPattern,
			Dependencies: ProcessDependencies[name],
			EnvVars:      envVars,
			MemoryLimit:  memoryLimit,
			ExtraArgs:    extraArgs,
			Port:         port,
		}
	}

	// Set default commands for each process
	c.setDefaultCommands()
}

func (c *Config) setDefaultCommands() {
	// Optimizer
	if cmd := os.Getenv("OPTIMIZER_CMD"); cmd != "" {
		c.Processes["optimizer"].Command = cmd
	} else {
		c.Processes["optimizer"].Command = "node scripts/build_kibana_platform_plugins --watch"
	}

	// ES Serverless
	if cmd := os.Getenv("ESSLS_CMD"); cmd != "" {
		c.Processes["essls"].Command = cmd
	} else {
		c.Processes["essls"].Command = "yarn es serverless --projectType elasticsearch_general_purpose --clean --kill -E xpack.inference.elastic.url=https://localhost:8443 -E xpack.inference.elastic.http.ssl.verification_mode=none"
	}

	// ES Stack
	if cmd := os.Getenv("ESSTACK_CMD"); cmd != "" {
		c.Processes["esstack"].Command = cmd
	} else {
		c.Processes["esstack"].Command = "yarn es snapshot --license trial --clean -E http.port=9201 -E transport.port=9301 -E xpack.inference.elastic.url=https://localhost:8443 -E xpack.inference.elastic.http.ssl.verification_mode=none"
	}

	// Kibana Serverless
	if cmd := os.Getenv("KBNSLS_CMD"); cmd != "" {
		c.Processes["kbnsls"].Command = cmd
	} else {
		c.Processes["kbnsls"].Command = "yarn serverless-es --config=config/kibana.serverless.dev.yml --server.port=5601 --no-optimizer"
	}

	// Kibana Stack
	if cmd := os.Getenv("KBNSTACK_CMD"); cmd != "" {
		c.Processes["kbnstack"].Command = cmd
	} else {
		c.Processes["kbnstack"].Command = "yarn start --config=config/kibana.stack.dev.yml --server.port=5611 --no-optimizer"
	}

	// Chrome processes use BrowserPath directly (handled in process.go)
	c.Processes["sls_chrome"].Command = ""
	c.Processes["stack_chrome"].Command = ""
}

// GetProcessNames returns all process names in dependency order
func (c *Config) GetProcessNames() []string {
	return []string{"optimizer", "essls", "esstack", "kbnsls", "kbnstack", "sls_chrome", "stack_chrome"}
}

// GetMainLogFile returns the path to the main stack log file
func (c *Config) GetMainLogFile() string {
	return filepath.Join(c.LogDir, "stack.log")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return expandPath(value)
	}
	return expandPath(defaultValue)
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

func parseArgs(args string) []string {
	// Simple space-separated parsing (doesn't handle quoted strings perfectly)
	var result []string
	for _, arg := range strings.Fields(args) {
		if arg != "" {
			result = append(result, arg)
		}
	}
	return result
}


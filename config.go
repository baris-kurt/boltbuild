package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the complete configuration for BoltBuild
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Client  ClientConfig  `yaml:"client"`
	Web     WebConfig     `yaml:"web"`
	Build   BuildConfig   `yaml:"build"`
	Logging LoggingConfig `yaml:"logging"`
}

// ServerConfig contains server-specific configuration
type ServerConfig struct {
	Port     int `yaml:"port"`
	Capacity int `yaml:"capacity"`
}

// ClientConfig contains client-specific configuration
type ClientConfig struct {
	Discovery DiscoveryConfig `yaml:"discovery"`
	Timeouts  TimeoutConfig   `yaml:"timeouts"`
}

// WebConfig contains web interface configuration
type WebConfig struct {
	Port int `yaml:"port"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level string `yaml:"level"` // "info", "debug"
}

// DiscoveryConfig contains server discovery settings
type DiscoveryConfig struct {
	Ports          []int         `yaml:"ports"`
	ScanInterval   time.Duration `yaml:"scan_interval"`
	ConnectTimeout time.Duration `yaml:"connect_timeout"`
	NetworkRange   NetworkRange  `yaml:"network_range"`
}

// NetworkRange defines the IP range for server discovery
type NetworkRange struct {
	Auto    bool   `yaml:"auto"`     // Auto-detect local network
	Subnet  string `yaml:"subnet"`   // e.g., "192.168.1"
	StartIP int    `yaml:"start_ip"` // Start IP in range (1-254)
	EndIP   int    `yaml:"end_ip"`   // End IP in range (1-254)
}

// TimeoutConfig contains various timeout settings
type TimeoutConfig struct {
	Build       time.Duration `yaml:"build"`
	Reconnect   time.Duration `yaml:"reconnect"`
	HealthCheck time.Duration `yaml:"health_check"`
}

// BuildConfig contains build system configurations
type BuildConfig struct {
	Environments map[string]BuildEnvironment `yaml:"environments"`
	TempDir      string                      `yaml:"temp_dir"`
	TempDeletion bool                        `yaml:"temp_deletion"`
}

// BuildEnvironment defines build settings for a specific language/environment
type BuildEnvironment struct {
	Name            string            `yaml:"name"`
	Command         string            `yaml:"command"`
	ProjectDir      string            `yaml:"project_dir"`
	ExecutionDir    string            `yaml:"execution_dir"`
	OutputPaths     []string          `yaml:"output_paths"`
	EnvVars         map[string]string `yaml:"env_vars"`
	PostBuildScript string            `yaml:"post_build_script"` // Script/executable to run on client after successful build
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:     8080,
			Capacity: 4,
		},
		Client: ClientConfig{
			Discovery: DiscoveryConfig{
				Ports:          []int{8080, 8081, 8082, 8083, 8084, 8085},
				ScanInterval:   10 * time.Second,
				ConnectTimeout: 2 * time.Second,
				NetworkRange: NetworkRange{
					Auto:    true,
					Subnet:  "",
					StartIP: 1,
					EndIP:   254,
				},
			},
			Timeouts: TimeoutConfig{
				Build:       120 * time.Second,
				Reconnect:   10 * time.Second,
				HealthCheck: 10 * time.Second,
			},
		},
		Web: WebConfig{
			Port: 8081,
		},
		Build: BuildConfig{
			TempDir:      "",   // Will use system temp dir if empty
			TempDeletion: true, // Default to deleting temp directories
			Environments: map[string]BuildEnvironment{},
		},
		Logging: LoggingConfig{
			Level: "info", // Default to info level (only show connections)
		},
	}
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(filename string) (*Config, error) {
	// Start with default config
	config := DefaultConfig()

	// Check if config file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// Create default config file
		if err := SaveConfig(config, filename); err != nil {
			return nil, fmt.Errorf("failed to create default config file: %v", err)
		}
		return config, nil
	}

	// Read config file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	// Validate and set defaults for missing fields
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %v", err)
	}

	return config, nil
}

// SaveConfig saves configuration to a YAML file
func SaveConfig(config *Config, filename string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	// Write to file
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate server config
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}
	if c.Server.Capacity <= 0 {
		return fmt.Errorf("invalid server capacity: %d", c.Server.Capacity)
	}

	// Validate web config
	if c.Web.Port <= 0 || c.Web.Port > 65535 {
		return fmt.Errorf("invalid web port: %d", c.Web.Port)
	}

	// Validate client discovery ports
	if len(c.Client.Discovery.Ports) == 0 {
		return fmt.Errorf("no discovery ports specified")
	}
	for _, port := range c.Client.Discovery.Ports {
		if port <= 0 || port > 65535 {
			return fmt.Errorf("invalid discovery port: %d", port)
		}
	}

	// Validate network range
	if !c.Client.Discovery.NetworkRange.Auto {
		if c.Client.Discovery.NetworkRange.Subnet == "" {
			return fmt.Errorf("subnet must be specified when auto-detection is disabled")
		}
		if c.Client.Discovery.NetworkRange.StartIP < 1 || c.Client.Discovery.NetworkRange.StartIP > 254 {
			return fmt.Errorf("invalid start IP: %d", c.Client.Discovery.NetworkRange.StartIP)
		}
		if c.Client.Discovery.NetworkRange.EndIP < 1 || c.Client.Discovery.NetworkRange.EndIP > 254 {
			return fmt.Errorf("invalid end IP: %d", c.Client.Discovery.NetworkRange.EndIP)
		}
		if c.Client.Discovery.NetworkRange.StartIP > c.Client.Discovery.NetworkRange.EndIP {
			return fmt.Errorf("start IP cannot be greater than end IP")
		}
	}

	// Validate timeouts
	if c.Client.Timeouts.Build <= 0 {
		return fmt.Errorf("invalid build timeout: %v", c.Client.Timeouts.Build)
	}
	if c.Client.Timeouts.Reconnect <= 0 {
		return fmt.Errorf("invalid reconnect timeout: %v", c.Client.Timeouts.Reconnect)
	}
	if c.Client.Timeouts.HealthCheck <= 0 {
		return fmt.Errorf("invalid health check timeout: %v", c.Client.Timeouts.HealthCheck)
	}

	// Validate build environments (only if they exist)
	for name, env := range c.Build.Environments {
		if env.Name == "" {
			return fmt.Errorf("name not specified for environment %s", name)
		}
		if env.Command == "" {
			return fmt.Errorf("command not specified for environment %s", name)
		}
		if env.ProjectDir == "" {
			return fmt.Errorf("project directory not specified for environment %s", name)
		}
		if env.ExecutionDir == "" {
			return fmt.Errorf("execution directory not specified for environment %s", name)
		}
	}

	return nil
}

// GetBuildEnvironment returns the build environment for a given language
func (c *Config) GetBuildEnvironment(language string) (*BuildEnvironment, bool) {
	env, exists := c.Build.Environments[language]
	return &env, exists
}

// GetTempDir returns the configured temp directory or system default
func (c *Config) GetTempDir() string {
	if c.Build.TempDir != "" {
		return c.Build.TempDir
	}
	return os.TempDir()
}

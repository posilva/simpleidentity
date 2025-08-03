package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	// Server configuration
	LogLevel        string        `mapstructure:"log-level"`
	LogPretty       bool          `mapstructure:"log-pretty"`
	HealthAddr      string        `mapstructure:"health-addr"`
	PprofAddr       string        `mapstructure:"pprof-addr"`
	GrpcAddr        string        `mapstructure:"grpc-addr"`
	HttpAddr        string        `mapstructure:"http-addr"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown-timeout"`
	Version         string        `mapstructure:"version"`
}

// Manager handles configuration loading and management
type Manager struct {
	viper *viper.Viper
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	v := viper.New()

	// Set up environment variable handling
	v.AutomaticEnv()
	v.SetEnvPrefix("SMPIDT")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	return &Manager{viper: v}
}

// SetDefaults sets default configuration values
func (m *Manager) SetDefaults() {
	// Server defaults
	m.viper.SetDefault("log-level", "info")
	m.viper.SetDefault("log-pretty", false)
	m.viper.SetDefault("health-addr", ":8080")
	m.viper.SetDefault("pprof-addr", ":6060")
	m.viper.SetDefault("grpc-addr", ":9090")
	m.viper.SetDefault("http-addr", ":8090")
	m.viper.SetDefault("shutdown-timeout", 30*time.Second)
	m.viper.SetDefault("version", "dev")
}

// Load loads configuration from environment variables and defaults
func (m *Manager) Load() (*Config, error) {
	m.SetDefaults()

	var config Config
	if err := m.viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	// Validate configuration
	if err := m.validate(&config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}

// BindFlags binds command line flags to the configuration
func (m *Manager) BindFlags(flags interface{}) error {
	// This will be used by cobra to bind flags
	// The interface{} parameter would typically be *pflag.FlagSet
	if flagSet, ok := flags.(interface{ VisitAll(func(interface{})) }); ok {
		// In a real implementation, you'd iterate through flags and bind them
		// For now, we'll use viper's automatic binding
		_ = flagSet
	}
	return nil
}

// validate performs configuration validation
func (m *Manager) validate(config *Config) error {
	// Validate log level
	validLogLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLogLevels, config.LogLevel) {
		return fmt.Errorf("invalid log level: %s, must be one of: %v", config.LogLevel, validLogLevels)
	}

	// Validate timeouts
	if config.ShutdownTimeout <= 0 {
		return fmt.Errorf("shutdown timeout must be positive, got: %v", config.ShutdownTimeout)
	}

	return nil
}

// Get returns a value from the configuration by key
func (m *Manager) Get(key string) interface{} {
	return m.viper.Get(key)
}

// GetString returns a string value from the configuration
func (m *Manager) GetString(key string) string {
	return m.viper.GetString(key)
}

// GetBool returns a bool value from the configuration
func (m *Manager) GetBool(key string) bool {
	return m.viper.GetBool(key)
}

// GetInt returns an int value from the configuration
func (m *Manager) GetInt(key string) int {
	return m.viper.GetInt(key)
}

// GetFloat64 returns a float64 value from the configuration
func (m *Manager) GetFloat64(key string) float64 {
	return m.viper.GetFloat64(key)
}

// GetDuration returns a duration value from the configuration
func (m *Manager) GetDuration(key string) time.Duration {
	return m.viper.GetDuration(key)
}

// Set sets a configuration value
func (m *Manager) Set(key string, value interface{}) {
	m.viper.Set(key, value)
}

// IsSet checks if a configuration key is set
func (m *Manager) IsSet(key string) bool {
	return m.viper.IsSet(key)
}

// AllSettings returns all configuration settings
func (m *Manager) AllSettings() map[string]interface{} {
	return m.viper.AllSettings()
}

// PrintConfig prints the current configuration (for debugging)
func (m *Manager) PrintConfig(config *Config) map[string]interface{} {
	settings := make(map[string]interface{})

	// Server settings
	settings["server"] = map[string]interface{}{
		"log_level":        config.LogLevel,
		"log_pretty":       config.LogPretty,
		"health_addr":      config.HealthAddr,
		"pprof_addr":       config.PprofAddr,
		"grpc_addr":        config.GrpcAddr,
		"http_addr":        config.HttpAddr,
		"shutdown_timeout": config.ShutdownTimeout,
		"version":          config.Version,
	}
	return settings
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Global configuration manager instance
var globalManager *Manager

// InitGlobal initializes the global configuration manager
func InitGlobal() *Manager {
	globalManager = NewManager()
	return globalManager
}

// Global returns the global configuration manager
func Global() *Manager {
	if globalManager == nil {
		globalManager = NewManager()
	}
	return globalManager
}

// LoadGlobal loads configuration using the global manager
func LoadGlobal() (*Config, error) {
	return Global().Load()
}

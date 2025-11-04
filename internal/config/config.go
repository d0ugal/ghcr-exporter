package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	promexporter_config "github.com/d0ugal/promexporter/config"
	"gopkg.in/yaml.v3"
)

// Duration uses promexporter Duration type
type Duration = promexporter_config.Duration

type Config struct {
	promexporter_config.BaseConfig

	GitHub   GitHubConfig   `yaml:"github"`
	Packages []PackageGroup `yaml:"packages"`
}

type GitHubConfig struct {
	Token promexporter_config.SensitiveString `yaml:"token"`
}

type PackageGroup struct {
	Owner string `yaml:"owner"`
	Repo  string `yaml:"repo,omitempty"` // Optional - if not provided, will discover all repos for owner
}

// GetName returns a unique name for this package group
func (p PackageGroup) GetName() string {
	if p.Repo == "" {
		return p.Owner + "-all"
	}

	return p.Owner + "-" + p.Repo
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string, configFromEnv bool) (*Config, error) {
	if configFromEnv {
		return loadFromEnv()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	setDefaults(&config)

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}

// loadFromEnv loads configuration from environment variables
func loadFromEnv() (*Config, error) {
	config := &Config{}

	// Load base configuration from environment
	baseConfig := &promexporter_config.BaseConfig{}

	// Server configuration
	if host := os.Getenv("GHCR_EXPORTER_SERVER_HOST"); host != "" {
		baseConfig.Server.Host = host
	} else {
		baseConfig.Server.Host = "0.0.0.0"
	}

	if portStr := os.Getenv("GHCR_EXPORTER_SERVER_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err != nil {
			return nil, fmt.Errorf("invalid server port: %w", err)
		} else {
			baseConfig.Server.Port = port
		}
	} else {
		baseConfig.Server.Port = 8080
	}

	// Logging configuration
	if level := os.Getenv("GHCR_EXPORTER_LOG_LEVEL"); level != "" {
		baseConfig.Logging.Level = level
	} else {
		baseConfig.Logging.Level = "info"
	}

	if format := os.Getenv("GHCR_EXPORTER_LOG_FORMAT"); format != "" {
		baseConfig.Logging.Format = format
	} else {
		baseConfig.Logging.Format = "json"
	}

	// Metrics configuration
	if intervalStr := os.Getenv("GHCR_EXPORTER_METRICS_COLLECTION_DEFAULT_INTERVAL"); intervalStr != "" {
		if interval, err := time.ParseDuration(intervalStr); err != nil {
			return nil, fmt.Errorf("invalid metrics default interval: %w", err)
		} else {
			baseConfig.Metrics.Collection.DefaultInterval = promexporter_config.Duration{Duration: interval}
			baseConfig.Metrics.Collection.DefaultIntervalSet = true
		}
	} else {
		baseConfig.Metrics.Collection.DefaultInterval = promexporter_config.Duration{Duration: time.Second * 30}
	}

	config.BaseConfig = *baseConfig

	// Apply generic environment variables (TRACING_ENABLED, PROFILING_ENABLED, etc.)
	// These are handled by promexporter and are shared across all exporters
	if err := promexporter_config.ApplyGenericEnvVars(&config.BaseConfig); err != nil {
		return nil, fmt.Errorf("failed to apply generic environment variables: %w", err)
	}

	// GitHub configuration
	if token := os.Getenv("GHCR_EXPORTER_GITHUB_TOKEN"); token != "" {
		config.GitHub.Token = promexporter_config.NewSensitiveString(token)
	}

	// Set defaults for any missing values
	setDefaults(config)

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// setDefaults sets default values for configuration
func setDefaults(config *Config) {
	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}

	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}

	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}

	if config.Logging.Format == "" {
		config.Logging.Format = "json"
	}

	if !config.Metrics.Collection.DefaultIntervalSet {
		config.Metrics.Collection.DefaultInterval = promexporter_config.Duration{Duration: time.Second * 30}
	}

	if config.GitHub.Token.IsEmpty() {
		config.GitHub.Token = promexporter_config.NewSensitiveString(os.Getenv("GITHUB_TOKEN"))
	}

	if len(config.Packages) == 0 {
		config.Packages = []PackageGroup{}
	}

	// Load packages from environment variables
	config.loadPackagesFromEnv()
}

// loadPackagesFromEnv loads package configuration from environment variables
func (c *Config) loadPackagesFromEnv() {
	// Look for package environment variables in the format GHCR_EXPORTER_PACKAGES_N_OWNER and GHCR_EXPORTER_PACKAGES_N_REPO
	for i := 0; i < 10; i++ { // Support up to 10 packages
		ownerKey := fmt.Sprintf("GHCR_EXPORTER_PACKAGES_%d_OWNER", i)
		repoKey := fmt.Sprintf("GHCR_EXPORTER_PACKAGES_%d_REPO", i)

		owner := os.Getenv(ownerKey)
		if owner == "" {
			continue // No more packages
		}

		repo := os.Getenv(repoKey)

		packageGroup := PackageGroup{
			Owner: owner,
			Repo:  repo,
		}

		c.Packages = append(c.Packages, packageGroup)

		fmt.Printf("Loaded package from env: owner=%s, repo=%s\n", owner, repo)
	}

	fmt.Printf("Total packages loaded: %d\n", len(c.Packages))
}

// Validate performs comprehensive validation of the configuration
func (c *Config) Validate() error {
	// Validate server configuration
	if err := c.validateServerConfig(); err != nil {
		return fmt.Errorf("server config: %w", err)
	}

	// Validate logging configuration
	if err := c.validateLoggingConfig(); err != nil {
		return fmt.Errorf("logging config: %w", err)
	}

	// Validate metrics configuration
	if err := c.validateMetricsConfig(); err != nil {
		return fmt.Errorf("metrics config: %w", err)
	}

	// Validate GitHub configuration
	if err := c.validateGitHubConfig(); err != nil {
		return fmt.Errorf("github config: %w", err)
	}

	return nil
}

func (c *Config) validateServerConfig() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", c.Server.Port)
	}

	return nil
}

func (c *Config) validateLoggingConfig() error {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("invalid logging level: %s", c.Logging.Level)
	}

	validFormats := map[string]bool{
		"json": true,
		"text": true,
	}
	if !validFormats[c.Logging.Format] {
		return fmt.Errorf("invalid logging format: %s", c.Logging.Format)
	}

	return nil
}

func (c *Config) validateMetricsConfig() error {
	if c.Metrics.Collection.DefaultInterval.Seconds() < 1 {
		return fmt.Errorf("default interval must be at least 1 second, got %d", c.Metrics.Collection.DefaultInterval.Seconds())
	}

	if c.Metrics.Collection.DefaultInterval.Seconds() > 86400 {
		return fmt.Errorf("default interval must be at most 86400 seconds (24 hours), got %d", c.Metrics.Collection.DefaultInterval.Seconds())
	}

	return nil
}

func (c *Config) validateGitHubConfig() error {
	if c.GitHub.Token.IsEmpty() {
		return fmt.Errorf("github token is required")
	}

	return nil
}

// GetPackageInterval returns the interval for a package group
func (c *Config) GetPackageInterval(group PackageGroup) int {
	if c.Metrics.Collection.DefaultIntervalSet {
		return c.Metrics.Collection.DefaultInterval.Seconds()
	}

	return 60 // Default to 60 seconds
}

// GetDisplayConfig returns configuration data safe for display
// Overrides BaseConfig to include GitHub configuration
func (c *Config) GetDisplayConfig() map[string]interface{} {
	// Get base configuration
	config := c.BaseConfig.GetDisplayConfig()

	// Add GitHub configuration (token will be redacted)
	config["GitHub Token"] = c.GitHub.Token

	return config
}

// GetDefaultInterval returns the default collection interval
func (c *Config) GetDefaultInterval() int {
	return c.Metrics.Collection.DefaultInterval.Seconds()
}

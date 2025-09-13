package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration represents a time duration that can be parsed from strings
type Duration struct {
	time.Duration
}

// UnmarshalYAML implements custom unmarshaling for duration strings
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var value interface{}
	if err := unmarshal(&value); err != nil {
		return err
	}

	switch v := value.(type) {
	case string:
		duration, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("invalid duration format '%s': %w", v, err)
		}

		d.Duration = duration
	case int:
		// Backward compatibility: treat as seconds
		d.Duration = time.Duration(v) * time.Second
	case int64:
		// Backward compatibility: treat as seconds
		d.Duration = time.Duration(v) * time.Second
	default:
		return fmt.Errorf("duration must be a string (e.g., '60s', '1h') or integer (seconds)")
	}

	return nil
}

// Seconds returns the duration in seconds
func (d *Duration) Seconds() int {
	return int(d.Duration.Seconds())
}

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Logging  LoggingConfig  `yaml:"logging"`
	Metrics  MetricsConfig  `yaml:"metrics"`
	GitHub   GitHubConfig   `yaml:"github"`
	Packages []PackageGroup `yaml:"packages"`
}

type GitHubConfig struct {
	Token string `yaml:"token"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"` // "json" or "text"
}

type MetricsConfig struct {
	Collection CollectionConfig `yaml:"collection"`
}

type CollectionConfig struct {
	DefaultInterval Duration `yaml:"default_interval"`
	// Track if the value was explicitly set
	DefaultIntervalSet bool `yaml:"-"`
}

// UnmarshalYAML implements custom unmarshaling to track if the value was set
func (c *CollectionConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Create a temporary struct to unmarshal into
	type tempCollectionConfig struct {
		DefaultInterval Duration `yaml:"default_interval"`
	}

	var temp tempCollectionConfig
	if err := unmarshal(&temp); err != nil {
		return err
	}

	c.DefaultInterval = temp.DefaultInterval
	c.DefaultIntervalSet = true

	return nil
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

// GetPackageInterval returns the interval for a package group
func (c *Config) GetPackageInterval(group PackageGroup) int {
	if c.Metrics.Collection.DefaultIntervalSet {
		return c.Metrics.Collection.DefaultInterval.Seconds()
	}

	return 60 // Default to 60 seconds
}

// LoadConfig loads configuration from a file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
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

	return &config, nil
}

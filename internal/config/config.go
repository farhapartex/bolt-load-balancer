package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type ServerConfig struct {
	Port         int           `yaml:"port"`
	Host         string        `yaml:"host"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

type BackendConfig struct {
	URL         string        `yaml:"url"`
	Weight      int           `yaml:"weight"`
	MaxFails    int           `yaml:"max_fails"`
	FailTimeout time.Duration `yaml:"fail_timeout"`
}

type HealthCheckConfig struct {
	Enabled        bool          `yaml:"enabled"`
	Interval       time.Duration `yaml:"interval"`
	Timeout        time.Duration `yaml:"timeout"`
	Path           string        `yaml:"path"`
	ExpectedStatus int           `yaml:"expected_status"`
}

type LoggingConfig struct {
	Level     string `yaml:"level"`
	Format    string `yaml:"format"`
	AccessLog bool   `yaml:"access_log"`
}

type Config struct {
	Server      ServerConfig      `yaml:"server"`
	Backends    []BackendConfig   `yaml:"backends"`
	Strategy    string            `yaml:"strategy"`
	HealthCheck HealthCheckConfig `yaml:"health_check"`
	Logging     LoggingConfig     `yaml:"logging"`
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         8080,
			Host:         "0.0.0.0",
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		Backends: []BackendConfig{
			{
				URL:         "http://localhost:8081",
				Weight:      1,
				MaxFails:    3,
				FailTimeout: 30 * time.Second,
			},
		},
		Strategy: "round_robin",
		HealthCheck: HealthCheckConfig{
			Enabled:        true,
			Interval:       30 * time.Second,
			Timeout:        5 * time.Second,
			Path:           "/health",
			ExpectedStatus: 200,
		},
		Logging: LoggingConfig{
			Level:     "info",
			Format:    "text",
			AccessLog: true,
		},
	}
}

func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535, got %d", c.Server.Port)
	}

	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}

	if len(c.Backends) == 0 {
		return fmt.Errorf("at least one backend must be configured")
	}

	for i, backend := range c.Backends {
		if backend.URL == "" {
			return fmt.Errorf("backend %d: URL cannot be empty", i)
		}

		if backend.Weight < 1 {
			c.Backends[i].Weight = 1
		}

		if backend.MaxFails < 1 {
			c.Backends[i].MaxFails = 3
		}

		if backend.FailTimeout <= 0 {
			c.Backends[i].FailTimeout = 30 * time.Second
		}
	}

	validStrategies := []string{"round_robin"}
	if c.Strategy == "" {
		c.Strategy = "round_robin"
	}

	isValidStrategy := false
	for _, strategy := range validStrategies {
		if c.Strategy == strategy {
			isValidStrategy = true
			break
		}
	}

	if !isValidStrategy {
		return fmt.Errorf("invalid load balancing strategy: %s. Supported strategies: %v",
			c.Strategy, validStrategies)
	}

	if c.HealthCheck.Interval <= 0 {
		c.HealthCheck.Interval = 30 * time.Second
	}

	if c.HealthCheck.Timeout <= 0 {
		c.HealthCheck.Timeout = 5 * time.Second
	}

	if c.HealthCheck.Path == "" {
		c.HealthCheck.Path = "/health"
	}

	if c.HealthCheck.ExpectedStatus == 0 {
		c.HealthCheck.ExpectedStatus = 200
	}

	validLogLevels := []string{"debug", "info", "warn", "error"}
	isValidLogLevel := false
	for _, level := range validLogLevels {
		if c.Logging.Level == level {
			isValidLogLevel = true
			break
		}
	}

	if !isValidLogLevel {
		c.Logging.Level = "info"
	}

	if c.Logging.Format != "json" && c.Logging.Format != "text" {
		c.Logging.Format = "text"
	}

	return nil
}

func (c *Config) SaveConfToFile(filename string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration to YAML: %w", err)
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write configuration file '%s': %w", filename, err)
	}

	return nil
}

func (c *Config) DataReprensation() string {
	// String returns a human-readable string representation of the configuration.
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Sprintf("Error marshaling config: %v", err)
	}
	return string(data)
}

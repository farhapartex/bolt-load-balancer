package config

import "time"

type ServerConfig struct {
	Port          int           `yaml:"port"`
	Host          string        `yaml:"host"`
	ReadTimeout   time.Duration `yaml:"read_timeout"`
	WriteTimeourk time.Duration `yaml:"write_timeout"`
	IdleTimeour   time.Duration `yaml:"idle_timeout"`
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

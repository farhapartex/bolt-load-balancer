package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

func LoadFromBytes(data []byte) (*Config, error) {
	config := DeafultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML configuration: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

func LoadFromFile(filename string) (*Config, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file '%s' does not exist", filename)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file '%s': %w", filename, err)
	}

	config, err := LoadFromBytes(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse configuration file '%s': %w", filename, err)
	}

	return config, nil
}

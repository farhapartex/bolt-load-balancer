package tests

import (
	"os"
	"testing"
	"time"

	"github.com/farhapartex/bolt-load-balancer/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()

	if cfg.Server.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", cfg.Server.Port)
	}

	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Expected default host '0.0.0.0', got %s", cfg.Server.Host)
	}

	if cfg.Server.ReadTimeout != 30*time.Second {
		t.Errorf("Expected read timeout 30s, got %v", cfg.Server.ReadTimeout)
	}

	if len(cfg.Backends) != 1 {
		t.Errorf("Expected 1 default backend, got %d", len(cfg.Backends))
	}

	if cfg.Strategy != "round_robin" {
		t.Errorf("Expected default strategy 'round_robin', got %s", cfg.Strategy)
	}

	if !cfg.HealthCheck.Enabled {
		t.Error("Expected health check to be enabled by default")
	}

	if cfg.HealthCheck.Interval != 30*time.Second {
		t.Errorf("Expected health check interval 30s, got %v", cfg.HealthCheck.Interval)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid config",
			config:      config.DefaultConfig(),
			expectError: false,
		},
		{
			name: "Invalid port - too low",
			config: &config.Config{
				Server: config.ServerConfig{Port: 0},
			},
			expectError: true,
			errorMsg:    "port must be between 1 and 65535",
		},
		{
			name: "Invalid port - too high",
			config: &config.Config{
				Server: config.ServerConfig{Port: 70000},
			},
			expectError: true,
			errorMsg:    "port must be between 1 and 65535",
		},
		{
			name: "No backends",
			config: &config.Config{
				Server:   config.ServerConfig{Port: 8080, Host: "0.0.0.0"},
				Backends: []config.BackendConfig{},
			},
			expectError: true,
			errorMsg:    "at least one backend must be configured",
		},
		{
			name: "Backend with empty URL",
			config: &config.Config{
				Server: config.ServerConfig{Port: 8080, Host: "0.0.0.0"},
				Backends: []config.BackendConfig{
					{URL: "", Weight: 1},
				},
			},
			expectError: true,
			errorMsg:    "URL cannot be empty",
		},
		{
			name: "Invalid strategy",
			config: &config.Config{
				Server: config.ServerConfig{Port: 8080, Host: "0.0.0.0"},
				Backends: []config.BackendConfig{
					{URL: "http://localhost:8081", Weight: 1},
				},
				Strategy: "invalid_strategy",
			},
			expectError: true,
			errorMsg:    "invalid load balancing strategy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorMsg)
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestLoadFromBytes(t *testing.T) {
	validYAML := `
server:
  port: 9090
  host: "localhost"
backends:
  - url: "http://test:8081"
    weight: 2
strategy: "round_robin"
health_check:
  enabled: true
logging:
  level: "debug"
`

	cfg, err := config.LoadFromBytes([]byte(validYAML))
	if err != nil {
		t.Fatalf("Failed to load valid YAML: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", cfg.Server.Port)
	}

	if cfg.Server.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got %s", cfg.Server.Host)
	}

	if len(cfg.Backends) != 1 {
		t.Errorf("Expected 1 backend, got %d", len(cfg.Backends))
	}

	if cfg.Backends[0].URL != "http://test:8081" {
		t.Errorf("Expected backend URL 'http://test:8081', got %s", cfg.Backends[0].URL)
	}

	if cfg.Backends[0].Weight != 2 {
		t.Errorf("Expected backend weight 2, got %d", cfg.Backends[0].Weight)
	}
}

func TestLoadFromBytesInvalidYAML(t *testing.T) {
	invalidYAML := `
server:
  port: "invalid_port"
  host: localhost
backends:
  - url: http://test:8081
`

	_, err := config.LoadFromBytes([]byte(invalidYAML))
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestConfigString(t *testing.T) {
	cfg := config.DefaultConfig()
	configStr := cfg.DataReprensation()

	if configStr == "" {
		t.Error("Config string should not be empty")
	}

	expectedParts := []string{"server:", "backends:", "strategy:", "health_check:"}
	for _, part := range expectedParts {
		if !contains(configStr, part) {
			t.Errorf("Config string should contain '%s'", part)
		}
	}
}

func TestSaveToFile(t *testing.T) {
	cfg := config.DefaultConfig()
	tmpFile := "test_config_output.yaml"

	defer os.Remove(tmpFile)

	err := cfg.SaveConfToFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to save config to file: %v", err)
	}

	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	loadedCfg, err := config.LoadFromFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loadedCfg.Server.Port != cfg.Server.Port {
		t.Errorf("Loaded config port mismatch: expected %d, got %d", cfg.Server.Port, loadedCfg.Server.Port)
	}

	if loadedCfg.Strategy != cfg.Strategy {
		t.Errorf("Loaded config strategy mismatch: expected %s, got %s", cfg.Strategy, loadedCfg.Strategy)
	}
}

func TestLoadFromFileNotFound(t *testing.T) {
	_, err := config.LoadFromFile("nonexistent_file.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}

	if !contains(err.Error(), "does not exist") {
		t.Errorf("Expected 'does not exist' error, got: %v", err)
	}
}

func contains(str, substr string) bool {
	return len(str) >= len(substr) &&
		(len(substr) == 0 || str[:len(str)-len(substr)+1] != str[:len(str)-len(substr)+1] ||
			str[len(str)-len(substr):] == substr ||
			findSubstring(str, substr))
}

func findSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

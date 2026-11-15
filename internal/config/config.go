package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ServerConfig holds HTTP server network configuration.
type ServerConfig struct {
	ProxyPort          int           `yaml:"proxy_port"`
	StudioPort         int           `yaml:"studio_port"`
	ReadTimeoutSeconds int           `yaml:"read_timeout_seconds"`
	WriteTimeoutSeconds int          `yaml:"write_timeout_seconds"`
}

// ReadTimeout returns the read timeout duration.
func (s ServerConfig) ReadTimeout() time.Duration {
	if s.ReadTimeoutSeconds <= 0 {
		return 30 * time.Second
	}
	return time.Duration(s.ReadTimeoutSeconds) * time.Second
}

// WriteTimeout returns the write timeout duration.
func (s ServerConfig) WriteTimeout() time.Duration {
	if s.WriteTimeoutSeconds <= 0 {
		return 60 * time.Second
	}
	return time.Duration(s.WriteTimeoutSeconds) * time.Second
}

// UpstreamConfig holds upstream LLM provider settings.
type UpstreamConfig struct {
	Provider     string           `yaml:"provider"`      // simulator, openai, anthropic, ollama, custom
	BaseURL      string           `yaml:"base_url"`      // e.g. https://api.openai.com/v1
	APIKey       string           `yaml:"api_key"`       // Bearer token or API key
	DefaultModel string           `yaml:"default_model"` // Fallback model name
	Fallbacks    []UpstreamConfig `yaml:"fallbacks,omitempty"`
}

// PolicyConfig defines runtime action rules for detected contradictions.
type PolicyConfig struct {
	DefaultAction             string  `yaml:"default_action"`              // rewrite, disclaimer, strict_halt, flag_only
	DisclaimerTemplate        string  `yaml:"disclaimer_template"`         // e.g. " [Correction: Official enterprise policy is {{.ExpectedValue}}, not {{.AssertedValue}}]"
	StrictHaltMessage         string  `yaml:"strict_halt_message"`         // Compliance abort message
	NumericalTolerancePercent float64 `yaml:"numerical_tolerance_percent"` // Numerical tolerance, e.g. 0.01 for 1%
}

// RegistryConfig holds settings for Ground Truth Knowledge Storage.
type RegistryConfig struct {
	DBPath           string `yaml:"db_path"`            // ":memory:" or file path
	InitialTruthFile string `yaml:"initial_truth_file"` // Optional path to JSON/YAML seed file
}

// TelemetryConfig controls audit logging and metrics retention.
type TelemetryConfig struct {
	AuditDBPath   string `yaml:"audit_db_path"`
	RetentionDays int    `yaml:"retention_days"`
}

// Config represents the complete application configuration tree.
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Upstream  UpstreamConfig  `yaml:"upstream"`
	Policy    PolicyConfig    `yaml:"policy"`
	Registry  RegistryConfig  `yaml:"registry"`
	Telemetry TelemetryConfig `yaml:"telemetry"`
}

// DefaultConfig returns safe enterprise default settings.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			ProxyPort:           8080,
			StudioPort:          9090,
			ReadTimeoutSeconds:  30,
			WriteTimeoutSeconds: 60,
		},
		Upstream: UpstreamConfig{
			Provider:     "simulator",
			BaseURL:      "https://api.openai.com/v1",
			APIKey:       "",
			DefaultModel: "gpt-4o",
		},
		Policy: PolicyConfig{
			DefaultAction:             "rewrite",
			DisclaimerTemplate:        " [Correction: Official enterprise policy is {{.ExpectedValue}}, not {{.AssertedValue}}]",
			StrictHaltMessage:         "Stream terminated by compliance filter: factual contradiction detected.",
			NumericalTolerancePercent: 0.0,
		},
		Registry: RegistryConfig{
			DBPath:           ":memory:",
			InitialTruthFile: "examples/ground_truth.json",
		},
		Telemetry: TelemetryConfig{
			AuditDBPath:   "claimguard_audit.db",
			RetentionDays: 90,
		},
	}
}

var envPattern = regexp.MustCompile(`\$\{([A-Z0-9_]+)\}`)

// expandEnvVars replaces ${VAR_NAME} with environment variable values.
func expandEnvVars(data []byte) []byte {
	return envPattern.ReplaceAllFunc(data, func(match []byte) []byte {
		key := strings.TrimSuffix(strings.TrimPrefix(string(match), "${"), "}")
		val, found := os.LookupEnv(key)
		if found {
			return []byte(val)
		}
		return match
	})
}

// Load reads and parses configuration from a YAML file.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	expanded := expandEnvVars(data)

	if err := yaml.Unmarshal(expanded, cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return cfg, nil
}

package config

import (
	"fmt"
	"os"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Server       ServerConfig       `toml:"server"`
	LoadBalancer LoadBalancerConfig `toml:"loadbalancer"`
	Logging      LoggingConfig      `toml:"logging"`
	RateLimit    RateLimitConfig    `toml:"ratelimit"`
	Metrics      MetricsConfig      `toml:"metrics"`
}

type ServerConfig struct {
	Port         int           `toml:"port"`
	Host         string        `toml:"host"`
	ReadTimeout  time.Duration `toml:"read_timeout"`
	WriteTimeout time.Duration `toml:"write_timeout"`
	TLS          TLSConfig     `toml:"tls"`
}

type TLSConfig struct {
	Enabled  bool   `toml:"enabled"`
	CertFile string `toml:"cert_file"`
	KeyFile  string `toml:"key_file"`
}

type LoadBalancerConfig struct {
	Algorithm           string          `toml:"algorithm"`
	SessionTTL          time.Duration   `toml:"session_ttl"`
	HealthCheckInterval time.Duration   `toml:"health_check_interval"`
	HealthCheckTimeout  time.Duration   `toml:"health_check_timeout"`
	Backends            []BackendConfig `toml:"backends"`
}

type BackendConfig struct {
	ID      string `toml:"id"`
	URL     string `toml:"url"`
	Weight  int    `toml:"weight"`
	Enabled bool   `toml:"enabled"`
}

type LoggingConfig struct {
	Level  string `toml:"level"`
	Format string `toml:"format"`
	Output string `toml:"output"`
}

type RateLimitConfig struct {
	Enabled           bool          `toml:"enabled"`
	RequestsPerSecond float64       `toml:"requests_per_second"`
	Burst             int           `toml:"burst"`
	CleanupInterval   time.Duration `toml:"cleanup_interval"`
}

type MetricsConfig struct {
	Enabled bool   `toml:"enabled"`
	Port    int    `toml:"port"`
	Path    string `toml:"path"`
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         8080,
			Host:         "0.0.0.0",
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			TLS: TLSConfig{
				Enabled: false,
			},
		},
		LoadBalancer: LoadBalancerConfig{
			Algorithm:           "round-robin",
			SessionTTL:          30 * time.Minute,
			HealthCheckInterval: 10 * time.Second,
			HealthCheckTimeout:  2 * time.Second,
			Backends:            []BackendConfig{},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		},
		RateLimit: RateLimitConfig{
			Enabled:           false,
			RequestsPerSecond: 100,
			Burst:             200,
			CleanupInterval:   1 * time.Minute,
		},
		Metrics: MetricsConfig{
			Enabled: true,
			Port:    9090,
			Path:    "/metrics",
		},
	}
}

func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := DefaultConfig()

	if err := toml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return config, nil
}

func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if len(c.LoadBalancer.Backends) == 0 {
		return fmt.Errorf("no backends configured")
	}

	for i, backend := range c.LoadBalancer.Backends {
		if backend.URL == "" {
			return fmt.Errorf("backend %d: URL is required", i)
		}
		if backend.Weight < 0 {
			return fmt.Errorf("backend %d: weight cannot be negative", i)
		}
		if backend.Weight == 0 {
			c.LoadBalancer.Backends[i].Weight = 1
		}
	}

	validAlgorithms := map[string]bool{
		"round-robin":          true,
		"weighted-round-robin": true,
		"least-connections":    false,
	}
	if !validAlgorithms[c.LoadBalancer.Algorithm] {
		return fmt.Errorf("unsupported algorithm: %s", c.LoadBalancer.Algorithm)
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", c.Logging.Level)
	}

	return nil
}

func (c *Config) SaveToFile(path string) error {
	data, err := toml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

package config

import (
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Proxy   ProxyConfig   `yaml:"proxy"`
	Storage StorageConfig `yaml:"storage"`
	Routes  []RouteConfig `yaml:"routes"`
}

type ServerConfig struct {
	Port     int    `yaml:"port"`
	Host     string `yaml:"host"`
	LogLevel string `yaml:"log_level"`
}

type ProxyConfig struct {
	Timeout       int  `yaml:"timeout"`
	MaxRetries    int  `yaml:"max_retries"`
	AutoRotate    bool `yaml:"auto_rotate"`
	StreamEnabled bool `yaml:"stream_enabled"`
}

type StorageConfig struct {
	DBPath     string `yaml:"db_path"`
	EncryptKey string `yaml:"encrypt_key"`
}

type RouteConfig struct {
	Pattern string `yaml:"pattern"`
	Target  string `yaml:"target"`
}

var (
	cfg  *Config
	once sync.Once
)

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:     8045,
			Host:     "0.0.0.0",
			LogLevel: "info",
		},
		Proxy: ProxyConfig{
			Timeout:       120,
			MaxRetries:    3,
			AutoRotate:    true,
			StreamEnabled: true,
		},
		Storage: StorageConfig{
			DBPath:     "./data/antigravity.db",
			EncryptKey: "",
		},
		Routes: []RouteConfig{
			{Pattern: "gpt-4*", Target: "gemini-3-pro-high"},
			{Pattern: "gpt-4o*", Target: "gemini-3-flash"},
			{Pattern: "gpt-3.5*", Target: "gemini-2.5-flash"},
			{Pattern: "o1-*", Target: "gemini-3-pro-high"},
			{Pattern: "o3-*", Target: "gemini-3-pro-high"},
			{Pattern: "claude-3-haiku-*", Target: "gemini-2.5-flash-lite"},
			{Pattern: "claude-haiku-*", Target: "gemini-2.5-flash-lite"},
			{Pattern: "claude-3-5-sonnet-*", Target: "claude-sonnet-4-5"},
			{Pattern: "claude-3-opus-*", Target: "claude-opus-4-5-thinking"},
			{Pattern: "claude-opus-4-*", Target: "claude-opus-4-5-thinking"},
		},
	}
}

// Load loads configuration from file
func Load(path string) (*Config, error) {
	var err error
	once.Do(func() {
		cfg = DefaultConfig()

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			if os.IsNotExist(readErr) {
				// Create default config file
				err = Save(path, cfg)
				return
			}
			err = readErr
			return
		}

		if unmarshalErr := yaml.Unmarshal(data, cfg); unmarshalErr != nil {
			err = unmarshalErr
			return
		}
	})

	return cfg, err
}

// Save saves configuration to file
func Save(path string, c *Config) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Get returns the current configuration
func Get() *Config {
	if cfg == nil {
		return DefaultConfig()
	}
	return cfg
}

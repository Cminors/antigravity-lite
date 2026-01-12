package config

import (
	"crypto/rand"
	"encoding/hex"
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
	Port        int    `yaml:"port" json:"port"`
	Host        string `yaml:"host" json:"host"`
	LogLevel    string `yaml:"log_level" json:"log_level"`
	APIKey      string `yaml:"api_key" json:"api_key"`
	AuthEnabled bool   `yaml:"auth_enabled" json:"auth_enabled"`
	LANAccess   bool   `yaml:"lan_access" json:"lan_access"`
	AutoStart   bool   `yaml:"autostart" json:"autostart"`
}

type ProxyConfig struct {
	Timeout       int    `yaml:"timeout" json:"timeout"`
	MaxRetries    int    `yaml:"max_retries" json:"max_retries"`
	AutoRotate    bool   `yaml:"auto_rotate" json:"auto_rotate"`
	StreamEnabled bool   `yaml:"stream_enabled" json:"stream_enabled"`
	ScheduleMode  string `yaml:"schedule_mode" json:"schedule_mode"`
	MaxWaitTime   int    `yaml:"max_wait_time" json:"max_wait_time"`
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
			Port:        8045,
			Host:        "127.0.0.1",
			LogLevel:    "info",
			APIKey:      GenerateAPIKey(),
			AuthEnabled: false,
			LANAccess:   false,
			AutoStart:   true,
		},
		Proxy: ProxyConfig{
			Timeout:       120,
			MaxRetries:    3,
			AutoRotate:    true,
			StreamEnabled: true,
			ScheduleMode:  "balance",
			MaxWaitTime:   60,
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

// GenerateAPIKey generates a random API key
func GenerateAPIKey() string {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "sk-default-key-please-regenerate"
	}
	return "sk-" + hex.EncodeToString(bytes)
}

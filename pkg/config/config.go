package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Model            string       `yaml:"model"`
	MaxTokens        int          `yaml:"max_tokens"`
	Temperature      float64      `yaml:"temperature"`
	ThinkingEnabled  bool         `yaml:"thinking_enabled"`
	ReasoningDisplay string       `yaml:"reasoning_display"`
	AutoCowork       bool         `yaml:"auto_cowork"`
	API              APIConfig    `yaml:"api"`
	APIKey           string       `yaml:"api_key,omitempty"`
	AllowedTools     []string     `yaml:"allowed_tools"`
	Memory           MemoryConfig `yaml:"memory"`
	Plans            PlansConfig  `yaml:"plans"`
	Web              WebConfig    `yaml:"web"`
	configPath       string       `yaml:"-"`
}

type APIConfig struct {
	BaseURL        string `yaml:"base_url"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
	MaxRetries     int    `yaml:"max_retries"`
}

type MemoryConfig struct {
	Enabled bool   `yaml:"enabled"`
	Dir     string `yaml:"dir"`
}

type PlansConfig struct {
	Dir string `yaml:"dir"`
}

type WebConfig struct {
	Enabled bool `yaml:"enabled"`
	Port    int  `yaml:"port"`
}

func Default() *Config {
	return &Config{
		Model:            "deepseek-v4-pro",
		MaxTokens:        32768,
		Temperature:      0.7,
		ThinkingEnabled:  false,
		ReasoningDisplay: "collapse",
		AutoCowork:       false,
		API: APIConfig{
			BaseURL:        "https://api.deepseek.com",
			TimeoutSeconds: 120,
			MaxRetries:     3,
		},
		Memory: MemoryConfig{
			Enabled: true,
			Dir:     "~/.deephermes/memory",
		},
		Plans: PlansConfig{Dir: "~/.deephermes/plans"},
		Web:   WebConfig{Enabled: false, Port: 8080},
	}
}

func Load(path string) (*Config, error) {
	cfg := Default()
	if path == "" {
		path = "config.yaml"
	}
	cfg.configPath = path
	if data, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}
	// User-level config overrides
	home, err := os.UserHomeDir()
	if err == nil {
		userPath := filepath.Join(home, ".deephermes", "config.yaml")
		cfg.configPath = userPath
		if data, err := os.ReadFile(userPath); err == nil {
			yaml.Unmarshal(data, cfg)
		}
	}
	cfg.applyEnvOverrides()
	cfg.expandPaths()
	return cfg, nil
}

// Save persists the current config to the user-level config file.
func (c *Config) Save() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	userDir := filepath.Join(home, ".deephermes")
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return err
	}
	userPath := filepath.Join(userDir, "config.yaml")
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(userPath, data, 0600)
}

// GetAPIKey returns the effective API key: config file > env var.
func (c *Config) GetAPIKey() string {
	if c.APIKey != "" {
		return c.APIKey
	}
	return os.Getenv("DEEPSEEK_API_KEY")
}

func (c *Config) applyEnvOverrides() {
	if v := os.Getenv("DEEPSEEK_MODEL"); v != "" {
		c.Model = v
	}
	if v := os.Getenv("DEEPSEEK_BASE_URL"); v != "" {
		c.API.BaseURL = v
	}
}

func (c *Config) expandPaths() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	c.Memory.Dir = strings.Replace(c.Memory.Dir, "~", home, 1)
	c.Memory.Dir = filepath.Clean(c.Memory.Dir)
	c.Plans.Dir = strings.Replace(c.Plans.Dir, "~", home, 1)
	c.Plans.Dir = filepath.Clean(c.Plans.Dir)
}

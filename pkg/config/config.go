package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Model            string       `yaml:"model"`
	Mode             string       `yaml:"mode"`
	Portable         bool         `yaml:"portable"`
	MinimizeToTray   bool         `yaml:"minimize_to_tray"`
	MaxTokens        int          `yaml:"max_tokens"`
	Temperature      float64      `yaml:"temperature"`
	ThinkingEnabled  bool         `yaml:"thinking_enabled"`
	ReasoningDisplay string       `yaml:"reasoning_display"`
	AutoCowork       bool         `yaml:"auto_cowork"`
	InitialPrompt    string       `yaml:"initial_prompt,omitempty"`
	RoleCard         string       `yaml:"role_card,omitempty"`
	WorldBook        string       `yaml:"world_book,omitempty"`
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
		Mode:             "code",
		Portable:         false,
		MinimizeToTray:   false,
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
	if portablePath, ok := portableConfigCandidate(); ok {
		cfg.configPath = portablePath
		if data, err := os.ReadFile(portablePath); err == nil {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, err
			}
			cfg.configPath = portablePath
			cfg.Portable = true
		} else if os.Getenv("DEEPHERMES_PORTABLE") == "1" {
			cfg.configPath = portablePath
			cfg.Portable = true
		}
	} else {
		// User-level config overrides
		home, err := os.UserHomeDir()
		if err == nil {
			userPath := filepath.Join(home, ".deephermes", "config.yaml")
			cfg.configPath = userPath
			if data, err := os.ReadFile(userPath); err == nil {
				yaml.Unmarshal(data, cfg)
			}
		}
	}
	cfg.applyEnvOverrides()
	cfg.expandPaths()
	return cfg, nil
}

// Save persists the current config to the user-level config file.
func (c *Config) Save() error {
	path, err := c.savePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if !c.Portable {
		if portablePath, ok := portableConfigCandidate(); ok && filepath.Clean(c.configPath) == filepath.Clean(portablePath) {
			_ = os.Remove(portablePath)
		}
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	c.configPath = path
	c.expandPaths()
	return os.WriteFile(path, data, 0600)
}

func (c *Config) ConfigPath() string {
	return c.configPath
}

func (c *Config) DataDir() string {
	if c.Portable {
		return portableDataDir()
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		wd, _ := os.Getwd()
		return filepath.Join(wd, ".deephermes")
	}
	return filepath.Join(home, ".deephermes")
}

func (c *Config) SessionsDir() string {
	return filepath.Join(c.DataDir(), "sessions")
}

func (c *Config) NormalizePaths() {
	c.expandPaths()
}

func (c *Config) savePath() (string, error) {
	if c.Portable {
		return filepath.Join(portableDataDir(), "config.yaml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".deephermes", "config.yaml"), nil
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
	base := c.DataDir()
	home := base
	if !c.Portable {
		if userHome, err := os.UserHomeDir(); err == nil && userHome != "" {
			home = userHome
		}
	}
	if c.Portable {
		c.Memory.Dir = filepath.Join(base, "memory")
		c.Plans.Dir = filepath.Join(base, "plans")
		return
	}
	c.Memory.Dir = strings.Replace(c.Memory.Dir, "~", home, 1)
	c.Memory.Dir = filepath.Clean(c.Memory.Dir)
	c.Plans.Dir = strings.Replace(c.Plans.Dir, "~", home, 1)
	c.Plans.Dir = filepath.Clean(c.Plans.Dir)
}

func portableConfigCandidate() (string, bool) {
	if os.Getenv("DEEPHERMES_PORTABLE") == "1" {
		return filepath.Join(portableDataDir(), "config.yaml"), true
	}
	for _, dir := range portableBaseDirs() {
		dataDir := filepath.Join(dir, "DeepHermesData")
		configPath := filepath.Join(dataDir, "config.yaml")
		if fileExists(configPath) || fileExists(filepath.Join(dir, "portable.flag")) {
			return configPath, true
		}
	}
	return "", false
}

func portableDataDir() string {
	if dir := strings.TrimSpace(os.Getenv("DEEPHERMES_PORTABLE_DIR")); dir != "" {
		return filepath.Clean(dir)
	}
	dirs := portableBaseDirs()
	if len(dirs) == 0 {
		wd, _ := os.Getwd()
		return filepath.Join(wd, "DeepHermesData")
	}
	return filepath.Join(dirs[0], "DeepHermesData")
}

func portableBaseDirs() []string {
	var dirs []string
	if exe, err := os.Executable(); err == nil && exe != "" {
		dirs = append(dirs, filepath.Dir(exe))
	}
	if wd, err := os.Getwd(); err == nil && wd != "" {
		seen := false
		cleanWd := filepath.Clean(wd)
		for _, dir := range dirs {
			if filepath.Clean(dir) == cleanWd {
				seen = true
				break
			}
		}
		if !seen {
			dirs = append(dirs, wd)
		}
	}
	return dirs
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

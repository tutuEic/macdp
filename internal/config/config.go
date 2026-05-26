package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server  ServerCfg  `yaml:"server"`
	Agents  []AgentCfg `yaml:"agents"`
	LLM     LLMCfg     `yaml:"llm"`
	Git     GitCfg     `yaml:"git"`
	Storage StorageCfg `yaml:"storage"`
}

type ServerCfg struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type AgentCfg struct {
	Name     string            `yaml:"name"`
	Type     string            `yaml:"type"`
	Enabled  bool              `yaml:"enabled"`
	Endpoint string            `yaml:"endpoint,omitempty"`
	Config   map[string]string `yaml:"config,omitempty"`
}

type LLMCfg struct {
	BaseURL   string `yaml:"base_url"`
	Model     string `yaml:"model"`
	APIKey    string `yaml:"api_key"`
	APIKeyEnv string `yaml:"api_key_env"`
}

func (c *LLMCfg) GetAPIKey() string {
	if c.APIKey != "" {
		return c.APIKey
	}
	if c.APIKeyEnv != "" {
		return os.Getenv(c.APIKeyEnv)
	}
	return ""
}

type GitCfg struct {
	WorktreeDir  string `yaml:"worktree_dir"`
	BranchPrefix string `yaml:"branch_prefix"`
	MergeStrategy string `yaml:"merge_strategy"`
}

type StorageCfg struct {
	Path string `yaml:"path"`
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerCfg{Host: "0.0.0.0", Port: 8080},
		Agents: []AgentCfg{
			{Name: "hermes", Type: "hermes", Enabled: true},
			{Name: "openclaw", Type: "openclaw", Enabled: true, Endpoint: "http://localhost:18789"},
			{Name: "codex", Type: "codex", Enabled: true},
			{Name: "claude-code", Type: "claude-code", Enabled: true},
		},
		LLM: LLMCfg{
			BaseURL: "https://api.deepseek.com/v1",
			Model:   "deepseek-chat",
		},
		Git: GitCfg{
			WorktreeDir:   ".macdp/worktrees",
			BranchPrefix:  "macdp/",
			MergeStrategy: "squash",
		},
		Storage: StorageCfg{Path: "macdp.db"},
	}
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultConfig(), nil
	}
	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

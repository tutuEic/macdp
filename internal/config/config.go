package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the top-level MACDP configuration.
type Config struct {
	Orchestrator string        `yaml:"orchestrator"`
	Agents       []AgentConfig `yaml:"agents"`
	Scheduler    SchedulerCfg  `yaml:"scheduler"`
	Git          GitCfg        `yaml:"git"`
	Review       ReviewCfg     `yaml:"review"`
	Budget       BudgetCfg     `yaml:"budget"`
	API          APICfg        `yaml:"api"`
}

type AgentConfig struct {
	Name          string   `yaml:"name"`
	Enabled       bool     `yaml:"enabled"`
	Role          string   `yaml:"role"` // orchestrator or worker
	Entrypoint    string   `yaml:"entrypoint"`
	Flags         []string `yaml:"flags"`
	MaxConcurrent int      `yaml:"max_concurrent"`
	Strengths     []string `yaml:"strengths"`
}

type SchedulerCfg struct {
	MaxParallel int `yaml:"max_parallel"`
	TaskTimeout int `yaml:"task_timeout_seconds"`
	MaxRetries  int `yaml:"max_retries"`
}

type GitCfg struct {
	WorktreeDir  string `yaml:"worktree_dir"`
	BranchPrefix string `yaml:"branch_prefix"`
	MergeStrategy string `yaml:"merge_strategy"`
}

type ReviewCfg struct {
	Enabled          bool `yaml:"enabled"`
	CrossReview      bool `yaml:"cross_review"`
	MaxReviewRounds  int  `yaml:"max_review_rounds"`
	ApprovalRequired bool `yaml:"approval_required"`
}

type BudgetCfg struct {
	MaxPerTaskUSD    float64 `yaml:"max_per_task_usd"`
	MaxPerProjectUSD float64 `yaml:"max_per_project_usd"`
}

type APICfg struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// DefaultConfig returns a sensible default configuration.
func DefaultConfig() *Config {
	return &Config{
		Orchestrator: "hermes",
		Agents: []AgentConfig{
			{Name: "hermes", Enabled: true, Role: "orchestrator", Entrypoint: "hermes", MaxConcurrent: 3},
			{Name: "claude-code", Enabled: true, Role: "worker", Entrypoint: "claude", Flags: []string{"--output-format", "json"}, MaxConcurrent: 3, Strengths: []string{"complex_code", "refactoring", "review"}},
			{Name: "codex", Enabled: true, Role: "worker", Entrypoint: "codex", Flags: []string{"--full-auto"}, MaxConcurrent: 5, Strengths: []string{"prototyping", "single_file", "scripts"}},
		},
		Scheduler: SchedulerCfg{MaxParallel: 5, TaskTimeout: 600, MaxRetries: 2},
		Git:       GitCfg{WorktreeDir: ".macdp/worktrees", BranchPrefix: "macdp/", MergeStrategy: "squash"},
		Review:    ReviewCfg{Enabled: true, CrossReview: true, MaxReviewRounds: 3},
		Budget:    BudgetCfg{MaxPerTaskUSD: 5.0, MaxPerProjectUSD: 50.0},
		API:       APICfg{Host: "0.0.0.0", Port: 8080},
	}
}

// Load reads config from a YAML file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save writes config to a YAML file.
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

package scoreconfig

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ScoreConfig holds score command configuration from .promptctl/score.yaml.
type ScoreConfig struct {
	Dirs     []string `yaml:"dirs"`
	Include  []string `yaml:"include"`
	Ignore   []string `yaml:"ignore"`
	MinScore int      `yaml:"min_score"`
	Rules    []string `yaml:"rules"`
}

// DefaultConfig returns the default score config when no file is present.
func DefaultConfig() ScoreConfig {
	return ScoreConfig{
		Include:  []string{"*.txt", "*.md"},
		MinScore: 80,
	}
}

// Load reads .promptctl/score.yaml from the directory that contains .promptctl.
// When the file is missing, returns default config (Include: ["*.txt","*.md"], MinScore: 80).
func Load(dir string) ScoreConfig {
	path := filepath.Join(dir, ".promptctl", "score.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultConfig()
	}
	var cfg ScoreConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig()
	}
	return cfg
}

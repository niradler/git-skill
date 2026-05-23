package runtimes

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/niradler/git-skill/internal/kind"
)

// ConfigFilename is the leaf filename for the user/project runtimes
// config. The user copy lives at $HOME/.config/git-skill/runtimes.yaml
// (or under $GIT_SKILL_USER_CONFIG when set); the project copy lives at
// <repoRoot>/.git-skill/runtimes.yaml.
const ConfigFilename = "runtimes.yaml"

// Config is the on-disk schema for runtimes.yaml. It mirrors the
// built-in registry layout: each entry under runtimes is a runtime
// name, and each entry under that is a kind ("skill" or "agent")
// with a Mapping override.
type Config struct {
	Runtimes map[string]map[string]Mapping `yaml:"runtimes"`
}

// LoadConfig reads a runtimes.yaml file. Returns (nil, nil) when the
// file is absent so callers can stack optional layers without branching
// on os.IsNotExist.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	c := &Config{}
	if err := yaml.Unmarshal(data, c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	for rt, kinds := range c.Runtimes {
		if rt == "" {
			return nil, fmt.Errorf("%s: runtime name must not be empty", path)
		}
		for ks, m := range kinds {
			if _, err := kind.Parse(ks); err != nil {
				return nil, fmt.Errorf("%s: runtime %q: %w", path, rt, err)
			}
			if m.To == "" {
				return nil, fmt.Errorf("%s: runtime %q kind %q: 'to' must be set", path, rt, ks)
			}
		}
	}
	return c, nil
}

// userConfigPath returns the resolved user-config path, honoring the
// GIT_SKILL_USER_CONFIG override (intended for tests and exotic setups).
// When $HOME is unavailable, returns "" so the caller can skip the user
// layer cleanly.
func userConfigPath() string {
	if override := os.Getenv("GIT_SKILL_USER_CONFIG"); override != "" {
		return override
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".config", "git-skill", ConfigFilename)
}

// projectConfigPath returns <repoRoot>/.git-skill/runtimes.yaml. The
// caller is responsible for passing a valid repo root; if repoRoot is
// empty, the function returns "" so the caller can skip the layer.
func projectConfigPath(repoRoot string) string {
	if repoRoot == "" {
		return ""
	}
	return filepath.Join(repoRoot, ".git-skill", ConfigFilename)
}

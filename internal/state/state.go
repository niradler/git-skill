package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/niradler/git-skill/internal/kind"
	"github.com/niradler/git-skill/internal/semver"
)

const Filename = "assets.json"
const SupportedVersion = 1

type Config struct {
	SkillsRoot string `json:"skillsRoot,omitempty"`
	AgentsRoot string `json:"agentsRoot,omitempty"`
}

type Entry struct {
	Spec      string   `json:"spec"`
	Remote    string   `json:"remote"`
	Runtimes  []string `json:"runtimes,omitempty"`
	Requires  []string `json:"requires,omitempty"`
	Dev       bool     `json:"dev,omitempty"`
	Version   string   `json:"version,omitempty"`
	Commit    string   `json:"commit"`
	Canonical string   `json:"canonical"`
}

type State struct {
	Version int
	Config  Config
	Assets  map[kind.Kind]map[string]Entry
}

type jsonShape struct {
	Version int                         `json:"version"`
	Config  Config                      `json:"config,omitempty"`
	Assets  map[string]map[string]Entry `json:"assets"`
}

func New() *State {
	return &State{
		Version: SupportedVersion,
		Config:  Config{SkillsRoot: "skills", AgentsRoot: "agents"},
		Assets:  map[kind.Kind]map[string]Entry{},
	}
}

func Read(repoRoot string) (*State, error) {
	data, err := os.ReadFile(filepath.Join(repoRoot, Filename))
	if os.IsNotExist(err) {
		return New(), nil
	}
	if err != nil {
		return nil, err
	}
	var s jsonShape
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse %s: %w", Filename, err)
	}
	if s.Version != SupportedVersion {
		return nil, fmt.Errorf("unsupported version %d in %s: only version %d is supported",
			s.Version, Filename, SupportedVersion)
	}
	st := New()
	st.Version = s.Version
	if s.Config.SkillsRoot != "" {
		st.Config.SkillsRoot = s.Config.SkillsRoot
	}
	if s.Config.AgentsRoot != "" {
		st.Config.AgentsRoot = s.Config.AgentsRoot
	}
	for ks, em := range s.Assets {
		k, err := kind.Parse(ks)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", Filename, err)
		}
		for name, e := range em {
			if _, err := semver.ParseSpec(e.Spec); err != nil {
				return nil, fmt.Errorf("%s: %s/%s: %w", Filename, ks, name, err)
			}
			if err := validateRequires(name, e.Requires); err != nil {
				return nil, fmt.Errorf("%s: %w", Filename, err)
			}
			if st.Assets[k] == nil {
				st.Assets[k] = map[string]Entry{}
			}
			st.Assets[k][name] = e
		}
	}
	return st, nil
}

func validateRequires(owner string, requires []string) error {
	for _, r := range requires {
		idx := strings.Index(r, "/")
		if idx <= 0 {
			return fmt.Errorf("entry %q: requires entry %q missing '<kind>/' prefix", owner, r)
		}
		if _, err := kind.Parse(r[:idx]); err != nil {
			return fmt.Errorf("entry %q: requires entry %q: %w", owner, r, err)
		}
	}
	return nil
}

func (s *State) Write(repoRoot string) error {
	s.Version = SupportedVersion
	out := jsonShape{Version: s.Version, Config: s.Config, Assets: map[string]map[string]Entry{}}
	for k, em := range s.Assets {
		out.Assets[k.String()] = em
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(repoRoot, Filename), append(data, '\n'), 0644)
}

func (s *State) Get(k kind.Kind, name string) (Entry, bool) {
	if s.Assets[k] == nil {
		return Entry{}, false
	}
	e, ok := s.Assets[k][name]
	return e, ok
}

func (s *State) Set(k kind.Kind, name string, e Entry) {
	if s.Assets == nil {
		s.Assets = map[kind.Kind]map[string]Entry{}
	}
	if s.Assets[k] == nil {
		s.Assets[k] = map[string]Entry{}
	}
	s.Assets[k][name] = e
}

func (s *State) Delete(k kind.Kind, name string) {
	if s.Assets[k] == nil {
		return
	}
	delete(s.Assets[k], name)
}

func (s *State) Root(k kind.Kind) string {
	switch k {
	case kind.Skill:
		return s.Config.SkillsRoot
	case kind.Agent:
		return s.Config.AgentsRoot
	}
	return ""
}

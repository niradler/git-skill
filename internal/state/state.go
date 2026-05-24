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

// RuntimeOverride is the consumer-side override for a single runtime in
// a lock entry. Empty fields mean "use the registry / manifest default".
type RuntimeOverride struct {
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
}

type Entry struct {
	Spec      string                     `json:"spec"`
	Remote    string                     `json:"remote"`
	Runtimes  map[string]RuntimeOverride `json:"runtimes,omitempty"`
	Requires  []string                   `json:"requires,omitempty"`
	Dev       bool                       `json:"dev,omitempty"`
	Version   string                     `json:"version,omitempty"`
	Commit    string                     `json:"commit"`
	Canonical string                     `json:"canonical"`
}

type State struct {
	Version int
	Config  Config
	Assets  map[kind.Kind]map[string]Entry
}

// entryShape mirrors Entry but keeps runtimes as raw JSON so we can detect
// and reject the old []string form with a precise error.
type entryShape struct {
	Spec      string          `json:"spec"`
	Remote    string          `json:"remote"`
	Runtimes  json.RawMessage `json:"runtimes,omitempty"`
	Requires  []string        `json:"requires,omitempty"`
	Dev       bool            `json:"dev,omitempty"`
	Version   string          `json:"version,omitempty"`
	Commit    string          `json:"commit"`
	Canonical string          `json:"canonical"`
}

type jsonShape struct {
	Version int                              `json:"version"`
	Config  Config                           `json:"config,omitempty"`
	Assets  map[string]map[string]entryShape `json:"assets"`
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
		for name, es := range em {
			if _, err := semver.ParseSpec(es.Spec); err != nil {
				return nil, fmt.Errorf("%s: %s/%s: %w", Filename, ks, name, err)
			}
			if err := validateRequires(name, es.Requires); err != nil {
				return nil, fmt.Errorf("%s: %w", Filename, err)
			}
			runtimes, err := parseRuntimes(es.Runtimes, ks, name)
			if err != nil {
				return nil, err
			}
			if st.Assets[k] == nil {
				st.Assets[k] = map[string]Entry{}
			}
			st.Assets[k][name] = Entry{
				Spec:      es.Spec,
				Remote:    es.Remote,
				Runtimes:  runtimes,
				Requires:  es.Requires,
				Dev:       es.Dev,
				Version:   es.Version,
				Commit:    es.Commit,
				Canonical: es.Canonical,
			}
		}
	}
	return st, nil
}

// parseRuntimes accepts only the object form {name: {from?, to?}}. The
// previous []string form is rejected with an explicit error.
func parseRuntimes(raw json.RawMessage, ks, name string) (map[string]RuntimeOverride, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	trimmed := strings.TrimSpace(string(raw))
	if strings.HasPrefix(trimmed, "[") {
		return nil, fmt.Errorf("%s: %s/%s: 'runtimes' is now an object {name: {from?, to?}}; the legacy []string form is no longer supported", Filename, ks, name)
	}
	out := map[string]RuntimeOverride{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("%s: %s/%s: parse runtimes: %w", Filename, ks, name, err)
	}
	if _, hasEmpty := out[""]; hasEmpty {
		return nil, fmt.Errorf("%s: %s/%s: runtime name must not be empty", Filename, ks, name)
	}
	return out, nil
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

// writeShape mirrors Entry for marshaling. Runtimes is the object form.
type writeShape struct {
	Spec      string                     `json:"spec"`
	Remote    string                     `json:"remote"`
	Runtimes  map[string]RuntimeOverride `json:"runtimes,omitempty"`
	Requires  []string                   `json:"requires,omitempty"`
	Dev       bool                       `json:"dev,omitempty"`
	Version   string                     `json:"version,omitempty"`
	Commit    string                     `json:"commit"`
	Canonical string                     `json:"canonical"`
}

type writeJSON struct {
	Version int                              `json:"version"`
	Config  Config                           `json:"config,omitempty"`
	Assets  map[string]map[string]writeShape `json:"assets"`
}

func (s *State) Write(repoRoot string) error {
	s.Version = SupportedVersion
	out := writeJSON{Version: s.Version, Config: s.Config, Assets: map[string]map[string]writeShape{}}
	for k, em := range s.Assets {
		ws := map[string]writeShape{}
		for name, e := range em {
			ws[name] = writeShape{
				Spec:      e.Spec,
				Remote:    e.Remote,
				Runtimes:  e.Runtimes,
				Requires:  e.Requires,
				Dev:       e.Dev,
				Version:   e.Version,
				Commit:    e.Commit,
				Canonical: strings.ReplaceAll(e.Canonical, `\`, "/"),
			}
		}
		out.Assets[k.String()] = ws
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

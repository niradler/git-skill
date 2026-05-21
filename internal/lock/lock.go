package lock

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const Filename = "skill.lock"

type Entry struct {
	Remote    string            `json:"remote"`
	Version   string            `json:"version,omitempty"`
	Commit    string            `json:"commit"`
	Canonical string            `json:"canonical"`
	Agents    map[string]string `json:"agents,omitempty"`
	Dev       bool              `json:"dev,omitempty"`
}

type Lock struct {
	Version int              `json:"lockfileVersion"`
	Skills  map[string]Entry `json:"skills"`
}

func Read(repoRoot string) (*Lock, error) {
	data, err := os.ReadFile(filepath.Join(repoRoot, Filename))
	if os.IsNotExist(err) {
		return &Lock{Version: 2, Skills: make(map[string]Entry)}, nil
	}
	if err != nil {
		return nil, err
	}

	// Decode into a raw map first so we can inspect both v1 and v2 fields.
	var raw struct {
		Version int `json:"lockfileVersion"`
		Skills  map[string]json.RawMessage `json:"skills"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	if raw.Skills == nil {
		raw.Skills = make(map[string]json.RawMessage)
	}

	l := &Lock{Version: 2, Skills: make(map[string]Entry)}
	for key, raw := range raw.Skills {
		var e Entry
		if err := json.Unmarshal(raw, &e); err != nil {
			return nil, err
		}
		// v1 migration: Dest → Canonical
		if e.Canonical == "" {
			var v1 struct {
				Dest string `json:"dest"`
			}
			_ = json.Unmarshal(raw, &v1)
			e.Canonical = v1.Dest
		}
		l.Skills[key] = e
	}
	return l, nil
}

func (l *Lock) Write(repoRoot string) error {
	l.Version = 2
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(repoRoot, Filename), append(data, '\n'), 0644)
}

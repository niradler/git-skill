package commands

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/niradler/git-skill/internal/git"
	"github.com/niradler/git-skill/internal/kind"
	"github.com/niradler/git-skill/internal/state"
)

func TestAddNewSkill(t *testing.T) {
	upstream, _ := makeUpstreamSkill(t, "acme/x")
	wd, _ := os.Getwd()
	os.Chdir(upstream)
	tip, _ := git.ResolveRef("refs/assets/skill/acme/x")
	git.UpdateRef("refs/asset-tags/skill/acme/x/v1.0.0", tip)
	os.Chdir(wd)

	consumer := t.TempDir()
	os.Chdir(consumer)
	defer os.Chdir(wd)
	exec.Command("git", "init", "-q").Run()
	exec.Command("git", "config", "core.autocrlf", "false").Run()
	Init(profileSkillOnly, nil, &bytes.Buffer{}, &bytes.Buffer{})

	var stdout, stderr bytes.Buffer
	err := Add(profileSkillOnly,
		[]string{"acme/x@v1.0.0", "--from", upstream, "--runtime", "claude"},
		&stdout, &stderr)
	if err != nil {
		t.Fatalf("Add: %v stderr=%s", err, stderr.String())
	}

	st, _ := state.Read(consumer)
	e, ok := st.Get(kind.Skill, "acme/x")
	if !ok {
		t.Fatalf("state entry missing")
	}
	if e.Spec == "" {
		t.Errorf("Spec not stored: %+v", e)
	}
	if e.Commit == "" {
		t.Errorf("Commit not resolved: %+v", e)
	}
	if e.Version != "v1.0.0" {
		t.Errorf("Version = %q, want v1.0.0", e.Version)
	}
	if _, err := os.Stat("skills/acme/x/SKILL.md"); err != nil {
		t.Errorf("canonical materialization missing: %v", err)
	}
}

// A7: --target <runtime>=<path> persists into the lock entry and drives
// materialization to the overridden path.
func TestAdd_TargetOverridePersistsAndMaterializes(t *testing.T) {
	upstream, _ := makeUpstreamSkill(t, "acme/x")
	wd, _ := os.Getwd()
	os.Chdir(upstream)
	tip, _ := git.ResolveRef("refs/assets/skill/acme/x")
	git.UpdateRef("refs/asset-tags/skill/acme/x/v1.0.0", tip)
	os.Chdir(wd)

	consumer := t.TempDir()
	os.Chdir(consumer)
	defer os.Chdir(wd)
	exec.Command("git", "init", "-q").Run()
	exec.Command("git", "config", "core.autocrlf", "false").Run()
	Init(profileSkillOnly, nil, &bytes.Buffer{}, &bytes.Buffer{})

	var stdout, stderr bytes.Buffer
	err := Add(profileSkillOnly,
		[]string{
			"acme/x@v1.0.0",
			"--from", upstream,
			"--runtime", "claude",
			"--target", "claude=.custom/claude/acme/x/",
		},
		&stdout, &stderr)
	if err != nil {
		t.Fatalf("Add: %v stderr=%s", err, stderr.String())
	}

	st, _ := state.Read(consumer)
	e, ok := st.Get(kind.Skill, "acme/x")
	if !ok {
		t.Fatalf("state entry missing")
	}
	override, ok := e.Runtimes["claude"]
	if !ok {
		t.Fatalf("claude runtime missing from lock: %+v", e.Runtimes)
	}
	if override.To != ".custom/claude/acme/x/" {
		t.Errorf("override.To = %q, want %q", override.To, ".custom/claude/acme/x/")
	}
	if _, err := os.Stat(".custom/claude/acme/x/SKILL.md"); err != nil {
		t.Errorf("override path not materialized: %v", err)
	}
}

func TestAdd_CanonicalIsPosixSlashes(t *testing.T) {
	upstream, _ := makeUpstreamSkill(t, "acme/x")
	wd, _ := os.Getwd()
	os.Chdir(upstream)
	tip, _ := git.ResolveRef("refs/assets/skill/acme/x")
	git.UpdateRef("refs/asset-tags/skill/acme/x/v1.0.0", tip)
	os.Chdir(wd)

	consumer := t.TempDir()
	os.Chdir(consumer)
	defer os.Chdir(wd)
	exec.Command("git", "init", "-q").Run()
	exec.Command("git", "config", "core.autocrlf", "false").Run()
	Init(profileSkillOnly, nil, &bytes.Buffer{}, &bytes.Buffer{})

	var stdout, stderr bytes.Buffer
	if err := Add(profileSkillOnly,
		[]string{"acme/x@v1.0.0", "--from", upstream, "--runtime", "claude"},
		&stdout, &stderr); err != nil {
		t.Fatalf("Add: %v stderr=%s", err, stderr.String())
	}

	st, _ := state.Read(consumer)
	e, _ := st.Get(kind.Skill, "acme/x")
	if e.Canonical != "skills/acme/x" {
		t.Errorf("Canonical = %q, want %q (posix slashes regardless of host OS)",
			e.Canonical, "skills/acme/x")
	}

	raw, err := os.ReadFile(filepath.Join(consumer, "assets.json"))
	if err != nil {
		t.Fatalf("read assets.json: %v", err)
	}
	if !strings.Contains(string(raw), `"canonical": "skills/acme/x"`) {
		t.Errorf("assets.json canonical not in expected posix form.\n--- content ---\n%s", raw)
	}
}

func TestState_WriteNormalizesBackslashCanonical(t *testing.T) {
	repo := t.TempDir()
	st := &state.State{
		Config: state.Config{SkillsRoot: "skills", AgentsRoot: "agents"},
	}
	st.Set(kind.Skill, "acme/x", state.Entry{
		Spec:      "v1.0.0",
		Remote:    "https://example/r",
		Commit:    "abc123",
		Version:   "v1.0.0",
		Canonical: `skills\acme\x`,
	})
	if err := st.Write(repo); err != nil {
		t.Fatalf("Write: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(repo, "assets.json"))
	if strings.Contains(string(raw), `\\`) {
		t.Errorf("Write did not normalize backslashes:\n%s", raw)
	}

	reread, _ := state.Read(repo)
	e, _ := reread.Get(kind.Skill, "acme/x")
	if e.Canonical != "skills/acme/x" {
		t.Errorf("after self-heal, Canonical = %q, want %q", e.Canonical, "skills/acme/x")
	}
}

func TestBuildRuntimes_FlagAndTargets(t *testing.T) {
	out, err := buildRuntimes("claude,codex", []string{"codex=.custom/codex/<name>/"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("len = %d, want 2: %+v", len(out), out)
	}
	if _, ok := out["claude"]; !ok {
		t.Errorf("claude missing")
	}
	if out["codex"].To != ".custom/codex/<name>/" {
		t.Errorf("codex.To = %q", out["codex"].To)
	}
}

func TestBuildRuntimes_TargetWithoutRuntimeIsError(t *testing.T) {
	_, err := buildRuntimes("claude", []string{"codex=.custom/<name>/"})
	if err == nil {
		t.Fatal("expected error: --target for runtime not in --runtime")
	}
}

func TestBuildRuntimes_MalformedTargetIsError(t *testing.T) {
	for _, bad := range []string{"claude", "claude=", "=foo"} {
		_, err := buildRuntimes("claude", []string{bad})
		if err == nil {
			t.Errorf("expected error for %q", bad)
		}
	}
}

func TestSplitFlagsAndPositional_BoolFlagDoesNotConsumePositional(t *testing.T) {
	boolFlags := map[string]bool{"dev": true}
	args := []string{"--dev", "acme/x", "--from", "https://example/r", "--runtime", "claude"}
	flags, positional := splitFlagsAndPositional(args, boolFlags)

	wantFlags := []string{"--dev", "--from", "https://example/r", "--runtime", "claude"}
	wantPositional := []string{"acme/x"}

	if !equalStrings(flags, wantFlags) {
		t.Errorf("flags = %v, want %v", flags, wantFlags)
	}
	if !equalStrings(positional, wantPositional) {
		t.Errorf("positional = %v, want %v", positional, wantPositional)
	}
}

func TestSplitFlagsAndPositional_ValueFlagStillConsumesValue(t *testing.T) {
	boolFlags := map[string]bool{"dev": true}
	args := []string{"acme/x", "--from", "https://example/r"}
	flags, positional := splitFlagsAndPositional(args, boolFlags)

	if !equalStrings(flags, []string{"--from", "https://example/r"}) {
		t.Errorf("flags = %v", flags)
	}
	if !equalStrings(positional, []string{"acme/x"}) {
		t.Errorf("positional = %v", positional)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

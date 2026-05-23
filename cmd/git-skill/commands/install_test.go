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
	"github.com/niradler/git-skill/internal/manifest"
	"github.com/niradler/git-skill/internal/state"
)

func makeUpstreamSkill(t *testing.T, name string) (string, string) {
	return makeUpstreamSkillFiles(t, name, map[string]string{
		"SKILL.md":  "---\nname: " + name + "\n---\n# " + name,
		"extra.txt": "payload",
	})
}

// makeUpstreamSkillFiles seeds an upstream skill repo with arbitrary files
// at the root of the asset tree. Used by tests that need to ship extras
// like git-skill.yaml or runtime-specific source files.
func makeUpstreamSkillFiles(t *testing.T, name string, files map[string]string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "t@t"},
		{"config", "user.name", "t"},
		{"config", "core.autocrlf", "false"},
	} {
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("%s", out)
		}
	}
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)
	os.MkdirAll("src", 0755)
	for path, body := range files {
		full := filepath.Join("src", path)
		if d := filepath.Dir(full); d != "." {
			os.MkdirAll(d, 0755)
		}
		os.WriteFile(full, []byte(body), 0644)
	}
	tree, _ := git.WriteTreeFromDir(filepath.Join(dir, "src"))
	commit, _ := git.CommitTree(tree, "init")
	_ = git.UpdateRef("refs/assets/skill/"+name, commit)
	return dir, commit
}

func TestInstallFromState(t *testing.T) {
	upstream, commit := makeUpstreamSkill(t, "acme/x")

	consumer := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(consumer)
	defer os.Chdir(wd)
	exec.Command("git", "init", "-q").Run()
	exec.Command("git", "config", "core.autocrlf", "false").Run()

	st := state.New()
	st.Set(kind.Skill, "acme/x", state.Entry{
		Spec:      "v1.0.0",
		Remote:    upstream,
		Runtimes:  map[string]state.RuntimeOverride{"claude": {}},
		Version:   "v1.0.0",
		Commit:    commit,
		Canonical: "skills/acme/x",
	})
	if err := st.Write(consumer); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := Install(profileSkillOnly, nil, &stdout, &stderr); err != nil {
		t.Fatalf("Install: %v stderr=%s", err, stderr.String())
	}
	if _, err := os.Stat("skills/acme/x/SKILL.md"); err != nil {
		t.Errorf("canonical SKILL.md missing: %v", err)
	}
	if _, err := os.Stat(".claude/skills/acme/x/SKILL.md"); err != nil {
		t.Errorf("runtime fan-out missing: %v", err)
	}
}

func makeUpstreamAgent(t *testing.T, name, body string) (string, string) {
	return makeUpstreamAgentFiles(t, name, map[string]string{"AGENT.md": body})
}

// makeUpstreamAgentFiles seeds an upstream agent repo with arbitrary
// files at the root of the asset tree (e.g. AGENT.md + agent.toml).
func makeUpstreamAgentFiles(t *testing.T, name string, files map[string]string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "t@t"},
		{"config", "user.name", "t"},
		{"config", "core.autocrlf", "false"},
	} {
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("%s", out)
		}
	}
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)
	os.MkdirAll("src", 0755)
	for path, body := range files {
		full := filepath.Join("src", path)
		if d := filepath.Dir(full); d != "." {
			os.MkdirAll(d, 0755)
		}
		os.WriteFile(full, []byte(body), 0644)
	}
	tree, _ := git.WriteTreeFromDir(filepath.Join(dir, "src"))
	commit, _ := git.CommitTree(tree, "init")
	_ = git.UpdateRef("refs/assets/agent/"+name, commit)
	return dir, commit
}

func TestInstallAgentFansOutAsFile(t *testing.T) {
	body := "---\nname: reviewer\n---\n# reviewer agent body"
	upstream, commit := makeUpstreamAgent(t, "acme/reviewer", body)

	consumer := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(consumer)
	defer os.Chdir(wd)
	exec.Command("git", "init", "-q").Run()
	exec.Command("git", "config", "core.autocrlf", "false").Run()

	st := state.New()
	st.Set(kind.Agent, "acme/reviewer", state.Entry{
		Spec:      "v1.0.0",
		Remote:    upstream,
		Runtimes:  map[string]state.RuntimeOverride{"claude": {}},
		Version:   "v1.0.0",
		Commit:    commit,
		Canonical: "agents/acme/reviewer",
	})
	if err := st.Write(consumer); err != nil {
		t.Fatal(err)
	}

	if err := Install(profileAgentOnly, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Install: %v", err)
	}

	if info, err := os.Stat("agents/acme/reviewer/AGENT.md"); err != nil || info.IsDir() {
		t.Errorf("canonical AGENT.md missing or not a file: %v", err)
	}

	rtPath := ".claude/agents/acme/reviewer.md"
	info, err := os.Stat(rtPath)
	if err != nil {
		t.Fatalf("runtime agent file missing at %s: %v", rtPath, err)
	}
	if info.IsDir() {
		t.Errorf("%s is a directory; agents must materialize as a single .md file", rtPath)
	}
	got, _ := os.ReadFile(rtPath)
	if string(got) != body {
		t.Errorf("runtime agent body mismatch:\n got=%q\nwant=%q", string(got), body)
	}
}

// A6: an agent installed for two runtimes (claude + codex) fans out to
// both destinations using the registry's per-runtime From mapping.
func TestInstallAgentDualRuntimes_ClaudeAndCodex(t *testing.T) {
	mdBody := "---\nname: reviewer\n---\n# reviewer"
	tomlBody := "name = \"reviewer\"\n"
	upstream, commit := makeUpstreamAgentFiles(t, "acme/reviewer", map[string]string{
		"AGENT.md":   mdBody,
		"agent.toml": tomlBody,
	})

	consumer := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(consumer)
	defer os.Chdir(wd)
	exec.Command("git", "init", "-q").Run()
	exec.Command("git", "config", "core.autocrlf", "false").Run()

	st := state.New()
	st.Set(kind.Agent, "acme/reviewer", state.Entry{
		Spec:      "v1.0.0",
		Remote:    upstream,
		Runtimes:  map[string]state.RuntimeOverride{"claude": {}, "codex": {}},
		Version:   "v1.0.0",
		Commit:    commit,
		Canonical: "agents/acme/reviewer",
	})
	if err := st.Write(consumer); err != nil {
		t.Fatal(err)
	}

	if err := Install(profileAgentOnly, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Install: %v", err)
	}

	got, err := os.ReadFile(".claude/agents/acme/reviewer.md")
	if err != nil {
		t.Fatalf("claude fan-out missing: %v", err)
	}
	if string(got) != mdBody {
		t.Errorf("claude body mismatch: %q", string(got))
	}
	got, err = os.ReadFile(".codex/agents/acme/reviewer.toml")
	if err != nil {
		t.Fatalf("codex fan-out missing: %v", err)
	}
	if string(got) != tomlBody {
		t.Errorf("codex body mismatch: %q", string(got))
	}
}

// Install respects a per-entry "to" override and writes to the
// overridden path instead of the registry default.
func TestInstall_RespectsToOverride(t *testing.T) {
	upstream, commit := makeUpstreamSkill(t, "acme/x")

	consumer := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(consumer)
	defer os.Chdir(wd)
	exec.Command("git", "init", "-q").Run()
	exec.Command("git", "config", "core.autocrlf", "false").Run()

	st := state.New()
	st.Set(kind.Skill, "acme/x", state.Entry{
		Spec:      "v1.0.0",
		Remote:    upstream,
		Runtimes:  map[string]state.RuntimeOverride{"claude": {To: ".custom/claude/acme/x/"}},
		Version:   "v1.0.0",
		Commit:    commit,
		Canonical: "skills/acme/x",
	})
	if err := st.Write(consumer); err != nil {
		t.Fatal(err)
	}

	if err := Install(profileSkillOnly, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if _, err := os.Stat(".custom/claude/acme/x/SKILL.md"); err != nil {
		t.Errorf("override path not materialized: %v", err)
	}
	if _, err := os.Stat(".claude/skills/acme/x/SKILL.md"); err == nil {
		t.Errorf("registry default path should not exist when overridden")
	}
}

func TestInstallRejectsMissingCommit(t *testing.T) {
	consumer := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(consumer)
	defer os.Chdir(wd)
	exec.Command("git", "init", "-q").Run()

	st := state.New()
	st.Set(kind.Skill, "acme/x", state.Entry{
		Spec:      "v1.0.0",
		Remote:    "https://example.invalid/repo.git",
		Canonical: "skills/acme/x",
	})
	if err := st.Write(consumer); err != nil {
		t.Fatal(err)
	}

	err := Install(profileSkillOnly, nil, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Errorf("expected error on entry with empty commit")
	}
}

// Manifest override: git-skill.yaml in the asset tree redirects the
// claude runtime's To path. The canonical materializes as usual but
// the fan-out lands at the manifest-declared path.
func TestInstall_ManifestOverridesRegistry(t *testing.T) {
	upstream, commit := makeUpstreamSkillFiles(t, "acme/x", map[string]string{
		"SKILL.md":        "---\nname: acme/x\n---\n# acme/x",
		manifest.Filename: "runtimes:\n  claude:\n    to: .alt/skills/<name>/\n",
	})

	consumer := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(consumer)
	defer os.Chdir(wd)
	exec.Command("git", "init", "-q").Run()
	exec.Command("git", "config", "core.autocrlf", "false").Run()

	st := state.New()
	st.Set(kind.Skill, "acme/x", state.Entry{
		Spec:      "v1.0.0",
		Remote:    upstream,
		Runtimes:  map[string]state.RuntimeOverride{"claude": {}},
		Version:   "v1.0.0",
		Commit:    commit,
		Canonical: "skills/acme/x",
	})
	if err := st.Write(consumer); err != nil {
		t.Fatal(err)
	}

	if err := Install(profileSkillOnly, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if _, err := os.Stat(".alt/skills/acme/x/SKILL.md"); err != nil {
		t.Errorf("manifest-declared path not materialized: %v", err)
	}
	if _, err := os.Stat(".claude/skills/acme/x/SKILL.md"); err == nil {
		t.Errorf("registry default path should not exist when overridden by manifest")
	}
}

// Manifest-only runtime: a runtime not registered in the built-in
// registry can still materialize when the manifest provides a full
// From+To mapping.
func TestInstall_ManifestOnlyRuntime(t *testing.T) {
	upstream, commit := makeUpstreamSkillFiles(t, "acme/x", map[string]string{
		"SKILL.md":        "---\nname: acme/x\n---\n# acme/x",
		manifest.Filename: "runtimes:\n  future:\n    from: .\n    to: .future/skills/<name>/\n",
	})

	consumer := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(consumer)
	defer os.Chdir(wd)
	exec.Command("git", "init", "-q").Run()
	exec.Command("git", "config", "core.autocrlf", "false").Run()

	st := state.New()
	st.Set(kind.Skill, "acme/x", state.Entry{
		Spec:      "v1.0.0",
		Remote:    upstream,
		Runtimes:  map[string]state.RuntimeOverride{"future": {}},
		Version:   "v1.0.0",
		Commit:    commit,
		Canonical: "skills/acme/x",
	})
	if err := st.Write(consumer); err != nil {
		t.Fatal(err)
	}

	if err := Install(profileSkillOnly, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if _, err := os.Stat(".future/skills/acme/x/SKILL.md"); err != nil {
		t.Errorf("manifest-only runtime path not materialized: %v", err)
	}
}

// Lock override beats manifest: when assets.json declares a To and
// the manifest also declares one, the lock entry wins.
func TestInstall_LockOverrideBeatsManifest(t *testing.T) {
	upstream, commit := makeUpstreamSkillFiles(t, "acme/x", map[string]string{
		"SKILL.md":        "---\nname: acme/x\n---\n# acme/x",
		manifest.Filename: "runtimes:\n  claude:\n    to: .alt/skills/<name>/\n",
	})

	consumer := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(consumer)
	defer os.Chdir(wd)
	exec.Command("git", "init", "-q").Run()
	exec.Command("git", "config", "core.autocrlf", "false").Run()

	st := state.New()
	st.Set(kind.Skill, "acme/x", state.Entry{
		Spec:      "v1.0.0",
		Remote:    upstream,
		Runtimes:  map[string]state.RuntimeOverride{"claude": {To: ".lock/skills/<name>/"}},
		Version:   "v1.0.0",
		Commit:    commit,
		Canonical: "skills/acme/x",
	})
	if err := st.Write(consumer); err != nil {
		t.Fatal(err)
	}

	if err := Install(profileSkillOnly, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if _, err := os.Stat(".lock/skills/acme/x/SKILL.md"); err != nil {
		t.Errorf("lock override path not materialized: %v", err)
	}
	if _, err := os.Stat(".alt/skills/acme/x/SKILL.md"); err == nil {
		t.Errorf("manifest path should not exist when lock override is set")
	}
}

// Malformed manifest at install time: the install bubbles up the
// parse failure rather than silently ignoring the file.
func TestInstall_MalformedManifestFails(t *testing.T) {
	upstream, commit := makeUpstreamSkillFiles(t, "acme/x", map[string]string{
		"SKILL.md":        "---\nname: acme/x\n---\n# acme/x",
		manifest.Filename: "runtimes: { not closed\n",
	})

	consumer := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(consumer)
	defer os.Chdir(wd)
	exec.Command("git", "init", "-q").Run()
	exec.Command("git", "config", "core.autocrlf", "false").Run()

	st := state.New()
	st.Set(kind.Skill, "acme/x", state.Entry{
		Spec:      "v1.0.0",
		Remote:    upstream,
		Runtimes:  map[string]state.RuntimeOverride{"claude": {}},
		Version:   "v1.0.0",
		Commit:    commit,
		Canonical: "skills/acme/x",
	})
	if err := st.Write(consumer); err != nil {
		t.Fatal(err)
	}

	if err := Install(profileSkillOnly, nil, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
		t.Error("expected install to fail on malformed manifest")
	}
}

// Lock override's From field also goes through <name> substitution,
// matching the registry/manifest convention. This locks behavior of
// the override.From branch in resolveMapping.
func TestInstall_LockOverrideFromSubstitution(t *testing.T) {
	upstream, commit := makeUpstreamSkillFiles(t, "acme/x", map[string]string{
		"SKILL.md":                 "---\nname: acme/x\n---\n# canonical",
		"variants/acme/x/SKILL.md": "---\nname: acme/x\n---\n# variant",
	})

	consumer := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(consumer)
	defer os.Chdir(wd)
	exec.Command("git", "init", "-q").Run()
	exec.Command("git", "config", "core.autocrlf", "false").Run()

	st := state.New()
	st.Set(kind.Skill, "acme/x", state.Entry{
		Spec:   "v1.0.0",
		Remote: upstream,
		Runtimes: map[string]state.RuntimeOverride{"claude": {
			From: "variants/<name>",
			To:   ".claude/skills/<name>/",
		}},
		Version:   "v1.0.0",
		Commit:    commit,
		Canonical: "skills/acme/x",
	})
	if err := st.Write(consumer); err != nil {
		t.Fatal(err)
	}

	if err := Install(profileSkillOnly, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Install: %v", err)
	}
	got, err := os.ReadFile(".claude/skills/acme/x/SKILL.md")
	if err != nil {
		t.Fatalf("expected fan-out from substituted From: %v", err)
	}
	if !strings.Contains(string(got), "# variant") {
		t.Errorf("expected the variant SKILL.md, got %q", string(got))
	}
}

// A manifest-only runtime (not in the registry) without a `to` cannot
// materialize, and install must surface that error. manifest.Load
// already rejects entries with neither from nor to, so this covers the
// case where `from` is set but `to` is empty - which install.go's
// resolveMapping rejects at line "mapping.To == "".
func TestInstall_ManifestOnlyRuntimeMissingToFails(t *testing.T) {
	upstream, commit := makeUpstreamSkillFiles(t, "acme/x", map[string]string{
		"SKILL.md":        "---\nname: acme/x\n---\n# acme/x",
		manifest.Filename: "runtimes:\n  future:\n    from: .\n",
	})

	consumer := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(consumer)
	defer os.Chdir(wd)
	exec.Command("git", "init", "-q").Run()
	exec.Command("git", "config", "core.autocrlf", "false").Run()

	st := state.New()
	st.Set(kind.Skill, "acme/x", state.Entry{
		Spec:      "v1.0.0",
		Remote:    upstream,
		Runtimes:  map[string]state.RuntimeOverride{"future": {}},
		Version:   "v1.0.0",
		Commit:    commit,
		Canonical: "skills/acme/x",
	})
	if err := st.Write(consumer); err != nil {
		t.Fatal(err)
	}

	if err := Install(profileSkillOnly, nil, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
		t.Error("expected install to fail when manifest-only runtime has no 'to'")
	}
}

// Project-level .git-skill/runtimes.yaml can declare a brand-new
// runtime; the lock entry references it; Install materializes it at
// the configured path. This proves the user/project config layer is
// merged into the registry the install path consults.
func TestInstall_ProjectRuntimesConfigAddsRuntime(t *testing.T) {
	t.Setenv("GIT_SKILL_USER_CONFIG", filepath.Join(t.TempDir(), "no-user-config.yaml"))

	upstream, commit := makeUpstreamSkill(t, "acme/x")

	consumer := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(consumer)
	defer os.Chdir(wd)
	exec.Command("git", "init", "-q").Run()
	exec.Command("git", "config", "core.autocrlf", "false").Run()

	if err := os.MkdirAll(".git-skill", 0755); err != nil {
		t.Fatal(err)
	}
	cfg := `runtimes:
  myfuture:
    skill:
      to: .myfuture/skills/<name>/
`
	if err := os.WriteFile(filepath.Join(".git-skill", "runtimes.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}

	st := state.New()
	st.Set(kind.Skill, "acme/x", state.Entry{
		Spec:      "v1.0.0",
		Remote:    upstream,
		Runtimes:  map[string]state.RuntimeOverride{"myfuture": {}},
		Version:   "v1.0.0",
		Commit:    commit,
		Canonical: "skills/acme/x",
	})
	if err := st.Write(consumer); err != nil {
		t.Fatal(err)
	}

	if err := Install(profileSkillOnly, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if _, err := os.Stat(".myfuture/skills/acme/x/SKILL.md"); err != nil {
		t.Errorf("project-config runtime path not materialized: %v", err)
	}
}

// Precedence chain end-to-end: built-in registry < user runtimes.yaml
// < project runtimes.yaml < manifest < lock override. All four layers
// declare a different `to` for the same (runtime, kind); the lock
// override must win.
func TestInstall_FullPrecedenceChain(t *testing.T) {
	userDir := t.TempDir()
	userCfg := filepath.Join(userDir, "runtimes.yaml")
	if err := os.WriteFile(userCfg, []byte(`runtimes:
  claude:
    skill:
      to: .user/<name>/
`), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_SKILL_USER_CONFIG", userCfg)

	upstream, commit := makeUpstreamSkillFiles(t, "acme/x", map[string]string{
		"SKILL.md":        "---\nname: acme/x\n---\n# acme/x",
		manifest.Filename: "runtimes:\n  claude:\n    to: .manifest/<name>/\n",
	})

	consumer := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(consumer)
	defer os.Chdir(wd)
	exec.Command("git", "init", "-q").Run()
	exec.Command("git", "config", "core.autocrlf", "false").Run()

	if err := os.MkdirAll(".git-skill", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(".git-skill", "runtimes.yaml"), []byte(`runtimes:
  claude:
    skill:
      to: .project/<name>/
`), 0644); err != nil {
		t.Fatal(err)
	}

	st := state.New()
	st.Set(kind.Skill, "acme/x", state.Entry{
		Spec:      "v1.0.0",
		Remote:    upstream,
		Runtimes:  map[string]state.RuntimeOverride{"claude": {To: ".lock/<name>/"}},
		Version:   "v1.0.0",
		Commit:    commit,
		Canonical: "skills/acme/x",
	})
	if err := st.Write(consumer); err != nil {
		t.Fatal(err)
	}

	if err := Install(profileSkillOnly, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Install: %v", err)
	}

	if _, err := os.Stat(".lock/acme/x/SKILL.md"); err != nil {
		t.Errorf("lock override should win: %v", err)
	}
	for _, dead := range []string{".user", ".project", ".manifest", ".claude"} {
		if _, err := os.Stat(filepath.Join(dead, "acme", "x", "SKILL.md")); err == nil {
			t.Errorf("%s/acme/x/SKILL.md should not exist when lock override wins", dead)
		}
	}
}

// Remove must clean up manifest-only fan-out paths. This proves the
// load-before-delete ordering in remove.go: the manifest is read while
// the canonical tree still exists, so the cleanup loop knows where
// .future/skills/acme/x/ was materialized.
func TestRemove_CleansUpManifestOnlyRuntime(t *testing.T) {
	upstream, commit := makeUpstreamSkillFiles(t, "acme/x", map[string]string{
		"SKILL.md":        "---\nname: acme/x\n---\n# acme/x",
		manifest.Filename: "runtimes:\n  future:\n    from: .\n    to: .future/skills/<name>/\n",
	})

	consumer := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(consumer)
	defer os.Chdir(wd)
	exec.Command("git", "init", "-q").Run()
	exec.Command("git", "config", "core.autocrlf", "false").Run()

	st := state.New()
	st.Set(kind.Skill, "acme/x", state.Entry{
		Spec:      "v1.0.0",
		Remote:    upstream,
		Runtimes:  map[string]state.RuntimeOverride{"future": {}},
		Version:   "v1.0.0",
		Commit:    commit,
		Canonical: "skills/acme/x",
	})
	if err := st.Write(consumer); err != nil {
		t.Fatal(err)
	}
	if err := Install(profileSkillOnly, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if _, err := os.Stat(".future/skills/acme/x/SKILL.md"); err != nil {
		t.Fatalf("precondition: install should materialize manifest-only path: %v", err)
	}

	if err := Remove(profileSkillOnly, []string{"acme/x"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := os.Stat(".future/skills/acme/x"); !os.IsNotExist(err) {
		t.Errorf("manifest-only fan-out path should be removed, got err=%v", err)
	}
	if _, err := os.Stat("skills/acme/x"); !os.IsNotExist(err) {
		t.Errorf("canonical tree should be removed, got err=%v", err)
	}
}

// Remove must also clean up runtimes that exist only because a
// project-level .git-skill/runtimes.yaml declared them. The cleanup
// path runs reg.Resolve(rt, k, name) - which already sees the project
// config - so the test pins the behavior: an entry whose runtime is
// not in the built-in registry but IS in the project config gets its
// fan-out path removed.
func TestRemove_CleansUpProjectConfigRuntime(t *testing.T) {
	t.Setenv("GIT_SKILL_USER_CONFIG", filepath.Join(t.TempDir(), "no-user-config.yaml"))

	upstream, commit := makeUpstreamSkill(t, "acme/x")

	consumer := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(consumer)
	defer os.Chdir(wd)
	exec.Command("git", "init", "-q").Run()
	exec.Command("git", "config", "core.autocrlf", "false").Run()

	if err := os.MkdirAll(".git-skill", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(".git-skill", "runtimes.yaml"), []byte(`runtimes:
  myfuture:
    skill:
      to: .myfuture/skills/<name>/
`), 0644); err != nil {
		t.Fatal(err)
	}

	st := state.New()
	st.Set(kind.Skill, "acme/x", state.Entry{
		Spec:      "v1.0.0",
		Remote:    upstream,
		Runtimes:  map[string]state.RuntimeOverride{"myfuture": {}},
		Version:   "v1.0.0",
		Commit:    commit,
		Canonical: "skills/acme/x",
	})
	if err := st.Write(consumer); err != nil {
		t.Fatal(err)
	}
	if err := Install(profileSkillOnly, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if _, err := os.Stat(".myfuture/skills/acme/x/SKILL.md"); err != nil {
		t.Fatalf("precondition: install should materialize project-config path: %v", err)
	}

	if err := Remove(profileSkillOnly, []string{"acme/x"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := os.Stat(".myfuture/skills/acme/x"); !os.IsNotExist(err) {
		t.Errorf("project-config fan-out path should be removed, got err=%v", err)
	}
	if _, err := os.Stat("skills/acme/x"); !os.IsNotExist(err) {
		t.Errorf("canonical tree should be removed, got err=%v", err)
	}
}

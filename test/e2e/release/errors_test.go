package release

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readAssetsJSON parses assets.json from dir and returns the raw bytes + parsed structure.
func readAssetsJSON(t *testing.T, dir string) ([]byte, map[string]any) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "assets.json"))
	if err != nil {
		t.Fatalf("read assets.json: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("parse assets.json: %v", err)
	}
	return data, m
}

// D1: empty commit on install - hand-write assets.json with no commit field.
// install must fail; assets.json must not be mutated.
func TestD1_EmptyCommitOnInstall(t *testing.T) {
	consumer := newRepo(t)
	consumer.runCLI("init")

	// Hand-write a state entry with no commit
	fixture := []byte(`{
  "version": 1,
  "config": {"skillsRoot": "skills", "agentsRoot": "agents"},
  "assets": {
    "skill": {
      "acme/x": {
        "spec": "v1.0.0",
        "remote": "file:///nonexistent",
        "commit": "",
        "canonical": "skills/acme/x"
      }
    }
  }
}
`)
	if err := os.WriteFile(filepath.Join(consumer.dir, "assets.json"), fixture, 0644); err != nil {
		t.Fatal(err)
	}

	beforeHash := fileHash(t, filepath.Join(consumer.dir, "assets.json"))

	_, _, err := consumer.runCLI("install")
	if err == nil {
		t.Errorf("install with empty commit should fail")
	}

	afterHash := fileHash(t, filepath.Join(consumer.dir, "assets.json"))
	if beforeHash != afterHash {
		t.Errorf("assets.json mutated on failed install (D1)")
	}
}

// D2: invalid --from - file:///nonexistent/path. add must fail; assets.json unchanged.
func TestD2_InvalidFrom(t *testing.T) {
	consumer := newRepo(t)
	consumer.runCLI("init")

	beforeHash := fileHash(t, filepath.Join(consumer.dir, "assets.json"))

	_, _, err := consumer.runCLI("add", "acme/x@v1.0.0", "--from", "file:///nonexistent/totally/gone")
	if err == nil {
		t.Errorf("add with invalid --from should fail")
	}

	afterHash := fileHash(t, filepath.Join(consumer.dir, "assets.json"))
	if beforeHash != afterHash {
		t.Errorf("assets.json mutated on failed add (D2)")
	}
}

// D3: kind mismatch warning. Write SKILL.md with frontmatter kind: skill, then
// commit via git-asset with --kind=agent. --kind overrides frontmatter, and the
// implementation emits a warning when frontmatter kind disagrees with --kind flag.
// Commit must succeed.
func TestD3_KindMismatchWarning(t *testing.T) {
	producer := newRepo(t)

	// SKILL.md with frontmatter declaring kind=skill
	writeFile(t, filepath.Join(producer.dir, "src", "SKILL.md"), []byte("---\nkind: skill\n---\n# skill body"))

	// Commit via git-asset with --kind=agent (mismatch: frontmatter says skill, flag says agent)
	_, stderr, err := producer.runCLIAs("git-asset", "commit", "acme/x", "-m", "v1", "--path", "src", "--kind=agent")
	// Commit should succeed (warning, not error)
	if err != nil {
		t.Fatalf("commit with kind mismatch should succeed (got error): %v\n%s", err, stderr)
	}
	// stderr must contain a warning about the mismatch
	if !strings.Contains(strings.ToLower(stderr), "warn") {
		t.Errorf("expected warning in stderr about kind mismatch, got: %q", stderr)
	}
}

// D4: update with no matching tag - spec ^v2.0.0 against remote with only v1.x.x.
// update must fail; assets.json unchanged.
func TestD4_UpdateNoMatchingTag(t *testing.T) {
	producer := newRepo(t)
	bare := newBareRepo(t)
	mustRun(t, producer.dir, "git", "remote", "add", "origin", bare)

	writeFile(t, filepath.Join(producer.dir, "src", "SKILL.md"), []byte("# skill"))
	producer.runCLI("commit", "acme/x", "-m", "v1", "--path", "src")
	producer.runCLI("tag", "acme/x", "v1.0.0")
	producer.runCLI("push", "origin")

	// Consumer adds with v1.0.0 first
	consumer := newRepo(t)
	consumer.runCLI("init")
	_, stderr, err := consumer.runCLI("add", "acme/x@v1.0.0", "--from", bare, "--runtime", "claude")
	if err != nil {
		t.Fatalf("add: %v\n%s", err, stderr)
	}

	// Manually edit assets.json to use ^v2.0.0 spec (no v2.x tags exist on remote).
	// add with @v1.0.0 stores spec as "v1.0.0" (exact), so we replace that literal.
	data, err := os.ReadFile(filepath.Join(consumer.dir, "assets.json"))
	if err != nil {
		t.Fatal(err)
	}
	patched := strings.ReplaceAll(string(data), `"v1.0.0"`, `"^v2.0.0"`)
	if err := os.WriteFile(filepath.Join(consumer.dir, "assets.json"), []byte(patched), 0644); err != nil {
		t.Fatal(err)
	}

	beforeHash := fileHash(t, filepath.Join(consumer.dir, "assets.json"))

	_, _, err = consumer.runCLI("update")
	if err == nil {
		t.Errorf("update with no matching ^v2.0.0 tag should fail")
	}

	afterHash := fileHash(t, filepath.Join(consumer.dir, "assets.json"))
	if beforeHash != afterHash {
		t.Errorf("assets.json mutated on failed update (D4)")
	}
}

// D5: hand-edited spec without re-resolve. Set commit manually; plain install
// must use that commit verbatim (no re-resolution).
func TestD5_InstallUsesCommitVerbatim(t *testing.T) {
	producer := newRepo(t)
	bare := newBareRepo(t)
	mustRun(t, producer.dir, "git", "remote", "add", "origin", bare)

	writeFile(t, filepath.Join(producer.dir, "src", "SKILL.md"), []byte("# v1 content"))
	producer.runCLI("commit", "acme/x", "-m", "v1", "--path", "src")
	producer.runCLI("tag", "acme/x", "v1.0.0")
	producer.runCLI("push", "origin")

	consumer := newRepo(t)
	consumer.runCLI("init")
	_, stderr, err := consumer.runCLI("add", "acme/x@v1.0.0", "--from", bare, "--runtime", "claude")
	if err != nil {
		t.Fatalf("add: %v\n%s", err, stderr)
	}

	// Read the pinned commit from assets.json
	_, parsed := readAssetsJSON(t, consumer.dir)
	assets := parsed["assets"].(map[string]any)
	skills := assets["skill"].(map[string]any)
	entry := skills["acme/x"].(map[string]any)
	pinnedCommit := entry["commit"].(string)

	// Remove on-disk copies to force re-install
	os.RemoveAll(filepath.Join(consumer.dir, "skills"))
	os.RemoveAll(filepath.Join(consumer.dir, ".claude"))

	// install must succeed using the verbatim pinned commit - no re-resolution
	_, stderr, err = consumer.runCLI("install")
	if err != nil {
		t.Fatalf("install with verbatim commit: %v\n%s", err, stderr)
	}

	// Verify assets.json commit is still the same (not re-resolved)
	_, parsed2 := readAssetsJSON(t, consumer.dir)
	assets2 := parsed2["assets"].(map[string]any)
	skills2 := assets2["skill"].(map[string]any)
	entry2 := skills2["acme/x"].(map[string]any)
	commit2 := entry2["commit"].(string)
	if commit2 != pinnedCommit {
		t.Errorf("install changed commit pin from %q to %q (should use verbatim)", pinnedCommit, commit2)
	}

	// Canonical file should be installed
	assertFileExists(t, filepath.Join(consumer.dir, "skills", "acme", "x", "SKILL.md"))
}

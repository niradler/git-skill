package release

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A1: single-author lifecycle - init, commit, tag, push, add, remove.
func TestA1_SingleAuthorLifecycle(t *testing.T) {
	// Producer setup
	producer := newRepo(t)
	bare := newBareRepo(t)
	mustRun(t, producer.dir, "git", "remote", "add", "origin", bare)

	// Write SKILL.md in src/
	writeFile(t, filepath.Join(producer.dir, "src", "SKILL.md"), []byte("---\nname: acme/x\n---\n# acme/x skill"))

	// commit, tag, push
	_, stderr, err := producer.runCLI("commit", "acme/x", "-m", "v1", "--path", "src")
	if err != nil {
		t.Fatalf("commit: %v\nstderr: %s", err, stderr)
	}
	_, stderr, err = producer.runCLI("tag", "acme/x", "v1.0.0")
	if err != nil {
		t.Fatalf("tag: %v\nstderr: %s", err, stderr)
	}
	_, stderr, err = producer.runCLI("push", "origin")
	if err != nil {
		t.Fatalf("push: %v\nstderr: %s", err, stderr)
	}

	// Consumer setup
	consumer := newRepo(t)
	_, stderr, err = consumer.runCLI("init")
	if err != nil {
		t.Fatalf("init: %v\nstderr: %s", err, stderr)
	}

	// Add skill from the bare upstream
	_, stderr, err = consumer.runCLI("add", "acme/x@v1.0.0", "--from", bare, "--runtime", "claude")
	if err != nil {
		t.Fatalf("add: %v\nstderr: %s", err, stderr)
	}

	// Assert canonical and runtime paths exist
	assertFileExists(t, filepath.Join(consumer.dir, "skills", "acme", "x", "SKILL.md"))
	assertFileExists(t, filepath.Join(consumer.dir, ".claude", "skills", "acme", "x", "SKILL.md"))

	// Assert assets.json has a 40-char hex commit
	data, err := os.ReadFile(filepath.Join(consumer.dir, "assets.json"))
	if err != nil {
		t.Fatalf("read assets.json: %v", err)
	}
	var top map[string]any
	if err := json.Unmarshal(data, &top); err != nil {
		t.Fatalf("parse assets.json: %v", err)
	}
	assets := top["assets"].(map[string]any)
	skills := assets["skill"].(map[string]any)
	entry := skills["acme/x"].(map[string]any)
	commit, _ := entry["commit"].(string)
	if len(commit) != 40 {
		t.Errorf("commit in assets.json should be 40-char hex, got %q", commit)
	}

	// Remove and assert absence
	_, stderr, err = consumer.runCLI("remove", "acme/x")
	if err != nil {
		t.Fatalf("remove: %v\nstderr: %s", err, stderr)
	}
	assertFileAbsent(t, filepath.Join(consumer.dir, "skills", "acme", "x", "SKILL.md"))
	assertFileAbsent(t, filepath.Join(consumer.dir, ".claude", "skills", "acme", "x", "SKILL.md"))

	// Re-read assets.json - entry must be gone
	data, _ = os.ReadFile(filepath.Join(consumer.dir, "assets.json"))
	var top2 map[string]any
	json.Unmarshal(data, &top2)
	assets2 := top2["assets"].(map[string]any)
	if skillMap, ok := assets2["skill"]; ok {
		if sm, ok := skillMap.(map[string]any); ok {
			if _, found := sm["acme/x"]; found {
				t.Errorf("assets.json still has acme/x after remove")
			}
		}
	}
}

// A2: iterate + update. v1.0.0 published, consumer adds with ^v1.0.0 range.
// Producer publishes v1.1.0. Consumer update resolves new commit.
func TestA2_IterateAndUpdate(t *testing.T) {
	producer := newRepo(t)
	bare := newBareRepo(t)
	mustRun(t, producer.dir, "git", "remote", "add", "origin", bare)

	// v1.0.0
	writeFile(t, filepath.Join(producer.dir, "src", "SKILL.md"), []byte("# v1.0.0 content"))
	producer.runCLI("commit", "acme/x", "-m", "v1", "--path", "src")
	producer.runCLI("tag", "acme/x", "v1.0.0")
	producer.runCLI("push", "origin")

	// Consumer: add with range spec
	consumer := newRepo(t)
	consumer.runCLI("init")
	_, stderr, err := consumer.runCLI("add", "acme/x@^v1.0.0", "--from", bare, "--runtime", "claude")
	if err != nil {
		t.Fatalf("add: %v\nstderr: %s", err, stderr)
	}

	// Capture pre-update commit from assets.json
	preHash := fileHash(t, filepath.Join(consumer.dir, "assets.json"))

	// Producer publishes v1.1.0 with new content
	writeFile(t, filepath.Join(producer.dir, "src", "SKILL.md"), []byte("# v1.1.0 content"))
	producer.runCLI("commit", "acme/x", "-m", "v1.1.0", "--path", "src")
	producer.runCLI("tag", "acme/x", "v1.1.0")
	producer.runCLI("push", "origin")

	// Consumer: update
	_, stderr, err = consumer.runCLI("update")
	if err != nil {
		t.Fatalf("update: %v\nstderr: %s", err, stderr)
	}

	// assets.json must have changed
	postHash := fileHash(t, filepath.Join(consumer.dir, "assets.json"))
	if preHash == postHash {
		t.Errorf("assets.json unchanged after update - expected new commit")
	}

	// Runtime file must contain new content
	assertFileContains(t, filepath.Join(consumer.dir, ".claude", "skills", "acme", "x", "SKILL.md"), "v1.1.0 content")
}

// A3: mixed kinds. Producer publishes acme/sk (skill) and acme/ag (agent).
// Consumer adds both. Verify canonical and runtime paths for each kind.
func TestA3_MixedKinds(t *testing.T) {
	producer := newRepo(t)
	bare := newBareRepo(t)
	mustRun(t, producer.dir, "git", "remote", "add", "origin", bare)

	// Skill: acme/sk
	writeFile(t, filepath.Join(producer.dir, "skill-src", "SKILL.md"), []byte("# skill body"))
	_, stderr, err := producer.runCLIAs("git-skill", "commit", "acme/sk", "-m", "skill v1", "--path", "skill-src")
	if err != nil {
		t.Fatalf("commit skill: %v\nstderr: %s", err, stderr)
	}
	producer.runCLIAs("git-skill", "tag", "acme/sk", "v1.0.0")

	// Agent: acme/ag
	writeFile(t, filepath.Join(producer.dir, "agent-src", "AGENT.md"), []byte("# agent body"))
	_, stderr, err = producer.runCLIAs("git-agent", "commit", "acme/ag", "-m", "agent v1", "--path", "agent-src")
	if err != nil {
		t.Fatalf("commit agent: %v\nstderr: %s", err, stderr)
	}
	producer.runCLIAs("git-agent", "tag", "acme/ag", "v1.0.0")

	// Push both via git-asset
	_, stderr, err = producer.runCLIAs("git-asset", "push", "origin")
	if err != nil {
		t.Fatalf("push: %v\nstderr: %s", err, stderr)
	}

	// Consumer: add both
	consumer := newRepo(t)
	consumer.runCLI("init")

	_, stderr, err = consumer.runCLIAs("git-skill", "add", "acme/sk@v1.0.0", "--from", bare, "--runtime", "claude")
	if err != nil {
		t.Fatalf("add skill: %v\nstderr: %s", err, stderr)
	}
	_, stderr, err = consumer.runCLIAs("git-agent", "add", "acme/ag@v1.0.0", "--from", bare, "--runtime", "claude")
	if err != nil {
		t.Fatalf("add agent: %v\nstderr: %s", err, stderr)
	}

	// Skill: canonical dir, runtime dir
	assertFileExists(t, filepath.Join(consumer.dir, "skills", "acme", "sk", "SKILL.md"))
	assertFileExists(t, filepath.Join(consumer.dir, ".claude", "skills", "acme", "sk", "SKILL.md"))

	// Agent: canonical dir contains AGENT.md; runtime is a single file (NOT a directory)
	assertFileExists(t, filepath.Join(consumer.dir, "agents", "acme", "ag", "AGENT.md"))
	rtAgent := filepath.Join(consumer.dir, ".claude", "agents", "acme", "ag.md")
	info, err := os.Stat(rtAgent)
	if err != nil {
		t.Fatalf("agent runtime file missing at %s: %v", rtAgent, err)
	}
	if info.IsDir() {
		t.Errorf("%s is a directory; agent runtime must be a single .md file", rtAgent)
	}
}

// A4: multi-remote. Two producer repos (A: skill, B: agent). Consumer adds
// from both. plain install (no args) restores both.
func TestA4_MultiRemote(t *testing.T) {
	// Producer A: skill
	producerA := newRepo(t)
	bareA := newBareRepo(t)
	mustRun(t, producerA.dir, "git", "remote", "add", "origin", bareA)
	writeFile(t, filepath.Join(producerA.dir, "src", "SKILL.md"), []byte("# skill from A"))
	producerA.runCLIAs("git-skill", "commit", "acme/sk", "-m", "v1", "--path", "src")
	producerA.runCLIAs("git-skill", "tag", "acme/sk", "v1.0.0")
	producerA.runCLIAs("git-skill", "push", "origin")

	// Producer B: agent
	producerB := newRepo(t)
	bareB := newBareRepo(t)
	mustRun(t, producerB.dir, "git", "remote", "add", "origin", bareB)
	writeFile(t, filepath.Join(producerB.dir, "src", "AGENT.md"), []byte("# agent from B"))
	producerB.runCLIAs("git-agent", "commit", "acme/ag", "-m", "v1", "--path", "src")
	producerB.runCLIAs("git-agent", "tag", "acme/ag", "v1.0.0")
	producerB.runCLIAs("git-agent", "push", "origin")

	// Consumer: add from both
	consumer := newRepo(t)
	consumer.runCLI("init")
	_, stderr, err := consumer.runCLIAs("git-skill", "add", "acme/sk@v1.0.0", "--from", bareA, "--runtime", "claude")
	if err != nil {
		t.Fatalf("add skill: %v\nstderr: %s", err, stderr)
	}
	_, stderr, err = consumer.runCLIAs("git-agent", "add", "acme/ag@v1.0.0", "--from", bareB, "--runtime", "claude")
	if err != nil {
		t.Fatalf("add agent: %v\nstderr: %s", err, stderr)
	}

	// Remove on-disk copies to simulate a fresh clone
	os.RemoveAll(filepath.Join(consumer.dir, "skills"))
	os.RemoveAll(filepath.Join(consumer.dir, "agents"))
	os.RemoveAll(filepath.Join(consumer.dir, ".claude"))

	// Plain install should restore both from assets.json
	_, stderr, err = consumer.runCLIAs("git-asset", "install")
	if err != nil {
		t.Fatalf("install: %v\nstderr: %s", err, stderr)
	}

	assertFileExists(t, filepath.Join(consumer.dir, "skills", "acme", "sk", "SKILL.md"))
	assertFileExists(t, filepath.Join(consumer.dir, "agents", "acme", "ag", "AGENT.md"))
}

// A5: dev mode. Producer publishes acme/x. Consumer adds with --dev.
// Edit canonical SKILL.md - content should reflect edit via symlink/junction/copy.
// install again - canonical NOT clobbered.
//
// In dev mode install preserves local edits to the canonical tree - only
// non-dev installs refresh canonical from the pinned commit (see installOne
// in cmd/git-skill/commands/install.go).
func TestA5_DevMode(t *testing.T) {
	producer := newRepo(t)
	bare := newBareRepo(t)
	mustRun(t, producer.dir, "git", "remote", "add", "origin", bare)
	writeFile(t, filepath.Join(producer.dir, "src", "SKILL.md"), []byte("# original content"))
	producer.runCLI("commit", "acme/x", "-m", "v1", "--path", "src")
	producer.runCLI("tag", "acme/x", "v1.0.0")
	producer.runCLI("push", "origin")

	consumer := newRepo(t)
	consumer.runCLI("init")
	_, stderr, err := consumer.runCLI("add", "acme/x@v1.0.0", "--from", bare, "--runtime", "claude", "--dev")
	if err != nil {
		t.Fatalf("add --dev: %v\nstderr: %s", err, stderr)
	}

	canonical := filepath.Join(consumer.dir, "skills", "acme", "x", "SKILL.md")
	runtime := filepath.Join(consumer.dir, ".claude", "skills", "acme", "x", "SKILL.md")

	// Overwrite canonical
	writeFile(t, canonical, []byte("# edited content"))

	// In dev mode: runtime should reflect the edit (symlink/junction) or be a copy (both acceptable)
	// We try to read via os.Lstat to see what we have; either way read through it.
	rtContent, rerr := os.ReadFile(runtime)
	if rerr != nil {
		t.Fatalf("read runtime after dev edit: %v", rerr)
	}

	// If it's a symlink/junction, content matches. If it's a copy, content is stale - that's acceptable.
	// Document: on Windows, junction is used for directories; the edit propagates automatically.
	// On non-Windows, a relative symlink propagates the edit.
	// A copy (fallback) would NOT propagate, but install must not clobber canonical.
	_ = rtContent // both cases accepted

	// install again - canonical must NOT be clobbered with "original content"
	_, stderr, err = consumer.runCLI("install")
	if err != nil {
		t.Fatalf("install after dev edit: %v\nstderr: %s", err, stderr)
	}
	body, _ := os.ReadFile(canonical)
	if strings.Contains(string(body), "original content") {
		t.Errorf("install clobbered canonical dev-edited file with original content")
	}
	if !strings.Contains(string(body), "edited content") {
		t.Errorf("canonical dev file should still contain 'edited content' after install, got: %s", body)
	}
}

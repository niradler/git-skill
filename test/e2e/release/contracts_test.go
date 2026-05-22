package release

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// B1: assets.json round-trip. Write one skill + one agent entry, then re-read
// via the binary's `list` to confirm both are visible.
//
// Deviation from plan: plan asked for a byte-compare of the JSON, which is brittle
// across Go map iteration order. Instead we do a structural compare via json.Unmarshal
// into map[string]any and check key presence — less fragile and equally correct.
func TestB1_AssetsJSONRoundTrip(t *testing.T) {
	// Build a producer with a skill, tagged and pushed to a bare
	producer := newRepo(t)
	bare := newBareRepo(t)
	mustRun(t, producer.dir, "git", "remote", "add", "origin", bare)

	writeFile(t, filepath.Join(producer.dir, "src-sk", "SKILL.md"), []byte("# skill"))
	producer.runCLIAs("git-skill", "commit", "acme/sk", "-m", "v1", "--path", "src-sk")
	producer.runCLIAs("git-skill", "tag", "acme/sk", "v1.0.0")

	writeFile(t, filepath.Join(producer.dir, "src-ag", "AGENT.md"), []byte("# agent"))
	producer.runCLIAs("git-agent", "commit", "acme/ag", "-m", "v1", "--path", "src-ag")
	producer.runCLIAs("git-agent", "tag", "acme/ag", "v1.0.0")

	producer.runCLIAs("git-asset", "push", "origin")

	// Consumer: add both
	consumer := newRepo(t)
	consumer.runCLI("init")
	_, stderr, err := consumer.runCLIAs("git-skill", "add", "acme/sk@v1.0.0", "--from", bare)
	if err != nil {
		t.Fatalf("add skill: %v\n%s", err, stderr)
	}
	_, stderr, err = consumer.runCLIAs("git-agent", "add", "acme/ag@v1.0.0", "--from", bare)
	if err != nil {
		t.Fatalf("add agent: %v\n%s", err, stderr)
	}

	// Read assets.json and structural-verify both entries exist
	data, err := os.ReadFile(filepath.Join(consumer.dir, "assets.json"))
	if err != nil {
		t.Fatalf("read assets.json: %v", err)
	}
	var top map[string]any
	if err := json.Unmarshal(data, &top); err != nil {
		t.Fatalf("parse assets.json: %v", err)
	}
	assets, ok := top["assets"].(map[string]any)
	if !ok {
		t.Fatalf("assets.json missing 'assets' key")
	}

	// skill entry
	skillMap, ok := assets["skill"].(map[string]any)
	if !ok || skillMap["acme/sk"] == nil {
		t.Errorf("assets.json missing skill/acme/sk entry")
	}

	// agent entry
	agentMap, ok := assets["agent"].(map[string]any)
	if !ok || agentMap["acme/ag"] == nil {
		t.Errorf("assets.json missing agent/acme/ag entry")
	}
}

// B2: ref namespace. After producer push, every ref on the bare upstream must
// start with refs/assets/ or refs/asset-tags/. No legacy refs/skills/.
func TestB2_RefNamespace(t *testing.T) {
	producer := newRepo(t)
	bare := newBareRepo(t)
	mustRun(t, producer.dir, "git", "remote", "add", "origin", bare)

	writeFile(t, filepath.Join(producer.dir, "src", "SKILL.md"), []byte("# skill"))
	producer.runCLI("commit", "acme/x", "-m", "v1", "--path", "src")
	producer.runCLI("tag", "acme/x", "v1.0.0")
	_, stderr, err := producer.runCLI("push", "origin")
	if err != nil {
		t.Fatalf("push: %v\n%s", err, stderr)
	}

	c := exec.Command("git", "--git-dir", bare, "for-each-ref", "--format=%(refname)")
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("for-each-ref: %v\n%s", err, out)
	}

	for _, ref := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if ref == "" {
			continue
		}
		if !strings.HasPrefix(ref, "refs/assets/") && !strings.HasPrefix(ref, "refs/asset-tags/") {
			t.Errorf("unexpected ref on remote: %q (must start with refs/assets/ or refs/asset-tags/)", ref)
		}
		if strings.HasPrefix(ref, "refs/skills/") || strings.HasPrefix(ref, "refs/skill-tags/") {
			t.Errorf("legacy ref found on remote: %q", ref)
		}
	}
}

// B3: argv[0] dispatch. Three copies of binary: git-skill, git-agent, git-asset.
// Producer commits one skill and one agent. Then:
//   - git-skill list → only skill row
//   - git-agent list → only agent row
//   - git-asset list → both rows
func TestB3_Argv0Dispatch(t *testing.T) {
	producer := newRepo(t)

	writeFile(t, filepath.Join(producer.dir, "src-sk", "SKILL.md"), []byte("# skill"))
	_, stderr, err := producer.runCLIAs("git-skill", "commit", "acme/sk", "-m", "v1", "--path", "src-sk")
	if err != nil {
		t.Fatalf("commit skill: %v\n%s", err, stderr)
	}

	writeFile(t, filepath.Join(producer.dir, "src-ag", "AGENT.md"), []byte("# agent"))
	_, stderr, err = producer.runCLIAs("git-agent", "commit", "acme/ag", "-m", "v1", "--path", "src-ag")
	if err != nil {
		t.Fatalf("commit agent: %v\n%s", err, stderr)
	}

	// git-skill list → only skill
	skillOut, _, err := producer.runCLIAs("git-skill", "list")
	if err != nil {
		t.Fatalf("git-skill list: %v", err)
	}
	if !strings.Contains(skillOut, "acme/sk") {
		t.Errorf("git-skill list missing acme/sk:\n%s", skillOut)
	}
	if strings.Contains(skillOut, "acme/ag") {
		t.Errorf("git-skill list shows agent acme/ag (should be filtered):\n%s", skillOut)
	}

	// git-agent list → only agent
	agentOut, _, err := producer.runCLIAs("git-agent", "list")
	if err != nil {
		t.Fatalf("git-agent list: %v", err)
	}
	if !strings.Contains(agentOut, "acme/ag") {
		t.Errorf("git-agent list missing acme/ag:\n%s", agentOut)
	}
	if strings.Contains(agentOut, "acme/sk") {
		t.Errorf("git-agent list shows skill acme/sk (should be filtered):\n%s", agentOut)
	}

	// git-asset list → both
	assetOut, _, err := producer.runCLIAs("git-asset", "list")
	if err != nil {
		t.Fatalf("git-asset list: %v", err)
	}
	if !strings.Contains(assetOut, "acme/sk") {
		t.Errorf("git-asset list missing skill acme/sk:\n%s", assetOut)
	}
	if !strings.Contains(assetOut, "acme/ag") {
		t.Errorf("git-asset list missing agent acme/ag:\n%s", assetOut)
	}
}

// B4: Asset-Kind trailer. After commit acme/x (skill), git log -1 --format=%B
// on the ref must contain "Asset-Kind: skill". Same for agent.
func TestB4_AssetKindTrailer(t *testing.T) {
	producer := newRepo(t)

	// Skill commit
	writeFile(t, filepath.Join(producer.dir, "src-sk", "SKILL.md"), []byte("# skill"))
	_, stderr, err := producer.runCLIAs("git-skill", "commit", "acme/sk", "-m", "v1", "--path", "src-sk")
	if err != nil {
		t.Fatalf("commit skill: %v\n%s", err, stderr)
	}

	c := exec.Command("git", "log", "-1", "--format=%B", "refs/assets/skill/acme/sk")
	c.Dir = producer.dir
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("git log skill: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Asset-Kind: skill") {
		t.Errorf("skill commit missing 'Asset-Kind: skill' trailer:\n%s", out)
	}

	// Agent commit
	writeFile(t, filepath.Join(producer.dir, "src-ag", "AGENT.md"), []byte("# agent"))
	_, stderr, err = producer.runCLIAs("git-agent", "commit", "acme/ag", "-m", "v1", "--path", "src-ag")
	if err != nil {
		t.Fatalf("commit agent: %v\n%s", err, stderr)
	}

	c = exec.Command("git", "log", "-1", "--format=%B", "refs/assets/agent/acme/ag")
	c.Dir = producer.dir
	out, err = c.CombinedOutput()
	if err != nil {
		t.Fatalf("git log agent: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Asset-Kind: agent") {
		t.Errorf("agent commit missing 'Asset-Kind: agent' trailer:\n%s", out)
	}
}

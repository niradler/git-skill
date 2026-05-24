# Version-controlling AI agent assets with git-skill and GitHub

If you've used Claude, Cursor, Codex, or any modern coding agent for more than a week, you've probably accumulated a folder of carefully tuned **skills** - system prompts, style guides, code review checklists, debugging protocols - and maybe a few **sub-agents** for specialized tasks. And if you work on a team, you've probably also discovered that those files don't have a story.

There's no version history. No diff when something changes. No way to pin a known-good revision. No mechanism for "I want the version Alice was using last Tuesday." You drop a folder in `.claude/skills/` and pray nothing drifts.

This is a solved problem for source code. It's been solved since 2005. The infrastructure is sitting in every developer's `git` binary. **git-skill** is a small CLI that says: skills and agents are directories of text files, that's exactly what git is good at, let's just use git.

This article walks through the full workflow: authoring an asset, publishing it to GitHub, consuming it from another repo, upgrading it, and integrating it into a team's day-to-day.

## The core idea

git-skill stores each asset as a git tree object under custom refs:

```
refs/assets/skill/code-review                    → latest commit (branch-like)
refs/asset-tags/skill/code-review/v1.0.0         → tagged release (immutable by convention)

refs/assets/agent/security-auditor               → latest commit
refs/asset-tags/agent/security-auditor/v0.3.0    → tagged release
```

The two built-in kinds are `skill` (materializes as a full directory tree at the runtime target) and `agent` (materializes as a single marker file at the target). Every commit carries an `Asset-Kind: skill` or `Asset-Kind: agent` trailer, so a downstream tool can recover the kind from the commit alone.

These refs live in any existing git repository - your dedicated assets repo, your dotfiles, even your main project repo. Push them to GitHub with `git skill push origin` and anyone with read access can `git skill add` + `git skill install` them.

The CLI shells out to git plumbing commands (`hash-object`, `mktree`, `commit-tree`, `update-ref`) and wraps them in a friendly interface. The spec ([SKILL-FORMAT.md](../SKILL-FORMAT.md)) is independent of the implementation - anyone can write a tool that reads and writes the same format.

## Installation

```bash
# Pin to a specific release — @latest can lag behind newly cut tags via
# the Go module proxy for tens of minutes.
go install github.com/niradler/git-skill/cmd/git-skill@v0.2.0

# Or build from source
git clone https://github.com/niradler/git-skill
cd git-skill
go build -o git-skill ./cmd/git-skill
sudo mv git-skill /usr/local/bin/
```

Because the binary is named `git-skill`, git automatically discovers it as a subcommand:

```bash
git skill --help
```

The same binary serves three roles based on its invocation name. Symlink it to enable all three:

```bash
ln -s "$(which git-skill)" /usr/local/bin/git-agent
ln -s "$(which git-skill)" /usr/local/bin/git-asset

git skill list   # operates on skills
git agent list   # operates on agents
git asset list --kind skill   # kind-agnostic
```

## Your first skill - author side

Walk through this in any existing git repository (or create one fresh with `git init`).

```bash
git skill init
```

This scaffolds `assets.json` at the repo root and adds a managed block to `.gitignore`. Author your skill under `skills/<name>/`:

```bash
mkdir -p skills/code-review
$EDITOR skills/code-review/SKILL.md
```

A minimal `SKILL.md`:

```markdown
---
name: code-review
description: Guidelines for reviewing pull requests
---

# Code review checklist

...
```

Snapshot it into `refs/assets/skill/code-review` and tag a release:

```bash
git skill commit code-review --path skills/code-review -m "initial cut"
# Tags require the leading 'v'. v0.2.0 enforces ^v\d+\.\d+\.\d+(-...)? — a
# bare 1.0.0 is rejected.
git skill tag code-review v1.0.0
git skill push origin
```

`git skill push origin` pushes `refs/assets/skill/*` and `refs/asset-tags/skill/*`. From the `git-agent` persona it would push the agent namespace; from `git-asset` (no kind) it would push both. **If you're publishing both kinds in one workflow step (typical in CI), use `git asset push origin` — `git skill push` only handles the skill namespace.**

## Consuming the asset from another repo

In a different repo (your app, your dotfiles, your CI image), onboard the asset and materialize it:

```bash
git skill init
git skill add code-review@v1.0.0 \
    --from https://github.com/acme/skills \
    --runtime claude
```

What just happened:

1. `add` resolved `v1.0.0` against the tags on the upstream remote, recorded both the spec (`v1.0.0`) and the resolved version + commit SHA in `assets.json`, and stored the canonical tree under `skills/code-review/`.
2. Because `--runtime claude` was passed, it also fanned out into `.claude/skills/code-review/`. **Without `--runtime`, only the canonical copy is written** — useful for skills you want to manage manually, but for Claude Code consumers `--runtime claude` is what you want.
3. The default `.gitignore` scaffolded by `init` ignores `/skills/` and `.claude/skills/` — so by default only `assets.json` is tracked, and teammates run `git skill install` on clone to materialize. If you want PR-reviewable materialized diffs (the model used in the [consumer demo repo](https://github.com/niradler/git-skill-consumer-demo)), remove those entries from `.gitignore`.

Commit `assets.json` alongside your code. Now any teammate who clones the repo can run `git skill install` and get byte-identical files.

`@<spec>` accepts an exact tag (`v1.0.0`, `v1.0.0-dev.7`) or a semver range when the producer ships prod releases. For dev tags (which are immutable points minted per CI run — see the next section), use exact pins.

## Upgrading

When the author cuts a new tag:

```bash
# author
git skill commit code-review --path skills/code-review -m "Add async-fn check"
git skill tag code-review v1.1.0
git skill push origin

# consumer
git skill update code-review
git add assets.json && git commit -m "bump code-review to v1.1.0"
```

`update` re-resolves the spec against the upstream tags. If the spec is a range like `^1.0.0`, the new `v1.1.0` gets picked up automatically. **If the spec is an exact tag (`v1.0.0` or any `-dev.N`), `update` is a no-op** — exact pins are immutable by definition. To move to a new exact tag, `git skill remove <name>` and re-add at the new tag.

## Dev tags and CI cadence

The publish flow above is one tag per author command. In a team setting, you typically want CI to mint a tag on every push to `main` so consumers can pick it up immediately. The convention is **dev tags**: `v<base>-dev.<run_number>` where `<base>` is your floor semver (e.g. `0.1.0` from `version.txt`) and `<run_number>` is the GitHub Actions run counter — monotonically increasing for the repo.

A skill at `v0.1.0-dev.42` is a snapshot of `refs/assets/skill/<name>` as it existed at the 42nd publish CI run. When the team is ready to declare a stable release, a separate manual workflow promotes a specific commit to a bare semver tag (`v1.0.0`).

This split — automatic immutable dev tags + manual prod tags — gives consumers a stable spec form for production (`^1.0.0`) and a sharp opt-in handle for early access (`v0.1.0-dev.42`).

## CI publish workflow

A minimal GitHub Actions job that publishes every changed skill/agent on push to `main`:

```yaml
- uses: actions/checkout@v4
  with: { fetch-depth: 0 }

- uses: actions/setup-go@v5
  with: { go-version: "1.22" }

- name: Install git-skill (+ git-asset alias)
  run: |
    go install github.com/niradler/git-skill/cmd/git-skill@v0.2.0
    gobin="$(go env GOPATH)/bin"
    echo "$gobin" >> "$GITHUB_PATH"
    # argv[0] dispatch: linking as git-asset enables the multi-kind profile.
    ln -sf "$gobin/git-skill" "$gobin/git-asset"

- name: Fetch existing asset refs
  # actions/checkout only pulls refs/heads/* (+ refs/tags/* with fetch-tags).
  # Custom refs/assets/* are NOT fetched, so `git skill commit` would
  # create orphan histories and the push would be rejected as non-FF.
  run: git fetch origin '+refs/assets/*:refs/assets/*' '+refs/asset-tags/*:refs/asset-tags/*' || true

- name: Detect changed skills/agents and tag dev versions
  run: |
    set -euo pipefail
    diff_range="${{ github.event.before }}...${{ github.sha }}"
    paths=$(git diff --name-only $diff_range -- 'skills/*' 'agents/*' \
      | awk -F'/' 'NF>=2 && ($1=="skills" || $1=="agents") {print $1"/"$2}' \
      | sort -u)

    for p in $paths; do
      kind_dir="$(dirname $p)"; name="$(basename $p)"
      [ "$kind_dir" = "skills" ] && kind_flag="" || kind_flag="--kind agent"

      base=$(tr -d '[:space:]' < "$p/version.txt" 2>/dev/null || echo "0.1.0")
      dev_version="${base}-dev.${{ github.run_number }}"

      # Kind flag goes BEFORE positional args. The CLI uses flag.Parse
      # which stops at the first non-flag positional.
      git skill commit $kind_flag --path "$p" -m "ci: ${{ github.sha }}" "$name"
      git skill tag    $kind_flag "$name" "v${dev_version}"
    done

- name: Push asset refs and tags (both kinds)
  # git skill push only pushes skill refs. git asset push covers both.
  run: git asset push origin
```

Four CI gotchas worth pinning in your memory, all of which I hit setting this up:

1. **Pin git-skill via `@v<X.Y.Z>`, not `@latest`.** The Go module proxy can lag behind newly cut tags by tens of minutes.
2. **Fetch `refs/assets/*` explicitly** before any `git skill commit`. `actions/checkout` doesn't pull custom refs.
3. **Use `git asset push` for multi-kind producers.** `git skill push` is skill-only.
4. **`--kind` flag before positional args.** `git skill tag --kind agent foo v0.1.0`, not `git skill tag foo --kind agent v0.1.0`.

For a working reference, see [`niradler/git-skill-demos/.github/workflows/publish.yml`](https://github.com/niradler/git-skill-demos/blob/main/.github/workflows/publish.yml).

## Promote: dev → prod

A separate manual workflow turns a dev tag into a prod release. Inputs are `skill` (name) and `version` (bare semver, no `-dev`). The workflow:

1. Validates inputs and runs structure-tier evals against the current canonical tree (no API key needed).
2. Tags the canonical commit with `v<version>`.
3. Pushes the new tag.

Reference: [`promote.yml`](https://github.com/niradler/git-skill-demos/blob/main/.github/workflows/promote.yml). The author triggers it via `gh workflow run promote.yml -f skill=code-review -f version=1.0.0` after the dev version has been validated locally.

## Evals

Skills tend to break in two ways: **structure drift** (a typo in the frontmatter, a missing required section, a broken pointer to an eval prompt) and **behavior drift** (the skill still parses fine but the model's outputs degrade). git-skill assumes both are worth catching, but they have different testing tiers.

**Structure tier — runs in CI.** Deterministic checks against the skill files: frontmatter present and well-formed, required H2 sections present, `eval/prompts.json` parses, every prompt id has a matching `Behavioral.<id>` section in `assertions.md`. No model call needed.

**Behavior tier — runs locally.** The author opens Claude Code, reads `eval/prompts.json` + `eval/assertions.md`, spawns a `Task` subagent per prompt with the full SKILL.md inlined verbatim, and grades the responses against the binary checkbox assertions. Pass = `passed/total >= passing_score` from `eval.config.yaml`. The promote workflow gates on this happening *before* the PR is opened — CI never sees an API key.

Why local-first? Because subagents are the testing harness the producer already has installed, and because CI-with-API-keys means token costs scale with PR volume in ways nobody enjoys. Reference skill: [`running-skill-evals`](https://github.com/niradler/git-skill-demos/tree/main/skills/running-skill-evals).

## Working with sub-agents

Agents share the same workflow, just under a different ref namespace and via the `git-agent` persona:

```bash
git skill init   # init is universal — same assets.json holds both kinds
mkdir -p agents/security-auditor
$EDITOR agents/security-auditor/AGENT.md
git agent commit security-auditor --path agents/security-auditor -m "v1"
git agent tag security-auditor v0.1.0
git agent push origin
```

On the consumer side:

```bash
git agent add security-auditor@v0.1.0 \
    --from https://github.com/acme/agents \
    --runtime claude
```

Agents materialize as a single marker file at the runtime target - Claude expects `AGENT.md` at `.claude/agents/<name>.md`; Codex expects `agent.toml` at `.codex/agents/<name>.toml`. The built-in registry covers both; per-asset overrides live in the asset's `git-skill.yaml`.

## Customizing where assets materialize

Three layers of customization sit on top of the built-in registry. Lowest precedence first:

1. **`~/.config/git-skill/runtimes.yaml`** - your machine-wide defaults.
2. **`<repo>/.git-skill/runtimes.yaml`** - repo-wide policy, committed alongside `assets.json`.
3. **`git-skill.yaml` in the asset tree** - author-declared overrides that ship with the asset.
4. **`--target runtime=path` on `git skill add`** - per-asset override pinned in the lock entry.

Example: send `claude` skills to an alternate path repo-wide:

```yaml
# .git-skill/runtimes.yaml
runtimes:
  claude:
    skill:
      to: .ai/claude/skills/<name>/
```

Or extend the tool to a runtime that isn't in the built-in registry:

```yaml
runtimes:
  myfuture:
    skill:
      to: .myfuture/skills/<name>/
    agent:
      from: AGENT.md
      to: .myfuture/agents/<name>.md
```

## Using private repos

git-skill shells out to plain `git fetch` / `git push`, so whatever credentials work for `git clone` work here:

```bash
# once per machine
gh auth login

git skill add internal-rubric@v2.0.0 \
    --from https://github.com/acme/private-skills \
    --runtime claude
```

## Inspecting without the CLI

Assets are plain git refs - any git-aware tool reads them natively:

```bash
git cat-file -p refs/assets/skill/code-review
# tree abc...
# author Your Name <you@example.com>
#
# initial cut
#
# Asset-Kind: skill

git ls-tree refs/assets/skill/code-review
# 100644 blob ...  SKILL.md

git log refs/assets/skill/code-review --oneline
git diff refs/asset-tags/skill/code-review/v1.0.0 \
         refs/asset-tags/skill/code-review/v1.1.0
```

This is the point: the CLI is convenience, not protocol. Drop it and your assets are still readable, still diff-able, still versioned.

## Cross-platform notes

- **Path separators in `assets.json`.** v0.2.0 on Windows writes OS-native separators (`skills\\code-review`) into the `canonical` field. Normalize to forward slashes before committing — lock files written on one OS should be byte-identical to lock files written on any other.
- **Symlinks on Windows.** Require developer mode or an admin shell. Without those privileges the materializer falls back to a plain copy (the changelog calls this out under "Cross-platform materialization").
- **Line endings.** `core.autocrlf=true` on Windows will rewrite line endings in materialized files. If you care about byte-stable runtime trees across the team, add `*.md text eol=lf` to `.gitattributes` in the consumer repo.

## End-to-end reference

Two repos that exercise everything above:

- **Producer:** [`niradler/git-skill-demos`](https://github.com/niradler/git-skill-demos) — six skills + one agent, CI publish + promote workflows, eval format, the `running-skill-evals` skill.
- **Consumer:** [`niradler/git-skill-consumer-demo`](https://github.com/niradler/git-skill-consumer-demo) — four PRs that walk through install, upgrade, multi-skill, and rollback against the producer above. See [`docs/demo-flows.md`](https://github.com/niradler/git-skill-consumer-demo/blob/main/docs/demo-flows.md).

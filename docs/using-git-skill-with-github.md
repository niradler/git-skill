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
# Either build from source
git clone https://github.com/niradler/git-skill
cd git-skill
go build -o git-skill ./cmd/git-skill
sudo mv git-skill /usr/local/bin/

# Or via go install
go install github.com/niradler/git-skill/cmd/git-skill@latest
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
git skill tag code-review 1.0.0
git skill push origin
```

`git skill push origin` pushes `refs/assets/skill/*` and `refs/asset-tags/skill/*`. From the `git-agent` persona it would push the agent namespace; from `git-asset` (no kind) it would push both.

## Consuming the asset from another repo

In a different repo (your app, your dotfiles, your CI image), onboard the asset and materialize it:

```bash
git skill init
git skill add acme/code-review@^1.0.0 \
    --from https://github.com/acme/skills
git skill install
```

What just happened:

1. `add` resolved `^1.0.0` against the tags on the upstream remote, recorded both the spec (`^1.0.0`) and the resolved version + commit SHA in `assets.json`, and stored the canonical tree under `skills/acme/code-review/`.
2. `install` read `assets.json`, fetched the pinned commit if missing, restored the canonical tree, and fanned out into every configured runtime path (default: `.claude/skills/acme/code-review/`).

Commit `assets.json` alongside your code. Now any teammate who clones the repo can run `git skill install` and get byte-identical files.

## Upgrading

When the author cuts a new tag:

```bash
# author
git skill commit code-review --path skills/code-review -m "Add async-fn check"
git skill tag code-review 1.1.0
git skill push origin

# consumer (per machine, once)
git skill update code-review
git add assets.json && git commit -m "bump code-review to v1.1.0"
```

`update` re-resolves the spec against the upstream tags. Because the spec was `^1.0.0`, `1.1.0` is in range and gets picked up automatically. Pin to an exact version with `acme/code-review@1.1.0` if you want explicit control instead.

## Working with sub-agents

Agents share the same workflow, just under a different ref namespace and via the `git-agent` persona:

```bash
git agent init
mkdir -p agents/security-auditor
$EDITOR agents/security-auditor/AGENT.md
git agent commit security-auditor --path agents/security-auditor -m "v1"
git agent tag security-auditor 0.1.0
git agent push origin
```

On the consumer side:

```bash
git agent add acme/security-auditor@^0.1.0 \
    --from https://github.com/acme/agents
git agent install
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

git skill add acme/internal-rubric@2.0.0 \
    --from https://github.com/acme/private-skills
git skill install
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

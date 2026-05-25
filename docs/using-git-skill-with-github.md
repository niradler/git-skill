# Version-controlling AI agent assets with git-skill and GitHub

If you've used Claude, Cursor, or Codex for more than a week, you've probably accumulated a folder of carefully tuned skills (system prompts, code review checklists, debugging protocols) and maybe a few sub-agents for specialized tasks. On a team, those files have no story. No version history. No diff when something changes. No way to pin a known-good revision. You drop a folder in `.claude/skills/` and pray nothing drifts.

Source code has had this solved since 2005. The infrastructure is sitting in every developer's `git` binary. **git-skill** says: skills and agents are directories of text files, that is exactly what git is good at, let's just use git.

This article walks the full workflow: authoring an asset, publishing to GitHub, consuming from another repo, upgrading, running evals, and integrating into a team.

## The core idea

git-skill stores each asset as a git tree under custom refs:

```
refs/assets/skill/code-review                    → latest commit (branch-like)
refs/asset-tags/skill/code-review/v1.0.0         → tagged release (immutable by convention)

refs/assets/agent/security-auditor               → latest commit
refs/asset-tags/agent/security-auditor/v0.3.0    → tagged release
```

The two built-in kinds are `skill` (a full directory tree, materialized at the runtime target) and `agent` (a single marker file inside the asset tree, materialized to a single file at the runtime target). Every commit carries an `Asset-Kind: skill` or `Asset-Kind: agent` trailer so a downstream tool can recover the kind from the commit alone.

These refs live in any git repository. Push them to GitHub with `git skill push origin` and anyone with read access can `git skill add` plus `git skill install` them.

The CLI shells out to git plumbing (`hash-object`, `mktree`, `commit-tree`, `update-ref`) and wraps it in a friendly interface. The spec ([SKILL-FORMAT.md](../SKILL-FORMAT.md)) is independent of the implementation. Anyone can write a tool that reads and writes the same format.

## Installation

```bash
# Pin to a specific release. @latest can lag tens of minutes behind newly cut tags
# via the Go module proxy.
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

git skill list                # operates on skills
git agent list                # operates on agents
git asset list --kind skill   # kind-agnostic
```

## Author a skill

Walk through this in any existing git repository (or `git init` a fresh one).

```bash
git skill init
```

This scaffolds `assets.json` at the repo root and adds a managed block to `.gitignore`. Author the skill under `skills/<name>/`:

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
# Tags require the leading 'v'. The CLI enforces ^v\d+\.\d+\.\d+(-...)?
# A bare 1.0.0 is rejected.
git skill tag code-review v1.0.0
git skill push origin
```

`git skill push origin` pushes the skill refs and tags. From the `git-agent` persona it pushes the agent namespace. From `git-asset` it pushes both. If you publish both kinds in one workflow step (typical in CI), use `git asset push origin`. `git skill push` only handles the skill namespace.

## Author an agent

Agents share the same workflow under a different ref namespace. The canonical asset tree contains a single marker file named `AGENT.md` (this is the source filename inside the asset tree, per the git-skill spec). It materializes to a runtime-specific destination on `install`. For Claude Code that destination is `.claude/agents/<name>.md`. For Codex it is `.codex/agents/<name>.toml`.

The `AGENT.md` body uses the Claude Code sub-agent format, so the materialized file is loadable as-is:

```bash
mkdir -p agents/security-auditor
$EDITOR agents/security-auditor/AGENT.md
```

`AGENT.md`:

```markdown
---
name: security-auditor
description: Scan a code diff or file set for OWASP-style vulnerabilities, hardcoded secrets, and missing auth checks.
---

You are a focused security review sub-agent. Read a diff or a small set of files and surface security issues. Do not refactor, restyle, or comment on architecture. Output findings only.

## Scope

- Injection (SQL, command, LDAP, template, header)
- XSS, path traversal, SSRF
- Unsafe deserialization, weak crypto, hardcoded keys
- Missing AuthN/AuthZ on new endpoints
- Secrets committed to the tree

## Output format

For each finding:

- **Severity:** critical | high | medium | low
- **Category:** one of the buckets above
- **Location:** path/to/file:line
- **Issue:** one sentence
- **Fix:** one sentence with the concrete remediation
```

Then publish:

```bash
git agent commit security-auditor --path agents/security-auditor -m "v1"
git agent tag security-auditor v0.1.0
git agent push origin
```

This is exactly what ships in the demo repo at [`niradler/git-skill-demos/agents/security-auditor`](https://github.com/niradler/git-skill-demos/tree/main/agents/security-auditor).

## Install assets in another repo

Real working examples against the demo repo:

```bash
git skill init

# Install a skill at a prod tag
git skill add code-review@v0.1.0 \
    --from https://github.com/niradler/git-skill-demos \
    --runtime claude

# Install another skill at a dev tag
git skill add using-git-skill@v0.1.0-dev.12 \
    --from https://github.com/niradler/git-skill-demos \
    --runtime claude

# Install an agent
git agent add security-auditor@v0.1.0-dev.12 \
    --from https://github.com/niradler/git-skill-demos \
    --runtime claude
```

What just happened:

1. `add` resolved the spec against the upstream tags, recorded both the spec (`v0.1.0`) and the resolved version plus commit SHA in `assets.json`, and stored the canonical tree under `skills/code-review/` (or `agents/security-auditor/`).
2. Because `--runtime claude` was passed, it also fanned out to `.claude/skills/code-review/` for the skill and `.claude/agents/security-auditor.md` for the agent. Without `--runtime`, only the canonical copy is written. For Claude Code consumers, pass `--runtime claude`.
3. The default `.gitignore` scaffolded by `init` ignores the canonical roots (`/skills/`, `/agents/`) and every runtime fanout directory (`.claude/skills/`, `.claude/agents/`, `.codex/agents/`, `.cursor/rules/`, `.agents/skills/`). By default only `assets.json` is tracked, and teammates run `git skill install` on clone to materialize. If you want PR-reviewable materialized diffs (the model used in the [consumer demo repo](https://github.com/niradler/git-skill-consumer-demo)), remove those entries from `.gitignore`.

Commit `assets.json` alongside your code. Any teammate who clones the repo can run `git skill install` and get byte-identical files.

`@<spec>` accepts an exact tag (`v1.0.0`, `v0.1.0-dev.7`) or a semver range when the producer ships prod releases. For dev tags (immutable points minted per CI run, see the next section), use exact pins.

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

`update` re-resolves the spec against the upstream tags. If the spec is a range like `^1.0.0`, the new `v1.1.0` gets picked up automatically. If the spec is an exact tag (`v1.0.0` or any `-dev.N`), `update` is a no-op. Exact pins are immutable by definition. To move to a new exact tag, `git skill remove <name>` and re-add at the new tag.

## Dev tags and CI cadence

In a team setting you typically want CI to mint a tag on every push to `main` so consumers can pick it up immediately. The convention is dev tags: `v<base>-dev.<run_number>` where `<base>` is your floor semver (e.g. `0.1.0` from `version.txt`) and `<run_number>` is the GitHub Actions run counter (monotonically increasing for the repo).

A skill at `v0.1.0-dev.42` is a snapshot of `refs/assets/skill/<name>` as it existed at the 42nd publish CI run. When the team is ready for a stable release, a separate manual workflow promotes a specific commit to a bare semver tag (`v1.0.0`).

This split (automatic immutable dev tags plus manual prod tags) gives consumers a stable spec form for production (`^1.0.0`) and a sharp opt-in handle for early access (`v0.1.0-dev.42`).

## CI publish workflow

A minimal GitHub Actions job that publishes every changed skill or agent on push to `main`:

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
  # actions/checkout only pulls refs/heads/* and refs/tags/*.
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

Four CI gotchas worth pinning in memory, all of which I hit setting this up:

1. **Pin git-skill via `@v<X.Y.Z>`, not `@latest`.** The Go module proxy can lag tens of minutes behind newly cut tags.
2. **Fetch `refs/assets/*` explicitly** before any `git skill commit`. `actions/checkout` does not pull custom refs.
3. **Use `git asset push` for multi-kind producers.** `git skill push` is skill-only.
4. **`--kind` flag before positional args.** `git skill tag --kind agent foo v0.1.0`, not `git skill tag foo --kind agent v0.1.0`.

Working reference: [`niradler/git-skill-demos/.github/workflows/publish.yml`](https://github.com/niradler/git-skill-demos/blob/main/.github/workflows/publish.yml).

## Promote: dev to prod

A separate manual workflow turns a dev tag into a prod release. Inputs are `skill` (name) and `version` (bare semver, no `-dev`). The workflow:

1. Validates inputs and runs structure-tier evals against the canonical tree (no API key needed).
2. Tags the canonical commit with `v<version>`.
3. Pushes the new tag.

Reference: [`promote.yml`](https://github.com/niradler/git-skill-demos/blob/main/.github/workflows/promote.yml). Trigger it with `gh workflow run promote.yml -f skill=code-review -f version=1.0.0` after the dev version has been validated.

## Evals

Skills break in two ways:

- **Structure drift.** A typo in the frontmatter, a missing required section, a broken pointer to an eval prompt. Caught by deterministic file checks, no model call needed. Easy to run in CI on every push.
- **Behavior drift.** The skill still parses fine but the model's outputs degrade. Needs real model runs against a fixed prompt set, graded against assertions.

For the behavior tier, the cleanest path is [microsoft/waza](https://github.com/microsoft/waza), a Go CLI for evaluating agent skills. Waza scaffolds eval suites with prompts, fixtures, and graders (`text`, `json_schema`, `prompt`-as-judge, `behavior`, `tool_calls`, `tool_constraint`, `action_sequence`, `skill_invocation`, inline `code`), runs them against a model, and reports pass rates.

Upstream waza targets hosted API executors. That works for raw-prompt benchmarks but misses the case where the thing under test is an agent prompt: a `SKILL.md` only meaningful inside an agent loop with tools, file access, and skill discovery. The fork [niradler/waza](https://github.com/niradler/waza) adds an `anthropic-cli` executor that shells out to the Claude CLI ([PR #1](https://github.com/niradler/waza/pull/1)):

```bash
claude --print --bare \
       --output-format stream-json --verbose \
       --permission-mode bypassPermissions \
       --add-dir <workspace> \
       --model <model-id>
```

Selected via `engine: anthropic-cli` in `.waza.yaml`. Per-task workspaces seeded from `inputs.files:` are mounted via `--add-dir` so the candidate agent can `Read` and `Write` real files. The executor parses the CLI's `stream-json --verbose` output, pairs `tool_use` and `tool_result` events by id, and feeds them to graders that depend on tool and skill telemetry. Every grader type works end-to-end against the CLI executor, not just hosted-API ones.

A CI workflow that runs evals against a skill on every PR:

```yaml
- uses: actions/checkout@v4

- name: Install Claude CLI
  run: npm install -g @anthropic-ai/claude-code

- name: Install waza (anthropic-cli fork)
  run: |
    git clone --depth 1 https://github.com/niradler/waza ~/waza
    cd ~/waza && go build -o /usr/local/bin/waza ./cmd/waza

- name: Run evals
  env:
    ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
  run: waza run skills/code-review/eval/suite.yaml
```

A suite is YAML with prompts, fixtures, graders, and a passing score. With `tasks: [tasks/*.yaml]` you keep large multi-file suites tidy without enumerating every file. A status flip from pass to fail blocks the PR. A score drop within tolerance surfaces as a warning.

## Working with sub-agents

The example in "Author an agent" above covers the producer side. On the consumer side:

```bash
git agent add security-auditor@v0.1.0-dev.12 \
    --from https://github.com/niradler/git-skill-demos \
    --runtime claude
```

`AGENT.md` from the canonical tree lands at `.claude/agents/security-auditor.md` and Claude Code picks it up on next launch. The destination filename is `<name>.md`, derived from the asset name. The source filename `AGENT.md` is the spec convention inside the canonical tree, not a runtime artifact.

The built-in runtime registry maps Claude and Codex out of the box. Per-asset overrides live in the asset's `git-skill.yaml`.

## Customizing where assets materialize

Four layers sit on top of the built-in registry. Lowest precedence first:

1. `~/.config/git-skill/runtimes.yaml`: machine-wide defaults.
2. `<repo>/.git-skill/runtimes.yaml`: repo-wide policy, committed alongside `assets.json`.
3. `git-skill.yaml` in the asset tree: author-declared overrides that ship with the asset.
4. `--target runtime=path` on `git skill add`: per-asset override pinned in the lock entry.

Send `claude` skills to an alternate path repo-wide:

```yaml
# .git-skill/runtimes.yaml
runtimes:
  claude:
    skill:
      to: .ai/claude/skills/<name>/
```

Extend the tool to a runtime that is not in the built-in registry:

```yaml
runtimes:
  myfuture:
    skill:
      to: .myfuture/skills/<name>/
    agent:
      from: AGENT.md
      to: .myfuture/agents/<name>.md
```

## Private repos

git-skill shells out to plain `git fetch` and `git push`. Whatever credentials work for `git clone` work here. Authenticate once with `gh auth login`, then `git skill add ... --from https://github.com/acme/private-skills` works the same as for a public repo.

## Inspecting without the CLI

Assets are plain git refs. Any git-aware tool reads them natively:

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

The CLI is convenience, not protocol. Drop it and your assets are still readable, diff-able, versioned.

## Cross-platform notes

- **Path separators in `assets.json`.** v0.2.0 on Windows writes OS-native separators (`skills\\code-review`) into the `canonical` field. Normalize to forward slashes before committing so lock files written on one OS are byte-identical to lock files written on any other.
- **Symlinks on Windows.** Require developer mode or an admin shell. Without those privileges the materializer falls back to a plain copy.
- **Line endings.** `core.autocrlf=true` on Windows will rewrite line endings in materialized files. If you care about byte-stable runtime trees across the team, add `*.md text eol=lf` to `.gitattributes` in the consumer repo.

## End-to-end reference

Two repos that exercise everything above:

- **Producer:** [`niradler/git-skill-demos`](https://github.com/niradler/git-skill-demos) covers six skills plus one agent, CI publish and promote workflows, eval format, and the `running-skill-evals` skill.
- **Consumer:** [`niradler/git-skill-consumer-demo`](https://github.com/niradler/git-skill-consumer-demo) walks four PRs through install, upgrade, multi-skill, and rollback against the producer above. See [`docs/demo-flows.md`](https://github.com/niradler/git-skill-consumer-demo/blob/main/docs/demo-flows.md).

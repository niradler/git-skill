# Version-controlling AI agent assets with git-skill and GitHub

Your team has 14 versions of the same code review skill across 5 repos, with no way to sync them. when one improves, the others don't. drop a folder in `.claude/skills/` and pray nothing drifts.

source code has had this solved since 2005, with a package manager layer on top for distribution. AI skills already live on GitHub. what's missing is the client.

git-skill is it. skills and sub-agents are packages, GitHub is the registry, git-skill is the package manager. one command to install, one file to track what's pinned, semver tags so you can roll back.

```bash
git skill add code-review@v1.0.0 --from https://github.com/niradler/git-skill-demos
```

that's the whole pitch. the rest of this post is how it works and how to run it on a team.

## The five-minute version

```bash
# author
git skill init
$EDITOR skills/code-review/SKILL.md
git skill commit code-review --path skills/code-review -m "initial cut"
git skill tag code-review v1.0.0
git skill push origin

# consumer
git skill init
git skill add code-review@v1.0.0 \
    --from https://github.com/niradler/git-skill-demos \
    --runtime claude
```

the author publishes a skill at v1.0.0. the consumer pins to that version, and `.claude/skills/code-review/` gets materialized on `git skill install`. everything tracked in `assets.json`, committed alongside your code.

## What's actually stored

skills and agents live as git trees under custom refs:

```
refs/assets/skill/code-review                    branch-like, latest commit
refs/asset-tags/skill/code-review/v1.0.0         tagged release, immutable by convention

refs/assets/agent/security-auditor               same shape, different namespace
refs/asset-tags/agent/security-auditor/v0.3.0
```

two built-in kinds: `skill` (a directory tree, materialized at the runtime target) and `agent` (a single marker file inside the asset tree, materialized to a single file at the runtime target). every commit carries an `Asset-Kind:` trailer so the kind is recoverable from the commit alone.

the CLI is a thin wrapper over git plumbing (`hash-object`, `mktree`, `commit-tree`, `update-ref`). the format spec ([SKILL-FORMAT.md](../SKILL-FORMAT.md)) is independent of the implementation. anyone can write a tool that reads and writes the same format. the CLI is convenience, not protocol. drop it and your assets are still readable, diffable, versioned.

that last sentence is the most important one in this post. you're not locked in.

## Installation

```bash
# pin to a specific release. @latest can lag tens of minutes behind newly cut tags
# via the Go module proxy.
go install github.com/niradler/git-skill/cmd/git-skill@v0.2.0
```

the binary is named `git-skill` so git discovers it as a subcommand:

```bash
git skill --help
```

the same binary serves three roles based on its invocation name. symlink it to enable all three:

```bash
ln -s "$(which git-skill)" /usr/local/bin/git-agent
ln -s "$(which git-skill)" /usr/local/bin/git-asset

git skill list                # skills only
git agent list                # agents only
git asset list --kind skill   # kind-agnostic
```

## Authoring a skill

`git skill init` scaffolds `assets.json` at the repo root and adds a managed block to `.gitignore`. author the skill under `skills/<name>/`:

```bash
mkdir -p skills/code-review
$EDITOR skills/code-review/SKILL.md
```

a minimal `SKILL.md`:

```markdown
---
name: code-review
description: Guidelines for reviewing pull requests
---

# Code review checklist

...
```

snapshot it and tag a release. tags require the leading `v`; a bare `1.0.0` is rejected.

```bash
git skill commit code-review --path skills/code-review -m "initial cut"
git skill tag code-review v1.0.0
git skill push origin
```

`git skill push origin` pushes the skill refs and tags. if you publish both skills and agents in one workflow step (typical in CI), use `git asset push origin` instead. `git skill push` is skill-only.

## Authoring an agent

same workflow, different ref namespace. the canonical tree contains a single marker file named `AGENT.md`. it materializes to a runtime-specific destination on install: `.claude/agents/<name>.md` for Claude Code, `.codex/agents/<name>.toml` for Codex.

the body uses the Claude Code sub-agent format, so the materialized file is loadable as-is:

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

publish:

```bash
git agent commit security-auditor --path agents/security-auditor -m "v1"
git agent tag security-auditor v0.1.0
git agent push origin
```

this is what ships in the demo repo at [`niradler/git-skill-demos/agents/security-auditor`](https://github.com/niradler/git-skill-demos/tree/main/agents/security-auditor).

## Installing in another repo

```bash
git skill init

git skill add code-review@v0.1.0 \
    --from https://github.com/niradler/git-skill-demos \
    --runtime claude

git skill add using-git-skill@v0.1.0-dev.12 \
    --from https://github.com/niradler/git-skill-demos \
    --runtime claude

git agent add security-auditor@v0.1.0-dev.12 \
    --from https://github.com/niradler/git-skill-demos \
    --runtime claude
```

what happens:

`add` resolves the spec against upstream tags, records both the spec (`v0.1.0`) and the resolved version plus commit SHA in `assets.json`, and stores the canonical tree under `skills/code-review/` or `agents/security-auditor/`.

`--runtime claude` fans out to `.claude/skills/code-review/` for skills and `.claude/agents/security-auditor.md` for agents. without `--runtime`, only the canonical copy is written. Claude Code consumers should pass `--runtime claude`.

the default `.gitignore` scaffolded by `init` ignores both the canonical roots (`/skills/`, `/agents/`) and every runtime fanout directory (`.claude/skills/`, `.claude/agents/`, `.codex/agents/`, `.cursor/rules/`, `.agents/skills/`). only `assets.json` is tracked. teammates run `git skill install` on clone to materialize. if you want PR-reviewable materialized diffs (the model used in the [consumer demo](https://github.com/niradler/git-skill-consumer-demo)), remove those entries from `.gitignore`.

commit `assets.json` alongside your code. anyone who clones the repo runs `git skill install` and gets byte-identical files.

`@<spec>` accepts an exact tag (`v1.0.0`, `v0.1.0-dev.7`) or a semver range when the producer ships prod releases. for dev tags (covered in the next section), use exact pins.

## Upgrading

```bash
# author
git skill commit code-review --path skills/code-review -m "Add async-fn check"
git skill tag code-review v1.1.0
git skill push origin

# consumer
git skill update code-review
git add assets.json && git commit -m "bump code-review to v1.1.0"
```

`update` re-resolves the spec against upstream tags. if the spec is a range like `^1.0.0`, the new `v1.1.0` is picked up automatically. if the spec is an exact tag, `update` is a no-op. exact pins are immutable by definition. to move to a new exact tag, `git skill remove <name>` and re-add at the new tag.

## Dev tags and CI cadence

in a team setting you want CI to mint a tag on every push to `main` so consumers can pick it up immediately. the convention is dev tags: `v<base>-dev.<run_number>` where `<base>` is your floor semver (e.g. `0.1.0` from `version.txt`) and `<run_number>` is the GitHub Actions run counter.

a skill at `v0.1.0-dev.42` is a snapshot of `refs/assets/skill/<name>` as it existed at the 42nd publish run. when the team is ready for a stable release, a separate manual workflow promotes a specific commit to a bare semver tag (`v1.0.0`).

automatic immutable dev tags plus manual prod tags gives consumers a stable spec form for production (`^1.0.0`) and a sharp opt-in handle for early access (`v0.1.0-dev.42`).

## CI publish workflow

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

four CI gotchas worth pinning in memory, all of which I hit setting this up:

1. pin git-skill via `@v<X.Y.Z>`, not `@latest`. the Go module proxy can lag tens of minutes behind newly cut tags.
2. fetch `refs/assets/*` explicitly before any `git skill commit`. `actions/checkout` does not pull custom refs.
3. use `git asset push` for multi-kind producers. `git skill push` is skill-only.
4. `--kind` flag goes before positional args: `git skill tag --kind agent foo v0.1.0`, not `git skill tag foo --kind agent v0.1.0`.

working reference: [`niradler/git-skill-demos/.github/workflows/publish.yml`](https://github.com/niradler/git-skill-demos/blob/main/.github/workflows/publish.yml).

## Promoting dev to prod

a separate manual workflow turns a dev tag into a prod release. inputs are `skill` (name) and `version` (bare semver, no `-dev`). the workflow validates inputs, runs structure-tier evals against the canonical tree (no API key needed), tags the canonical commit with `v<version>`, and pushes the new tag.

reference: [`promote.yml`](https://github.com/niradler/git-skill-demos/blob/main/.github/workflows/promote.yml). trigger with `gh workflow run promote.yml -f skill=code-review -f version=1.0.0`.

## Evals

skills break in two ways.

**structure drift.** a typo in the frontmatter, a missing required section, a broken pointer to an eval prompt. caught by deterministic file checks, no model call needed. cheap enough to run on every push.

**behavior drift.** the skill still parses fine but the model's outputs degrade. needs real model runs against a fixed prompt set, graded against assertions.

for the behavior tier, [microsoft/waza](https://github.com/microsoft/waza) is a Go CLI for evaluating agent skills. it scaffolds eval suites with prompts, fixtures, and graders (`text`, `json_schema`, `prompt`-as-judge, `behavior`, `tool_calls`, `tool_constraint`, `action_sequence`, `skill_invocation`, inline `code`), runs them against a model, and reports pass rates.

upstream waza targets hosted API executors. that works for raw-prompt benchmarks but misses the case where the thing under test is an agent prompt: a `SKILL.md` only meaningful inside an agent loop with tools, file access, and skill discovery. the fork [niradler/waza](https://github.com/niradler/waza) adds an `anthropic-cli` executor that shells out to the Claude CLI ([PR #1](https://github.com/niradler/waza/pull/1)):

```bash
claude --print --bare \
       --output-format stream-json --verbose \
       --permission-mode bypassPermissions \
       --add-dir <workspace> \
       --model <model-id>
```

selected via `engine: anthropic-cli` in `.waza.yaml`. per-task workspaces seeded from `inputs.files:` are mounted via `--add-dir` so the candidate agent can `Read` and `Write` real files. the executor parses the CLI's `stream-json --verbose` output, pairs `tool_use` and `tool_result` events by id, and feeds them to graders that depend on tool and skill telemetry. every grader type works end-to-end against the CLI executor, not just hosted-API ones.

a CI workflow that runs evals against a skill on every PR:

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

a suite is YAML with prompts, fixtures, graders, and a passing score. with `tasks: [tasks/*.yaml]` you keep large multi-file suites tidy without enumerating every file. a status flip from pass to fail blocks the PR. a score drop within tolerance surfaces as a warning.

## Customizing where assets materialize

four layers sit on top of the built-in runtime registry. lowest precedence first:

1. `~/.config/git-skill/runtimes.yaml`: machine-wide defaults.
2. `<repo>/.git-skill/runtimes.yaml`: repo-wide policy, committed alongside `assets.json`.
3. `git-skill.yaml` in the asset tree: author-declared overrides that ship with the asset.
4. `--target runtime=path` on `git skill add`: per-asset override pinned in the lock entry.

send `claude` skills to an alternate path repo-wide:

```yaml
# .git-skill/runtimes.yaml
runtimes:
  claude:
    skill:
      to: .ai/claude/skills/<name>/
```

extend to a runtime not in the built-in registry:

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

git-skill shells out to plain `git fetch` and `git push`. whatever credentials work for `git clone` work here. `gh auth login` once, then `git skill add ... --from https://github.com/acme/private-skills` works the same as for a public repo.

## Inspecting without the CLI

assets are plain git refs. any git-aware tool reads them natively:

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

## Cross-platform notes

**path separators in `assets.json`.** v0.2.0 on Windows writes OS-native separators (`skills\\code-review`) into the `canonical` field. normalize to forward slashes before committing so lock files written on one OS are byte-identical to lock files written on any other.

**symlinks on Windows.** require developer mode or an admin shell. without those privileges the materializer falls back to a plain copy.

**line endings.** `core.autocrlf=true` on Windows will rewrite line endings in materialized files. if you care about byte-stable runtime trees across the team, add `*.md text eol=lf` to `.gitattributes` in the consumer repo.

## End-to-end reference

two repos that exercise everything above:

**producer:** [`niradler/git-skill-demos`](https://github.com/niradler/git-skill-demos) covers six skills plus one agent, CI publish and promote workflows, eval format, and the `running-skill-evals` skill.

**consumer:** [`niradler/git-skill-consumer-demo`](https://github.com/niradler/git-skill-consumer-demo) walks four PRs through install, upgrade, multi-skill, and rollback against the producer above. see [`docs/demo-flows.md`](https://github.com/niradler/git-skill-consumer-demo/blob/main/docs/demo-flows.md).

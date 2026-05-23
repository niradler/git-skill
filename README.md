# git-skill

[![CI](https://github.com/niradler/git-skill/actions/workflows/ci.yml/badge.svg)](https://github.com/niradler/git-skill/actions/workflows/ci.yml)

Git-native versioning for AI agent assets — **skills** and **agents** — using git's own object model.

Inspired by [Building on Git's Primitives](https://remenos.codes/building-on-gits-primitives/) and [git-native-issue](https://github.com/remenoscodes/git-native-issue).

## Why

AI agent assets — instruction bundles, sub-agents, prompt templates — are proliferating fast. Every team customizes them, every platform ships its own set, and there's no versioning, no diffing, no collaboration model. You drop a folder somewhere and hope for the best.

Git already solved this for code. Assets are directories of text files. The infrastructure is already there.

## How it works

Assets are stored as git tree objects under `refs/assets/<kind>/<name>`. Each change is a commit. Versions are tagged refs under `refs/asset-tags/<kind>/<name>/v<semver>`. Sync is push/fetch. The entire "package manager" is git.

```
refs/assets/skill/code-review              → latest commit
refs/asset-tags/skill/code-review/v1.0.0   → tagged release
refs/assets/agent/security-auditor         → latest commit
refs/asset-tags/agent/security-auditor/v0.3.0
```

Two kinds today:

| Kind  | Marker file | Materializes as              |
|-------|-------------|------------------------------|
| skill | `SKILL.md`  | full directory tree          |
| agent | `AGENT.md`  | single marker file at target |

No database. No server. No registry API. The plumbing is `hash-object`, `mktree`, `commit-tree`, `update-ref`. Reading the source teaches you how git works.

## Install

```bash
# Easiest:
go install github.com/niradler/git-skill/cmd/git-skill@latest

# Or from source:
git clone https://github.com/niradler/git-skill
cd git-skill
go build -o git-skill ./cmd/git-skill
sudo mv git-skill /usr/local/bin/

# git auto-discovers it as a subcommand:
git skill list
```

The same binary serves three roles based on its invocation name:

| Binary name | Default kind | Notes |
|-------------|--------------|-------|
| `git-skill` | `skill`      | Skill-focused commands |
| `git-agent` | `agent`      | Agent-focused commands |
| `git-asset` | (none)       | Kind-agnostic; `--kind` flag required for ambiguous calls |

Symlink or copy the same `git-skill` binary to `git-agent` / `git-asset` to enable all three.

Requires Go 1.22+ and git 2.x. Tested on Linux, macOS, and Windows. Windows symlinks need developer mode or admin; otherwise the tool falls back to junction or copy.

## Quickstart — producer

Track and publish a skill from your repo.

```bash
# 1. Scaffold assets.json + .gitignore entries
git skill init

# 2. Author a skill under skills/<name>/ with a SKILL.md
mkdir -p skills/code-review
$EDITOR skills/code-review/SKILL.md

# 3. Snapshot it into refs/assets/skill/code-review
git skill commit code-review --path skills/code-review -m "initial cut"

# 4. Tag a release
git skill tag code-review 1.0.0

# 5. Push refs and tags to a remote
git skill push origin
```

Authoring an **agent** is the same flow with `git-agent` and an `AGENT.md` marker.

## Quickstart — consumer

Install someone else's asset into your repo.

```bash
# 1. Scaffold consumer state
git skill init

# 2. Add a skill from a remote, pinning to a semver spec
git skill add acme/code-review@^1.0.0 \
    --from https://github.com/acme/skills

# 3. Materialize all assets in assets.json into runtime paths
git skill install
```

`assets.json` now records the resolved commit. Commit it alongside your code — anyone who clones your repo can run `git skill install` and get the exact same trees back.

To upgrade later:

```bash
git skill update code-review     # re-resolve within the existing spec
git skill remove code-review     # drop it
git skill discover https://github.com/acme/skills  # list what's available
```

## assets.json

A single file at the repo root. Carries both **intent** (what you asked for) and **resolution** (what you got):

```json
{
  "version": 1,
  "assets": {
    "skill": {
      "code-review": {
        "remote": "https://github.com/acme/skills",
        "spec": "^1.0.0",
        "version": "1.2.0",
        "commit": "9f3c1a…",
        "canonical": "skills/code-review",
        "runtimes": { "claude": {} }
      }
    },
    "agent": { … }
  }
}
```

- Check it into git.
- `install` reads `commit` and restores the working tree verbatim.
- `update` re-resolves `spec` and rewrites `version` + `commit`.

## Runtimes

`runtimes` is a map keyed by runtime name; each entry may carry per-asset overrides (`from`, `to`). Empty `{}` means "use the registry defaults". Built-in targets:

| Runtime  | Skill target                | Agent source → target                          |
|----------|-----------------------------|------------------------------------------------|
| claude   | `.claude/skills/<name>/`    | `AGENT.md` → `.claude/agents/<name>.md`        |
| codex    | `.agents/skills/<name>/`    | `agent.toml` → `.codex/agents/<name>.toml`     |
| cursor   | `.cursor/rules/<name>/`     | _(not supported)_                              |
| opencode | `.agents/skills/<name>/`    | _(not supported)_                              |

A trailing `/` on the target means directory materialization (the whole canonical tree fans out); no trailing slash means a single-file mapping. Authors of agents that target codex ship an `agent.toml` alongside the `AGENT.md` (no in-tool format conversion).

Override the target on the fly with `--target <runtime>=<path>` (repeatable). The override is persisted into `assets.json`:

```bash
git skill add code-review --from … --runtime claude --target claude=.custom/skills/<name>/
```

Skills materialize as full trees; agent fan-outs depend on the runtime (Claude: `AGENT.md`; Codex: `agent.toml`). On Unix the tool prefers relative symlinks back to the canonical tree. On Windows it falls back to junctions or full copies.

### Asset manifest (`git-skill.yaml`)

Optional. Lives at the root of the canonical tree and lets the asset author declare per-runtime `from`/`to` overrides — or extend `git-skill` to runtimes that aren't in the built-in registry — without consumers needing to set `--target`.

```yaml
# git-skill.yaml at the root of the asset tree
kind: agent   # optional author hint
runtimes:
  claude:
    from: prompts/reviewer.md
    to: .claude/agents/<name>.md
  custom-tool:                  # not in the built-in registry
    from: src/<name>.txt
    to: .custom-tool/<name>/
```

`<name>` placeholders are substituted in both `from` and `to`. See [Resolution precedence](#resolution-precedence) below for where the manifest sits in the full chain.

### User/project runtimes config

Two optional config files extend or override the built-in registry without touching individual assets:

- **`~/.config/git-skill/runtimes.yaml`** — user-global. Override with `$GIT_SKILL_USER_CONFIG` (intended for tests).
- **`<repo>/.git-skill/runtimes.yaml`** — project-local, committable.

```yaml
# .git-skill/runtimes.yaml
runtimes:
  myfuture:
    skill:
      to: .myfuture/skills/<name>/
    agent:
      from: AGENT.md
      to: .myfuture/agents/<name>.md
  claude:                          # also fine: override a built-in
    skill:
      to: .alt/claude/<name>/
```

### Resolution precedence

Full precedence chain (low → high): built-in registry < user config < project config < asset manifest < lock entry override.

Canonical trees live at `skills/<name>/` and `agents/<name>/` by default (configurable via `config.skillsRoot` / `config.agentsRoot` in `assets.json`). PRs to add new runtimes are welcome; see [`internal/runtimes/registry.go`](./internal/runtimes/registry.go).

## Commands

```
init                  scaffold assets.json + .gitignore block
commit <name>         snapshot canonical tree into refs/assets/<kind>/<name>
tag <name> <semver>   tag a commit as v<semver>
push <remote>         push refs/assets/<kind>/* and refs/asset-tags/<kind>/*
fetch <remote>        fetch the mirror

list                  enumerate local refs/assets with latest tag
log <name>            history of refs/assets/<kind>/<name>
diff <name> A B       diff two tagged versions
show <name>           metadata + tag list

add <ns>/<name>[@spec] --from <url>   onboard remote asset → assets.json
update [<name>]       re-resolve spec → new version+commit, materialize
remove <name>         drop from assets.json + delete canonical + runtime paths
install               materialize every asset in assets.json
discover <url>        enumerate remote assets
```

Run with `--help` for full flag listings.

`install` and `remove` manage the canonical path and configured runtime targets in `assets.json`.
Do not point runtime targets at user-owned files or directories; those paths are treated as git-skill-managed materialization outputs.

## Asset-Kind trailer

Every commit written by `commit` includes a trailer identifying the kind:

```
Asset-Kind: skill
```

This makes the commit self-describing — any indexer or downstream tool can recover the kind from the commit alone without consulting refs.

## Design Principles

1. **Don't build what git already provides.** Versioning, sync, merge, identity, history — it's all there.
2. **Plumbing, not porcelain.** The tool composes `hash-object`, `mktree`, `commit-tree`, `update-ref`. Reading the source teaches you how git works.
3. **The spec is the contribution.** The CLI is a reference implementation. The format is what matters.

## Documentation

- [SKILL-FORMAT.md](./SKILL-FORMAT.md) — storage format specification
- [ARCHITECTURE.md](./ARCHITECTURE.md) — CLI vs platform separation
- [CONTRIBUTING.md](./CONTRIBUTING.md) — how to contribute
- [CHANGELOG.md](./CHANGELOG.md) — release notes

## License

MIT

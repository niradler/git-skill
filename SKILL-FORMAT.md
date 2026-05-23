# SKILL-FORMAT.md

> A specification for storing AI agent **assets** — skills and agents — as git-native objects.
> Version: 0.2.0-draft

## Overview

This document defines a format for storing, versioning, and distributing AI agent assets using git's object model. An **asset** is a directory of text files that guides an AI agent's behavior; the two kinds are `skill` (a full instruction bundle materialized as a directory) and `agent` (a single marker file materialized at a runtime target).

The format requires no external database, server, or registry. Assets are git objects, stored in git, synced by git.

## Concepts

| Concept            | Git primitive                                       | Notes                                                      |
|--------------------|-----------------------------------------------------|------------------------------------------------------------|
| Asset identity     | `refs/assets/<kind>/<name>`                         | One ref per asset, points to latest commit                 |
| Asset state        | Tree pointed to by HEAD commit                      | Full snapshot of the asset directory                       |
| History            | Commit chain under the ref                          | Append-only, content-addressed                             |
| Version release    | `refs/asset-tags/<kind>/<name>/v<semver>`           | Named pointer to a commit                                  |
| Kind metadata      | `Asset-Kind: <kind>` git trailer on every commit    | Makes the commit self-describing without ref context       |
| Consumer state     | `assets.json` committed at repo root                | Intent (`spec`) + resolution (`version`, `commit`)         |
| Sync               | `git fetch` / `git push`                            | Standard transport, refspec-based                          |

## Kinds

Two kinds are defined:

| Kind  | Marker file | Materializes as              |
|-------|-------------|------------------------------|
| skill | `SKILL.md`  | full directory tree at the runtime target |
| agent | `AGENT.md`  | single file at the runtime target (per runtime: `AGENT.md` on Claude, `agent.toml` on Codex, etc.) |

The kind of a given asset is resolved through a 4-tier discriminator (highest tier wins):

1. **lock entry override** — `kind:` in `assets.json` for an installed asset.
2. **commit trailer** — `Asset-Kind: <kind>` on the most recent commit of the ref.
3. **frontmatter** — `kind:` field in the marker file's YAML frontmatter.
4. **filename** — `SKILL.md` ⇒ `skill`; `AGENT.md` ⇒ `agent`.

Conflicts produce a warning but do not fail the operation; the higher tier wins.

## Directory Structure

A canonical asset tree contains a marker file at its root. Skills typically include reference material and scripts; agents typically ship a single instruction file plus runtime adapters.

```
<asset-name>/
  SKILL.md or AGENT.md       # Required marker. Frontmatter + body.
  git-skill.yaml             # Optional asset manifest (see "Asset manifest").
  LICENSE.txt                # Optional.
  reference/*.md             # Optional.
  scripts/*                  # Optional.
  agent.toml                 # Optional (Codex agents).
  .assetignore               # Optional gitignore-style filter applied during copy materialization.
```

## Marker file frontmatter

The marker file (`SKILL.md` or `AGENT.md`) MUST begin with YAML frontmatter delimited by `---`:

```yaml
---
name: frontend-design
description: Create production-grade frontend interfaces…
version: 1.2.0
kind: skill        # optional; resolved through the 4-tier discriminator
license: MIT
---
```

### Required fields

- `name` — Unique asset identifier. Lowercase, hyphens allowed; may include a namespace (`acme/code-review`). Must match the ref name.
- `description` — Human-readable description.

### Optional fields

- `version` — SemVer string. Also recorded in the commit trailer on each snapshot.
- `kind` — `skill` or `agent`. Used to break ties in the 4-tier discriminator.
- `license` — SPDX identifier or reference to LICENSE.txt.

## Ref naming

### Asset refs

```
refs/assets/<kind>/<name>
```

Names MAY contain namespace prefixes separated by `/`:

```
refs/assets/skill/frontend-design          # shared skill
refs/assets/skill/nir/boxy                 # user-namespaced skill
refs/assets/agent/acme-corp/security-auditor  # org-namespaced agent
```

### Version tags

```
refs/asset-tags/<kind>/<name>/v<semver>
```

Examples:

```
refs/asset-tags/skill/frontend-design/v1.0.0
refs/asset-tags/skill/frontend-design/v1.1.0
refs/asset-tags/agent/security-auditor/v0.3.0
```

Version tags are lightweight refs pointing directly to the asset commit.

## Commit format

Every commit on an asset ref SHOULD include an `Asset-Kind` trailer naming the kind:

```
Add error handling guidance

Asset-Kind: skill
Skill-Version: 1.2.0
```

The trailer is preserved verbatim across rewrites. A reader can recover the kind from the commit alone, without consulting the ref. Implementations MUST NOT reject commits that lack the trailer (older history is valid) and MUST preserve unknown trailers.

## Consumer state: `assets.json`

Consumers record onboarded assets in a single committed file at the repo root:

```json
{
  "version": 1,
  "assets": {
    "skill": {
      "acme/code-review": {
        "remote": "https://github.com/acme/skills",
        "spec": "^1.0.0",
        "version": "1.2.0",
        "commit": "9f3c1a…",
        "canonical": "skills/acme/code-review",
        "runtimes": { "claude": {} }
      }
    },
    "agent": { "...": "..." }
  }
}
```

- `spec` records intent (semver range, exact tag, or commit SHA).
- `version` + `commit` record resolution (what `install` will restore).
- `runtimes` is a map from runtime name to per-asset override (`from`, `to`). Empty `{}` means "use the registry default". The legacy `["claude", "codex"]` list form is explicitly rejected.

## Asset manifest: `git-skill.yaml`

Optional. Lives at the root of the canonical asset tree. Asset authors declare per-runtime `from`/`to` overrides without forcing consumers to set `--target`, or extend the tool to runtimes not in the built-in registry.

```yaml
# git-skill.yaml
kind: agent     # optional author hint, fed into the 4-tier discriminator
runtimes:
  claude:
    from: prompts/reviewer.md
    to: .claude/agents/<name>.md
  custom-tool:               # not in the built-in registry
    from: src/<name>.txt
    to: .custom-tool/<name>/
```

`<name>` placeholders are substituted in both `from` and `to`. A trailing `/` on `to` means directory fanout; no trailing slash means a single-file mapping.

## Runtimes config: `runtimes.yaml`

Two optional config files extend or override the built-in runtime registry without touching individual assets:

- `~/.config/git-skill/runtimes.yaml` — user-global.
- `<repo>/.git-skill/runtimes.yaml` — project-local, committable.

Schema:

```yaml
runtimes:
  myfuture:
    skill:
      to: .myfuture/skills/<name>/
    agent:
      from: AGENT.md
      to: .myfuture/agents/<name>.md
```

Full precedence chain (low → high): built-in registry < user config < project config < asset manifest < lock entry override.

## Sync

Assets sync using standard git transport. Implementations push and fetch the asset and tag namespaces:

```bash
git push <remote> 'refs/assets/*:refs/assets/*'
git fetch <remote> '+refs/assets/*:refs/assets/*'

git push <remote> 'refs/asset-tags/*:refs/asset-tags/*'
git fetch <remote> '+refs/asset-tags/*:refs/asset-tags/*'
```

No custom protocol is required. Any git remote (SSH, HTTPS, local path) works.

## Compatibility

Asset refs live under `refs/assets/` and `refs/asset-tags/`, which do not conflict with standard git refs (`refs/heads/`, `refs/tags/`, `refs/remotes/`).

## Legacy refs (pre-0.2)

Prior drafts used `refs/skills/<name>` and `refs/skill-tags/<name>/v<semver>` and only supported skills. The 0.2 format introduces the `<kind>` segment and adds agents. Implementations MAY ship a one-shot migration for legacy refs but are not required to read them at runtime.

## Example session

```bash
# Author a skill.
git skill init
mkdir -p skills/code-review
$EDITOR skills/code-review/SKILL.md

git skill commit code-review --path skills/code-review -m "initial"
git skill tag code-review 1.0.0
git skill push origin

# Consume it from another repo.
git skill init
git skill add acme/code-review@^1.0.0 \
    --from https://github.com/acme/skills
git skill install
# → fetches the pinned commit and materializes into the configured runtime paths.
```

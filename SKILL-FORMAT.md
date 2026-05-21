# SKILL-FORMAT.md

> A specification for storing AI agent skills as git-native objects.
> Version: 0.1.0-draft

## Overview

This document defines a format for storing, versioning, and distributing AI agent
skills using git's object model. A skill is a directory of text files that guides
an AI agent's behavior for a specific task domain.

The format requires no external database, server, or registry. Skills are git objects,
stored in git, synced by git.

## Concepts

| Concept             | Git primitive                     | Notes                                      |
|---------------------|-----------------------------------|--------------------------------------------|
| Skill identity      | `refs/skills/<name>`              | One ref per skill, points to latest commit  |
| Skill state         | Tree pointed to by HEAD commit    | Full snapshot of the skill directory        |
| History             | Commit chain under the ref        | Append-only, content-addressed              |
| Version release     | `refs/skill-tags/<name>/<version>`| Named pointer to a commit                  |
| Metadata            | Git trailers on commits           | Parseable with `git interpret-trailers`     |
| Sync                | `git fetch` / `git push`          | Standard transport, refspec-based           |

## Directory Structure

A skill is a directory containing at minimum a `SKILL.md` file:

```
<skill-name>/
  SKILL.md              # Required. Frontmatter + instructions.
  LICENSE.txt           # Optional.
  reference/            # Optional. Supporting documents.
    *.md
  scripts/              # Optional. Code bundled with the skill.
    *.py | *.sh | *.js
```

## SKILL.md Frontmatter

The `SKILL.md` file MUST begin with YAML frontmatter delimited by `---`:

```yaml
---
name: frontend-design
description: Create production-grade frontend interfaces...
version: 1.2.0
license: MIT
---
```

### Required fields

- `name` — Unique skill identifier. Lowercase, hyphens allowed. Must match the
  directory name and the ref name.
- `description` — Human-readable description.

### Optional fields

- `version` — SemVer string. Also recorded in the commit trailer on each snapshot.
- `license` — SPDX identifier or reference to LICENSE.txt.

## Ref Naming

### Skill refs

```
refs/skills/<name>
```

Names MAY contain namespace prefixes separated by `/`:

```
refs/skills/frontend-design          # shared skill
refs/skills/nir/boxy                 # user-namespaced skill
refs/skills/acme-corp/onboarding     # org-namespaced skill
```

### Version tags

```
refs/skill-tags/<name>/v<semver>
```

Examples:
```
refs/skill-tags/frontend-design/v1.0.0
refs/skill-tags/frontend-design/v1.1.0
```

Version tags are lightweight refs pointing directly to the skill commit.

## Commit Format

Every commit on a skill ref MAY include a `Skill-Version` trailer:

```
Add error handling guidance

Skill-Version: 1.2.0
```

Implementations MUST NOT reject commits that lack the trailer, and MUST preserve
unknown trailers.

## Sync

Skills sync using standard git transport:

```bash
# Push all skills to a remote
git push <remote> 'refs/skills/*:refs/skills/*'

# Fetch all skills from a remote
git fetch <remote> '+refs/skills/*:refs/skills/*'

# Push/fetch version tags
git push <remote> 'refs/skill-tags/*:refs/skill-tags/*'
git fetch <remote> '+refs/skill-tags/*:refs/skill-tags/*'
```

No custom protocol is required. Any git remote (SSH, HTTPS, local path) works.

## Compatibility

Skill refs live under `refs/skills/` and `refs/skill-tags/`, which do not conflict
with standard git refs (`refs/heads/`, `refs/tags/`, `refs/remotes/`).

## Example Session

```bash
git skill init frontend-design "Production-grade frontend interfaces"

vim .skills/frontend-design/SKILL.md

git skill commit frontend-design -m "Add typography guidelines"

git skill tag frontend-design 1.0.0

git skill push origin

# On another machine:
git skill fetch origin
git skill install frontend-design@v1.0.0 /path/to/agent/skills/
```

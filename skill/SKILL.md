---
name: git-skill
description: Use git-skill whenever the user is managing, versioning, publishing, sharing, or consuming AI agent skills, especially when they mention .skills/, SKILL.md, skill.lock, "version a skill", "publish a skill", "install a skill", "sync skills", refs/skills/, or want to share skills via git/GitHub. Trigger even when the user doesn't explicitly name the tool - anything that smells like skill versioning, distribution, or reproducible installs across machines falls in scope.
version: 0.1.0
license: MIT
---

# git-skill

## What it is

git-skill is a CLI that treats AI agent skills the way git treats source code. Skills are directories of text files. git-skill stores each one as a versioned, content-addressed git object under a dedicated ref namespace, then leans on standard `git push` / `git fetch` for distribution. No registry, no API server, no separate database - just git.

It is named `git-skill` so `git skill <cmd>` works out of the box (git auto-discovers subcommands on PATH).

## Why it exists

Skills proliferate fast. Every team customizes Claude/Cursor/etc. with shared instructions, but the tooling around them is missing the basics: history, diffing, version pins, reproducible installs, audit trails. People drop folders in `.claude/skills/` and hope nothing drifts. git-skill fixes this by reusing the infrastructure git already provides for the same problem space (versioned text-file collections).

Understanding the *why* matters because most of the design choices follow from it:

- **Skills as refs, not branches.** Skills live under `refs/skills/<name>` so they don't pollute `refs/heads/` and can coexist with any existing repo's branches/tags.
- **Tagged releases.** `refs/skill-tags/<name>/vX.Y.Z` gives immutable version pins, separate from a repo's own release tags.
- **Lockfile for reproducibility.** `skill.lock` pins exact commit SHAs so two machines can guarantee they're running the identical skill bytes, even if remote tags get moved.
- **Symlinks to agent dirs.** A skill is installed once into a canonical path (`.skills/<name>`) and symlinked into each agent's expected location. Update once, every agent picks it up.

## When to invoke this skill

Reach for git-skill anytime any of these are true:

- The user wants to version-control or release AI skills (Claude, Cursor, OpenAI agents, custom).
- The repo already has a `.skills/` directory, a `skill.lock` file, or a `SKILL.md` file with YAML frontmatter.
- The user talks about installing a shared skill, syncing skills across machines, or pinning a skill version.
- The user wants to publish skills to GitHub or fetch them from a git remote.
- The user has an existing skill folder and wants to start tracking it formally.
- The user asks about diffing skill versions, rolling back a skill, or auditing skill changes.

Heuristic: if the conversation is about *managing* skills (as opposed to *writing* skill content), git-skill is in scope.

### Do not invoke for

- Pure SKILL.md content drafting with no versioning intent. Helping the user word a system prompt does not require git-skill unless they then ask to version, publish, install, sync, or pin it.
- General questions about what skills are or how AI agents use them.
- Editing `.skills/<name>/` files directly to fix a typo. That is just a file edit; only reach for `git skill commit` once the user wants to snapshot the change.

## Before running commands

A short pre-flight saves a lot of pain later:

1. **Confirm a git repo.** `git rev-parse --is-inside-work-tree` should print `true`. git-skill stores skills under custom refs inside an existing repo - there is no "skill repo" mode.
2. **Confirm the tool is installed and on PATH.** `git skill version` should print the version. If it errors with "unknown command", the user has not installed git-skill (`go install github.com/niradler/git-skill/cmd/git-skill@latest`).
3. **Figure out which workflow applies.** Author (`init`, `commit`, `tag`, `push`) vs consumer (`get`, `sync`, `install`). The same user can be both, but for a given task it is one or the other.
4. **Look at existing state.** `git skill list` shows tracked skills. `cat skill.lock` (if present) shows pinned versions. These tell you whether you are bootstrapping or working with an established setup.

## Core mental model

| Concept | Git primitive | What it gives you |
|---|---|---|
| Skill identity | `refs/skills/<name>` | One ref per skill, advances on every commit |
| Skill state | The tree the ref's commit points to | Full snapshot of the skill directory |
| History | Commit chain under the ref | Append-only audit trail |
| Version release | `refs/skill-tags/<name>/v<semver>` | Immutable named pointer to a commit |
| Sync | `git push` / `git fetch` with refspecs | Works over SSH, HTTPS, local path, any git remote |
| Reproducible install | `skill.lock` (JSON, commit SHA-pinned) | Same bytes on every machine |

By convention, skills live at `.skills/<name>/` in the repo and are symlinked into agent-specific paths (`.claude/skills/<name>`, `.agents/skills/<name>`). That gives every agent a stable path while keeping a single source of truth.

## Author workflow

The author owns a skill and ships new versions of it.

```bash
# Create a new skill - scaffolds SKILL.md and makes the first commit.
git skill init my-skill "A skill that does cool things"

# Edit the scaffolded SKILL.md.
$EDITOR .skills/my-skill/SKILL.md

# Snapshot the current state. Each commit lives on refs/skills/my-skill.
git skill commit my-skill -m "Add error handling guidance"

# Review history and changes.
git skill log my-skill
git skill diff my-skill                # latest vs previous
git skill diff my-skill v1.0.0 HEAD    # two specific revisions

# Tag a release. Creates refs/skill-tags/my-skill/v1.0.0.
git skill tag my-skill 1.0.0

# Publish skills AND tags in one shot. (Plain `git push` would skip the tags.)
git skill push origin

# Already have a skill directory you want to start tracking?
git skill track my-existing-skill ./path/to/skill/dir
```

## Consumer workflow

The consumer wants to install someone else's skill and keep it pinned.

```bash
# Fetch from a remote, install to a target dir, and pin the commit SHA in skill.lock.
git skill get https://github.com/org/repo skill-name@v1.0.0 .claude/skills/

# Reinstall all pinned skills from skill.lock (new machine, fresh checkout, CI).
git skill sync

# Just install a skill that's already in your local object store.
git skill install skill-name@v1.0.0 .claude/skills/

# Fetch refs without installing (rare - useful for inspection/diff).
git skill fetch origin

# Limit which agent symlinks get created.
git skill get <remote> <skill>@v1.0.0 .claude/skills/ --agent claude

# Editable install for skill authors: skips writing a lockfile entry so the next
# `git skill sync` won't clobber in-place edits. The skill is still installed.
git skill get <remote> <skill> .claude/skills/ --dev
```

## Inspecting skills

```bash
git skill list                          # all tracked skills
git skill show my-skill                 # metadata + version list + latest commit
git skill log my-skill -n 5             # last 5 commits with Skill-Version trailers
git skill diff my-skill                 # last change
git skill diff my-skill v1.0.0 v2.0.0   # diff between tagged versions
```

`git skill show` is the right starting point if you have a skill name but don't remember what version is current or what changed.

## skill.lock

`skill.lock` is the reproducibility contract. It pins exact commit SHAs and records where each skill came from. It is JSON, lives at the repo root, and should be committed:

```json
{
  "lockfileVersion": 2,
  "skills": {
    "frontend-design": {
      "remote": "https://github.com/org/skills.git",
      "version": "v1.0.0",
      "commit": "abc123def456...",
      "canonical": ".skills/frontend-design",
      "agents": {
        "claude": ".claude/skills/frontend-design",
        "cursor": ".agents/skills/frontend-design"
      },
      "dev": false
    }
  }
}
```

`git skill sync` reads this file and reinstalls every entry at the pinned `commit` (not the `version` - tags can move; commit SHAs cannot). This is why the lockfile matters: even if someone moves `v1.0.0` to a new commit upstream, every machine that ran `git skill sync` against the same lock gets identical bytes.

The reinstall is atomic: the new tree is materialized in a sibling directory and then renamed into place. Files that were deleted upstream are removed from the local install - this is what makes "same bytes on every machine" actually true.

`dev: true` marks an entry as locally-mutable. `git skill get --dev` does not write a lockfile entry at all (so the consumer is also the author and intends to edit in place). `sync` ignores any `dev: true` entry it finds. Switch to a normal install (`git skill get` without `--dev`) when you're ready to pin a version.

## SKILL.md format

Every skill directory must contain a `SKILL.md` with YAML frontmatter:

```markdown
---
name: my-skill
description: What this skill does in one sentence.
version: 1.0.0
license: MIT
---

# My Skill

Instruction content for the AI agent goes here.
```

Required: `name` (must match the ref/directory name), `description`. Optional: `version` (semver, recorded in commit trailers as `Skill-Version:`), `license` (SPDX identifier).

A skill can also include supporting files:

```
my-skill/
  SKILL.md              # required
  references/           # supporting docs the skill links to
  scripts/              # executable helpers bundled with the skill
  LICENSE.txt           # optional, if license differs from repo
```

All files become part of the git tree. `git skill diff` diffs everything.

## Ref naming

Two namespaces, isolated from normal git refs:

```
refs/skills/<name>                    # latest commit
refs/skill-tags/<name>/v<semver>      # tagged version
```

Namespacing is supported in the skill name itself:

```
refs/skills/acme-corp/onboarding
refs/skill-tags/acme-corp/onboarding/v1.0.0
```

Use this for org-owned or user-owned skills.

## Safety: consuming skills from remotes you do not control

Skill content is code-adjacent: it runs through an AI agent and can include shell commands, references to other tools, and bundled scripts. Treat third-party skills with the same care you treat third-party dependencies.

- **Pin to commit SHAs via `skill.lock`.** Never rely on a moving tag alone for trusted skills. Once `git skill get name@v1.0.0` writes the lock, the SHA is what guarantees reproducibility - tags can be force-moved upstream.
- **Review `skill.lock` diffs.** A diff that changes `commit:` for an existing skill means the upstream content moved. Run `git skill diff <name> <old-sha> <new-sha>` before merging.
- **Inspect fetched content before installing.** `git skill fetch <remote>` populates the local store without installing. Then `git cat-file -p refs/skills/<name>:SKILL.md` lets you read the body without writing files anywhere.
- **Be wary of bundled `scripts/` directories.** A skill can ship executables. If you do not need them, install with `--agent claude` (or whatever single agent you use) to keep blast radius small, and audit `scripts/` before running anything.
- **Do not run `git skill sync` on an untrusted PR** that touches `skill.lock`. The lockfile points at remotes and commits; a malicious PR can swap a benign skill for a hostile one.

## Common pitfalls

- **Tagging without pushing.** `git skill tag` only updates a local ref. Run `git skill push origin` afterwards - it pushes both `refs/skills/*` and `refs/skill-tags/*`. Plain `git push` would skip them.
- **Forgetting to commit `skill.lock`.** Without the lockfile in version control, `git skill sync` can't run for the rest of the team. Always treat `skill.lock` like `package-lock.json` or `go.sum`.
- **Version field vs. tag.** The `version:` line in `SKILL.md` frontmatter is metadata (it becomes a commit trailer). The *actual* versioned ref is only created by `git skill tag <name> <ver>`. They are not auto-synced - bump both.
- **Tracking a directory without SKILL.md.** `git skill track` will refuse. Scaffold it (`git skill init`) or create a SKILL.md by hand first.
- **Modifying installed skill files in place.** Edits to `.claude/skills/<name>/...` mostly land in the canonical `.skills/<name>` (since they're symlinks), but for non-dev installs the right workflow is to edit in the author repo and re-publish.

## Detecting whether to use git-skill in a repo

When working in an unfamiliar codebase, these signals indicate git-skill is (or should be) in play:

- A `skill.lock` file at the repo root → consumer setup; use `git skill sync` / `git skill get`.
- A `.skills/` directory with `SKILL.md` files inside → author setup; use `git skill commit` / `git skill tag` / `git skill push`.
- `refs/skills/*` refs in the repo (check with `git for-each-ref refs/skills/`) → skills are tracked here.
- The CLI `git-skill` binary on PATH → user has it installed; safe to use.

If none of these exist but the user is asking about managing skills, start with `git skill init <name>` to bootstrap.

## References

- [SKILL-FORMAT.md](../SKILL-FORMAT.md) - formal specification of the storage format
- [ARCHITECTURE.md](../ARCHITECTURE.md) - CLI vs. registry-platform separation
- [docs/using-git-skill-with-github.md](../docs/using-git-skill-with-github.md) - end-to-end tutorial
- Source: `cmd/git-skill/main.go` - every command's implementation lives in one file

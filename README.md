# git-skill

[![CI](https://github.com/niradler/git-skill/actions/workflows/ci.yml/badge.svg)](https://github.com/niradler/git-skill/actions/workflows/ci.yml)

Git-native skill versioning. Track, version, diff, and sync AI agent skills using git's own object model.

Inspired by [Building on Git's Primitives](https://remenos.codes/building-on-gits-primitives/) and [git-native-issue](https://github.com/remenoscodes/git-native-issue).

## Why

Skills — the instruction bundles that guide AI agents — are proliferating fast. Every team customizes them, every agent platform ships its own set, and there's no versioning, no diffing, no collaboration model. You drop a folder somewhere and hope for the best.

Git already solved this for code. Skills are directories of text files. The infrastructure is already there.

## How it works

Skills are stored as git tree objects under `refs/skills/<name>`. Each change is a commit. Versions are tagged refs. Sync is push/fetch. The entire "package manager" is git.

```
refs/skills/frontend-design          → latest commit
refs/skill-tags/frontend-design/v1.0 → tagged release
```

No database. No server. No registry API. Three plumbing commands to create a skill, one format string to read its state.

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

Requires Go 1.22+ and git 2.x. Tested on Linux and macOS. Windows works for everything except agent-directory symlinks (which need developer mode or admin privileges).

## Usage

```bash
# Create a new skill
git skill init my-skill "A skill that does cool things"

# Edit it
vim .skills/my-skill/SKILL.md

# Commit a snapshot
git skill commit my-skill -m "Add error handling guidance"

# See history
git skill log my-skill

# Diff versions
git skill diff my-skill

# Tag a release
git skill tag my-skill 1.0.0

# Push to remote
git skill push origin

# Fetch from remote (another machine)
git skill fetch origin

# Install a skill to a target directory
git skill install my-skill@v1.0.0 /path/to/agent/skills/

# List everything
git skill list
```

## Format Spec

The real deliverable is [SKILL-FORMAT.md](./SKILL-FORMAT.md) — a standalone specification for storing skills in git, independent of this tool. If the community adopts the format, skills become portable across platforms.

## Design Principles

1. **Don't build what git already provides.** Versioning, sync, merge, identity, history — it's all there.
2. **Plumbing, not porcelain.** The tool uses `hash-object`, `mktree`, `commit-tree`, `update-ref`. Reading the source teaches you how git works.
3. **The spec is the contribution.** The CLI is a reference implementation. The format is what matters.

## Using git-skill as an AI agent skill

This repo ships a ready-to-install skill that teaches AI agents (Claude, Cursor, etc.) how to use git-skill. Track and push it from this repo:

```bash
git skill track git-skill ./skill
git skill tag git-skill 0.1.0
git skill push origin
```

Then consumers can install it:

```bash
git skill get https://github.com/niradler/git-skill git-skill@v0.1.0 .claude/skills/
```

## Documentation

- [SKILL-FORMAT.md](./SKILL-FORMAT.md) — storage format specification
- [ARCHITECTURE.md](./ARCHITECTURE.md) — CLI vs platform separation
- [CONTRIBUTING.md](./CONTRIBUTING.md) — how to contribute
- [docs/using-git-skill-with-github.md](./docs/using-git-skill-with-github.md) — full tutorial

## License

MIT

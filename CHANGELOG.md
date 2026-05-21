# Changelog

All notable changes to git-skill are documented here. Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions follow [SemVer](https://semver.org/).

## [Unreleased]

## [0.1.1] - 2026-05-22

Initial public release.

### Added
- `git skill init` — scaffold a new skill with a starter `SKILL.md`.
- `git skill track` — start tracking an existing skill directory.
- `git skill commit` — snapshot the current state of a skill into `refs/skills/<name>`.
- `git skill log` / `git skill diff` / `git skill show` — inspect history.
- `git skill tag` — create immutable `refs/skill-tags/<name>/vX.Y.Z` releases.
- `git skill push` / `git skill fetch` — sync skill refs and tags with any git remote.
- `git skill install` / `git skill get` — install a skill locally, write `skill.lock`.
- `git skill sync` — reinstall every skill in `skill.lock` at its pinned commit.
- `git skill list` — show all locally-tracked skills.
- `skill.lock` v2 — JSON lockfile pinning skills to exact commit SHAs. Paths stored as repo-relative forward-slash form so a lockfile written on one OS is portable across Windows, macOS, and Linux.
- Agent symlinks created with targets relative to the symlink's parent directory, so they resolve correctly with relative install paths and survive the repo being moved.
- `SKILL-FORMAT.md` — standalone specification of the storage format.
- Bundled AI-agent skill (`skill/SKILL.md`) so coding agents can use the tool natively.

### Known limitations
- Windows symlinks require developer mode or admin privileges (everything else works on Windows).
- The agent symlink list is hard-coded to `claude` and `cursor`; PRs to expand it are welcome.
- `version:` in `SKILL.md` frontmatter is not auto-synced with `git skill tag`.

[Unreleased]: https://github.com/niradler/git-skill/compare/v0.1.1...HEAD
[0.1.1]: https://github.com/niradler/git-skill/releases/tag/v0.1.1

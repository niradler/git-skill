# Changelog

All notable changes to git-skill are documented here. Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions follow [SemVer](https://semver.org/).

## [Unreleased]

### Added
- **Assets format** — unified storage for both **skills** and **agents** under `refs/assets/<kind>/<name>` (was `refs/skills/<name>`).
- **Three CLI personas** via argv[0] dispatch: `git-skill`, `git-agent`, `git-asset` share one binary; the invocation name sets the default kind.
- **`assets.json`** — single committed file replacing `skill.lock`. Carries both intent (`spec`) and resolution (`version`, `commit`) so a fresh clone + `install` restores the exact working tree.
- **`git skill add <ns>/<name>[@spec] --from <url>`** — onboard a remote asset, resolving a semver spec against tags on first add.
- **`git skill update [<name>]`** — re-resolve spec to a new version+commit and re-materialize.
- **`git skill remove <name>`** — drop from `assets.json`, delete canonical + runtime paths.
- **`git skill install`** — vercel-style materialization: read `assets.json`, fetch pinned commit, restore canonical, link/copy into each configured runtime path.
- **`git skill discover <url>`** — enumerate remote assets via `ls-remote`.
- **`Asset-Kind: <kind>`** commit trailer on every snapshot, making commits self-describing.
- **4-tier kind discriminator** — existing ref > `--kind` flag > frontmatter > marker filename > profile default. Conflicts produce a warning.
- **Cross-platform materialization** — Unix relative symlinks; Windows junctions or copy fallback when symlink privilege is unavailable.
- **`.assetignore`** — gitignore-style filter applied during copy materialization (dev-mode symlinks see the full tree).
- **Built-in runtime registry** — `claude`, `cursor`, `codex`, `opencode`. Each entry is now a `{from, to}` mapping; trailing `/` on `to` means directory fanout, no slash means single-file. Adds `codex:agent` default (`agent.toml` → `.codex/agents/<name>.toml`).
- **`--target <runtime>=<path>`** flag on `git skill add` — override the install target for a runtime; the override is persisted into `assets.json`.
- **Dev mode** — `e.Dev == true` preserves local edits to the canonical tree on re-install. Non-dev installs always refresh from the pinned commit.
- **Release E2E suite** (`test/e2e/release/`) — 16 tests covering flows, contracts, platform, errors, and idempotency on a real git repo.

### Changed
- **Refs layout** — `refs/skills/<name>` → `refs/assets/skill/<name>`; `refs/skill-tags/<name>/v…` → `refs/asset-tags/skill/<name>/v…`. Agents live under `refs/assets/agent/<name>` and `refs/asset-tags/agent/<name>/v…`.
- **State file** — `skill.lock` is gone; consumers use `assets.json`.
- **Indexer (SkillHub)** — now indexes both kinds from `refs/assets/<kind>/<name>` and tag refs under `refs/asset-tags/<kind>/<name>/v…`.
- **`assets.json` `runtimes` field** — was `["claude", "codex"]` (list of names); now an object `{"claude": {}, "codex": {"to": ".custom/<name>/"}}`. Each value is a per-runtime override (`from`, `to`); empty `{}` means "use the registry default". The legacy `[]string` form is explicitly rejected with a clear error.

### Removed
- `git skill track` — superseded by the `init` + author-under-`skills/<name>/` or `agents/<name>/` + `commit --path` flow.
- `git skill get` — replaced by `add` + `install`. The standalone `skillhub get` subcommand has also been deleted.
- `git skill sync` — `install` is now idempotent and re-materializes everything in `assets.json`.
- `skill.lock` v2 — replaced by `assets.json`.

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

# Contributing to git-skill

## Setup

```bash
git clone https://github.com/niradler/git-skill
cd git-skill
go build ./...   # verify it builds
go test ./...    # run unit tests
bash test.sh     # run integration tests
```

Requirements: Go 1.22+, git 2.x.

## Running tests

```bash
go test ./...        # unit tests
bash test.sh         # end-to-end integration tests
go vet ./...         # static analysis
```

Unit tests live alongside their packages (`*_test.go`). Integration tests in `test.sh` exercise the full CLI against a real git repository.

The `internal/git` tests call the `git` binary directly and require a git user identity. If you see author/committer errors, run:

```bash
git config --global user.email "you@example.com"
git config --global user.name "Your Name"
```

## Project layout

```
cmd/git-skill/main.go        — entry point; argv[0] profile dispatch
cmd/git-skill/dispatch.go    — Run() routes name → command function
cmd/git-skill/wire.go        — registers each command in the dispatch table
cmd/git-skill/commands/      — one file per subcommand (add, install, remove, …)
internal/git/plumbing.go     — thin wrappers around git plumbing commands
internal/gitops/             — Asset-Kind trailer, fetch, ls-remote discovery
internal/skill/meta.go       — SKILL.md / AGENT.md parsing and scaffolding
internal/kind/kind.go        — 4-tier kind discriminator (lock/trailer/frontmatter/filename)
internal/refs/refs.go        — refs/assets/<kind>/<name> naming conventions
internal/state/state.go      — assets.json serialization (intent + resolution)
internal/manifest/manifest.go — git-skill.yaml asset manifest
internal/runtimes/           — built-in registry + user/project runtimes.yaml
internal/fs/                 — cross-platform symlink / junction / copy fan-out
internal/assetignore/        — .assetignore gitignore-style filter
internal/semver/semver.go    — SemVer 2.0 comparator and spec parser
```

Runtime dependencies are minimal: the Go stdlib, the `git` binary, plus `gopkg.in/yaml.v3` (manifest + runtimes.yaml parsing) and `golang.org/x/sys` (Windows symlink/junction APIs).

## Design principles

1. **Don't duplicate git.** Versioning, diffing, transport, history — git already provides all of this. git-skill adds a thin layer of conventions on top.
2. **Plumbing, not porcelain.** The code uses `hash-object`, `mktree`, `commit-tree`, `update-ref`. If you need a new operation, look for the right plumbing command rather than parsing porcelain output.
3. **The spec is the contribution.** `SKILL-FORMAT.md` is the real deliverable. The CLI is a reference implementation. Changes to the format require updating the spec.
4. **No external dependencies.** Keep it that way unless there's a compelling reason.

## Making changes

- Keep each commit focused on one thing.
- Run `go test ./... && bash test.sh` before pushing.
- Update `SKILL-FORMAT.md` if you change the storage format or ref naming.
- Update each command's `--help` text if you add or change a command. The top-level command list is generated automatically from the dispatch table (see `printHelp` in `cmd/git-skill/main.go`).

## Submitting a pull request

Open a PR against `main`. The CI runs unit tests on Linux, macOS, and Windows across Go 1.22 and 1.23, plus `bash test.sh` integration tests on Linux and macOS.

If you're adding a new command, include:
- A new file in `cmd/git-skill/commands/` whose exported function matches `func(commands.Profile, []string, io.Writer, io.Writer) error`
- A `register("<name>", ...)` entry in `cmd/git-skill/wire.go` that wraps the new command function
- Coverage in `test.sh`
- Unit tests where the logic is non-trivial

## Reporting issues

Open a GitHub issue with:
- The exact command you ran
- The output you got
- The output you expected
- Your OS and `git --version`

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
cmd/git-skill/main.go     — CLI entry point; one function per command
internal/git/plumbing.go  — thin wrappers around git plumbing commands
internal/skill/meta.go    — SKILL.md parsing and scaffolding
internal/refs/refs.go     — ref naming conventions
internal/lock/lock.go     — skill.lock serialization
```

The CLI has zero external dependencies — only the Go stdlib and the `git` binary.

## Design principles

1. **Don't duplicate git.** Versioning, diffing, transport, history — git already provides all of this. git-skill adds a thin layer of conventions on top.
2. **Plumbing, not porcelain.** The code uses `hash-object`, `mktree`, `commit-tree`, `update-ref`. If you need a new operation, look for the right plumbing command rather than parsing porcelain output.
3. **The spec is the contribution.** `SKILL-FORMAT.md` is the real deliverable. The CLI is a reference implementation. Changes to the format require updating the spec.
4. **No external dependencies.** Keep it that way unless there's a compelling reason.

## Making changes

- Keep each commit focused on one thing.
- Run `go test ./... && bash test.sh` before pushing.
- Update `SKILL-FORMAT.md` if you change the storage format or ref naming.
- Update the help text in `cmdXxx` and the `usage()` function if you add or change a command.

## Submitting a pull request

Open a PR against `main`. The CI runs unit tests on Linux and macOS across Go 1.22 and 1.23.

If you're adding a new command, include:
- The implementation in `cmd/git-skill/main.go`
- A case in the `switch` in `main()`
- An entry in `usage()`
- Coverage in `test.sh`
- Unit tests where the logic is non-trivial

## Reporting issues

Open a GitHub issue with:
- The exact command you ran
- The output you got
- The output you expected
- Your OS and `git --version`

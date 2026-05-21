# Version-controlling AI agent skills with git-skill and GitHub

If you've used Claude, Cursor, or any modern coding agent for more than a week, you've probably accumulated a folder of carefully-tuned skill files — system prompts, style guides, code review checklists, debugging protocols. And if you work on a team, you've probably also discovered that these files don't have a story.

There's no version history. No diff when something changes. No way to pin a known-good revision. No mechanism for "I want the version Alice was using last Tuesday." You drop a folder in `.claude/skills/` and pray nothing drifts.

This is a solved problem for source code. It's been solved since 2005. The infrastructure is sitting in every developer's `git` binary. **git-skill** is a small CLI that says: skills are directories of text files, that's exactly what git is good at, let's just use git.

This article walks through the full workflow: authoring a skill, publishing it to GitHub, consuming it from another machine, upgrading it, and integrating it into a team's day-to-day.

## The core idea

Under the hood, git-skill stores each skill as a git tree object under a custom ref:

```
refs/skills/my-skill            → the latest commit (like a branch)
refs/skill-tags/my-skill/v1.0.0 → a tagged release (like a tag, immutable by convention)
```

These refs live in any existing git repository — your skills repo, your dotfiles, even your main project repo. Push them to GitHub with `git skill push origin` and anyone with read access can `git skill get` them.

The CLI has zero external dependencies. It is ~1,100 lines of Go that shell out to git plumbing commands (`hash-object`, `mktree`, `commit-tree`, `update-ref`) and wrap them in a friendly interface. The spec ([SKILL-FORMAT.md](../SKILL-FORMAT.md)) is independent of the implementation — anyone can write a tool that reads and writes the same format.

## Installation

```bash
# Either build from source
git clone https://github.com/niradler/git-skill
cd git-skill
go build -o git-skill ./cmd/git-skill
sudo mv git-skill /usr/local/bin/

# Or via go install
go install github.com/niradler/git-skill/cmd/git-skill@latest
```

Because the binary is named `git-skill`, git automatically discovers it as a subcommand:

```bash
git skill version   # → git-skill 0.1.0
```

This is a deliberate naming choice — every git skill command looks and feels like a native git operation, because under the hood it *is* one.

## Your first skill

Walk through this in any existing git repository (or create one fresh with `git init`).

```bash
git skill init code-review "Guidelines for reviewing pull requests"
```

This does three things:

1. Creates `.skills/code-review/SKILL.md` from a template
2. Hashes the directory and creates a commit
3. Points `refs/skills/code-review` at that commit

The scaffolded SKILL.md:

```markdown
---
name: code-review
description: Guidelines for reviewing pull requests
version: 0.1.0
---

# code-review

TODO: Describe what this skill does.

## Instructions

TODO: Add the instructions that guide the AI agent.
```

Open the file in your editor and fill it in. A realistic skill looks something like:

```markdown
---
name: code-review
description: Use when reviewing pull requests. Covers security, performance, style.
version: 0.1.0
license: MIT
---

# code-review

When reviewing a pull request, walk through these checks before approving:

## Security
- Any new user input touching auth, DB, or filesystem? Trace it for injection paths.
- New secrets? They must come from env vars or a secret manager, never literals.

## Performance
- Any new N+1 queries? Look for loops calling DB methods.
- Any unbounded list operations on user input?

## Style
- Imports grouped: stdlib, third-party, local — in that order.
- New public APIs have doc comments.

## Output
Post a single review comment that summarizes findings under three headings:
"Must fix", "Should fix", "Nit". Skip empty sections.
```

Snapshot it:

```bash
git skill commit code-review -m "Add initial code review checklist"
```

You can see history at any time:

```bash
git skill log code-review
# abc1234 Add initial code review checklist (2m ago) <Your Name>
# 1.0.0
# def5678 Initialize skill: code-review (5m ago) <Your Name>
# 1.0.0
```

And diff between revisions:

```bash
git skill diff code-review          # latest vs previous
```

## Tagging a release

When the skill is ready to share:

```bash
git skill tag code-review 1.0.0
```

This creates `refs/skill-tags/code-review/v1.0.0` pointing at the current commit. The tag is intended to be immutable by convention — `refs/skill-tags/*` are plain git refs, so an author with push access *can* technically force-move one. The `skill.lock` file (covered below) is the real reproducibility guarantee: it pins the commit SHA, not the tag, so even if a tag gets moved upstream, every machine that ran `git skill sync` against the same lock gets identical bytes.

## Publishing to GitHub

Create any GitHub repository (`my-skills`, `dotfiles`, your main project repo — anything). Then:

```bash
git remote add origin https://github.com/you/my-skills.git
git skill push origin
```

`git skill push` handles both skill refs (`refs/skills/*`) and tag refs (`refs/skill-tags/*`) in one command. Plain `git push` would skip them, which is the most common gotcha — get into the habit of using `git skill push`.

That's it. Your skill is now on GitHub. There's no website to register on, no API token to manage, no package metadata to fill out. The GitHub repository *is* the registry.

## Consuming skills from GitHub

On a different machine, or for a teammate:

```bash
git skill get https://github.com/you/my-skills.git code-review@v1.0.0 .claude/skills/
```

The third argument is a **parent directory**, not the skill directory itself. git-skill appends the skill name, so this command installs into `.claude/skills/code-review/`.

It does four things in one shot:

1. Fetches `refs/skills/code-review` and any tags from the remote.
2. Resolves the `v1.0.0` tag to a specific commit.
3. Extracts the tree atomically into `.claude/skills/code-review/`. "Atomically" here means the new tree is written to a sibling dir and renamed into place — so files that were deleted upstream are actually removed locally, not left behind.
4. Records the pinned commit SHA in `skill.lock`.

The resulting `skill.lock` entry:

```json
{
  "lockfileVersion": 2,
  "skills": {
    "code-review": {
      "remote": "https://github.com/you/my-skills.git",
      "version": "v1.0.0",
      "commit": "abc1234def5678...",
      "canonical": ".claude/skills/code-review",
      "agents": {
        "cursor": ".agents/skills/code-review"
      }
    }
  }
}
```

A few notes on what's stored:

- `canonical` and any path under `agents` are **repo-relative, forward-slash** — written once, readable on Windows, macOS, and Linux from the same lockfile.
- `agents` lists only the symlinks git-skill actually created. If the install target is already an agent's expected path (e.g. installing into `.claude/skills/<name>`), no self-symlink is needed for that agent, and it won't appear here.
- `remote` should be a URL every teammate can reach — `https://github.com/...` or `git@github.com:...`. A local filesystem path (`C:/Users/me/...`, `/home/me/...`) will work for you alone but won't resolve on anyone else's machine, and on Linux git will misread a Windows path like `C:/...` as an SSH host.

Commit `skill.lock`. Now any teammate can reproduce the exact skill state:

```bash
git skill sync
# fetching code-review from https://github.com/you/my-skills.git...
# synced code-review → .claude/skills/code-review (abc1234)
```

`sync` works against the pinned **commit** SHA, not the tag. Under the hood it fetches the skill's ref from the recorded remote if the commit isn't already in the local object store, then resolves the lock's commit and extracts that tree. If the upstream tag got force-moved to a new commit and the old one is no longer reachable, `sync` fails loudly — which is the right outcome, because the lockfile contract is broken at that point.

## Team workflow

The lifecycle of a team-managed skill:

**Skill author** (one person, owns the skill):

```bash
git skill commit code-review -m "Add security review checklist"
git skill tag code-review 1.1.0
git skill push origin
```

**Skill consumers** (everyone else):

```bash
# Update lockfile to the new version
git skill get https://github.com/you/my-skills.git code-review@v1.1.0 .claude/skills/

# Commit the lockfile update
git add skill.lock
git commit -m "chore: bump code-review skill to v1.1.0"
```

**Anyone pulling a branch** (including CI):

```bash
git skill sync   # reads skill.lock, reinstalls everything
```

This last bit is the unlock for CI integration. If your CI runs Claude or Cursor as part of the build (e.g. for automated code review), `git skill sync` in CI ensures the build uses the exact same skill bytes as every developer's machine.

## Upgrading a skill (safely)

Before bumping the version pin, look at what changed:

```bash
# If the consuming repo has a remote named "origin" pointing at the skills repo:
git skill fetch origin

# Otherwise (no remote set up), pass the URL directly. This is the same URL
# you used with `git skill get`:
git skill fetch https://github.com/you/my-skills.git

git skill diff code-review v1.0.0 v1.1.0     # see the diff between two tagged versions
```

If the diff looks good:

```bash
git skill get https://github.com/you/my-skills.git code-review@v1.1.0 .claude/skills/
git add skill.lock && git commit -m "bump code-review to v1.1.0"
```

This is the same upgrade pattern as `npm install foo@1.1.0 && commit package-lock.json`. Familiar muscle memory, applied to skills.

## Importing skills you already have

If your `.claude/skills/` directory is already full of useful skill files:

```bash
git skill track my-skill .claude/skills/my-skill
```

`track` validates that the directory has a SKILL.md, snapshots it as the first commit, and starts tracking it under `refs/skills/my-skill`. From here, the normal workflow applies: commit, tag, push.

## One repo, many skills

A single GitHub repository can host any number of skills:

```bash
git skill list
# SKILL              LAST CHANGE                       WHEN
# code-review        Add security checklist            2 hours ago
# pr-description     Tighten wording                   1 day ago
# commit-message     Initial commit                    3 days ago

git skill push origin
# Pushes refs/skills/code-review, refs/skills/pr-description, refs/skills/commit-message
# and all their version tags
```

Consumers target individual skills by name:

```bash
git skill get https://github.com/you/skills.git code-review@v1.0.0 .claude/skills/
git skill get https://github.com/you/skills.git pr-description@v2.1.0 .claude/skills/
```

Because skill refs live in `refs/skills/` (not `refs/heads/` or `refs/tags/`), they don't conflict with normal git operations. Your branches, your PRs, your release tags — all untouched.

## What it looks like under the hood

Every git-skill command maps to a small set of git plumbing operations. `git skill commit` is essentially:

```bash
# 1. Hash every file in .skills/my-skill/ into the object store
git hash-object -w .skills/my-skill/SKILL.md          # → blob SHA

# 2. Build a tree object from those blobs
git mktree <<< "100644 blob <sha>	SKILL.md"          # → tree SHA

# 3. Create a commit pointing to the tree
git commit-tree <tree-sha> -p <parent> -m "message"   # → commit SHA

# 4. Advance the skill's ref
git update-ref refs/skills/my-skill <commit-sha>
```

That's the entire mechanism. Skills are first-class git objects. Everything in git's tooling works on them natively:

```bash
git cat-file -p refs/skills/code-review
# tree abc123
# parent def456
# author you <you@example.com> 1700000000 +0000
#
# Add initial code review checklist
#
# Skill-Version: 1.0.0

git ls-tree refs/skills/code-review
# 100644 blob aaa... SKILL.md
# 040000 tree bbb... references
```

You don't have to use the `git skill` CLI to consume the output. Any tool that can read a git tree (which is to say, any tool) can read a skill.

## Private repos and authentication

git-skill shells out to plain `git fetch` / `git push` under the hood. It does not implement its own auth — whatever credentials `git clone https://github.com/you/private-skills.git` would use are exactly what `git skill get https://github.com/you/private-skills.git ...` will use.

In practice that means one of:

- **HTTPS** — `git skill get` reads from a [credential helper](https://git-scm.com/docs/gitcredentials). On macOS that's Keychain; on Linux it's typically `libsecret` or the gh CLI. On a fresh machine, run `gh auth login` (or `git config --global credential.helper ...`) once and you're set.
- **SSH** — Use the SSH URL: `git skill get git@github.com:you/private-skills.git code-review@v1.0.0 .claude/skills/`. The CLI passes that through to git verbatim; your `~/.ssh/config` and `ssh-agent` handle the rest.
- **GitHub Actions / CI** — `actions/checkout@v4` configures a token for the current repo only. To fetch skills from a *different* private repo, generate a fine-scoped PAT or a GitHub App token, write it into the credential helper before `git skill sync`. (CI typically uses one bot account for all private skill repos.)

If `git skill get` returns an authentication error, run the equivalent `git fetch <url> refs/skills/*:refs/skills/*` directly — that strips git-skill out of the loop and isolates the auth problem to plain git.

## Reviewing untrusted skill content

Skill files live inside an AI agent's prompt. If you `git skill get` something from a stranger, you're effectively letting them edit your system prompt. Treat that with the same skepticism as installing an npm package from a stranger.

A 30-second audit before installing from an untrusted remote:

```bash
# 1. Fetch refs only — no files written to working dirs yet.
git skill fetch https://github.com/stranger/skills.git

# 2. Read the SKILL.md without checking it out anywhere.
git cat-file -p refs/skills/their-skill:SKILL.md | less

# 3. List every file the skill contains.
git ls-tree -r refs/skills/their-skill

# 4. If satisfied, install. Then commit skill.lock so the SHA is pinned.
git skill get https://github.com/stranger/skills.git their-skill@v1.0.0 .claude/skills/
git add skill.lock && git commit -m "add their-skill@v1.0.0"
```

Bundled `scripts/` are the highest-risk surface. Read them.

## Windows support

git-skill itself is pure Go and builds on Windows. Everything *except* agent symlinks works out of the box.

The symlink piece — automatically linking `.claude/skills/<name>` to the canonical `.skills/<name>` — requires either Developer Mode (Settings → Privacy & security → For developers → Developer Mode) or running as an Administrator, because Windows requires elevated privileges to create symlinks by default.

Two practical workarounds:

- **Install directly into the agent's skills dir.** `git skill get <remote> <name>@v1.0.0 .claude/skills/` puts the files exactly where Claude expects them, with no symlink involved.
- **Use `--agent` to restrict.** `git skill install <name>@v1.0.0 .skills/ --agent claude` skips the cursor symlink and only creates the claude one — useful if you only have one agent installed.

## What to commit

For a consumer setup:

- **Always commit `skill.lock`.** It's the reproducibility contract; without it `git skill sync` is a no-op.
- **Usually commit `.skills/`.** Some teams check in the materialized skill files so newcomers don't need to run `git skill sync` on first checkout, and so the actual skill content is visible in PRs.
- **Sometimes skip `.skills/`.** If the skill is large or frequently updated, gitignore it and treat `git skill sync` as a setup step (similar to `npm install`).

Either choice is fine. The lockfile is non-negotiable; the materialized files are a convenience.

For an author setup, commit the skill source dir alongside `skill.lock`. The `refs/skills/*` and `refs/skill-tags/*` refs live in `.git/` and travel via `git skill push`, not via the working tree.

## Rolling back

A bad skill update is a one-command rollback:

```bash
git skill get https://github.com/you/my-skills.git code-review@v1.0.0 .claude/skills/
git add skill.lock && git commit -m "revert code-review to v1.0.0"
git skill sync
```

That's identical to a forward upgrade — there's no special "rollback" mode because there doesn't need to be. The lockfile and the tag are the only state.

For the author, every release lives at its own immutable tag (`refs/skill-tags/code-review/v1.0.0`), so consumers don't need the author to publish a "v1.0.1 that's actually v1.0.0 again" — they just pin to the old tag.

## When *not* to use git-skill

A few honest scenarios where git-skill isn't the right fit:

- **One-off personal skills** that no one else will ever consume. The overhead of `init`, `commit`, `tag`, `push` is more than the value.
- **Skills with frequent runtime tuning by non-engineers.** If the people editing the skill don't use git, give them a Google Doc or a web UI. git-skill is for engineering teams.
- **Massive binary assets bundled with a skill.** Git is bad at large binaries. Use a separate object store and reference URLs from the SKILL.md.
- **Cross-org public discovery.** git-skill has no "browse all skills" mechanism. It's a distribution and versioning tool, not a registry. A registry would sit on top of the format ([ARCHITECTURE.md](../ARCHITECTURE.md) sketches one).

## Limitations to know

- **Windows symlinks require admin or developer mode.** The agent path symlinks (`.claude/skills/<name>` → `.skills/<name>`) need symlink privileges. This is being tracked.
- **No skill validation beyond SKILL.md presence.** The CLI does not enforce semver, lint the YAML, or check links. Add your own pre-push hooks if you want stricter checks.
- **The agent list is hard-coded** to `claude` and `cursor` today. PRs to add more agents are welcome.

## Summary

| Task | Command |
|------|---------|
| Create a skill | `git skill init <name> "<desc>"` |
| Import existing | `git skill track <name> <dir>` |
| Snapshot changes | `git skill commit <name> -m "<msg>"` |
| View history | `git skill log <name>` |
| See what changed | `git skill diff <name>` |
| Tag a release | `git skill tag <name> <ver>` |
| Publish | `git skill push origin` |
| Install from GitHub | `git skill get <url> <name>@v<ver> <dest>` |
| Sync pinned versions | `git skill sync` |
| List all skills | `git skill list` |

The format spec lives in [SKILL-FORMAT.md](../SKILL-FORMAT.md). If you adopt the format in a different tool, skills become portable across implementations. That portability is the actual point — the CLI is a reference implementation; the format is the contribution.

## Try it

```bash
go install github.com/niradler/git-skill/cmd/git-skill@latest

cd <some-repo>
git skill init my-first-skill "Just testing it out"
git skill log my-first-skill
```

If you build something interesting on top of the format, or want to push the spec forward, open an issue at [github.com/niradler/git-skill](https://github.com/niradler/git-skill).

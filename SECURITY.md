# Security Policy

## Reporting a vulnerability

If you find a security issue in git-skill, please **do not** open a public GitHub issue.

Instead, email the maintainer directly with:

- A description of the issue
- Steps to reproduce
- The affected version (`git skill version`)
- Any suggested fix

Contact: see the repository owner profile at [github.com/niradler](https://github.com/niradler).

You can expect an initial response within a few days.

## Scope

In-scope concerns include:

- Shell or argument injection through skill names, ref names, or remote URLs.
- Path traversal via skill install destinations.
- Symlink-related privilege issues during install/sync.
- Tampering with `skill.lock` that bypasses commit-SHA pinning.

Out-of-scope:

- Bugs in the underlying `git` binary or git's transport protocols.
- Anything that requires write access to the user's local `.git` directory (treated as already-trusted state).
- Issues in third-party skill content fetched from a remote - content is the publisher's responsibility.

## Threat model: consuming third-party skills

Skill files are agent-instruction code. They get loaded into an AI agent's prompt and can steer the agent's behavior. Some risks are inherent to that:

- **Prompt-level attacks.** A hostile `SKILL.md` can instruct the agent to exfiltrate secrets, write backdoors, mislead the user, or invoke tools in dangerous ways. There is no sandbox at the prompt layer - the agent reads the skill the same way it reads your own instructions. Treat installing an untrusted skill like running an untrusted shell script.
- **Bundled scripts.** Skills can ship a `scripts/` directory with executables. git-skill does not run them, but a skill's instructions may tell the agent to. Audit `scripts/` (and the SKILL.md text that references them) before installing.
- **Lockfile tampering.** A malicious PR can change a `commit:` in `skill.lock` to point at hostile content under the same `version:` string. Always diff `skill.lock` in code review; never `git skill sync` from an untrusted branch.
- **Moved upstream tags.** A tag like `v1.0.0` is a plain git ref and can be force-moved by a publisher with write access. The lockfile's `commit:` SHA is what actually pins bytes - the `version:` string is informational.

Mitigations git-skill provides:

- **SHA pinning** in `skill.lock` - `git skill sync` resolves the commit SHA, not the tag, so a moved upstream tag does not silently change installed bytes.
- **Atomic install** - files deleted upstream are removed locally; you cannot end up with stale dangerous files lingering after an upgrade.
- **Fetch without install** - `git skill fetch <remote>` populates the local object store without writing anywhere in the work tree. `git cat-file -p refs/skills/<name>:SKILL.md` lets you read the body before installing.
- **Path validation** - `skill.lock` paths that escape the repo root are rejected by `git skill sync`.

Mitigations git-skill **does not** provide:

- No signature verification on commits or tags (use `git tag -v` / signed commits yourself if you need this).
- No content scanning. The CLI does not inspect skill bodies for prompt-injection patterns.
- No sandboxing of bundled scripts. They are plain files on disk after install.

If you consume skills from outside your organization, treat them as third-party dependencies: pin exact versions, review diffs, and limit what agents can do at the agent layer (tool allowlists, sandboxing) rather than relying on the skill itself to be benign.

## Hardening tips for users

- Treat `skill.lock` like `package-lock.json` or `go.sum`: review changes before merging.
- Pin to specific versions (`name@v1.0.0`), not just `name`, when consuming from remotes you don't control.
- Audit `git log refs/skills/<name>` after fetching to see what changed before installing.
- Before first install from an unfamiliar remote: `git skill fetch <url>` then `git cat-file -p refs/skills/<name>:SKILL.md` to read the body without writing files.
- Inspect bundled `scripts/` directories. If you don't need them, install with `--agent <single>` to keep blast radius small.
- Never run `git skill sync` on an untrusted PR that touches `skill.lock`.

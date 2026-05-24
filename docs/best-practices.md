# git-skill best practices

End-to-end guidance for authoring and consuming skills with `git-skill` ≥ v0.2.0. Each rule below is illustrated by a real PR in the [producer](https://github.com/niradler/git-skill-demos) and [consumer](https://github.com/niradler/git-skill-consumer-demo) demo repos.

> Throughout this doc, "skill" is the primary unit. Agents (single-file `agent.md` / `agent.toml`) are supported by the same machinery and follow the same rules — `skill` and `agent` are sibling asset kinds under `refs/assets/<kind>/<name>`.

---

## 1. Producer: repo layout

```
skills/<name>/
  SKILL.md            # required, with YAML frontmatter
  eval/
    prompts.json      # behavioral eval prompts
    assertions.md     # Structural + Behavioral.<id> checkboxes
    eval.config.yaml  # judge model + passing_score
  version.txt         # base semver, used for dev-tag minting
agents/<name>/
  AGENT.md            # required for agent assets
```

Why `version.txt` is separate from the SKILL.md frontmatter: the frontmatter is the **public contract** read by the runtime. `version.txt` is the **CI mint base** for dev tags (`<base>-dev.<run_number>`). Separating them keeps the runtime contract stable while CI iterates.

See: [`skills/code-review/`](https://github.com/niradler/git-skill-demos/tree/main/skills/code-review) in the producer.

---

## 2. Producer: CI publish workflow

A skill is "published" when CI commits a snapshot to `refs/assets/skill/<name>` and tags it `refs/asset-tags/skill/<name>/v<semver>`. The producer demo wires this up in [`.github/workflows/publish.yml`](https://github.com/niradler/git-skill-demos/blob/main/.github/workflows/publish.yml).

Key conventions:

- **Pin the git-skill version.** Use `go install ...@v0.2.0`, not `@latest`. CI workflows are part of your skill's release surface — pin them like any other dependency.
- **Fetch custom refs before commit.** `actions/checkout@v4` only fetches `refs/heads/*` and `refs/tags/*`. You must explicitly `git fetch origin '+refs/assets/*:refs/assets/*' '+refs/asset-tags/*:refs/asset-tags/*'` before `git skill commit`, or the push will be rejected as non-fast-forward against the existing `refs/assets/<kind>/<name>` history.
- **Use the multi-kind `git-asset` profile to push.** `git skill push` only pushes skill refs. To push both skills and agents in one CI step, symlink the binary as `git-asset` and call `git asset push origin`.
- **Diff against the prior commit, not HEAD.** Use `git diff $before...$after -- 'skills/*' 'agents/*'` so each push only re-publishes what actually changed. Provide a `force_all` workflow_dispatch input as an escape hatch for CI-config changes or initial-state fixes.
- **Skip idempotency: compare current tree to published ref.** Even when a file in `skills/<name>/` changed, the canonical tree may be identical (whitespace-only, etc.). Compare `git rev-parse refs/assets/skill/<name>^{tree}` to `git ls-tree HEAD -- skills/<name>` and skip the commit + tag when they match.
- **Mint dev tags from `version.txt` + run number.** `<base>-dev.<github.run_number>`. The run number is monotonically increasing across the repo, so two skills published in the same run get different bases but the same dev suffix — fine because they live in different namespaces.
- **Tag args order matters.** `git skill tag <name> v<semver>` — the kind flag goes *before* positional args because the CLI uses standard `flag.Parse` which stops at the first non-flag positional.

---

## 3. Producer: promote workflow (dev → prod)

A separate manual workflow turns a battle-tested dev version into a prod release. The producer's [`promote.yml`](https://github.com/niradler/git-skill-demos/blob/main/.github/workflows/promote.yml) takes `skill` + `version` inputs and:

1. Verifies the skill directory + SKILL.md exist
2. Validates the version is bare semver (rejects `-dev` suffix)
3. Runs structure-tier evals (no API key needed)
4. Tags the canonical commit as `v<semver>` (no `-dev`)

**Behavior evals are run locally before opening the promotion PR**, not in CI. CI never gets an `ANTHROPIC_API_KEY`. See section 5 for the local flow.

---

## 4. Producer: eval format

Two tiers, both deterministic-by-design:

### Structure tier (CI)
[`tools/eval-runner/run_evals.py --tier structure`](https://github.com/niradler/git-skill-demos/blob/main/tools/eval-runner) checks the structural assertions in `eval/assertions.md` against the skill files: frontmatter present, required H2 sections present, `prompts.json` parses, every prompt id has a matching `Behavioral.<id>` section, etc.

### Behavior tier (local)
The skill author runs `running-skill-evals` (a skill in the producer demo) inside their Claude Code session. The flow:

1. Read `eval/prompts.json` and `eval/assertions.md`.
2. For each prompt, spawn a `Task` subagent with the full SKILL.md inlined verbatim. **Subagents start with empty context** — they do not inherit the parent session's loaded skills, so you must inline the SKILL.md every time.
3. Score subagent responses against the `Behavioral.<id>` checkboxes (binary checkbox pass/fail).
4. `prompt_score = passed / total`. The prompt passes if `prompt_score >= passing_score` from `eval.config.yaml` (default 0.8).
5. `skill_score = mean(prompt_scores)`. All prompts must pass for the skill to be promotable.

See [`skills/running-skill-evals/SKILL.md`](https://github.com/niradler/git-skill-demos/blob/main/skills/running-skill-evals/SKILL.md) for the full procedure.

---

## 5. Consumer: install flow

```bash
git skill add <name>@<spec> --from <producer-url> --runtime claude
```

- `<spec>` is a tag (e.g. `v0.1.0-dev.12`) or eventually a semver range. As of v0.2.0, exact tags are the most reliable spec form.
- `--from` registers the remote in `assets.json`. Subsequent `update` / `remove` for this skill don't need it.
- `--runtime <name>` materializes into the runtime-specific path (`.claude/skills/<name>/` for `claude`, `.cursor/rules/<name>.mdc` for `cursor`, etc.). Omitted = canonical-only.

The first consumer PR in the demo is [`consumer-demo#1`](https://github.com/niradler/git-skill-consumer-demo/pull/1).

---

## 6. Consumer: upgrade flow

For **dev tags** (immutable per CI run), do a remove + re-add:

```bash
git skill remove <name>
git skill add <name>@<new-tag> --from <producer-url> --runtime claude
```

For **prod tags with a semver range** (`^0.1`), use:

```bash
git skill update <name>
```

`update` re-resolves the existing spec — if the spec is already at an exact dev tag, nothing happens. Example: [`consumer-demo#2`](https://github.com/niradler/git-skill-consumer-demo/pull/2).

---

## 7. Consumer: rollback flow

Same machinery as upgrade — `remove` + `add` at the older tag:

```bash
git skill remove <name>
git skill add <name>@<older-tag> --from <producer-url> --runtime claude
```

`assets.json` snaps back to the older version + commit. The materialized trees revert in place. A fresh clone + `git skill install` resolves directly to the older tag; no replay needed.

Example: [`consumer-demo#4`](https://github.com/niradler/git-skill-consumer-demo/pull/4).

---

## 8. Consumer: multi-skill setup

One `assets.json` can hold any number of skill entries. Each pins independently. Example: [`consumer-demo#3`](https://github.com/niradler/git-skill-consumer-demo/pull/3).

---

## 9. Cross-platform notes

- **Path separators in `assets.json`.** Always commit forward-slash paths (`skills/code-review`, not `skills\code-review`). The canonical-path writer in the current v0.2.0 release emits OS-native separators — fix this in your PR by hand until the tool normalizes (tracked as a known issue).
- **Windows symlinks.** Require developer mode or admin. Without it, the materializer falls back to a copy. Lock files written on Windows are interchangeable with macOS/Linux ones as long as paths use `/`.
- **Line endings.** Git's `core.autocrlf` on Windows can rewrite line endings in materialized files. Add a `.gitattributes` rule (`*.md text eol=lf`) in your consumer repo if you need byte-stable trees across platforms.

---

## 10. Versioning discipline

| Producer action | Tag minted |
|---|---|
| Push to `main` that changes a skill | `v<base>-dev.<run_number>` |
| Manual `promote.yml` dispatch | `v<version>` (bare semver, no `-dev`) |

| Consumer pin form | Behavior |
|---|---|
| `v0.1.0-dev.12` (exact dev tag) | Immutable. `update` is a no-op. |
| `v0.1.0` (exact prod tag) | Immutable. `update` is a no-op. |
| `^0.1` (range — when supported) | `update` resolves to highest matching prod tag. |

Rule of thumb: **pin to exact tags by default. Use ranges only when you trust the producer's release cadence.**

---

## 11. Things this doc deliberately does not cover

- Authoring SKILL.md content — that's the skill author's craft, not the tool's concern. See the producer demo for examples.
- Agent vs skill semantics — both kinds use the same machinery. The differences (single-file vs directory tree, frontmatter conventions) are documented in the git-skill README.
- Private remotes / auth — `git skill` defers to git's standard credential helpers. If `git fetch <url>` works, so does `git skill add ... --from <url>`.

# Architecture

> git-skill is to SkillsHub what git is to GitHub.
> The CLI works without the platform. The platform amplifies the CLI.

---

## Components

### 1. git-skill (CLI) — this repo

Open-source Go binary. Works with **any** git remote — GitHub, GitLab, self-hosted, or SkillsHub.

**Owns:**

- Local asset storage as git tree objects under `refs/assets/<kind>/<name>`, with two built-in kinds — `skill` and `agent`.
- Versioned history via commit chain; tagged releases under `refs/asset-tags/<kind>/<name>/v<semver>`.
- `Asset-Kind` commit trailer on every snapshot, making commits self-describing.
- `assets.json` — single committed file carrying both intent (`spec`) and resolution (`version` + `commit`) so a fresh clone + `install` restores the exact working tree.
- `.assetignore` — gitignore-style filter applied during copy materialization.
- Push/fetch over standard git transport (SSH, HTTPS, local path).
- Cross-platform materialization — Unix relative symlinks; Windows junctions or copy fallback when symlink privilege is unavailable.
- Three CLI personas via argv[0] dispatch: `git-skill`, `git-agent`, `git-asset` share one binary; the invocation name sets the default kind.
- Built-in runtime registry (`claude`, `cursor`, `codex`, `opencode`) plus extension points: per-asset `git-skill.yaml` manifests and user/project `runtimes.yaml` configs.

**Commands:**

| Command | What it does |
|---|---|
| `init` | Scaffold `assets.json` + managed `.gitignore` block |
| `commit` | Snapshot canonical tree into `refs/assets/<kind>/<name>` (writes `Asset-Kind` trailer) |
| `tag` | Tag a commit as `refs/asset-tags/<kind>/<name>/v<semver>` |
| `push / fetch` | Sync `refs/assets/<kind>/*` and `refs/asset-tags/<kind>/*` with any remote |
| `list / log / diff / show` | Inspect assets and history |
| `add` | Resolve `<ns>/<name>[@spec] --from <url>` → fetch + record in `assets.json` |
| `update` | Re-resolve `spec` to a new version + commit, re-materialize |
| `remove` | Drop from `assets.json`, delete canonical + runtime paths |
| `install` | Read `assets.json`, fetch pinned commits, restore canonical trees, fan out into runtime paths |
| `discover` | Enumerate remote assets via `ls-remote` |

**Does NOT own:**

- Discovery / search across repos
- Evals, ratings, quality signals
- Web UI, org management, access control
- API beyond git transport

---

### 2. SkillsHub (platform) — separate repo

Hosted SaaS. Analogous to GitHub for code repos.

**Owns:**

- Asset registry — indexed by `name`, `description`, `kind`, version, author, org.
- Web UI — browse, search, preview skills and agents.
- Evals — run an asset against test inputs, track pass rates across versions.
- Version diff UI — visual diff between two tagged versions.
- Ratings / quality signals — community trust score, install counts.
- Org namespacing — `skillshub.io/acme-corp/code-review`.
- Auth — API tokens for `git skill push skillshub.io/...`.
- Webhooks — notify downstream when a version is published.
- `skills.sh`-compatible directory listing (JSON API).

**Does NOT own:**

- The git object format (defined by the CLI + `SKILL-FORMAT.md`).
- Local install mechanics.
- `assets.json` format.

---

## Ownership Boundaries

```
┌─────────────────────────────────────────────────────────────┐
│                        AUTHOR MACHINE                       │
│                                                             │
│  skills/code-review/SKILL.md            ← author edits      │
│  refs/assets/skill/code-review          ← git skill commit  │
│  refs/asset-tags/skill/code-review/v1.0.0 ← git skill tag   │
│  (push)                                 ← git skill push    │
└────────────────────────┬────────────────────────────────────┘
                         │  git push (HTTPS / SSH)
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                      SKILLSHUB PLATFORM                     │
│                                                             │
│  git remote (bare repo per namespace)                       │
│  indexer  →  search, versions, evals, ratings               │
│  web UI   →  browse, diff, install instructions             │
│  API      →  JSON asset metadata, eval results              │
└────────────────────────┬────────────────────────────────────┘
                         │  git fetch / git skill add + install
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                      CONSUMER MACHINE                       │
│                                                             │
│  git skill add acme/code-review@^1.0.0 --from <url>         │
│    → resolves spec, writes assets.json                      │
│  git skill install                                          │
│    → fetches pinned commit, restores canonical,             │
│      fans out into .claude/skills/code-review/ etc.         │
│  assets.json is committed alongside app code.               │
└─────────────────────────────────────────────────────────────┘
```

---

## Integration Points

### CLI → Platform

| Action | Mechanism |
|---|---|
| Publish | `git skill push skillshub.io/org/repo` (standard git push of `refs/assets/<kind>/*` + `refs/asset-tags/<kind>/*`) |
| Onboard remote asset | `git skill add acme/code-review@^1.0.0 --from skillshub.io/acme` |
| Materialize | `git skill install` (fetches the pinned commit recorded in `assets.json`) |
| Enumerate remote | `git skill discover <url>` (uses `git ls-remote`) |

The CLI never calls a SkillsHub HTTP API directly. It talks git.

### Platform → CLI

The platform is a git remote. It accepts pushes and serves fetches via standard git smart-HTTP. The indexer runs as a post-receive hook that reads `refs/assets/<kind>/*` and `refs/asset-tags/<kind>/*` to build the search index and trigger evals. The `Asset-Kind` commit trailer lets the indexer recover the kind from the commit alone.

---

## Data Flows

### Publishing an asset

```bash
# Author tree at skills/code-review/
git skill init
$EDITOR skills/code-review/SKILL.md
git skill commit code-review --path skills/code-review -m "initial"
git skill tag code-review 1.0.0
git skill push skillshub.io/acme-corp
# → platform indexes it, runs evals, makes it discoverable
```

### Consuming an asset

```bash
git skill add acme/code-review@^1.0.0 \
    --from skillshub.io/acme-corp
git skill install
# → fetches pinned commit, materializes canonical + runtime paths,
#   commit assets.json so teammates get byte-identical files
```

### Reproducing a teammate's setup

```bash
git clone <their-app-repo>
git skill install
# → reads assets.json, fetches each pinned commit, restores trees
```

---

## Key Design Decisions

**CLI is standalone by design.** A team can use git-skill with a private GitHub repo and never touch SkillsHub. This keeps the open source tool genuinely useful and drives adoption that feeds the platform.

**`assets.json` is the differentiator.** Many package managers install from HEAD on every run. `assets.json` pins exact commit SHAs alongside the semver spec — installs are reproducible, auditable, rollback-able. Evals, compliance reports, and other platform features build on top of this pin.

**Two kinds, one storage shape.** Skills materialize as full directory trees; agents materialize as a single marker file at the runtime target. The storage format and CLI flow are identical — the runtime registry decides how the fan-out happens.

**Layered runtime resolution.** Per-asset overrides live in the lock entry, asset-author overrides in `git-skill.yaml`, repo-wide policy in `<repo>/.git-skill/runtimes.yaml`, machine-wide defaults in `~/.config/git-skill/runtimes.yaml`, factory defaults in the built-in registry. Precedence is fully documented in [`README.md`](./README.md#resolution-precedence).

**Platform adds value the CLI cannot.** Evals require compute and persistent state. Search requires an index. Ratings require a community. None of these belong in a local CLI. The platform owns them cleanly.

**Standard git transport keeps the moat small.** Any team can self-host a bare git repo and use the full CLI feature set. SkillsHub competes on features (evals, UI, discovery) not on lock-in.

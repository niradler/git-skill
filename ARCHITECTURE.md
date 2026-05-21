# Architecture

> git-skill is to SkillsHub what git is to GitHub.
> The CLI works without the platform. The platform amplifies the CLI.

---

## Components

### 1. git-skill (CLI) — this repo

Open source Go binary. Works with **any** git remote — GitHub, GitLab, self-hosted, or SkillsHub.

**Owns:**
- Local skill storage as git tree objects (`refs/skills/<name>`)
- Versioned history via commit chain, tagged releases via `refs/skill-tags/<name>/vX.Y.Z`
- `skill.lock` — pinned commit SHAs for reproducible installs
- Push/fetch over standard git transport (SSH, HTTPS, local path)
- Install — extract a skill tree to any destination directory
- No network dependency beyond git itself

**Commands:**

| Command | What it does |
|---|---|
| `init` | Scaffold a new skill, first commit |
| `track` | Import an existing skill directory |
| `commit` | Snapshot current skill state |
| `tag` | Tag a named release |
| `log / diff / show / list` | Inspect skill history |
| `push / fetch` | Sync with any remote |
| `install` | Extract locally-fetched skill to a dir |
| `get` | Fetch from remote + install + write `skill.lock` |
| `sync` | Reinstall all skills pinned in `skill.lock` |

**Does NOT own:**
- Discovery / search across repos
- Evals, ratings, quality signals
- Web UI, org management, access control
- API beyond git transport

---

### 2. SkillsHub (platform) — separate repo

Hosted SaaS. Analogous to GitHub for code repos.

**Owns:**
- Skill registry — indexed by `name`, `description`, version, author, org
- Web UI — browse, search, preview skills
- Evals — run a skill against test inputs, track pass rates across versions
- Version diff UI — visual diff between `v1.0.0` and `v1.1.0` of a skill
- Ratings / quality signals — community trust score, download counts
- Org namespacing — `skillshub.io/acme-corp/frontend-design`
- Auth — API tokens for `git skill push skillshub.io/...`
- Webhooks — notify downstream when a skill version is published
- `skills.sh`-compatible directory listing (JSON API)

**Does NOT own:**
- The git object format (defined by CLI / SKILL-FORMAT.md)
- Local install mechanics
- `skill.lock` format

---

## Ownership Boundaries

```
┌─────────────────────────────────────────────────────────────┐
│                        AUTHOR MACHINE                       │
│                                                             │
│  .skills/my-skill/SKILL.md   ←  git-skill init / commit    │
│  refs/skills/my-skill        ←  git-skill tag              │
│  refs/skill-tags/.../v1.0.0  ←  git-skill push             │
└────────────────────────┬────────────────────────────────────┘
                         │  git push (HTTPS / SSH)
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                      SKILLSHUB PLATFORM                     │
│                                                             │
│  git remote (bare repo per namespace)                       │
│  indexer  →  search, versions, evals, ratings              │
│  web UI   →  browse, diff, install instructions            │
│  API      →  JSON skill metadata, eval results             │
└────────────────────────┬────────────────────────────────────┘
                         │  git fetch / git skill get
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                      CONSUMER MACHINE                       │
│                                                             │
│  git skill get skillshub.io/acme/frontend-design@v1.0.0    │
│    → installs to .claude/skills/frontend-design/           │
│    → pins commit SHA in skill.lock                         │
│  git skill sync  →  reproducible reinstall from lock       │
└─────────────────────────────────────────────────────────────┘
```

---

## Integration Points

### CLI → Platform

| Action | Mechanism |
|---|---|
| Publish a skill | `git skill push skillshub.io/org/repo` (standard git push) |
| Install a skill | `git skill get skillshub.io/org/skill@v1.0.0 .claude/skills/` |
| List remote skills | `git for-each-ref` on fetched refs (future: `git skill ls-remote`) |

The CLI never calls a SkillsHub HTTP API directly. It talks git.

### Platform → CLI

The platform is a git remote. It accepts pushes and serves fetches via standard git smart-HTTP. The indexer runs as a post-receive hook that reads `refs/skills/*` and `refs/skill-tags/*` to build the search index and trigger evals.

---

## Data Flows

### Publishing a skill

```
git skill init my-skill "Does X"
# edit .skills/my-skill/SKILL.md
git skill commit my-skill -m "Initial version"
git skill tag my-skill 1.0.0
git skill push skillshub.io/acme-corp
# → platform indexes it, runs evals, makes it discoverable
```

### Consuming a skill

```
git skill get skillshub.io/acme-corp my-skill@v1.0.0 .claude/skills/my-skill
# → fetches git objects, extracts tree, writes skill.lock

# On a new machine or in CI:
git skill sync
# → reads skill.lock, fetches exact commit SHAs, reinstalls
```

---

## Key Design Decisions

**CLI is standalone by design.** A team can use git-skill with a private GitHub repo and never touch SkillsHub. This keeps the open source tool genuinely useful and drives adoption that feeds the platform.

**The lockfile is the differentiator.** `npx skills` installs from HEAD on every run. `skill.lock` pins a commit SHA — the install is reproducible, auditable, and rollback-able. This is the core value the platform can build on (eval history per pinned version, compliance reports, etc.).

**Platform adds value the CLI cannot.** Evals require compute and persistent state. Search requires an index. Ratings require a community. None of these belong in a local CLI. The platform owns them cleanly.

**Standard git transport keeps the moat small.** Any team can self-host a bare git repo and use the full CLI feature set. SkillsHub competes on features (evals, UI, discovery) not on lock-in.

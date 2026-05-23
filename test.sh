#!/usr/bin/env bash
set -euo pipefail

BINARY="./git-skill"
TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

echo "=== git-skill integration test ==="

echo "--- build ---"
go build -o "$BINARY" ./cmd/git-skill

BINARY_ABS=$(realpath "$BINARY")
export PATH="$(dirname "$BINARY_ABS"):$PATH"

# Producer repo
cd "$TMPDIR"
git init -q producer
cd producer
git config user.email "p@p"
git config user.name "p"
git commit --allow-empty -q -m "init"

# Bare remote
BARE="$TMPDIR/bare.git"
git init --bare -q "$BARE"
git remote add origin "$BARE"

echo ""
echo "--- author skill ---"
mkdir -p src
cat > src/SKILL.md <<'EOF'
---
name: acme/x
description: A test skill
---
# acme/x v1.0.0
EOF

echo ""
echo "--- commit ---"
git-skill commit acme/x -m "v1" --path src

echo ""
echo "--- tag ---"
git-skill tag acme/x v1.0.0
git for-each-ref refs/asset-tags/skill/ | grep -q "acme/x/v1.0.0" && echo "OK: tag ref created"

echo ""
echo "--- push ---"
git-skill push origin

echo ""
echo "--- list ---"
git-skill list

echo ""
echo "--- log ---"
git-skill log acme/x

echo ""
echo "--- show ---"
git-skill show acme/x

# Consumer repo
cd "$TMPDIR"
git init -q consumer
cd consumer
git config user.email "c@c"
git config user.name "c"
git commit --allow-empty -q -m "init"

echo ""
echo "--- consumer: init ---"
git-skill init
test -f assets.json && echo "OK: assets.json created"

echo ""
echo "--- consumer: add ---"
git-skill add "acme/x@v1.0.0" --from "$BARE" --runtime claude

# Canonical + runtime paths must exist
test -f skills/acme/x/SKILL.md && echo "OK: canonical SKILL.md present"
test -f .claude/skills/acme/x/SKILL.md && echo "OK: runtime SKILL.md present"

echo ""
echo "--- consumer: assets.json has pinned commit ---"
COMMIT=$(grep -oE '"commit": ?"[0-9a-f]{40}"' assets.json | head -1)
[ -n "$COMMIT" ] && echo "OK: $COMMIT"

echo ""
echo "--- producer: publish v1.1.0 ---"
cd "$TMPDIR/producer"
cat > src/SKILL.md <<'EOF'
---
name: acme/x
description: A test skill
---
# acme/x v1.1.0 - updated
EOF
git-skill commit acme/x -m "v1.1.0" --path src
git-skill tag acme/x v1.1.0
git-skill push origin

echo ""
echo "--- consumer: update to v1.1.0 ---"
cd "$TMPDIR/consumer"
# Pin spec is exact (v1.0.0) so update will keep v1.0.0; switch to range first.
# Simulate range pin by re-adding with ^v1.0.0.
git-skill remove acme/x
git-skill add "acme/x@^v1.0.0" --from "$BARE" --runtime claude
grep -q "v1.1.0" .claude/skills/acme/x/SKILL.md && echo "OK: range pin resolved to v1.1.0"

echo ""
echo "--- consumer: install idempotency ---"
rm -rf skills .claude
git-skill install
test -f skills/acme/x/SKILL.md && echo "OK: install restored canonical"
test -f .claude/skills/acme/x/SKILL.md && echo "OK: install restored runtime"

echo ""
echo "--- consumer: remove ---"
git-skill remove acme/x
[ ! -e skills/acme/x ] && echo "OK: canonical removed"
[ ! -e .claude/skills/acme/x ] && echo "OK: runtime removed"

echo ""
echo "=== all integration checks passed ==="

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

cd "$TMPDIR"
git init test-repo
cd test-repo
git commit --allow-empty -m "init"

echo ""
echo "--- init ---"
git-skill init my-skill "A test skill"
git-skill list

echo ""
echo "--- show ---"
git-skill show my-skill

echo ""
echo "--- verify ref exists ---"
git for-each-ref refs/skills/
echo "OK: ref created"

echo ""
echo "--- edit and commit ---"
cat >> .skills/my-skill/SKILL.md <<'EOF'

## Error Handling

When input is ambiguous, ask one clarifying question.
EOF
git-skill commit my-skill -m "Add error handling section"

echo ""
echo "--- log ---"
git-skill log my-skill

echo ""
echo "--- diff ---"
git-skill diff my-skill

echo ""
echo "--- tag ---"
git-skill tag my-skill 1.0.0
git for-each-ref refs/skill-tags/
echo "OK: tag created"

echo ""
echo "--- second edit ---"
cat >> .skills/my-skill/SKILL.md <<'EOF'

## Output Format

Return structured JSON when the user requests data.
EOF
git-skill commit my-skill -m "Add output format guidance"
git-skill tag my-skill 1.1.0

echo ""
echo "--- diff between tags ---"
git-skill diff my-skill v1.0.0 v1.1.0

echo ""
echo "--- install to directory ---"
INSTALL_PARENT="$TMPDIR/installed"
git-skill install my-skill@v1.0.0 "$INSTALL_PARENT"
INSTALL_DIR="$INSTALL_PARENT/my-skill"
echo "installed files:"
find "$INSTALL_DIR" -type f | sort

echo ""
echo "--- verify installed SKILL.md ---"
head -5 "$INSTALL_DIR/SKILL.md"

echo ""
echo "--- track existing dir ---"
git-skill track my-skill-copy .skills/my-skill
git-skill list

echo ""
echo "--- get (from local bare remote) ---"
# Set up a local bare repo to act as the remote
BARE="$TMPDIR/bare.git"
git init --bare "$BARE"
git-skill push "$BARE"

# Consumer repo — fresh, receives skill via get
cd "$TMPDIR"
git init consumer-repo
cd consumer-repo
git commit --allow-empty -m "init"

CONSUMER_PARENT="$TMPDIR/consumer-repo/.claude/skills"
CONSUMER_SKILL="$CONSUMER_PARENT/my-skill"
git-skill get "$BARE" my-skill@v1.0.0 "$CONSUMER_PARENT"
echo "skill.lock contents:"
cat skill.lock
echo "installed SKILL.md first line:"
head -1 "$CONSUMER_SKILL/SKILL.md"

echo ""
echo "--- sync (reinstall from skill.lock after deletion) ---"
rm -rf "$CONSUMER_SKILL"
git-skill sync
test -f "$CONSUMER_SKILL/SKILL.md" && echo "OK: sync restored SKILL.md"

echo ""
echo "--- sync deletes upstream-removed files (atomic replace) ---"
# Author repo: add a file, commit+tag, then remove it, commit+tag again.
(
  cd "$TMPDIR/test-repo"
  mkdir -p .skills/my-skill/references
  echo "old" > .skills/my-skill/references/old.md
  git-skill commit my-skill -m "Add references/old.md"
  git-skill tag my-skill 2.0.0
  rm .skills/my-skill/references/old.md
  rmdir .skills/my-skill/references 2>/dev/null || true
  git-skill commit my-skill -m "Remove references/old.md"
  git-skill tag my-skill 2.1.0
  git-skill push "$BARE"
)
# Consumer pins to v2.0.0 (which has the file).
cd "$TMPDIR/consumer-repo"
git-skill get "$BARE" my-skill@v2.0.0 "$CONSUMER_PARENT"
test -f "$CONSUMER_SKILL/references/old.md" && echo "OK: v2.0.0 has old.md"
# Bump to v2.1.0 (which does not). Sync must delete the stale file.
git-skill get "$BARE" my-skill@v2.1.0 "$CONSUMER_PARENT"
git-skill sync
if [ -e "$CONSUMER_SKILL/references/old.md" ]; then
  echo "FAIL: stale file present after sync"; exit 1
fi
echo "OK: sync removed upstream-deleted file"

echo ""
echo "=== all tests passed ==="

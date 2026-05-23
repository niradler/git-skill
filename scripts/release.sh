#!/usr/bin/env bash
# scripts/release.sh - cross-compile binaries and (optionally) cut a GitHub release.
#
# Usage:
#   scripts/release.sh <version>            # build only, artifacts to dist/
#   scripts/release.sh <version> --release  # build + tag + push + gh release create
#
# Version is semver without the leading v (e.g. 0.1.0). The tag will be v<version>.
#
# Requirements: bash, go, git, sha256sum, gh (only for --release).

set -euo pipefail

VERSION="${1:-}"
MODE="${2:-build}"

if [[ -z "$VERSION" ]]; then
  echo "Usage: scripts/release.sh <version> [--release]" >&2
  echo "  version: semver without v prefix (e.g. 0.1.0)" >&2
  echo "  --release: also create git tag, push, and a GitHub release" >&2
  exit 1
fi

VERSION="${VERSION#v}"
if [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.\-]+)?$ ]]; then
  echo "error: version '$VERSION' is not valid semver (e.g. 0.1.0 or 0.1.0-rc1)" >&2
  exit 1
fi

TAG="v${VERSION}"
RELEASE=0
[[ "$MODE" == "--release" ]] && RELEASE=1

echo "==> preflight"

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "error: must run inside a git repo" >&2
  exit 1
fi

if ! git diff --quiet --exit-code || ! git diff --cached --quiet --exit-code; then
  echo "error: working tree has uncommitted changes - commit or stash first" >&2
  exit 1
fi

if [[ "$RELEASE" == "1" ]] && git rev-parse "refs/tags/$TAG" >/dev/null 2>&1; then
  echo "error: tag $TAG already exists locally" >&2
  exit 1
fi

if [[ "$RELEASE" == "1" ]] && ! command -v gh >/dev/null 2>&1; then
  echo "error: gh CLI not installed - required for --release" >&2
  exit 1
fi

echo "==> tests"
go test ./...

echo "==> building $TAG"
rm -rf dist
mkdir -p dist

LDFLAGS="-s -w -X main.version=${VERSION}"

build() {
  local os=$1 arch=$2 ext=${3:-}
  local out="dist/git-skill_${VERSION}_${os}_${arch}${ext}"
  printf '  %-15s -> %s\n' "$os/$arch" "$out"
  GOOS=$os GOARCH=$arch CGO_ENABLED=0 \
    go build -trimpath -ldflags "$LDFLAGS" -o "$out" ./cmd/git-skill
}

build linux   amd64
build linux   arm64
build darwin  amd64
build darwin  arm64
build windows amd64 .exe
build windows arm64 .exe

echo "==> checksums"
(cd dist && sha256sum git-skill_* > checksums.txt)
cat dist/checksums.txt

echo "==> artifacts"
ls -lh dist/

if [[ "$RELEASE" != "1" ]]; then
  echo
  echo "build complete. To publish:"
  echo "  scripts/release.sh $VERSION --release"
  exit 0
fi

echo "==> extracting release notes from CHANGELOG.md"
NOTES_FILE="$(mktemp)"
trap 'rm -f "$NOTES_FILE"' EXIT
awk -v v="$VERSION" '
  /^## \[/ {
    if (capture) exit
    if (index($0, "[" v "]") > 0) { capture = 1; next }
  }
  capture { print }
' CHANGELOG.md > "$NOTES_FILE"

if [[ ! -s "$NOTES_FILE" ]]; then
  echo "warning: no CHANGELOG section for [$VERSION] - using auto-generated notes" >&2
  GH_NOTES_FLAG=(--generate-notes)
else
  GH_NOTES_FLAG=(--notes-file "$NOTES_FILE")
fi

echo "==> tagging $TAG"
git tag -a "$TAG" -m "Release $TAG"

echo "==> pushing $TAG to origin"
git push origin "$TAG"

echo "==> creating GitHub release $TAG"
gh release create "$TAG" dist/git-skill_* dist/checksums.txt \
  --title "$TAG" \
  "${GH_NOTES_FLAG[@]}"

echo
echo "done. release: $(gh release view "$TAG" --json url -q .url)"

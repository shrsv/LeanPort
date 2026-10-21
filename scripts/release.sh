#!/usr/bin/env bash
# Build the six release binaries, checksums, and publish with gh.
# Usage: scripts/release.sh vX.Y.Z        (DRY_RUN=1 to stop before gh)
set -euo pipefail

VERSION="${1:-}"
if [ -z "$VERSION" ]; then
	echo "usage: $0 vX.Y.Z" >&2
	exit 2
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST="$ROOT/dist/$VERSION"
COMMIT="$(git -C "$ROOT" rev-parse --short HEAD)"
DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

export PATH="${HOME}/.go/bin:${PATH}"
rm -rf "$DIST"
mkdir -p "$DIST"

for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do
	os="${target%/*}"
	arch="${target#*/}"
	ext=""
	[ "$os" = "windows" ] && ext=".exe"
	out="$DIST/leanport_${VERSION}_${os}_${arch}${ext}"
	echo "→ $out"
	(cd "$ROOT" && CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
		go build -trimpath \
		-ldflags "-s -w -X main.version=$VERSION -X main.commit=$COMMIT -X main.date=$DATE" \
		-o "$out" ./cmd/leanport)
done

(cd "$DIST" && sha256sum ./* >checksums.txt)
echo
echo "artifacts in $DIST"
(cd "$DIST" && ls -lh)

if [ "${DRY_RUN:-0}" = "1" ]; then
	echo "DRY_RUN=1: not publishing"
	exit 0
fi

if ! git -C "$ROOT" rev-parse -q --verify "refs/tags/$VERSION" >/dev/null; then
	echo "tag $VERSION does not exist; create and push it first" >&2
	exit 1
fi

gh release create "$VERSION" "$DIST"/* \
	--repo shrsv/LeanPort \
	--title "LeanPort $VERSION" \
	--generate-notes

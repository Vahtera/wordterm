#!/usr/bin/env bash
# Builds standalone Wordterm executables into dist/.
#   ./build.sh            vet, test, then build all six targets
#   ./build.sh --native   vet, test, then build only for this machine
# Set VERSION=1.2.3 to stamp the About dialog (default: git describe, or "dev").
set -euo pipefail
cd "$(dirname "$0")"

VERSION="${VERSION:-$(git describe --tags --always 2>/dev/null || echo dev)}"
TARGETS=(linux/amd64 linux/arm64 windows/amd64 windows/arm64 darwin/amd64 darwin/arm64)
if [[ "${1:-}" == "--native" ]]; then
  TARGETS=("$(go env GOOS)/$(go env GOARCH)")
fi

echo "== go vet"
go vet ./...
echo "== go test"
go test ./...

mkdir -p dist
for t in "${TARGETS[@]}"; do
  os="${t%/*}"; arch="${t#*/}"; ext=""
  [[ "$os" == "windows" ]] && ext=".exe"
  out="dist/wordterm-${os}-${arch}${ext}"
  echo "== building ${out}"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
    go build -trimpath -ldflags "-s -w -X wordterm/internal/app.Version=${VERSION}" -o "$out" .
done

( cd dist && { sha256sum wordterm-* 2>/dev/null || shasum -a 256 wordterm-*; } > SHA256SUMS )
echo "== done: dist/ (version ${VERSION})"

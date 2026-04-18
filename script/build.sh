#!/usr/bin/env bash
set -euo pipefail

# build_script_override for cli/gh-extension-precompile@v2.
# Outputs binaries into dist/<os>-<arch> (the naming gh expects).
# Restricted to the platforms we actually ship.

mkdir -p dist

platforms=(
  "darwin arm64"
  "linux arm"
  "linux arm64"
)

for p in "${platforms[@]}"; do
  read -r goos goarch <<<"$p"
  out="dist/${goos}-${goarch}"
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 \
    go build -trimpath -ldflags="-s -w" -o "$out" .
done

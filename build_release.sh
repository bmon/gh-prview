#!/bin/bash
set -e

# based on https://github.com/cli/gh-extension-precompile/blob/trunk/build_and_release.sh

platforms=(
  darwin-amd64
  darwin-arm64
  linux-amd64
  linux-arm64
  windows-amd64
  windows-arm64
)

prerelease=""
if [[ $GH_RELEASE_TAG = *-* ]]; then
  prerelease="--prerelease"
fi

draft_release=""
if [[ "$DRAFT_RELEASE" = "true" ]]; then
  draft_release="--draft"
fi

IFS=$'\n' read -d '' -r -a supported_platforms < <(go tool dist list) || true

for p in "${platforms[@]}"; do
  goos="${p%-*}"
  goarch="${p#*-}"
  if [[ " ${supported_platforms[*]} " != *" ${goos}/${goarch} "* ]]; then
    echo "warning: skipping unsupported platform $p" >&2
    continue
  fi
  ext=""
  if [ "$goos" = "windows" ]; then
    ext=".exe"
  fi
  cc=""
  cgo_enabled="${CGO_ENABLED:-0}"
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED="$cgo_enabled" CC="$cc" go build -trimpath -ldflags="-s -w" -o "dist/${p}${ext}" cmd/main.go
done

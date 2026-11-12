#!/bin/sh
# Builds fsr for every platform and publishes it as a GitHub release, which is where
# install.sh / install.ps1 download from. Needs Go 1.26+ and the gh CLI (logged in).
# Usage: sh fsr/release.sh v0.1.0
set -e
VERSION="$1"
case "$VERSION" in v*) ;; *) echo "usage: sh fsr/release.sh vX.Y.Z" >&2; exit 1 ;; esac
cd "$(dirname "$0")"
go test ./...
rm -rf dist && mkdir dist
for t in darwin/arm64 darwin/amd64 linux/amd64 linux/arm64 windows/amd64 windows/arm64; do
  os=${t%/*}; arch=${t#*/}; ext=; [ "$os" = windows ] && ext=.exe
  CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath -tags nodynamic \
    -ldflags "-s -w -X main.version=$VERSION" -o "dist/fsr-$os-$arch$ext" .
done
if command -v sha256sum >/dev/null 2>&1; then
  (cd dist && sha256sum fsr-* > sha256sums.txt)
else
  (cd dist && shasum -a 256 fsr-* > sha256sums.txt)
fi
gh release create "$VERSION" dist/* --repo mozm-ctrl/fs-reimburse-skill --target main --title "$VERSION" --generate-notes

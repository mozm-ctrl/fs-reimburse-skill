#!/bin/sh
# Installs the fsr CLI on macOS, Linux and Windows Git Bash; nothing else needs to be installed.
# Usage: curl -fsSL https://raw.githubusercontent.com/mozm-ctrl/fs-reimburse-skill/main/install.sh | sh
# Options (after `sh -s --`): --bin-dir DIR, --version vX.Y.Z
set -e
REPO=mozm-ctrl/fs-reimburse-skill
BIN_DIR="$HOME/.local/bin"
VERSION=latest
while [ $# -gt 0 ]; do
  case "$1" in
    --bin-dir) BIN_DIR="$2"; shift 2 ;;
    --version) VERSION="$2"; shift 2 ;;
    *) echo "unknown option $1" >&2; exit 1 ;;
  esac
done

case "$(uname -s)" in
  Darwin) OS=darwin ;;
  Linux) OS=linux ;;
  MINGW* | MSYS* | CYGWIN*)
    # Git Bash: use the PowerShell installer so fsr.exe also works from PowerShell and cmd.
    exec powershell.exe -NoProfile -ExecutionPolicy Bypass -Command \
      "irm https://raw.githubusercontent.com/$REPO/main/install.ps1 | iex" ;;
  *) echo "不支持的系统：$(uname -s)" >&2; exit 1 ;;
esac
case "$(uname -m)" in
  x86_64 | amd64) ARCH=amd64 ;;
  arm64 | aarch64) ARCH=arm64 ;;
  *) echo "不支持的 CPU：$(uname -m)" >&2; exit 1 ;;
esac

if [ "$VERSION" = latest ]; then
  BASE="https://github.com/$REPO/releases/latest/download"
else
  BASE="https://github.com/$REPO/releases/download/$VERSION"
fi
fetch() {
  if command -v curl >/dev/null 2>&1; then curl -fsSL -o "$1" "$2"
  elif command -v wget >/dev/null 2>&1; then wget -qO "$1" "$2"
  else echo "需要 curl 或 wget 来下载 fsr" >&2; exit 1; fi
}
NAME="fsr-$OS-$ARCH"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
fetch "$TMP/$NAME" "$BASE/$NAME"
fetch "$TMP/sha256sums.txt" "$BASE/sha256sums.txt"
WANT=$(awk -v n="$NAME" '$2 == n {print $1}' "$TMP/sha256sums.txt")
if command -v sha256sum >/dev/null 2>&1; then
  GOT=$(sha256sum "$TMP/$NAME" | awk '{print $1}')
else
  GOT=$(shasum -a 256 "$TMP/$NAME" | awk '{print $1}')
fi
if [ -z "$WANT" ] || [ "$WANT" != "$GOT" ]; then
  echo "下载的文件校验失败，请重试" >&2
  exit 1
fi
mkdir -p "$BIN_DIR"
mv "$TMP/$NAME" "$BIN_DIR/fsr"
chmod 755 "$BIN_DIR/fsr"
echo "已安装 $("$BIN_DIR/fsr" version) 到 $BIN_DIR/fsr"
case ":$PATH:" in
  *":$BIN_DIR:"*) ;;
  *) echo "注意：$BIN_DIR 不在 PATH 里，请把 export PATH=\"$BIN_DIR:\$PATH\" 加到 ~/.zshrc 或 ~/.bashrc" ;;
esac

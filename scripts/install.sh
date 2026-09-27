#!/bin/bash
set -e

# Repository Details
REPO="hmsoft0815/mlc_mcptester"
BINARY_NAME="mcp-tester"

# Farbcodes für die Ausgabe
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Starting installation of $BINARY_NAME...${NC}"

# 1. Architektur und OS erkennen
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# 2. Neueste Version von GitHub abrufen
echo -e "Finding latest version..."
LATEST_TAG=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_TAG" ]; then
    echo "Could not find latest release. Please check your internet connection."
    exit 1
fi

# v entfernen für den Dateinamen (GoReleaser Template)
VERSION=${LATEST_TAG#v}

# 3. Download URL konstruieren
# Format: mcp-tester_0.1.3_linux_amd64.tar.gz
FILENAME="${BINARY_NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/$LATEST_TAG/$FILENAME"

# Ziel: im Benutzerverzeichnis, ohne sudo; INSTALL_DIR überschreibt
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

echo -e "Downloading $BINARY_NAME $LATEST_TAG for $OS/$ARCH..."
curl -fsSL "$URL" -o "$TMP/$FILENAME"

# 4. Entpacken und installieren
tar -xzf "$TMP/$FILENAME" -C "$TMP" $BINARY_NAME
mkdir -p "$INSTALL_DIR"
install -m 755 "$TMP/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"

echo -e "${GREEN}Installed $BINARY_NAME $LATEST_TAG to $INSTALL_DIR/$BINARY_NAME${NC}"

# 5. PATH prüfen
case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *) echo -e "Add ${BLUE}$INSTALL_DIR${NC} to your PATH, e.g. in ~/.bashrc or ~/.zshrc:"
       echo -e "  ${BLUE}export PATH=\"$INSTALL_DIR:\$PATH\"${NC}" ;;
esac

# 6. Nächste Schritte
echo
echo "Next steps:"
if "$INSTALL_DIR/$BINARY_NAME" agent-skill --help >/dev/null 2>&1; then
    echo -e "  ${BLUE}$BINARY_NAME agent-skill install${NC}   # teach your coding agents (Claude Code, Gemini CLI, OpenCode, Codex) to use it"
fi
echo -e "  ${BLUE}$BINARY_NAME inspect -c \"<command that starts your MCP server>\"${NC}"
echo "  Docs: https://github.com/$REPO#readme"

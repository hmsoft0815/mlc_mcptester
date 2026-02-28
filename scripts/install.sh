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

echo -e "Downloading $BINARY_NAME $LATEST_TAG for $OS/$ARCH..."
curl -L "$URL" -o "$FILENAME"

# 4. Entpacken
echo -e "Extracting..."
tar -xzf "$FILENAME" $BINARY_NAME

# 5. Aufräumen
rm "$FILENAME"

# 6. Abschlussmeldung
chmod +x $BINARY_NAME
mv $BINARY_NAME /tmp/$BINARY_NAME # Sicherstellen, dass wir im Pfad schieben können falls gewünscht

echo -e "${GREEN}Successfully downloaded $BINARY_NAME!${NC}"
echo -e "To install it globally, run:"
echo -e "  ${BLUE}sudo mv /tmp/$BINARY_NAME /usr/local/bin/${NC}"
echo -e "Or use it locally from: ${BLUE}/tmp/$BINARY_NAME${NC}"

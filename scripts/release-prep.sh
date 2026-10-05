#!/usr/bin/env bash
# Writes the version from VERSION into every file that names it, so a release
# needs only VERSION and the CHANGELOG entries under "## [Unreleased]".
# The binary (ldflags), the installer (makensis) and GoReleaser (git tag) take
# the version on their own; this covers the rest. Safe to run twice.
set -euo pipefail
cd "$(dirname "$0")/.."

version=$(tr -d '[:space:]' < VERSION)
if ! [[ $version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "VERSION is not x.y.z: '$version'" >&2
  exit 1
fi
today=$(date +%F)
semver='[0-9]+\.[0-9]+\.[0-9]+'

# CHANGELOG: the open section becomes the release
if grep -q '^## \[Unreleased\]$' CHANGELOG.md; then
  sed -i "s/^## \[Unreleased\]$/## [$version] - $today/" CHANGELOG.md
  echo "CHANGELOG.md: [Unreleased] -> [$version] - $today"
elif grep -q "^## \[$version\]" CHANGELOG.md; then
  echo "CHANGELOG.md: [$version] already there"
else
  echo "CHANGELOG.md has neither [Unreleased] nor [$version] — write the entries first" >&2
  exit 1
fi

# SPEC_COVERAGE: only the tester version; the "as of" date stands for a spec
# re-check and is changed by hand when one is done
for f in docs/SPEC_COVERAGE.md docs/SPEC_COVERAGE.de.md; do
  sed -i -E "s/mcp-tester \*\*$semver\*\*/mcp-tester **$version**/" "$f"
  echo "$f: mcp-tester $version"
done

# Product page: a separate private clone, absent in public checkouts
meta=mlcprodweb/meta.yaml
if [ -f "$meta" ]; then
  sed -i -E "s/^version: \"$semver\"/version: \"$version\"/; s/mcp-tester-setup-v$semver-windows-amd64\.exe/mcp-tester-setup-v$version-windows-amd64.exe/" "$meta"
  echo "$meta: version and installer $version"
else
  echo "$meta not found — product page skipped"
fi

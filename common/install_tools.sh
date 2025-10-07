#!/usr/bin/env bash
set -euo pipefail

# ------------------------------------------------------------------
# install-tools.sh
# Installs Go developer tools declared in tools.go
# ------------------------------------------------------------------

echo "🧩 Installing Go tools..."

# Ensure we run from the module root
cd "$(dirname "$0")"

# Ensure go.mod exists
if [ ! -f "go.mod" ]; then
  echo "❌ Error: go.mod not found. Please run this script from your module root."
  exit 1
fi

# Use the tools build tag to list import paths
TOOL_PKGS=$(go list -f '{{.ImportPath}}' -tags=tools 2>/dev/null || true)

if [ -z "$TOOL_PKGS" ]; then
  echo "⚠️  No tools found. Make sure you have a tools.go file with '_ \"import/path\"' entries."
  exit 0
fi

for pkg in $TOOL_PKGS; do
  echo "🔧 Installing $pkg ..."
  go install "$pkg@latest"
done

echo "✅ All tools installed successfully!"
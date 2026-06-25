#!/usr/bin/env bash
# Validates that the create-vhs skill has the binaries it needs before any
# storyboard, compile, or record work begins. Exit non-zero on the first hard miss.
set -euo pipefail

missing=0
report_missing() {
  local bin="$1" hint="$2"
  echo "MISSING: $bin"
  echo "  install: $hint"
  missing=1
}

check() {
  local bin="$1" hint="$2"
  local path
  path="$(command -v "$bin" 2>/dev/null || true)"
  if [ -z "$path" ] || [ ! -x "$path" ]; then
    report_missing "$bin" "$hint"
  else
    echo "OK: $bin -> $path"
  fi
}

# Soft check: warn but do not fail (the binary is optional or has a fallback).
check_soft() {
  local bin="$1" hint="$2"
  local path
  path="$(command -v "$bin" 2>/dev/null || true)"
  if [ -z "$path" ] || [ ! -x "$path" ]; then
    echo "WARN: $bin not found ($hint)"
  else
    echo "OK: $bin -> $path"
  fi
}

echo "create-vhs prereq check"
echo "-----------------------"

# Hard requirements.
check vhs  "go install github.com/charmbracelet/vhs@latest  (also needs ttyd + ffmpeg on PATH)"
check node "https://nodejs.org  (v18+; the compiler runs via 'npx tsx')"

# Soft requirements.
check_soft ffmpeg "brew install ffmpeg | apt install ffmpeg  (required by vhs for .mp4 output; .gif works without it)"
check_soft ttyd   "brew install ttyd | apt install ttyd  (vhs needs it at record time)"

# rudder-cli: accept either a PATH install or the repo-local ./bin/rudder-cli build.
repo_root="$(git rev-parse --show-toplevel 2>/dev/null || echo .)"
if command -v rudder-cli >/dev/null 2>&1; then
  echo "OK: rudder-cli -> $(command -v rudder-cli)"
elif [ -x "$repo_root/bin/rudder-cli" ]; then
  echo "OK: rudder-cli -> $repo_root/bin/rudder-cli (repo build; add 'export PATH=\"$repo_root/bin:\$PATH\"' to your scene 'setup')"
else
  echo "WARN: rudder-cli not found on PATH or at $repo_root/bin/rudder-cli"
  echo "  build it: make build   (then reference ./bin/rudder-cli via scene 'setup')"
fi

if [ "$missing" -ne 0 ]; then
  echo ""
  echo "Prerequisite check FAILED. Install the missing binaries above and re-run."
  exit 1
fi

echo ""
echo "All hard prerequisites satisfied."

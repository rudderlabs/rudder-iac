#!/usr/bin/env bash
# Render one (or all) demo dirs to BOTH recording.gif and recording.mp4.
# Run from the repo root:  bash docs/demos/record-demos.sh [demo-dir ...]
# With no args it records every docs/demos/2*-* dir.
#
# Toolchain assumptions (override via env if your paths differ):
#   NODE_BIN  — a node >= 18 (npx tsx needs it); default: newest nvm v20.x
#   VHS_BIN   — charmbracelet/vhs; default: ~/go/bin/vhs
set -euo pipefail

root="$(git rev-parse --show-toplevel)"
compile="$root/.claude/skills/create-vhs/scripts/compile.ts"

NODE_BIN="${NODE_BIN:-$(ls -d "$HOME"/.nvm/versions/node/v20.* 2>/dev/null | tail -1)/bin/node}"
VHS_BIN="${VHS_BIN:-$HOME/go/bin/vhs}"
NPX="$(dirname "$NODE_BIN")/npx"
export PATH="$(dirname "$NODE_BIN"):$(dirname "$VHS_BIN"):$PATH"

dirs=("$@")
if [ ${#dirs[@]} -eq 0 ]; then
  # Portable (bash 3.2): glob the dated demo dirs directly.
  dirs=("$root"/docs/demos/2*-*/)
fi

# Build the CLI + the seed helper so ./bin is current (the configs put it on PATH).
( cd "$root" && go build -o ./bin/rudder-cli ./cli/cmd/rudder-cli )
( cd "$root" && go build -o ./bin/seed-unmanaged ./docs/demos/scripts/seed-unmanaged )

for d in "${dirs[@]}"; do
  d="${d%/}"
  [ -f "$d/scenes.config.ts" ] || { echo "skip (no scenes.config.ts): $d"; continue; }
  for fmt in gif mp4; do
    echo "==> $d  →  recording.$fmt"
    DEMO_OUT="recording.$fmt" "$NPX" tsx "$compile" "$d"
    out="$d/recording.$fmt"
    # vhs occasionally races ttyd and produces a 0-byte file; retry up to 3x.
    for attempt in 1 2 3; do
      rm -f "$out"
      "$VHS_BIN" "$d/tape.tape" || true
      if [ -s "$out" ]; then break; fi
      echo "   (attempt $attempt produced no output; retrying)"
    done
    [ -s "$out" ] || echo "   WARN: $out still empty after retries"
  done
done

echo "Done. Recordings: docs/demos/*/recording.{gif,mp4}"

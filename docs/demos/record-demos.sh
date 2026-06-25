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
  # Record the GIF live, then transcode it to MP4 with ffmpeg. We do NOT record
  # the MP4 live: live h264 encoding starves ttyd and drops/scrambles keystrokes
  # on longer demos. Transcoding the finished GIF gives a faithful, clean MP4 and
  # avoids running the (mutating) demo a second time.
  echo "==> $d  →  recording.gif"
  DEMO_OUT="recording.gif" "$NPX" tsx "$compile" "$d"
  gif="$d/recording.gif"
  for attempt in 1 2 3; do
    rm -f "$gif"
    "$VHS_BIN" "$d/tape.tape" || true
    if [ -s "$gif" ]; then break; fi
    echo "   (attempt $attempt produced no output; retrying)"
  done
  if [ -s "$gif" ]; then
    echo "==> $d  →  recording.mp4 (transcoded from gif)"
    ffmpeg -y -i "$gif" -movflags +faststart -pix_fmt yuv420p \
      -vf "scale=trunc(iw/2)*2:trunc(ih/2)*2" "$d/recording.mp4" >/dev/null 2>&1 \
      || echo "   WARN: mp4 transcode failed"
  else
    echo "   WARN: $gif still empty after retries"
  fi
done

# Defensive: a recording race can occasionally skip a demo's final `delete`
# scene, leaving a throwaway behind. Sweep them so the workspace is left clean.
bash "$root/docs/demos/cleanup.sh" || true

echo "Done. Recordings: docs/demos/*/recording.{gif,mp4}"

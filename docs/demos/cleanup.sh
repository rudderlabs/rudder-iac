#!/usr/bin/env bash
# Remove any throwaway demo resources left in the workspace. The build/adopt
# demos self-clean, but a vhs typing race can occasionally skip the final
# `delete` scene, so this is the deterministic safety net. Safe to run anytime.
set -uo pipefail

root="$(git rev-parse --show-toplevel)"
cli="$root/bin/rudder-cli"
seed="$root/bin/seed-unmanaged"

# Managed demo sources, addressable by their external id.
for id in demo-orders-source orders-pipeline; do
  if "$cli" delete event-stream-source "$id" --confirm >/dev/null 2>&1; then
    echo "removed managed source: $id"
  fi
done

# Unmanaged demo sources (no external id) — delete by remote id via the api.
# Remote ids are 20+ char alnum tokens; pull them off any line naming a demo source.
"$cli" get event-stream-source 2>/dev/null \
  | grep -E "Demo Orders Source|Legacy Orders Source" \
  | grep -oE '[0-9A-Za-z]{20,}' \
  | while read -r rid; do
      [ -n "$rid" ] || continue
      if "$seed" -delete "$rid" >/dev/null 2>&1; then
        echo "removed unmanaged source: $rid"
      fi
    done

echo "cleanup done"

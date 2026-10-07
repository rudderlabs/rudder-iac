#!/usr/bin/env bash
# Record a generated demo to an asciicast, stamping its provenance into the
# cast header so a recording can never be mistaken for a different branch or
# a different backend.
#
# asciinema is not installed in this environment, so recording goes through
# scripts/demo_cast.py (stdlib pty) instead of `asciinema rec`. That script
# accepts the demo's manifest.json directly and folds it into the header
# itself, so — unlike the asciinema version of this script — there is no
# separate jq merge step afterwards.
#
# Usage: scripts/demo-cast.sh TestAccountsApply [outdir]
set -euo pipefail

TEST="${1:?usage: demo-cast.sh <TestName> [outdir]}"
OUT="${2:-casts}"
DEMO_DIR="demos/$TEST"
CAST="$OUT/$TEST.cast"

[ -x "$DEMO_DIR/demo.sh" ] || {
	echo "refusing: no generated demo at $DEMO_DIR/demo.sh — run 'make demo DEMO_TEST=$TEST' first" >&2
	exit 1
}

# Matches api/client/client.go's BASE_URL fallback and the demo-record guard
# in the Makefile: the recorded script's first act is a workspace-wiping
# `destroy --confirm=false`, so this must refuse before anything runs, the
# same way demo-record does.
PROD_API_HOST="api.rudderstack.com"

if [ -z "${RUDDERSTACK_API_URL:-}" ]; then
	echo "refusing: RUDDERSTACK_API_URL is unset, so the recorded run would target $PROD_API_HOST." >&2
	echo "  demo.sh's first command is 'destroy --confirm=false' — it wipes the target workspace." >&2
	echo "  Export the same backend the demo was generated against, e.g.:" >&2
	echo "    export RUDDERSTACK_API_URL=http://localhost:15580" >&2
	exit 1
fi
case "$RUDDERSTACK_API_URL" in
*$PROD_API_HOST*)
	echo "refusing: RUDDERSTACK_API_URL points at production, and this recording wipes its workspace." >&2
	exit 1
	;;
esac

command -v python3 >/dev/null || {
	echo "refusing: python3 not found — the recorder is scripts/demo_cast.py (stdlib only)" >&2
	exit 1
}

mkdir -p "$OUT"
rm -f "$CAST"

# RUDDER_DEMO_RECORDING suppresses the generated script's annotation footer:
# a prompt (or an exit 3) in the middle of a cast is the one thing nobody
# wants to watch.
RUDDER_DEMO_RECORDING=1 python3 "$(dirname "$0")/demo_cast.py" \
	"$DEMO_DIR/demo.sh" "$CAST" \
	--manifest "$DEMO_DIR/manifest.json"

echo "wrote $CAST"
python3 -c '
import json, sys
with open(sys.argv[1]) as f:
    header = json.loads(f.readline())
m = header.get("rudderDemo", {})
git = m.get("git", {})
backend = m.get("backend", {})
ref = git.get("ref")
sha = git.get("sha")
profile = backend.get("profile")
kind = backend.get("kind")
state = "annotated" if m.get("annotated") else "derived"
print("  {} @ {} · {} ({}) · {}".format(ref, sha, profile, kind, state))
' "$CAST"

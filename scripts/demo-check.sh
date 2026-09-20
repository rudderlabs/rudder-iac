#!/usr/bin/env bash
# Regenerate demos/<TEST>/ from its committed journal + events into a temp
# directory, then diff that against what's actually committed. Any
# difference is drift: either demo.sh (or a fixture) was hand-edited, or the
# generator changed without anyone re-running `make demo-generate`.
#
# This does NOT replay demo.sh — there is no hook in the shipped rudder-cli
# to journal against, so a replay's journal would be empty and have nothing
# to compare. Instead demogen runs again over the same recorded inputs,
# which is deterministic, so the committed demo directory is exactly what it
# should still look like.
#
# recordedAt, the git block, and cliVersion (== git describe, so it moves
# with every commit) are excluded from the comparison: they are wall-clock-
# and commit-derived and change on every regeneration by design. Comparing
# them would fail this check on every run, drift or not, which is the one
# thing that would make the check useless.
#
# No backend, no network: demogen only reads the journal and event stream
# already on disk. It never invokes rudder-cli or talks to an API.
#
# Usage: scripts/demo-check.sh TestAccountsApply
set -euo pipefail

TEST="${1:?usage: demo-check.sh <TestName>}"
DEMO_DIR="demos/$TEST"
JOURNAL="demos/$TEST.jsonl"
EVENTS="demos/$TEST.events.json"
DEMO_PROFILE="${DEMO_PROFILE:-mini}"

[ -f "$JOURNAL" ] || { echo "refusing: no committed journal at $JOURNAL" >&2; exit 1; }
[ -f "$EVENTS" ] || { echo "refusing: no committed events at $EVENTS" >&2; exit 1; }
[ -d "$DEMO_DIR" ] || { echo "refusing: no committed demo at $DEMO_DIR — run 'make demo-generate DEMO_TEST=$TEST' first" >&2; exit 1; }
[ -f "demos/profiles/$DEMO_PROFILE.env" ] || { echo "refusing: no such profile demos/profiles/$DEMO_PROFILE.env" >&2; exit 1; }

# The binary path lives only in the journal's own recorded argv (it was a
# t.TempDir the original test run built into). Rediscover it the same way
# `make demo-generate` does — there is nothing to build here to get a fresh
# one from.
BIN="$(grep -o '"argv":\["[^"]*"' "$JOURNAL" | sed -E 's/.*\["//;s/"$//' | grep -E '/rudder-cli(\.exe)?$' | head -1)"
if [ -z "$BIN" ]; then
	echo "refusing: no rudder-cli invocation found in $JOURNAL to discover the CLI binary path from" >&2
	exit 1
fi

set -a
# shellcheck disable=SC1090
. "demos/profiles/$DEMO_PROFILE.env"
set +a

TMP_OUT="$(mktemp -d -t demo-check-XXXXXX)"
trap 'rm -rf "$TMP_OUT"' EXIT

go run ./cli/tests/demo/cmd/demogen \
	-journal "$JOURNAL" \
	-events "$EVENTS" \
	-out "$TMP_OUT" \
	-repo-root . \
	-bin "$BIN" \
	-profile "$RUDDER_DEMO_PROFILE" \
	-backend-kind "$RUDDER_DEMO_BACKEND_KIND" \
	-api-url "$RUDDERSTACK_API_URL" >/dev/null

GOT_DIR="$TMP_OUT/$TEST"
[ -d "$GOT_DIR" ] || { echo "refusing: demogen did not emit $TEST from $JOURNAL + $EVENTS" >&2; exit 1; }

status=0

if ! diff -u --label "committed demo.sh" --label "regenerated demo.sh" \
	"$DEMO_DIR/demo.sh" "$GOT_DIR/demo.sh" >"$TMP_OUT/demo.sh.diff"; then
	echo "demo.sh has drifted from what the journal produces:"
	cat "$TMP_OUT/demo.sh.diff"
	echo
	status=1
fi

# The fixture tree: everything demogen copies from testdata next to the
# script. demo.sh, manifest.json, README.md and ANNOTATE.md are all
# generated files, handled here or deliberately skipped (see the header).
if ! diff -ru \
	--label "committed $TEST" --label "regenerated $TEST" \
	--exclude=demo.sh --exclude=manifest.json --exclude=README.md --exclude=ANNOTATE.md \
	"$DEMO_DIR" "$GOT_DIR" >"$TMP_OUT/fixtures.diff"; then
	echo "the fixture tree has drifted from what the journal produces:"
	cat "$TMP_OUT/fixtures.diff"
	echo
	status=1
fi

# manifest.json as a whole carries recordedAt/git/cliVersion (see header), so
# compare only its steps and gaps — stdlib json, no new dependency.
if ! python3 - "$DEMO_DIR/manifest.json" "$GOT_DIR/manifest.json" <<'PY'
import json
import subprocess
import sys
import tempfile


def steps_and_gaps(path):
    with open(path) as f:
        m = json.load(f)
    return {"steps": m.get("steps", []), "gaps": m.get("gaps", [])}


want_path, got_path = sys.argv[1], sys.argv[2]
want = steps_and_gaps(want_path)
got = steps_and_gaps(got_path)

if want == got:
    sys.exit(0)

print("manifest.json's steps/gaps have drifted from what the journal produces:")
sys.stdout.flush()

with tempfile.NamedTemporaryFile("w", suffix=".json", delete=False) as wf, \
        tempfile.NamedTemporaryFile("w", suffix=".json", delete=False) as gf:
    json.dump(want, wf, indent=2, sort_keys=True)
    wf.write("\n")
    json.dump(got, gf, indent=2, sort_keys=True)
    gf.write("\n")

subprocess.run([
    "diff", "-u",
    "--label", "committed manifest.json (steps/gaps)", wf.name,
    "--label", "regenerated manifest.json (steps/gaps)", gf.name,
])
print()
sys.exit(1)
PY
then
	status=1
fi

if [ "$status" -ne 0 ]; then
	echo "$TEST has drifted from demos/$TEST.jsonl + demos/$TEST.events.json." >&2
	echo "Either demo.sh (or a fixture) was hand-edited, or the generator changed" >&2
	echo "without regenerating. Fix with: make demo-generate DEMO_TEST=$TEST" >&2
	exit 1
fi

echo "no drift: $TEST matches demos/$TEST.jsonl + demos/$TEST.events.json"

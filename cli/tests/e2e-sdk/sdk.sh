#!/usr/bin/env bash
# Sends one 'SDK Probe' track from each real SDK (Node with gzip, Python, Go,
# JS v3 in Chromium) to one listener, each with its own write key, and
# asserts that each one arrived once with its bytes intact. Run `npm ci`,
# `npx playwright install chromium` and `pip install -r requirements.txt`
# first. PYTHON names the interpreter (default python3).
set -euo pipefail
cd "$(dirname "$0")"
# shellcheck source=listen.sh
. ./listen.sh

trap on_exit EXIT
start_listener

node node.mjs "$url"
"${PYTHON:-python3}" sdk_probe.py "$url"
(cd go && go run . "$url")
node jsv3.mjs "$url"

wait_for 4 'SDK Probe'
if grep '^refused:' "$log" >&2; then exit 1; fi
jq -e -s '[.[] | select(.event == "SDK Probe")] |
  length == 4 and ([.[].properties.sdk] | sort) == ["go","jsv3","node","python"] and all(.[]; .properties.n == 1)' "$events"
# stdout has no write key, so the per-key check reads the store.
curl -fsS "$url/_dev/v1/events?view=counts&since=$cur" |
  jq -e '[.summary.byWriteKey["node","python","go","jsv3"].events] | all(. == 1)'
echo "all four SDKs: ok"

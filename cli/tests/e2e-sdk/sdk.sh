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

rudder-cli local event-stream events summary --url "$url" --server-id "$sid" --since "$cur" --event 'SDK Probe' --min 4 --wait 30s --json |
  jq -e '.timedOut == false and .summary.byEvent["SDK Probe"] == 4 and ([.summary.byWriteKey["node","python","go","jsv3"].events] | all(. == 1))'
rudder-cli local event-stream events summary --url "$url" --since "$cur" --json |
  jq -e '.summary.requests.failed == 0'
rudder-cli local event-stream events list --url "$url" --since "$cur" --event 'SDK Probe' --fields properties --json |
  jq -e -s 'length == 4 and ([.[].properties.sdk] | sort) == ["go","jsv3","node","python"] and all(.[]; .properties.n == 1)'
echo "all four SDKs: ok"

#!/usr/bin/env bash
# Runs the fixture app against a fresh listener RUNS times (default 10) and
# asserts counts, failures and bytes with jq -e. Then it runs the app once
# with DEFECT=wrong-type and requires the byte check to fail, which proves the
# check can fail. Run `npm ci` and `npx playwright install chromium` here first.
set -euo pipefail
cd "$(dirname "$0")"
# shellcheck source=listen.sh
. ./listen.sh

# run_app starts the fixture server with the given defect, clicks the page
# and sends the backend event.
run_app() {
  local out
  out=$(mktemp)
  WRITE_KEY=web DATA_PLANE_URL="$url" DEFECT="$1" node server.mjs >"$out" &
  web=$!
  until [ -s "$out" ]; do kill -0 "$web" 2>/dev/null || exit 1; sleep 0.1; done
  node fire.mjs "$(jq -r .url "$out")"
  node backend.mjs "$url"
}

# sent_is_right is the byte check: one Suggestion Sent, position a number.
sent_is_right() {
  rudder-cli local event-stream events list --url "$url" --since "$cur" --event 'Suggestion Sent' --fields properties --json |
    jq -e -s 'length == 1 and .[0].properties.position == 2 and .[0].properties.edited == false'
}

stop() {
  kill "$pid" "$web" 2>/dev/null || true
  wait "$pid" "$web" 2>/dev/null || true
  unset pid web url
}

trap on_exit EXIT
for run in $(seq "${RUNS:-10}"); do
  start_listener
  run_app none

  rudder-cli local event-stream events summary --url "$url" --server-id "$sid" --since "$cur" \
    --event 'Suggestion Shown' --event 'Suggestion Clicked' --event 'Suggestion Sent' --event 'Order Completed' \
    --min 4 --wait 30s --json |
    jq -e '.timedOut == false and ([.summary.byEvent[]] | all(. == 1)) and .summary.rejected.events == 0'
  # Filters narrow every count, so the failure and control checks run without them.
  rudder-cli local event-stream events summary --url "$url" --since "$cur" --json |
    jq -e '.summary.requests.failed == 0 and .summary.control.sourceConfig >= 1'
  sent_is_right

  stop
  echo "run $run: ok"
done

start_listener
run_app wrong-type
rudder-cli local event-stream events summary --url "$url" --since "$cur" --event 'Suggestion Sent' --min 1 --wait 30s --json |
  jq -e '.timedOut == false'
if sent_is_right; then
  echo "the byte check did not catch DEFECT=wrong-type" >&2
  exit 1
fi
stop
echo "DEFECT=wrong-type: the byte check failed, as it must"

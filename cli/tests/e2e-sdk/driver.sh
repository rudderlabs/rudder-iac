#!/usr/bin/env bash
# Runs the fixture app against a fresh listener RUNS times (default 10) and
# asserts counts, failures and bytes on its stdout with jq -e. Then it runs the app once
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

# sent_is_right is the byte check on stdout: one Suggestion Sent, position a
# number.
sent_is_right() {
  jq -e -s '[.[] | select(.event == "Suggestion Sent")] |
    length == 1 and .[0].properties.position == 2 and .[0].properties.edited == false' "$events"
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

  wait_for 4 'Suggestion Shown' 'Suggestion Clicked' 'Suggestion Sent' 'Order Completed'
  for name in 'Suggestion Shown' 'Suggestion Clicked' 'Suggestion Sent' 'Order Completed'; do
    [ "$(count "$name")" -eq 1 ] || { echo "want one $name, got $(count "$name")" >&2; exit 1; }
  done
  if grep '^refused:' "$log" >&2; then exit 1; fi
  # stdout carries events only; the SDK's config call is a control request.
  curl -fsS "$url/_dev/v1/events?view=counts&since=$cur" | jq -e '.summary.control.sourceConfig >= 1'
  sent_is_right
  # The store holds the same events as stdout.
  rudder-cli sources events --url "$url" --server-id "$sid" --since "$cur" --event 'Suggestion Sent' --json |
    jq -e -s 'length == 1'

  stop
  echo "run $run: ok"
done

start_listener
run_app wrong-type
wait_for 1 'Suggestion Sent'
if sent_is_right; then
  echo "the byte check did not catch DEFECT=wrong-type" >&2
  exit 1
fi
stop
echo "DEFECT=wrong-type: the byte check failed, as it must"

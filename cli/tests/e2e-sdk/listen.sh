# shellcheck shell=bash
# Sourced by driver.sh and sdk.sh. Needs rudder-cli, jq and curl on PATH.

export RUDDERSTACK_CLI_EXPERIMENTAL=true
export RUDDERSTACK_X_SOURCES_LISTEN=true
export RUDDERSTACK_CLI_TELEMETRY_DISABLED=true

# start_listener starts sources listen --store on a free port. It sets events
# and log to the files that hold its stdout and stderr, and pid, url, sid and
# cur from the ready line. It exits with the log when the listener stops before
# it is ready.
start_listener() {
  local d ready
  d=$(mktemp -d)
  events="$d/events" log="$d/log"
  rudder-cli sources listen --port 0 --store >"$events" 2>"$log" &
  pid=$!
  until grep -q '^{"ready":true' "$log"; do
    kill -0 "$pid" 2>/dev/null || { cat "$log" >&2; exit 1; }
    sleep 0.1
  done
  ready=$(grep -m1 '^{"ready":true' "$log")
  url=$(jq -r .url <<<"$ready")
  # shellcheck disable=SC2034 # read by the sourcing script
  sid=$(jq -r .serverId <<<"$ready")
  cur=$(jq -r .cursor <<<"$ready")
}

# count prints how many events on stdout so far have one of the given names.
# fromjson? skips a line the listener is still writing.
count() {
  jq -nR '$ARGS.positional as $names | [inputs | fromjson? | select(.event as $e | $names | index($e))] | length' \
    --args "$@" <"$events"
}

# wait_for waits up to 30s until stdout holds at least N events with one of
# the given names.
wait_for() {
  local n=$1 i
  shift
  for i in $(seq 150); do
    [ "$(count "$@")" -ge "$n" ] && return 0
    sleep 0.2
  done
  echo "timed out waiting for $n of: $*" >&2
  return 1
}

# on_exit prints the request records to stderr when the script fails, then
# stops the processes it started.
on_exit() {
  local rc=$?
  if [ "$rc" -ne 0 ] && [ -n "${log:-}" ]; then
    cat "$log" >&2 || true
  fi
  if [ "$rc" -ne 0 ] && [ -n "${url:-}" ]; then
    curl -sS "$url/_dev/v1/requests?since=${cur:-0}&kind=all&view=compact" >&2 || true
  fi
  # shellcheck disable=SC2086 # the PIDs split on purpose
  kill ${pid:-} ${web:-} 2>/dev/null || true
  exit "$rc"
}

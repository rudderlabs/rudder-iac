# shellcheck shell=bash
# Sourced by driver.sh and sdk.sh. Needs rudder-cli, jq and curl on PATH.

export RUDDERSTACK_CLI_EXPERIMENTAL=true
export RUDDERSTACK_X_SOURCES_LISTEN=true
export RUDDERSTACK_CLI_TELEMETRY_DISABLED=true

# start_listener starts sources listen --store on a free port and sets pid, url,
# sid and cur from its ready line on stderr. It exits with the listener's log when the listener
# stops before it is ready.
start_listener() {
  local d ready
  d=$(mktemp -d)
  rudder-cli sources listen --port 0 --store >"$d/events" 2>"$d/log" &
  pid=$!
  until grep -q '^{"ready":true' "$d/log"; do
    kill -0 "$pid" 2>/dev/null || { cat "$d/log" >&2; exit 1; }
    sleep 0.1
  done
  ready=$(grep -m1 '^{"ready":true' "$d/log")
  url=$(jq -r .url <<<"$ready")
  # shellcheck disable=SC2034 # read by the sourcing script
  sid=$(jq -r .serverId <<<"$ready")
  cur=$(jq -r .cursor <<<"$ready")
}

# on_exit prints the request records to stderr when the script fails, then
# stops the processes it started.
on_exit() {
  local rc=$?
  if [ "$rc" -ne 0 ] && [ -n "${url:-}" ]; then
    curl -sS "$url/_dev/v1/requests?since=${cur:-0}&kind=all&view=compact" >&2 || true
  fi
  # shellcheck disable=SC2086 # the PIDs split on purpose
  kill ${pid:-} ${web:-} 2>/dev/null || true
  exit "$rc"
}

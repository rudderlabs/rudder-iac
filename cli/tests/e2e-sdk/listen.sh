# shellcheck shell=bash
# Sourced by driver.sh and sdk.sh. Needs rudder-cli, jq and curl on PATH.

export RUDDERSTACK_CLI_EXPERIMENTAL=true
export RUDDERSTACK_X_DEV_LISTEN=true
export RUDDERSTACK_CLI_TELEMETRY_DISABLED=true

# start_listener starts dev listen on a free port and sets pid, url, sid and
# cur from its ready line. It exits with the listener's log when the listener
# stops before it is ready.
start_listener() {
  local d
  d=$(mktemp -d)
  rudder-cli dev listen --port 0 >"$d/ready" 2>"$d/log" &
  pid=$!
  until [ -s "$d/ready" ]; do
    kill -0 "$pid" 2>/dev/null || { cat "$d/log" >&2; exit 1; }
    sleep 0.1
  done
  url=$(jq -r .url "$d/ready")
  # shellcheck disable=SC2034 # read by the sourcing script
  sid=$(jq -r .serverId "$d/ready")
  cur=$(jq -r .cursor "$d/ready")
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

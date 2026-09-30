Capture the events your app sends to RudderStack on a local server, then read what arrived.
dev listen accepts requests exactly like RudderStack ingestion and keeps each one in memory.
Point an app at it, use the app, then ask what arrived. No deploy and no workspace are needed.

The loop:

  1. Start the listener in the background: rudder-cli dev listen --port 0 > ready.json &
  2. Read the url, serverId and cursor from the ready line in ready.json.
  3. Point the app's SDK at the url and use the app.
  4. Read what arrived: rudder-cli dev events --url URL

Experimental: enable with RUDDERSTACK_CLI_EXPERIMENTAL=true RUDDERSTACK_X_DEV_LISTEN=true.
In an interactive shell you can also export RUDDERSTACK_CLI_EXPERIMENTAL=true once and run
rudder-cli experimental enable devListen.

## Commands

  dev listen                Run the capture server in the foreground
  dev events                Counts and a diagnosis of what arrived
  dev events list           The events, one line each, with the same counts
  dev requests list         Raw requests, including rejected and control requests
  dev requests show <seq>   One request in full

## Start and stop

dev listen runs in the foreground. When the port is bound it prints one JSON line on stdout,
then nothing more on stdout:

  {"ready":true,"url":"http://127.0.0.1:4321","port":4321,"bind":"127.0.0.1","pid":81234,
   "serverId":"4d35ce0ebaf262a0","cursor":0,"ui":"http://127.0.0.1:4321/_dev/ui/", ...}

Keep url, serverId and cursor. Every read command takes the url with --url or the
RUDDERSTACK_DEV_URL variable. Let the shell run it in the background and wait for the line:

  rudder-cli dev listen --port 4321 > ready.json 2> listen.log &
  until [ -s ready.json ]; do sleep 0.1; done
  export RUDDERSTACK_DEV_URL=$(jq -r .url ready.json)

Stop it with Ctrl-C in the foreground or with kill "$(jq -r .pid ready.json)". Both drain open
requests and exit 0. --port 0 lets the system pick a free port. A fixed --port that is taken
fails at start with port_in_use and exit 1. Use --bind 0.0.0.0 only inside a container; the
listener warns when it is not on loopback.

## Point your app at it

The url is the data plane URL and, for the browser SDK, the config URL too. By default any
non-empty write key works. dev listen --write-key KEY accepts only the keys you list and
rejects every other key with 401; that is an allowlist, not a filter.

  Browser SDK: load('dev', URL, { configUrl: URL }). Without configUrl the SDK asks
  api.rudderstack.com and never reaches the listener.
  Server SDKs (Node, Go, Python): set dataPlaneUrl to URL and flush before the process exits.

When events do not arrive, check that the data plane URL points at the listener on the same
port, that the config URL points at it too, and that the app's own analytics switch is on.
Build-time settings (Vite, Next.js) need a restart after a change, and a fixed --port.

## Read what arrived

Start cheap and ask for more only when the cheaper answer does not settle the question.

  0  rudder-cli dev events --since 0 --json
     Counts by event, write key and source, and a diagnosis. events is empty. About 0.7 KB.
  1  rudder-cli dev events list --since 0 --json
     Adds one line per event: seq, time, type, name, write key, status. About 120 bytes each.
  2  rudder-cli dev events list --since 0 --event 'Order Completed' --view compact --json
     The payload without the SDK context keys. About 0.5 KB per event.
  3  rudder-cli dev events list --since 0 --event 'Order Completed' --fields properties --json
     Only the paths you name. --fields replaces --view. About 110 bytes per event.
  4  rudder-cli dev requests show 8 --json
     One request in full: headers, raw body, response. About 3 KB.

Every dev events answer has the same keys: apiVersion, serverId, since, cursor, hasMore,
timedOut, summary, view, omitted, truncated and events. summary holds requests, events,
byEvent, byWriteKey, control, bySource and diagnosis. Each name you pass with --event and each
key you pass with --write-key is in byEvent or byWriteKey, with 0 when nothing arrived. When a
view hides something, omitted.next is the command that shows it. A page is capped at 24000
bytes; --max-bytes 0 lifts the cap.

seq counts every request, so gaps in seq are control requests (source config, CORS preflight),
not lost events. Property values are shown exactly as the app sent them.

## Filter

Filters run on the listener, before the cap, and narrow the summary and the events alike.

  --since N                  Only requests after cursor N. Read the cursor before you act
  --event NAME               Exact event name; repeat for any of several
  --type TYPE                track, identify, page, screen, group or alias
  --user-id ID               One user's events; --anonymous-id ID likewise
  --status-code CODE         The HTTP status the listener answered
  --write-key KEY            One service, by the key it sent; repeat for any of several
  --wait 30s --min 1         Hold the call until enough events match, up to 110s
  --kind KIND                dev requests list only: ingestion, control or all
  --failed                   dev requests list only: rejected requests

## Checks with jq -e

There is no assertion flag. jq -e exits 1 when the answer is false, so each line is a CI step.

Did the event arrive exactly once:

  rudder-cli dev events --since "$cur" --event 'Order Completed' --json | jq -e '.summary.byEvent["Order Completed"] == 1'

An absent event shows 0:

  rudder-cli dev events --since "$cur" --event 'Order Refunded' --json | jq -e '.summary.byEvent["Order Refunded"] == 0'

Does a property have the right type:

  rudder-cli dev events list --since "$cur" --event 'Order Completed' --fields properties --json | jq -e 'all(.events[]; .properties.total | type == "number")'

Wait for an event the app sends later:

  rudder-cli dev events --since "$cur" --event 'Order Completed' --wait 30s --json | jq -e '.timedOut == false'

The raw body the app sent, when a value looks wrong:

  rudder-cli dev requests show 8 --fields request.body --json | jq -r '.request.body'

One event per line for grep, head and wc:

  rudder-cli dev events list --since 0 --json | jq -c '.events[]'

## Several services, one listener

Give each service its own write key, as each RudderStack source has one, then slice by it:

  rudder-cli dev events --since "$cur" --write-key api-key --write-key worker-key --json | jq '.summary.byWriteKey'

A key other than dev is stored masked (first and last 4 characters) and reported that way.

## Runs in parallel

Each listener has its own port and serverId and shares nothing: no state file, no lock, no
discovery. Captures stay in memory. Start one listener per app or test job with --port 0 and
pass --server-id with every read, so a read never hits another or a restarted listener; it
fails with server_changed instead.

## When nothing arrives

Run dev events and read summary.diagnosis. Each entry names the next command.

  nothing_received      No request reached the listener. Check the app's URLs and port
  no_browser_traffic    Server SDKs sent requests, the browser none. Check configUrl
  preflight_only        The browser sent CORS preflights only
  sdk_config_rejected   The browser SDK could not load its config
  sdk_loaded_no_events  The SDK loaded its config but sent no events
  auth_rejected         A write key was missing or not on the allowlist
  body_rejected         Bad JSON, batch shape, size or identity
  all_accepted          Everything that arrived was accepted. It says nothing about events
                        that never came: name them with --event and look for a 0

The control requests behind a diagnosis: rudder-cli dev requests list --kind control --json

## Review in a browser

dev listen serves a read-only review page on the same port, at the ui url of the ready line.
It shows the counts and the diagnosis, a live list of events newest first (time, type, event,
status, write key), a search over event names and payload values, a failed-only switch, a
write key filter, and a detail pane with the properties and the full request JSON.

## Output and exit codes

Read commands print a table, or one JSON document with --json. Messages and errors go to
stderr; with --json an error is one JSON object with code, message and next. Nothing prompts.

  0  Any successful read, including an empty result or a --wait that ran out (timedOut is
     true). Help. A clean stop on SIGINT or SIGTERM.
  1  An error: a bad flag, no URL, no server at the URL, or server_changed.

## Agent and CI loop

One shell call, so it also works where background processes end when the call returns:

  d=$(mktemp -d)
  rudder-cli dev listen --port 0 > "$d/ready" 2> "$d/log" &
  pid=$!; trap 'kill "$pid" 2>/dev/null' EXIT
  until [ -s "$d/ready" ]; do sleep 0.1; done
  url=$(jq -r .url "$d/ready"); sid=$(jq -r .serverId "$d/ready"); cur=$(jq -r .cursor "$d/ready")
  ./run-the-app   # sends events to $url
  rudder-cli dev events --url "$url" --server-id "$sid" --since "$cur" --event 'Order Completed' --json | jq -e '.summary.byEvent["Order Completed"] == 1'
  rudder-cli dev events list --url "$url" --server-id "$sid" --since "$cur" --event 'Order Completed' --fields properties --json

The HTTP API behind these commands is listed at URL/_dev/v1/ and this guide at URL/_dev/v1/guide.

package dev

import (
	"github.com/MakeNowJust/heredoc/v2"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
)

// The dev group help is the workflow guide itself; the listener serves the
// same bytes at /_dev/v1/guide.
var (
	devLong    = devlisten.Guide()
	devExample = heredoc.Doc(`
		# Start a listener in the background and keep its ready line
		$ rudder-cli dev listen --port 0 > ready.json &

		# Counts and a diagnosis of what arrived
		$ rudder-cli dev events --url http://127.0.0.1:4321

		# The properties of one event, as JSON
		$ rudder-cli dev events list --url http://127.0.0.1:4321 --event 'Order Completed' --fields properties --json
	`)
)

var (
	listenLong = heredoc.Doc(`
		Run a local server that accepts RudderStack SDK requests like RudderStack ingestion and keeps
		every request in memory. It runs in the foreground until Ctrl-C or kill PID, and exits 0.

		When the port is bound it prints one JSON ready line on stdout and nothing more there:
		ready, url, port, bind, pid, serverId, cursor, writeKey, writeKeyPolicy and ui. Point the
		SDK dataPlaneUrl at url; a browser SDK also needs configUrl set to url. Server SDKs must
		flush before the process exits; in development set them to flush after 1 event. Open ui in
		a browser to review the events.

		--port 0 lets the system pick a free port, so parallel runs never collide. A fixed port that
		is taken fails at start with port_in_use and exit 1. Use --bind 0.0.0.0 only in a
		container. By default any write key is accepted, a missing one included. --write-key is an
		allowlist: it rejects every other key and a missing key with 401. On dev events and dev
		requests list, --write-key filters instead.

		Listeners share nothing: captures stay in memory, and there is no state or lock file.
	`)
	listenExample = heredoc.Doc(`
		# Foreground, on a fixed port
		$ rudder-cli dev listen --port 4321

		# Background for an agent or CI job; read url and pid from the ready line
		$ rudder-cli dev listen --port 0 > ready.json 2> listen.log &

		# Accept only two write keys
		$ rudder-cli dev listen --port 4321 --write-key web-key --write-key api-key
	`)
)

var (
	eventsLong = heredoc.Doc(`
		Print the summary of what arrived: counts of requests, events by type, byEvent,
		byWriteKey, control requests, SDK families, and a diagnosis whose next field is the
		command to run next. Filters narrow every count. Each --event NAME and --write-key KEY
		you pass is listed with 0 when nothing arrived, so a check is one jq -e call.
		all_accepted describes what arrived, not what did not.

		Read in rungs, cheapest first: dev events, then dev events list, then --view compact,
		then --fields PATH, then dev requests show SEQ. Every successful read exits 0, also when
		--wait runs out (timedOut is true in the JSON); errors exit 1.
	`)
	eventsExample = heredoc.Doc(`
		$ rudder-cli dev events --url http://127.0.0.1:4321

		# Assert a count in CI
		$ rudder-cli dev events --url "$url" --since "$cur" --event 'Order Completed' --json | jq -e '.summary.byEvent["Order Completed"] == 1'

		# Wait up to 30s for an event the app sends later
		$ rudder-cli dev events --url "$url" --since "$cur" --event 'Order Completed' --wait 30s --json
	`)

	eventsListLong = heredoc.Doc(`
		List the captured events after a cursor. The default view is list: one line per event
		with seq, time, type, event name, write key and status code. --view compact adds the
		payload without the SDK context keys; --view full adds message and enrichedMessage.
		--fields PATH returns only those dotted paths (context.X reads message.context.X) and
		cannot be combined with --view.

		The JSON envelope has cursor, hasMore, total (events matching the filters), returned,
		omitted, truncated, next and events. The summary is in dev events. A page is capped at
		--max-bytes; truncated.next names the call that continues. --limit 0 returns only the
		cursor. seq counts every request, so gaps are control requests, not lost events.
	`)
	eventsListExample = heredoc.Doc(`
		$ rudder-cli dev events list --url http://127.0.0.1:4321

		# The properties of one event
		$ rudder-cli dev events list --url "$url" --since "$cur" --event 'Order Completed' --fields properties --json

		# Check a property type
		$ rudder-cli dev events list --url "$url" --since "$cur" --event 'Order Completed' --fields properties --json | jq -e 'all(.events[]; .properties.total | type == "number")'

		# One event per line
		$ rudder-cli dev events list --url "$url" --json | jq -c '.events[]'
	`)
)

var (
	requestsLong = heredoc.Doc(`
		Inspect whole captured requests: the only view of rejected requests, requests that carried
		no event, and control traffic such as /sourceConfig and CORS preflights. Use it when
		events did not arrive or a request was refused.
	`)

	requestsListLong = heredoc.Doc(`
		List captured requests after a cursor. The default view is list: seq, kind, method, route,
		status code, outcome and event count. --view compact adds receivedAt, rejection and the
		event names; --view full is the whole record. --fields PATH returns those dotted paths and
		always keeps seq and request.method.

		--kind control shows /sourceConfig, preflights (method OPTIONS) and unknown paths. --failed
		keeps rejected requests; --failed=false keeps accepted ones. --write-key filters by the key a
		request was sent with.
	`)
	requestsListExample = heredoc.Doc(`
		# Why were requests refused
		$ rudder-cli dev requests list --url http://127.0.0.1:4321 --failed --view compact --json

		# Did the browser SDK load its configuration
		$ rudder-cli dev requests list --url http://127.0.0.1:4321 --kind control
	`)

	requestsShowLong = heredoc.Doc(`
		Show one captured request in full: headers (credentials redacted), the raw body as the app
		sent it, the response, the rejection and the ids of each event. The compact view leaves
		out each event's message copies, because the body holds them; --view full adds them.
	`)
	requestsShowExample = heredoc.Doc(`
		# The raw body the app sent
		$ rudder-cli dev requests show 8 --url http://127.0.0.1:4321 --fields request.body --json

		$ rudder-cli dev requests show 8 --url http://127.0.0.1:4321
	`)
)

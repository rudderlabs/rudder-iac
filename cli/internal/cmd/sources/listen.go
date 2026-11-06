package sources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
)

const (
	// shutdownTimeout bounds the drain after a signal.
	shutdownTimeout = 5 * time.Second
	// trackTimeout keeps an offline machine from holding the exit.
	trackTimeout = 2 * time.Second
)

var log = logger.New("sources")

type listenOptions struct {
	port       int
	bind       string
	writeKeys  []string
	allowHosts []string
	store      bool
	out        string
}

func newCmdListen() *cobra.Command {
	var opts listenOptions
	cmd := &cobra.Command{
		Use:   "listen",
		Short: "Capture SDK requests on a local endpoint",
		Long: heredoc.Doc(`
			Run a local server that answers RudderStack SDK requests like RudderStack ingestion. It runs
			in the foreground until Ctrl-C or kill PID, and exits 0.

			Every accepted event goes to stdout as one JSON line, exactly as the SDK sent it, so you can
			pipe it to grep or jq, or redirect it to a file. Each refused request prints one line on
			stderr with its status and reason. By default the listener keeps nothing in memory.

			When the port is bound it prints one JSON ready line on stderr: ready, url, port, bind, pid,
			serverId, writeKey and writeKeyPolicy. Point the SDK data plane URL at url. Web SDKs
			also need configUrl, and mobile SDKs controlPlaneUrl, set to url. Server SDKs must flush
			before the process exits.

			--store (-s) also keeps every request in memory, up to 10,000 requests or 64 MiB with the
			oldest evicted first, and turns on the query API that sources events reads and the review
			page at ui. The ready line then carries cursor and ui too.

			--out FILE writes the events to FILE instead of stdout. It refuses a file that exists, so two
			listeners never share one, and each line starts with a _dev object: the listener's serverId,
			the line's seq from 1, and receivedAt. An event with a _dev of its own gets the listener's
			last instead, where jq reads it. If an event cannot be written, to stdout or to FILE,
			the listener stops and exits 1 with output_failed on stderr.

			--port 0 lets the system pick a free port, so parallel runs never collide. A fixed port that
			is taken fails at start with port_in_use and exit 1. Use --bind 0.0.0.0 only in a
			container. By default any write key is accepted, a missing one included. --write-key is an
			allowlist: it rejects every other key and a missing key with 401. On sources events,
			--write-key filters instead.

			Listeners share nothing, and there is no state or lock file. rudder-cli sources --help has
			the whole guide.
		`),
		Example: heredoc.Doc(`
			# Print events as they arrive
			$ rudder-cli sources listen --port 4321

			# Keep only the Order Completed events
			$ rudder-cli sources listen --port 4321 | grep '"event":"Order Completed"'

			# Background for an agent or CI job, in a fresh folder: events to a new file, the
			# ready line in the log
			$ d=$(mktemp -d)
			$ rudder-cli sources listen --port 0 --out "$d/events" 2> "$d/log" &

			# Keep requests in memory for sources events and the review page
			$ rudder-cli sources listen --port 4321 --store

			# Accept only two write keys
			$ rudder-cli sources listen --port 4321 --write-key web-key --write-key api-key
		`),
		Args: noArgs,
		RunE: func(cmd *cobra.Command, _ []string) (err error) {
			e := opts.validate()
			if e == nil && cmd.Flags().Changed("out") && opts.out == "" {
				e = &usageError{message: "--out needs a file name", next: `rudder-cli sources listen --out "$(mktemp -d)/events"`}
			}
			if e != nil {
				return fail(cmd, e)
			}
			defer func() {
				track("sources listen", err,
					telemetry.KV{K: "loopbackBind", V: devlisten.IsLoopback(opts.bind)},
					telemetry.KV{K: "fixedPort", V: opts.port != 0},
					telemetry.KV{K: "store", V: opts.store},
				)
			}()

			ctx, release := stopOnSignal(cmd.Context())
			defer release()
			// A write to a closed stdout pipe kills the process with SIGPIPE
			// before it can say why. Notified, the write returns EPIPE instead.
			pipe := make(chan os.Signal, 1)
			signal.Notify(pipe, syscall.SIGPIPE)
			defer signal.Stop(pipe)
			return runListen(ctx, cmd.OutOrStdout(), cmd.ErrOrStderr(), opts, cmd.Root().Version)
		},
	}

	f := cmd.Flags()
	f.IntVar(&opts.port, "port", 0, "TCP port to listen on; 0 lets the system pick a free port")
	f.StringVar(&opts.bind, "bind", devlisten.DefaultBind, "IP address to listen on; 0.0.0.0 inside a container only")
	f.StringArrayVar(&opts.writeKeys, "write-key", nil,
		"Accept only this write key and answer 401 to others; repeat for more (default: accept any key)")
	f.StringArrayVar(&opts.allowHosts, "allow-host", nil,
		"Also accept this Host name on the query API, such as a container service name; repeat for more")
	f.BoolVarP(&opts.store, "store", "s", false,
		"Keep requests in memory and serve the query API and review page for sources events")
	f.StringVar(&opts.out, "out", "",
		"Write the events to this new file instead of stdout, each line with a _dev object; refuses a file that exists")
	return cmd
}

func (o listenOptions) validate() *usageError {
	if o.port < 0 || o.port > 65535 {
		return &usageError{
			message: fmt.Sprintf("--port must be 0 to 65535, got %d", o.port),
			next:    "rudder-cli sources listen --port 0",
		}
	}
	if net.ParseIP(o.bind) == nil {
		return &usageError{
			message: fmt.Sprintf("--bind must be an IP address such as 127.0.0.1, got %q", o.bind),
			next:    "rudder-cli sources listen --bind 127.0.0.1",
		}
	}
	for _, key := range o.writeKeys {
		if key == "" {
			return &usageError{message: "--write-key needs a key", next: "rudder-cli sources listen --help"}
		}
	}
	for _, host := range o.allowHosts {
		if host == "" {
			return &usageError{message: "--allow-host needs a host name", next: "rudder-cli sources listen --help"}
		}
		// The check compares names only, so a name with a port never matches.
		if name, _, err := net.SplitHostPort(host); err == nil {
			return &usageError{
				message: fmt.Sprintf("--allow-host takes a host name without a port, got %q", host),
				next:    "rudder-cli sources listen --allow-host " + shellWord(name),
			}
		}
	}
	return nil
}

// runListen serves until ctx ends. Stdout carries the accepted events only,
// so a pipe sees nothing else; the ready line goes to stderr.
func runListen(ctx context.Context, stdout, stderr io.Writer, opts listenOptions, version string) error {
	out, outName := stdout, "stdout"
	if opts.out != "" {
		// O_EXCL makes a second listener fail at start instead of sharing
		// the file. The events can hold personal data, so only the owner reads it.
		f, err := os.OpenFile(opts.out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		switch {
		case errors.Is(err, fs.ErrExist):
			return startupError(stderr, "out_exists", fmt.Errorf("--out %s exists; give a new file", opts.out),
				`rudder-cli sources listen --out "$(mktemp -d)/events"`)
		case err != nil:
			return startupError(stderr, "listen_failed", fmt.Errorf("creating --out: %w", err), "rudder-cli sources listen --help")
		}
		defer func() { _ = f.Close() }()
		out, outName = f, opts.out
	}
	srv, err := devlisten.Start(devlisten.Config{
		Port:       opts.port,
		Bind:       opts.bind,
		WriteKeys:  opts.writeKeys,
		AllowHosts: opts.allowHosts,
		Version:    version,
		Store:      opts.store,
		Out:        out,
		Stamp:      opts.out != "",
		Log:        stderr,
	})
	switch {
	case errors.Is(err, devlisten.ErrPortInUse):
		return startupError(stderr, "port_in_use", err, "rudder-cli sources listen --port 0")
	case err != nil:
		return startupError(stderr, "listen_failed", err, "rudder-cli sources listen --help")
	}

	ready := srv.Ready()
	log.Info("started", "serverId", ready.ServerID, "bind", ready.Bind, "port", ready.Port)
	if !devlisten.IsLoopback(ready.Bind) {
		fmt.Fprintf(stderr, "warning: listening on %s. The query API has no authentication. "+
			"Any host that reaches this port can read it.\n", net.JoinHostPort(ready.Bind, strconv.Itoa(ready.Port)))
	}
	line, _ := json.Marshal(ready)
	fmt.Fprintln(stderr, string(line))
	if isTerminal(stderr) {
		fmt.Fprintf(stderr, "sources listen: listening at %s (pid %d); Ctrl-C to stop.\n", ready.URL, ready.PID)
		if ready.UI != "" {
			fmt.Fprintf(stderr, "Review in a browser: %s\n", ready.UI)
			fmt.Fprintf(stderr, "Read events with: rudder-cli sources events --url %s\n", ready.URL)
		}
	}

	var served, outFailed error
	select {
	case <-ctx.Done():
		log.Info("stopping", "serverId", ready.ServerID, "reason", "signal")
	case served = <-srv.Done():
		log.Error("stopping", "serverId", ready.ServerID, "reason", "error", "error", served)
	case outFailed = <-srv.OutFailed():
		log.Error("stopping", "serverId", ready.ServerID, "reason", "output failed", "error", outFailed)
	}
	closeCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Close(closeCtx); err != nil {
		// Close has dropped the connections left, so the stop still succeeded.
		log.Warn("shutdown deadline exceeded", "serverId", ready.ServerID, "error", err)
	}
	if outFailed == nil {
		// A write can fail while a signal or a serve error stops the
		// listener, or while Close drains the requests in flight.
		select {
		case outFailed = <-srv.OutFailed():
		default:
		}
	}
	if outFailed != nil {
		// Counted after Close, so it includes the requests that drained.
		lost := srv.Lost()
		events := "events were"
		if lost == 1 {
			events = "event was"
		}
		return startupError(stderr, "output_failed",
			fmt.Errorf("writing events to %s: %w; the listener stopped, and %d accepted %s never written",
				outName, outFailed, lost, events),
			"rudder-cli sources listen --help")
	}
	if served != nil {
		return startupError(stderr, "listen_failed", fmt.Errorf("serving: %w", served), "rudder-cli sources listen --help")
	}
	return nil
}

// cliError is the error object of the query API, with a null status because
// no HTTP answer carried it.
type cliError struct {
	Status  *int    `json:"status"`
	Code    string  `json:"code"`
	Message string  `json:"message"`
	Param   *string `json:"param"`
	Details any     `json:"details"`
	Next    string  `json:"next"`
}

// startupError prints one JSON error object as the last stderr line, because
// a script that starts the listener in the background reads its log.
func startupError(stderr io.Writer, code string, err error, next string) error {
	line, _ := json.Marshal(map[string]cliError{"error": {Code: code, Message: err.Error(), Next: next}})
	fmt.Fprintln(stderr, string(line))
	return &cmderrors.SilentError{Err: err}
}

// stopOnSignal ends ctx on SIGINT or SIGTERM. A second signal exits 1 at
// once, for a user who does not want to wait for the drain.
func stopOnSignal(parent context.Context) (context.Context, func()) {
	ctx, cancel := context.WithCancel(parent)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		select {
		case <-signals:
		case <-done:
			return
		}
		cancel()
		select {
		case <-signals:
			os.Exit(1)
		case <-done:
		}
	}()
	return ctx, func() {
		signal.Stop(signals)
		close(done)
		cancel()
	}
}

// track sends no flag values, and waits at most trackTimeout, so an offline
// machine does not hold the exit.
func track(command string, err error, extras ...telemetry.KV) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		telemetry.TrackCommand(command, err, extras...)
	}()
	select {
	case <-done:
	case <-time.After(trackTimeout):
	}
}

// isTerminal checks w itself, because ui.IsTerminal checks stdout, which a
// script redirects to read the ready line.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

package dev

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

var log = logger.New("dev")

type listenOptions struct {
	port       int
	bind       string
	writeKeys  []string
	allowHosts []string
}

func newCmdListen() *cobra.Command {
	var opts listenOptions
	cmd := &cobra.Command{
		Use:   "listen",
		Short: "Capture the requests an app sends, on a local port",
		Long: heredoc.Doc(`
			Run a local listener that answers RudderStack SDKs as RudderStack does and
			keeps every request in memory until it stops.

			It prints one JSON line on stdout when it is ready: url, serverId, cursor and
			pid. Stop it with Ctrl-C or kill <pid>. The captures go with it.
		`),
		Example: heredoc.Doc(`
			# Foreground, on a fixed port
			$ rudder-cli dev listen --port 4321

			# Background for an agent or CI job; read url and pid from the ready line
			$ rudder-cli dev listen > ready.json &

			# Accept only two write keys
			$ rudder-cli dev listen --write-key web --write-key api
		`),
		Args: noArgs,
		RunE: func(cmd *cobra.Command, _ []string) (err error) {
			if e := opts.validate(); e != nil {
				return fail(cmd, e)
			}
			defer func() { track(err, opts) }()

			ctx, release := stopOnSignal(cmd.Context())
			defer release()
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
	return cmd
}

func (o listenOptions) validate() *usageError {
	if o.port < 0 || o.port > 65535 {
		return &usageError{
			message: fmt.Sprintf("--port must be 0 to 65535, got %d", o.port),
			next:    "rudder-cli dev listen --port 0",
		}
	}
	if net.ParseIP(o.bind) == nil {
		return &usageError{
			message: fmt.Sprintf("--bind must be an IP address such as 127.0.0.1, got %q", o.bind),
			next:    "rudder-cli dev listen --bind 127.0.0.1",
		}
	}
	for _, key := range o.writeKeys {
		if key == "" {
			return &usageError{message: "--write-key needs a key", next: "rudder-cli dev listen --help"}
		}
	}
	for _, host := range o.allowHosts {
		if host == "" {
			return &usageError{message: "--allow-host needs a host name", next: "rudder-cli dev listen --help"}
		}
		// The check compares names only, so a name with a port never matches.
		if name, _, err := net.SplitHostPort(host); err == nil {
			return &usageError{
				message: fmt.Sprintf("--allow-host takes a host name without a port, got %q", host),
				next:    "rudder-cli dev listen --allow-host " + shellWord(name),
			}
		}
	}
	return nil
}

// runListen serves until ctx ends. The ready line is all it writes on stdout.
func runListen(ctx context.Context, stdout, stderr io.Writer, opts listenOptions, version string) error {
	srv, err := devlisten.Start(devlisten.Config{
		Port:       opts.port,
		Bind:       opts.bind,
		WriteKeys:  opts.writeKeys,
		AllowHosts: opts.allowHosts,
		Version:    version,
	})
	switch {
	case errors.Is(err, devlisten.ErrPortInUse):
		return startupError(stderr, "port_in_use", err, "rudder-cli dev listen --port 0")
	case err != nil:
		return startupError(stderr, "listen_failed", err, "rudder-cli dev listen --help")
	}

	ready := srv.Ready()
	log.Info("started", "serverId", ready.ServerID, "bind", ready.Bind, "port", ready.Port)
	if !devlisten.IsLoopback(ready.Bind) {
		fmt.Fprintf(stderr, "warning: listening on %s. The query API has no authentication. "+
			"Any host that reaches this port can read it.\n", net.JoinHostPort(ready.Bind, strconv.Itoa(ready.Port)))
	}
	line, _ := json.Marshal(ready)
	fmt.Fprintln(stdout, string(line))
	if isTerminal(stderr) {
		fmt.Fprintf(stderr, "dev listen: capturing at %s (pid %d); Ctrl-C to stop.\n", ready.URL, ready.PID)
		fmt.Fprintf(stderr, "Review in a browser: %s\n", ready.UI)
		fmt.Fprintf(stderr, "Read events with: rudder-cli dev events --url %s\n", ready.URL)
	}

	var served error
	select {
	case <-ctx.Done():
		log.Info("stopping", "serverId", ready.ServerID, "reason", "signal")
	case served = <-srv.Done():
		log.Error("stopping", "serverId", ready.ServerID, "reason", "error", "error", served)
	}
	closeCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Close(closeCtx); err != nil {
		// Close has dropped the connections left, so the stop still succeeded.
		log.Warn("shutdown deadline exceeded", "serverId", ready.ServerID, "error", err)
	}
	if served != nil {
		return startupError(stderr, "listen_failed", fmt.Errorf("serving: %w", served), "rudder-cli dev listen --help")
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

// track sends no flag values: only whether the bind is loopback and the port fixed.
func track(err error, opts listenOptions) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		telemetry.TrackCommand("dev listen", err,
			telemetry.KV{K: "loopbackBind", V: devlisten.IsLoopback(opts.bind)},
			telemetry.KV{K: "fixedPort", V: opts.port != 0},
		)
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

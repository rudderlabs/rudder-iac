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
	"sync/atomic"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
)

const (
	shutdownTimeout = 5 * time.Second
	detachIdleExit  = 30 * time.Minute
	hintNameLen     = 64
)

type listenOptions struct {
	port          int
	bind          string
	idleExit      time.Duration
	idleExitSet   bool
	detach        bool
	quiet         bool
	noStateFile   bool
	detachedChild bool
	// progress is true when stderr is a terminal and --quiet is off.
	progress bool
}

func newCmdListen(deps Deps) *cobra.Command {
	var opts listenOptions
	cmd := &cobra.Command{
		Use:   "listen",
		Short: "Capture SDK requests on a local endpoint",
		Long: "Capture the requests your app sends to a local RudderStack endpoint.\n\n" +
			"This command blocks in the foreground. Read events from another shell.\n" +
			"Set the SDK data-plane URL and configUrl to the ready URL.\n" +
			"Stdout carries exactly one JSON ready line; progress goes to stderr.\n\n" +
			"Use --port for a fixed URL, --quiet to suppress progress, or --detach to return after\n" +
			"readiness. Stop with Ctrl-C or rudder-cli dev stop. --idle-exit defaults to off in the\n" +
			"foreground and 30m with --detach. Clients find the server through the state file\n" +
			"dev-listen.json in the config directory, which follows -c.",
		Example: "  # Foreground; leave this shell running\n" +
			"  rudder-cli dev listen --port 4321\n\n" +
			"  # Agent; returns one ready JSON line after startup\n" +
			"  rudder-cli dev listen --detach\n\n" +
			"  # Cleanup\n" +
			"  rudder-cli dev stop",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.idleExitSet = cmd.Flags().Changed("idle-exit")
			opts.progress = !opts.quiet && isTerminal(cmd.ErrOrStderr())
			if opts.detach {
				return runDetach(cmd.OutOrStdout(), cmd.ErrOrStderr(), opts, selfSpawner())
			}
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			return runListen(ctx, cmd.OutOrStdout(), cmd.ErrOrStderr(), deps, opts)
		},
	}
	f := cmd.Flags()
	f.IntVar(&opts.port, "port", 0, "Port to listen on; 0 lets the OS pick a free one")
	f.StringVar(&opts.bind, "bind", devlisten.DefaultBind, "Address to bind; use 0.0.0.0 in a container")
	f.DurationVar(&opts.idleExit, "idle-exit", 0, "Stop after `DURATION` without captures or queries; 0 is off")
	f.BoolVar(&opts.detach, "detach", false, "Start in the background and return after the ready line")
	f.BoolVar(&opts.quiet, "quiet", false, "Print no progress on stderr; warnings and errors stay")
	f.BoolVar(&opts.noStateFile, "no-state-file", false, "Do not write the state file; clients then need --url")
	f.BoolVar(&opts.detachedChild, "detached-child", false, "")
	_ = f.MarkHidden("detached-child")
	return cmd
}

// startupError prints the error object as the last stderr line, so a
// detached or redirected start is diagnosed from the log alone.
func startupError(stderr io.Writer, e *cliError) error {
	line, _ := json.Marshal(map[string]any{"error": asAPIError(e)})
	fmt.Fprintln(stderr, string(line))
	return &cmderrors.SilentError{Err: e}
}

// runListen serves until ctx ends or the server stops itself. The ready
// line is the only stdout output.
func runListen(ctx context.Context, stdout, stderr io.Writer, deps Deps, opts listenOptions) error {
	statePath, err := claimStateFile(ctx, deps, opts)
	if err != nil {
		return startupError(stderr, err)
	}
	var stats captureStats
	srv, err := startServer(stderr, opts, &stats)
	if err != nil {
		return startupError(stderr, err)
	}
	ready := publishReady(stdout, stderr, srv, opts, statePath)
	if opts.detachedChild {
		detachStdio()
	}

	reason := "signal"
	select {
	case <-ctx.Done():
	case <-srv.Done():
		reason = srv.StopReason()
	}
	return closeListen(stderr, srv, ready, statePath, reason, opts, &stats)
}

// closeListen drains the server, removes the state file it owns and prints
// the summary line.
func closeListen(stderr io.Writer, srv *devlisten.Server, ready devlisten.Ready, statePath, reason string,
	opts listenOptions, stats *captureStats,
) error {
	log.Info("stopping", "serverId", ready.ServerID, "reason", reason)
	closeCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	closeErr := srv.Close(closeCtx)
	if statePath != "" {
		removeStateIfOwned(statePath, ready.ServerID)
	}
	if closeErr != nil {
		log.Error("shutdown deadline exceeded", "error", closeErr)
		return fmt.Errorf("stopping dev listen: %w", closeErr)
	}
	if opts.progress {
		fmt.Fprintf(stderr, "dev listen stopped (%s, %d requests, %d failed, up %s)\n", reason,
			stats.requests.Load(), stats.failed.Load(), time.Since(ready.StartedAt).Round(time.Second))
	}
	return nil
}

// claimStateFile returns the state file path, "" with --no-state-file. A
// live server named by the file is already_running; a stale file is removed.
func claimStateFile(ctx context.Context, deps Deps, opts listenOptions) (string, *cliError) {
	if opts.noStateFile {
		return "", nil
	}
	path := stateFilePath(deps.ConfigDir())
	prev, err := readState(path)
	if err != nil {
		return path, nil
	}
	if verifyState(ctx, prev) {
		return "", &cliError{Code: "already_running", Message: "a server is already running at " + prev.URL,
			Next: "rudder-cli dev stop"}
	}
	_ = os.Remove(path)
	return path, nil
}

func startServer(stderr io.Writer, opts listenOptions, stats *captureStats) (*devlisten.Server, *cliError) {
	srv, err := devlisten.Start(context.Background(),
		devlisten.WithPort(opts.port), devlisten.WithBind(opts.bind), devlisten.WithIdleExit(opts.idleExit),
		devlisten.WithCaptureHook(func(c devlisten.Capture) {
			stats.add(c)
			if opts.progress {
				fmt.Fprintln(stderr, hintLine(c))
			}
		}))
	switch {
	case errors.Is(err, devlisten.ErrPortInUse):
		return nil, &cliError{Code: "port_in_use", Message: err.Error(), Next: "rudder-cli dev listen --port 0"}
	case err != nil:
		return nil, &cliError{Code: "listen_failed", Message: err.Error(), Next: "rudder-cli dev listen --help"}
	}
	return srv, nil
}

// publishReady writes the state file, then the ready line, so a poller
// that sees the line can query at once.
func publishReady(stdout, stderr io.Writer, srv *devlisten.Server, opts listenOptions, statePath string) devlisten.Ready {
	ready := srv.Ready()
	log.Info("started", "port", ready.Port, "bind", ready.Bind, "serverId", ready.ServerID, "stateFile", statePath)
	if !isLoopback(opts.bind) {
		fmt.Fprintln(stderr, bindWarning(opts.bind, ready.Port))
	}
	if statePath != "" {
		ready.StateFile = &statePath
		if err := writeState(statePath, ready); err != nil {
			ready.StateFile = nil
			fmt.Fprintf(stderr, "warning: discovery file unavailable (%v). Clients need --url %s\n", err, ready.URL)
		}
	}
	line, _ := json.Marshal(ready)
	fmt.Fprintln(stdout, string(line))
	if opts.progress {
		fmt.Fprintf(stderr, "dev listen: capturing at %s; Ctrl-C to stop.\n", ready.URL)
		fmt.Fprintf(stderr, "SDK setup: dataPlaneUrl=%s configUrl=%s writeKey=%s\n", ready.URL, ready.URL, ready.WriteKey)
		fmt.Fprintf(stderr, "Read events from another shell: rudder-cli dev events list --since %d\n", ready.Cursor)
	}
	return ready
}

type captureStats struct {
	requests atomic.Int64
	failed   atomic.Int64
}

func (s *captureStats) add(c devlisten.Capture) {
	if c.Kind != "ingestion" {
		return
	}
	s.requests.Add(1)
	if c.Outcome != "accepted" || c.StatusCode < 200 || c.StatusCode > 299 {
		s.failed.Add(1)
	}
}

// hintLine is one stderr line per captured request. It never carries a
// payload; captured names are quoted with control characters escaped.
func hintLine(c devlisten.Capture) string {
	show := "rudder-cli dev requests show " + strconv.FormatUint(c.Seq, 10)
	if c.Kind == "control" {
		return fmt.Sprintf("seq=%d kind=control route=%s statusCode=%d | %s", c.Seq, clean(c.Route, hintNameLen),
			c.StatusCode, show)
	}
	line := fmt.Sprintf("seq=%d route=%s type=%s event=%s statusCode=%d", c.Seq, clean(c.Route, hintNameLen),
		dash(clean(c.Type, hintNameLen)), quoteOrDash(c.Event), c.StatusCode)
	if c.Stage != "" {
		line += " stage=" + c.Stage
	}
	if c.Events > 1 {
		line += " events=" + strconv.Itoa(c.Events)
	}
	return line + " | " + show
}

func quoteOrDash(s string) string {
	if s == "" {
		return "-"
	}
	if runes := []rune(s); len(runes) > hintNameLen {
		s = string(runes[:hintNameLen]) + "..."
	}
	return strconv.Quote(s)
}

func bindWarning(bind string, port int) string {
	return fmt.Sprintf("warning: listening on %s:%d. The query API has no authentication.", bind, port)
}

func isLoopback(bind string) bool {
	if bind == "localhost" {
		return true
	}
	ip := net.ParseIP(bind)
	return ip != nil && ip.IsLoopback()
}

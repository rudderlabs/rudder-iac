package dev

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
)

const shutdownTimeout = 5 * time.Second

type listenOptions struct {
	port      int
	bind      string
	writeKeys []string
	// note is true when stderr is a terminal: the startup note is for a
	// human only.
	note bool
}

func newCmdListen(deps Deps) *cobra.Command {
	var opts listenOptions
	cmd := &cobra.Command{
		Use:     "listen",
		Short:   "Capture SDK requests on a local endpoint",
		Long:    "Capture the requests your app sends to a local RudderStack endpoint. It runs in the foreground.",
		Example: "  rudder-cli dev listen --port 4321 > ready.json &",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			defer func() {
				deps.Track("dev listen", err, telemetry.KV{K: "bind", V: opts.bind},
					telemetry.KV{K: "writeKeys", V: len(opts.writeKeys)})
			}()
			opts.note = isTerminal(cmd.ErrOrStderr())
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			return runListen(ctx, cmd.OutOrStdout(), cmd.ErrOrStderr(), opts)
		},
	}
	f := cmd.Flags()
	f.IntVar(&opts.port, "port", 0, "TCP port; 0 lets the OS pick a free one")
	f.StringVar(&opts.bind, "bind", devlisten.DefaultBind, "Address to bind; 0.0.0.0 in a container")
	f.StringArrayVar(&opts.writeKeys, "write-key", nil,
		"Accept only this write `KEY` and reject every other key with 401; repeatable (default: accept any key)")
	return cmd
}

// startupError prints the error object as the last stderr line, so a
// redirected start is diagnosed from the log alone.
func startupError(stderr io.Writer, e *cliError) error {
	line, _ := json.Marshal(map[string]any{"error": asAPIError(e)})
	fmt.Fprintln(stderr, string(line))
	return &cmderrors.SilentError{Err: e}
}

// runListen serves until ctx ends or the server stops. The ready line is
// the only stdout output.
func runListen(ctx context.Context, stdout, stderr io.Writer, opts listenOptions) error {
	srv, cerr := startServer(opts)
	if cerr != nil {
		return startupError(stderr, cerr)
	}
	ready := publishReady(stdout, stderr, srv, opts)

	select {
	case <-ctx.Done():
	case <-srv.Done():
	}
	log.Info("stopping", "serverId", ready.ServerID)
	closeCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Close(closeCtx); err != nil {
		log.Error("shutdown deadline exceeded", "serverId", ready.ServerID, "error", err)
		return fmt.Errorf("stopping dev listen: %w", err)
	}
	return nil
}

func startServer(opts listenOptions) (*devlisten.Server, *cliError) {
	srv, err := devlisten.Start(context.Background(),
		devlisten.WithPort(opts.port), devlisten.WithBind(opts.bind), devlisten.WithWriteKeys(opts.writeKeys...))
	switch {
	case errors.Is(err, devlisten.ErrPortInUse):
		return nil, &cliError{Code: "port_in_use", Message: err.Error(), Next: "rudder-cli dev listen --port 0"}
	case err != nil:
		return nil, &cliError{Code: "listen_failed", Message: err.Error(), Next: "rudder-cli dev listen --help"}
	}
	return srv, nil
}

// publishReady writes the ready line, flushed by Fprintln on an unbuffered
// writer, then the startup note when stderr is a terminal.
func publishReady(stdout, stderr io.Writer, srv *devlisten.Server, opts listenOptions) devlisten.Ready {
	ready := srv.Ready()
	log.Info("started", "port", ready.Port, "bind", ready.Bind, "serverId", ready.ServerID)
	if !devlisten.IsLoopback(opts.bind) {
		fmt.Fprintln(stderr, bindWarning(opts.bind, ready.Port))
	}
	line, _ := json.Marshal(ready)
	fmt.Fprintln(stdout, string(line))
	if opts.note {
		fmt.Fprintf(stderr, "dev listen: capturing at %s (pid %d); Ctrl-C to stop.\n", ready.URL, ready.PID)
		fmt.Fprintf(stderr, "Review in a browser: %s\n", ready.UI)
		fmt.Fprintf(stderr, "Read events with: rudder-cli dev events --url %s\n", ready.URL)
	}
	return ready
}

func bindWarning(bind string, port int) string {
	return fmt.Sprintf("warning: listening on %s:%d. The query API has no authentication. CORS reflects every "+
		"origin with credentials, so any web page can post events into this capture.", bind, port)
}

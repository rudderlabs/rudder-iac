package dev

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
)

const shutdownTimeout = 5 * time.Second

type listenOptions struct {
	port int
	bind string
}

func newCmdListen() *cobra.Command {
	var opts listenOptions
	cmd := &cobra.Command{
		Use:   "listen",
		Short: "Start a local server that captures the events an app sends",
		Long: "Start a local server that accepts events like RudderStack ingestion and keeps every request.\n" +
			"Stdout carries one JSON ready line with url, serverId and cursor; everything else goes to stderr.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			return runListen(ctx, cmd.OutOrStdout(), cmd.ErrOrStderr(), opts)
		},
	}
	cmd.Flags().IntVar(&opts.port, "port", 0, "port to listen on; 0 lets the OS pick a free one")
	cmd.Flags().StringVar(&opts.bind, "bind", devlisten.DefaultBind, "address to bind; use 0.0.0.0 in a container")
	return cmd
}

// runListen serves until ctx ends. The ready line is the only stdout output.
func runListen(ctx context.Context, stdout, stderr io.Writer, opts listenOptions) error {
	srv, err := devlisten.Start(context.Background(), devlisten.WithPort(opts.port), devlisten.WithBind(opts.bind))
	if err != nil {
		return fmt.Errorf("starting dev listen: %w", err)
	}
	ready := srv.Ready()
	log.Info("started", "port", ready.Port, "bind", ready.Bind, "serverId", ready.ServerID)

	if !isLoopback(opts.bind) {
		fmt.Fprintf(stderr, "warning: listening on %s:%d. The query API has no authentication.\n", opts.bind, ready.Port)
	}
	line, err := json.Marshal(ready)
	if err != nil {
		return fmt.Errorf("encoding ready line: %w", err)
	}
	fmt.Fprintln(stdout, string(line))
	fmt.Fprintf(stderr, "dev listen: capturing on %s (Ctrl-C to stop)\n", ready.URL)

	<-ctx.Done()
	log.Info("stopping", "serverId", ready.ServerID)
	closeCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Close(closeCtx); err != nil {
		log.Error("shutdown deadline exceeded", "error", err)
		return fmt.Errorf("stopping dev listen: %w", err)
	}
	fmt.Fprintf(stderr, "dev listen stopped (signal, up %s)\n", time.Since(ready.StartedAt).Round(time.Second))
	return nil
}

func isLoopback(bind string) bool {
	if bind == "localhost" {
		return true
	}
	ip := net.ParseIP(bind)
	return ip != nil && ip.IsLoopback()
}

package dev

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
)

type execOptions struct {
	port   int
	settle time.Duration
	expect []string
	flags  clientFlags
}

func newCmdExec() *cobra.Command {
	var o execOptions
	cmd := &cobra.Command{
		Use:   "exec [flags] -- COMMAND [ARG...]",
		Short: "Run a command against a listener that lives as long as the command",
		Long: "Start a listener in this process, run COMMAND with RUDDERSTACK_DATA_PLANE_URL and\n" +
			"RUDDERSTACK_DEV_URL set to it, wait --settle after COMMAND exits, print the summary and stop.\n" +
			"Use it where a detached server does not survive between shell calls, such as a sandbox.\n\n" +
			"COMMAND's stdout and stderr go to stderr, so stdout holds the summary only. The exit code\n" +
			"is COMMAND's when it failed; else 1 when an --expect event is missing or its count differs;\n" +
			"else 0. The app must read RUDDERSTACK_DATA_PLANE_URL, or use --port with its fixed port.",
		Example: "  rudder-cli dev exec --expect 'Order Completed' --json -- node backend.mjs\n" +
			"  rudder-cli dev exec --port 4321 --expect 'Suggestion Sent=1' --json -- npm run e2e",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExec(cmd, o, args)
		},
	}
	f := cmd.Flags()
	f.IntVar(&o.port, "port", 0, "Port to listen on; 0 lets the OS pick a free one")
	f.DurationVar(&o.settle, "settle", 2*time.Second, "Wait `DURATION` after COMMAND exits for late requests")
	f.StringArrayVar(&o.expect, "expect", nil, "Check that event `NAME[=COUNT]` arrived; repeat for more")
	f.BoolVarP(&o.flags.json, "json", "j", false, "Output the summary as JSON")
	f.StringVar(&o.flags.jq, "jq", "", "Filter the JSON summary with a jq `EXPR`; needs --json")
	return cmd
}

func runExec(cmd *cobra.Command, o execOptions, args []string) error {
	out := output{stdout: cmd.OutOrStdout(), stderr: cmd.ErrOrStderr(), flags: o.flags}
	if err := o.flags.check(cmd, nil); err != nil {
		return out.fail(err)
	}
	if cmd.ArgsLenAtDash() != 0 {
		return out.fail(usageError("rudder-cli dev exec --json -- "+shellWord(args[0]),
			"put COMMAND after --, so its flags are not read as flags of dev exec"))
	}
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	srv, cerr := startServer(io.Discard, listenOptions{port: o.port, bind: devlisten.DefaultBind}, &captureStats{})
	if cerr != nil {
		return out.fail(cerr)
	}
	defer closeServer(srv)
	since := srv.Ready().Cursor
	fmt.Fprintf(cmd.ErrOrStderr(), "dev exec: capturing at %s; RUDDERSTACK_DATA_PLANE_URL is set for COMMAND\n", srv.URL())

	exitCode, runErr := runChild(ctx, args, srv.URL(), cmd.ErrOrStderr())
	if runErr != nil {
		return out.fail(runErr)
	}
	_ = sleepContext(ctx, o.settle)

	s, err := srv.Client().Summary(ctx, devlisten.SummaryQuery{Since: since, Expect: o.expect})
	if err != nil {
		return out.fail(err)
	}
	if err := out.result(ctx, s.Raw, "", func(w io.Writer) { printSummary(w, s) }); err != nil {
		return err
	}
	return execOutcome(out, exitCode, s)
}

// runChild runs COMMAND with the listener URL in its environment. A
// command that ran and failed returns its exit code, not an error.
func runChild(ctx context.Context, args []string, url string, stdio io.Writer) (int, error) {
	child := exec.CommandContext(ctx, args[0], args[1:]...)
	child.Env = append(os.Environ(), "RUDDERSTACK_DATA_PLANE_URL="+url, "RUDDERSTACK_DEV_URL="+url)
	child.Stdout, child.Stderr = stdio, stdio
	err := child.Run()
	var exitErr *exec.ExitError
	switch {
	case errors.As(err, &exitErr):
		return exitErr.ExitCode(), nil
	case err != nil:
		return 0, &cliError{Code: "command_failed", Message: err.Error(),
			Next: "rudder-cli dev exec --help"}
	}
	return 0, nil
}

// execOutcome turns the child exit code and the expectations into the exit
// code of dev exec.
func execOutcome(out output, exitCode int, s devlisten.Summary) error {
	if exitCode != 0 {
		return withExitCode(out.fail(&cliError{Code: "command_failed",
			Message: fmt.Sprintf("COMMAND exited with %d", exitCode), Next: "rudder-cli dev exec --help"}), exitCode)
	}
	for _, d := range s.Diagnosis {
		if d.Code == "expected_missing" || d.Code == "expected_count_mismatch" {
			return withExitCode(out.fail(&cliError{Code: d.Code, Message: d.Message,
				Details: map[string]any{"events": d.Events}, Next: d.Next}), 1)
		}
	}
	return nil
}

// withExitCode sets the exit code of the SilentError that fail returns.
func withExitCode(err error, code int) error {
	var silent *cmderrors.SilentError
	if errors.As(err, &silent) {
		silent.Code = code
	}
	return err
}

func closeServer(srv *devlisten.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	_ = srv.Close(ctx)
}

func sleepContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

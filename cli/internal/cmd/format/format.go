package format

import (
	"fmt"
	"io"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/format"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/spf13/cobra"
)

var fmtLog = logger.New("root", logger.Attr{Key: "cmd", Value: "fmt"})

// NewCmdFmt builds the `rudder-cli fmt` command: a deterministic, idempotent
// formatter for spec YAML that normalizes layout while preserving comments and
// key order.
func NewCmdFmt() *cobra.Command {
	var (
		check bool
		diff  bool
	)

	cmd := &cobra.Command{
		Use:   "fmt [path...]",
		Short: "Format spec YAML files into canonical form",
		Long: heredoc.Doc(`
			Rewrites spec YAML files into a canonical form: normalized indentation,
			whitespace and quoting. Comments and key order are preserved, and the
			formatting is idempotent and semantics-preserving.

			With no paths, the current directory is formatted recursively.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli fmt
			$ rudder-cli fmt ./specs
			$ rudder-cli fmt --check ./specs
			$ rudder-cli fmt --diff spec.yaml
		`),
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if check && diff {
				return fmt.Errorf("--check and --diff are mutually exclusive")
			}

			defer func() {
				telemetry.TrackCommand("fmt", err,
					telemetry.KV{K: "check", V: check},
					telemetry.KV{K: "diff", V: diff},
				)
			}()

			results, err := format.Run(args, format.Options{Check: check, Diff: diff})
			if err != nil {
				return fmt.Errorf("formatting specs: %w", err)
			}

			return report(cmd.OutOrStdout(), results, check, diff)
		},
	}

	cmd.Flags().BoolVar(&check, "check", false, "Report files that need formatting and exit non-zero without writing")
	cmd.Flags().BoolVar(&diff, "diff", false, "Print a unified diff of changes without writing")
	return cmd
}

// report prints per-file outcomes and returns an error when appropriate:
// per-file formatting failures abort with an error; in check mode any file that
// would change yields a SilentError (non-zero exit, no stderr noise).
func report(w io.Writer, results []format.Result, check, diff bool) error {
	var (
		changed int
		failed  error
	)

	for _, r := range results {
		if r.Err != nil {
			fmtLog.Error("formatting file", "path", r.Path, "error", r.Err)
			failed = r.Err
			continue
		}
		if !r.Changed {
			continue
		}
		changed++

		if diff {
			fmt.Fprint(w, r.Diff)
			continue
		}
		fmt.Fprintln(w, r.Path)
	}

	if failed != nil {
		return failed
	}

	if check && changed > 0 {
		fmt.Fprintf(w, "%d file(s) need formatting\n", changed)
		return &cmderrors.SilentError{Err: fmt.Errorf("%d file(s) not formatted", changed)}
	}

	return nil
}

package dev

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

// usageError carries next, a command that runs as typed, so a user or an
// agent always has a step that parses.
type usageError struct {
	message string
	next    string
}

func (e *usageError) Error() string { return e.message }

// fail prints e with its next command and returns a SilentError, so the
// root prints nothing more and exits 1.
func fail(cmd *cobra.Command, e *usageError) error {
	w := cmd.ErrOrStderr()
	fmt.Fprintln(w, ui.Error(e))
	fmt.Fprintln(w, "Next: "+e.next)
	return &cmderrors.SilentError{Err: e}
}

// groupArgs rejects any argument to a command group, because a group only
// receives arguments that name no subcommand.
func groupArgs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	return fail(cmd, &usageError{
		message: fmt.Sprintf("unknown command %q for %q", args[0], cmd.CommandPath()),
		next:    cmd.CommandPath() + " --help",
	})
}

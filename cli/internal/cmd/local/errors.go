package local

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

// usageError carries next, a command that runs as typed, so a user or an
// agent always has a step that parses. code defaults to usage.
type usageError struct {
	code    string
	message string
	next    string
}

func (e *usageError) Error() string { return e.message }

// fail prints e with its next command, as one JSON object when the caller
// asked for JSON, and returns a SilentError, so the root prints nothing more
// and exits 1.
func fail(cmd *cobra.Command, e *usageError) error {
	w := cmd.ErrOrStderr()
	if wantsJSON(cmd) {
		code := e.code
		if code == "" {
			code = "usage"
		}
		line, _ := json.Marshal(map[string]cliError{"error": {Code: code, Message: e.message, Next: e.next}})
		fmt.Fprintln(w, string(line))
		return &cmderrors.SilentError{Err: e}
	}
	fmt.Fprintln(w, ui.Error(e))
	fmt.Fprintln(w, "Next: "+e.next)
	return &cmderrors.SilentError{Err: e}
}

// wantsJSON finds --json also after a bad flag, where parsing stopped
// before it. Only commands with a --json flag answer in JSON.
func wantsJSON(cmd *cobra.Command) bool {
	f := cmd.Flags().Lookup("json")
	if f == nil {
		return false
	}
	if f.Changed {
		return f.Value.String() == "true"
	}
	for _, arg := range os.Args[1:] {
		if arg == "--" {
			break
		}
		if arg == "--json" || arg == "-j" || arg == "--json=true" {
			return true
		}
	}
	return false
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

// noArgs rejects any argument to a command that takes none.
func noArgs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	return fail(cmd, &usageError{
		message: fmt.Sprintf("unknown argument %q for %q", args[0], cmd.CommandPath()),
		next:    cmd.CommandPath() + " --help",
	})
}

var plainWord = regexp.MustCompile(`^[A-Za-z0-9._/:@%+=,-]+$`)

// shellWord quotes a value the user typed, so a next command runs as printed.
func shellWord(s string) string {
	if plainWord.MatchString(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

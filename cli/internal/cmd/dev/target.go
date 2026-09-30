package dev

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
)

// clientSlack is added to --wait for the default HTTP deadline, so the
// server's timedOut answer arrives before the client gives up.
const clientSlack = 5 * time.Second

// clientFlags are the flags every client command shares. Only url,
// timeout, json and jq are CLI-only; server-id is the serverId parameter.
type clientFlags struct {
	url      string
	serverID string
	timeout  time.Duration
	json     bool
	jq       string
}

func (f *clientFlags) register(fs *pflag.FlagSet, withServerID bool) {
	fs.StringVar(&f.url, "url", "", "Server URL; default RUDDERSTACK_DEV_URL, then the state file")
	if withServerID {
		fs.StringVar(&f.serverID, "server-id", "", "Filter by server: a restarted server answers server_changed")
	}
	fs.DurationVar(&f.timeout, "timeout", 0, "HTTP `DURATION` for the call; default --wait plus 5s")
	fs.BoolVarP(&f.json, "json", "j", false, "Output the server response as JSON")
	fs.StringVar(&f.jq, "jq", "", "Filter the JSON output with a jq `EXPR`; needs --json")
}

// check refuses --jq without --json; next is the same command with --json.
func (f clientFlags) check(cmd *cobra.Command, args []string) error {
	if f.jq != "" && !f.json {
		return usageError(commandLine(cmd, args)+" --json", "--jq needs --json")
	}
	return nil
}

// resolve finds the server: --url, then RUDDERSTACK_DEV_URL, then the state
// file verified by serverId. An explicit URL sends --server-id only when it
// is given; the state file pins its own serverId.
func resolve(ctx context.Context, deps Deps, f clientFlags, wait time.Duration) (*devlisten.Client, error) {
	opts := f.clientOptions(wait)

	if url := firstNonEmpty(f.url, deps.DevURL()); url != "" {
		return devlisten.NewClient(url, opts...), nil
	}

	path := stateFilePath(deps.ConfigDir())
	ready, err := readState(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, &cliError{Code: "server_unreachable",
			Message: "no server found: no --url, no RUDDERSTACK_DEV_URL and no state file at " + path,
			Next:    nextListen}
	}
	if err != nil || !verifyState(ctx, ready) {
		return nil, &cliError{Code: "stale_state",
			Message: "the state file " + path + " names " + ready.URL + ", which is not running that server",
			Next:    nextListen}
	}
	if f.serverID == "" {
		opts = append(opts, devlisten.WithServerID(ready.ServerID))
	}
	return devlisten.NewClient(ready.URL, opts...), nil
}

func (f clientFlags) clientOptions(wait time.Duration) []devlisten.ClientOption {
	timeout := f.timeout
	if timeout == 0 {
		timeout = wait + clientSlack
	}
	opts := []devlisten.ClientOption{
		devlisten.WithHTTPClient(&http.Client{
			Timeout:   timeout,
			Transport: http.DefaultTransport.(*http.Transport).Clone(),
		}),
		devlisten.WithoutServerIDPin(),
	}
	if f.serverID != "" {
		opts = append(opts, devlisten.WithServerID(f.serverID))
	}
	return opts
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

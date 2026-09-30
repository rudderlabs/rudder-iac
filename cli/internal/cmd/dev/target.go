package dev

import (
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
)

// clientSlack is added to --wait for the default HTTP deadline, so the
// server's timedOut answer arrives before the client gives up.
const clientSlack = 5 * time.Second

// clientFlags are the flags every read command shares. url, timeout and
// json are CLI-only; server-id is the serverId parameter.
type clientFlags struct {
	url      string
	serverID string
	timeout  time.Duration
	json     bool
}

// register adds the shared flags. waits is true for a command with --wait,
// whose default timeout grows with it.
func (f *clientFlags) register(fs *pflag.FlagSet, waits bool) {
	timeout := "HTTP timeout of the call (default 5s)"
	if waits {
		timeout = "HTTP timeout of the call (default --wait plus 5s)"
	}
	fs.StringVar(&f.url, "url", "", "Listener URL from the ready line (default RUDDERSTACK_DEV_URL)")
	fs.StringVar(&f.serverID, "server-id", "", "serverId from the ready line, so another or a restarted listener answers server_changed")
	fs.DurationVar(&f.timeout, "timeout", 0, timeout)
	fs.BoolVarP(&f.json, "json", "j", false, "Output as JSON")
}

// resolve builds the client for --url, then RUDDERSTACK_DEV_URL. There is
// no discovery: several listeners run side by side and share nothing.
func resolve(cmd *cobra.Command, args []string, deps Deps, f clientFlags, wait time.Duration) (*devlisten.Client, error) {
	url := f.url
	if url == "" {
		url = deps.DevURL()
	}
	if url == "" {
		return nil, usageError(commandLine(cmd, args)+" --url URL",
			"no server URL: pass --url with the url of the dev listen ready line, or set RUDDERSTACK_DEV_URL")
	}
	return devlisten.NewClient(url, f.clientOptions(wait)...), nil
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

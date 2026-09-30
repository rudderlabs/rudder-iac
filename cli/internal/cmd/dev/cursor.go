package dev

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

type cursorResult struct {
	Cursor   uint64 `json:"cursor"`
	ServerID string `json:"serverId"`
}

func newCmdCursor(deps Deps) *cobra.Command {
	var o clientFlags
	cmd := &cobra.Command{
		Use:   "cursor",
		Short: "Print the current cursor (GET /_dev/v1/info)",
		Long: "Print the cursor and serverId of the running server as one JSON object. Read it before an\n" +
			"action, then pass it as --since to see only what the action sent. Output is always JSON.",
		Example: "  rudder-cli dev cursor\n" +
			"  rudder-cli dev cursor --jq .cursor",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.json = true
			out := output{stdout: cmd.OutOrStdout(), stderr: cmd.ErrOrStderr(), flags: o}
			client, err := resolve(cmd.Context(), deps, o, 0)
			if err != nil {
				return out.fail(err)
			}
			info, err := client.Info(cmd.Context())
			if err != nil {
				return out.fail(err)
			}
			raw, _ := json.Marshal(cursorResult{Cursor: info.Cursor, ServerID: info.ServerID})
			return out.result(cmd.Context(), raw, "", func(w io.Writer) { fmt.Fprintln(w, string(raw)) })
		},
	}
	o.register(cmd.Flags(), false)
	_ = cmd.Flags().MarkHidden("json")
	return cmd
}

// Package agentinit implements `rudder-cli init`, which prepares an app
// directory so the user's coding agent can set up RudderStack in it. It reads
// the workspace behind the access token and never changes the workspace.
package agentinit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/spf13/cobra"
)

const (
	defaultSkillsSource = "rudderlabs/rudder-agent-skills"
	defaultSkillsAgent  = "claude-code"
	skillsSourceEnv     = "RUDDER_SKILLS_SOURCE"
	accessTokenEnv      = "RUDDERSTACK_ACCESS_TOKEN" //nolint:gosec // an environment variable name, not a credential

	nextStep = "Open your coding agent in this directory and say: set up RudderStack"
)

// ErrTokenRejected is returned when the API answers 401 for the access token.
var ErrTokenRejected = errors.New("the RudderStack API rejected the access token")

// lookPath is a variable so tests can stand in for a missing git or npx.
var lookPath = exec.LookPath

type options struct {
	location    string
	skills      bool
	skillsAgent string
	json        bool
}

type result struct {
	Workspace workspaceSummary `json:"workspace"`
	Files     []fileAction     `json:"files"`
	Skills    skillsResult     `json:"skills"`
	Next      string           `json:"next"`
}

type workspaceSummary struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	DataPlaneURL string `json:"dataPlaneURL"`
	Region       string `json:"region"`
}

type skillsResult struct {
	Ran      bool `json:"ran"`
	ExitCode *int `json:"exitCode"`
}

func NewCmdInit() *cobra.Command {
	opts := options{}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Prepare a directory for a coding agent to set up RudderStack",
		Long: heredoc.Docf(`
			Prepare an app directory so a coding agent can set up RudderStack in it.

			init checks the access token and then writes, or updates in place:
			  rudder/README.md   where the specs go and how to pass secrets
			  .gitignore         a rudder/*.vars.yaml line so secrets stay out of git
			  .mcp.json          the RudderStack MCP server, using the same access token
			  AGENTS.md          the workspace and the rules the agent follows
			  CLAUDE.md          an @AGENTS.md import

			It runs git init when the directory is not in a git work tree. It never
			changes the workspace, and a second run changes nothing.

			--skills installs the RudderStack agent skills with npx. Set %s
			to install from another source, for testing unreleased skills.
		`, skillsSourceEnv),
		Example: heredoc.Doc(`
			$ rudder-cli init
			$ rudder-cli init -l ./my-app --skills
			$ rudder-cli init --json
		`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			defer func() {
				telemetry.TrackCommand(telemetry.CommandName(cmd), err, []telemetry.KV{
					{K: "skills", V: opts.skills},
					{K: "json", V: opts.json},
				}...)
			}()

			d, err := app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}

			workspace, err := resolveWorkspace(cmd.Context(), d.Client())
			if err != nil {
				return err
			}

			return run(cmd.Context(), workspace, opts, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}

	cmd.Flags().StringVarP(&opts.location, "location", "l", ".", "Path to the app directory to prepare")
	cmd.Flags().BoolVar(&opts.skills, "skills", false, "Install the RudderStack agent skills with npx")
	cmd.Flags().StringVar(&opts.skillsAgent, "skills-agent", defaultSkillsAgent, "Agent to install the skills for, passed to npx skills add -a")
	cmd.Flags().BoolVar(&opts.json, "json", false, "Print the result as one JSON object")

	return cmd
}

// resolveWorkspace reads the workspace behind the access token, as
// `workspace info` does.
func resolveWorkspace(ctx context.Context, c *client.Client) (*client.Workspace, error) {
	workspace, err := c.Workspaces.GetByAuthToken(ctx)
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.HTTPStatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("%w (HTTP 401): create a new access token and run `rudder-cli auth login`, or set it in %s", ErrTokenRejected, accessTokenEnv)
		}
		return nil, fmt.Errorf("fetching workspace information: %w", err)
	}
	if workspace == nil {
		return nil, errors.New("fetching workspace information: the API returned no workspace for this access token")
	}

	return workspace, nil
}

func run(ctx context.Context, workspace *client.Workspace, opts options, stdout, stderr io.Writer) error {
	res := &result{
		Workspace: summarize(workspace),
		Files:     []fileAction{},
		Next:      nextStep,
	}

	if !opts.json {
		fmt.Fprintf(stdout, "Workspace: %s (%s)\n", workspace.Name, workspace.ID)
	}

	if err := os.MkdirAll(opts.location, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", opts.location, err)
	}

	actions, err := prepare(ctx, opts.location, workspace.ID, workspace.Name)
	res.Files = append(res.Files, actions...)
	if !opts.json {
		for _, a := range actions {
			fmt.Fprintln(stdout, a.String())
		}
	}
	if err != nil {
		return err
	}

	if slices.Contains(actions, fileAction{Path: gitDir, Action: actionSkipped, Note: gitMissingNote}) {
		fmt.Fprintln(stderr, ui.Warning("git is not installed, so init did not run git init. Run it yourself so you can revert the agent's changes."))
	}

	// Claude Code expands ${RUDDERSTACK_ACCESS_TOKEN} in .mcp.json from the
	// environment, so a token kept only in the config file does not reach MCP.
	if os.Getenv(accessTokenEnv) == "" {
		fmt.Fprintln(stderr, ui.Warning(fmt.Sprintf(".mcp.json reads the access token from %s, which is not set in this shell. Export it before you start your coding agent.", accessTokenEnv)))
	}

	if opts.skills {
		// npx output streams to stderr under --json so stdout stays one JSON object.
		toolOut := stdout
		if opts.json {
			toolOut = stderr
		}
		res.Skills = installSkills(ctx, opts.location, opts.skillsAgent, toolOut, stderr)
		if !opts.json {
			fmt.Fprintln(stdout, skillsLine(res.Skills))
		}
	}

	if opts.json {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		enc.SetEscapeHTML(false)
		return enc.Encode(res)
	}

	fmt.Fprintln(stdout, "Ready. "+nextStep)
	return nil
}

func summarize(workspace *client.Workspace) workspaceSummary {
	dataPlaneURL := ""
	if workspace.DataPlaneURL != nil {
		dataPlaneURL = *workspace.DataPlaneURL
	}

	return workspaceSummary{
		ID:           workspace.ID,
		Name:         workspace.Name,
		DataPlaneURL: dataPlaneURL,
		Region:       workspace.Region,
	}
}

// installSkills runs `npx skills add` in dir. A missing npx or a failed install
// is a warning, not a failure: AGENTS.md tells the agent how to install the
// onboarding skill itself.
func installSkills(ctx context.Context, dir, agent string, toolOut, stderr io.Writer) skillsResult {
	source := os.Getenv(skillsSourceEnv)
	if source == "" {
		source = defaultSkillsSource
	}
	args := []string{"-y", "skills", "add", source, "-y", "-a", agent, "-s", "*"}
	// The printed form quotes the trailing * so it does not glob when pasted
	// into a shell.
	printable := "npx " + strings.Join(args[:len(args)-1], " ") + " '*'"

	npx, err := lookPath("npx")
	if err != nil {
		fmt.Fprintln(stderr, ui.Warning(fmt.Sprintf("npx is not installed, so the agent skills were not installed. Run this in %s: %s", dir, printable)))
		return skillsResult{}
	}

	c := exec.CommandContext(ctx, npx, args...)
	c.Dir = dir
	c.Stdout, c.Stderr = toolOut, stderr

	code := 0
	if err := c.Run(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			fmt.Fprintln(stderr, ui.Warning(fmt.Sprintf("running %s: %s", printable, err)))
			return skillsResult{}
		}
		code = exitErr.ExitCode()
		fmt.Fprintln(stderr, ui.Warning(fmt.Sprintf("%s exited with code %d. Run it again in %s, or let the agent install the skill as AGENTS.md describes.", printable, code, dir)))
	}

	return skillsResult{Ran: true, ExitCode: &code}
}

func skillsLine(s skillsResult) string {
	switch {
	case !s.Ran:
		return fmt.Sprintf("%-10s agent skills (npx did not run)", actionSkipped)
	case *s.ExitCode != 0:
		return fmt.Sprintf("%-10s agent skills (npx exited with code %d)", "failed", *s.ExitCode)
	default:
		return fmt.Sprintf("%-10s agent skills", "installed")
	}
}

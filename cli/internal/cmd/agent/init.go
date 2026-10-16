package agent

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/spf13/cobra"
)

const (
	agentsFileName = "AGENTS.md"
	skillsPackage  = "rudderlabs/rudder-agent-skills"
	readyMessage   = "Ready. Open your coding agent here and say: set up RudderStack"

	versionPlaceholder = "{{RUDDER_CLI_VERSION}}"
)

// Named so that the skills command a user sees and the one init runs cannot differ.
var skillsCommand = []string{"npx", "skills", "add", skillsPackage}

//go:embed agents_template.md
var agentsTemplate string

// InitOptions configures one run of init.
type InitOptions struct {
	Dir           string
	Force         bool
	PrintSkills   bool
	InstallSkills bool
	Version       string
}

// InitDeps holds the effects init has outside the working directory, so tests
// can replace them.
type InitDeps struct {
	// Workspace resolves the workspace behind the configured access token.
	Workspace func(ctx context.Context) (*client.Workspace, error)
	// Run executes an external command with the terminal attached.
	Run func(ctx context.Context, name string, args ...string) error
}

func NewCmdInit() *cobra.Command {
	var opts InitOptions

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Write AGENTS.md so a coding agent can set up RudderStack in this project",
		Long: heredoc.Doc(`
			Prepares the current directory for a coding agent. The command is
			non-interactive. It verifies the access token, reports what it sees in the
			directory, and writes an AGENTS.md file.

			It writes no spec, runs no apply and creates no resource in the workspace.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli agent init
			$ rudder-cli agent init --force
			$ rudder-cli agent init --skills
			$ rudder-cli agent init --install-skills
		`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			defer func() {
				telemetry.TrackCommand(telemetry.CommandName(cmd), err)
			}()

			opts.Version = app.GetVersion()

			return RunInit(cmd.Context(), cmd.OutOrStdout(), opts, InitDeps{
				Workspace: lookupWorkspace,
				Run:       runWithTerminal,
			})
		},
	}

	cmd.Flags().StringVarP(&opts.Dir, "location", "l", ".", "Project directory to inspect and write AGENTS.md into")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Overwrite an existing AGENTS.md")
	cmd.Flags().BoolVar(&opts.PrintSkills, "skills", false, "Print the command that installs the RudderStack agent skills, without running it")
	cmd.Flags().BoolVar(&opts.InstallSkills, "install-skills", false, "Install the RudderStack agent skills by running npx")

	return cmd
}

// RunInit performs the steps in order and stops at the first failure, so a bad
// token or an existing AGENTS.md never leaves a partial result behind.
func RunInit(ctx context.Context, out io.Writer, opts InitOptions, deps InitDeps) error {
	info, err := os.Stat(opts.Dir)
	if err != nil {
		return fmt.Errorf("reading project directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", opts.Dir)
	}

	workspace, err := deps.Workspace(ctx)
	if err != nil {
		return fmt.Errorf("verifying access token: %w", err)
	}
	fmt.Fprintf(out, "Workspace: %s\n", workspace.Name)

	fmt.Fprintf(out, "Project: %s\n", DetectProject(opts.Dir).Summary())

	if err := writeAgentsFile(opts.Dir, opts.Version, opts.Force); err != nil {
		return err
	}
	fmt.Fprintf(out, "Wrote %s\n", agentsFileName)

	switch {
	case opts.InstallSkills:
		fmt.Fprintf(out, "Installing skills: %s\n", strings.Join(skillsCommand, " "))
		if err := deps.Run(ctx, skillsCommand[0], skillsCommand[1:]...); err != nil {
			return fmt.Errorf("installing skills (%s): %w", strings.Join(skillsCommand, " "), err)
		}
	case opts.PrintSkills:
		fmt.Fprintf(out, "To install the agent skills, run: %s\n", strings.Join(skillsCommand, " "))
	}

	fmt.Fprintln(out, readyMessage)
	return nil
}

// writeAgentsFile creates AGENTS.md. Without force it opens the file with
// O_EXCL, so an existing file is refused atomically rather than after a check.
func writeAgentsFile(dir, version string, force bool) error {
	var (
		path    = filepath.Join(dir, agentsFileName)
		content = strings.ReplaceAll(agentsTemplate, versionPlaceholder, version)
		flags   = os.O_WRONLY | os.O_CREATE | os.O_EXCL
	)
	if force {
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}

	f, err := os.OpenFile(path, flags, 0o644)
	if errors.Is(err, os.ErrExist) {
		return fmt.Errorf("%s already exists in %s, rerun with --force to overwrite it", agentsFileName, dir)
	}
	if err != nil {
		return fmt.Errorf("creating %s: %w", agentsFileName, err)
	}

	if _, err := f.WriteString(content); err != nil {
		f.Close()
		return fmt.Errorf("writing %s: %w", agentsFileName, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("writing %s: %w", agentsFileName, err)
	}
	return nil
}

// lookupWorkspace makes the call `workspace info` makes.
func lookupWorkspace(ctx context.Context) (*client.Workspace, error) {
	d, err := app.NewDeps()
	if err != nil {
		return nil, fmt.Errorf("initialising dependencies: %w", err)
	}

	return d.Client().Workspaces.GetByAuthToken(ctx)
}

func runWithTerminal(ctx context.Context, name string, args ...string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%s is not on PATH, install Node.js to use it: %w", name, err)
	}

	c := exec.CommandContext(ctx, name, args...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	return c.Run()
}

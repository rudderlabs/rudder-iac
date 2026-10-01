package initcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/formatter"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

// FlowEventStream is the only flow the wizard builds today: one event stream
// source, one destination, one connection. The flag exists so the reverse-ETL
// and Profiles arms can be added as further values without changing the command
// surface.
const FlowEventStream = "event-stream"

// DestinationWebhook is the only destination the wizard writes today. Its
// config is ordinary (a URL), so the generated project needs no secrets file.
const DestinationWebhook = "webhook"

// sourceTypeChoices is the short list offered interactively.
//
// ponytail: the full set of event stream source types is the `oneof` tag on
// SourceSpec.SourceDefinition (cli/internal/providers/event-stream/source/model.go).
// Rather than mirror all seventeen here and have two lists drift, the prompt
// offers the six that cover the demo and --source-type passes anything through
// to the project validator, which owns the real check and words it better.
var sourceTypeChoices = []string{
	"javascript",
	"node",
	"python",
	"android",
	"ios",
	"go",
}

// Prompter is the three ui helpers the wizard asks questions through, held as
// fields rather than taken from the ui package directly so a test can count the
// questions and answer them.
type Prompter struct {
	Ask    func(question, def string) (string, error)
	Select func(question string, options []string) (string, error)
}

// DefaultPrompter wires the prompter to the terminal.
func DefaultPrompter() Prompter {
	return Prompter{Ask: ui.Ask, Select: ui.Select}
}

// Answers is everything the wizard needs to write a project.
type Answers struct {
	SourceType  string
	Destination string
	WebhookURL  string
}

// Resolve fills in the answers from flags, from package.json, and finally from
// the three questions. Questions already answered by a flag are not asked, and
// with yes set nothing is asked at all: a required answer with no default is an
// error instead, so an unattended run fails loudly rather than blocking on a
// prompt nobody is there to see.
func Resolve(baseDir string, opts Options, p Prompter) (Answers, error) {
	a := Answers{
		SourceType:  opts.SourceType,
		Destination: opts.Destination,
		WebhookURL:  opts.WebhookURL,
	}

	if a.SourceType == "" {
		if detected := sourceTypeFromPackageJSON(baseDir); detected != "" {
			ui.PrintInfo(fmt.Sprintf("Found package.json, sending events from a %s source.", detected))
			a.SourceType = detected
		}
	}

	if a.SourceType == "" {
		if opts.Yes {
			return a, fmt.Errorf("no package.json in %s to infer the source type from: pass --source-type", baseDir)
		}
		chosen, err := p.Select("What is sending events?", sourceTypeChoices)
		if err != nil {
			return a, err
		}
		a.SourceType = chosen
	}

	if a.Destination == "" {
		if opts.Yes {
			a.Destination = DestinationWebhook
		} else {
			// ponytail: a one-option Select rather than a switch over
			// destination kinds. The question has to exist in the demo script,
			// and the list grows by appending to it.
			chosen, err := p.Select("Where should your product data go?", []string{DestinationWebhook})
			if err != nil {
				return a, err
			}
			a.Destination = chosen
		}
	}
	if a.Destination != DestinationWebhook {
		return a, fmt.Errorf("destination %q is not supported yet, only %q is", a.Destination, DestinationWebhook)
	}

	if a.WebhookURL == "" {
		if opts.Yes {
			return a, fmt.Errorf("a webhook URL has no default: pass --webhook-url")
		}
		url, err := p.Ask("What URL should the events be delivered to?", "")
		if err != nil {
			return a, err
		}
		a.WebhookURL = url
	}
	if a.WebhookURL == "" {
		return a, fmt.Errorf("a webhook URL is required")
	}

	return a, nil
}

// sourceTypeFromPackageJSON reads the target directory's package.json so the
// first question can be skipped when the project already says what it is.
//
// ponytail: a package.json means JavaScript and nothing finer. Telling a web
// app from a Node service would mean reading the dependency list and guessing,
// so the wizard picks the web SDK and leaves --source-type for the rest.
func sourceTypeFromPackageJSON(baseDir string) string {
	data, err := os.ReadFile(filepath.Join(baseDir, "package.json"))
	if err != nil {
		return ""
	}
	if !json.Valid(data) {
		return ""
	}
	return "javascript"
}

// Names are the identifiers the generated specs use, derived from the target
// directory so two projects applied to one workspace do not collide.
type Names struct {
	Source      string
	Destination string
	Connection  string
}

func NamesFor(baseDir string) Names {
	base := slug(filepath.Base(mustAbs(baseDir)))
	return Names{
		Source:      base + "-source",
		Destination: base + "-webhook",
		Connection:  base + "-to-webhook",
	}
}

func mustAbs(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func slug(s string) string {
	out := strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(s), "-"), "-")
	if out == "" {
		return "my-app"
	}
	return out
}

// Spec envelopes are written as structs rather than maps so the four envelope
// fields come out in the order the fixtures use; a map would be emitted
// alphabetically.
type specEnvelope struct {
	Version  string   `yaml:"version"`
	Kind     string   `yaml:"kind"`
	Metadata metadata `yaml:"metadata"`
	Spec     any      `yaml:"spec"`
}

type metadata struct {
	Name string `yaml:"name"`
}

type sourceBody struct {
	ID      string `yaml:"id"`
	Name    string `yaml:"name"`
	Type    string `yaml:"type"`
	Enabled bool   `yaml:"enabled"`
}

type destinationBody struct {
	ID                string        `yaml:"id"`
	DisplayName       string        `yaml:"display_name"`
	Type              string        `yaml:"type"`
	Enabled           bool          `yaml:"enabled"`
	DefinitionVersion int64         `yaml:"definition_version"`
	Config            webhookConfig `yaml:"config"`
}

type webhookConfig struct {
	WebhookURL     string            `yaml:"webhook_url"`
	WebhookMethod  string            `yaml:"webhook_method"`
	ConnectionMode map[string]string `yaml:"connection_mode"`
}

type connectionsBody struct {
	Connections []connectionEntry `yaml:"connections"`
}

type connectionEntry struct {
	ID          string `yaml:"id"`
	Source      string `yaml:"source"`
	Destination string `yaml:"destination"`
	Enabled     bool   `yaml:"enabled"`
}

// Entities builds the three specs the wizard writes.
//
// The layout is sources/, destinations/ and connections.yaml directly at the
// target location rather than under a wrapper directory, so every later command
// is 'rudder-cli apply' with no -l argument.
//
// ponytail: no secrets.vars.yaml. A webhook URL is ordinary config. The webhook
// definition's only secret key is headers.to, which this project does not set,
// so a vars file would be empty, and an empty vars file reads as "something is
// missing here" to the next person who opens the directory.
func Entities(names Names, a Answers) []writer.FormattableEntity {
	// connection_mode is keyed by the destination-side token, not by the source
	// type: common.SourceTypeToken maps javascript to web and every server SDK
	// to cloud, and the connection rule compares against the token.
	token := common.SourceTypeToken(a.SourceType, "")

	return []writer.FormattableEntity{
		{
			RelativePath: filepath.Join("sources", names.Source+".yaml"),
			Content: specEnvelope{
				Version:  specVersion,
				Kind:     "event-stream-source",
				Metadata: metadata{Name: names.Source},
				Spec: sourceBody{
					ID:      names.Source,
					Name:    names.Source,
					Type:    a.SourceType,
					Enabled: true,
				},
			},
		},
		{
			RelativePath: filepath.Join("destinations", names.Destination+".yaml"),
			Content: specEnvelope{
				Version:  specVersion,
				Kind:     "destination",
				Metadata: metadata{Name: names.Destination},
				Spec: destinationBody{
					ID:                names.Destination,
					DisplayName:       names.Destination,
					Type:              DestinationWebhook,
					Enabled:           true,
					DefinitionVersion: 1,
					Config: webhookConfig{
						WebhookURL:     a.WebhookURL,
						WebhookMethod:  "POST",
						ConnectionMode: map[string]string{token: "cloud"},
					},
				},
			},
		},
		{
			RelativePath: "connections.yaml",
			Content: specEnvelope{
				Version:  specVersion,
				Kind:     "event-stream-connections",
				Metadata: metadata{Name: names.Connection},
				Spec: connectionsBody{
					Connections: []connectionEntry{{
						ID:          names.Connection,
						Source:      "#event-stream-source:" + names.Source,
						Destination: "#destination:" + names.Destination,
						Enabled:     true,
					}},
				},
			},
		},
	}
}

const specVersion = "rudder/v1"

// EnsureNotInitialised refuses a directory the wizard has already written to.
//
// writer.Write opens files with O_EXCL and so refuses on its own, but its error
// reads as "file exists" and says nothing about what to do. A run can be stopped
// by the directory or by the workspace, and the two have completely different
// fixes, so each says which one it was and what the way out is.
func EnsureNotInitialised(baseDir string, names Names) error {
	for _, e := range Entities(names, Answers{}) {
		path := filepath.Join(baseDir, e.RelativePath)
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf(
				"this directory has already been initialised: %s exists.\n"+
					"       To apply what is written, run 'rudder-cli apply -l %s'.\n"+
					"       To start over, delete the generated sources/, destinations/ and connections.yaml first",
				path, baseDir,
			)
		}
	}
	return nil
}

// Generate writes the project.
func Generate(ctx context.Context, baseDir string, names Names, a Answers) ([]string, error) {
	entities := Entities(names, a)
	formatters := formatter.Setup(formatter.DefaultYAML, formatter.DefaultText)

	if err := writer.Write(ctx, baseDir, formatters, entities); err != nil {
		return nil, fmt.Errorf("writing project: %w", err)
	}

	paths := make([]string, 0, len(entities))
	for _, e := range entities {
		paths = append(paths, filepath.Join(baseDir, e.RelativePath))
	}
	return paths, nil
}

// Deletions lists what a plan would remove, as URNs.
//
// apply reconciles the whole workspace: anything present remotely and absent
// locally is deleted, and there is no flag to narrow that. A wizard run in a
// workspace that already holds resources the local project does not describe
// would therefore destroy them, so the plan is inspected before it is executed.
func Deletions(plan *planner.Plan) []string {
	if plan == nil {
		return nil
	}

	var urns []string
	for _, op := range plan.Operations {
		if op.Type == planner.Delete {
			urns = append(urns, op.Resource.URN())
		}
	}
	return urns
}

// GuardDeletions decides whether an apply that removes resources may proceed.
// Unattended, it refuses: a run with nobody watching must not be the thing that
// empties a workspace. Interactively it names every resource and asks.
func GuardDeletions(plan *planner.Plan, interactive bool, confirm func(string) (bool, error)) error {
	urns := Deletions(plan)
	if len(urns) == 0 {
		return nil
	}

	ui.PrintWarning(fmt.Sprintf("Applying this project would remove %d resource(s) that exist in the workspace:", len(urns)))
	for _, urn := range urns {
		ui.Printf("  - %s\n", urn)
	}

	if !interactive {
		return fmt.Errorf(
			"refusing to remove %d existing resource(s) without confirmation: run init from a terminal, or start from an empty workspace",
			len(urns),
		)
	}

	ok, err := confirm("Remove them and continue?")
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("aborted: nothing was applied")
	}
	return nil
}

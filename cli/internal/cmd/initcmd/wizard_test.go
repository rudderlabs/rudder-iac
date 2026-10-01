package initcmd

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testWorkspace() *client.Workspace {
	return &client.Workspace{ID: "test-workspace-id", Name: "Test Workspace"}
}

// refusingPrompter fails the test if the wizard asks anything. Used by the
// unattended cases, where a prompt is the bug.
func refusingPrompter(t *testing.T) Prompter {
	t.Helper()
	return Prompter{
		Ask: func(question, _ string) (string, error) {
			t.Fatalf("unattended run asked %q", question)
			return "", nil
		},
		Select: func(question string, _ []string) (string, error) {
			t.Fatalf("unattended run asked %q", question)
			return "", nil
		},
	}
}

// TestUnattendedRunCreatesResources is the failure this guards: the syncer's own
// confirmation prompt auto-declines when there is no terminal and then returns
// nil, so an apply that asked would create nothing and still exit zero. The
// wizard therefore passes WithAskConfirmation(false) and owns the decision.
//
// Proving it needs the real syncer, because the bug lives in the option, not in
// the wizard's own branching.
func TestUnattendedRunCreatesResources(t *testing.T) {
	stdinClosed(t)

	// A flagged-through run asks nothing.
	answers, err := Resolve(t.TempDir(), Options{
		SourceType: "node",
		WebhookURL: "https://hooks.example.com/rudder",
		Yes:        true,
	}, refusingPrompter(t))
	require.NoError(t, err)
	assert.Equal(t, Answers{
		SourceType:  "node",
		Destination: DestinationWebhook,
		WebhookURL:  "https://hooks.example.com/rudder",
	}, answers)

	event := testutils.NewMockEvent("event1", resources.ResourceData{"name": "Signed Up"})
	graph := resources.NewGraph()
	graph.AddResource(event)

	provider := &testutils.DataCatalogProvider{
		InitialState:       state.EmptyState(),
		ReconstructedState: state.EmptyState(),
	}

	require.NoError(t, apply(context.Background(), provider, testWorkspace(), graph))

	require.Len(t, provider.OperationLog, 1, "the apply must have reached the provider")
	assert.Equal(t, "Create", provider.OperationLog[0].Operation)
}

// TestDeletionsRefusedWhenUnattended covers the case that matters most: apply
// reconciles the whole workspace, so a plan can carry deletions the wizard never
// asked for. Unattended, that must stop the run.
func TestDeletionsRefusedWhenUnattended(t *testing.T) {
	existing := testutils.NewMockEvent("event-already-there", resources.ResourceData{"name": "Existing"})
	remote := state.EmptyState()
	remote.AddResource(&state.ResourceState{
		ID:     existing.ID(),
		Type:   existing.Type(),
		Input:  resources.ResourceData{"name": "Existing"},
		Output: resources.ResourceData{"id": "remote-id"},
	})

	provider := &testutils.DataCatalogProvider{
		InitialState:       remote,
		ReconstructedState: remote,
	}

	// The wizard's project describes nothing that exists remotely.
	plan, err := dryRun(context.Background(), provider, testWorkspace(), resources.NewGraph())
	require.NoError(t, err)
	assert.Equal(t, []string{existing.URN()}, Deletions(plan))
	assert.Empty(t, provider.OperationLog, "a dry run must not touch the provider")

	err = GuardDeletions(plan, false, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "refusing to remove 1 existing resource(s)")

	// Interactively, a no is an abort rather than a silent apply.
	err = GuardDeletions(plan, true, func(string) (bool, error) { return false, nil })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nothing was applied")

	// And a yes lets the run continue.
	require.NoError(t, GuardDeletions(plan, true, func(string) (bool, error) { return true, nil }))

	// A plan with no deletions never asks.
	require.NoError(t, GuardDeletions(&planner.Plan{}, false, nil))
}

// TestZeroSignalAsksTheMinimum is the interactive path with nothing to go on:
// no flags, no package.json. Three questions, no more, and the answers land
// where they belong.
func TestZeroSignalAsksTheMinimum(t *testing.T) {
	var asked []string

	p := Prompter{
		Select: func(question string, options []string) (string, error) {
			asked = append(asked, question)
			return options[0], nil
		},
		Ask: func(question, def string) (string, error) {
			asked = append(asked, question)
			return "https://hooks.example.com/rudder", nil
		},
	}

	answers, err := Resolve(t.TempDir(), Options{}, p)
	require.NoError(t, err)

	assert.Equal(t, []string{
		"What is sending events?",
		"Where should your product data go?",
		"What URL should the events be delivered to?",
	}, asked)
	assert.Equal(t, Answers{
		SourceType:  sourceTypeChoices[0],
		Destination: DestinationWebhook,
		WebhookURL:  "https://hooks.example.com/rudder",
	}, answers)
}

// TestPackageJSONSkipsTheSourceQuestion is the other half of "three questions at
// most": a directory that already says what it is gets asked two.
func TestPackageJSONSkipsTheSourceQuestion(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"shop"}`), 0o600))

	var asked []string
	p := Prompter{
		Select: func(question string, options []string) (string, error) {
			asked = append(asked, question)
			return options[0], nil
		},
		Ask: func(question, _ string) (string, error) {
			asked = append(asked, question)
			return "https://hooks.example.com/rudder", nil
		},
	}

	answers, err := Resolve(dir, Options{}, p)
	require.NoError(t, err)

	assert.Equal(t, []string{
		"Where should your product data go?",
		"What URL should the events be delivered to?",
	}, asked)
	assert.Equal(t, "javascript", answers.SourceType)
}

// TestYesFailsWithoutARequiredAnswer: --yes must fail rather than prompt when an
// answer has no default, so an unattended run cannot hang.
func TestYesFailsWithoutARequiredAnswer(t *testing.T) {
	_, err := Resolve(t.TempDir(), Options{SourceType: "node", Yes: true}, refusingPrompter(t))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--webhook-url")

	_, err = Resolve(t.TempDir(), Options{WebhookURL: "https://hooks.example.com/x", Yes: true}, refusingPrompter(t))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--source-type")
}

// TestGeneratedSpecsRoundTrip: the specs the wizard writes must survive the
// loaders that read them back. The envelope decodes strictly (specs.New uses
// KnownFields(true)) and so does each provider body, so an unknown field is a
// failure on both sides, which is what makes a typo in the generator loud
// instead of silently dropped.
func TestGeneratedSpecsRoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "shop-web")
	require.NoError(t, os.MkdirAll(dir, 0o755))

	names := NamesFor(dir)
	answers := Answers{
		SourceType:  "javascript",
		Destination: DestinationWebhook,
		WebhookURL:  "https://hooks.example.com/rudder",
	}

	paths, err := Generate(context.Background(), dir, names, answers)
	require.NoError(t, err)
	require.Len(t, paths, 3)

	// Writing again must fail rather than clobber what is already there, and the
	// refusal must name the directory as the cause rather than the workspace.
	err = EnsureNotInitialised(dir, names)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already been initialised")
	assert.Contains(t, err.Error(), "rudder-cli apply -l "+dir)

	_, err = Generate(context.Background(), dir, names, answers)
	require.Error(t, err)

	assert.Equal(t, []string{
		filepath.Join(dir, "sources", names.Source+".yaml"),
		filepath.Join(dir, "destinations", names.Destination+".yaml"),
		filepath.Join(dir, "connections.yaml"),
	}, paths, "the layout is sources/, destinations/ and connections.yaml at the root")

	byKind := map[string]*specs.Spec{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		require.NoError(t, err)

		spec, err := specs.New(data)
		require.NoError(t, err, "generated %s does not parse", path)
		require.NoError(t, spec.Validate(), "generated %s is missing an envelope field", path)
		assert.Equal(t, specs.SpecVersionV1, spec.Version)
		byKind[spec.Kind] = spec

		// The same bytes with one extra envelope key must be rejected.
		_, err = specs.New(append(data, []byte("\nsurprise: true\n")...))
		require.Error(t, err, "the envelope must reject unknown fields")
	}

	require.Contains(t, byKind, "event-stream-source")
	require.Contains(t, byKind, "destination")
	require.Contains(t, byKind, "event-stream-connections")

	// javascript maps to the web token on the destination side, so that is the
	// connection_mode key the connection rule looks for.
	assert.Equal(t, map[string]any{
		"id":                 names.Destination,
		"display_name":       names.Destination,
		"type":               "webhook",
		"enabled":            true,
		"definition_version": 1,
		"config": map[string]any{
			"webhook_url":     "https://hooks.example.com/rudder",
			"webhook_method":  "POST",
			"connection_mode": map[string]any{"web": "cloud"},
		},
	}, byKind["destination"].Spec)

	assert.Equal(t, map[string]any{
		"connections": []any{map[string]any{
			"id":          names.Connection,
			"source":      "#event-stream-source:" + names.Source,
			"destination": "#destination:" + names.Destination,
			"enabled":     true,
		}},
	}, byKind["event-stream-connections"].Spec)

	// The destination provider's own body decodes strictly too: an unknown key
	// inside spec fails there rather than reaching the API.
	var ds destination.DestinationSpec
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{Result: &ds, ErrorUnused: true})
	require.NoError(t, err)
	require.NoError(t, decoder.Decode(byKind["destination"].Spec))

	withUnknown := map[string]any{}
	for k, v := range byKind["destination"].Spec {
		withUnknown[k] = v
	}
	withUnknown["surprise"] = true

	var rejected destination.DestinationSpec
	decoder, err = mapstructure.NewDecoder(&mapstructure.DecoderConfig{Result: &rejected, ErrorUnused: true})
	require.NoError(t, err)
	assert.Error(t, decoder.Decode(withUnknown), "the provider body must reject unknown fields")
}

// TestNoSecretsVarFile: a webhook URL is ordinary config, so the generated
// project has no vars file. An empty one would read as a missing step.
func TestNoSecretsVarFile(t *testing.T) {
	dir := t.TempDir()

	_, err := Generate(context.Background(), dir, NamesFor(dir), Answers{
		SourceType:  "node",
		Destination: DestinationWebhook,
		WebhookURL:  "https://hooks.example.com/rudder",
	})
	require.NoError(t, err)

	var found []string
	require.NoError(t, filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}
		found = append(found, rel)
		return nil
	}))

	names := NamesFor(dir)
	assert.ElementsMatch(t, []string{
		filepath.Join("sources", names.Source+".yaml"),
		filepath.Join("destinations", names.Destination+".yaml"),
		"connections.yaml",
	}, found)
}

func TestSlug(t *testing.T) {
	assert.Equal(t, "my-shop-web", slug("My Shop_Web"))
	assert.Equal(t, "my-app", slug("///"))
	assert.Equal(t, "shop", slug("shop"))
}

// stdinClosed replaces stdin with a closed descriptor for the duration of the
// test, so a prompt would fail rather than silently read EOF from the harness.
func stdinClosed(t *testing.T) {
	t.Helper()

	devNull, err := os.Open(os.DevNull)
	require.NoError(t, err)

	original := os.Stdin
	os.Stdin = devNull
	t.Cleanup(func() {
		os.Stdin = original
		devNull.Close()
	})
}

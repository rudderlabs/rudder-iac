package main

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/tests/demo"
	"github.com/stretchr/testify/assert"
)

func TestDeriveProse(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"strips should", "TestProjectApply/rudder_specs/should_create_entities_in_catalog_from_project", "Create entities in catalog from project"},
		{"strips should mid-phrase", "TestProjectApply/migrated_create_specs_should_produce_the_same_state", "Migrated create specs produce the same state"},
		{"terse subtest", "TestAccountsApply/apply_create", "Apply create"},
		{"top-level test name", "TestConnectionsApply", "Connections apply"},
		{"already a sentence", "TestDestinationsApply/re-apply_churns_only_the_write-only_secret", "Re-apply churns only the write-only secret"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, DeriveProse(tc.in))
		})
	}
}

func TestIsThin(t *testing.T) {
	assert.True(t, IsThin("TestTransformationsTest/success"), "one word says nothing about what happened")
	assert.True(t, IsThin("TestTransformationsTest/failure"))
	assert.False(t, IsThin("TestProjectApply/should_create_entities_in_catalog_from_project"))
	assert.False(t, IsThin("TestAccountsApply/apply_create"))
}

func TestAnnotated(t *testing.T) {
	assert.True(t, Annotated(Step{Records: []demo.Record{
		{Kind: demo.KindSay, Text: "why"},
		{Kind: demo.KindExec, Argv: []string{"rudder-cli", "apply"}},
	}}))
	assert.False(t, Annotated(Step{Records: []demo.Record{
		{Kind: demo.KindExec, Argv: []string{"rudder-cli", "apply"}},
	}}))
}

// A thin, annotated step and a rich, unannotated step exercise the two
// functions independently — neither takes the other's input, but this pins
// that their outputs are allowed to disagree.
func TestIsThinAndAnnotatedAreIndependent(t *testing.T) {
	thinAnnotated := Step{
		Test: "TestTransformationsTest/success",
		Records: []demo.Record{
			{Kind: demo.KindSay, Text: "this is the regression DEX-917 guards"},
			{Kind: demo.KindExec, Argv: []string{"rudder-cli", "apply"}},
		},
	}
	assert.True(t, IsThin(thinAnnotated.Test))
	assert.True(t, Annotated(thinAnnotated))

	richUnannotated := Step{
		Test: "TestProjectApply/rudder_specs/should_create_entities_in_catalog_from_project",
		Records: []demo.Record{
			{Kind: demo.KindExec, Argv: []string{"rudder-cli", "apply"}},
		},
	}
	assert.False(t, IsThin(richUnannotated.Test))
	assert.False(t, Annotated(richUnannotated))
}

func TestVerificationIsVisibleForReadOnlyTail(t *testing.T) {
	cases := []struct {
		name string
		argv []string
		want string
	}{
		{"list", []string{"rudder-cli", "workspace", "retl-connections", "list"}, "visible"},
		{"json flag", []string{"rudder-cli", "workspace", "accounts", "list", "--json"}, "visible"},
		{"validate", []string{"rudder-cli", "validate", "-l", "project"}, "visible"},
		{"dry run", []string{"rudder-cli", "apply", "-l", "project", "--dry-run"}, "visible"},
		{"preview", []string{"rudder-cli", "retl-sources", "preview", "vip"}, "visible"},
		{"apply is a write", []string{"rudder-cli", "apply", "-l", "project", "--confirm=false"}, "none"},
		{"destroy is a write", []string{"rudder-cli", "destroy", "--confirm=false"}, "none"},
		// R13: --json is an output format, not a read/write property. It must
		// not, on its own, make a write look like it proved itself on screen.
		{"json on a write is still a write", []string{"rudder-cli", "apply", "-l", "project", "--json", "--confirm=false"}, "none"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Verification(Step{Records: []demo.Record{
				{Kind: demo.KindExec, Argv: []string{"rudder-cli", "apply"}},
				{Kind: demo.KindExec, Argv: tc.argv},
			}})
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestVerificationIgnoresTrailingNarration(t *testing.T) {
	// A demo.Say after the read-back must not hide the verification.
	got := Verification(Step{Records: []demo.Record{
		{Kind: demo.KindExec, Argv: []string{"rudder-cli", "workspace", "accounts", "list"}},
		{Kind: demo.KindSay, Text: "and there it is"},
	}})
	assert.Equal(t, "visible", got)
}

func TestVerificationOfEmptyStep(t *testing.T) {
	assert.Equal(t, "none", Verification(Step{}))
}

// The last command in a step can itself be a write, distinct from a step that
// ran no commands at all — both must not collapse to indistinguishable
// "none"s for the wrong reason.
func TestVerificationOfWriteOnlyStepIsNoneNotEmpty(t *testing.T) {
	got := Verification(Step{Records: []demo.Record{
		{Kind: demo.KindExec, Argv: []string{"rudder-cli", "destroy", "--confirm=false"}},
	}})
	assert.Equal(t, "none", got)
}

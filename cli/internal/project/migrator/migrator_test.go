package migrator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWriteSpecs_PreservesOriginalOrder exercises the full
// parse -> mutate -> write pipeline that the migrate command runs, and
// asserts the written file keeps the original's key and list-item order
// despite map-based intermediate representation.
func TestWriteSpecs_PreservesOriginalOrder(t *testing.T) {
	t.Parallel()

	// Valid rudder/v0.1 properties spec — shape mirrors the real
	// cli/tests/testdata/project/create/properties.yaml fixture and passes
	// `rudder-cli validate`. Uses only schema-valid fields per
	// cli/internal/providers/datacatalog/rules/property/property_config_valid.go.
	original := `version: rudder/v0.1
kind: properties
metadata:
  name: "api_tracking"
spec:
  properties:
    - id: "api_method"
      name: "API Method"
      type: "string"
      description: "HTTP method used"
      propConfig:
        enum:
          - "GET"
          - "POST"
          - "DELETE"
    - id: "http_retry_count"
      name: "HTTP Retry Count"
      type: "integer"
      description: "Number of retries"
      propConfig:
        minimum: 0
        maximum: 10
        multipleOf: 2
    - id: "user_mail"
      name: "User Email"
      description: "Email address"
      type: "string"
      propConfig:
        format: "email"
        minLength: 5
        maxLength: 100
    - id: "tags"
      name: "Tags"
      type: "array"
      description: "Request tags"
      propConfig:
        itemTypes:
          - "string"
        minItems: 1
        maxItems: 10
    - id: "captcha_solved"
      name: "Captcha Solved"
      description: "Whether captcha was solved"
      type: "boolean"
`

	spec, err := specs.New([]byte(original))
	require.NoError(t, err)
	require.NotNil(t, spec.OriginalNode(), "spec.New must cache the original yaml.Node")

	// Simulate migration-style edits at multiple depths:
	//   - top-level version bump
	//   - scalar mutations inside propConfig blocks
	//   - rename propConfig -> config on every property (mirrors the real
	//     datacatalog migration). The renamed parent is a brand-new key on
	//     the new side, so it appends at the end of the property; its
	//     inner keys have no corresponding original parent and therefore
	//     fall back to the encoder's alphabetic order.
	spec.Version = specs.SpecVersionV1
	props := spec.Spec["properties"].([]any)
	for _, p := range props {
		pm := p.(map[string]any)
		switch pm["id"] {
		case "http_retry_count":
			pm["propConfig"].(map[string]any)["multipleOf"] = 5
		case "user_mail":
			pm["propConfig"].(map[string]any)["minLength"] = 6
		}
		if cfg, ok := pm["propConfig"]; ok {
			pm["config"] = cfg
			delete(pm, "propConfig")
		}
	}

	tmpPath := filepath.Join(t.TempDir(), "properties.yaml")

	m := &Migrator{}
	err = m.WriteSpecs(map[string]*specs.Spec{tmpPath: spec})
	require.NoError(t, err)

	got, err := os.ReadFile(tmpPath)
	require.NoError(t, err)

	expected := `version: "rudder/v1"
kind: "properties"
metadata:
  name: "api_tracking"
spec:
  properties:
    - id: "api_method"
      name: "API Method"
      type: "string"
      description: "HTTP method used"
      config:
        enum:
          - "GET"
          - "POST"
          - "DELETE"
    - id: "http_retry_count"
      name: "HTTP Retry Count"
      type: "integer"
      description: "Number of retries"
      config:
        maximum: 10
        minimum: 0
        multipleOf: 5
    - id: "user_mail"
      name: "User Email"
      description: "Email address"
      type: "string"
      config:
        format: "email"
        maxLength: 100
        minLength: 6
    - id: "tags"
      name: "Tags"
      type: "array"
      description: "Request tags"
      config:
        itemTypes:
          - "string"
        maxItems: 10
        minItems: 1
    - id: "captcha_solved"
      name: "Captcha Solved"
      description: "Whether captcha was solved"
      type: "boolean"
`
	assert.Equal(t, expected, string(got))
}

// TestWriteSpecs_TrackingPlan_PreservesOriginalOrder exercises the write path
// on a tracking plan spec — shape mirrors cli/tests/testdata/project/create/
// trackingplan.yaml and passes `rudder-cli validate` (with companion events /
// properties files). Covers identity matching across two keys:
//   - `id` for spec.rules[*]
//   - `$ref` for rule.properties[*] and case.properties[*]
//
// Variant cases, the `variants[*]` list, and scalar `match` lists go through
// positional recursion. That is sound here because cases/variants are
// typically max=1 per schema validator, and scalar sequences have no
// mappings to reorder.
func TestWriteSpecs_TrackingPlan_PreservesOriginalOrder(t *testing.T) {
	t.Parallel()

	original := `version: rudder/v0.1
kind: tp
metadata:
  name: "api_tracking"
spec:
  id: "api_tracking"
  display_name: "API Tracking"
  description: "Tracking plan for an e-commerce application."
  rules:
    - type: "event_rule"
      id: "login"
      event:
        $ref: "#/events/api_tracking/login"
        allow_unplanned: false
      properties:
        - $ref: "#/properties/api_tracking/api_method"
          required: true
        - $ref: "#/properties/api_tracking/username"
          required: true
        - $ref: "#/properties/api_tracking/password"
          required: true
      variants:
        - type: "discriminator"
          discriminator: "#/properties/api_tracking/api_method"
          cases:
            - display_name: "Create Entity"
              match:
                - "POST"
              description: "applies on POST"
              properties:
                - $ref: "#/properties/api_tracking/user_agent"
                  required: true
                - $ref: "#/properties/api_tracking/host"
                  required: false
    - type: "event_rule"
      id: "logout"
      event:
        $ref: "#/events/api_tracking/logout"
        allow_unplanned: false
`

	spec, err := specs.New([]byte(original))
	require.NoError(t, err)
	require.NotNil(t, spec.OriginalNode())

	// Apply a handful of migration-style changes at different depths:
	//   - bump the top-level version
	//   - flip allow_unplanned on the login rule's event
	//   - mark api_method as not required on the login rule
	//   - mark host as required inside the variant's "Create Entity" case
	//   - reorder the rules list (logout first, login second) so the
	//     formatter must put login back first in the output
	//   - reorder login's properties list (password, api_method, username)
	//     so the formatter must put them back in their original order
	// The expected output below asserts every original position is restored.
	spec.Version = specs.SpecVersionV1

	rules := spec.Spec["rules"].([]any)
	loginRule := rules[0].(map[string]any)
	loginRule["event"].(map[string]any)["allow_unplanned"] = true
	loginProps := loginRule["properties"].([]any)
	loginProps[0].(map[string]any)["required"] = false
	cases := loginRule["variants"].([]any)[0].(map[string]any)["cases"].([]any)
	cases[0].(map[string]any)["properties"].([]any)[1].(map[string]any)["required"] = true

	loginProps[0], loginProps[1], loginProps[2] = loginProps[2], loginProps[0], loginProps[1]
	rules[0], rules[1] = rules[1], rules[0]

	tmpPath := filepath.Join(t.TempDir(), "trackingplan.yaml")
	m := &Migrator{}
	err = m.WriteSpecs(map[string]*specs.Spec{tmpPath: spec})
	require.NoError(t, err)

	got, err := os.ReadFile(tmpPath)
	require.NoError(t, err)

	expected := `version: "rudder/v1"
kind: "tp"
metadata:
  name: "api_tracking"
spec:
  id: "api_tracking"
  display_name: "API Tracking"
  description: "Tracking plan for an e-commerce application."
  rules:
    - type: "event_rule"
      id: "login"
      event:
        $ref: "#/events/api_tracking/login"
        allow_unplanned: true
      properties:
        - $ref: "#/properties/api_tracking/api_method"
          required: false
        - $ref: "#/properties/api_tracking/username"
          required: true
        - $ref: "#/properties/api_tracking/password"
          required: true
      variants:
        - type: "discriminator"
          discriminator: "#/properties/api_tracking/api_method"
          cases:
            - display_name: "Create Entity"
              match:
                - "POST"
              description: "applies on POST"
              properties:
                - $ref: "#/properties/api_tracking/user_agent"
                  required: true
                - $ref: "#/properties/api_tracking/host"
                  required: true
    - type: "event_rule"
      id: "logout"
      event:
        $ref: "#/events/api_tracking/logout"
        allow_unplanned: false
`
	assert.Equal(t, expected, string(got))
}

// A separate module, so the SDK under test adds nothing to the rudder-iac
// module and ./... skips it.
module github.com/rudderlabs/rudder-iac/cli/tests/e2e-sdk/go

go 1.25.0

require github.com/rudderlabs/analytics-go/v4 v4.3.1

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/grafana/jsonparser v0.0.0-20250908162026-5c2524e07b4c // indirect
	github.com/segmentio/backo-go v1.1.0 // indirect
)

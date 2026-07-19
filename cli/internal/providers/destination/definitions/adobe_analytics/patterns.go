package adobeanalytics

import "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"

func init() {
	// Delimiter enums cannot use dynamic_or_oneof because `|` is the
	// go-playground/validator tag separator.
	funcs.NewPattern(
		"adobe_analytics_delimiter",
		`^(\||:|,|;|/)$`,
		"must be one of | : , ; /",
	)

	// schema.json URL fields: (?!.*\.ngrok\.io)^(.{0,100})$ — allow length, reject ngrok.
	funcs.NewPatternWithReject(
		"adobe_analytics_url",
		`^(.{0,100})$`,
		`\.ngrok\.io`,
		"must be at most 100 characters and must not contain .ngrok.io",
	)
}

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
}

package common

import "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"

const (
	// PatternDestinationHTTPURL is the validate:"pattern=..." name for destination
	// HTTP(S) URL fields (GA4 sdk_base_url / server_container_url and peers).
	PatternDestinationHTTPURL = "destination_http_url"
)

const (
	// Allow UI dynamic values, IaC variable substitution, and absolute HTTP(S) URLs.
	// Empty values are handled by omitempty on the field tag (not by this pattern).
	// Mirrors terraform-provider GA4 URL allow regex, plus CLI {{ .VAR }} forms.
	destinationHTTPURLPattern = `(^\{\{.*\|\|(.*)\}\}$)|(^env[.].+)|(^\{\{\s*\.[A-Za-z_][A-Za-z0-9_]*(?:\s*\|\s*((?:[^}]|}[^}])*?))?\s*\}\}$)|^(?:http(s)?://)?[\w.-]+(?:\.[\w\.-]+)+[\w\-\._~:\/?#[\]@!\$&'\(\)\*\+,;=.]*$`

	// Reject ngrok hosts (terraform StringNotMatchesRegexp / schema negative lookahead).
	// Go's regexp engine has no lookahead, so rejection is a separate pattern.
	destinationNgrokPattern = `.*\.ngrok\.io.*`

	destinationHTTPURLErrorMessage = "must be a valid HTTP(S) URL and must not use an ngrok host"
)

func init() {
	funcs.NewPatternWithReject(
		PatternDestinationHTTPURL,
		destinationHTTPURLPattern,
		destinationNgrokPattern,
		destinationHTTPURLErrorMessage,
	)
}

package devlisten

import _ "embed"

// guide is the workflow guide. The dev command help, GET /_dev/v1/guide and
// later rudder-cli docs all read these bytes.
//
//go:embed guide.md
var guide string

// Guide returns the workflow guide as Markdown.
func Guide() string { return guide }

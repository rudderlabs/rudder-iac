package destination

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
)

const (
	// destinationDisplayNameRegexPattern mirrors the RudderStack UI constraint:
	// /^[\w .-]{2,100}$/
	destinationDisplayNameRegexPattern = `^[\w .-]{2,100}$`
	destinationDisplayNameRegexTag     = "destination_display_name"
	destinationDisplayNameErrorMessage = "must be 2-100 characters and contain only letters, digits, underscores, spaces, periods, and hyphens"
)

func init() {
	funcs.NewPattern(
		destinationDisplayNameRegexTag,
		destinationDisplayNameRegexPattern,
		destinationDisplayNameErrorMessage,
	)
}

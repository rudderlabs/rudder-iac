package googleads

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
)

const (
	// conversionIDPattern is the real value constraint from integrations-config
	// googleads schema.json / terraform StringMatchesRegexp, with upstream
	// env./{{…}} alternations stripped.
	conversionIDPattern      = `^AW-(.{0,100})$`
	conversionIDPatternTag   = "googleads_conversion_id"
	conversionIDErrorMessage = "must be a Google Ads conversion ID starting with AW-"
)

func init() {
	funcs.NewPattern(conversionIDPatternTag, conversionIDPattern, conversionIDErrorMessage)
}

package ui

// Controls maps each query parameter of the routes the page reads to the
// element that drives it, so everything the command line can ask is on the
// page too. TestEveryQueryParameterHasAControl keeps it complete.
var Controls = map[string]map[string]string{
	"events": {
		"since":       "only-new",
		"serverId":    "server",
		"limit":       "rows",
		"view":        "detail-tabs",
		"fields":      "f-fields",
		"maxBytes":    "export",
		"event":       "f-event",
		"type":        "f-type",
		"statusCode":  "f-status",
		"writeKey":    "f-key",
		"userId":      "f-user",
		"anonymousId": "f-anon",
		"wait":        "pause",
		"min":         "live-dot",
	},
	"requests": {
		"since":      "only-new",
		"serverId":   "server",
		"limit":      "rows",
		"kind":       "f-kind",
		"statusCode": "f-status",
		"writeKey":   "f-key",
		"failed":     "f-failed",
		"view":       "detail-tabs",
		"fields":     "detail-tabs",
		"maxBytes":   "export",
	},
}

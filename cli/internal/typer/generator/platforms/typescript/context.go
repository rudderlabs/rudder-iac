package typescript

// TSInterfaceProperty is one property in a generated TS interface.
type TSInterfaceProperty struct {
	Name     string // camelCase TS identifier (or quoted serial form when not a valid identifier)
	Type     string // resolved TS type expression, e.g. "string" or "string | null"
	Comment  string
	Optional bool // true → name?: T (omittable under exactOptionalPropertyTypes)
	// QuotedName is true when Name must be wrapped in double quotes (e.g. the
	// JSON key has characters like spaces or `-` that aren't valid in
	// identifier syntax). The template emits `"name"?: T` in that case.
	QuotedName bool
}

// TSInterface → export interface X { ... }
type TSInterface struct {
	Name       string
	Comment    string
	Properties []TSInterfaceProperty
}

// TSMethodArgument is one parameter on a RudderTyper method.
type TSMethodArgument struct {
	Name     string
	Type     string
	Comment  string
	Optional bool // true → name?: T
}

// TSSDKArgument is one positional argument forwarded to the underlying
// RudderAnalytics call. Value is a pre-computed TS expression.
type TSSDKArgument struct {
	Value string
}

// TSAnalyticsMethod is one method on the generated RudderTyper class.
type TSAnalyticsMethod struct {
	Name            string
	Comment         string
	EventName       string
	MethodArguments []TSMethodArgument
	SDKMethodName   string // "identify" for now; future: "track", "screen", "group"
	SDKArguments    []TSSDKArgument
}

// TSContext is the root data object passed to RudderTyper.ts.tmpl.
type TSContext struct {
	Interfaces       []TSInterface
	AnalyticsMethods []TSAnalyticsMethod
	// EventContext is the ruddertyper provenance map injected into every event's
	// options.context. Values are pre-formatted TS literal expressions.
	EventContext        map[string]string
	RudderCLIVersion    string
	TrackingPlanName    string
	TrackingPlanID      string
	TrackingPlanVersion int
	TrackingPlanURL     string
}

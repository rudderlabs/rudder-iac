package golang

// GoContext holds every name, type expression and comment of the generated
// file; the templates only lay them out.
type GoContext struct {
	RudderCLIVersion string
	PackageName      string
	PlanName         string
	PlanURL          string
	PlanID           string
	PlanVersion      int
	// EventContext maps each context.ruddertyper key to its Go literal.
	EventContext map[string]string
	// Imports lists the standard library packages the file uses.
	Imports            []string
	UsesPtr            bool
	UsesWithAdditional bool
	// QuickStart is the method the package doc calls; nil when there is none.
	QuickStart    *GoMethod
	PropertyTypes []GoTypeAlias
	Payloads      []*GoPayload
	Methods       []GoMethod
}

// GoTypeAlias is a `type Name = Type` declaration.
type GoTypeAlias struct {
	Name string
	Doc  string
	Type string
}

// GoPayload is an event payload type: a struct, or a map alias for an open
// rule that declares no properties.
type GoPayload struct {
	Name     string
	Doc      string
	MapAlias bool
	// Open structs carry undeclared keys in AdditionalProperties.
	Open bool
	// DeclaredKeys are the wire keys the schema declares, which
	// AdditionalProperties can never supply, including those of skipped fields.
	DeclaredKeys []string
	Fields       []GoField
}

// GoField is a payload struct field, in byte order of its wire key.
type GoField struct {
	Name string
	// Key is the wire key, exactly as in the tracking plan.
	Key string
	// Type is the field's declared type, a pointer when Pointer is set.
	Type     string
	Required bool
	Pointer  bool
}

// GoMethod is an event method on RudderTyperAnalytics.
type GoMethod struct {
	Name  string
	Doc   string
	Event string
	// Payload is nil when the rule's schema is empty and closed.
	Payload *GoPayload
}

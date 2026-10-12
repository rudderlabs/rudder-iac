package golang

// GoContext holds every name, type expression and comment of the generated
// file; the templates only lay them out.
type GoContext struct {
	RudderCLIVersion   string
	PackageName        string
	PlanName           string
	PlanURL            string
	PlanID             string
	PlanVersion        int
	UsesPtr            bool
	UsesNull           bool
	UsesNullable       bool
	UsesWithAdditional bool
	// QuickStart is the method the package doc calls; nil when there is none.
	QuickStart    *GoMethod
	PropertyTypes []GoPropertyType
	Payloads      []*GoPayload
	Methods       []GoMethod
}

// GoPropertyType is one declaration of the property types section: an alias,
// a union or an enum.
type GoPropertyType struct {
	Alias *GoTypeAlias
	Union *GoUnion
	Enum  *GoEnum
}

// GoTypeAlias is a `type Name = Type` declaration.
type GoTypeAlias struct {
	Name string
	Doc  string
	Type string
}

// GoUnion is a struct holding one value of its member JSON types, which only
// its constructors can set.
type GoUnion struct {
	Name    string
	Doc     string
	Members []GoUnionMember
	// HasNull is set when null is a member, so the zero value is null;
	// otherwise the zero value cannot be sent.
	HasNull bool
}

// GoUnionMember is the constructor of one union member.
type GoUnionMember struct {
	// Name is the candidate New{Union}{Member} until addConstructors replaces
	// it with the registered name, after every type name is registered.
	Name string
	// Doc follows Name in the constructor's doc comment.
	Doc string
	// Type is the parameter type; empty for the null member, whose
	// constructor takes no argument.
	Type string
}

// GoEnum is a defined enum type: typed constants over Base or, when the values
// have different JSON types, a struct that only its value functions build.
type GoEnum struct {
	Name string
	Doc  string
	// Base is the underlying type; empty for values of different JSON types.
	Base string
	// ConstDoc heads the constants.
	ConstDoc string
	// ValuesFunc is the candidate {Name}Values until addConstructors replaces
	// it with the registered name.
	ValuesFunc string
	Values     []GoEnumValue
}

// GoEnumValue is the constant, or the value function, of one enum value.
type GoEnumValue struct {
	// Name is the candidate {Enum}{Value} until addConstructors replaces it
	// with the registered name.
	Name string
	// Text is the value as Go source writes it: "a", 1, 2.5 or true.
	Text string
	// Expr is the value a mixed enum's function holds, typed where Text alone
	// would default to int: int64(1).
	Expr string
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

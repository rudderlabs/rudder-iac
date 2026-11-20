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
	QuickStart *GoMethod
	// The type sections, in file order. Each type is followed by the types it
	// hoists.
	CustomTypes   []GoDecl
	PropertyTypes []GoDecl
	Payloads      []GoDecl
	Methods       []GoMethod
	// out is the section that add appends to.
	out *[]GoDecl
}

func (ctx *GoContext) add(d GoDecl) { *ctx.out = append(*ctx.out, d) }

// GoDecl is one type declaration: an alias, a union, an enum or a struct.
type GoDecl struct {
	Alias  *GoTypeAlias
	Union  *GoUnion
	Enum   *GoEnum
	Struct *GoStruct
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
	// Name is the candidate New{Union}{Member} until registerValueNames replaces
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
	// ValuesFunc is the candidate {Name}Values until registerValueNames replaces
	// it with the registered name.
	ValuesFunc string
	Values     []GoEnumValue
}

// GoEnumValue is the constant, or the value function, of one enum value.
type GoEnumValue struct {
	// Name is the candidate {Enum}{Value} until registerValueNames replaces it
	// with the registered name.
	Name string
	// Text is the value as Go source writes it: "a", 1, 2.5 or true.
	Text string
	// Expr is the value a mixed enum's function holds, typed where Text alone
	// would default to int: int64(1).
	Expr string
}

// GoStruct is an object type: an event payload, an object custom type or a
// hoisted nested object. A payload of an open rule that declares no
// properties is a map alias instead.
type GoStruct struct {
	Name     string
	Doc      string
	MapAlias bool
	// Payload structs also have ToProperties.
	Payload bool
	// Open structs carry undeclared keys in AdditionalProperties.
	Open bool
	// DeclaredKeys are the wire keys the schema declares, which
	// AdditionalProperties can never supply, including those of skipped fields.
	DeclaredKeys []string
	Fields       []GoField
}

// GoField is a struct field, in byte order of its wire key.
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
	Payload *GoStruct
}

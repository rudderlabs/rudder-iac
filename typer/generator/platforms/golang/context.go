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
	// UsesRequired is set when a struct has a required variant field.
	UsesRequired bool
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
	// afterValueNames finish the variant code that spells enum constants and
	// union constructors, once registerValueNames has named them. Each gets
	// the aliases, enums and unions by name.
	afterValueNames []func(decls map[string]GoDecl)
}

func (ctx *GoContext) add(d GoDecl) { *ctx.out = append(*ctx.out, d) }

// GoDecl is one type declaration: an alias, a union, an enum, a struct or a
// variant interface.
type GoDecl struct {
	Alias   *GoTypeAlias
	Union   *GoUnion
	Enum    *GoEnum
	Struct  *GoStruct
	Variant *GoVariant
}

// GoVariant is the sealed interface of a discriminated union, which only its
// case, Absent and Default structs implement.
type GoVariant struct {
	Name string
	Doc  string
	// Key is the discriminator's wire key.
	Key string
	// Result is what Discriminator returns: the discriminator's type, or
	// (value T, ok bool) when the discriminator is optional.
	Result           string
	DiscriminatorDoc string
	// Payload is set on payload interfaces, which also list To{Payload}, as
	// GoStruct.Payload.
	Payload string
	// Implementations are the structs, in file order.
	Implementations []string
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
	// Payload is set on event payloads: Properties, or Traits for identify
	// and group. It names the To{Payload} method of a payload struct and the
	// analytics map type it returns.
	Payload string
	// Open structs carry undeclared keys in AdditionalProperties.
	Open bool
	// DeclaredKeys are the wire keys the schema declares, which
	// AdditionalProperties can never supply, including a variant case's
	// discriminator.
	DeclaredKeys []string
	Fields       []GoField
	// Variant is the interface that a case, Absent or Default struct
	// implements. A method's payload that is a variant interface is a
	// GoStruct naming it, with Variant set to it.
	Variant *GoVariant
	// DiscriminatorDoc documents the Discriminator method, which returns
	// Returns. The method names its receiver only when Returns reads it, as
	// Default's does.
	DiscriminatorDoc string
	Returns          string
	ReadsReceiver    bool
	// Sends is the discriminator value a case's toMap writes; empty for
	// Absent, which sends none, and Default, whose field holds it.
	Sends string
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
	// Variant fields hold a variant interface, whose nil and typed-nil
	// values toMap must catch itself.
	Variant bool
	// Note follows "required" or "optional" in the field's comment.
	Note string
}

// GoMethod is an event method on RudderTyperAnalytics.
type GoMethod struct {
	Name  string
	Doc   string
	Event string
	// Payload is nil when the rule's schema is empty and closed.
	Payload *GoStruct
}

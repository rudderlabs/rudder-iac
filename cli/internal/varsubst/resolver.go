package varsubst

// Resolver provides variable values by name.
type Resolver interface {
	Resolve(name string) (value string, found bool)
}

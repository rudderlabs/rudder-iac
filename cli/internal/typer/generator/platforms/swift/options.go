package swift

// SwiftOptions holds platform-specific options for Swift code generation.
// These can be passed via --option flags in the CLI, e.g.:
//
//	--option outputFileName=Events.swift
type SwiftOptions struct {
	OutputFileName string `mapstructure:"outputFileName" description:"Name of the generated Swift file. Defaults to RudderTyper.swift"`
}

// DefaultOptions returns the default Swift generation options.
func (g *Generator) DefaultOptions() any {
	return SwiftOptions{
		OutputFileName: "RudderTyper.swift",
	}
}

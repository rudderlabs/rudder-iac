package source

const (
	ResourceType = "event-stream-source"

	IDKey               = "id" // remote id
	NameKey          = "name"
	EnabledKey          = "enabled"
	SourceDefinitionKey = "source_definition"
)

var sourceDefinitions = []string{
	"Java",	
	"DotNet",
	"PHP",
	"Flutter",
	"Cordova",
	"Rust",
	"ReactNative",
	"Python",
	"iOS",
	"Android",
	"Javascript",
	"Go",
	"Node",
	"Ruby",
	"Unity",
}

type sourceSpec struct {
	LocalId      string `mapstructure:"id"`
	Name    string `mapstructure:"name"`
	SourceDefinition string `mapstructure:"source_definition"`
	Enabled bool   `mapstructure:"enabled"`
}
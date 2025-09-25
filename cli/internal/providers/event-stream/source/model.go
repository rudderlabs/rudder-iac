package source

const (
	ResourceType = "event-stream-source"

	IDKey               = "id" // remote id
	NameKey             = "name"
	EnabledKey          = "enabled"
	SourceDefinitionKey = "type"
)

var sourceDefinitions = []string{
	"java",
	"dotnet",
	"php",
	"flutter",
	"cordova",
	"rust",
	"react_native",
	"python",
	"ios",
	"android",
	"javascript",
	"go",
	"node",
	"ruby",
	"unity",
}

type sourceSpec struct {
	LocalId          string `mapstructure:"id"`
	Name             string `mapstructure:"name"`
	SourceDefinition string `mapstructure:"type"`
	Enabled          bool   `mapstructure:"enabled"`
}

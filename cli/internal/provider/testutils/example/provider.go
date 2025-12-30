package example

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/backend"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/handlers/book"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/handlers/writer"
)

// NewProvider creates a new example provider with all resource handlers
func NewProvider(backend *backend.Backend) provider.Provider {
	handlers := []provider.Handler{
		writer.NewHandler(backend),
		book.NewHandler(backend),
	}

	return provider.NewBaseProvider(handlers)
}

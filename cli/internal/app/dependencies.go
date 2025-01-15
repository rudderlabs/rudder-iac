package app

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
	"github.com/rudderlabs/rudder-iac/cli/pkg/provider"
)

var (
	p  syncer.Provider
	sm syncer.StateManager
	s  *syncer.ProjectSyncer
)

func Initialise(version string) (err error) {
	sm = newStateManager()

	p, err = newProvider(version)
	if err != nil {
		return fmt.Errorf("creating provider: %w", err)
	}

	s = syncer.New(p, sm)
	return
}

func newStateManager() syncer.StateManager {
	return &state.LocalManager{
		BaseDir: config.GetConfigDir(),
	}
}

func newProvider(version string) (syncer.Provider, error) {
	cfg := config.GetConfig()
	rawClient, err := client.New(
		cfg.Auth.AccessToken,
		client.WithBaseURL(cfg.APIURL),
		client.WithUserAgent("rudder-cli/"+version),
	)

	if err != nil {
		return nil, fmt.Errorf("creating client: %w", err)
	}
	return provider.NewCatalogProvider(client.NewRudderDataCatalog(rawClient)), nil
}

func StateManager() syncer.StateManager {
	return sm
}

func Provider() syncer.Provider {
	return p
}

func Syncer() *syncer.ProjectSyncer {
	return s
}

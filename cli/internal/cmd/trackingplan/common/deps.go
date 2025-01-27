package common

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
)

func NewSyncer() (*syncer.ProjectSyncer, error) {
	d, err := app.NewDeps()
	if err != nil {
		return nil, err
	}

	s, err := syncer.New(d.Provider(), d.StateManager())
	if err != nil {
		return nil, err
	}

	return s, nil
}

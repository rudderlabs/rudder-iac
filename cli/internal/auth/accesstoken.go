package auth

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

func Login() error {
	accessToken, err := ui.AskSecret("Enter your access token:")
	if err != nil {
		return fmt.Errorf("error reading access token: %w", err)
	}

	config.SetAccessToken(accessToken)

	return nil
}

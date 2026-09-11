package cmderrors

import (
	"errors"

	"github.com/rudderlabs/rudder-iac/api/client"
)

const (
	featureDisabledMessage  = "This feature is not enabled for your workspace. Please contact support to enable it."
	permissionDeniedMessage = "Access denied: Your configured token does not have sufficient permissions to perform this action."
)

// SilentError wraps an error that should cause a non-zero exit code without
// printing an error message to stderr. This is useful for commands that produce
// structured output (e.g., JSON) where the output already contains all failure
// information and an additional stderr message would be redundant or disruptive
// to machine-readable output.
type SilentError struct {
	Err error
}

func (e *SilentError) Error() string {
	return e.Err.Error()
}

func (e *SilentError) Unwrap() error {
	return e.Err
}

func FormatUserFacingError(err error) string {
	if err == nil {
		return ""
	}

	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		if apiErr.IsFeatureDisabled() {
			return featureDisabledMessage
		}
		if apiErr.IsPermissionDenied() {
			return permissionDeniedMessage
		}
	}

	return err.Error()
}

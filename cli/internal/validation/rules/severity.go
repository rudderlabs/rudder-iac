package rules

// Severity represents the severity level of a validation result.
// It indicates how critical a validation issue is and determines
// whether it should block deployment (Error), warn the user (Warning),
// or provide informational feedback (Info).
type Severity int

const (
	// Info indicates informational messages that don't require action.
	// These are typically best practice suggestions or tips.
	Info Severity = iota

	// Warning indicates potential issues that should be reviewed.
	// These don't block deployment but should be addressed.
	Warning

	// Error indicates validation failures that must be fixed.
	// These block deployment and require immediate attention.
	Error
)

// String returns the string representation of the severity level.
// This is useful for formatting output and logging.
func (s Severity) String() string {
	switch s {
	case Info:
		return "info"
	case Warning:
		return "warning"
	case Error:
		return "error"
	default:
		return "unknown"
	}
}

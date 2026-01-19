package rules

// ValidationResult represents the outcome of validating a single aspect of a spec.
// It contains all the information needed to report a validation issue to the user,
// including severity, location, and descriptive message.
type ValidationResult struct {
	// RuleID is the unique identifier of the rule that produced this result
	RuleID string

	// Severity indicates how critical this result is (Error, Warning, or Info)
	Severity Severity

	// Message is a human-readable description of the validation result
	Message string

	// FilePath is the absolute path to the file being validated
	FilePath string

	// FileName is the name of the file being validated (without directory path)
	FileName string

	// Reference is an optional JSON pointer or spec reference (e.g., "#/properties/group/id")
	// that indicates where in the spec the issue was found
	Reference string

	// Examples provides valid and invalid usage examples for this validation result.
	// This is useful for generating rich diagnostic information (e.g., JSONDiagnostic format)
	Examples Examples
}

// IsError returns true if this result represents an error-level issue.
// Errors typically block deployment and require immediate attention.
func (vr *ValidationResult) IsError() bool {
	return vr.Severity == Error
}

// IsWarning returns true if this result represents a warning-level issue.
// Warnings are potential problems that should be reviewed but don't block deployment.
func (vr *ValidationResult) IsWarning() bool {
	return vr.Severity == Warning
}

// IsInfo returns true if this result represents informational feedback.
// Info messages provide best practice suggestions and don't require action.
func (vr *ValidationResult) IsInfo() bool {
	return vr.Severity == Info
}
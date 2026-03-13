package validations

import dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"

// RunStatus indicates whether validation was executed or skipped
type RunStatus int

const (
	RunStatusExecuted    RunStatus = iota
	RunStatusNoResources
)

// ResourceValidation holds the validation result for a single resource
type ResourceValidation struct {
	ID           string
	DisplayName  string
	ResourceType string // "model" or "relationship"
	Issues       []dgClient.ValidationIssue
	Err          error
}

// HasErrors returns true if there are any error-severity issues or an execution error
func (rv *ResourceValidation) HasErrors() bool {
	if rv.Err != nil {
		return true
	}
	for _, issue := range rv.Issues {
		if issue.Severity == "error" {
			return true
		}
	}
	return false
}

// HasWarnings returns true if there are any warning-severity issues
func (rv *ResourceValidation) HasWarnings() bool {
	for _, issue := range rv.Issues {
		if issue.Severity == "warning" {
			return true
		}
	}
	return false
}

// ValidationResults holds all validation results for a run
type ValidationResults struct {
	Status    RunStatus
	Resources []*ResourceValidation
}

// HasFailures returns true if any resource has errors
func (vr *ValidationResults) HasFailures() bool {
	for _, r := range vr.Resources {
		if r.HasErrors() {
			return true
		}
	}
	return false
}

// ErrorCount returns the number of resources with errors
func (vr *ValidationResults) ErrorCount() int {
	count := 0
	for _, r := range vr.Resources {
		if r.HasErrors() {
			count++
		}
	}
	return count
}

// WarningCount returns the number of resources with warnings (but no errors)
func (vr *ValidationResults) WarningCount() int {
	count := 0
	for _, r := range vr.Resources {
		if !r.HasErrors() && r.HasWarnings() {
			count++
		}
	}
	return count
}

// PassCount returns the number of resources that passed validation
func (vr *ValidationResults) PassCount() int {
	count := 0
	for _, r := range vr.Resources {
		if !r.HasErrors() && !r.HasWarnings() {
			count++
		}
	}
	return count
}

// ResourcesByType returns resources filtered by type
func (vr *ValidationResults) ResourcesByType(resourceType string) []*ResourceValidation {
	var result []*ResourceValidation
	for _, r := range vr.Resources {
		if r.ResourceType == resourceType {
			result = append(result, r)
		}
	}
	return result
}
